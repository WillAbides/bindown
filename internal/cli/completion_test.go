package cli

import (
	"io/ioutil"
	"path/filepath"
	"sort"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v2"
	"github.com/willabides/bindown/v2/internal/testutil"
)

func Test_findConfigFileForCompletion(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Run("missing default", func(t *testing.T) {
			got := findConfigFileForCompletion([]string{})
			assert.Equal(t, "", got)
		})

		t.Run("exists", func(t *testing.T) {
			dir := testutil.TmpDir(t)
			configFile := filepath.Join(dir, "bindown.yml")
			err := ioutil.WriteFile(configFile, nil, 0600)
			require.NoError(t, err)
			testutil.ChDir(t, dir)
			got := findConfigFileForCompletion([]string{})
			assert.Equal(t, configFile, got)
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
		assert.NotNil(t, got.Downloader("gobin", "darwin", "amd64"))
	})

	t.Run("empty config file", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		configFile := filepath.Join(dir, "bindown.yml")
		err := ioutil.WriteFile(configFile, []byte("no valid yaml here"), 0600)
		require.NoError(t, err)
		testutil.ChDir(t, dir)
		got := completionConfig(nil)
		assert.Nil(t, got)
	})
}

func Test_allBins(t *testing.T) {
	assert.Equal(t, []string{}, allBins(nil))

	cfg := &bindown.ConfigFile{
		Config: bindown.Config{
			Downloaders: map[string][]*bindown.Downloader{
				"foo": {{}},
				"bar": {
					{BinName: "baz"},
					{BinName: "qux"},
					{BinName: "qux"},
					{BinName: "qux"},
				},
			},
		},
	}

	got := allBins(cfg)
	sort.Strings(got)
	assert.Equal(t, []string{"baz", "foo", "qux"}, got)
}

func Test_binCompleter(t *testing.T) {
	got := binCompleter.Options(kong.CompleterArgs{})
	assert.Empty(t, got)
	assert.NotNil(t, got)

	configFile := createConfigFile(t, "ex1.yaml")
	setConfigFileEnvVar(t, configFile)
	got = binCompleter.Options(kong.CompleterArgs{})
	assert.Equal(t, []string{"gobin"}, got)
}

func Test_binPathCompleter(t *testing.T) {
	got := binPathCompleter.Options(kong.CompleterArgs{"foo", "bar/baz/"})
	assert.Empty(t, got)
	assert.NotNil(t, got)

	configFile := createConfigFile(t, "ex1.yaml")
	setConfigFileEnvVar(t, configFile)
	got = binPathCompleter.Options(kong.CompleterArgs{"foo", "bar/baz/"})
	assert.Equal(t, []string{"bar/baz/gobin"}, got)
}
