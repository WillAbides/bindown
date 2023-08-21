package builddep

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v52/github"
	"github.com/willabides/bindown/v4/internal/bindown"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

var forbiddenOS = map[string]bool{
	"js":     true,
	"wasip1": true,
}

var forbiddenArch = map[string]bool{
	"wasm": true,
}

func distSystems() []bindown.System {
	var systems []bindown.System
	for _, system := range strings.Split(strings.TrimSpace(bindown.GoDists), "\n") {
		systems = append(systems, bindown.System(system))
	}
	return systems
}

func AddDependency(
	ctx context.Context,
	cfg *bindown.Config,
	name, version string,
	homepage, description string,
	urls []string,
) error {
	return addDependency(ctx, cfg, name, version, homepage, description, urls, nil)
}

func addDependency(
	ctx context.Context,
	cfg *bindown.Config,
	name, version string,
	homepage, description string,
	urls []string,
	selector selectCandidateFunc,
) error {
	systems := distSystems()
	if cfg.Systems != nil {
		systems = append(systems[:0], cfg.Systems...)
	}
	groups := parseDownloads(urls, name, version, systems)
	var regrouped []*depGroup
	for _, g := range groups {
		gg, err := g.regroupByArchivePath(ctx, name, version, selector)
		if err != nil {
			return err
		}
		regrouped = append(regrouped, gg...)
	}
	built := buildConfig(name, version, regrouped)
	err := built.AddChecksums([]string{name}, built.Dependencies[name].Systems)
	if err != nil {
		return err
	}
	err = built.Validate(name, built.Systems)
	if err != nil {
		b, e := yaml.Marshal(&bindown.Config{
			Dependencies: built.Dependencies,
			Templates:    built.Templates,
			URLChecksums: built.URLChecksums,
		})
		if e != nil {
			b = []byte(fmt.Sprintf("could not marshal invalid config: %v", e))
		}
		return fmt.Errorf("generated config is invalid: %v\n\n%s", err, string(b))
	}
	for k, v := range built.Dependencies {
		if cfg.Dependencies == nil {
			cfg.Dependencies = make(map[string]*bindown.Dependency)
		}
		cfg.Dependencies[k] = v
	}
	for k, v := range built.Templates {
		if homepage != "" {
			v.Homepage = &homepage
		}
		if description != "" {
			v.Description = &description
		}
		if cfg.Templates == nil {
			cfg.Templates = make(map[string]*bindown.Dependency)
		}
		cfg.Templates[k] = v
	}
	for k, v := range built.URLChecksums {
		if cfg.URLChecksums == nil {
			cfg.URLChecksums = make(map[string]string)
		}
		cfg.URLChecksums[k] = v
	}
	return nil
}

type systemSub struct {
	val        string
	normalized string
	priority   int
	idx        int
}

func osSubs(systems []bindown.System) []systemSub {
	subs := []systemSub{
		{val: "apple-darwin", normalized: "darwin"},
		{val: "unknown-linux-gnu", normalized: "linux", priority: -1},
		{val: "unknown-linux-musl", normalized: "linux"},
		{val: "pc-windows-msvc", normalized: "windows"},
		{val: "pc-windows-gnu", normalized: "windows", priority: -1},
		{val: "apple", normalized: "darwin"},
		{val: "osx", normalized: "darwin"},
		{val: "macos", normalized: "darwin"},
		{val: "mac", normalized: "darwin"},
		{val: "windows", normalized: "windows"},
		{val: "darwin", normalized: "darwin"},
		{val: "win64", normalized: "windows"},
		{val: "win", normalized: "windows"},
	}
	if systems == nil {
		systems = distSystems()
	}
	for _, dist := range systems {
		if !slices.ContainsFunc(subs, func(sub systemSub) bool {
			return sub.val == dist.OS()
		}) {
			subs = append(subs, systemSub{val: dist.OS(), normalized: dist.OS()})
		}
	}
	slices.SortFunc(subs, func(a, b systemSub) int {
		return cmp.Compare(len(b.val), len(a.val))
	})
	return subs
}

