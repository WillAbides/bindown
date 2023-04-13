package bindown

import (
	"encoding/json"
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
		cfg := mustConfigFromYAML(t, `---
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
		dep := want.clone()
		dep.applyOverrides("windows/amd64", 0)
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
		dep.applyOverrides("linux/amd64", 0)
		assertDependencyEqual(t, &want, dep)
	})
}

func assertDependencyEqual(t testing.TB, want, got *Dependency) {
	t.Helper()
	wantJson, err := json.Marshal(want)
	require.NoError(t, err)
	gotJson, err := json.Marshal(got)
	require.NoError(t, err)
	require.JSONEq(t, string(wantJson), string(gotJson))
}
