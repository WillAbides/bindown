package bindown

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func configFromYaml(t *testing.T, yml string) *Config {
	t.Helper()
	got := new(Config)
	decoder := yaml.NewDecoder(strings.NewReader(yml))
	decoder.KnownFields(true)
	require.NoError(t, decoder.Decode(got))
	return got
}

func TestConfig_UnsetDependencyVars(t *testing.T) {
	t.Run("deletes", func(t *testing.T) {
		cfg := configFromYaml(t, `
dependencies:
  foo:
    vars:
      foo: bar
      baz: qux
`)
		want := map[string]string{
			"baz": "qux",
		}
		err := cfg.UnsetDependencyVars("foo", []string{"a", "foo", "b"})
		require.NoError(t, err)
		require.Equal(t, want, cfg.Dependencies["foo"].Vars)
	})

	t.Run("nil vars", func(t *testing.T) {
		cfg := configFromYaml(t, `
dependencies:
  foo: {}
`)
		err := cfg.UnsetDependencyVars("foo", []string{"a", "foo", "b"})
		require.NoError(t, err)
		require.Nil(t, cfg.Dependencies["foo"].Vars)
	})
}

func TestConfig_UnsetTemplateVars(t *testing.T) {
	t.Run("deletes", func(t *testing.T) {
		cfg := configFromYaml(t, `
templates:
  foo:
    vars:
      foo: bar
      baz: qux
`)
		want := map[string]string{
			"baz": "qux",
		}
		err := cfg.UnsetTemplateVars("foo", []string{"a", "foo", "b"})
		require.NoError(t, err)
		require.Equal(t, want, cfg.Templates["foo"].Vars)
	})

	t.Run("nil vars", func(t *testing.T) {
		cfg := configFromYaml(t, `
templates:
  foo: {}
`)
		err := cfg.UnsetTemplateVars("foo", []string{"a", "foo", "b"})
		require.NoError(t, err)
		require.Nil(t, cfg.Templates["foo"].Vars)
	})
}

func TestConfig_SetDependencyVars(t *testing.T) {
	t.Run("replaces and adds", func(t *testing.T) {
		cfg := configFromYaml(t, `
dependencies:
  foo:
    vars:
      foo: bar
      baz: qux
`)
		want := map[string]string{
			"foo": "a",
			"baz": "qux",
			"b":   "c",
		}
		err := cfg.SetDependencyVars("foo", map[string]string{
			"foo": "a",
			"b":   "c",
		})
		require.NoError(t, err)
		require.Equal(t, want, cfg.Dependencies["foo"].Vars)
	})

	t.Run("nil vars", func(t *testing.T) {
		cfg := configFromYaml(t, `
dependencies:
  foo: {}
`)
		want := map[string]string{
			"foo": "a",
			"b":   "c",
		}
		err := cfg.SetDependencyVars("foo", map[string]string{
			"foo": "a",
			"b":   "c",
		})
		require.NoError(t, err)
		require.Equal(t, want, cfg.Dependencies["foo"].Vars)
	})
}

func TestConfig_SetTemplateVars(t *testing.T) {
	t.Run("replaces and adds", func(t *testing.T) {
		cfg := configFromYaml(t, `
templates:
  foo:
    vars:
      foo: bar
      baz: qux
`)
		want := map[string]string{
			"foo": "a",
			"baz": "qux",
			"b":   "c",
		}
		err := cfg.SetTemplateVars("foo", map[string]string{
			"foo": "a",
			"b":   "c",
		})
		require.NoError(t, err)
		require.Equal(t, want, cfg.Templates["foo"].Vars)
	})

	t.Run("nil vars", func(t *testing.T) {
		cfg := configFromYaml(t, `
templates:
  foo: {}
`)
		want := map[string]string{
			"foo": "a",
			"b":   "c",
		}
		err := cfg.SetTemplateVars("foo", map[string]string{
			"foo": "a",
			"b":   "c",
		})
		require.NoError(t, err)
		require.Equal(t, want, cfg.Templates["foo"].Vars)
	})
}

