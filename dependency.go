package bindown

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/mholt/archiver/v3"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// DependencyOverride overrides a dependency's configuration
type DependencyOverride struct {
	OverrideMatcher OverrideMatcher `json:"matcher" yaml:"matcher,omitempty"`
	Dependency      Dependency      `json:"dependency" yaml:",omitempty"`
}

func (o *DependencyOverride) clone() *DependencyOverride {
	return &DependencyOverride{
		Dependency:      *(o.Dependency.clone()),
		OverrideMatcher: o.OverrideMatcher.clone(),
	}
}

// OverrideMatcher contains a list or oses and arches to match an override. If either os or arch is empty, all oses and arches match.
type OverrideMatcher map[string][]string

func (o OverrideMatcher) matches(info SystemInfo, vars map[string]string) bool {
	if vars == nil {
		vars = map[string]string{}
	}
	for varName, patterns := range o {
		val, ok := vars[varName]
		if !ok {
			if varName == "os" {
				val = info.OS
			}
			if varName == "arch" {
				val = info.Arch
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

func (o OverrideMatcher) clone() OverrideMatcher {
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

func (d *Dependency) url() (string, error) {
	if d.URL == nil {
		return "", fmt.Errorf("no URL configured")
	}
	return *d.URL, nil
}

func (d *Dependency) clone() *Dependency {
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
	//nolint:gocritic // rangeValCopy -- memory allocation is not a concern here
	for i, override := range dep.Overrides {
		dep.Overrides[i] = *override.clone()
	}
	return &dep
}

// interpolateVars executes go templates in values
func (d *Dependency) interpolateVars(system SystemInfo) error {
	for _, p := range []*string{d.URL, d.ArchivePath, d.BinName} {
		if p == nil {
			continue
		}
		var err error
		*p, err = executeTemplate(*p, system.OS, system.Arch, d.Vars)
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
		d.Link = overrideValue(d.Link, dependency.Link)
		d.ArchivePath = overrideValue(d.ArchivePath, dependency.ArchivePath)
		d.BinName = overrideValue(d.BinName, dependency.BinName)
		d.URL = overrideValue(d.URL, dependency.URL)
		maps.Copy(d.Vars, dependency.Vars)
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

func linkBin(link, extractDir, archivePath, binName string) error {
	archivePath = filepath.FromSlash(archivePath)
	if archivePath == "" {
		archivePath = filepath.FromSlash(binName)
	}
	absExtractDir, err := filepath.Abs(extractDir)
	if err != nil {
		return err
	}
	extractedBin := filepath.Join(absExtractDir, archivePath)
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

	extractedBin, err = filepath.EvalSymlinks(extractedBin)
	if err != nil {
		return err
	}

	dst, err := filepath.Rel(linkDir, extractedBin)
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

// getURLChecksum returns the checksum of the file at dlURL. If tempFile is specified
// it will be used as the temporary file to download the file to and it will be the caller's
// responsibility to clean it up. Otherwise, a temporary file will be created and cleaned up
// automatically.
func getURLChecksum(dlURL, tempFile string) (_ string, errOut error) {
	if tempFile == "" {
		downloadDir, err := os.MkdirTemp("", "bindown")
		if err != nil {
			return "", err
		}
		tempFile = filepath.Join(downloadDir, "foo")
		defer func() {
			cleanupErr := os.RemoveAll(downloadDir)
			if errOut == nil {
				errOut = cleanupErr
			}
		}()
	}
	err := downloadFile(tempFile, dlURL)
	if err != nil {
		return "", err
	}
	return fileChecksum(tempFile)
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
