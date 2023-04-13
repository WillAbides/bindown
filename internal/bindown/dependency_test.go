package bindown

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func requireEqualDependency(t *testing.T, want, got *Dependency) {
	t.Helper()
	require.Equal(t, want.URL, got.URL)
	require.Equal(t, want.ArchivePath, got.ArchivePath)
	require.Equal(t, want.BinName, got.BinName)
	require.Equal(t, want.Link, got.Link)
	require.Equal(t, want.Vars, got.Vars)
	require.Equal(t, want.Overrides, got.Overrides)
}

func TestGetURLChecksum(t *testing.T) {
	ts := serveFile(t, filepath.Join("testdata", "downloadables", "foo.tar.gz"), "/foo/foo.tar.gz", "")
	got, err := getURLChecksum(ts.URL+"/foo/foo.tar.gz", "")
	require.NoError(t, err)
	require.Equal(t, fooChecksum, got)
}

func TestDependency_applyTemplate(t *testing.T) {
	t.Run("no template", func(t *testing.T) {
		dep := &Dependency{
			URL: ptr("foo"),
		}
		want := &Dependency{
			URL: ptr("foo"),
		}
		err := dep.applyTemplate(nil, 0)
		require.NoError(t, err)
		requireEqualDependency(t, want, dep)
	})

	t.Run("missing grandparent template", func(t *testing.T) {
		dep := &Dependency{
			Template: ptr("foo"),
		}
		templates := map[string]*Dependency{
			"foo": {
				Template: ptr("bar"),
			},
		}
		err := dep.applyTemplate(templates, 0)
		require.Error(t, err)
	})

	t.Run("missing template", func(t *testing.T) {
		dep := &Dependency{
			Template: ptr("bar"),
		}
		err := dep.applyTemplate(nil, 0)
		require.Error(t, err)
	})

	t.Run("basic", func(t *testing.T) {
		cfg := configFromYaml(t, `---
templates:
  parentTemplate:
    url: parentTemplateURL
  template1:
    template: parentTemplate
    link: true
    archive_path: templateArchivePath
    vars:
      foo: "template foo"
      bar: "template bar"
    overrides:
      - matcher:
          os: [darwin]
          arch: [amd64]
        dependency:
          url: templateOverrideURL
dependencies:
  myDependency:
    template: template1
    link: false
    archive_path: dependencyArchivePath
    vars:
      foo: "dependency foo"
      baz: "dependency baz"
    overrides:
      - matcher:
          os: [darwin]
          arch: [amd64]
        dependency:
          url: dependencyOverrideURL
  want:
    template: template1
    link: false
    archive_path: dependencyArchivePath
    url: parentTemplateURL
    vars:
      foo: "dependency foo"
      baz: "dependency baz"
      bar: "template bar"
    overrides:
      - matcher:
          os: [darwin]
          arch: [amd64]
        dependency:
          url: templateOverrideURL
      - matcher:
          os: [darwin]
          arch: [amd64]
        dependency:
          url: dependencyOverrideURL
`)
		dep := cfg.Dependencies["myDependency"]
		err := dep.applyTemplate(cfg.Templates, 0)
		require.NoError(t, err)
		assertDependencyEqual(t, cfg.Dependencies["want"], dep)
	})
}

