package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/willabides/bindown/v2"
	"github.com/willabides/bindown/v2/internal/util"
	"go.uber.org/multierr"
)

//go:generate mockgen -source config.go -destination internal/mocks/mock_config.go -package mocks

//Downloader interface for *bindown.Downloader
type Downloader interface {
	ErrString(binary string) string
	MatchesOS(opSys string) bool
	MatchesArch(arch string) bool
	HasChecksum(checksum string) bool
	UpdateChecksum(cellarDir string) error
	Install(downloaderName, cellarDir, targetDir string, force bool) error
	Validate(cellarDir string) error
}

func unmarshalDownloaders(p []byte) (map[string][]*bindown.Downloader, error) {
	downloaders := map[string][]*bindown.Downloader{}
	decoder := json.NewDecoder(bytes.NewReader(p))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&downloaders)
	if err != nil {
		return nil, err
	}
	return downloaders, nil
}

//LoadConfig returns a Config from a config reader
func LoadConfig(config io.Reader) (*Config, error) {
	var cfg Config
	err := json.NewDecoder(config).Decode(&cfg)
	return &cfg, err
}

//Config is downloaders configuration
type Config struct {
	Downloaders map[string][]Downloader `json:"downloaders,omitempty"`
}

func downloadersToInterface(dls map[string][]*bindown.Downloader) map[string][]Downloader {
	result := make(map[string][]Downloader, len(dls))
	for key, downloaders := range dls {
		result[key] = make([]Downloader, len(downloaders))
		for i, downloader := range downloaders {
			result[key][i] = downloader
		}
	}
	return result
}

//UnmarshalJSON implements json.Unmarshaler
func (c *Config) UnmarshalJSON(p []byte) error {
	jsm := struct {
		Downloaders map[string][]*bindown.Downloader `json:"downloaders,omitempty"`
	}{}
	decoder := json.NewDecoder(bytes.NewReader(p))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&jsm)
	if err == nil {
		c.Downloaders = downloadersToInterface(jsm.Downloaders)
		return nil
	}
	dls, err := unmarshalDownloaders(p)
	if err != nil {
		return err
	}
	c.Downloaders = downloadersToInterface(dls)
	return nil
}

// Downloader returns a Downloader for the given binary, os and arch.
func (c *Config) Downloader(binary, opSys, arch string) Downloader {
	l, ok := c.Downloaders[binary]
	if !ok {
		return nil
	}
	for _, d := range l {
		if d.MatchesOS(opSys) && d.MatchesArch(arch) {
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
		err = downloader.UpdateChecksum(cellarDir)
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
				fmt.Errorf("error validating %s: %v", downloader.ErrString(binary), err),
			)
		}
	}
	return resErr
}

//File represents a config file
type File struct {
	filename string
	*Config
}

//NewConfigFile creates a *File for the file at filename
func NewConfigFile(filename string) (*File, error) {
	b, err := ioutil.ReadFile(filename) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("couldn't read config file: %s", filename)
	}
	config, err := LoadConfig(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("couldn't load config: %v", err)
	}
	return &File{
		filename: filename,
		Config:   config,
	}, nil
}

//WriteFile writes config back to the file
func (c *File) WriteFile() error {
	b, err := json.MarshalIndent(c.Config, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(c.filename, b, 0640)
}
