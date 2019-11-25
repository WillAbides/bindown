package bindownloader

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

//LoadConfig returns a Config from a config reader
func LoadConfig(config io.Reader) (Config, error) {
	var dls Config
	err := json.NewDecoder(config).Decode(&dls)
	if err != nil {
		return nil, err
	}
	return dls, nil
}

//LoadConfigFile returns a Config from the path to a config file
func LoadConfigFile(configFile string) (Config, error) {
	configReader, err := os.Open(configFile) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("couldn't read config file: %s", configFile)
	}
	defer logCloseErr(configReader)
	return LoadConfig(configReader)
}

// Config map binary names to Config
type Config map[string][]*Downloader

// Downloader returns a Downloader for the given binary, os and arch.
func (c Config) Downloader(binary, os, arch string) *Downloader {
	l, ok := c[binary]
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
