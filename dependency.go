package bindown

import (
	"fmt"

	"github.com/willabides/bindown/v3/internal/util"
)

// DependencyOverride overrides a dependency's configuration
type DependencyOverride struct {
	OverrideMatcher OverrideMatcher `json:"matcher" yaml:"matcher,omitempty"`
	Dependency      Dependency      `json:"dependency" yaml:",omitempty"`
}

func (o *DependencyOverride) clone() *DependencyOverride {
	dl := o.Dependency.clone()
	return &DependencyOverride{
		Dependency:      *dl,
		OverrideMatcher: o.OverrideMatcher.clone(),
	}
}

// OverrideMatcher contains a list or oses and arches to match an override. If either os or arch is empty, all oses and arches match.
type OverrideMatcher struct {
	OS   []string `json:"os,omitempty" yaml:",omitempty"`
	Arch []string `json:"arch,omitempty" yaml:",omitempty"`
}

func (m OverrideMatcher) matches(info SystemInfo) bool {
	return m.archMatch(info.Arch) && m.osMatch(info.OS)
}

func (m OverrideMatcher) osMatch(os string) bool {
	if len(m.OS) == 0 {
		return true
	}
	return stringSliceContains(m.OS, os)
}

func (m OverrideMatcher) archMatch(arch string) bool {
	if len(m.Arch) == 0 {
		return true
	}
	return stringSliceContains(m.Arch, arch)
}

func (m OverrideMatcher) clone() OverrideMatcher {
	matcher := OverrideMatcher{
		OS:   make([]string, len(m.OS)),
		Arch: make([]string, len(m.Arch)),
	}
	copy(matcher.OS, m.OS)
	copy(matcher.Arch, m.Arch)
	return matcher
}

// Dependency is something to download, extract and install
type Dependency struct {
	Template      *string                      `json:"template,omitempty" yaml:",omitempty"`
	URL           *string                      `json:"url,omitempty" yaml:",omitempty"`
	ArchivePath   *string                      `json:"archive_path,omitempty" yaml:"archive_path,omitempty"`
	BinName       *string                      `json:"bin,omitempty" yaml:"bin,omitempty"`
	Link          *bool                        `json:"link,omitempty" yaml:",omitempty"`
	Vars          map[string]string            `json:"vars,omitempty" yaml:",omitempty"`
	Overrides     []DependencyOverride         `json:"overrides,omitempty" yaml:",omitempty"`
	Substitutions map[string]map[string]string `json:"substitutions,omitempty" yaml:",omitempty"`
}

func cloneSubstitutions(subs map[string]map[string]string) map[string]map[string]string {
	if subs == nil {
		return nil
	}
	result := make(map[string]map[string]string, len(subs))
	for k, v := range subs {
		result[k] = util.CopyStringMap(v)
	}
	return result
}

func varsWithSubstitutions(vars map[string]string, subs map[string]map[string]string) map[string]string {
	if vars == nil || subs == nil {
		return vars
	}
	vars = util.CopyStringMap(vars)
	for key, val := range vars {
		varSubs := subs[key]
		if varSubs == nil {
			continue
		}
		sub, ok := varSubs[val]
		if !ok {
			continue
		}
		vars[key] = sub
	}
	return vars
}

func (d *Dependency) clone() *Dependency {
	dep := *d
	if d.Vars != nil {
		dep.Vars = util.CopyStringMap(d.Vars)
	}
	if d.Overrides != nil {
		dep.Overrides = make([]DependencyOverride, len(d.Overrides))
		for i, override := range d.Overrides {
			dep.Overrides[i] = *override.clone()
		}
	}
	dep.Substitutions = cloneSubstitutions(d.Substitutions)
	return &dep
}

const maxTemplateDepth = 2

func (d *Dependency) applyTemplate(templates map[string]*Dependency, depth int) error {
	if depth > maxTemplateDepth {
		return nil
	}
	templateName := d.Template
	if templateName == nil || len(*templateName) == 0 {
		return nil
	}
	if templates == nil {
		templates = map[string]*Dependency{}
	}
	tmpl, ok := templates[*templateName]
	if !ok {
		return fmt.Errorf("no template named %s", *templateName)
	}
	newDL := tmpl.clone()
	err := newDL.applyTemplate(templates, depth+1)
	if err != nil {
		return err
	}
	newDL.Template = nil
	if newDL.Vars == nil && d.Vars != nil {
		newDL.Vars = make(map[string]string, len(d.Vars))
	}
	for k, v := range d.Vars {
		newDL.Vars[k] = v
	}
	if d.ArchivePath != nil {
		newDL.ArchivePath = d.ArchivePath
	}
	if d.BinName != nil {
		newDL.BinName = d.BinName
	}
	if d.URL != nil {
		newDL.URL = d.URL
	}
	if d.Link != nil {
		newDL.Link = d.Link
	}
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
	for _, override := range overrides {
		d.Overrides = append(d.Overrides, *override.clone())
	}
}

const maxOverrideDepth = 2

func (d *Dependency) applyOverrides(info SystemInfo, depth int) {
	for _, override := range d.Overrides {
		if !override.OverrideMatcher.matches(info) {
			continue
		}
		o := &override.Dependency
		if depth <= maxOverrideDepth {
			o.applyOverrides(info, depth+1)
		}
		if o.Link != nil {
			d.Link = o.Link
		}
		if d.Vars == nil {
			d.Vars = make(map[string]string, len(o.Vars))
		}
		for k, v := range o.Vars {
			d.Vars[k] = v
		}
		if o.ArchivePath != nil {
			d.ArchivePath = o.ArchivePath
		}
		if o.BinName != nil {
			d.BinName = o.BinName
		}
		if o.URL != nil {
			d.URL = o.URL
		}
	}
	d.Overrides = nil
}

func stringSliceContains(sl []string, item string) bool {
	for _, s := range sl {
		if s == item {
			return true
		}
	}
	return false
}

func boolPtr(val bool) *bool {
	return &val
}

func stringPtr(val string) *string {
	return &val
}
