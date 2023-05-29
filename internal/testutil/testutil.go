package testutil

import (
	"bytes"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func AssertExecutable(t *testing.T, mode fs.FileMode) {
	t.Helper()
	// Windows doesn't have executable bits
	if runtime.GOOS == "windows" {
		return
	}
	assert.Equal(t, fs.FileMode(0o110), mode&0o110)
}
