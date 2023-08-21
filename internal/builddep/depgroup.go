package builddep

import (
	"context"
	"fmt"
	"maps"
	"path"

	"github.com/willabides/bindown/v4/internal/bindown"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
)

type depGroup struct {
	urlSuffix         string
	url               string
	archivePathSuffix string
	archivePath       string
	binName           string
	systems           []bindown.System
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

func (g *depGroup) regroupByArchivePath(ctx context.Context, binName, version string, selectCandidate selectCandidateFunc) ([]*depGroup, error) {
	gr := g.clone()
	if len(gr.files) == 0 {
		return []*depGroup{gr}, nil
	}
	// trust that if the first isn't an archive, none of them are
	if !gr.files[0].isArchive && !gr.files[0].isCompress {
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
		Overrideable: bindown.Overrideable{
			URL:         &g.url,
			ArchivePath: &g.archivePath,
			BinName:     &g.binName,
			Vars: map[string]string{
				"urlSuffix":         g.urlSuffix,
				"archivePathSuffix": g.archivePathSuffix,
			},
			Substitutions: map[string]map[string]string{},
		},
		RequiredVars: []string{"version"},
		Systems:      slices.Clone(g.systems),
	}
	if g.substitutions != nil {
		if len(g.substitutions["os"]) > 0 {
			dep.Substitutions["os"] = maps.Clone(g.substitutions["os"])
		}
		if len(g.substitutions["arch"]) > 0 {
			dep.Substitutions["arch"] = maps.Clone(g.substitutions["arch"])
		}
	}
	slices.Sort(dep.Systems)
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
			if !slices.ContainsFunc(systems, func(system bindown.System) bool {
				return system.OS() == normalized
			}) {
				delete(dep.Substitutions["os"], normalized)
			}
		}
		overrides = append(overrides, bindown.DependencyOverride{
			OverrideMatcher: matcher,
			Dependency:      dep.Overrideable,
		})
	}
	return overrides
}

func (g *depGroup) matchers(otherGroups []*depGroup) (result []struct {
	matcher map[string][]string
	systems []bindown.System
},
) {
	var otherSystems []bindown.System
	for _, other := range otherGroups {
		otherSystems = append(otherSystems, other.systems...)
	}
	systems := slices.Clone(g.systems)
	for len(systems) > 0 {
		r := struct {
			matcher map[string][]string
			systems []bindown.System
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
