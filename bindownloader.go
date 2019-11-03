package bindownloader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
)

//LoadConfig returns a Downloaders from a config reader
func LoadConfig(config io.Reader) (Downloaders, error) {
	var dls Downloaders
	err := json.NewDecoder(config).Decode(&dls)
	if err != nil {
		return nil, err
	}
	return dls, nil
}

//LoadConfigFile returns a Downloaders from the path to a config file
func LoadConfigFile(configFile string) (Downloaders, error) {
	configBytes, err := ioutil.ReadFile(configFile) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("couldn't read config file: %s", configFile)
	}
	return LoadConfig(bytes.NewReader(configBytes))
}
