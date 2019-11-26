package testhelper

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

// ProjectPath exchanges a path relative to the project root for an absolute path
func ProjectPath(path ...string) string {
	return filepath.Join(ProjectRoot(), filepath.Join(path...))
}

// ProjectRoot returns the absolute path of the project root
func ProjectRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// TmpDir returns the path to a newly created tmp dir and a function for deleting that dir
func TmpDir(t *testing.T) (string, func()) {
	t.Helper()
	projectTmp := ProjectPath("tmp")
	err := os.MkdirAll(projectTmp, 0750)
	require.NoError(t, err)
	tmpdir, err := ioutil.TempDir(projectTmp, "")
	require.NoError(t, err)
	return tmpdir, func() {
		require.NoError(t, os.RemoveAll(tmpdir))
	}
}

//DownloadablesPath path to a file or directory in testdata/downloadables
func DownloadablesPath(path string) string {
	return filepath.Join(ProjectPath("testdata", "downloadables"), filepath.FromSlash(path))
}

//ServeFile runs a test server
func ServeFile(file, path, query string) *httptest.Server {
	file = filepath.FromSlash(file)
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.RawQuery != query {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.ServeFile(w, req, file)
	})
	return httptest.NewServer(mux)
}

//AssertEqualFiles asserts that files at want and actual have the same content
func AssertEqualFiles(t testing.TB, want, actual string) bool {
	t.Helper()
	cmp := equalfile.New(nil, equalfile.Options{})
	equal, err := cmp.CompareFile(want, actual)
	assert.NoError(t, err)
	return assert.True(t, equal)
}
