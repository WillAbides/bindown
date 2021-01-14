package testutil

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

// FooChecksum is the checksum of downloadablesPath("foo.tar.gz")
const FooChecksum = "f7fa712caea646575c920af17de3462fe9d08d7fe062b9a17010117d5fa4ed88"

// MustWriteFile write a file or fails
func MustWriteFile(t *testing.T, filename string, content []byte) {
	t.Helper()
	err := ioutil.WriteFile(filename, content, 0o600)
	require.NoError(t, err)
}

// MustReadFile reads a file or fails
func MustReadFile(t *testing.T, filename string) []byte {
	t.Helper()
	got, err := ioutil.ReadFile(filename) //nolint:gosec
	require.NoError(t, err)
	return got
}

// ProjectRoot returns the absolute path of the project root
func ProjectRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(file, "..", "..", "..")
}

// DownloadablesPath path to testdata/downloadables
func DownloadablesPath(path string) string {
	return filepath.Join(ProjectPath("testdata", "downloadables"), filepath.FromSlash(path))
}

// ProjectPath exchanges a path relative to the project root for an absolute path
func ProjectPath(path ...string) string {
	return filepath.Join(ProjectRoot(), filepath.Join(path...))
}

// TmpDir returns the path to a newly created tmp dir and a function for deleting that dir
func TmpDir(t *testing.T) string {
	t.Helper()
	projectTmp := ProjectPath("tmp")
	err := os.MkdirAll(projectTmp, 0o750)
	require.NoError(t, err)
	tmpdir, err := ioutil.TempDir(projectTmp, "")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(tmpdir))
	})
	return tmpdir
}

// ChDir changes the working directory for the duration of the test
func ChDir(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		t.Helper()
		require.NoError(t, os.Chdir(wd))
	})
}

// ServeFile starts an http server
func ServeFile(t *testing.T, file, path, query string) *httptest.Server {
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

// AssertEqualFiles asserts two files are equal
func AssertEqualFiles(t testing.TB, want, actual string) bool {
	t.Helper()
	cmp := equalfile.New(nil, equalfile.Options{})
	equal, err := cmp.CompareFile(want, actual)
	assert.NoError(t, err)
	return assert.True(t, equal)
}
