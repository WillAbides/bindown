package bindown

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type DependencyOverride struct {
	// Limits the override to configurations matching all of the matchers. Keys may be "os", "arch" or any variable name.
	// Values are an array of values to match. Any matching value will match. If a value can be interpreted as a
	// semantic version it will be treated as such.
	OverrideMatcher map[string][]string `json:"matcher" yaml:"matcher,omitempty"`

	// Values to override the parent dependency
	Dependency Dependency `json:"dependency" yaml:",omitempty"`
}

type Dependency struct {
	// The homepage for this dependency. Informational only.
	Homepage *string `json:"homepage,omitempty" yaml:",omitempty"`

	// A description of the dependency. Informational only.
	Description *string `json:"description,omitempty" yaml:",omitempty"`

	// A template for this dependency. Any unset fields in this dependency will be set by values from the template.
	// Overrides in the dependency and its template are concatenated with the template's overrides coming first.
	// Vars and substitutions are both combined with the dependency's value taking precedence.
	Template *string `json:"template,omitempty" yaml:",omitempty"`

	// The url to download a dependency from.
	URL *string `json:"url,omitempty" yaml:",omitempty"`

	// The path in the downloaded archive where the binary is located. Default is ./<bin>
	ArchivePath *string `json:"archive_path,omitempty" yaml:"archive_path,omitempty"`

	// The name of the binary to be installed. Default is the name of the dependency.
	BinName *string `json:"bin,omitempty" yaml:"bin,omitempty"`

	// Whether to create a symlink to the bin instead of copying it.
	Link *bool `json:"link,omitempty" yaml:",omitempty"`

	// A list of variables that can be used in 'url', 'archive_path' and 'bin'.
	//
	// Two variables are always added based on the current environment: 'os' and 'arch'. Those are the operating
	// system and architecture as defined by go's GOOS and GOARCH variables. I should document what those are
	// somewhere.
	//
	// You can reference a variable using golang template syntax. For example, you could have a url set to
	// `https://example.org/mydependency/v{{.version}}/mydependency-{{.os}}-{{.arch}}.tar.gz`.  If you define the var
	// 'version: 1.2.3' and run bindown on a 64 bit Linux system, it will download
	// `https://example.org/mydependency/v1.2.3/mydependency-linux-amd64.tar.gz`.
	Vars map[string]string `json:"vars,omitempty" yaml:",omitempty"`

	// List of systems this dependency supports. Systems are in the form of os/architecture.
	Systems []System `json:"systems,omitempty" yaml:"systems,omitempty"`

	// A list of variables that must be present for an install to succeed
	RequiredVars []string `json:"required_vars,omitempty" yaml:"required_vars,omitempty"`

	// Overrides allows you to override values depending on the os and architecture of the target system.
	Overrides []DependencyOverride `json:"overrides,omitempty" yaml:",omitempty"`

	// Substitutions will substitute values from vars. The key is the name of the variable to substitute. The value is
	// a map of substitutions. { "os": { "linux": "Linux", "darwin": "MacOS" } } is an example of a substitution that
	// will update the os variable.
	Substitutions map[string]map[string]string `json:"substitutions,omitempty" yaml:",omitempty"`

	built    bool
	name     string
	checksum string
	url      string
	system   System
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

func (d *Dependency) clone() *Dependency {
	overrides := slices.Clone(d.Overrides)
	for i, override := range overrides {
		matcher := maps.Clone(override.OverrideMatcher)
		for k, v := range matcher {
			matcher[k] = slices.Clone(v)
		}
		overrides[i] = DependencyOverride{
			OverrideMatcher: matcher,
			Dependency:      *(override.Dependency.clone()),
		}
	}
	return &Dependency{
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
	newDL := tmpl.clone()
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
	newDL.Systems = slices.Clone(newDL.Systems)

	if len(d.Overrides) > 0 {
		newDL.Overrides = append(newDL.Overrides, d.Overrides...)
	}
	*d = *newDL
	return nil
}

const maxOverrideDepth = 2

func (d *Dependency) applyOverrides(system System, depth int) {
	for i := range d.Overrides {
		systemVars := maps.Clone(d.Vars)
		if systemVars == nil {
			systemVars = make(map[string]string)
		}
		if _, ok := systemVars["os"]; !ok {
			systemVars["os"] = system.OS()
		}
		if _, ok := systemVars["arch"]; !ok {
			systemVars["arch"] = system.Arch()
		}
		match := !slices.ContainsFunc(maps.Keys(d.Overrides[i].OverrideMatcher), func(varName string) bool {
			overridePatterns := d.Overrides[i].OverrideMatcher[varName]
			val := systemVars[varName]
			// A match is found if the value is an exact match for a pattern or if the
			// pattern is a valid semver constraint and the value is a valid semver that
			// satisfies the constraint.
			matcher := func(pattern string) bool {
				if pattern == val {
					return true
				}
				constraint, err := semver.NewConstraint(pattern)
				if err != nil {
					return false
				}
				version, err := semver.NewVersion(val)
				if err != nil {
					return false
				}
				return constraint.Check(version)
			}
			return !slices.ContainsFunc(overridePatterns, matcher)
		})
		if !match {
			continue
		}
		dependency := &d.Overrides[i].Dependency
		if depth <= maxOverrideDepth {
			dependency.applyOverrides(system, depth+1)
		}
		for subType, mp := range dependency.Substitutions {
			if d.Substitutions == nil {
				d.Substitutions = make(map[string]map[string]string)
			}
			if d.Substitutions[subType] == nil {
				d.Substitutions[subType] = make(map[string]string)
			}
			for k, v := range mp {
				d.Substitutions[subType][k] = v
			}
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