func TestConfig_addTemplateFromSource(t *testing.T) {
	ctx := context.Background()
	t.Run("file", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			cfg := new(Config)
			src := filepath.Join("testdata", "configs", "ex1.yaml")
			srcCfg, err := LoadConfigFile(ctx, src, true)
			require.NoError(t, err)
			err = cfg.addTemplateFromSource(ctx, src, "goreleaser", "mygoreleaser")
			require.NoError(t, err)
			require.Equal(t, srcCfg.Templates["goreleaser"], cfg.Templates["mygoreleaser"])
		})

		t.Run("missing template", func(t *testing.T) {
			cfg := new(Config)
			src := filepath.Join("testdata", "configs", "ex1.yaml")
			err := cfg.addTemplateFromSource(ctx, src, "fake", "myfake")
			require.EqualError(t, err, `source has no template named "fake"`)
		})

		t.Run("missing file", func(t *testing.T) {
			cfg := new(Config)
			src := filepath.Join("testdata", "configs", "thisdoesnotexist.yaml")
			err := cfg.addTemplateFromSource(ctx, src, "fake", "myfake")
			require.Error(t, err)
			require.True(t, os.IsNotExist(err))
		})
	})

	t.Run("http", func(t *testing.T) {
		srcFile := filepath.Join("testdata", "configs", "ex1.yaml")
		ts := serveFile(t, srcFile, "/ex1.yaml", "")
		cfg := new(Config)
		src := ts.URL + "/ex1.yaml"
		srcCfg, err := LoadConfigFile(ctx, srcFile, true)
		require.NoError(t, err)
		err = cfg.addTemplateFromSource(ctx, src, "goreleaser", "mygoreleaser")
		require.NoError(t, err)
		require.Equal(t, srcCfg.Templates["goreleaser"], cfg.Templates["mygoreleaser"])
	})
}

func TestConfig_InstallDependency(t *testing.T) {
	t.Run("raw file", func(t *testing.T) {
		dir := t.TempDir()
		servePath := filepath.Join("testdata", "downloadables", filepath.FromSlash("rawfile/foo"))
		ts := serveFile(t, servePath, "/foo/foo", "")
		depURL := ts.URL + "/foo/foo"
		binDir := filepath.Join(dir, "bin")
		require.NoError(t, os.MkdirAll(binDir, 0o755))
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
		require.True(t, fileExists(wantBin))
		stat, err := os.Stat(wantBin)
		require.NoError(t, err)
		require.False(t, stat.IsDir())
		require.Equal(t, os.FileMode(0o750), stat.Mode().Perm()&0o750)
	})

	t.Run("bin in root", func(t *testing.T) {
		dir := t.TempDir()
		servePath := filepath.Join("testdata", "downloadables", "fooinroot.tar.gz")
		ts := serveFile(t, servePath, "/foo/fooinroot.tar.gz", "")
		depURL := ts.URL + "/foo/fooinroot.tar.gz"
		binDir := filepath.Join(dir, "bin")
		require.NoError(t, os.MkdirAll(binDir, 0o755))
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
		require.True(t, fileExists(wantBin))
		stat, err := os.Stat(wantBin)
		require.NoError(t, err)
		require.False(t, stat.IsDir())
		require.Equal(t, os.FileMode(0o750), stat.Mode().Perm()&0o750)
	})

	t.Run("wrong checksum", func(t *testing.T) {
		dir := t.TempDir()
		servePath := filepath.Join("testdata", "downloadables", "fooinroot.tar.gz")
		ts := serveFile(t, servePath, "/foo/fooinroot.tar.gz", "")
		depURL := ts.URL + "/foo/fooinroot.tar.gz"
		binDir := filepath.Join(dir, "bin")
		require.NoError(t, os.MkdirAll(binDir, 0o755))
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
		require.False(t, fileExists(wantBin))
	})
}

func TestConfig_addChecksums(t *testing.T) {
	ts1 := serveFile(t, filepath.Join("testdata", "downloadables", "foo.tar.gz"), "/foo/foo.tar.gz", "")
	ts2 := serveFile(t, filepath.Join("testdata", "downloadables", "foo.tar.gz"), "/foo/foo.tar.gz", "")
	ts3 := serveFile(t, filepath.Join("testdata", "downloadables", "foo.tar.gz"), "/foo/foo.tar.gz", "")
	ts4 := serveFile(t, filepath.Join("testdata", "downloadables", "foo.tar.gz"), "/foo/foo.tar.gz", "")
	ts5 := serveFile(t, filepath.Join("testdata", "downloadables", "foo.tar.gz"), "/foo/foo.tar.gz", "")
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
						OverrideMatcher: OverrideMatcher{"os": []string{"darwin"}},
					},
					{
						Dependency:      Dependency{URL: stringPtr(dl5)},
						OverrideMatcher: OverrideMatcher{"os": []string{"windows"}},
					},
				},
			},
			"d2": {
				URL: stringPtr(dl3),
				Overrides: []DependencyOverride{
					{
						Dependency:      Dependency{URL: stringPtr(dl4)},
						OverrideMatcher: OverrideMatcher{"os": []string{"darwin"}},
					},
				},
			},
		},
	}
	err := cfg.AddChecksums(nil, []SystemInfo{
		newSystemInfo("darwin", "amd64"),
		newSystemInfo("linux", "amd64"),
	})
	require.NoError(t, err)
	require.Len(t, cfg.URLChecksums, 4)
	require.Equal(t, map[string]string{
		dl1: fooChecksum,
		dl2: fooChecksum,
		dl3: fooChecksum,
		dl4: fooChecksum,
	}, cfg.URLChecksums)
}

