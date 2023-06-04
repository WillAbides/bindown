package bindown

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestValidateConfig(t *testing.T) {
	t.Run("valid yaml", func(t *testing.T) {
		cfg, err := os.ReadFile(filepath.Join("testdata", "configs", "ex1.yaml"))
		require.NoError(t, err)
		err = validateConfig(cfg)
		require.NoError(t, err)
	})

	t.Run("valid json", func(t *testing.T) {
		cfgContent, err := os.ReadFile(filepath.Join("testdata", "configs", "ex1.yaml"))
		require.NoError(t, err)
		var data any
		err = yaml.Unmarshal(cfgContent, &data)
		require.NoError(t, err)
		cfg, err := json.Marshal(data)
		require.NoError(t, err)
		require.NoError(t, err)
		err = validateConfig(cfg)
		require.NoError(t, err)
	})

	t.Run("empty", func(t *testing.T) {
		cfg := []byte("")
		err := validateConfig(cfg)
		require.Error(t, err)
	})

	t.Run("invalid yaml", func(t *testing.T) {
		cfg := []byte(`
dependencies:
  golangci-lint: surprise string
  goreleaser:
    template: goreleaser
    vars:
      version: 1.2.3
url_checksums:
  foo: deadbeef
  bar: []
`)
		err := validateConfig(cfg)
		require.Error(t, err)
	})

	t.Run("invalid json", func(t *testing.T) {
		cfg := []byte(`
{
  "dependencies": {
    "golangci-lint": "surprise string",
    "goreleaser": {
      "template": "goreleaser",
      "vars": {
        "version": "1.2.3"
      }
    }
  },
  "url_checksums": {
    "foo": "deadbeef",
    "bar": [

    ]
  }
}`)
		err := validateConfig(cfg)
		require.Error(t, err)
	})
}
