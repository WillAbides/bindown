package bindown

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v3/internal/testutil"
	"gopkg.in/yaml.v2"
)

func configFromYaml(t *testing.T, yml string) *Config {
	t.Helper()
	got := new(Config)
	err := yaml.UnmarshalStrict([]byte(yml), got)
	require.NoError(t, err)
	return got
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
						Dependency{URL: stringPtr(dl2)},
						OverrideMatcher{OS: []string{"darwin"}},
					},
					{
						Dependency{URL: stringPtr(dl5)},
						OverrideMatcher{OS: []string{"windows"}},
					},
				},
			},
			"d2": {
				URL: stringPtr(dl3),
				Overrides: []DependencyOverride{
					{
						Dependency{URL: stringPtr(dl4)},
						OverrideMatcher{OS: []string{"darwin"}},
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
