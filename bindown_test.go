package bindown

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v3/internal/testutil"
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

	t.Run("file doesn't exist", func(t *testing.T) {
		cfg, err := LoadConfigFile(testutil.ProjectPath("testdata/configs/fake.yaml"))
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
		require.Nil(t, cfg)
	})

	t.Run("invalid", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		file, err := os.Create(filepath.Join(dir, "ex1.yaml"))
		require.NoError(t, err)
		_, err = file.WriteString("foo")
		require.NoError(t, err)
		require.NoError(t, file.Close())
		cfg, err := LoadConfigFile(file.Name())
		require.Error(t, err)
		require.Nil(t, cfg)
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
	got = config.Downloader("fake", "windows", "amd64")
	assert.Nil(t, got)
}

func TestConfig_UpdateChecksums(t *testing.T) {
	ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "foo=bar")
	dlURL := ts.URL + "/foo/foo.tar.gz?foo=bar"
	getConfig := func() Config {
		return Config{
			Downloaders: map[string][]*Downloader{
				"foo": {
					{
						OS:          "windows",
						Arch:        "amd64",
						URL:         dlURL,
						Checksum:    "deadbeef",
						ArchivePath: "bin/foo.txt",
						BinName:     "foo.exe",
					},
					{
						OS:          "darwin",
						Arch:        "amd64",
						URL:         dlURL,
						Checksum:    "deadbeef",
						ArchivePath: "bin/foo.txt",
					},
				},
				"bar": {
					{
						OS:          "windows",
						Arch:        "amd64",
						URL:         dlURL,
						Checksum:    "deadbeef",
						ArchivePath: "bin/foo.txt",
						BinName:     "foo.exe",
					},
					{
						OS:          "darwin",
						Arch:        "amd64",
						URL:         dlURL,
						Checksum:    "deadbeef",
						ArchivePath: "bin/foo.txt",
					},
				},
			},
		}
	}
	t.Run("updates all", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		config := getConfig()
		err := config.UpdateChecksums("", dir)
		require.NoError(t, err)
		for _, downloaders := range config.Downloaders {
			for _, downloader := range downloaders {
				require.Equal(t, testutil.FooChecksum, downloader.Checksum)
			}
		}
	})

	t.Run("updates one", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		config := getConfig()
		err := config.UpdateChecksums("bar", dir)
		require.NoError(t, err)
		for _, downloader := range config.Downloaders["foo"] {
			require.Equal(t, "deadbeef", downloader.Checksum)
		}
		for _, downloader := range config.Downloaders["bar"] {
			require.Equal(t, testutil.FooChecksum, downloader.Checksum)
		}
	})

	t.Run("downloader doesn't exist", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		config := getConfig()
		err := config.UpdateChecksums("fake", dir)
		require.EqualError(t, err, `nothing configured for "fake"`)
	})

	t.Run("can't download", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		config := Config{
			Downloaders: map[string][]*Downloader{
				"foo": {
					{
						OS:          "windows",
						Arch:        "amd64",
						URL:         "https://invalidurl/foo.tar.gz",
						Checksum:    "deadbeef",
						ArchivePath: "bin/foo.txt",
						BinName:     "foo.exe",
					},
				},
			},
		}
		err := config.UpdateChecksums("foo", dir)
		require.Error(t, err)
	})
}

func TestConfig_Validate(t *testing.T) {
	ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "foo=bar")
	dlURL := ts.URL + "/foo/foo.tar.gz?foo=bar"
	validConfig := Config{
		Downloaders: map[string][]*Downloader{
			"foo": {
				{
					OS:          "windows",
					Arch:        "amd64",
					URL:         dlURL,
					Checksum:    testutil.FooChecksum,
					ArchivePath: "bin/foo.txt",
					BinName:     "foo.exe",
				},
				{
					OS:          "darwin",
					Arch:        "amd64",
					URL:         dlURL,
					Checksum:    testutil.FooChecksum,
					ArchivePath: "bin/foo.txt",
				},
			},
			"bar": {
				{
					OS:          "windows",
					Arch:        "amd64",
					URL:         dlURL,
					Checksum:    testutil.FooChecksum,
					ArchivePath: "bin/foo.txt",
					BinName:     "foo.exe",
				},
				{
					OS:          "darwin",
					Arch:        "amd64",
					URL:         dlURL,
					Checksum:    testutil.FooChecksum,
					ArchivePath: "bin/foo.txt",
				},
			},
		},
	}
	t.Run("all valid", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		config := validConfig
		err := config.Validate("", dir)
		require.NoError(t, err)
	})

	t.Run("one valid", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		config := validConfig
		err := config.Validate("bar", dir)
		require.NoError(t, err)
	})

	t.Run("downloader doesn't exist", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		config := validConfig
		err := config.Validate("fake", dir)
		require.EqualError(t, err, `nothing configured for "fake"`)
	})

	t.Run("bad checksum", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		config := Config{
			Downloaders: map[string][]*Downloader{
				"foo": {
					{
						OS:          "windows",
						Arch:        "amd64",
						URL:         dlURL,
						Checksum:    "deadbeef",
						ArchivePath: "bin/foo.txt",
						BinName:     "foo.exe",
					},
				},
			},
		}
		err := config.Validate("foo", dir)
		require.Error(t, err)
		require.True(t, strings.HasPrefix(err.Error(), `could not validate downloader:`))
	})
}

func Test_eqOS(t *testing.T) {
	require.True(t, eqOS("Darwin", "macOS"))
	require.True(t, eqOS("asdf", "ASDF"))
}
