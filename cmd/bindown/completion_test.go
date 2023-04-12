package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/posener/complete"
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
	ctx := context.Background()
	t.Run("no config file", func(t *testing.T) {
		got := completionConfig(ctx, []string{})
		require.Nil(t, got)
	})

	t.Run("valid config file", func(t *testing.T) {
		configFile := createConfigFile(t, "ex1.yaml")
		setConfigFileEnvVar(t, configFile)
		got := completionConfig(ctx, nil)
		require.NotNil(t, got)
		require.NotNil(t, got.Dependencies["golangci-lint"])
	})

	t.Run("empty config file", func(t *testing.T) {
		dir := t.TempDir()
		configFile := filepath.Join(dir, "bindown.yml")
		err := os.WriteFile(configFile, []byte("no valid yaml here"), 0o600)
		require.NoError(t, err)
		inDir(t, dir, func() {
			got := completionConfig(ctx, nil)
			require.Nil(t, got)
		})
	})
}

func Test_binCompleter(t *testing.T) {
	ctx := context.Background()
	got := binCompleter(ctx).Predict(complete.Args{})
	require.Empty(t, got)
	require.NotNil(t, got)

	configFile := createConfigFile(t, "ex1.yaml")
	setConfigFileEnvVar(t, configFile)
	got = binCompleter(ctx).Predict(complete.Args{})
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

func setConfigFileEnvVar(t *testing.T, file string) {
	t.Helper()
	err := os.Setenv("BINDOWN_CONFIG_FILE", file)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("BINDOWN_CONFIG_FILE"))
	})
}

func createConfigFile(t *testing.T, sourceFile string) string {
	t.Helper()
	sourceFile = testdataPath("configs/" + sourceFile)
	dir := t.TempDir()
	dest := filepath.Join(dir, "bindown.config")
	copyFile(t, sourceFile, dest)
	return dest
}

func copyFile(t *testing.T, sourceFile, destFile string) {
	t.Helper()
	source, err := os.Open(sourceFile)
	require.NoError(t, err)
	dest, err := os.Create(destFile)
	require.NoError(t, err)
	_, err = io.Copy(dest, source)
	require.NoError(t, err)
	require.NoError(t, source.Close())
	require.NoError(t, dest.Close())
}