func archSubs(systems []bindown.System) []systemSub {
	subs := []systemSub{
		{val: "amd64", normalized: "amd64"},
		{val: "arm64", normalized: "arm64"},
		{val: "x86_64", normalized: "amd64"},
		{val: "x86_32", normalized: "386"},
		{val: "x86", normalized: "386"},
		{val: "x64", normalized: "amd64"},
		{val: "64bit", normalized: "amd64"},
		{val: "64-bit", normalized: "amd64"},
		{val: "aarch64", normalized: "arm64"},
		{val: "aarch_64", normalized: "arm64"},
		{val: "ppcle_64", normalized: "ppc64le"},
		{val: "s390x_64", normalized: "s390x"},
		{val: "i386", normalized: "386"},
		{val: "armv6", normalized: "arm"},
		{val: "armv7", normalized: "arm"},
		{val: "armv5", normalized: "arm"},
		{val: "armv6l", normalized: "arm"},
		{val: "armv6hf", normalized: "arm"},
	}
	if systems == nil {
		systems = distSystems()
	}
	for _, dist := range systems {
		if !slices.ContainsFunc(subs, func(sub systemSub) bool {
			return sub.val == dist.Arch()
		}) {
			subs = append(subs, systemSub{val: dist.Arch(), normalized: dist.Arch()})
		}
	}
	slices.SortFunc(subs, func(a, b systemSub) int {
		return cmp.Compare(len(b.val), len(a.val))
	})
	return subs
}

func matchSub(filename string, subs []systemSub) *systemSub {
	downcased := strings.ToLower(filename)
	for _, sub := range subs {
		idx := strings.Index(downcased, sub.val)
		if idx == -1 {
			continue
		}
		casedVal := filename[idx : idx+len(sub.val)]
		return &systemSub{
			val:        casedVal,
			normalized: sub.normalized,
			priority:   sub.priority,
			idx:        idx,
		}
	}
	return nil
}

func parseOs(filename string, systems []bindown.System) *systemSub {
	sub := matchSub(filename, osSubs(systems))
	if sub != nil {
		return sub
	}
	if strings.HasSuffix(strings.ToLower(filename), ".exe") {
		return &systemSub{
			val:        "",
			normalized: "windows",
			idx:        -1,
		}
	}
	return nil
}

func parseArch(filename string, systems []bindown.System) *systemSub {
	sub := matchSub(filename, archSubs(systems))
	if sub != nil {
		return sub
	}
	return &systemSub{
		val:        "",
		normalized: "amd64",
		idx:        -1,
		priority:   -1,
	}
}

var archiveSuffixes = []string{
	".tar.br",
	".tbr",
	".tar.bz2",
	".tbz2",
	".tar.gz",
	".tgz",
	".tar.lz4",
	".tlz4",
	".tar.sz",
	".tsz",
	".tar.xz",
	".txz",
	".tar.zst",
	".tzst",
	".rar",
	".zip",
}

var compressSuffixes = []string{
	".br",
	".gz",
	".bz2",
	".lz4",
	".sz",
	".xz",
	".zst",
}

