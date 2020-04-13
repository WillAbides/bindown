package bindown

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/willabides/bindown/v3/internal/downloader"
)

//Config is our main config
type Config struct {
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

func (c *Config) getDependency(name string) *Dependency {
	if c.Dependencies == nil {
		return nil
	}
	return c.Dependencies[name]
}

//BinName returns the bin name for a downloader on a given system
func (c *Config) BinName(dep string, system SystemInfo) (string, error) {
	dl, err := c.buildDownloader(dep, system)
	if err != nil {
		return "", err
	}
	if dl.BinName != "" {
		return dl.BinName, nil
	}
	return dep, nil
}

func (c *Config) buildDownloader(depName string, info SystemInfo) (*downloader.Downloader, error) {
	dep := c.getDependency(depName)
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
	dl := &downloader.Downloader{
		OS:   info.OS,
		Arch: info.Arch,
		Vars: varsWithSubstitutions(dep.Vars, dep.Substitutions),
	}
	if dep.URL != nil {
		dl.URL = *dep.URL
	}
	if dep.ArchivePath != nil {
		dl.ArchivePath = *dep.ArchivePath
	}
	if dep.BinName != nil {
		dl.BinName = *dep.BinName
	}
	if dl.BinName == "" {
		dl.BinName = filepath.Base(depName)
	}
	if dep.Link != nil {
		dl.Link = *dep.Link
	}
	return dl, nil
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
func (c *Config) AddChecksums(opts *ConfigAddChecksumsOptions) error {
	if opts == nil {
		opts = &ConfigAddChecksumsOptions{}
	}
	deps := opts.Dependencies
	if len(deps) == 0 && c.Dependencies != nil {
		deps = make([]string, 0, len(c.Dependencies))
		for dlName := range c.Dependencies {
			deps = append(deps, dlName)
		}
	}
	var err error
	for _, depName := range deps {
		dp := c.getDependency(depName)
		if dp == nil {
			return fmt.Errorf("no dependency configured with the name %q", depName)
		}
		for _, system := range opts.Systems {
			err = c.addChecksum(depName, system)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Config) addChecksum(dependencyName string, sysInfo SystemInfo) error {
	dl, err := c.buildDownloader(dependencyName, sysInfo)
	if err != nil {
		return err
	}
	dlURL, err := dl.GetURL()
	if err != nil {
		return err
	}
	existingSum := c.URLChecksums[dlURL]
	if existingSum != "" {
		return nil
	}
	sum, err := dl.GetUpdatedChecksum(downloader.UpdateChecksumOpts{
		URLChecksums: c.URLChecksums,
	})
	if err != nil {
		return err
	}
	if c.URLChecksums == nil {
		c.URLChecksums = make(map[string]string, 1)
	}
	c.URLChecksums[dlURL] = sum
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
	for _, depName := range dependencies {
		for _, system := range systems {
			dl, err := c.buildDownloader(depName, system)
			if err != nil {
				return err
			}
			dlURL, err := dl.GetURL()
			if err != nil {
				return err
			}

			checksum, ok := c.URLChecksums[dlURL]
			if !ok {
				return fmt.Errorf("no checksum for the url %q", dlURL)
			}
			err = dl.Validate(downloader.ValidateOpts{
				DownloaderName: depName,
				Checksum:       checksum,
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

//DownloadDependency download a dependency
func (c Config) DownloadDependency(dependencyName string, sysInfo SystemInfo, opts *ConfigDownloadDependencyOpts) (string, error) {
	if opts == nil {
		opts = &ConfigDownloadDependencyOpts{}
	}
	targetFile := opts.TargetFile
	dl, err := c.buildDownloader(dependencyName, sysInfo)
	if err != nil {
		return "", err
	}

	dlURLStr, err := dl.GetURL()
	if err != nil {
		return "", err
	}

	checksum, ok := c.URLChecksums[dlURLStr]
	if !ok {
		return "", fmt.Errorf("no checksum for the url %q", dlURLStr)
	}

	if targetFile == "" {
		dlURL, err := url.Parse(dlURLStr)
		if err != nil {
			return "", err
		}
		pwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		targetFile = filepath.Join(pwd, path.Base(dlURL.EscapedPath()))
	}
	return targetFile, dl.Download(targetFile, checksum, opts.Force)
}

//ConfigInstallOpts provides options for Config.Install
type ConfigInstallOpts struct {
	// CellarDir is the directory where downloads and extractions will be placed.  Default is a <TargetDir>/.bindown
	CellarDir string
	// TargetDir is the directory where the executable should end up
	TargetDir string
	// Force - whether to force the install even if it already exists
	Force bool
}

//Install installs a dependency
func (c Config) Install(dependencyName string, sysInfo SystemInfo, opts *ConfigInstallOpts) error {
	if opts == nil {
		opts = &ConfigInstallOpts{}
	}
	dl, err := c.buildDownloader(dependencyName, sysInfo)
	if err != nil {
		return err
	}
	dlURL, err := dl.GetURL()
	if err != nil {
		return err
	}
	checksum, ok := c.URLChecksums[dlURL]
	if !ok {
		return fmt.Errorf("no checksum for the url %q", dlURL)
	}
	return dl.Install(downloader.InstallOpts{
		DownloaderName: dependencyName,
		CellarDir:      opts.CellarDir,
		TargetDir:      opts.TargetDir,
		Force:          opts.Force,
		Checksum:       checksum,
	})
}

//ExtractPath returns the path where a dependency will be extracted
func (c Config) ExtractPath(dependencyName string, sysInfo SystemInfo, cellarDir string) (string, error) {
	dl, err := c.buildDownloader(dependencyName, sysInfo)
	if err != nil {
		return "", err
	}
	sub := dl.ExtractsSubName(c.URLChecksums)
	return filepath.Join(cellarDir, "extracts", sub), nil
}
