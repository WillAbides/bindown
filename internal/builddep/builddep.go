package builddep

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v52/github"
	"github.com/mholt/archiver/v4"
	"github.com/willabides/bindown/v3"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

//go:generate sh -c "go tool dist list > go_dist_list.txt"

//go:embed go_dist_list.txt
var _goDists string

var forbiddenOS = map[string]bool{
	"js": true,
}

var forbiddenArch = map[string]bool{
	"arm":  true,
	"wasm": true,
}

func distSystems() []bindown.SystemInfo {
	var dists []bindown.SystemInfo
	for _, s := range strings.Split(strings.TrimSpace(_goDists), "\n") {
		parts := strings.Split(s, "/")
		dists = append(dists, bindown.SystemInfo{
			OS:   parts[0],
			Arch: parts[1],
		})
	}
	return dists
}

func AddDependency(ctx context.Context, cfg *bindown.Config, name, version string, urls []string) error {
	return addDependency(ctx, cfg, name, version, urls, nil)
}

func addDependency(ctx context.Context, cfg *bindown.Config, name, version string, urls []string, selector selectCandidateFunc) error {
	groups := parseDownloads(urls, name, version, cfg.Systems)
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
	err = built.Validate(nil, built.Systems)
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

type dlFile struct {
	origUrl      string
	url          string
	osSub        *systemSub
	archSub      *systemSub
	suffix       string
	isArchive    bool
	priority     int
	archiveFiles []*archiveFile
	checksum     string
}

func (f *dlFile) clone() *dlFile {
	clone := *f
	clone.archiveFiles = slices.Clone(f.archiveFiles)
	for i, file := range f.archiveFiles {
		cf := *file
		clone.archiveFiles[i] = &cf
	}
	osSub := *f.osSub
	clone.osSub = &osSub
	archSub := *f.archSub
	clone.archSub = &archSub
	return &clone
}

func (f *dlFile) setArchiveFiles(ctx context.Context, binName, version string) error {
	if !f.isArchive {
		return nil
	}
	parsedUrl, err := url.Parse(f.origUrl)
	if err != nil {
		return err
	}
	filename := path.Base(parsedUrl.EscapedPath())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.origUrl, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		//nolint:errcheck // ignore error
		_ = resp.Body.Close()
	}()
	hasher := sha256.New()
	reader := io.TeeReader(resp.Body, hasher)
	format, reader, err := archiver.Identify(filename, reader)
	if err != nil {
		if errors.Is(err, archiver.ErrNoMatch) {
			err = fmt.Errorf("unable to identify archive format for %s", filename)
		}
		return err
	}
	// reader needs to be an io.ReaderAt and io.Seeker for zip
	_, isZip := format.(archiver.Zip)
	if isZip {
		var b []byte
		b, err = io.ReadAll(reader)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	extractor, ok := format.(archiver.Extractor)
	if !ok {
		return errors.New("format does not support extraction")
	}
	err = extractor.Extract(ctx, reader, nil, func(_ context.Context, af archiver.File) error {
		if af.IsDir() {
			return nil
		}
		executable := af.Mode().Perm()&0o100 != 0
		if !executable && f.osSub.normalized == "windows" {
			executable = strings.HasSuffix(af.Name(), ".exe")
		}
		f.archiveFiles = append(f.archiveFiles, parseArchiveFile(af.NameInArchive, binName, f.osSub.val, f.archSub.val, version, executable))
		return nil
	})
	if err != nil {
		return err
	}
	slices.SortFunc(f.archiveFiles, archiveFileLess)
	// read remaining bytes to calculate hash
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return err
	}
	f.checksum = hex.EncodeToString(hasher.Sum(nil))
	return err
}

func (f *dlFile) system() bindown.SystemInfo {
	if f.osSub == nil || f.archSub == nil {
		panic("system called on dlFile without osSub or archSub")
	}
	return bindown.SystemInfo{
		OS:   f.osSub.normalized,
		Arch: f.archSub.normalized,
	}
}

type archiveFile struct {
	origName    string
	name        string
	suffix      string
	tmplCount   int
	executable  bool
	containsBin bool
}

// archiveFileLess puts executables first,
// then containsBin,
// then the most templated files,
// then the shortest path, then alphabetically
func archiveFileLess(a, b *archiveFile) bool {
	if a.executable != b.executable {
		return a.executable
	}
	if a.containsBin != b.containsBin {
		return a.containsBin
	}
	fTmpls := strings.Count(a.name, "{{")
	otherTmpls := strings.Count(b.name, "{{")
	if fTmpls != otherTmpls {
		return fTmpls > otherTmpls
	}
	fSlashes := strings.Count(a.origName, "/")
	otherSlashes := strings.Count(b.origName, "/")
	if fSlashes != otherSlashes {
		return fSlashes < otherSlashes
	}
	return a.origName < b.origName
}