func parseDownload(dlURL, version string, systems []bindown.System) (*dlFile, bool) {
	tmpl := dlURL
	osSub := parseOs(dlURL, systems)
	if osSub == nil {
		return nil, false
	}
	if osSub.idx != -1 {
		tmpl = tmpl[:osSub.idx] + "{{.os}}" + tmpl[osSub.idx+len(osSub.val):]
	}
	archSub := parseArch(tmpl, systems)
	if archSub == nil {
		return nil, false
	}
	if archSub.idx != -1 {
		tmpl = tmpl[:archSub.idx] + "{{.arch}}" + tmpl[archSub.idx+len(archSub.val):]
	}
	if forbiddenArch[archSub.normalized] || forbiddenOS[osSub.normalized] {
		return nil, false
	}
	if !slices.ContainsFunc(systems, func(sys bindown.System) bool {
		return sys.OS() == osSub.normalized && sys.Arch() == archSub.normalized
	}) {
		return nil, false
	}
	isArchive := false
	isCompress := false
	suffix := ""
	for _, s := range archiveSuffixes {
		if strings.HasSuffix(dlURL, s) {
			suffix = s
			isArchive = true
			break
		}
	}
	if !isArchive {
		for _, s := range compressSuffixes {
			if strings.HasSuffix(dlURL, s) {
				suffix = s
				isCompress = true
				break
			}
		}
	}
	if strings.HasSuffix(dlURL, ".exe") {
		suffix = ".exe"
	}
	tmpl = tmpl[:len(tmpl)-len(suffix)] + "{{.urlSuffix}}"
	if version != "" {
		tmpl = strings.ReplaceAll(tmpl, version, "{{.version}}")
	}
	priority := 0
	if osSub != nil {
		priority += osSub.priority
	}
	if archSub != nil {
		priority += archSub.priority
	}
	return &dlFile{
		origUrl:    dlURL,
		url:        tmpl,
		osSub:      osSub,
		archSub:    archSub,
		suffix:     suffix,
		isArchive:  isArchive,
		isCompress: isCompress,
		priority:   priority,
	}, true
}

func parseDownloads(dlUrls []string, binName, version string, allowedSystems []bindown.System) []*depGroup {
	systemFiles := map[bindown.System][]*dlFile{}
	for _, dlUrl := range dlUrls {
		f, ok := parseDownload(dlUrl, version, allowedSystems)
		if !ok {
			continue
		}
		dlSystem := f.system()
		systemFiles[dlSystem] = append(systemFiles[dlSystem], f)
	}
	for system := range systemFiles {
		if len(systemFiles[system]) < 2 {
			continue
		}
		// remove all but the highest priority
		slices.SortFunc(systemFiles[system], func(a, b *dlFile) int {
			return cmp.Compare(b.priority, a.priority)
		})
		cutOff := slices.IndexFunc(systemFiles[system], func(f *dlFile) bool {
			return f.priority < systemFiles[system][0].priority
		})
		if cutOff != -1 {
			systemFiles[system] = systemFiles[system][:cutOff]
		}
	}

	urlFrequency := map[string]int{}
	for _, files := range systemFiles {
		for _, f := range files {
			urlFrequency[f.url]++
		}
	}

	for system := range systemFiles {
		if len(systemFiles[system]) < 2 {
			continue
		}
		// prefer templates that are used more often
		slices.SortFunc(systemFiles[system], func(a, b *dlFile) int {
			return cmp.Compare(urlFrequency[b.url], urlFrequency[a.url])
		})
		cutOff := slices.IndexFunc(systemFiles[system], func(f *dlFile) bool {
			return urlFrequency[f.url] < urlFrequency[systemFiles[system][0].url]
		})
		if cutOff != -1 {
			systemFiles[system] = systemFiles[system][:cutOff]
		}
		if len(systemFiles[system]) == 1 {
			continue
		}
		// prefer archives then compresses
		var archiveSystems, compressSystems, plainSystems []*dlFile
		for _, file := range systemFiles[system] {
			switch {
			case file.isArchive:
				archiveSystems = append(archiveSystems, file)
			case file.isCompress:
				compressSystems = append(compressSystems, file)
			default:
				plainSystems = append(plainSystems, file)
			}
		}
		var sf []*dlFile
		switch {
		case len(archiveSystems) > 0:
			sf = archiveSystems
		case len(compressSystems) > 0:
			sf = compressSystems
		default:
			sf = plainSystems
		}
		if len(sf) < 2 {
			systemFiles[system] = sf
		}
		// now arbitrarily pick the first one alphabetically by origUrl
		slices.SortFunc(sf, func(a, b *dlFile) int {
			return cmp.Compare(a.origUrl, b.origUrl)
		})
		systemFiles[system] = sf[:1]
	}

	// special handling to remap darwin/arm64 to darwin/amd64
	if len(systemFiles["darwin/amd64"]) > 0 && len(systemFiles["darwin/arm64"]) == 0 && slices.Contains(allowedSystems, "darwin/arm64") {
		f := systemFiles["darwin/amd64"][0].clone()
		f.archSub.normalized = "arm64"
		f.priority -= 2
		systemFiles["darwin/arm64"] = append(systemFiles["darwin/arm64"], f)
	}

	var groups []*depGroup
	systems := bindown.MapKeys(systemFiles)

	slices.SortFunc(systems, func(a, b bindown.System) int {
		if len(systemFiles[a]) == 0 || len(systemFiles[b]) == 0 {
			return cmp.Compare(len(systemFiles[b]), len(systemFiles[a]))
		}
		aFile := systemFiles[a][0]
		bFile := systemFiles[b][0]
		if aFile.priority != bFile.priority {
			return cmp.Compare(bFile.priority, aFile.priority)
		}
		return cmp.Compare(a, b)
	})
	for _, system := range systems {
		file := systemFiles[system][0]
		idx := slices.IndexFunc(groups, func(g *depGroup) bool {
			return g.fileAllowed(file, binName)
		})
		if idx != -1 {
			groups[idx].addFile(file, binName)
			continue
		}
		group := &depGroup{
			substitutions: map[string]map[string]string{
				"os":   {},
				"arch": {},
			},
			overrideMatcher: map[string][]string{},
		}
		group.addFile(file, binName)
		groups = append(groups, group)
	}
	slices.SortFunc(groups, func(a, b *depGroup) int {
		return cmp.Compare(len(a.files), len(b.files))
	})
	return groups
}

