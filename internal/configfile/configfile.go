package configfile

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/willabides/bindown/v3"
	"github.com/willabides/bindown/v3/internal/jsonschema"
	"gopkg.in/yaml.v2"
)

//ConfigFile is a file containing config
type ConfigFile struct {
	filename string
	bindown.Config
}

//New returns a new *ConfigFile
func New(filename string, config bindown.Config) *ConfigFile {
	return &ConfigFile{
		filename: filename,
		Config:   config,
	}
}

//LoadConfigFile loads a config file
func LoadConfigFile(filename string) (*ConfigFile, error) {
	data, err := ioutil.ReadFile(filename) //nolint:gosec
	if err != nil {
		return nil, err
	}
	result := ConfigFile{
		filename: filename,
	}
	err = jsonschema.ValidateConfig(data)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &result.Config)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

//Write writes a file to disk
func (c *ConfigFile) Write(outputJSON bool) error {
	var data []byte
	var err error
	if filepath.Ext(c.filename) == ".json" {
		outputJSON = true
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
	return ioutil.WriteFile(c.filename, data, 0600)
}
