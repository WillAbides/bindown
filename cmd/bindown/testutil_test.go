package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Netflix/go-expect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v4/internal/bindown"
	"github.com/willabides/bindown/v4/internal/expecttest"
	"gopkg.in/yaml.v3"
)

type simpleFileReader struct {
	io.Reader
}

func (s simpleFileReader) Fd() uintptr {
	return 0
}

type cmdRunner struct {
	t          testing.TB
	configFile string
	cache      string
	tmpDir     string
	stdin      io.Reader
}

func newCmdRunner(t testing.TB) *cmdRunner {
	t.Helper()
	dir := testTmp(t)
	cacheDir := filepath.Join(dir, "cache")
	configfile := filepath.Join(dir, ".bindown.yaml")
	runner := &cmdRunner{
		t:          t,
		cache:      cacheDir,
		configFile: configfile,
		tmpDir:     dir,
	}
	t.Cleanup(func() {
		// ignore errors because it fails on test with missing or invalid config files
		runner.run("cache", "clear")
	})
	return runner
}

func (c *cmdRunner) run(commandLine ...string) *runCmdResult {
	ctx := context.Background()
	c.t.Helper()
	result := runCmdResult{t: c.t}
	if c.configFile != "" {
		commandLine = append(commandLine, "--configfile", c.configFile)
	}
	if c.cache != "" {
		commandLine = append(commandLine, "--cache", c.cache)
	}
	Run(
		ctx,
		commandLine,
		&runOpts{
			stdin:   simpleFileReader{c.stdin},
			stdout:  SimpleFileWriter{&result.stdOut},
			stderr:  SimpleFileWriter{&result.stdErr},
			cmdName: "cmd",
			exitHandler: func(i int) {
				result.exited = true
				result.exitVal = i
			},
		},
	)
	return &result
}

func (c *cmdRunner) runExpect(expectFunc func(*expect.Console), commandLine ...string) *runCmdResult {
	t := c.t
	t.Helper()
	ctx := context.Background()

	if c.configFile != "" {
		commandLine = append(commandLine, "--configfile", c.configFile)
	}
	if c.cache != "" {
		commandLine = append(commandLine, "--cache", c.cache)
	}

	result := runCmdResult{t: t}

	testFunc := func(console *expect.Console) {
		Run(
			ctx,
			commandLine,
			&runOpts{
				stdin:   console.Tty(),
				stdout:  console.Tty(),
				stderr:  console.Tty(),
				cmdName: "cmd",
				exitHandler: func(i int) {
					result.exited = true
					result.exitVal = i
				},
			},
		)
	}

	expecttest.Run(t, expectFunc, testFunc, expecttest.WithConsoleOpt(expect.WithStdout(&result.stdOut)))

	return &result
}

func mustConfigFromYAML(t *testing.T, yml string) *bindown.Config {
	t.Helper()
	got, err := bindown.ConfigFromYAML(context.Background(), []byte(yml))
	require.NoError(t, err)
	return got
}

func (c *cmdRunner) writeConfigYaml(content string) {
	c.t.Helper()
	err := os.WriteFile(c.configFile, []byte(content), 0o600)
	assert.NoError(c.t, err)
}

func (c *cmdRunner) getConfigFile() *bindown.Config {
	c.t.Helper()
	cfgFile, err := bindown.NewConfig(context.Background(), c.configFile, false)
	assert.NoError(c.t, err)
	return cfgFile
}

func (c *cmdRunner) assertConfigYaml(want string) {
	c.t.Helper()
	got, err := os.ReadFile(c.configFile)
	if !assert.NoError(c.t, err) {
		return
	}
	assert.Equal(c.t, normalizeYaml(c.t, want), normalizeYaml(c.t, string(got)))
}

func normalizeYaml(t testing.TB, val string) string {
	t.Helper()
	var data any
	err := yaml.Unmarshal([]byte(val), &data)
	require.NoError(t, err)
	var out bytes.Buffer
	err = bindown.EncodeYaml(&out, data)
	require.NoError(t, err)
	return out.String()
}

type runCmdResult struct {
	t       testing.TB
	stdOut  bytes.Buffer
	stdErr  bytes.Buffer
	exited  bool
	exitVal int
}

func (r *runCmdResult) assertStdOut(want string) {
	r.t.Helper()
	assertEqualOrMatch(r.t, want, r.stdOut.String())
}

func (r *runCmdResult) assertStdErr(want string) {
	r.t.Helper()
	assertEqualOrMatch(r.t, want, r.stdErr.String())
}

func (r *runCmdResult) getExtractDir() string {
	r.t.Helper()
	stdout := r.stdOut.String()
	re := regexp.MustCompile(`(?m)^extracted .+ to (.*)$`)
	matches := re.FindStringSubmatch(stdout)
	if !assert.Len(r.t, matches, 2) {
		return ""
	}
	return matches[1]
}

type resultState struct {
	stdout string
	stderr string
	exit   int
}

func (r *runCmdResult) assertState(state resultState) {
	r.t.Helper()
	r.assertStdOut(state.stdout)
	r.assertStdErr(state.stderr)
	assert.Equal(r.t, state.exit, r.exitVal)
	assert.Equal(r.t, state.exit != 0, r.exited)
}

// fooChecksum is the checksum of downloadablesPath("foo.tar.gz")
const fooChecksum = "f7fa712caea646575c920af17de3462fe9d08d7fe062b9a17010117d5fa4ed88"

func serveErr(t *testing.T, errCode int) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, http.StatusText(errCode), errCode)
	}))
	t.Cleanup(ts.Close)
	return ts
}

func assertEqualOrMatch(t testing.TB, want, got string) {
	t.Helper()
	if want == "" {
		assert.Equal(t, "", got)
		return
	}
	want = strings.TrimSpace(want)
	got = strings.TrimSpace(got)
	if want == got {
		return
	}
	re, err := regexp.Compile(want)
	if err != nil {
		assert.Equal(t, strings.TrimSpace(want), got)
		return
	}
	assert.Regexp(t, re, got)
}

func testInDir(t testing.TB, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if !assert.NoError(t, err) {
		return
	}
	t.Cleanup(func() {
		assert.NoError(t, os.Chdir(orig))
	})
	assert.NoError(t, os.Chdir(dir))
}

// testTmp is like t.TempDir but it uses a directory in this repo's tmp directory.
// This is useful so that there can be a relative path from the resulting directory to
// directories in this repo.
func testTmp(t testing.TB) string {
	t.Helper()
	tmpDir := filepath.FromSlash("../../tmp/_test")
	err := os.MkdirAll(tmpDir, 0o777)
	require.NoError(t, err)
	dir, err := os.MkdirTemp(tmpDir, "")
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(dir))
	})
	abs, err := filepath.Abs(dir)
	require.NoError(t, err)
	return abs
}

func testdataPath(f string) string {
	return filepath.Join(
		filepath.FromSlash("../../internal/bindown/testdata"),
		filepath.FromSlash(f),
	)
}
