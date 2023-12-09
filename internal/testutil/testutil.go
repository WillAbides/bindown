package testutil

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var bindownBinOnce sync.Once

func BindownBin() string {
	bindownBinPath := filepath.Join(RepoRoot(), "tmp", "_test", "bindown")
	bindownBinOnce.Do(func() {
		cmd := exec.Command(goExec(), "build", "-o", bindownBinPath, "./cmd/bindown")
		cmd.Dir = RepoRoot()
		err := cmd.Run()
		if err != nil {
			panic(fmt.Sprintf("error building bindown: %v", err))
		}
	})
	return bindownBinPath
}

// goExec returns te path to the go executable to use for tests.
func goExec() string {
	goRoot := runtime.GOROOT()
	if goRoot != "" {
		p := filepath.Join(goRoot, "bin", "go")
		info, err := os.Stat(p)
		if err == nil && !info.IsDir() {
			return p
		}
	}
	p, err := exec.LookPath("go")
	if err != nil {
		panic("unable to find go executable")
	}
	return p
}

// ServeFile starts an HTTP server
func ServeFile(t *testing.T, file, path, query string) *httptest.Server {
	t.Helper()
	file = filepath.FromSlash(file)
	content, err := os.ReadFile(file)
	require.NoError(t, err)
	// normalize line endings
	content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.RawQuery != query {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_, e := w.Write(content)
		assert.NoError(t, e)
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

// ServeFiles starts an HTTP server serving the given files.
// files is a map of URL paths to local file paths.
func ServeFiles(t *testing.T, files map[string]string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	contents := make(map[string][]byte, len(files))
	var err error
	for path, file := range files {
		contents[path], err = os.ReadFile(file)
		// normalize line endings
		contents[path] = bytes.ReplaceAll(contents[path], []byte("\r\n"), []byte("\n"))
		assert.NoError(t, err)
		mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
			_, e := w.Write(contents[req.URL.Path])
			assert.NoError(t, e)
		})
	}
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

// AssertFile asserts that the file at filename exists and has the given properties.
func AssertFile(t *testing.T, filename string, wantExecutable, wantLink bool) bool {
	t.Helper()
	linfo, err := os.Lstat(filename)
	if !assert.NoError(t, err) {
		return false
	}
	var ok bool
	if wantLink {
		ok = assert.True(t, linfo.Mode()&fs.ModeSymlink != 0, "expected %s to be a symlink", filename)
	} else {
		ok = assert.False(t, linfo.Mode()&fs.ModeSymlink != 0, "expected %s to not be a symlink", filename)
	}
	if !ok {
		return false
	}
	// windows doesn't have executable bit so we can't check it
	if runtime.GOOS == "windows" {
		return false
	}
	info, err := os.Stat(filename)
	if !assert.NoError(t, err) {
		return false
	}
	if wantExecutable {
		ok = assert.True(t, info.Mode()&0o110 != 0, "expected %s to be executable", filename)
	} else {
		ok = assert.False(t, info.Mode()&0o110 != 0, "expected %s to not be executable", filename)
	}
	return ok
}

// RepoRoot returns the absolute path to the root of this repo
func RepoRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(filename), "..", "..")
	abs, err := filepath.Abs(dir)
	if err != nil {
		panic(err)
	}
	return abs
}
