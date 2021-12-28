package bindown

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/require"
)

func assertValidationErr(t *testing.T, want []string, got error) {
	t.Helper()
	wantErr := fmt.Sprintf("invalid config:\n%s", strings.Join(want, "\n"))
	require.EqualError(t, got, wantErr)
}

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
		cfg, err := yaml.YAMLToJSON(cfgContent)
		require.NoError(t, err)
		err = validateConfig(ctx, cfg)
		require.NoError(t, err)
	})

	t.Run("empty", func(t *testing.T) {
		cfg := []byte("")
		err := validateConfig(ctx, cfg)
		assertValidationErr(t, []string{
			`/: type should be object, got null`,
		}, err)
	})

	t.Run("invalid yaml", func(t *testing.T) {
		cfg := []byte(`
dependencies:
  golangci-lint: surprise string
  goreleaser:
    template: goreleaser
    vars:
      version: 12
url_checksums:
  foo: deadbeef
  bar: []
`)
		wantErrs := []string{
			`/dependencies/golangci-lint: "surprise string" type should be object, got string`,
			`/url_checksums/bar: [] type should be string, got array`,
		}
		err := validateConfig(ctx, cfg)
		assertValidationErr(t, wantErrs, err)
	})

	t.Run("invalid json", func(t *testing.T) {
		cfg := []byte(`
{
  "dependencies": {
    "golangci-lint": "surprise string",
    "goreleaser": {
      "template": "goreleaser",
      "vars": {
        "version": 12
      }
    }
  },
  "url_checksums": {
    "foo": "deadbeef",
    "bar": [

    ]
  }
}`)
		wantErrs := []string{
			`/dependencies/golangci-lint: "surprise string" type should be object, got string`,
			`/url_checksums/bar: [] type should be string, got array`,
		}
		err := validateConfig(ctx, cfg)
		assertValidationErr(t, wantErrs, err)
	})
}
