package bindown

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func requireEqualDownloadable(t *testing.T, want, got Downloadable) {
	t.Helper()
	require.Equal(t, want.URL, got.URL)
	require.Equal(t, want.ArchivePath, got.ArchivePath)
	require.Equal(t, want.BinName, got.BinName)
	require.Equal(t, want.Link, got.Link)
	require.Equal(t, want.Vars, got.Vars)
	require.Equal(t, want.Overrides, got.Overrides)
	require.Equal(t, want.KnownBuilds, got.KnownBuilds)
}

func TestDownloadable_applyTemplate(t *testing.T) {
	t.Run("no template", func(t *testing.T) {
		downloadable := &Downloadable{
			URL: stringPtr("foo"),
		}
		want := &Downloadable{
			URL: stringPtr("foo"),
		}
		err := downloadable.applyTemplate(nil, 0)
		require.NoError(t, err)
		requireEqualDownloadable(t, *want, *downloadable)
	})

	t.Run("missing grandparent template", func(t *testing.T) {
		downloadable := &Downloadable{
			Template: stringPtr("foo"),
		}
		templates := map[string]*Downloadable{
			"foo": {
				Template: stringPtr("bar"),
			},
		}
		err := downloadable.applyTemplate(templates, 0)
		require.Error(t, err)
	})

	t.Run("missing template", func(t *testing.T) {
		downloadable := &Downloadable{
			Template: stringPtr("bar"),
		}
		err := downloadable.applyTemplate(nil, 0)
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
    known_builds: [tmplOS/tmplArch]
    overrides:
      - os: [darwin]
        arch: [amd64]
        url: templateOverrideURL
downloadables:
  myDownloadable:
    template: template1
    link: false
    archive_path: downloadableArchivePath
    vars:
      foo: "downloadable foo"
      baz: "downloadable baz"
    known_builds: [downloadableOS/downloadableArch]
    overrides:
      - os: [darwin]
        arch: [amd64]
        url: downloadableOverrideURL
  want:
    link: false
    archive_path: downloadableArchivePath
    url: parentTemplateURL
    vars:
      foo: "downloadable foo"
      baz: "downloadable baz"
      bar: "template bar"
    known_builds:
      - tmplOS/tmplArch
      - downloadableOS/downloadableArch
    overrides:
      - os: [darwin]
        arch: [amd64]
        url: templateOverrideURL      
      - os: [darwin]
        arch: [amd64]
        url: downloadableOverrideURL
`)
		downloadable := cfg.Downloadables["myDownloadable"]
		err := downloadable.applyTemplate(cfg.Templates, 0)
		require.NoError(t, err)
		requireEqualDownloadable(t, *cfg.Downloadables["want"], *downloadable)
	})
}

func Test_Downloadable_applyOverrides(t *testing.T) {
	t.Run("nil overrides", func(t *testing.T) {
		want := Downloadable{
			ArchivePath: stringPtr("archivePath"),
			Link:        nil,
			Vars: map[string]string{
				"foo": "bar",
			},
		}
		downloadable := want.clone()
		downloadable.applyOverrides(newSystemInfo("windows", "amd64"), 0)
		requireEqualDownloadable(t, want, *downloadable)
	})

	t.Run("simple override", func(t *testing.T) {
		downloadable := &Downloadable{
			ArchivePath: stringPtr("archivePath"),
			Vars: map[string]string{
				"foo": "bar",
				"baz": "qux",
			},
			Overrides: []DownloadableOverride{
				{
					DownloadableMatcher: DownloadableMatcher{
						OS: []string{"linux"},
					},
					Downloadable: Downloadable{
						Link: boolPtr(true),
						Vars: map[string]string{
							"foo": "not bar",
							"bar": "moo",
						},
						Overrides: []DownloadableOverride{
							{
								DownloadableMatcher: DownloadableMatcher{
									Arch: []string{"amd64"},
								},
								Downloadable: Downloadable{
									ArchivePath: stringPtr("it's amd64"),
								},
							},
							{
								DownloadableMatcher: DownloadableMatcher{
									Arch: []string{"x86"},
								},
								Downloadable: Downloadable{
									ArchivePath: stringPtr("it's x86"),
								},
							},
						},
					},
				},
			},
		}
		want := Downloadable{
			ArchivePath: stringPtr("it's amd64"),
			Link:        boolPtr(true),
			Vars: map[string]string{
				"foo": "not bar",
				"baz": "qux",
				"bar": "moo",
			},
		}
		downloadable.applyOverrides(newSystemInfo("linux", "amd64"), 0)
		requireEqualDownloadable(t, want, *downloadable)
	})
}
