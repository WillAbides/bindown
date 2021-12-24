package testutil

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
)

// FooChecksum is the checksum of downloadablesPath("foo.tar.gz")
const FooChecksum = "f7fa712caea646575c920af17de3462fe9d08d7fe062b9a17010117d5fa4ed88"

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
