package bindown

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v3/internal/testutil"
	"github.com/willabides/bindown/v3/internal/util"
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

func Test_extract(t *testing.T) {
	dir := testutil.TmpDir(t)
	downloadDir := filepath.Join(dir, "download")
	extractDir := filepath.Join(dir, "extract")
	require.NoError(t, os.MkdirAll(downloadDir, 0o750))
	archivePath := filepath.Join(downloadDir, "foo.tar.gz")
	err := util.CopyFile(testutil.DownloadablesPath("foo.tar.gz"), archivePath, nil)
	require.NoError(t, err)
	err = extract(archivePath, extractDir)
	require.NoError(t, err)
}

func Test_copyBin(t *testing.T) {
	dir := testutil.TmpDir(t)
	extractDir := filepath.Join(dir, ".bindown", "extracts", "deadbeef")
	binName := "bleep"
	require.NoError(t, os.MkdirAll(extractDir, 0o750))
	err := util.CopyFile(testutil.DownloadablesPath("rawfile/foo"), filepath.Join(extractDir, binName), nil)
	require.NoError(t, err)
	target := filepath.Join(dir, "bin", "foo")
	err = copyBin(target, extractDir, "", binName)
	require.NoError(t, err)
	testutil.AssertEqualFiles(t, filepath.Join(extractDir, binName), target)
}

func Test_linkBin(t *testing.T) {
	dir := testutil.TmpDir(t)
	extractDir := filepath.Join(dir, ".bindown", "extracts", "deadbeef")
	binName := "bleep"
	require.NoError(t, os.MkdirAll(extractDir, 0o750))
	err := util.CopyFile(testutil.DownloadablesPath("rawfile/foo"), filepath.Join(extractDir, binName), nil)
	require.NoError(t, err)
	target := filepath.Join(dir, "bin", "foo")
	err = linkBin(target, extractDir, "", binName)
	require.NoError(t, err)
	testutil.AssertEqualFiles(t, filepath.Join(extractDir, binName), target)
}

func Test_downloadFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
		err := downloadFile(filepath.Join(dir, "bar.tar.gz"), ts.URL+"/foo/foo.tar.gz")
		assert.NoError(t, err)
		testutil.AssertEqualFiles(t, testutil.DownloadablesPath("foo.tar.gz"), filepath.Join(dir, "bar.tar.gz"))
	})

	t.Run("404", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
		err := downloadFile(filepath.Join(dir, "bar.tar.gz"), ts.URL+"/wrongpath")
		assert.Error(t, err)
	})
}

func TestGetURLChecksum(t *testing.T) {
	ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
	got, err := getURLChecksum(ts.URL + "/foo/foo.tar.gz")
	require.NoError(t, err)
	require.Equal(t, testutil.FooChecksum, got)
}

func TestDependency_applyTemplate(t *testing.T) {
	t.Run("no template", func(t *testing.T) {
		dep := &Dependency{
			URL: stringPtr("foo"),
		}
		want := &Dependency{
			URL: stringPtr("foo"),
		}
		err := dep.applyTemplate(nil, 0)
		require.NoError(t, err)
		requireEqualDependency(t, want, dep)
	})

	t.Run("missing grandparent template", func(t *testing.T) {
		dep := &Dependency{
			Template: stringPtr("foo"),
		}
		templates := map[string]*Dependency{
			"foo": {
				Template: stringPtr("bar"),
			},
		}
		err := dep.applyTemplate(templates, 0)
		require.Error(t, err)
	})

	t.Run("missing template", func(t *testing.T) {
		dep := &Dependency{
			Template: stringPtr("bar"),
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
		requireEqualDependency(t, cfg.Dependencies["want"], dep)
	})
}

func Test_Dependency_applyOverrides(t *testing.T) {
	t.Run("nil overrides", func(t *testing.T) {
		want := Dependency{
			ArchivePath: stringPtr("archivePath"),
			Link:        nil,
			Vars: map[string]string{
				"foo": "bar",
			},
		}
		dep := want.clone()
		dep.applyOverrides(newSystemInfo("windows", "amd64"), 0)
		requireEqualDependency(t, &want, dep)
	})

	t.Run("simple override", func(t *testing.T) {
		dep := &Dependency{
			ArchivePath: stringPtr("archivePath"),
			Vars: map[string]string{
				"foo":     "bar",
				"baz":     "qux",
				"version": "1.2.3",
			},
			Overrides: []DependencyOverride{
				{
					OverrideMatcher: OverrideMatcher{
						OS: []string{"linux"},
						Vars: map[string][]string{
							"foo":     {"bar"},
							"version": {"asdf", "1.2.4", "1.x"},
						},
					},
					Dependency: Dependency{
						Link: boolPtr(true),
						Vars: map[string]string{
							"foo": "not bar",
							"bar": "moo",
						},
						Overrides: []DependencyOverride{
							{
								OverrideMatcher: OverrideMatcher{
									Arch: []string{"amd64"},
								},
								Dependency: Dependency{
									ArchivePath: stringPtr("it's amd64"),
								},
							},
							{
								OverrideMatcher: OverrideMatcher{
									Arch: []string{"x86"},
								},
								Dependency: Dependency{
									ArchivePath: stringPtr("it's x86"),
								},
							},
						},
					},
				},
			},
		}
		want := Dependency{
			ArchivePath: stringPtr("it's amd64"),
			Link:        boolPtr(true),
			Vars: map[string]string{
				"foo":     "not bar",
				"baz":     "qux",
				"bar":     "moo",
				"version": "1.2.3",
			},
		}
		dep.applyOverrides(newSystemInfo("linux", "amd64"), 0)
		requireEqualDependency(t, &want, dep)
	})
}