// archiveFileGroupable returns true if a and b can be in the same top-level dependency
func archiveFileGroupable(a, b *archiveFile) bool {
	return a.name == b.name && a.suffix == b.suffix
}

func osSubs(systems []bindown.SystemInfo) []systemSub {
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
		distOS := dist.OS
		if !slices.ContainsFunc(subs, func(sub systemSub) bool {
			return sub.val == distOS
		}) {
			subs = append(subs, systemSub{val: distOS, normalized: distOS})
		}
	}
	slices.SortFunc(subs, func(a, b systemSub) bool {
		return len(a.val) > len(b.val)
	})
	return subs
}

func archSubs(systems []bindown.SystemInfo) []systemSub {
	subs := []systemSub{
		{val: "amd64", normalized: "amd64"},
		{val: "arm64", normalized: "arm64"},
		{val: "x86_64", normalized: "amd64"},
		{val: "x86", normalized: "386"},
		{val: "x64", normalized: "amd64"},
		{val: "64bit", normalized: "amd64"},
		{val: "64-bit", normalized: "amd64"},
		{val: "aarch64", normalized: "arm64"},
		{val: "i386", normalized: "386"},
	}
	if systems == nil {
		systems = distSystems()
	}
	for _, dist := range systems {
		distArch := dist.Arch
		if !slices.ContainsFunc(subs, func(sub systemSub) bool {
			return sub.val == distArch
		}) {
			subs = append(subs, systemSub{val: distArch, normalized: distArch})
		}
	}
	slices.SortFunc(subs, func(a, b systemSub) bool {
		return len(a.val) > len(b.val)
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

func parseOs(filename string, systems []bindown.SystemInfo) *systemSub {
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

func parseArch(filename string, systems []bindown.SystemInfo) *systemSub {
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
	".br",
	".gz",
	".bz2",
	".lz4",
	".sz",
	".xz",
	".zst",
}

func parseDownload(dlURL, version string, systems []bindown.SystemInfo) (*dlFile, bool) {
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
	if !slices.ContainsFunc(systems, func(sys bindown.SystemInfo) bool {
		return sys.OS == osSub.normalized && sys.Arch == archSub.normalized
	}) {
		return nil, false
	}
	isArchive := false
	suffix := ""
	for _, s := range archiveSuffixes {
		if strings.HasSuffix(dlURL, s) {
			suffix = s
			isArchive = true
			break
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
		origUrl:   dlURL,
		url:       tmpl,
		osSub:     osSub,
		archSub:   archSub,
		suffix:    suffix,
		isArchive: isArchive,
		priority:  priority,
	}, true
}

func parseArchiveFile(origName, binName, osName, archName, version string, executable bool) *archiveFile {
	a := archiveFile{
		origName:    origName,
		name:        origName,
		executable:  executable,
		containsBin: strings.Contains(origName, binName),
	}
	for {
		idx := strings.Index(a.name, osName)
		if idx == -1 {
			break
		}
		a.tmplCount++
		a.name = a.name[:idx] + "{{.os}}" + a.name[idx+len(osName):]
	}
	for {
		idx := strings.Index(a.name, archName)
		if idx == -1 {
			break
		}
		a.tmplCount++
		a.name = a.name[:idx] + "{{.arch}}" + a.name[idx+len(archName):]
	}
	for {
		idx := strings.Index(a.name, version)
		if idx == -1 {
			break
		}
		a.tmplCount++
		a.name = a.name[:idx] + "{{.version}}" + a.name[idx+len(version):]
	}
	// .exe is the only suffix we care about
	if strings.HasSuffix(a.name, ".exe") {
		a.suffix = ".exe"
		a.name = a.name[:len(a.name)-4]
	}
	a.name += "{{.archivePathSuffix}}"
	return &a
}

func parseDownloads(dlUrls []string, binName, version string, allowedSystems []bindown.SystemInfo) []*depGroup {
	systemFiles := map[string][]*dlFile{}
	for _, dlUrl := range dlUrls {
		f, ok := parseDownload(dlUrl, version, allowedSystems)
		if !ok {
			continue
		}
		dlSystem := f.system()
		systemFiles[dlSystem.String()] = append(systemFiles[dlSystem.String()], f)
	}
	for system := range systemFiles {
		if len(systemFiles[system]) < 2 {
			continue
		}
		// remove all but the highest priority
		slices.SortFunc(systemFiles[system], func(a, b *dlFile) bool {
			return a.priority > b.priority
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
		slices.SortFunc(systemFiles[system], func(a, b *dlFile) bool {
			return urlFrequency[a.url] > urlFrequency[b.url]
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
		// prefer archives
		slices.SortFunc(systemFiles[system], func(a, b *dlFile) bool {
			return a.isArchive && !b.isArchive
		})
		cutOff = slices.IndexFunc(systemFiles[system], func(f *dlFile) bool {
			return !f.isArchive
		})
		if cutOff != -1 {
			systemFiles[system] = systemFiles[system][:cutOff]
		}
		if len(systemFiles[system]) == 1 {
			continue
		}
		// now arbitrarily pick the first one alphabetically by origUrl
		slices.SortFunc(systemFiles[system], func(a, b *dlFile) bool {
			return a.origUrl < b.origUrl
		})
		systemFiles[system] = systemFiles[system][:1]
	}

	templates := maps.Keys(urlFrequency)
	slices.SortFunc(templates, func(a, b string) bool {
		return urlFrequency[a] > urlFrequency[b]
	})

	var groups []*depGroup
	systems := maps.Keys(systemFiles)
	slices.Sort(systems)
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
	slices.SortFunc(groups, func(a, b *depGroup) bool {
		return len(a.files) > len(b.files)
	})
	return groups
}

func systemLess(a, b bindown.SystemInfo) bool {
	if a.OS != b.OS {
		return a.OS < b.OS
	}
	return a.Arch < b.Arch
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
	slices.SortFunc(dep.Systems, systemLess)
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
				Vars: map[string]string{
					"version": version,
				},
			},
		},
		Templates: map[string]*bindown.Dependency{
			name: dep,
		},
		URLChecksums: checksums,
	}
}

type depGroup struct {
	urlSuffix         string
	url               string
	archivePathSuffix string
	archivePath       string
	binName           string
	systems           []bindown.SystemInfo
	files             []*dlFile
	substitutions     map[string]map[string]string
	overrideMatcher   map[string][]string
}

func (g *depGroup) clone() *depGroup {
	clone := *g
	clone.files = slices.Clone(g.files)
	for i := range clone.files {
		clone.files[i] = clone.files[i].clone()
	}
	clone.substitutions = map[string]map[string]string{}
	for k, v := range g.substitutions {
		clone.substitutions[k] = map[string]string{}
		for k2, v2 := range v {
			clone.substitutions[k][k2] = v2
		}
	}
	clone.overrideMatcher = map[string][]string{}
	for k, v := range g.overrideMatcher {
		clone.overrideMatcher[k] = slices.Clone(v)
	}
	clone.systems = slices.Clone(g.systems)
	return &clone
}

type archiveFileCandidate struct {
	archiveFile *archiveFile
	matches     []*dlFile
	nonMatches  []*dlFile
}

type selectCandidateFunc func([]*archiveFileCandidate, *archiveFileCandidate) error

func defaultSelectCandidateFunc(candidates []*archiveFileCandidate, candidate *archiveFileCandidate) error {
	options := make([]string, len(candidates))
	optionsMap := map[string]*archiveFileCandidate{}
	for i := range candidates {
		text := fmt.Sprintf("%s - (%s)", candidates[i].archiveFile.name, candidates[i].archiveFile.origName)
		options[i] = text
		optionsMap[text] = candidates[i]
	}
	var choice string
	err := survey.AskOne(&survey.Select{
		Message: "Select the correct archive file",
		Options: options,
	}, &choice)
	if err != nil {
		return err
	}
	*candidate = *optionsMap[choice]
	return nil
}

func (g *depGroup) regroupByArchivePath(ctx context.Context, binName, version string, selectCandidate selectCandidateFunc) ([]*depGroup, error) {
	gr := g.clone()
	if len(gr.files) == 0 {
		return []*depGroup{gr}, nil
	}
	// trust that if the first isn't an archive, none of them are
	if !gr.files[0].isArchive {
		gr.archivePath = path.Base(gr.files[0].url)
		return []*depGroup{gr}, nil
	}
	errGroup, ctx := errgroup.WithContext(ctx)
	for i := range gr.files {
		i := i
		if gr.files[i].archiveFiles != nil {
			continue
		}
		errGroup.Go(func() error {
			err := gr.files[i].setArchiveFiles(ctx, binName, version)
			if err != nil {
				return err
			}
			if len(gr.files[i].archiveFiles) == 0 {
				return fmt.Errorf("no archive files found for %s", gr.files[i].origUrl)
			}
			return nil
		})
	}
	err := errGroup.Wait()
	if err != nil {
		return nil, err
	}

	var candidates []*archiveFileCandidate

	for i := range gr.files[0].archiveFiles {
		c := archiveFileCandidate{
			archiveFile: gr.files[0].archiveFiles[i],
		}
		for _, df := range gr.files {
			match := slices.ContainsFunc(df.archiveFiles, func(af *archiveFile) bool {
				return archiveFileGroupable(c.archiveFile, af)
			})
			if match {
				c.matches = append(c.matches, df)
				continue
			}
			c.nonMatches = append(c.nonMatches, df)
		}
		candidates = append(candidates, &c)
	}

	var selectedCandidate archiveFileCandidate
	if selectCandidate == nil {
		selectCandidate = defaultSelectCandidateFunc
	}
	err = selectCandidate(candidates, &selectedCandidate)
	if err != nil {
		return nil, err
	}

	nextGr := gr.clone()

	gr.archivePath = selectedCandidate.archiveFile.name
	gr.archivePathSuffix = selectedCandidate.archiveFile.suffix
	gr.files = selectedCandidate.matches
	groups := []*depGroup{gr}
	if len(selectedCandidate.nonMatches) == 0 {
		return groups, nil
	}
	gr.systems = gr.systems[:0]
	for _, f := range gr.files {
		gr.systems = append(gr.systems, f.system())
	}
	nextGr.files = selectedCandidate.nonMatches
	nextGr.systems = nextGr.systems[:0]
	for _, f := range nextGr.files {
		nextGr.systems = append(nextGr.systems, f.system())
	}
	var moreGroups []*depGroup
	moreGroups, err = nextGr.regroupByArchivePath(ctx, binName, version, selectCandidate)
	if err != nil {
		return nil, err
	}
	groups = append(groups, moreGroups...)
	return groups, nil
}

func (g *depGroup) dependency() *bindown.Dependency {
	dep := bindown.Dependency{
		URL:          &g.url,
		BinName:      &g.binName,
		ArchivePath:  &g.archivePath,
		RequiredVars: []string{"version"},
		Vars: map[string]string{
			"urlSuffix":         g.urlSuffix,
			"archivePathSuffix": g.archivePathSuffix,
		},
		Substitutions: map[string]map[string]string{},
		Systems:       g.systems,
	}
	if g.substitutions != nil {
		if len(g.substitutions["os"]) > 0 {
			dep.Substitutions["os"] = maps.Clone(g.substitutions["os"])
		}
		if len(g.substitutions["arch"]) > 0 {
			dep.Substitutions["arch"] = maps.Clone(g.substitutions["arch"])
		}
	}
	slices.SortFunc(dep.Systems, func(a, b bindown.SystemInfo) bool {
		return a.String() < b.String()
	})
	return &dep
}

func (g *depGroup) overrides(otherGroups []*depGroup) []bindown.DependencyOverride {
	dep0 := otherGroups[0].dependency()
	var overrides []bindown.DependencyOverride
	for _, m := range g.matchers(otherGroups) {
		dep := g.dependency()
		dep.Systems = nil
		dep.RequiredVars = nil
		for k, v := range dep.Vars {
			if dep0.Vars[k] == v {
				delete(dep.Vars, k)
			}
		}
		if *dep0.URL == *dep.URL {
			dep.URL = nil
		}
		if *dep0.ArchivePath == *dep.ArchivePath {
			dep.ArchivePath = nil
		}
		if *dep0.BinName == *dep.BinName {
			dep.BinName = nil
		}
		matcher := m.matcher
		systems := m.systems
		for normalized := range dep.Substitutions["os"] {
			if !slices.ContainsFunc(systems, func(s bindown.SystemInfo) bool {
				return s.OS == normalized
			}) {
				delete(dep.Substitutions["os"], normalized)
			}
		}
		overrides = append(overrides, bindown.DependencyOverride{
			OverrideMatcher: matcher,
			Dependency:      *dep,
		})
	}
	return overrides
}

func splitSystems(systems []bindown.SystemInfo, fn func(s bindown.SystemInfo) bool) (matching, nonMatching []bindown.SystemInfo) {
	for _, system := range systems {
		if fn(system) {
			matching = append(matching, system)
		} else {
			nonMatching = append(nonMatching, system)
		}
	}
	return
}

func systemsMatcher(systems, otherSystems []bindown.SystemInfo) (_ map[string][]string, matcherSystems, remainingSystems []bindown.SystemInfo) {
	var oses, arches, otherOses, otherArches, exclusiveOses, exclusiveArches []string
	for _, system := range systems {
		if !slices.Contains(oses, system.OS) {
			oses = append(oses, system.OS)
		}
		if !slices.Contains(arches, system.Arch) {
			arches = append(arches, system.Arch)
		}
	}
	for _, system := range otherSystems {
		if !slices.Contains(otherOses, system.OS) {
			otherOses = append(otherOses, system.OS)
		}
		if !slices.Contains(otherArches, system.Arch) {
			otherArches = append(otherArches, system.Arch)
		}
	}
	for _, s := range oses {
		if !slices.Contains(otherOses, s) {
			exclusiveOses = append(exclusiveOses, s)
		}
	}
	if len(exclusiveOses) > 0 {
		s, r := splitSystems(systems, func(system bindown.SystemInfo) bool {
			return slices.Contains(exclusiveOses, system.OS)
		})
		return map[string][]string{"os": exclusiveOses}, s, r
	}
	for _, s := range arches {
		if !slices.Contains(otherArches, s) {
			exclusiveArches = append(exclusiveArches, s)
		}
	}
	if len(exclusiveArches) > 0 {
		s, r := splitSystems(systems, func(system bindown.SystemInfo) bool {
			return slices.Contains(exclusiveArches, system.Arch)
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
			if system.Arch == a {
				matcherSystems = append(matcherSystems, system)
				archOses = append(archOses, system.OS)
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
		if system.OS == o {
			osArches = append(osArches, system.Arch)
		}
	}
	s, r := splitSystems(systems, func(system bindown.SystemInfo) bool {
		return system.OS == o
	})
	return map[string][]string{
		"os":   {o},
		"arch": osArches,
	}, s, r
}

func (g *depGroup) matchers(otherGroups []*depGroup) (result []struct {
	matcher map[string][]string
	systems []bindown.SystemInfo
},
) {
	var otherSystems []bindown.SystemInfo
	for _, other := range otherGroups {
		otherSystems = append(otherSystems, other.systems...)
	}
	systems := slices.Clone(g.systems)
	for len(systems) > 0 {
		r := struct {
			matcher map[string][]string
			systems []bindown.SystemInfo
		}{}
		r.matcher, r.systems, systems = systemsMatcher(systems, otherSystems)
		result = append(result, r)
	}
	return result
}

func (g *depGroup) addFile(f *dlFile, binName string) {
	g.url = f.url
	g.binName = binName
	g.urlSuffix = f.suffix
	g.systems = append(g.systems, f.system())
	g.files = append(g.files, f)
	g.substitutions["os"][f.osSub.normalized] = f.osSub.val
	g.substitutions["arch"][f.archSub.normalized] = f.archSub.val
	if !slices.Contains(g.overrideMatcher["os"], f.osSub.normalized) {
		g.overrideMatcher["os"] = append(g.overrideMatcher["os"], f.osSub.normalized)
	}
	if !slices.Contains(g.overrideMatcher["arch"], f.archSub.normalized) {
		g.overrideMatcher["arch"] = append(g.overrideMatcher["arch"], f.archSub.normalized)
	}
}

func (g *depGroup) fileAllowed(f *dlFile, binName string) bool {
	if f.suffix != g.urlSuffix ||
		f.url != g.url ||
		binName != g.binName {
		return false
	}
	subVal := g.substitutions["os"][f.osSub.normalized]
	if subVal != "" && subVal != f.osSub.val {
		return false
	}
	subVal = g.substitutions["arch"][f.archSub.normalized]
	if subVal != "" && subVal != f.archSub.val {
		return false
	}

	return true
}

func QueryGitHubRelease(ctx context.Context, repo, tag, tkn string) (urls []string, version string, _ error) {
	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: tkn},
	)))
	splitRepo := strings.Split(repo, "/")
	orgName, repoName := splitRepo[0], splitRepo[1]
	var release *github.RepositoryRelease
	var err error
	if tag == "" {
		release, _, err = client.Repositories.GetLatestRelease(ctx, orgName, repoName)
		if err != nil {
			return nil, "", err
		}
		tag = release.GetTagName()
	} else {
		release, _, err = client.Repositories.GetReleaseByTag(ctx, orgName, repoName, tag)
		if err != nil {
			return nil, "", err
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
	return urls, version, nil
}
