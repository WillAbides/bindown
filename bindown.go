package bindown

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type configFileFormat int

const (
	formatJSON configFileFormat = iota + 1
	formatYAML
)

//ConfigFile represents a config file
type ConfigFile struct {
	format configFileFormat
	file   string
	Config
}

func (c *ConfigFile) Write() error {
	var data []byte
	var err error
	switch c.format {
	case formatJSON:
		data, err = json.MarshalIndent(&c.Config, "", "  ")
	case formatYAML:
		data, err = yaml.Marshal(&c.Config)
	}
	if err != nil {
		return err
	}
	return ioutil.WriteFile(c.file, data, 0600)
}

//LoadConfigFile loads a config file
func LoadConfigFile(file string) (*ConfigFile, error) {
	data, err := ioutil.ReadFile(file) //nolint:gosec
	if err != nil {
		return nil, err
	}
	cfg, err := loadConfigFromJSON(data)
	if err == nil {
		return &ConfigFile{
			format: formatJSON,
			file:   file,
			Config: *cfg,
		}, nil
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &ConfigFile{
		format: formatYAML,
		file:   file,
		Config: config,
	}, nil
}

func loadConfigFromJSON(data []byte) (*Config, error) {
	var cfg Config
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&cfg)
	if err == nil {
		return &cfg, nil
	}
	decoder = json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	dls := cfg.Downloaders
	err = decoder.Decode(&dls)
	if err != nil {
		return nil, err
	}
	cfg.Downloaders = dls
	return &cfg, nil
}

//Config is downloaders configuration
type Config struct {
	Downloaders map[string][]*Downloader `json:"downloaders,omitempty" yaml:"downloaders"`
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

//UpdateChecksums updates checksums
func (c *Config) UpdateChecksums(downloaderName, cellarDir string) error {
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
		for _, downloader := range downloaders {
			err := downloader.UpdateChecksum(UpdateChecksumOpts{
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
			err := downloader.Validate(binary, cellarDir)
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
