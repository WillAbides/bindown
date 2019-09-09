package bindownloader

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/udhos/equalfile"
)

func tmpDir(t *testing.T) (string, func()) {
	t.Helper()
	err := os.MkdirAll("tmp", 0750)
	require.NoError(t, err)
	name, err := ioutil.TempDir("tmp", "")
	require.NoError(t, err)
	return name, func() {
		err := os.RemoveAll(name)
		require.NoError(t, err)
	}
}

func serveFile(file, path, query string) *httptest.Server {
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

func assertEqualFiles(t testing.TB, want, actual string) bool {
	t.Helper()
	cmp := equalfile.New(nil, equalfile.Options{})
	equal, err := cmp.CompareFile(want, actual)
	assert.NoError(t, err)
	return assert.True(t, equal)
}

func Test_downloaders_install(t *testing.T) {
	dir, teardown := tmpDir(t)
	defer teardown()
	ts := serveFile("./testdata/downloadables/foo.tar.gz", "/foo/foo.tar.gz", "")
	d := downloaders{
		"foo": []*downloader{
			{
				Arch:     runtime.GOARCH,
				OS:       runtime.GOOS,
				URL:      ts.URL + "/foo/foo.tar.gz",
				Checksum: "f7fa712caea646575c920af17de3462fe9d08d7fe062b9a17010117d5fa4ed88",
				BinName:  "foo.txt",
				MoveFrom: "bin/foo.txt",
			},
		},
	}
	err := d.installTool("foo", dir, true)
	assert.NoError(t, err)
}

func Test_fromFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir, teardown := tmpDir(t)
		defer teardown()
		file := filepath.Join(dir, "buildtools.json")

		// language=json
		content := `
{
  "gobin": [
    {
      "os": "darwin",
	  "arch": "amd64",
      "url": "https://github.com/myitcv/gobin/releases/download/v0.0.10/darwin-amd64",
      "checksum": "84ed966949e06bebd7d006bc343caf9d736932fd8b37df5cb5b268a28d07bd30",
      "move-from": "darwin-amd64",
      "bin": "gobin"
    },
    {
      "os": "linux",
	  "arch": "amd64",
      "url": "https://github.com/myitcv/gobin/releases/download/v0.0.10/linux-amd64",
      "checksum": "415266d9af98578067051653f5057ea267c51ebf085408df48b118a8b978bac6",
      "move-from": "linux-amd64",
      "bin": "gobin"
    }
  ]
}
`
		err := ioutil.WriteFile(file, []byte(content), 0640)
		require.NoError(t, err)
		d, err := fromFile(file)
		assert.NoError(t, err)
		assert.Equal(t, "gobin", d["gobin"][0].BinName)
	})
}

func Test_downloadFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir, teardown := tmpDir(t)
		defer teardown()
		ts := serveFile("./testdata/downloadables/foo.tar.gz", "/foo/foo.tar.gz", "")
		err := downloadFile(filepath.Join(dir, "bar.tar.gz"), ts.URL+"/foo/foo.tar.gz")
		assert.NoError(t, err)
		assertEqualFiles(t, "./testdata/downloadables/foo.tar.gz", filepath.Join(dir, "bar.tar.gz"))
	})

	t.Run("404", func(t *testing.T) {
		dir, teardown := tmpDir(t)
		defer teardown()
		ts := serveFile("./testdata/downloadables/foo.tar.gz", "/foo/foo.tar.gz", "")
		err := downloadFile(filepath.Join(dir, "bar.tar.gz"), ts.URL+"/wrongpath")
		assert.Error(t, err)
	})

	t.Run("bad url", func(t *testing.T) {
		dir, teardown := tmpDir(t)
		defer teardown()
		err := downloadFile(filepath.Join(dir, "bar.tar.gz"), "https://bad/url")
		assert.Error(t, err)
	})

	t.Run("bad target", func(t *testing.T) {
		dir, teardown := tmpDir(t)
		defer teardown()
		ts := serveFile("./testdata/downloadables/foo.tar.gz", "/foo/foo.tar.gz", "")
		err := downloadFile(filepath.Join(dir, "notreal", "bar.tar.gz"), ts.URL+"/foo/foo.tar.gz")
		assert.Error(t, err)
	})
}

func Test_downloader_validateChecksum(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir, teardown := tmpDir(t)
		defer teardown()
		d := &downloader{
			URL:      "foo/foo.tar.gz",
			Checksum: "f7fa712caea646575c920af17de3462fe9d08d7fe062b9a17010117d5fa4ed88",
		}
		err := copy.Copy(
			filepath.FromSlash("testdata/downloadables/foo.tar.gz"),
			filepath.Join(dir, "foo.tar.gz"),
		)
		require.NoError(t, err)
		err = d.validateChecksum(dir)
		assert.NoError(t, err)
		assert.True(t, fileExists(filepath.Join(dir, "foo.tar.gz")))
	})

	t.Run("missing file", func(t *testing.T) {
		dir, teardown := tmpDir(t)
		defer teardown()
		d := &downloader{
			URL:      "foo/foo.tar.gz",
			Checksum: "f7fa712caea646575c920af17de3462fe9d08d7fe062b9a17010117d5fa4ed88",
		}

		err := d.validateChecksum(dir)
		assert.Error(t, err)
	})

	t.Run("mismatch", func(t *testing.T) {
		dir, teardown := tmpDir(t)
		defer teardown()
		d := &downloader{
			URL:      "foo/foo.tar.gz",
			Checksum: "deadbeef",
		}
		err := copy.Copy(
			filepath.FromSlash("testdata/downloadables/foo.tar.gz"),
			filepath.Join(dir, "foo.tar.gz"),
		)
		require.NoError(t, err)
		err = d.validateChecksum(dir)
		assert.Error(t, err)
		assert.False(t, fileExists(filepath.Join(dir, "foo.tar.gz")))
	})
}
