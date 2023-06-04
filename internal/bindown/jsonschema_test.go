package bindown

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateConfig(t *testing.T) {
	ctx := context.Background()
	t.Run("valid yaml", func(t *testing.T) {
		cfg, err := os.ReadFile(filepath.Join("testdata", "configs", "ex1.yaml"))
		require.NoError(t, err)
		err = validateConfig(ctx, cfg)
		require.NoError(t, err)
	})

	t.Run("valid json", func(t *testing.T) {
		cfgContent, err := os.ReadFile(filepath.Join("testdata", "configs", "ex1.yaml"))
		require.NoError(t, err)
		cfg, err := yaml2json(cfgContent)
		require.NoError(t, err)
		err = validateConfig(ctx, cfg)
		require.NoError(t, err)
	})

	t.Run("empty", func(t *testing.T) {
		cfg := []byte("")
		err := validateConfig(ctx, cfg)
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
		err := validateConfig(ctx, cfg)
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
		err := validateConfig(ctx, cfg)
		require.Error(t, err)
	})
}
