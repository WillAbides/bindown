package cli

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_findConfigFileForCompletion(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Run("missing default", func(t *testing.T) {
			got := findConfigFileForCompletion([]string{})
			assert.Equal(t, "", got)
		})

		t.Run("exists", func(t *testing.T) {
			dir := t.TempDir()
			configFile := filepath.Join(dir, "bindown.yml")
			err := os.WriteFile(configFile, nil, 0o600)
			require.NoError(t, err)
			want, err := os.ReadFile(configFile)
			require.NoError(t, err)
			require.NoError(t, os.Chdir(dir))
			got := findConfigFileForCompletion([]string{})
			gotContent, err := os.ReadFile(got)
			require.NoError(t, err)
			assert.Equal(t, string(want), string(gotContent))
		})
	})

	t.Run("from command line", func(t *testing.T) {
		configFile := createConfigFile(t, "ex1.yaml")
		got := findConfigFileForCompletion([]string{"foo", "--configfile", configFile, "bar"})
		assert.Equal(t, configFile, got)
	})

	t.Run("from environment variable", func(t *testing.T) {
		configFile := createConfigFile(t, "ex1.yaml")
		setConfigFileEnvVar(t, configFile)
		got := findConfigFileForCompletion([]string{})
		assert.Equal(t, configFile, got)
	})
}

func Test_completionConfig(t *testing.T) {
	t.Run("no config file", func(t *testing.T) {
		got := completionConfig([]string{})
		assert.Nil(t, got)
	})

	t.Run("valid config file", func(t *testing.T) {
		configFile := createConfigFile(t, "ex1.yaml")
		setConfigFileEnvVar(t, configFile)
		got := completionConfig(nil)
		assert.NotNil(t, got)
		assert.NotNil(t, got.Dependencies["golangci-lint"])
	})

	t.Run("empty config file", func(t *testing.T) {
		dir := t.TempDir()
		configFile := filepath.Join(dir, "bindown.yml")
		err := os.WriteFile(configFile, []byte("no valid yaml here"), 0o600)
		require.NoError(t, err)
		require.NoError(t, os.Chdir(dir))
		got := completionConfig(nil)
		assert.Nil(t, got)
	})
}

func Test_binCompleter(t *testing.T) {
	got := binCompleter.Options(kong.CompleterArgs{})
	assert.Empty(t, got)
	assert.NotNil(t, got)

	configFile := createConfigFile(t, "ex1.yaml")
	setConfigFileEnvVar(t, configFile)
	got = binCompleter.Options(kong.CompleterArgs{})
	sort.Strings(got)
	assert.Equal(t, []string{"golangci-lint", "goreleaser"}, got)
}
