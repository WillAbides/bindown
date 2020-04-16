package bindown

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/willabides/bindown/v3/internal/util"
)

//Config is our main config
type Config struct {
	Cache        string                 `json:"cache,omitempty" yaml:"cache,omitempty"`
	InstallDir   string                 `json:"install_dir,omitempty" yaml:"install_dir,omitempty"`
	Dependencies map[string]*Dependency `json:"dependencies,omitempty" yaml:",omitempty"`
	Templates    map[string]*Dependency `json:"templates,omitempty" yaml:",omitempty"`
	URLChecksums map[string]string      `json:"url_checksums,omitempty" yaml:"url_checksums,omitempty"`
}

//SystemInfo contains os and architecture for a target system
type SystemInfo struct {
	OS   string
	Arch string
}

func newSystemInfo(os, arch string) SystemInfo {
	return SystemInfo{
		OS:   os,
		Arch: arch,
	}
}

func (s *SystemInfo) String() string {
	return fmt.Sprintf("%s/%s", s.OS, s.Arch)
}

//UnmarshalText implements encoding.TextUnmarshaler
func (s *SystemInfo) UnmarshalText(text []byte) error {
	parts := bytes.Split(text, []byte{'/'})
	if len(parts) != 2 {
		return fmt.Errorf(`systemInfo must be in the form "os/architecture"`)
	}
	s.OS = string(parts[0])
	s.Arch = string(parts[1])
	return nil
}

//MarshalText implements encoding.TextMarshaler
func (s SystemInfo) MarshalText() (text []byte, err error) {
	return []byte(s.String()), nil
}

//BinName returns the bin name for a downloader on a given system
func (c *Config) BinName(depName string, system SystemInfo) (string, error) {
	dep, err := c.buildDependency(depName, system)
	if err != nil {
		return "", err
	}
	if dep.BinName != nil && *dep.BinName != "" {
		return *dep.BinName, nil
	}
	return depName, nil
}

func (c *Config) buildDependency(depName string, info SystemInfo) (*Dependency, error) {
	dep := c.Dependencies[depName]
	if dep == nil {
		return nil, fmt.Errorf("no dependency configured with the name %q", depName)
	}
	dep = dep.clone()
	err := dep.applyTemplate(c.Templates, 0)
	if err != nil {
		return nil, err
	}
	dep.applyOverrides(info, 0)
	if dep.Vars == nil {
		dep.Vars = map[string]string{}
	}
	if _, ok := dep.Vars["os"]; !ok {
		dep.Vars["os"] = info.OS
	}
	if _, ok := dep.Vars["arch"]; !ok {
		dep.Vars["arch"] = info.Arch
	}
	dep.Vars = varsWithSubstitutions(dep.Vars, dep.Substitutions)
	err = dep.interpolateVars(info)
	if err != nil {
		return nil, err
	}
	return dep, nil
}

func (c *Config) allDependencyNames() []string {
	if len(c.Dependencies) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(c.Dependencies))
	for dl := range c.Dependencies {
		result = append(result, dl)
	}
	return result
}

//ConfigAddChecksumsOptions contains options for Config.AddChecksums
type ConfigAddChecksumsOptions struct {

	// Only add checksums for these dependencies. When Dependencies is empty, AddChecksums adds checksums for all
	//configured dependencies.
	Dependencies []string

	// Only add checksums for these system targets. When Systems is empty, AddChecksums adds checksums for all known
	// builds configured for each dependency.
	Systems []SystemInfo
}

