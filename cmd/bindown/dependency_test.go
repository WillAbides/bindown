package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v3"
)

func Test_dependencyUpdateVarCmd(t *testing.T) {
	for _, td := range []struct {
		name      string
		config    bindown.Config
		args      []string
		wantVars  map[string]string
		wantState resultState
	}{
		{
			name: "no changes",
			config: bindown.Config{
				Dependencies: map[string]*bindown.Dependency{
					"dep1": {URL: ptr("foo")},
				},
				URLChecksums: map[string]string{"foo": "0000"},
			},
			args: []string{"dependency", "update-vars", "dep1"},
		},
		{
			name: "create var",
			config: bindown.Config{
				Dependencies: map[string]*bindown.Dependency{
					"dep1": {
						URL:          ptr("foo"),
						RequiredVars: []string{"foo"},
					},
				},
				URLChecksums: map[string]string{"foo": "0000"},
			},
			args:     []string{"dependency", "update-vars", "dep1", "--set", "foo=bar"},
			wantVars: map[string]string{"foo": "bar"},
		},
		{
			name: "update var",
			config: bindown.Config{
				Dependencies: map[string]*bindown.Dependency{
					"dep1": {
						URL: ptr("foo"),
						Vars: map[string]string{
							"foo": "bar",
						},
					},
				},
				URLChecksums: map[string]string{"foo": "0000"},
			},
			args:     []string{"dependency", "update-vars", "dep1", "--set", "foo=baz"},
			wantVars: map[string]string{"foo": "baz"},
		},
		{
			name: "unset var",
			config: bindown.Config{
				Dependencies: map[string]*bindown.Dependency{
					"dep1": {
						URL: ptr("foo"),
						Vars: map[string]string{
							"foo": "bar",
							"baz": "qux",
						},
					},
				},
				URLChecksums: map[string]string{"foo": "0000"},
			},
			args:     []string{"dependency", "update-vars", "dep1", "--unset", "foo"},
			wantVars: map[string]string{"baz": "qux"},
		},
		{
			name: "unset all vars",
			config: bindown.Config{
				Dependencies: map[string]*bindown.Dependency{
					"dep1": {
						URL: ptr("foo"),
						Vars: map[string]string{
							"foo": "bar",
							"baz": "qux",
						},
					},
				},
				URLChecksums: map[string]string{"foo": "0000"},
			},
			args: []string{"dependency", "update-vars", "dep1", "--unset", "foo,baz"},
		},
		{
			name: "unset on empty vars",
			config: bindown.Config{
				Dependencies: map[string]*bindown.Dependency{
					"dep1": {
						URL: ptr("foo"),
					},
				},
				URLChecksums: map[string]string{"foo": "0000"},
			},
			args: []string{"dependency", "update-vars", "dep1", "--unset", "foo"},
		},
		{
			name: "no-op on non-existent dependency",
			config: bindown.Config{
				Dependencies: map[string]*bindown.Dependency{
					"dep1": {URL: ptr("foo")},
				},
				URLChecksums: map[string]string{"foo": "0000"},
			},
			args: []string{"dependency", "update-vars", "fake"},
			wantState: resultState{
				stderr: `cmd: error: no dependency configured with the name "fake"`,
				exit:   1,
			},
		},
		{
			name: "set var on non-existent dependency",
			config: bindown.Config{
				Dependencies: map[string]*bindown.Dependency{
					"dep1": {URL: ptr("foo")},
				},
				URLChecksums: map[string]string{"foo": "0000"},
			},
			args: []string{"dependency", "update-vars", "fake", "--set", "foo=bar"},
			wantState: resultState{
				stderr: `cmd: error: dependency "fake" does not exist`,
				exit:   1,
			},
		},
		{
			name: "set var on non-existent dependency",
			config: bindown.Config{
				Dependencies: map[string]*bindown.Dependency{
					"dep1": {URL: ptr("foo")},
				},
				URLChecksums: map[string]string{"foo": "0000"},
			},
			args: []string{"dependency", "update-vars", "fake", "--unset", "foo"},
			wantState: resultState{
				stderr: `cmd: error: dependency "fake" does not exist`,
				exit:   1,
			},
		},
		{
			name: "error adding checksums",
			config: bindown.Config{
				Dependencies: map[string]*bindown.Dependency{
					"dep1": {URL: ptr("https://")},
				},
			},
			args: []string{"dependency", "update-vars", "dep1", "--set", "foo=bar"},
			wantState: resultState{
				stderr: `cmd: error: Get "https:": http: no Host in request URL`,
				exit:   1,
			},
		},
		{
			name: "--skipchecksums",
			config: bindown.Config{
				Dependencies: map[string]*bindown.Dependency{
					"dep1": {URL: ptr("foo")},
				},
			},
			args: []string{"dependency", "update-vars", "dep1", "--set", "foo=bar", "--skipchecksums"},
			wantVars: map[string]string{
				"foo": "bar",
			},
		},
		{
			name: "missing required vars",
			config: bindown.Config{
				Dependencies: map[string]*bindown.Dependency{
					"dep1": {
						URL: ptr("foo"),
						Vars: map[string]string{
							"foo": "bar",
						},
						RequiredVars: []string{"qux"},
					},
				},
			},
			args:     []string{"dependency", "update-vars", "dep1", "--set", "foo=baz"},
			wantVars: map[string]string{"foo": "baz"},
		},
	} {
		t.Run(td.name, func(t *testing.T) {
			runner := newCmdRunner(t)
			runner.writeConfig(&td.config)
			result := runner.run(td.args...)
			result.assertState(td.wantState)
			if td.wantVars != nil {
				require.Equal(t, td.wantVars, runner.getConfigFile().Dependencies["dep1"].Vars)
			} else {
				require.Empty(t, runner.getConfigFile().Dependencies["dep1"].Vars)
			}
		})
	}
}

