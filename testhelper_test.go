package bindown

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/udhos/equalfile"
)

func downloadablesPath(path string) string {
	return filepath.Join(projectPath("testdata", "downloadables"), filepath.FromSlash(path))
}

// projectPath exchanges a path relative to the project root for an absolute path
func projectPath(path ...string) string {
	return filepath.Join(projectRoot(), filepath.Join(path...))
}

// projectRoot returns the absolute path of the project root
func projectRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}

// tmpDir returns the path to a newly created tmp dir and a function for deleting that dir
func tmpDir(t *testing.T) string {
	t.Helper()
	projectTmp := projectPath("tmp")
	err := os.MkdirAll(projectTmp, 0750)
	require.NoError(t, err)
	tmpdir, err := ioutil.TempDir(projectTmp, "")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(tmpdir))
	})
	return tmpdir
}

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

func assertEqualFiles(t testing.TB, want, actual string) bool {
	t.Helper()
	cmp := equalfile.New(nil, equalfile.Options{})
	equal, err := cmp.CompareFile(want, actual)
	assert.NoError(t, err)
	return assert.True(t, equal)
}