func TestConfig_BuildDependency(t *testing.T) {
	cfg := &Config{
		Dependencies: map[string]*Dependency{
			"dut": {
				URL: stringPtr("https://{{.os}}"),
				Overrides: []DependencyOverride{
					{
						OverrideMatcher: OverrideMatcher{
							"arch": []string{"testArch"},
							"os":   []string{"testOS"},
						},
						Dependency: Dependency{
							URL: stringPtr("https://{{.os}}-{{.var1}}-{{.var2}}"),
							Vars: map[string]string{
								"var1": "overrideV1",
								"var2": "overrideV2",
							},
						},
					},
				},
				Vars: map[string]string{
					"var1": "v1",
					"var2": "v2",
				},
			},
		},
	}
	dep, err := cfg.BuildDependency("dut", newSystemInfo("testOS", "testArch"))
	require.NoError(t, err)
	require.Equal(t, "https://testOS-overrideV1-overrideV2", *dep.URL)
	require.Equal(t, "https://{{.os}}-{{.var1}}-{{.var2}}", *cfg.Dependencies["dut"].Overrides[0].Dependency.URL)
}

func TestConfig_addChecksum(t *testing.T) {
	ts1 := serveFile(t, filepath.Join("testdata", "downloadables", "foo.tar.gz"), "/testOS2-v1-v2", "")
	ts2 := serveFile(t, filepath.Join("testdata", "downloadables", "foo.tar.gz"), "/testOS-overrideV1-overrideV2", "")
	dlURL := ts1.URL + "/{{.os}}-{{.var1}}-{{.var2}}"
	dlURL2 := ts2.URL + "/{{.os}}-{{.var1}}-{{.var2}}"
	overrideCheckedURL := ts2.URL + "/testOS-overrideV1-overrideV2"
	checkedURL := ts1.URL + "/testOS2-v1-v2"
	cfg := &Config{
		Dependencies: map[string]*Dependency{
			"dut": {
				URL: stringPtr(dlURL),
				Overrides: []DependencyOverride{
					{
						OverrideMatcher: OverrideMatcher{
							"arch": []string{"testArch"},
							"os":   []string{"testOS"},
						},
						Dependency: Dependency{
							URL: stringPtr(dlURL2),
							Vars: map[string]string{
								"var1": "overrideV1",
								"var2": "overrideV2",
							},
						},
					},
				},
				Vars: map[string]string{
					"var1": "v1",
					"var2": "v2",
				},
			},
		},
	}
	want := &Config{
		Dependencies: map[string]*Dependency{
			"dut": {
				URL: stringPtr(dlURL),
				Overrides: []DependencyOverride{
					{
						OverrideMatcher: OverrideMatcher{
							"arch": []string{"testArch"},
							"os":   []string{"testOS"},
						},
						Dependency: Dependency{
							URL: stringPtr(dlURL2),
							Vars: map[string]string{
								"var1": "overrideV1",
								"var2": "overrideV2",
							},
						},
					},
				},
				Vars: map[string]string{
					"var1": "v1",
					"var2": "v2",
				},
			},
		},
		URLChecksums: map[string]string{
			checkedURL:         fooChecksum,
			overrideCheckedURL: fooChecksum,
		},
	}
	err := cfg.addChecksum("dut", newSystemInfo("testOS", "testArch"))
	require.NoError(t, err)
	err = cfg.addChecksum("dut", newSystemInfo("testOS2", "foo"))
	require.NoError(t, err)

	b, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	fmt.Println(string(b))
	require.Equal(t, want, cfg)
}
