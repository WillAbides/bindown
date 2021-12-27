package bindown

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/mholt/archiver/v3"
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
	OS   []string
	Arch []string
	Vars map[string][]string
}

// MarshalJSON implements the json.Marshaler interface
func (o OverrideMatcher) MarshalJSON() ([]byte, error) {
	mp := make(map[string][]string, len(o.Vars)+2)
	for k, v := range o.Vars {
		mp[k] = v
	}
	if len(o.OS) > 0 {
		mp["os"] = o.OS
	}
	if len(o.Arch) > 0 {
		mp["arch"] = o.Arch
	}
	return json.Marshal(mp)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (o *OverrideMatcher) UnmarshalJSON(data []byte) error {
	var mp map[string][]string
	if err := json.Unmarshal(data, &mp); err != nil {
		return err
	}
	o.OS = mp["os"]
	delete(mp, "os")
	o.Arch = mp["arch"]
	delete(mp, "arch")
	o.Vars = make(map[string][]string, len(mp))
	for k, v := range mp {
		o.Vars[k] = v
	}
	return nil
}

// MarshalYAML implements the yaml.Marshaler interface
func (o OverrideMatcher) MarshalYAML() (interface{}, error) {
	mp := make(map[string][]string, len(o.Vars)+2)
	for k, v := range o.Vars {
		mp[k] = v
	}
	if len(o.OS) > 0 {
		mp["os"] = o.OS
	}
	if len(o.Arch) > 0 {
		mp["arch"] = o.Arch
	}
	return mp, nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (o *OverrideMatcher) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var mp map[string][]string
	if err := unmarshal(&mp); err != nil {
		return err
	}
	o.OS = mp["os"]
	delete(mp, "os")
	o.Arch = mp["arch"]
	delete(mp, "arch")
	o.Vars = make(map[string][]string, len(mp))
	for k, v := range mp {
		o.Vars[k] = v
	}
	return nil
}

func (o OverrideMatcher) matches(info SystemInfo, vars map[string]string) bool {
	excluded := func(patterns []string, val string) bool {
		for _, pattern := range patterns {
			if pattern == val {
				return false
			}
			// if pattern can be parsed as a semver constraint return true if val meets the constraint
			if constraint, err := semver.NewConstraint(pattern); err == nil {
				if version, err := semver.NewVersion(val); err == nil {
					if constraint.Check(version) {
						return false
					}
				}
			}
		}
		return true
	}
	if len(o.OS) > 0 && excluded(o.OS, info.OS) {
		return false
	}
	if len(o.Arch) > 0 && excluded(o.Arch, info.Arch) {
		return false
	}
	for varName, patterns := range o.Vars {
		if excluded(patterns, vars[varName]) {
			return false
		}
	}
	return true
}

func (o OverrideMatcher) clone() OverrideMatcher {
	matcher := OverrideMatcher{
		OS:   make([]string, len(o.OS)),
		Arch: make([]string, len(o.Arch)),
		Vars: make(map[string][]string, len(o.Vars)),
	}
	copy(matcher.OS, o.OS)
	copy(matcher.Arch, o.Arch)
	for k, v := range o.Vars {
		matcher.Vars[k] = make([]string, len(v))
		copy(matcher.Vars[k], v)
	}
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
	RequiredVars  []string                     `json:"required_vars,omitempty" yaml:"required_vars,omitempty"`
	Overrides     []DependencyOverride         `json:"overrides,omitempty" yaml:",omitempty"`
	Substitutions map[string]map[string]string `json:"substitutions,omitempty" yaml:",omitempty"`
	Systems       []SystemInfo                 `json:"systems,omitempty" yaml:"systems,omitempty"`
}

func cloneSubstitutions(subs map[string]map[string]string) map[string]map[string]string {
	if subs == nil {
		return nil
	}
	result := make(map[string]map[string]string, len(subs))
	for k, v := range subs {
		result[k] = copyStringMap(v)
	}
	return result
}

func varsWithSubstitutions(vars map[string]string, subs map[string]map[string]string) map[string]string {
	if vars == nil || subs == nil {
		return vars
	}
	vars = copyStringMap(vars)
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
		dep.Vars = copyStringMap(d.Vars)
	}
	if d.URL != nil {
		val := *d.URL
		d.URL = &val
	}
	if d.ArchivePath != nil {
		val := *d.ArchivePath
		d.ArchivePath = &val
	}
	if d.Template != nil {
		val := *d.Template
		d.Template = &val
	}
	if d.BinName != nil {
		val := *d.BinName
		d.BinName = &val
	}
	if d.Link != nil {
		val := *d.Link
		d.Link = &val
	}
	if d.Overrides != nil {
		dep.Overrides = make([]DependencyOverride, len(d.Overrides))
		for i := range d.Overrides {
			dep.Overrides[i] = *d.Overrides[i].clone()
		}
	}
	dep.Substitutions = cloneSubstitutions(d.Substitutions)
	return &dep
}

