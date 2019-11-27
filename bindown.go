package bindown

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
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
	defer logCloseErr(configReader)
	return LoadConfig(configReader)
}

//Config is downloaders configuration
type Config struct {
	Downloaders map[string][]*Downloader `json:"downloaders,omitempty"`
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