func buildConfig(name, version string, groups []*depGroup) *bindown.Config {
	dep := groups[0].dependency()
	checksums := map[string]string{}
	for i := 0; i < len(groups); i++ {
		group := groups[i]
		for _, file := range group.files {
			checksums[file.origUrl] = file.checksum
		}
		if i == 0 {
			continue
		}
		otherGroups := slices.Clone(groups[:i])
		otherGroups = append(otherGroups, groups[i+1:]...)
		dep.Systems = append(dep.Systems, group.systems...)
		dep.Overrides = append(dep.Overrides, group.overrides(otherGroups)...)
	}
	slices.Sort(dep.Systems)
	for tp := range dep.Substitutions {
		for k, v := range dep.Substitutions[tp] {
			if k == v {
				delete(dep.Substitutions[tp], k)
			}
		}
		if len(dep.Substitutions[tp]) == 0 {
			delete(dep.Substitutions, tp)
		}
	}
	for i := range dep.Overrides {
		for tp := range dep.Overrides[i].Dependency.Substitutions {
			for k, v := range dep.Overrides[i].Dependency.Substitutions[tp] {
				if k != v {
					continue
				}
				var depVal string
				var ok bool
				if dep.Substitutions != nil && dep.Substitutions[tp] != nil {
					depVal, ok = dep.Substitutions[tp][k]
				}
				if !ok {
					if k == v {
						delete(dep.Overrides[i].Dependency.Substitutions[tp], k)
					}
					continue
				}
				if depVal == v {
					delete(dep.Overrides[i].Dependency.Substitutions[tp], k)
				}
			}
			if len(dep.Overrides[i].Dependency.Substitutions[tp]) == 0 {
				delete(dep.Overrides[i].Dependency.Substitutions, tp)
			}
		}
	}
	return &bindown.Config{
		Systems: dep.Systems,
		Dependencies: map[string]*bindown.Dependency{
			name: {
				Template: &name,
				Overrideable: bindown.Overrideable{
					Vars: map[string]string{
						"version": version,
					},
				},
			},
		},
		Templates: map[string]*bindown.Dependency{
			name: dep,
		},
		URLChecksums: checksums,
	}
}

func splitSystems(systems []bindown.System, fn func(s bindown.System) bool) (matching, nonMatching []bindown.System) {
	for _, system := range systems {
		if fn(system) {
			matching = append(matching, system)
		} else {
			nonMatching = append(nonMatching, system)
		}
	}
	return
}

