package bindown

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

//ConfigFile represents a config file
type ConfigFile struct {
	file string
	Config
}

func (c *ConfigFile) Write() error {
	var data []byte
	var err error
	data, err = yaml.Marshal(&c.Config)
	must(err)
	return ioutil.WriteFile(c.file, data, 0600)
}

//LoadConfigFile loads a config file
func LoadConfigFile(file string) (*ConfigFile, error) {
	data, err := ioutil.ReadFile(file) //nolint:gosec
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &ConfigFile{
		file:   file,
		Config: config,
	}, nil
}

//Config is downloaders configuration
type Config struct {
	Downloaders  map[string][]*Downloader `yaml:"downloaders"`
	URLChecksums map[string]string        `yaml:"url_checksums,omitempty"`
}

func (c *Config) urlChecksum(url string) string {
	if c.URLChecksums == nil {
		c.URLChecksums = map[string]string{}
	}
	return c.URLChecksums[url]
}

//DownloaderExtractDir returns the directory where a downloader's archive will be extracted
func (c *Config) DownloaderExtractDir(downloader *Downloader, binName, cellarDir string) (string, error) {
	dl := downloader.clone()
	dl.setDefaultBinName(binName)
	err := dl.applyTemplates()
	if err != nil {
		return "", err
	}
	sub := dl.extractsSubName(c.URLChecksums)
	return filepath.Join(cellarDir, "extracts", sub), nil
}

// Downloader returns a Downloader for the given binary, os and arch.
func (c *Config) Downloader(binary, os, arch string) *Downloader {
	l, ok := c.Downloaders[binary]
	if !ok {
		return nil
	}
	for _, d := range l {
		if !eqOS(os, d.OS) {
			continue
		}
		if strings.EqualFold(arch, d.Arch) {
			return d
		}
	}
	return nil
}

//AddDownloaderChecksums adds checksums to c.URLChecksums
func (c *Config) AddDownloaderChecksums(downloaderName, cellarDir string) error {
	if c.URLChecksums == nil {
		c.URLChecksums = make(map[string]string, 1)
	}
	names := c.allDownloaderNames()
	if downloaderName != "" {
		names = []string{downloaderName}
	}
	for _, name := range names {
		binary := filepath.Base(name)
		downloaders, ok := c.Downloaders[binary]
		if !ok {
			return fmt.Errorf("nothing configured for %q", binary)
		}
		for i := range downloaders {
			url, err := downloaders[i].url()
			if err != nil {
				return err
			}
			if c.URLChecksums[url] != "" {
				downloaders[i].Checksum = ""
				continue
			}
			sum, err := downloaders[i].getUpdatedChecksum(UpdateChecksumOpts{
				CellarDir:    cellarDir,
				URLChecksums: c.URLChecksums,
			})
			if err != nil {
				return err
			}
			downloaders[i].Checksum = ""
			c.URLChecksums[url] = sum
		}
		c.Downloaders[binary] = downloaders
	}
	return nil
}

func (c *Config) allDownloaderNames() []string {
	names := make([]string, 0, len(c.Downloaders))
	for name := range c.Downloaders {
		names = append(names, name)
	}
	return names
}

//Validate installs downloaders to a temp directory and returns an error if it was unsuccessful.
// If cellarDir is empty, it uses a temp directory.
// If downloaderName is empty, it validates all downloaders.
func (c *Config) Validate(downloaderName string, cellarDir string) error {
	names := c.allDownloaderNames()
	if downloaderName != "" {
		names = []string{downloaderName}
	}
	for _, name := range names {
		binary := filepath.Base(name)
		dls, ok := c.Downloaders[binary]
		if !ok {
			return fmt.Errorf("nothing configured for %q", binary)
		}
		for _, downloader := range dls {
			dl := downloader.clone()
			url, err := downloader.url()
			if err != nil {
				return err
			}
			if dl.Checksum == "" {
				dl.Checksum = c.urlChecksum(url)
			}
			err = dl.Validate(ValidateOpts{
				DownloaderName: binary,
				CellarDir:      cellarDir,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func eqOS(a, b string) bool {
	return strings.EqualFold(normalizeOS(a), normalizeOS(b))
}

var osAliases = map[string]string{
	"osx":   "darwin",
	"macos": "darwin",
}

func normalizeOS(os string) string {
	for alias, aliasedOs := range osAliases {
		if strings.EqualFold(alias, os) {
			return aliasedOs
		}
	}
	return strings.ToLower(os)
}
