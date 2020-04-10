package bindown

import (
	"fmt"
)

// DownloadableOverride overrides a downloadable
type DownloadableOverride struct {
	Downloadable        `yaml:",inline"`
	DownloadableMatcher `yaml:",inline"`
}

func (o *DownloadableOverride) clone() *DownloadableOverride {
	dl := o.Downloadable.clone()
	return &DownloadableOverride{
		Downloadable:        *dl,
		DownloadableMatcher: o.DownloadableMatcher.clone(),
	}
}

// DownloadableMatcher contains a list or oses and arches to match a downloadable override
// is either os or arch is empty, all oses and arches match.
type DownloadableMatcher struct {
	OS   []string `yaml:",omitempty"`
	Arch []string `yaml:",omitempty"`
}

func (m DownloadableMatcher) matches(info SystemInfo) bool {
	return m.archMatch(info.Arch) && m.osMatch(info.OS)
}

func (m DownloadableMatcher) osMatch(os string) bool {
	if len(m.OS) == 0 {
		return true
	}
	return stringSliceContains(m.OS, os)
}

func (m DownloadableMatcher) archMatch(arch string) bool {
	if len(m.Arch) == 0 {
		return true
	}
	return stringSliceContains(m.Arch, arch)
}

func (m DownloadableMatcher) clone() DownloadableMatcher {
	matcher := DownloadableMatcher{
		OS:   make([]string, len(m.OS)),
		Arch: make([]string, len(m.Arch)),
	}
	copy(matcher.OS, m.OS)
	copy(matcher.Arch, m.Arch)
	return matcher
}

// Downloadable defines how a downloader is built
type Downloadable struct {
	Template    *string                `yaml:",omitempty"`
	URL         *string                `yaml:",omitempty"`
	ArchivePath *string                `yaml:"archive_path,omitempty"`
	BinName     *string                `yaml:"bin,omitempty"`
	Link        *bool                  `yaml:",omitempty"`
	Vars        map[string]string      `yaml:"vars,omitempty"`
	Overrides   []DownloadableOverride `yaml:"overrides,omitempty"`
	KnownBuilds []SystemInfo           `yaml:"known_builds,omitempty"`
}

func (d *Downloadable) clone() *Downloadable {
	downloadable := *d
	if d.Vars != nil {
		downloadable.Vars = make(map[string]string, len(d.Vars))
		for k, v := range d.Vars {
			downloadable.Vars[k] = v
		}
	}
	if d.Overrides != nil {
		downloadable.Overrides = make([]DownloadableOverride, len(d.Overrides))
		for i, override := range d.Overrides {
			downloadable.Overrides[i] = *override.clone()
		}
	}
	if d.KnownBuilds != nil {
		downloadable.KnownBuilds = make([]SystemInfo, len(d.KnownBuilds))
		copy(downloadable.KnownBuilds, d.KnownBuilds)
	}
	return &downloadable
}

func (d *Downloadable) addKnownBuild(info SystemInfo) {
	if d.KnownBuilds == nil {
		d.KnownBuilds = make([]SystemInfo, 0, 1)
	}
	for _, kb := range d.KnownBuilds {
		if kb.equal(&info) {
			return
		}
	}
	d.KnownBuilds = append(d.KnownBuilds, info)
}

const maxTemplateDepth = 2

func (d *Downloadable) applyTemplate(templates map[string]*Downloadable, depth int) error {
	if depth > maxTemplateDepth {
		return nil
	}
	templateName := d.Template
	if templateName == nil || len(*templateName) == 0 {
		return nil
	}
	if templates == nil {
		templates = map[string]*Downloadable{}
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
	if d.KnownBuilds != nil {
		newDL.KnownBuilds = append(newDL.KnownBuilds, d.KnownBuilds...)
	}
	*d = *newDL
	return nil
}

func (d *Downloadable) addOverrides(overrides []DownloadableOverride) {
	if len(overrides) == 0 {
		return
	}
	if d.Overrides == nil {
		d.Overrides = make([]DownloadableOverride, 0, len(overrides))
	}
	for _, override := range overrides {
		d.Overrides = append(d.Overrides, *override.clone())
	}
}

const maxOverrideDepth = 2

func (d *Downloadable) applyOverrides(info SystemInfo, depth int) {
	for _, override := range d.Overrides {
		if !override.DownloadableMatcher.matches(info) {
			continue
		}
		o := &override.Downloadable
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
		if o.KnownBuilds != nil {
			d.KnownBuilds = append(d.KnownBuilds, o.KnownBuilds...)
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
