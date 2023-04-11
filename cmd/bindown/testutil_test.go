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

	"github.com/stretchr/testify/assert"
	"github.com/willabides/bindown/v3"
)

type cmdRunner struct {
	t          testing.TB
	configFile string
	cache      string
	tmpDir     string
	stdin      io.Reader
}

func newCmdRunner(t testing.TB) *cmdRunner {
	t.Helper()
	dir := t.TempDir()
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
			stdin:   c.stdin,
			stdout:  &result.stdOut,
			stderr:  &result.stdErr,
			cmdName: "cmd",
			exitHandler: func(i int) {
				result.exited = true
				result.exitVal = i
			},
		},
	)
	return &result
}

func (c *cmdRunner) writeConfigYaml(content string) {
	c.t.Helper()
	err := os.WriteFile(c.configFile, []byte(content), 0o600)
	assert.NoError(c.t, err)
}

func (c *cmdRunner) writeConfig(config *bindown.Config) {
	c.t.Helper()
	cfgFile := &bindown.ConfigFile{
		Filename: c.configFile,
		Config:   *config,
	}
	assert.NoError(c.t, cfgFile.Write(false))
}

func (c *cmdRunner) getConfigFile() *bindown.ConfigFile {
	c.t.Helper()
	cfgFile, err := bindown.LoadConfigFile(context.Background(), c.configFile, false)
	assert.NoError(c.t, err)
	return cfgFile
}

func (c *cmdRunner) assertConfigYaml(want string) {
	c.t.Helper()
	got, err := os.ReadFile(c.configFile)
	if !assert.NoError(c.t, err) {
		return
	}
	assert.Equal(c.t, strings.TrimSpace(want), strings.TrimSpace(string(got)))
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

// serveFile starts an HTTP server
func serveFile(t *testing.T, file, path, query string) *httptest.Server {
	t.Helper()
	file = filepath.FromSlash(file)
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.RawQuery != query {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.ServeFile(w, req, file)
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

// serveFiles starts an HTTP server serving the given files.
// files is a map of URL paths to local file paths.
func serveFiles(t *testing.T, files map[string]string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for path, file := range files {
		f := filepath.FromSlash(file)
		mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
			http.ServeFile(w, req, f)
		})
	}
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func serveErr(t *testing.T, errCode int) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, http.StatusText(errCode), errCode)
	}))
	t.Cleanup(ts.Close)
	return ts
}

func ptr[T any](val T) *T {
	return &val
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