func Test_dependencyShowConfigCmd(t *testing.T) {
	baseCfg := bindown.Config{
		Dependencies: map[string]*bindown.Dependency{
			"dep1": {
				URL: ptr("foo"),
				Vars: map[string]string{
					"foo": "bar",
					"baz": "qux",
				},
			},
		},
	}
	for _, td := range []struct {
		name      string
		config    bindown.Config
		args      []string
		wantState resultState
	}{
		{
			name:   "json output",
			config: baseCfg,
			args:   []string{"dependency", "show-config", "dep1", "--json"},
			wantState: resultState{
				stdout: `
{
  "url": "foo",
  "vars": {
    "baz": "qux",
    "foo": "bar"
  }
}
`,
			},
		},
		{
			name:   "yaml output",
			config: baseCfg,
			args:   []string{"dependency", "show-config", "dep1"},
			wantState: resultState{
				stdout: `
url: foo
vars:
  baz: qux
  foo: bar
`,
			},
		},
		{
			name:   "non-existent dependency",
			config: baseCfg,
			args:   []string{"dependency", "show-config", "fake"},
			wantState: resultState{
				stderr: `cmd: error: no dependency named "fake"`,
				exit:   1,
			},
		},
	} {
		t.Run(td.name, func(t *testing.T) {
			runner := newCmdRunner(t)
			runner.writeConfig(&td.config)
			result := runner.run(td.args...)
			result.assertState(td.wantState)
		})
	}
}

func Test_dependencyInfoCmd(t *testing.T) {
	baseCfg := bindown.Config{
		Systems: []bindown.SystemInfo{
			{OS: "darwin", Arch: "amd64"},
			{OS: "linux", Arch: "386"},
			{OS: "linux", Arch: "arm64"},
		},
		Dependencies: map[string]*bindown.Dependency{
			"dep1": {
				Link: ptr(true),
				URL:  ptr("foo-{{ .foo }}-{{ .version }}-{{ .os }}-{{ .arch }}"),
				Vars: map[string]string{
					"foo":     "bar",
					"baz":     "qux",
					"version": "1.2.3",
				},
				ArchivePath: ptr("{{ .version }}-{{ .os }}-{{ .arch }}/bin/foo"),
				Systems: []bindown.SystemInfo{
					{OS: "darwin", Arch: "amd64"},
					{OS: "linux", Arch: "386"},
					{OS: "windows", Arch: "386"},
				},
				Overrides: []bindown.DependencyOverride{
					{
						OverrideMatcher: bindown.OverrideMatcher{
							"os": []string{"darwin"},
						},
						Dependency: bindown.Dependency{
							Link: ptr(false),
						},
					},
				},
			},
		},
	}
	for _, td := range []struct {
		name      string
		config    bindown.Config
		args      []string
		wantState resultState
	}{
		{
			name:   "json output",
			config: baseCfg,
			args:   []string{"dependency", "info", "dep1", "--json"},
			wantState: resultState{
				stdout: `
{
  "darwin/amd64": {
    "url": "foo-bar-1.2.3-darwin-amd64",
    "archive_path": "1.2.3-darwin-amd64/bin/foo",
    "bin": "dep1",
    "link": false
  },
  "linux/386": {
    "url": "foo-bar-1.2.3-linux-386",
    "archive_path": "1.2.3-linux-386/bin/foo",
    "bin": "dep1",
    "link": true
  }
}
`,
			},
		},
		{
			name:   "yaml output",
			config: baseCfg,
			args:   []string{"dependency", "info", "dep1"},
			wantState: resultState{
				stdout: `
darwin/amd64:
  url: foo-bar-1.2.3-darwin-amd64
  archive_path: 1.2.3-darwin-amd64/bin/foo
  bin: dep1
  link: false
linux/386:
  url: foo-bar-1.2.3-linux-386
  archive_path: 1.2.3-linux-386/bin/foo
  bin: dep1
  link: true
`,
			},
		},
	} {
		t.Run(td.name, func(t *testing.T) {
			runner := newCmdRunner(t)
			runner.writeConfig(&td.config)
			result := runner.run(td.args...)
			result.assertState(td.wantState)
		})
	}
}

