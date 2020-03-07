package bindown

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v2/internal/testutil"
	"gopkg.in/yaml.v2"
)

func TestConfigFile_Write(t *testing.T) {
	config := Config{
		Downloaders: map[string][]*Downloader{
			"foo": {
				{
					OS:          "windows",
					Arch:        "amd64",
					URL:         "http://fake",
					Checksum:    "deadbeef",
					ArchivePath: "foo/foo.exe",
					Link:        true,
					BinName:     "foo.exe",
				},
			},
		},
	}
	t.Run("json", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		file := filepath.Join(dir, "bindown.json")
		err := ioutil.WriteFile(file, []byte("overwrite me"), 0600)
		require.NoError(t, err)
		configFile := &ConfigFile{
			format: formatJSON,
			file:   file,
			Config: config,
		}
		err = configFile.Write()
		require.NoError(t, err)
		got, err := ioutil.ReadFile(file)
		require.NoError(t, err)
		var gotConfig Config
		err = json.Unmarshal(got, &gotConfig)
		require.NoError(t, err)
		assert.Equal(t, config, gotConfig)
	})

	t.Run("yaml", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		file := filepath.Join(dir, "bindown.json")
		err := ioutil.WriteFile(file, []byte("overwrite me"), 0600)
		require.NoError(t, err)
		configFile := &ConfigFile{
			format: formatYAML,
			file:   file,
			Config: config,
		}
		err = configFile.Write()
		require.NoError(t, err)
		got, err := ioutil.ReadFile(file)
		require.NoError(t, err)
		var gotConfig Config
		err = yaml.Unmarshal(got, &gotConfig)
		require.NoError(t, err)
		assert.Equal(t, config, gotConfig)
	})
}

func TestLoadConfigFile(t *testing.T) {
	t.Run("json1", func(t *testing.T) {
		cfgPath := testutil.ProjectPath("testdata", "configs", "ex1.json")
		cfg, err := LoadConfigFile(cfgPath)
		assert.NoError(t, err)
		assert.Equal(t, "darwin-amd64", cfg.Downloaders["gobin"][0].ArchivePath)
		assert.True(t, cfg.Downloaders["gobin"][0].Link)
		assert.Equal(t, formatJSON, cfg.format)
		assert.Equal(t, cfgPath, cfg.file)
	})

	t.Run("yaml1", func(t *testing.T) {
		cfgPath := testutil.ProjectPath("testdata", "configs", "ex1.yaml")
		cfg, err := LoadConfigFile(cfgPath)
		assert.NoError(t, err)
		assert.Equal(t, "darwin-amd64", cfg.Downloaders["gobin"][0].ArchivePath)
		assert.True(t, cfg.Downloaders["gobin"][0].Link)
		assert.Equal(t, formatYAML, cfg.format)
		assert.Equal(t, cfgPath, cfg.file)
	})

	t.Run("downloadersonly", func(t *testing.T) {
		cfgPath := testutil.ProjectPath("testdata", "configs", "downloadersonly.json")
		cfg, err := LoadConfigFile(cfgPath)
		assert.NoError(t, err)
		assert.Equal(t, "darwin-amd64", cfg.Downloaders["gobin"][0].ArchivePath)
		assert.True(t, cfg.Downloaders["gobin"][0].Link)
		assert.Equal(t, formatJSON, cfg.format)
		assert.Equal(t, cfgPath, cfg.file)
	})
}

func TestConfig_Downloader(t *testing.T) {
	config := Config{
		Downloaders: map[string][]*Downloader{
			"foo": {
				{
					OS:          "windows",
					Arch:        "amd64",
					URL:         "http://fake/windows",
					Checksum:    "deadbeef",
					ArchivePath: "foo/foo.exe",
					BinName:     "foo.exe",
				},
				{
					OS:          "darwin",
					Arch:        "amd64",
					URL:         "http://fake/darwin",
					Checksum:    "deadbeef",
					ArchivePath: "foo/foo",
				},
			},
		},
	}

	got := config.Downloader("foo", "windows", "amd64")
	assert.Equal(t, "http://fake/windows", got.URL)
	got = config.Downloader("foo", "Windows", "AMD64")
	assert.Equal(t, "http://fake/windows", got.URL)
	got = config.Downloader("foo", "linux", "amd64")
	assert.Nil(t, got)
}
