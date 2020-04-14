package bindown

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v3/internal/testutil"
	"github.com/willabides/bindown/v3/internal/util"
	"gopkg.in/yaml.v2"
)

func configFromYaml(t *testing.T, yml string) *Config {
	t.Helper()
	got := new(Config)
	err := yaml.UnmarshalStrict([]byte(yml), got)
	require.NoError(t, err)
	return got
}

func TestConfig_InstallDependency(t *testing.T) {
	t.Run("raw file", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		servePath := testutil.DownloadablesPath("rawfile/foo")
		ts := testutil.ServeFile(t, servePath, "/foo/foo", "")
		depURL := ts.URL + "/foo/foo"
		binDir := filepath.Join(dir, "bin")
		util.Must(os.MkdirAll(binDir, 0755))
		cacheDir := filepath.Join(dir, ".bindown")
		config := &Config{
			InstallDir: binDir,
			Cache:      cacheDir,
			URLChecksums: map[string]string{
				depURL: "f044ff8b6007c74bcc1b5a5c92776e5d49d6014f5ff2d551fab115c17f48ac41",
			},
			Dependencies: map[string]*Dependency{
				"foo": {
					URL: &depURL,
				},
			},
		}
		wantBin := filepath.Join(binDir, "foo")
		gotPath, err := config.InstallDependency("foo", newSystemInfo("darwin", "amd64"), &ConfigInstallDependencyOpts{})
		require.NoError(t, err)
		require.Equal(t, wantBin, gotPath)
		require.True(t, util.FileExists(wantBin))
		stat, err := os.Stat(wantBin)
		require.NoError(t, err)
		require.False(t, stat.IsDir())
		require.Equal(t, os.FileMode(0750), stat.Mode().Perm())
	})

	t.Run("bin in root", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		servePath := testutil.DownloadablesPath("fooinroot.tar.gz")
		ts := testutil.ServeFile(t, servePath, "/foo/fooinroot.tar.gz", "")
		depURL := ts.URL + "/foo/fooinroot.tar.gz"
		binDir := filepath.Join(dir, "bin")
		util.Must(os.MkdirAll(binDir, 0755))
		cacheDir := filepath.Join(dir, ".bindown")
		config := &Config{
			InstallDir: binDir,
			Cache:      cacheDir,
			URLChecksums: map[string]string{
				depURL: "27dcce60d1ed72920a84dd4bc01e0bbd013e5a841660e9ee2e964e53fb83c0b3",
			},
			Dependencies: map[string]*Dependency{
				"foo": {
					URL: &depURL,
				},
			},
		}
		wantBin := filepath.Join(binDir, "foo")
		gotPath, err := config.InstallDependency("foo", newSystemInfo("darwin", "amd64"), &ConfigInstallDependencyOpts{})
		require.NoError(t, err)
		require.Equal(t, wantBin, gotPath)
		require.True(t, util.FileExists(wantBin))
		stat, err := os.Stat(wantBin)
		require.NoError(t, err)
		require.False(t, stat.IsDir())
		require.Equal(t, os.FileMode(0750), stat.Mode().Perm())
	})

	t.Run("wrong checksum", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		servePath := testutil.DownloadablesPath("fooinroot.tar.gz")
		ts := testutil.ServeFile(t, servePath, "/foo/fooinroot.tar.gz", "")
		depURL := ts.URL + "/foo/fooinroot.tar.gz"
		binDir := filepath.Join(dir, "bin")
		util.Must(os.MkdirAll(binDir, 0755))
		cacheDir := filepath.Join(dir, ".bindown")
		config := &Config{
			InstallDir: binDir,
			Cache:      cacheDir,
			URLChecksums: map[string]string{
				depURL: "0000000000000000000000000000000000000000000000000000000000000000",
			},
			Dependencies: map[string]*Dependency{
				"foo": {
					URL: &depURL,
				},
			},
		}
		wantBin := filepath.Join(binDir, "foo")
		_, err := config.InstallDependency("foo", newSystemInfo("darwin", "amd64"), &ConfigInstallDependencyOpts{})
		require.Error(t, err)
		require.False(t, util.FileExists(wantBin))
	})
}

func TestConfig_addChecksums(t *testing.T) {
	ts1 := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
	ts2 := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
	ts3 := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
	ts4 := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
	ts5 := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
	dl1 := ts1.URL + "/foo/foo.tar.gz"
	dl2 := ts2.URL + "/foo/foo.tar.gz"
	dl3 := ts3.URL + "/foo/foo.tar.gz"
	dl4 := ts4.URL + "/foo/foo.tar.gz"
	dl5 := ts5.URL + "/foo/foo.tar.gz"
	cfg := &Config{
		Dependencies: map[string]*Dependency{
			"d1": {
				URL: stringPtr(dl1),
				Overrides: []DependencyOverride{
					{
						Dependency:      Dependency{URL: stringPtr(dl2)},
						OverrideMatcher: OverrideMatcher{OS: []string{"darwin"}},
					},
					{
						Dependency:      Dependency{URL: stringPtr(dl5)},
						OverrideMatcher: OverrideMatcher{OS: []string{"windows"}},
					},
				},
			},
			"d2": {
				URL: stringPtr(dl3),
				Overrides: []DependencyOverride{
					{
						Dependency:      Dependency{URL: stringPtr(dl4)},
						OverrideMatcher: OverrideMatcher{OS: []string{"darwin"}},
					},
				},
			},
		},
	}
	err := cfg.AddChecksums(&ConfigAddChecksumsOptions{
		Systems: []SystemInfo{
			newSystemInfo("darwin", "amd64"),
			newSystemInfo("linux", "amd64"),
		},
	})
	require.NoError(t, err)
	require.Len(t, cfg.URLChecksums, 4)
	require.Equal(t, map[string]string{
		dl1: testutil.FooChecksum,
		dl2: testutil.FooChecksum,
		dl3: testutil.FooChecksum,
		dl4: testutil.FooChecksum,
	}, cfg.URLChecksums)
}

func TestConfig_addChecksum(t *testing.T) {
	ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
	dlURL := ts.URL + "/foo/foo.tar.gz"
	cfg := &Config{
		Dependencies: map[string]*Dependency{
			"dut": {
				URL: stringPtr(dlURL),
			},
		},
	}
	want := &Config{
		Dependencies: map[string]*Dependency{
			"dut": {
				URL: stringPtr(dlURL),
			},
		},
		URLChecksums: map[string]string{
			dlURL: testutil.FooChecksum,
		},
	}
	err := cfg.addChecksum("dut", newSystemInfo("testOS", "testArch"))
	require.NoError(t, err)
	require.Equal(t, want, cfg)
}
