package cli

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v3"
	"github.com/willabides/bindown/v3/internal/configfile"
	"github.com/willabides/bindown/v3/internal/testutil"
	"github.com/willabides/bindown/v3/internal/util"
)

type runCmdResult struct {
	stdOut  bytes.Buffer
	stdErr  bytes.Buffer
	exited  bool
	exitVal int
}

func (r runCmdResult) assertStdOut(t *testing.T, want string) {
	t.Helper()
	assert.Equal(t, want, strings.TrimSpace(r.stdOut.String()))
}

func (r runCmdResult) assertStdErr(t *testing.T, want string) {
	t.Helper()
	assert.Equal(t, want, strings.TrimSpace(r.stdErr.String()))
}

func (r runCmdResult) assertError(t *testing.T, msg string) {
	t.Helper()
	assert.NotZero(t, r.exitVal)
	wantPrefix := fmt.Sprintf("cmdname: error: %s", msg)
	stdErr := r.stdErr.String()
	hasPrefix := strings.HasPrefix(stdErr, wantPrefix)
	assert.True(t, hasPrefix, "stdErr should start with %q\ninstead stdErr was %q", wantPrefix, stdErr)
}

func runCmd(commandLine ...string) runCmdResult {
	result := runCmdResult{}
	Run(commandLine,
		kong.Name("cmdname"),
		kong.Writers(&result.stdOut, &result.stdErr),
		kong.Exit(func(i int) {
			result.exited = true
			result.exitVal = i
		}),
	)
	return result
}

func createConfigFile(t *testing.T, sourceFile string) string {
	t.Helper()
	sourceFile = testutil.ProjectPath("testdata", "configs", sourceFile)
	dir := testutil.TmpDir(t)
	dest := filepath.Join(dir, "bindown.config")
	err := util.CopyFile(sourceFile, dest, nil)
	require.NoError(t, err)
	return dest
}

func createConfigFileWithContent(t *testing.T, filename, content string) string {
	t.Helper()
	dir := testutil.TmpDir(t)
	dest := filepath.Join(dir, filename)
	err := ioutil.WriteFile(dest, []byte(content), 0600)
	require.NoError(t, err)
	return dest
}

func writeFileFromConfig(t *testing.T, cfg bindown.Config, writeJSON bool) string {
	t.Helper()
	dir := testutil.TmpDir(t)
	dest := filepath.Join(dir, "bindown.config")
	cfgFile := configfile.New(dest, cfg)
	err := cfgFile.Write(writeJSON)
	require.NoError(t, err)
	return dest
}

func setConfigFileEnvVar(t *testing.T, file string) {
	t.Helper()
	err := os.Setenv("BINDOWN_CONFIG_FILE", file)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("BINDOWN_CONFIG_FILE"))
	})
}

func strPtr(s string) *string {
	return &s
}

func TestAddChecksums(t *testing.T) {
	ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
	dlURL := ts.URL + "/foo/foo.tar.gz"
	cfg := bindown.Config{
		Dependencies: map[string]*bindown.Dependency{
			"foo": {
				URL:         &dlURL,
				ArchivePath: strPtr("bin/foo.txt"),
			},
		},
	}
	cfgFile := writeFileFromConfig(t, cfg, false)
	result := runCmd("add-checksums", "--configfile", cfgFile, "--system", "darwin/amd64", "foo")
	result.assertStdOut(t, "")
	result.assertStdErr(t, "")
	require.Zero(t, result.exitVal)
	gotCfg, err := configfile.LoadConfigFile(cfgFile, false)
	require.NoError(t, err)
	require.Equal(t, testutil.FooChecksum, gotCfg.URLChecksums[dlURL])
}

func TestValidate(t *testing.T) {
	ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
	dlURL := ts.URL + "/foo/foo.tar.gz"
	cfg := bindown.Config{
		Dependencies: map[string]*bindown.Dependency{
			"foo": {
				URL:         &dlURL,
				ArchivePath: strPtr("bin/foo.txt"),
			},
		},
		URLChecksums: map[string]string{
			dlURL: testutil.FooChecksum,
		},
	}
	cfgFile := writeFileFromConfig(t, cfg, false)
	result := runCmd("validate", "--configfile", cfgFile, "--system", "darwin/amd64", "foo")
	result.assertStdOut(t, "")
	result.assertStdErr(t, "")
	require.Zero(t, result.exitVal)
}