//AddChecksums downloads, calculates checksums and adds them to the config's URLChecksums. AddChecksums skips urls that
//already exist in URLChecksums.
func (c *Config) AddChecksums(dependencies []string, systems []SystemInfo) error {
	if len(dependencies) == 0 && c.Dependencies != nil {
		dependencies = make([]string, 0, len(c.Dependencies))
		for dlName := range c.Dependencies {
			dependencies = append(dependencies, dlName)
		}
	}
	var err error
	for _, depName := range dependencies {
		dp := c.Dependencies[depName]
		if dp == nil {
			return fmt.Errorf("no dependency configured with the name %q", depName)
		}
		for _, system := range systems {
			err = c.addChecksum(depName, system)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Config) addChecksum(dependencyName string, sysInfo SystemInfo) error {
	dep, err := c.buildDependency(dependencyName, sysInfo)
	if err != nil {
		return err
	}
	if dep.URL == nil {
		return fmt.Errorf("no URL configured")
	}
	existingSum := c.URLChecksums[*dep.URL]
	if existingSum != "" {
		return nil
	}
	sum, err := getURLChecksum(*dep.URL)
	if err != nil {
		return err
	}
	if c.URLChecksums == nil {
		c.URLChecksums = make(map[string]string, 1)
	}
	c.URLChecksums[*dep.URL] = sum
	return nil
}

//ConfigValidateOptions contains options for Config.Validate
type ConfigValidateOptions struct {

	// Only validates these dependencies. When Dependencies is empty, Validate validates all configured dependencies.
	Dependencies []string

	// Only validates system targets. When Systems is empty, AddChecksums validates all known builds configured for each
	//dependency.
	Systems []SystemInfo
}

//Validate installs the downloader to a temporary directory and returns an error if it was unsuccessful.
func (c *Config) Validate(dependencies []string, systems []SystemInfo) error {
	runtime.Version()
	if len(dependencies) == 0 {
		dependencies = c.allDependencyNames()
	}
	tmpCacheDir, err := ioutil.TempDir("", "bindown-cache")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(tmpCacheDir) //nolint:errcheck
	}()
	tmpBinDir, err := ioutil.TempDir("", "bindown-bin")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(tmpBinDir) //nolint:errcheck
	}()
	c.InstallDir = tmpBinDir
	c.Cache = tmpCacheDir
	for _, depName := range dependencies {
		for _, system := range systems {
			_, err = c.InstallDependency(depName, system, &ConfigInstallDependencyOpts{
				Force: true,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//ConfigDownloadDependencyOpts options for Config.DownloadDependency
type ConfigDownloadDependencyOpts struct {
	TargetFile string
	Force      bool
}

//extractsCacheDir returns the cache directory for an extraction based on the download's checksum and dependency name
func (c *Config) extractsCacheDir(dependencyName, checksum string) string {
	return filepath.Join(c.Cache, "extracts", util.MustHexHash(fnv.New64a(), []byte(checksum), []byte(dependencyName)))
}

//downloadCacheDir returns the cache directory for a file based on its checksum
func (c *Config) downloadCacheDir(checksum string) string {
	return filepath.Join(c.Cache, "downloads", util.MustHexHash(fnv.New64a(), []byte(checksum)))
}

//DownloadDependency downloads a dependency
func (c *Config) DownloadDependency(dependencyName string, sysInfo SystemInfo, opts *ConfigDownloadDependencyOpts) (string, error) {
	if opts == nil {
		opts = &ConfigDownloadDependencyOpts{}
	}
	targetFile := opts.TargetFile
	dep, err := c.buildDependency(dependencyName, sysInfo)
	if err != nil {
		return "", err
	}
	if dep.URL == nil {
		return "", fmt.Errorf("no URL configured")
	}

	checksum, err := c.dependencyChecksum(dependencyName, sysInfo)
	if err != nil {
		return "", err
	}

	if targetFile == "" {
		dlFile, err := urlFilename(*dep.URL)
		if err != nil {
			return "", err
		}
		cacheDir := c.downloadCacheDir(checksum)
		targetFile = filepath.Join(cacheDir, dlFile)
	}
	return targetFile, download(strFromPtr(dep.URL), targetFile, checksum, opts.Force)
}

func urlFilename(dlURL string) (string, error) {
	u, err := url.Parse(dlURL)
	if err != nil {
		return "", err
	}
	return path.Base(u.EscapedPath()), nil
}

func (c *Config) dependencyChecksum(dependencyName string, sysInfo SystemInfo) (string, error) {
	dep, err := c.buildDependency(dependencyName, sysInfo)
	if err != nil {
		return "", err
	}
	if dep.URL == nil {
		return "", fmt.Errorf("no URL configured")
	}
	checksum, ok := c.URLChecksums[*dep.URL]
	if !ok {
		return "", fmt.Errorf("no checksum for the url %q", *dep.URL)
	}
	return checksum, nil
}

//ConfigExtractDependencyOpts options for Config.ExtractDependency
type ConfigExtractDependencyOpts struct {
	TargetDirectory string
	Force           bool
}

//ExtractDependency downloads and extracts a dependency
func (c *Config) ExtractDependency(dependencyName string, sysInfo SystemInfo, opts *ConfigExtractDependencyOpts) (string, error) {
	if opts == nil {
		opts = &ConfigExtractDependencyOpts{}
	}
	downloadPath, err := c.DownloadDependency(dependencyName, sysInfo, &ConfigDownloadDependencyOpts{
		Force: opts.Force,
	})
	if err != nil {
		return "", err
	}
	downloadDir := filepath.Dir(downloadPath)
	dep, err := c.buildDependency(dependencyName, sysInfo)
	if err != nil {
		return "", err
	}
	if dep.URL == nil {
		return "", fmt.Errorf("no URL configured")
	}

	targetDir := opts.TargetDirectory
	if targetDir == "" {
		var checksum string
		checksum, err = c.dependencyChecksum(dependencyName, sysInfo)
		if err != nil {
			return "", err
		}
		targetDir = c.extractsCacheDir(dependencyName, checksum)
	}
	dlFile, err := urlFilename(*dep.URL)
	if err != nil {
		return "", err
	}
	err = extract(filepath.Join(downloadDir, dlFile), targetDir)
	if err != nil {
		return "", err
	}
	return targetDir, nil
}

//ConfigInstallDependencyOpts provides options for Config.InstallDependency
type ConfigInstallDependencyOpts struct {
	// TargetPath is the path where the executable should end up
	TargetPath string
	// Force - whether to force the install even if it already exists
	Force bool
}

//InstallDependency downloads, extracts and installs a dependency
func (c *Config) InstallDependency(dependencyName string, sysInfo SystemInfo, opts *ConfigInstallDependencyOpts) (string, error) {
	if opts == nil {
		opts = &ConfigInstallDependencyOpts{}
	}
	extractDir, err := c.ExtractDependency(dependencyName, sysInfo, &ConfigExtractDependencyOpts{
		Force: opts.Force,
	})
	if err != nil {
		return "", err
	}
	targetPath := opts.TargetPath
	if targetPath == "" {
		var binName string
		binName, err = c.BinName(dependencyName, sysInfo)
		if err != nil {
			return "", err
		}
		targetPath = filepath.Join(c.InstallDir, binName)
	}
	dep, err := c.buildDependency(dependencyName, sysInfo)
	if err != nil {
		return "", err
	}

	binName := strFromPtr(dep.BinName)
	if binName == "" {
		binName = dependencyName
	}

	if boolFromPtr(dep.Link) {
		return targetPath, linkBin(targetPath, extractDir, strFromPtr(dep.ArchivePath), binName)
	}

	return targetPath, copyBin(targetPath, extractDir, strFromPtr(dep.ArchivePath), binName)
}
