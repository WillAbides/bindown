package bindown

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfigFile(t *testing.T) {
	t.Run("json1", func(t *testing.T) {
		cfgPath := projectPath("testdata", "configs", "ex1.json")
		cfg, err := LoadConfigFile(cfgPath)
		assert.NoError(t, err)
		assert.Equal(t, "darwin-amd64", cfg.Downloaders["gobin"][0].ArchivePath)
		assert.True(t, cfg.Downloaders["gobin"][0].Link)
		assert.Equal(t, formatJSON, cfg.format)
		assert.Equal(t, cfgPath, cfg.file)
	})

	t.Run("yaml1", func(t *testing.T) {
		cfgPath := projectPath("testdata", "configs", "ex1.yaml")
		cfg, err := LoadConfigFile(cfgPath)
		assert.NoError(t, err)
		assert.Equal(t, "darwin-amd64", cfg.Downloaders["gobin"][0].ArchivePath)
		assert.True(t, cfg.Downloaders["gobin"][0].Link)
		assert.Equal(t, formatYAML, cfg.format)
		assert.Equal(t, cfgPath, cfg.file)
	})

	t.Run("downloadersonly", func(t *testing.T) {
		cfgPath := projectPath("testdata", "configs", "downloadersonly.json")
		cfg, err := LoadConfigFile(cfgPath)
		assert.NoError(t, err)
		assert.Equal(t, "darwin-amd64", cfg.Downloaders["gobin"][0].ArchivePath)
		assert.True(t, cfg.Downloaders["gobin"][0].Link)
		assert.Equal(t, formatJSON, cfg.format)
		assert.Equal(t, cfgPath, cfg.file)
	})
}