func TestInstall(t *testing.T) {
	ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
	dlURL := ts.URL + "/foo/foo.tar.gz"
	cfg := bindown.Config{
		Dependencies: map[string]*bindown.Dependency{
			"foo": {
				URL:         &dlURL,
				ArchivePath: strPtr("bin/foo.txt"),
			},
		},
		URLChecksums: map[string]string{
			dlURL: testutil.FooChecksum,
		},
	}

	cfgFile := writeFileFromConfig(t, cfg, false)
	dir := testutil.TmpDir(t)
	result := runCmd("install", "--configfile", cfgFile, "--system", "darwin/amd64", "--output", filepath.Join(dir, "foo"), "foo")
	result.assertStdOut(t, fmt.Sprintf("installed foo to %s/foo", dir))
	result.assertStdErr(t, "")
	require.Zero(t, result.exitVal)
	got := testutil.MustReadFile(t, filepath.Join(dir, "foo"))
	require.Equal(t, "bar\n", string(got))
}

func TestFormat(t *testing.T) {
	t.Run("invalid config file", func(t *testing.T) {
		cfgFile := createConfigFile(t, "invalid1.yaml")
		setConfigFileEnvVar(t, cfgFile)
		result := runCmd("format", "--configfile", cfgFile)
		result.assertStdOut(t, "")
		result.assertError(t, fmt.Sprintf(`error loading config from "%s": invalid config:
                /dependencies/golangci-lint: "boo" type should be object
                /templates/golangci-lint/link: "true" type should be boolean`, cfgFile))
		assert.NotZero(t, result.exitVal)
	})

	t.Run("formats the config file", func(t *testing.T) {
		cfgFile := createConfigFileWithContent(t, "bindown.yml", `

dependencies:
  goreleaser:
    template: goreleaser
    vars:
        version: 0.120.7
  golangci-lint:
    template: golangci-lint
    vars:
      version: 1.23.7

`)
		want := `dependencies:
  golangci-lint:
    template: golangci-lint
    vars:
      version: 1.23.7
  goreleaser:
    template: goreleaser
    vars:
      version: 0.120.7
`

		result := runCmd("format", "--configfile", cfgFile)
		result.assertStdErr(t, "")
		result.assertStdOut(t, "")
		assert.Zero(t, result.exitVal)
		got := testutil.MustReadFile(t, cfgFile)
		require.Equal(t, want, string(got))
	})

	t.Run("writes json with json extension", func(t *testing.T) {
		cfgFile := createConfigFileWithContent(t, "bindown.json", `
dependencies:
  golangci-lint:
    template: golangci-lint
    vars:
      version: 1.23.7
  goreleaser:
    template: goreleaser
    vars:
      version: 0.120.7
`)
		want := `{
  "dependencies": {
    "golangci-lint": {
      "template": "golangci-lint",
      "vars": {
        "version": "1.23.7"
      }
    },
    "goreleaser": {
      "template": "goreleaser",
      "vars": {
        "version": "0.120.7"
      }
    }
  }
}`

		result := runCmd("format", "--configfile", cfgFile)
		result.assertStdErr(t, "")
		result.assertStdOut(t, "")
		assert.Zero(t, result.exitVal)
		got := testutil.MustReadFile(t, cfgFile)
		require.JSONEq(t, want, string(got))
	})

	t.Run("config format no config file", func(t *testing.T) {
		result := runCmd("format")
		assert.NotZero(t, result.exitVal)
		result.assertStdOut(t, "")
		result.assertError(t, "error loading config")
	})
}

func TestVersion(t *testing.T) {
	result := runCmd("version")
	result.assertStdOut(t, "cmdname: version unknown")
	result.assertStdErr(t, "")
}
