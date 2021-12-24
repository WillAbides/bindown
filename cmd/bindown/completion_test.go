package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/require"
)

func Test_findConfigFileForCompletion(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Run("missing default", func(t *testing.T) {
			got := findConfigFileForCompletion([]string{})
			require.Equal(t, "", got)
		})

		t.Run("exists", func(t *testing.T) {
			dir := t.TempDir()
			configFile := filepath.Join(dir, "bindown.yml")
			err := os.WriteFile(configFile, nil, 0o600)
			require.NoError(t, err)
			want, err := os.ReadFile(configFile)
			require.NoError(t, err)
			inDir(t, dir, func() {
				got := findConfigFileForCompletion([]string{})
				gotContent, err := os.ReadFile(got)
				require.NoError(t, err)
				require.Equal(t, string(want), string(gotContent))
			})
		})
	})

	t.Run("from command line", func(t *testing.T) {
		configFile := createConfigFile(t, "ex1.yaml")
		got := findConfigFileForCompletion([]string{"foo", "--configfile", configFile, "bar"})
		require.Equal(t, configFile, got)
	})

	t.Run("from environment variable", func(t *testing.T) {
		configFile := createConfigFile(t, "ex1.yaml")
		setConfigFileEnvVar(t, configFile)
		got := findConfigFileForCompletion([]string{})
		require.Equal(t, configFile, got)
	})
}

func Test_completionConfig(t *testing.T) {
	t.Run("no config file", func(t *testing.T) {
		got := completionConfig([]string{})
		require.Nil(t, got)
	})

	t.Run("valid config file", func(t *testing.T) {
		configFile := createConfigFile(t, "ex1.yaml")
		setConfigFileEnvVar(t, configFile)
		got := completionConfig(nil)
		require.NotNil(t, got)
		require.NotNil(t, got.Dependencies["golangci-lint"])
	})

	t.Run("empty config file", func(t *testing.T) {
		dir := t.TempDir()
		configFile := filepath.Join(dir, "bindown.yml")
		err := os.WriteFile(configFile, []byte("no valid yaml here"), 0o600)
		require.NoError(t, err)
		inDir(t, dir, func() {
			got := completionConfig(nil)
			require.Nil(t, got)
		})
	})
}

func Test_binCompleter(t *testing.T) {
	got := binCompleter.Options(kong.CompleterArgs{})
	require.Empty(t, got)
	require.NotNil(t, got)

	configFile := createConfigFile(t, "ex1.yaml")
	setConfigFileEnvVar(t, configFile)
	got = binCompleter.Options(kong.CompleterArgs{})
	sort.Strings(got)
	require.Equal(t, []string{"golangci-lint", "goreleaser"}, got)
}

// inDir runs f in the given directory.
func inDir(t *testing.T, dir string, f func()) {
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(dir)
	require.NoError(t, err)
	f()
	err = os.Chdir(oldDir)
	require.NoError(t, err)
}