func Test_dependencyListCmd(t *testing.T) {
	baseCfg := bindown.Config{
		Dependencies: map[string]*bindown.Dependency{
			"dep1": {
				URL: ptr("foo"),
			},
			"dep2": {
				URL: ptr("bar"),
			},
		},
	}
	for _, td := range []struct {
		name      string
		config    bindown.Config
		args      []string
		wantState resultState
	}{
		{
			config: baseCfg,
			args:   []string{"dependency", "list", "--json"},
			wantState: resultState{
				stdout: `
dep1
dep2
`,
			},
		},
	} {
		t.Run(td.name, func(t *testing.T) {
			runner := newCmdRunner(t)
			runner.writeConfig(&td.config)
			result := runner.run(td.args...)
			result.assertState(td.wantState)
		})
	}
}

func Test_dependencyRemoveCmd(t *testing.T) {
	baseCfg := bindown.Config{
		Dependencies: map[string]*bindown.Dependency{
			"dep1": {
				URL: ptr("foo"),
			},
			"dep2": {
				URL: ptr("bar"),
			},
		},
	}
	for _, td := range []struct {
		name      string
		config    bindown.Config
		args      []string
		wantState resultState
		wantDeps  []string
	}{
		{
			name:     "remove one",
			config:   baseCfg,
			args:     []string{"dependency", "remove", "dep1"},
			wantDeps: []string{"dep2"},
		},
		{
			name:   "non-existent",
			config: baseCfg,
			args:   []string{"dependency", "remove", "dep3"},
			wantState: resultState{
				stderr: `cmd: error: no dependency named "dep3"`,
				exit:   1,
			},
			wantDeps: []string{"dep1", "dep2"},
		},
	} {
		t.Run(td.name, func(t *testing.T) {
			runner := newCmdRunner(t)
			runner.writeConfig(&td.config)
			result := runner.run(td.args...)
			result.assertState(td.wantState)
			cfg := runner.getConfigFile()
			if len(td.wantDeps) == 0 {
				if len(cfg.Dependencies) != 0 {
					t.Fatalf("expected no dependencies, got %d", len(cfg.Dependencies))
				}
			} else {
				if len(cfg.Dependencies) != len(td.wantDeps) {
					t.Fatalf("expected %d dependencies, got %d", len(td.wantDeps), len(cfg.Dependencies))
				}
				for _, dep := range td.wantDeps {
					if _, ok := cfg.Dependencies[dep]; !ok {
						t.Fatalf("expected dependency %q to exist", dep)
					}
				}
			}
		})
	}
}

