package bindown

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v2"
)

// ConfigFile is a file containing config
type ConfigFile struct {
	Filename string
	Config
}

// LoadConfigFile loads a config file
func LoadConfigFile(ctx context.Context, filename string, noDefaultDirs bool) (*ConfigFile, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	cfg, err := configFromYAML(ctx, data)
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

func (c *ConfigFile) writeContent(w io.Writer, outputJSON bool) error {
	if len(c.Systems) > 0 {
		sort.Slice(c.Systems, func(i, j int) bool {
			return c.Systems[i].String() < c.Systems[j].String()
		})
	}
	if outputJSON {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(c)
	}
	encoder := yaml.NewEncoder(w)
	err := encoder.Encode(&c.Config)
	if err != nil {
		return err
	}
	return encoder.Close()
}

// Write writes a file to disk
func (c *ConfigFile) Write(outputJSON bool) error {
	if filepath.Ext(c.Filename) == ".json" {
		outputJSON = true
	}
	file, err := os.Create(c.Filename)
	if err != nil {
		return err
	}
	err = c.writeContent(file, outputJSON)
	if err != nil {
		return err
	}
	return file.Close()
}