func Test_Dependency_applyOverrides(t *testing.T) {
	t.Run("nil overrides", func(t *testing.T) {
		want := Dependency{
			ArchivePath: ptr("archivePath"),
			Link:        nil,
			Vars: map[string]string{
				"foo": "bar",
			},
		}
		dep := want.Clone()
		dep.applyOverrides(newSystemInfo("windows", "amd64"), 0)
		requireEqualDependency(t, &want, dep)
	})

	t.Run("simple override", func(t *testing.T) {
		dep := &Dependency{
			ArchivePath: ptr("archivePath"),
			Vars: map[string]string{
				"foo":     "bar",
				"baz":     "qux",
				"version": "1.2.3",
			},
			Overrides: []DependencyOverride{
				{
					OverrideMatcher: OverrideMatcher{
						"os":      {"linux"},
						"foo":     {"bar"},
						"version": {"asdf", "1.2.4", "1.x"},
					},
					Dependency: Dependency{
						Link: ptr(true),
						Vars: map[string]string{
							"foo": "not bar",
							"bar": "moo",
						},
						Overrides: []DependencyOverride{
							{
								OverrideMatcher: OverrideMatcher{
									"arch": []string{"amd64"},
								},
								Dependency: Dependency{
									ArchivePath: ptr("it's amd64"),
								},
							},
							{
								OverrideMatcher: OverrideMatcher{
									"arch": []string{"x86"},
								},
								Dependency: Dependency{
									ArchivePath: ptr("it's x86"),
								},
							},
						},
					},
				},
			},
		}
		want := Dependency{
			ArchivePath: ptr("it's amd64"),
			Link:        ptr(true),
			Vars: map[string]string{
				"foo":     "not bar",
				"baz":     "qux",
				"bar":     "moo",
				"version": "1.2.3",
			},
		}
		dep.applyOverrides(newSystemInfo("linux", "amd64"), 0)
		assertDependencyEqual(t, &want, dep)
	})
}

func TestOverrideMatcher_matches(t *testing.T) {
	for _, td := range []struct {
		name    string
		matcher OverrideMatcher
		info    SystemInfo
		vars    map[string]string
		want    bool
	}{
		{
			name: "empty matcher always matches",
			want: true,
		},
		{
			name: "os match",
			matcher: OverrideMatcher{
				"os": {"windows", "darwin"},
			},
			info: newSystemInfo("darwin", "amd64"),
			want: true,
		},
		{
			name: "os mismatch",
			matcher: OverrideMatcher{
				"os": {"windows", "darwin"},
			},
			info: newSystemInfo("linux", "amd64"),
			want: false,
		},
		{
			name: "arch match",
			matcher: OverrideMatcher{
				"arch": {"amd64", "arm64"},
			},
			info: newSystemInfo("linux", "amd64"),
			want: true,
		},
		{
			name: "arch mismatch",
			matcher: OverrideMatcher{
				"arch": {"amd64", "arm64"},
			},
			info: newSystemInfo("linux", "386"),
			want: false,
		},
		{
			name: "var match",
			matcher: OverrideMatcher{
				"foo": {"bar", "baz"},
			},
			vars: map[string]string{
				"foo": "bar",
			},
			want: true,
		},
		{
			name: "var mismatch",
			matcher: OverrideMatcher{
				"foo": {"bar", "baz"},
			},
			vars: map[string]string{
				"foo": "qux",
			},
		},
		{
			name: "var not set",
			matcher: OverrideMatcher{
				"foo": {"bar", "baz"},
			},
			want: false,
		},
		{
			name: "var match with semver glob",
			matcher: OverrideMatcher{
				"foo": {"bar", "baz", "1.*"},
			},
			vars: map[string]string{
				"foo": "1.2.3",
			},
			want: true,
		},
		{
			name: "semver glob with non-semver version",
			matcher: OverrideMatcher{
				"foo": {"1.*", "bar", "baz"},
			},
			vars: map[string]string{
				"foo": "bar",
			},
			want: true,
		},
		{
			name: "empty os var overrides system info",
			matcher: OverrideMatcher{
				"os": {"windows", "darwin"},
			},
			vars: map[string]string{
				"os": "",
			},
			info: newSystemInfo("linux", "amd64"),
		},
	} {
		t.Run(td.name, func(t *testing.T) {
			require.Equal(t, td.want, td.matcher.matches(td.info, td.vars))
		})
	}
}

func assertDependencyEqual(t testing.TB, want, got *Dependency) {
	t.Helper()
	wantJson, err := json.Marshal(want)
	require.NoError(t, err)
	gotJson, err := json.Marshal(got)
	require.NoError(t, err)
	require.JSONEq(t, string(wantJson), string(gotJson))
}