func Test_dependencyAddCmd(t *testing.T) {
	t.Run("from existing template", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`
systems: ["linux/amd64", "darwin/amd64"]
templates:
  tmpl:
    url: foo-{{ .os }}-{{ .arch }}-{{ .version }}
url_checksums:
  foo-linux-amd64-1.2.3: deadbeef
  foo-darwin-amd64-1.2.3: deadbeef
`)
		result := runner.run("dependency", "add", "dep1", "tmpl", "--var=version=1.2.3")
		result.assertState(resultState{})
		cfg := runner.getConfigFile()
		wantDep := &bindown.Dependency{
			Template: ptr("tmpl"),
			Vars: map[string]string{
				"version": "1.2.3",
			},
		}
		require.Equal(t, wantDep, cfg.Dependencies["dep1"])
	})

	t.Run("from missing template", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`{}`)
		result := runner.run("dependency", "add", "dep1", "tmpl")
		result.assertState(resultState{
			exit:   1,
			stderr: `cmd: error: no template named "tmpl"`,
		})
	})

	t.Run("from missing source", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`{}`)
		result := runner.run("dependency", "add", "dep1", "tmpl", "--source=foo")
		result.assertState(resultState{
			exit:   1,
			stderr: `cmd: error: open foo: no such file or directory`,
		})
	})

	t.Run("from file source", func(t *testing.T) {
		runner := newCmdRunner(t)
		srcFile := filepath.Join(runner.tmpDir, "template-source.yaml")
		runner.writeConfigYaml(fmt.Sprintf(`
url_checksums:
  foo-linux-amd64-1.2.3: deadbeef
  foo-darwin-amd64-1.2.3: deadbeef
template_sources:
  origin: %q
`, srcFile))

		err := os.WriteFile(srcFile, []byte(`
systems: ["linux/amd64", "darwin/amd64"]
templates:
  tmpl:
    url: foo-{{ .os }}-{{ .arch }}-{{ .version }}
`), 0o600)
		require.NoError(t, err)
		result := runner.run("dependency", "add", "dep1", "tmpl", "--var=version=1.2.3", "--source=origin")
		result.assertState(resultState{})
		wantDep := &bindown.Dependency{
			Template: ptr("origin#tmpl"),
			Vars: map[string]string{
				"version": "1.2.3",
			},
		}
		cfg := runner.getConfigFile()
		require.Equal(t, wantDep, cfg.Dependencies["dep1"])
	})

	t.Run("using source-name syntax", func(t *testing.T) {
		runner := newCmdRunner(t)
		srcFile := filepath.Join(runner.tmpDir, "template-source.yaml")
		runner.writeConfigYaml(fmt.Sprintf(`
url_checksums:
  foo-linux-amd64-1.2.3: deadbeef
  foo-darwin-amd64-1.2.3: deadbeef
template_sources:
  origin: %q
`, srcFile))

		err := os.WriteFile(srcFile, []byte(`
systems: ["linux/amd64", "darwin/amd64"]
templates:
  tmpl:
    url: foo-{{ .os }}-{{ .arch }}-{{ .version }}
`), 0o600)
		require.NoError(t, err)
		result := runner.run("dependency", "add", "dep1", "origin#tmpl", "--var=version=1.2.3")
		result.assertState(resultState{})
		wantDep := &bindown.Dependency{
			Template: ptr("origin#tmpl"),
			Vars: map[string]string{
				"version": "1.2.3",
			},
		}
		cfg := runner.getConfigFile()
		require.Equal(t, wantDep, cfg.Dependencies["dep1"])
	})

	t.Run("prompts for required vars", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`
systems: ["linux/amd64", "darwin/amd64"]
templates:
  tmpl:
    url: foo-{{ .os }}-{{ .arch }}-{{ .version }}
    required_vars: ["version", "foo"]
url_checksums:
  foo-linux-amd64-1.2.3: deadbeef
  foo-darwin-amd64-1.2.3: deadbeef
`)
		runner.stdin = strings.NewReader("1.2.3\nbar")
		result := runner.run("dependency", "add", "dep1", "tmpl")
		result.assertState(resultState{
			stdout: `Please enter a value for required variable "version":	Please enter a value for required variable "foo":`,
		})
		cfg := runner.getConfigFile()
		wantDep := &bindown.Dependency{
			Template: ptr("tmpl"),
			Vars: map[string]string{
				"version": "1.2.3",
				"foo":     "bar",
			},
		}
		require.Equal(t, wantDep, cfg.Dependencies["dep1"])
	})

	t.Run("with http server", func(t *testing.T) {
		downloadablesDir := filepath.FromSlash("../../testdata/downloadables")
		tar := filepath.Join(downloadablesDir, "runnable.tar.gz")
		zip := filepath.Join(downloadablesDir, "runnable_windows.zip")

		server := serveFiles(t, map[string]string{
			"/foo/v1.2.3/foo-darwin-amd64.tar.gz": tar,
			"/foo/v1.2.3/foo-darwin-arm64.tar.gz": tar,
			"/foo/v1.2.3/foo-linux-amd64.tar.gz":  tar,
			"/foo/v1.2.3/foo-windows-amd64.zip":   zip,
		})

		srcPath := filepath.FromSlash("../../testdata/configs/dep-add-source.yaml")
		runner := newCmdRunner(t)
		runner.writeConfig(&bindown.Config{
			Systems: []bindown.SystemInfo{
				{OS: "darwin", Arch: "amd64"},
				{OS: "darwin", Arch: "arm64"},
				{OS: "linux", Arch: "amd64"},
				{OS: "windows", Arch: "amd64"},
			},
			TemplateSources: map[string]string{
				"origin": srcPath,
			},
		})
		wantStdout := `Please enter a value for required variable "version":	Please enter a value for required variable "addr":	`
		runner.stdin = strings.NewReader(fmt.Sprintf("%s\n%s", "1.2.3", server.URL))
		result := runner.run("dependency", "add", "foo", "tmpl1", "--source", "origin")
		require.Equal(t, 0, result.exitVal)
		require.Equal(t, wantStdout, result.stdOut.String())
		gotCfg := runner.getConfigFile()
		wantDep := &bindown.Dependency{
			Template: ptr("origin#tmpl1"),
			Vars: map[string]string{
				"version": "1.2.3",
				"addr":    server.URL,
			},
		}
		wantChecksums := map[string]string{
			fmt.Sprintf("%s/foo/v1.2.3/foo-darwin-amd64.tar.gz", server.URL): "fb2fe41a34b77ee180def0cb9a222d8776a6e581106009b64f35983da291ab6e",
			fmt.Sprintf("%s/foo/v1.2.3/foo-darwin-arm64.tar.gz", server.URL): "fb2fe41a34b77ee180def0cb9a222d8776a6e581106009b64f35983da291ab6e",
			fmt.Sprintf("%s/foo/v1.2.3/foo-linux-amd64.tar.gz", server.URL):  "fb2fe41a34b77ee180def0cb9a222d8776a6e581106009b64f35983da291ab6e",
			fmt.Sprintf("%s/foo/v1.2.3/foo-windows-amd64.zip", server.URL):   "141aad02bfacdd9e9e0460459d572fbabda2b47c39c26ad82b4ea3b4f1548545",
		}
		require.Equal(t, wantDep, gotCfg.Dependencies["foo"])
		require.NotEmpty(t, gotCfg.Templates["origin#tmpl1"])
		require.Equal(t, wantChecksums, gotCfg.URLChecksums)
	})
}

