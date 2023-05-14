package bindown

import (
	"encoding/json"
	"fmt"
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
		dep := mustConfigFromYAML(t, `dependencies: {dep1: {url: "foo"}}`).Dependencies["dep1"]
		want := mustConfigFromYAML(t, `dependencies: {dep1: {url: "foo"}}`).Dependencies["dep1"]
		err := dep.applyTemplate(nil, 0)
		require.NoError(t, err)
		requireEqualDependency(t, want, dep)
	})

	t.Run("missing grandparent template", func(t *testing.T) {
		cfg := mustConfigFromYAML(t, `
dependencies:
  dep1:
    template: foo
templates:
  foo:
    template: bar
`)
		err := cfg.Dependencies["dep1"].applyTemplate(cfg.Templates, 0)
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

	t.Run("maxTemplateDepth", func(t *testing.T) {
		templates := map[string]*Dependency{}
		for i := 0; i < maxTemplateDepth; i++ {
			name := fmt.Sprintf("template%d", i)
			nextName := fmt.Sprintf("template%d", i+1)
			templates[name] = &Dependency{Template: ptr(nextName)}
		}
		templates[fmt.Sprintf("template%d", maxTemplateDepth)] = &Dependency{
			Overrideable: Overrideable{URL: ptr("foo")},
		}
		dep := &Dependency{
			Template: ptr("template0"),
		}
		err := dep.applyTemplate(templates, 0)
		require.EqualError(t, err, fmt.Sprintf("max template depth of %d exceeded", maxTemplateDepth))
	})
}

func Test_Dependency_applyOverrides(t *testing.T) {
	t.Run("nil overrides", func(t *testing.T) {
		want := mustConfigFromYAML(t, `
dependencies:
  dep1:
    archive_path: archivePath
    vars: {foo: bar}
`).Dependencies["dep1"]
		dep := want.clone()
		err := dep.applyOverrides("windows/amd64", 0)
		require.NoError(t, err)
		requireEqualDependency(t, want, dep)
	})

	t.Run("simple override", func(t *testing.T) {
		cfg := mustConfigFromYAML(t, `
dependencies:
  dep1:
    archive_path: archivePath
    vars:
      foo: bar
      baz: qux
      version: 1.2.3
    overrides:
      - matcher:
          os: [linux]
          foo: [bar]
          version: [asdf, 1.2.4, 1.x]
        dependency:
          link: true
          vars:
            foo: not bar
            bar: moo
          overrides:
            - matcher:
                arch: [amd64]
              dependency:
                archive_path: it's amd64
                overrides:
                  - matcher:
                      arch: [ amd64 ]
                    dependency:
                      archive_path: still amd64
`)

		dep := cfg.Dependencies["dep1"]
		want := mustConfigFromYAML(t, `
dependencies:
  dep1:
    archive_path: still amd64
    link: true
    vars:
      foo: not bar
      baz: qux
      bar: moo
      version: 1.2.3
`).Dependencies["dep1"]
		err := dep.applyOverrides("linux/amd64", 0)
		require.NoError(t, err)
		assertDependencyEqual(t, want, dep)
	})

	t.Run("maxOverrideDepth", func(t *testing.T) {
		dep := &Dependency{}
		latest := &dep.Overrideable
		for i := 0; i < maxOverrideDepth+1; i++ {
			latest.Overrides = []DependencyOverride{{OverrideMatcher: map[string][]string{"os": {"darwin"}}}}
			latest = &latest.Overrides[0].Dependency
		}
		err := dep.applyOverrides("darwin/amd64", 0)
		require.EqualError(t, err, fmt.Sprintf("max override depth of %d exceeded", maxOverrideDepth))
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
