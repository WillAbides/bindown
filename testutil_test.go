package bindown

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func newSystemInfo(goOs, goArch string) SystemInfo {
	return SystemInfo{
		OS:   goOs,
		Arch: goArch,
	}
}

func ptr[T any](val T) *T {
	return &val
}

func tmpDir(t testing.TB) string {
	dir := t.TempDir()
	t.Cleanup(func() {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			return os.Chmod(path, 0o777)
		})
		assert.NoError(t, err)
	})
	return dir
}
