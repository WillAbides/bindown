package bindownloader

import (
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_downloadFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir, teardown := tmpDir(t)
		defer teardown()
		ts := serveFile(fooPath, "/foo/foo.tar.gz", "")
		err := downloadFile(filepath.Join(dir, "bar.tar.gz"), ts.URL+"/foo/foo.tar.gz")
		assert.NoError(t, err)
		assertEqualFiles(t, fooPath, filepath.Join(dir, "bar.tar.gz"))
	})

	t.Run("404", func(t *testing.T) {
		dir, teardown := tmpDir(t)
		defer teardown()
		ts := serveFile(fooPath, "/foo/foo.tar.gz", "")
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
		ts := serveFile(fooPath, "/foo/foo.tar.gz", "")
		err := downloadFile(filepath.Join(dir, "notreal", "bar.tar.gz"), ts.URL+"/foo/foo.tar.gz")
		assert.Error(t, err)
	})
}

func Test_downloader_validateChecksum(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir, teardown := tmpDir(t)
		defer teardown()
		d := &Downloader{
			URL:      "foo/foo.tar.gz",
			Checksum: "f7fa712caea646575c920af17de3462fe9d08d7fe062b9a17010117d5fa4ed88",
		}
		err := copy.Copy(
			fooPath,
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
		d := &Downloader{
			URL:      "foo/foo.tar.gz",
			Checksum: "f7fa712caea646575c920af17de3462fe9d08d7fe062b9a17010117d5fa4ed88",
		}

		err := d.validateChecksum(dir)
		assert.Error(t, err)
	})

	t.Run("mismatch", func(t *testing.T) {
		dir, teardown := tmpDir(t)
		defer teardown()
		d := &Downloader{
			URL:      "foo/foo.tar.gz",
			Checksum: "deadbeef",
		}
		err := copy.Copy(
			fooPath,
			filepath.Join(dir, "foo.tar.gz"),
		)
		require.NoError(t, err)
		err = d.validateChecksum(dir)
		assert.Error(t, err)
		assert.False(t, fileExists(filepath.Join(dir, "foo.tar.gz")))
	})
}

func TestDownloader_Install(t *testing.T) {
	dir, teardown := tmpDir(t)
	defer teardown()
	ts := serveFile(fooPath, "/foo/foo.tar.gz", "")
	d := &Downloader{
		URL:      ts.URL + "/foo/foo.tar.gz",
		Checksum: "f7fa712caea646575c920af17de3462fe9d08d7fe062b9a17010117d5fa4ed88",
		BinName:  "foo.txt",
		MoveFrom: "bin/foo.txt",
		Arch:     "amd64",
		OS:       "darwin",
	}
	err := d.Install(InstallOpts{
		TargetDir: dir,
		Force:     true,
	})
	assert.NoError(t, err)
}