func Test_dependencyValidateCmd(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		downloadablesDir := filepath.FromSlash("../../testdata/downloadables")
		tar := filepath.Join(downloadablesDir, "runnable.tar.gz")
		zip := filepath.Join(downloadablesDir, "runnable_windows.zip")

		server := serveFiles(t, map[string]string{
			"/foo/v1.2.3/foo-darwin-amd64.tar.gz": tar,
			"/foo/v1.2.3/foo-darwin-arm64.tar.gz": tar,
			"/foo/v1.2.3/foo-linux-amd64.tar.gz":  tar,
			"/foo/v1.2.3/foo-windows-amd64.zip":   zip,
		})

		runner := newCmdRunner(t)
		runner.writeConfig(&bindown.Config{
			Systems: []bindown.SystemInfo{
				{OS: "darwin", Arch: "amd64"},
				{OS: "darwin", Arch: "arm64"},
				{OS: "linux", Arch: "amd64"},
				{OS: "windows", Arch: "amd64"},
			},
			Dependencies: map[string]*bindown.Dependency{
				"foo": {
					URL:         ptr("{{ .addr }}/foo/v{{ .version }}/foo-{{ .os }}-{{ .arch }}{{ .urlsuffix }}"),
					ArchivePath: ptr("bin/runnable{{ .archivepathsuffix }}"),
					Vars: map[string]string{
						"version":           "1.2.3",
						"addr":              server.URL,
						"archivepathsuffix": ".sh",
						"urlsuffix":         ".tar.gz",
					},
					Overrides: []bindown.DependencyOverride{
						{
							OverrideMatcher: bindown.OverrideMatcher{
								"os": []string{"windows"},
							},
							Dependency: bindown.Dependency{
								Vars: map[string]string{
									"archivepathsuffix": ".bat",
									"urlsuffix":         ".zip",
								},
							},
						},
					},
				},
			},
			URLChecksums: map[string]string{
				fmt.Sprintf("%s/foo/v1.2.3/foo-darwin-amd64.tar.gz", server.URL): "fb2fe41a34b77ee180def0cb9a222d8776a6e581106009b64f35983da291ab6e",
				fmt.Sprintf("%s/foo/v1.2.3/foo-darwin-arm64.tar.gz", server.URL): "fb2fe41a34b77ee180def0cb9a222d8776a6e581106009b64f35983da291ab6e",
				fmt.Sprintf("%s/foo/v1.2.3/foo-linux-amd64.tar.gz", server.URL):  "fb2fe41a34b77ee180def0cb9a222d8776a6e581106009b64f35983da291ab6e",
				fmt.Sprintf("%s/foo/v1.2.3/foo-windows-amd64.zip", server.URL):   "141aad02bfacdd9e9e0460459d572fbabda2b47c39c26ad82b4ea3b4f1548545",
			},
		})
		result := runner.run("dependency", "validate", "foo")
		result.assertState(resultState{})
	})
}
