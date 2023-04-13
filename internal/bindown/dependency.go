package bindown

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// DependencyOverride overrides a dependency's configuration
type DependencyOverride struct {
	OverrideMatcher OverrideMatcher `json:"matcher" yaml:"matcher,omitempty"`
	Dependency      Dependency      `json:"dependency" yaml:",omitempty"`
}

func (o *DependencyOverride) Clone() *DependencyOverride {
	return &DependencyOverride{
		Dependency:      *(o.Dependency.Clone()),
		OverrideMatcher: o.OverrideMatcher.Clone(),
	}
}

// OverrideMatcher contains a list or oses and arches to match an override. If either os or arch is empty, all oses and arches match.
type OverrideMatcher map[string][]string

func (o OverrideMatcher) matches(system System, vars map[string]string) bool {
	if vars == nil {
		vars = map[string]string{}
	}
	for varName, patterns := range o {
		val, ok := vars[varName]
		if !ok {
			if varName == "os" {
				val = system.OS()
			}
			if varName == "arch" {
				val = system.Arch()
			}
		}
		match := false
		for _, pattern := range patterns {
			if pattern == val {
				match = true
				break
			}
			// If pattern can be parsed as a semver constraint and val can be parsed as a semver version, check if val meets the constraint
			constraint, err := semver.NewConstraint(pattern)
			if err != nil {
				continue
			}
			version, err := semver.NewVersion(val)
			if err != nil {
				continue
			}
			if constraint.Check(version) {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	return true
}

func (o OverrideMatcher) Clone() OverrideMatcher {
	clone := maps.Clone(o)
	for i := range clone {
		clone[i] = slices.Clone(clone[i])
	}
	return clone
}

// Dependency is something to download, extract and install
type Dependency struct {
	Template      *string                      `json:"template,omitempty" yaml:",omitempty"`
	URL           *string                      `json:"url,omitempty" yaml:",omitempty"`
	ArchivePath   *string                      `json:"archive_path,omitempty" yaml:"archive_path,omitempty"`
	BinName       *string                      `json:"bin,omitempty" yaml:"bin,omitempty"`
	Link          *bool                        `json:"link,omitempty" yaml:",omitempty"`
	Vars          map[string]string            `json:"vars,omitempty" yaml:",omitempty"`
	RequiredVars  []string                     `json:"required_vars,omitempty" yaml:"required_vars,omitempty"`
	Overrides     []DependencyOverride         `json:"overrides,omitempty" yaml:",omitempty"`
	Substitutions map[string]map[string]string `json:"substitutions,omitempty" yaml:",omitempty"`
	Systems       []SystemInfo                 `json:"systems,omitempty" yaml:"systems,omitempty"`

	built    bool
	name     string
	checksum string
	url      string
	system   SystemInfo
}

func cloneSubstitutions(subs map[string]map[string]string) map[string]map[string]string {
	clone := maps.Clone(subs)
	for k, v := range clone {
		clone[k] = maps.Clone(v)
	}
	return clone
}

func varsWithSubstitutions(vars map[string]string, subs map[string]map[string]string) map[string]string {
	if vars == nil || subs == nil {
		return vars
	}
	vars = maps.Clone(vars)
	for key, val := range vars {
		if subs[key] == nil || subs[key][val] == "" {
			continue
		}
		vars[key] = subs[key][val]
	}
	return vars
}

func (d *Dependency) Clone() *Dependency {
	dep := Dependency{
		Vars:          maps.Clone(d.Vars),
		URL:           clonePointer(d.URL),
		ArchivePath:   clonePointer(d.ArchivePath),
		Template:      clonePointer(d.Template),
		BinName:       clonePointer(d.BinName),
		Link:          clonePointer(d.Link),
		Overrides:     slices.Clone(d.Overrides),
		Substitutions: cloneSubstitutions(d.Substitutions),
		Systems:       slices.Clone(d.Systems),
		RequiredVars:  slices.Clone(d.RequiredVars),
	}
	for i, override := range dep.Overrides {
		dep.Overrides[i] = *override.Clone()
	}
	return &dep
}

// interpolateVars executes go templates in values
func (d *Dependency) interpolateVars(system System) error {
	for _, p := range []*string{d.URL, d.ArchivePath, d.BinName} {
		if p == nil {
			continue
		}
		var err error
		*p, err = executeTemplate(*p, system.OS(), system.Arch(), d.Vars)
		if err != nil {
			return err
		}
	}
	return nil
}

const maxTemplateDepth = 2

func (d *Dependency) applyTemplate(templates map[string]*Dependency, depth int) error {
	if depth > maxTemplateDepth {
		return nil
	}
	templateName := d.Template
	if templateName == nil || *templateName == "" {
		return nil
	}
	if templates == nil {
		templates = map[string]*Dependency{}
	}
	tmpl, ok := templates[*templateName]
	if !ok {
		return fmt.Errorf("no template named %s", *templateName)
	}
	newDL := tmpl.Clone()
	err := newDL.applyTemplate(templates, depth+1)
	if err != nil {
		return err
	}
	newDL.Template = d.Template
	if newDL.Vars == nil && d.Vars != nil {
		newDL.Vars = make(map[string]string, len(d.Vars))
	}
	maps.Copy(newDL.Vars, d.Vars)
	newDL.ArchivePath = overrideValue(newDL.ArchivePath, d.ArchivePath)
	newDL.BinName = overrideValue(newDL.BinName, d.BinName)
	newDL.URL = overrideValue(newDL.URL, d.URL)
	newDL.Link = overrideValue(newDL.Link, d.Link)
	if d.RequiredVars != nil {
		newDL.RequiredVars = append(newDL.RequiredVars, d.RequiredVars...)
	}
	newDL.Systems = slices.Clone(d.Systems)
	newDL.addOverrides(d.Overrides)
	*d = *newDL
	return nil
}

func (d *Dependency) addOverrides(overrides []DependencyOverride) {
	if len(overrides) == 0 {
		return
	}
	if d.Overrides == nil {
		d.Overrides = make([]DependencyOverride, 0, len(overrides))
	}
	for i := range overrides {
		d.Overrides = append(d.Overrides, *overrides[i].Clone())
	}
}

const maxOverrideDepth = 2

func (d *Dependency) applyOverrides(system System, depth int) {
	for i := range d.Overrides {
		if !d.Overrides[i].OverrideMatcher.matches(system, d.Vars) {
			continue
		}
		dependency := &d.Overrides[i].Dependency
		if depth <= maxOverrideDepth {
			dependency.applyOverrides(system, depth+1)
		}
		d.Link = overrideValue(d.Link, dependency.Link)
		d.ArchivePath = overrideValue(d.ArchivePath, dependency.ArchivePath)
		d.BinName = overrideValue(d.BinName, dependency.BinName)
		d.URL = overrideValue(d.URL, dependency.URL)
		maps.Copy(d.Vars, dependency.Vars)
	}
	d.Overrides = nil
}

func linkBin(link, src string) error {
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(link), 0o750)
	if err != nil {
		return err
	}
	var linkDir string
	linkDir, err = filepath.Abs(filepath.Dir(link))
	if err != nil {
		return err
	}

	linkDir, err = filepath.EvalSymlinks(linkDir)
	if err != nil {
		return err
	}

	absSrc, err = filepath.EvalSymlinks(absSrc)
	if err != nil {
		return err
	}

	dst, err := filepath.Rel(linkDir, absSrc)
	if err != nil {
		return err
	}
	err = os.RemoveAll(link)
	if err != nil {
		return err
	}
	err = os.Symlink(dst, link)
	if err != nil {
		return err
	}
	info, err := os.Stat(link)
	if err != nil {
		return err
	}
	return os.Chmod(link, info.Mode().Perm()|0o750)
}
