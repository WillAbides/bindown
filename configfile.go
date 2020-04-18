package bindown

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v2"
)

//ConfigFile is a file containing config
type ConfigFile struct {
	Filename string
	Config
}

//LoadConfigFile loads a config file
func LoadConfigFile(filename string, noDefaultDirs bool) (*ConfigFile, error) {
	data, err := ioutil.ReadFile(filename) //nolint:gosec
	if err != nil {
		return nil, err
	}
	cfg, err := configFromYAML(data)
	if err != nil {
		return nil, err
	}
	result := ConfigFile{
		Filename: filename,
		Config:   *cfg,
	}
	configDir := filepath.Dir(filename)
	if noDefaultDirs {
		return &result, nil
	}
	if result.Cache == "" {
		result.Cache = filepath.Join(configDir, ".bindown")
	}
	if result.InstallDir == "" {
		result.InstallDir = filepath.Join(configDir, "bin")
	}
	return &result, nil
}

//Write writes a file to disk
func (c *ConfigFile) Write(outputJSON bool) error {
	var data []byte
	var err error
	if filepath.Ext(c.Filename) == ".json" {
		outputJSON = true
	}
	if len(c.Systems) > 0 {
		sort.Slice(c.Systems, func(i, j int) bool {
			return c.Systems[i].String() < c.Systems[j].String()
		})
	}
	switch outputJSON {
	case true:
		data, err = json.MarshalIndent(&c.Config, "", "  ")
	case false:
		data, err = yaml.Marshal(&c.Config)
	}
	if err != nil {
		return err
	}

	return ioutil.WriteFile(c.Filename, data, 0600)
}