// interpolateVars executes go templates in values
func (d *Dependency) interpolateVars(system SystemInfo) error {
	interpolate := func(tmpl string) (string, error) {
		return executeTemplate(tmpl, system.OS, system.Arch, d.Vars)
	}
	if d.URL != nil {
		val, err := interpolate(*d.URL)
		if err != nil {
			return err
		}
		d.URL = &val
	}
	if d.ArchivePath != nil {
		val, err := interpolate(*d.ArchivePath)
		if err != nil {
			return err
		}
		d.ArchivePath = &val
	}
	if d.BinName != nil {
		val, err := interpolate(*d.BinName)
		if err != nil {
			return err
		}
		d.BinName = &val
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
	if d.RequiredVars != nil {
		newDL.RequiredVars = append(newDL.RequiredVars, d.RequiredVars...)
	}
	if len(d.Systems) > 0 {
		newDL.Systems = make([]SystemInfo, len(d.Systems))
		copy(newDL.Systems, d.Systems)
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
	for i := range overrides {
		d.Overrides = append(d.Overrides, *overrides[i].clone())
	}
}

const maxOverrideDepth = 2

func (d *Dependency) applyOverrides(info SystemInfo, depth int) {
	for i := range d.Overrides {
		if !d.Overrides[i].OverrideMatcher.matches(info, d.Vars) {
			continue
		}
		dependency := &d.Overrides[i].Dependency
		if depth <= maxOverrideDepth {
			dependency.applyOverrides(info, depth+1)
		}
		if dependency.Link != nil {
			d.Link = dependency.Link
		}
		if d.Vars == nil {
			d.Vars = make(map[string]string, len(dependency.Vars))
		}
		for k, v := range dependency.Vars {
			d.Vars[k] = v
		}
		if dependency.ArchivePath != nil {
			d.ArchivePath = dependency.ArchivePath
		}
		if dependency.BinName != nil {
			d.BinName = dependency.BinName
		}
		if dependency.URL != nil {
			d.URL = dependency.URL
		}
	}
	d.Overrides = nil
}

func boolPtr(val bool) *bool {
	return &val
}

func stringPtr(val string) *string {
	return &val
}

func boolFromPtr(bPtr *bool) bool {
	if bPtr == nil {
		return false
	}
	return *bPtr
}

func strFromPtr(sPtr *string) string {
	if sPtr == nil {
		return ""
	}
	return *sPtr
}

func linkBin(target, extractDir, archivePath, binName string) error {
	archivePath = filepath.FromSlash(archivePath)
	if archivePath == "" {
		archivePath = filepath.FromSlash(binName)
	}
	var err error
	if fileExists(target) {
		err = os.RemoveAll(target)
		if err != nil {
			return err
		}
	}
	extractDir, err = filepath.Abs(extractDir)
	if err != nil {
		return err
	}
	extractedBin := filepath.Join(extractDir, archivePath)
	err = os.MkdirAll(filepath.Dir(target), 0o750)
	if err != nil {
		return err
	}
	var linkTargetDir string
	linkTargetDir, err = filepath.Abs(filepath.Dir(target))
	if err != nil {
		return err
	}

	linkTargetDir, err = filepath.EvalSymlinks(linkTargetDir)
	if err != nil {
		return err
	}

	extractedBin, err = filepath.EvalSymlinks(extractedBin)
	if err != nil {
		return err
	}

	var dst string
	dst, err = filepath.Rel(linkTargetDir, extractedBin)
	if err != nil {
		return err
	}
	err = os.Symlink(dst, target)
	if err != nil {
		return err
	}
	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	return os.Chmod(target, info.Mode().Perm()|0o750)
}

func copyBin(target, extractDir, archivePath, binName string) error {
	archivePath = filepath.FromSlash(archivePath)
	if archivePath == "" {
		archivePath = filepath.FromSlash(binName)
	}
	var err error
	if fileExists(target) {
		err = os.RemoveAll(target)
		if err != nil {
			return err
		}
	}
	extractDir, err = filepath.Abs(extractDir)
	if err != nil {
		return err
	}
	extractedBin := filepath.Join(extractDir, archivePath)
	err = os.MkdirAll(filepath.Dir(target), 0o750)
	if err != nil {
		return err
	}
	err = copyFile(extractedBin, target, nil)
	if err != nil {
		return err
	}
	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	return os.Chmod(target, info.Mode().Perm()|0o750)
}

func logCloseErr(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Println(err)
	}
}

// extract extracts an archive
func extract(archivePath, extractDir string) error {
	dlName := filepath.Base(archivePath)
	downloadDir := filepath.Dir(archivePath)
	extractSumFile := filepath.Join(downloadDir, ".extractsum")

	if wantSum, sumErr := os.ReadFile(extractSumFile); sumErr == nil {
		var exs string
		exs, sumErr = directoryChecksum(extractDir)
		if sumErr == nil && exs == strings.TrimSpace(string(wantSum)) {
			return nil
		}
	}

	err := os.RemoveAll(extractDir)
	if err != nil {
		return err
	}
	err = os.MkdirAll(extractDir, 0o750)
	if err != nil {
		return err
	}
	tarPath := filepath.Join(downloadDir, dlName)
	_, err = archiver.ByExtension(dlName)
	if err != nil {
		return copyFile(tarPath, filepath.Join(extractDir, dlName), logCloseErr)
	}
	err = archiver.Unarchive(tarPath, extractDir)
	if err != nil {
		return err
	}
	extractSum, err := directoryChecksum(extractDir)
	if err != nil {
		return err
	}

	return os.WriteFile(extractSumFile, []byte(extractSum), 0o600)
}

// getURLChecksum returns the checksum of what is returned from this url
func getURLChecksum(dlURL string) (string, error) {
	downloadDir, err := os.MkdirTemp("", "bindown")
	if err != nil {
		return "", err
	}
	dlPath := filepath.Join(downloadDir, "foo")
	err = downloadFile(dlPath, dlURL)
	if err != nil {
		_ = os.RemoveAll(downloadDir) //nolint:errcheck // we already have an error to report
		return "", err
	}
	sum, err := fileChecksum(dlPath)
	cleanupErr := os.RemoveAll(downloadDir)
	if err == nil {
		err = cleanupErr
	}
	return sum, err
}

func downloadFile(targetPath, url string) error {
	err := os.MkdirAll(filepath.Dir(targetPath), 0o750)
	if err != nil {
		return err
	}
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer logCloseErr(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed downloading %s", url)
	}
	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer logCloseErr(out)
	_, err = io.Copy(out, resp.Body)
	return err
}

func download(dlURL, outputPath, checksum string, force bool) error {
	var err error
	if force {
		err = os.RemoveAll(outputPath)
		if err != nil {
			return err
		}
	}
	ok, err := fileExistsWithChecksum(outputPath, checksum)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	err = downloadFile(outputPath, dlURL)
	if err != nil {
		return err
	}
	return validateFileChecksum(outputPath, checksum)
}

func validateFileChecksum(filename, checksum string) error {
	result, err := fileChecksum(filename)
	if err != nil {
		return err
	}
	if checksum != result {
		defer func() {
			delErr := os.RemoveAll(filename)
			if delErr != nil {
				log.Printf("Error deleting suspicious file at %q. Please delete it manually", filename)
			}
		}()
		return fmt.Errorf(`checksum mismatch in downloaded file %q 
wanted: %s
got: %s`, filename, checksum, result)
	}
	return nil
}
