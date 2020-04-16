package bindown

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v3"
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

//interpolateVars executes go templates in values
func (d *Dependency) interpolateVars(system SystemInfo) error {
	interpolate := func(tmpl string) (string, error) {
		return util.ExecuteTemplate(tmpl, system.OS, system.Arch, d.Vars)
	}
	var err error
	if d.URL != nil {
		*d.URL, err = interpolate(*d.URL)
		if err != nil {
			return err
		}
	}
	if d.ArchivePath != nil {
		*d.ArchivePath, err = interpolate(*d.ArchivePath)
		if err != nil {
			return err
		}
	}
	if d.BinName != nil {
		*d.BinName, err = interpolate(*d.BinName)
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
	if util.FileExists(target) {
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
	err = os.MkdirAll(filepath.Dir(target), 0750)
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
	return os.Chmod(target, 0750) //nolint:gosec
}

func copyBin(target, extractDir, archivePath, binName string) error {
	archivePath = filepath.FromSlash(archivePath)
	if archivePath == "" {
		archivePath = filepath.FromSlash(binName)
	}
	var err error
	if util.FileExists(target) {
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
	err = os.MkdirAll(filepath.Dir(target), 0750)
	if err != nil {
		return err
	}
	err = util.CopyFile(extractedBin, target, nil)
	if err != nil {
		return err
	}
	return os.Chmod(target, 0750) //nolint:gosec
}

func logCloseErr(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Println(err)
	}
}

//extract extracts an archive
func extract(archivePath, extractDir string) error {
	dlName := filepath.Base(archivePath)
	downloadDir := filepath.Dir(archivePath)
	extractSumFile := filepath.Join(downloadDir, ".extractsum")

	if wantSum, sumErr := ioutil.ReadFile(extractSumFile); sumErr == nil { //nolint:gosec
		var exs string
		exs, sumErr = util.DirectoryChecksum(extractDir)
		if sumErr == nil && exs == strings.TrimSpace(string(wantSum)) {
			return nil
		}
	}

	err := os.RemoveAll(extractDir)
	if err != nil {
		return err
	}
	err = os.MkdirAll(extractDir, 0750)
	if err != nil {
		return err
	}
	tarPath := filepath.Join(downloadDir, dlName)
	_, err = archiver.ByExtension(dlName)
	if err != nil {
		return util.CopyFile(tarPath, filepath.Join(extractDir, dlName), logCloseErr)
	}
	err = archiver.Unarchive(tarPath, extractDir)
	if err != nil {
		return err
	}
	extractSum, err := util.DirectoryChecksum(extractDir)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(extractSumFile, []byte(extractSum), 0640)
}

//getURLChecksum returns the checksum of what is returned from this url
func getURLChecksum(dlURL string) (string, error) {
	downloadDir, err := ioutil.TempDir("", "bindown")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = os.RemoveAll(downloadDir) //nolint:errcheck
	}()
	dlPath := filepath.Join(downloadDir, "foo")
	err = downloadFile(dlPath, dlURL)
	if err != nil {
		return "", err
	}
	return util.FileChecksum(dlPath)
}

func downloadFile(targetPath, url string) error {
	err := os.MkdirAll(filepath.Dir(targetPath), 0750)
	if err != nil {
		return err
	}
	resp, err := http.Get(url) //nolint:gosec
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
	ok, err := util.FileExistsWithChecksum(outputPath, checksum)
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
	result, err := util.FileChecksum(filename)
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