func systemsMatcher(systems, otherSystems []bindown.System) (_ map[string][]string, matcherSystems, remainingSystems []bindown.System) {
	var oses, arches, otherOses, otherArches, exclusiveOses, exclusiveArches []string
	for _, system := range systems {
		if !slices.Contains(oses, system.OS()) {
			oses = append(oses, system.OS())
		}
		if !slices.Contains(arches, system.Arch()) {
			arches = append(arches, system.Arch())
		}
	}
	for _, system := range otherSystems {
		if !slices.Contains(otherOses, system.OS()) {
			otherOses = append(otherOses, system.OS())
		}
		if !slices.Contains(otherArches, system.Arch()) {
			otherArches = append(otherArches, system.Arch())
		}
	}
	for _, s := range oses {
		if !slices.Contains(otherOses, s) {
			exclusiveOses = append(exclusiveOses, s)
		}
	}
	if len(exclusiveOses) > 0 {
		s, r := splitSystems(systems, func(system bindown.System) bool {
			return slices.Contains(exclusiveOses, system.OS())
		})
		return map[string][]string{"os": exclusiveOses}, s, r
	}
	for _, s := range arches {
		if !slices.Contains(otherArches, s) {
			exclusiveArches = append(exclusiveArches, s)
		}
	}
	if len(exclusiveArches) > 0 {
		s, r := splitSystems(systems, func(system bindown.System) bool {
			return slices.Contains(exclusiveArches, system.Arch())
		})
		return map[string][]string{"arch": exclusiveArches}, s, r
	}
	if (len(oses) == 0) != (len(arches) == 0) {
		panic("inconsistent systems")
	}
	if len(oses) == 0 {
		return nil, nil, systems
	}
	if len(arches) < len(oses) {
		a := arches[0]
		var archOses []string
		for _, system := range systems {
			if system.Arch() == a {
				matcherSystems = append(matcherSystems, system)
				archOses = append(archOses, system.OS())
				continue
			}
			remainingSystems = append(remainingSystems, system)
		}
		return map[string][]string{
			"arch": {a},
			"os":   archOses,
		}, matcherSystems, remainingSystems
	}
	o := oses[0]
	var osArches []string
	for _, system := range systems {
		if system.OS() == o {
			osArches = append(osArches, system.Arch())
		}
	}
	s, r := splitSystems(systems, func(system bindown.System) bool {
		return system.OS() == o
	})
	return map[string][]string{
		"os":   {o},
		"arch": osArches,
	}, s, r
}

func QueryGitHubRelease(ctx context.Context, repo, tag, tkn string) (urls []string, version, homepage, description string, _ error) {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: tkn},
	)))
	splitRepo := strings.Split(repo, "/")
	orgName, repoName := splitRepo[0], splitRepo[1]
	repoResp, _, err := client.Repositories.Get(ctx, orgName, repoName)
	if err != nil {
		return nil, "", "", "", err
	}
	description = repoResp.GetDescription()
	homepage = repoResp.GetHomepage()
	if homepage == "" {
		homepage = repoResp.GetHTMLURL()
	}
	var release *github.RepositoryRelease
	if tag == "" {
		release, _, err = client.Repositories.GetLatestRelease(ctx, orgName, repoName)
		if err != nil {
			return nil, "", "", "", err
		}
		tag = release.GetTagName()
	} else {
		release, _, err = client.Repositories.GetReleaseByTag(ctx, orgName, repoName, tag)
		if err != nil {
			return nil, "", "", "", err
		}
	}

	if version == "" {
		version = tag
		if strings.HasPrefix(version, "v") {
			_, err = semver.NewVersion(version[1:])
			if err == nil {
				version = version[1:]
			}
		}
	}
	for _, asset := range release.Assets {
		urls = append(urls, asset.GetBrowserDownloadURL())
	}
	return urls, version, homepage, description, nil
}
