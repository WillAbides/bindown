package bindown

import (
	"bytes"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/willabides/bindown/v3/internal/downloader"
)

//Config is our main config
type Config struct {
	Downloadables map[string]*Downloadable `yaml:"downloadables"`
	Templates     map[string]*Downloadable `yaml:"templates,omitempty"`
	URLChecksums  map[string]string        `yaml:"url_checksums,omitempty"`
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

func (s *SystemInfo) equal(other *SystemInfo) bool {
	return s.OS == other.OS && s.Arch == other.Arch
}

func (c *Config) getDownloadable(name string) *Downloadable {
	if c.Downloadables == nil {
		return nil
	}
	return c.Downloadables[name]
}

//BinName returns the bin name for a downloader on a given system
func (c *Config) BinName(downloadableName string, system SystemInfo) (string, error) {
	dl, err := c.buildDownloader(downloadableName, system)
	if err != nil {
		return "", err
	}
	if dl.BinName != "" {
		return dl.BinName, nil
	}
	return downloadableName, nil
}

func (c *Config) buildDownloader(downloadableName string, info SystemInfo) (*downloader.Downloader, error) {
	downloadable := c.getDownloadable(downloadableName)
	if downloadable == nil {
		return nil, fmt.Errorf("no downloadable configured with the name %q", downloadableName)
	}

	downloadable = downloadable.clone()
	err := downloadable.applyTemplate(c.Templates, 0)
	if err != nil {
		return nil, err
	}
	downloadable.applyOverrides(info, 0)
	dl := &downloader.Downloader{
		OS:   info.OS,
		Arch: info.Arch,
		Vars: downloadable.Vars,
	}
	if downloadable.URL != nil {
		dl.URL = *downloadable.URL
	}
	if downloadable.ArchivePath != nil {
		dl.ArchivePath = *downloadable.ArchivePath
	}
	if downloadable.BinName != nil {
		dl.BinName = *downloadable.BinName
	}
	if dl.BinName == "" {
		dl.BinName = filepath.Base(downloadableName)
	}
	if downloadable.Link != nil {
		dl.Link = *downloadable.Link
	}
	return dl, nil
}

func (c *Config) downloadableKnownBuilds(downloadableName string) ([]SystemInfo, error) {
	downloadable := c.getDownloadable(downloadableName)
	if downloadable == nil {
		return []SystemInfo{}, nil
	}
	downloadable = downloadable.clone()
	err := downloadable.applyTemplate(c.Templates, 0)
	if err != nil {
		return nil, err
	}
	result := make([]SystemInfo, len(downloadable.KnownBuilds))
	if downloadable.KnownBuilds != nil {
		copy(result, downloadable.KnownBuilds)
	}
	return result, nil
}

func (c *Config) allDownloadableNames() []string {
	if len(c.Downloadables) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(c.Downloadables))
	for dl := range c.Downloadables {
		result = append(result, dl)
	}
	return result
}

//ConfigAddChecksumsOptions contains options for Config.AddChecksums
type ConfigAddChecksumsOptions struct {

	// Only add checksums for these downloadables. When Downloadables is empty, AddChecksums adds checksums for all
	//configured downloadables.
	Downloadables []string

	// Only add checksums for these system targets. When Systems is empty, AddChecksums adds checksums for all known
	// builds configured for each downloadable.
	Systems []SystemInfo
}

//AddChecksums downloads, calculates checksums and adds them to the config's URLChecksums. AddChecksums skips urls that
//already exist in URLChecksums.
func (c *Config) AddChecksums(opts *ConfigAddChecksumsOptions) error {
	if opts == nil {
		opts = &ConfigAddChecksumsOptions{}
	}
	downloadables := opts.Downloadables
	if len(downloadables) == 0 && c.Downloadables != nil {
		downloadables = make([]string, 0, len(c.Downloadables))
		for dlName := range c.Downloadables {
			downloadables = append(downloadables, dlName)
		}
	}
	var err error
	for _, dlName := range downloadables {
		downloadable := c.getDownloadable(dlName)
		if downloadable == nil {
			return fmt.Errorf("no downloadable configured with the name %q", dlName)
		}
		sysTodo := opts.Systems
		if len(sysTodo) == 0 {
			sysTodo, err = c.downloadableKnownBuilds(dlName)
			if err != nil {
				return err
			}
		}
		for _, system := range sysTodo {
			err = c.addChecksum(dlName, system)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Config) addChecksum(downloadableName string, sysInfo SystemInfo) error {
	dl, err := c.buildDownloader(downloadableName, sysInfo)
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
	c.Downloadables[downloadableName].addKnownBuild(sysInfo)
	if c.URLChecksums == nil {
		c.URLChecksums = make(map[string]string, 1)
	}
	c.URLChecksums[dlURL] = sum
	return nil
}

//ConfigValidateOptions contains options for Config.Validate
type ConfigValidateOptions struct {

	// Only validates these downloadables. When Downloadables is empty, Validate validates all configured downloadables.
	Downloadables []string

	// Only validates system targets. When Systems is empty, AddChecksums validates all known builds configured for each downloadable.
	Systems []SystemInfo
}

//Validate installs the downloader to a temporary directory and returns an error if it was unsuccessful.
func (c *Config) Validate(downloadables []string, systems []SystemInfo) error {
	runtime.Version()
	if len(downloadables) == 0 {
		downloadables = c.allDownloadableNames()
	}
	var err error
	for _, downloadableName := range downloadables {
		sysInfos := systems
		if len(sysInfos) == 0 {
			sysInfos, err = c.downloadableKnownBuilds(downloadableName)
			if err != nil {
				return err
			}
		}
		for _, sysInfo := range sysInfos {
			dl, err := c.buildDownloader(downloadableName, sysInfo)
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
				DownloaderName: downloadableName,
				Checksum:       checksum,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
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

//Install installs a downloadable
func (c Config) Install(downloadableName string, sysInfo SystemInfo, opts *ConfigInstallOpts) error {
	if opts == nil {
		opts = &ConfigInstallOpts{}
	}
	dl, err := c.buildDownloader(downloadableName, sysInfo)
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
		DownloaderName: downloadableName,
		CellarDir:      opts.CellarDir,
		TargetDir:      opts.TargetDir,
		Force:          opts.Force,
		Checksum:       checksum,
	})
}

//ExtractPath returns the path where a downloadable will be extracted
func (c Config) ExtractPath(downloadableName string, sysInfo SystemInfo, cellarDir string) (string, error) {
	dl, err := c.buildDownloader(downloadableName, sysInfo)
	if err != nil {
		return "", err
	}
	sub := dl.ExtractsSubName(c.URLChecksums)
	return filepath.Join(cellarDir, "extracts", sub), nil
}
