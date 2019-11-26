package bindown

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/willabides/bindown/v2/internal/util"
	"go.uber.org/multierr"
)

//LoadConfig returns a Config from a config reader
func LoadConfig(config io.Reader) (*Config, error) {
	configBytes, err := ioutil.ReadAll(config)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(configBytes))
	decoder.DisallowUnknownFields()
	var cfg Config
	err = decoder.Decode(&cfg)
	if err != nil {
		decoder = json.NewDecoder(bytes.NewReader(configBytes))
		decoder.DisallowUnknownFields()
		dls := cfg.Downloaders
		err = decoder.Decode(&dls)
		if err == nil {
			cfg.Downloaders = dls
		}
	}
	if err != nil {
		return nil, err
	}
	return &cfg, err
}

//LoadConfigFile returns a Config from the path to a config file
func LoadConfigFile(configFile string) (*Config, error) {
	configReader, err := os.Open(configFile) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("couldn't read config file: %s", configFile)
	}
	defer util.LogCloseErr(configReader)
	return LoadConfig(configReader)
}

//Config is downloaders configuration
type Config struct {
	Downloaders map[string][]*Downloader `json:"downloaders,omitempty"`
}

// Downloader returns a Downloader for the given binary, os and arch.
func (c *Config) Downloader(binary, opSys, arch string) *Downloader {
	l, ok := c.Downloaders[binary]
	if !ok {
		return nil
	}
	for _, d := range l {
		if !eqOS(opSys, d.OS) {
			continue
		}
		if strings.EqualFold(arch, d.Arch) {
			return d
		}
	}
	return nil
}

//UpdateChecksums updates the checksums for binary's downloaders
func (c *Config) UpdateChecksums(binary, cellarDir string) error {
	if len(c.Downloaders[binary]) == 0 {
		return fmt.Errorf("nothing configured for binary %q", binary)
	}
	var err error
	if cellarDir == "" {
		cellarDir, err = ioutil.TempDir("", "bindown")
		if err != nil {
			return err
		}
		defer func() {
			_ = util.Rm(cellarDir) //nolint:errcheck
		}()
	}
	for _, downloader := range c.Downloaders[binary] {
		err = downloader.UpdateChecksum(UpdateChecksumOpts{
			DownloaderName: binary,
			CellarDir:      cellarDir,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

//Validate runs validate on all Downloaders for the given binary.
//error may be a multierr. Individual errors can be retrieved with multierr.Errors(err)
func (c *Config) Validate(binary, cellarDir string) error {
	downloaders := c.Downloaders[binary]
	if len(downloaders) == 0 {
		return fmt.Errorf("nothing configured for binary %q", binary)
	}
	var resErr error
	for _, downloader := range downloaders {
		err := downloader.Validate(cellarDir)
		if err != nil {
			resErr = multierr.Combine(
				resErr,
				fmt.Errorf("error validating %s - %s - %s: %v", binary, downloader.OS, downloader.Arch, err),
			)
		}
	}
	return resErr
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

//ConfigFile represents a config file
type ConfigFile struct {
	filename string
	*Config
}

//NewConfigFile creates a *ConfigFile for the file at filename
func NewConfigFile(filename string) (*ConfigFile, error) {
	b, err := ioutil.ReadFile(filename) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("couldn't read config file: %s", filename)
	}
	config, err := LoadConfig(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("couldn't load config: %v", err)
	}
	return &ConfigFile{
		filename: filename,
		Config:   config,
	}, nil
}

//WriteFile writes config back to the file
func (c *ConfigFile) WriteFile() error {
	b, err := json.MarshalIndent(c.Config, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(c.filename, b, 0640)
}
