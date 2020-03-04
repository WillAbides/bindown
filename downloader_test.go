package bindown

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustCopyFile(t *testing.T, src, dst string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0750))
	require.NoError(t, copyFile(src, dst))
}

func Test_downloadFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir := tmpDir(t)
		ts := serveFile(t, downloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
		err := downloadFile(filepath.Join(dir, "bar.tar.gz"), ts.URL+"/foo/foo.tar.gz")
		assert.NoError(t, err)
		assertEqualFiles(t, downloadablesPath("foo.tar.gz"), filepath.Join(dir, "bar.tar.gz"))
	})

	t.Run("404", func(t *testing.T) {
		dir := tmpDir(t)
		ts := serveFile(t, downloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
		err := downloadFile(filepath.Join(dir, "bar.tar.gz"), ts.URL+"/wrongpath")
		assert.Error(t, err)
	})

	t.Run("bad url", func(t *testing.T) {
		dir := tmpDir(t)
		err := downloadFile(filepath.Join(dir, "bar.tar.gz"), "https://bad/url")
		assert.Error(t, err)
	})

	t.Run("bad target", func(t *testing.T) {
		dir := tmpDir(t)
		ts := serveFile(t, downloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
		err := downloadFile(filepath.Join(dir, "notreal", "bar.tar.gz"), ts.URL+"/foo/foo.tar.gz")
		assert.Error(t, err)
	})
}

func Test_downloader_validateChecksum(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir := tmpDir(t)
		d := &Downloader{
			URL:      "foo/foo.tar.gz",
			Checksum: "f7fa712caea646575c920af17de3462fe9d08d7fe062b9a17010117d5fa4ed88",
		}
		mustCopyFile(t, downloadablesPath("foo.tar.gz"), filepath.Join(dir, "foo.tar.gz"))
		err := d.validateChecksum(dir)
		assert.NoError(t, err)
		assert.True(t, fileExists(filepath.Join(dir, "foo.tar.gz")))
	})

	t.Run("missing file", func(t *testing.T) {
		dir := tmpDir(t)
		d := &Downloader{
			URL:      "foo/foo.tar.gz",
			Checksum: "f7fa712caea646575c920af17de3462fe9d08d7fe062b9a17010117d5fa4ed88",
		}

		err := d.validateChecksum(dir)
		assert.Error(t, err)
	})

	t.Run("mismatch", func(t *testing.T) {
		dir := tmpDir(t)
		d := &Downloader{
			URL:      "foo/foo.tar.gz",
			Checksum: "deadbeef",
		}
		mustCopyFile(t, downloadablesPath("foo.tar.gz"), filepath.Join(dir, "foo.tar.gz"))
		err := d.validateChecksum(dir)
		assert.Error(t, err)
		assert.False(t, fileExists(filepath.Join(dir, "foo.tar.gz")))
	})
}

func TestDownloader_extract(t *testing.T) {
	dir := tmpDir(t)
	d := &Downloader{
		URL:      "foo/foo.tar.gz",
		Checksum: "f7fa712caea646575c920af17de3462fe9d08d7fe062b9a17010117d5fa4ed88",
	}
	downloadDir := filepath.Join(dir, "download")
	extractDir := filepath.Join(dir, "extract")
	mustCopyFile(t, downloadablesPath("foo.tar.gz"), filepath.Join(downloadDir, "foo.tar.gz"))
	err := d.extract(downloadDir, extractDir)
	assert.NoError(t, err)
}

func TestDownloader_Install(t *testing.T) {
	t.Run("raw file", func(t *testing.T) {
		dir := tmpDir(t)
		servePath := downloadablesPath("rawfile/foo")
		ts := serveFile(t, servePath, "/foo/foo", "")
		d := &Downloader{
			URL:      ts.URL + "/foo/foo",
			Checksum: "f044ff8b6007c74bcc1b5a5c92776e5d49d6014f5ff2d551fab115c17f48ac41",
			BinName:  "foo",
			Arch:     "amd64",
			OS:       "darwin",
		}
		err := d.Install(InstallOpts{
			TargetDir: dir,
			Force:     true,
		})
		assert.NoError(t, err)
		assert.True(t, fileExists(filepath.Join(dir, "foo")))
		stat, err := os.Stat(filepath.Join(dir, "foo"))
		assert.NoError(t, err)
		assert.False(t, stat.IsDir())
		assert.Equal(t, os.FileMode(0755), stat.Mode().Perm())
	})

	t.Run("bin in root", func(t *testing.T) {
		dir := tmpDir(t)
		servePath := downloadablesPath("fooinroot.tar.gz")
		ts := serveFile(t, servePath, "/foo/fooinroot.tar.gz", "")
		d := &Downloader{
			URL:      ts.URL + "/foo/fooinroot.tar.gz",
			Checksum: "27dcce60d1ed72920a84dd4bc01e0bbd013e5a841660e9ee2e964e53fb83c0b3",
			BinName:  "foo",
			Arch:     "amd64",
			OS:       "darwin",
		}
		err := d.Install(InstallOpts{
			TargetDir: dir,
			Force:     true,
		})
		assert.NoError(t, err)
		assert.True(t, fileExists(filepath.Join(dir, "foo")))
		stat, err := os.Stat(filepath.Join(dir, "foo"))
		assert.NoError(t, err)
		assert.False(t, stat.IsDir())
		assert.Equal(t, os.FileMode(0755), stat.Mode().Perm())
	})

	t.Run("move", func(t *testing.T) {
		dir := tmpDir(t)
		ts := serveFile(t, downloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "foo=bar")
		d := &Downloader{
			URL:         ts.URL + "/foo/foo.tar.gz?foo=bar",
			Checksum:    "f7fa712caea646575c920af17de3462fe9d08d7fe062b9a17010117d5fa4ed88",
			BinName:     "foo.txt",
			ArchivePath: "bin/foo.txt",
			Arch:        "amd64",
			OS:          "darwin",
		}
		err := d.Install(InstallOpts{
			TargetDir: dir,
			Force:     true,
		})
		assert.NoError(t, err)
	})

	t.Run("legacy MoveFrom", func(t *testing.T) {
		dir := tmpDir(t)
		ts := serveFile(t, downloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "foo=bar")
		d := &Downloader{
			URL:      ts.URL + "/foo/foo.tar.gz?foo=bar",
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
	})

	t.Run("link", func(t *testing.T) {
		dir := tmpDir(t)
		ts := serveFile(t, downloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "foo=bar")
		d := &Downloader{
			URL:        ts.URL + "/foo/foo.tar.gz?foo=bar",
			Checksum:   "f7fa712caea646575c920af17de3462fe9d08d7fe062b9a17010117d5fa4ed88",
			BinName:    "foo",
			LinkSource: "bin/foo.txt",
			Arch:       "amd64",
			OS:         "darwin",
		}
		err := d.Install(InstallOpts{
			TargetDir: dir,
			Force:     true,
		})
		assert.NoError(t, err)
		linksTo, err := os.Readlink(filepath.Join(dir, "foo"))
		assert.NoError(t, err)
		absLinkTo := filepath.Join(dir, linksTo)
		assert.True(t, fileExists(absLinkTo))
	})

	t.Run("legacy LinkSource", func(t *testing.T) {
		dir := tmpDir(t)
		ts := serveFile(t, downloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "foo=bar")
		d := &Downloader{
			URL:         ts.URL + "/foo/foo.tar.gz?foo=bar",
			Checksum:    "f7fa712caea646575c920af17de3462fe9d08d7fe062b9a17010117d5fa4ed88",
			BinName:     "foo",
			ArchivePath: "bin/foo.txt",
			Link:        true,
			Arch:        "amd64",
			OS:          "darwin",
		}
		err := d.Install(InstallOpts{
			TargetDir: dir,
			Force:     true,
		})
		assert.NoError(t, err)
		linksTo, err := os.Readlink(filepath.Join(dir, "foo"))
		assert.NoError(t, err)
		absLinkTo := filepath.Join(dir, linksTo)
		assert.True(t, fileExists(absLinkTo))
	})
}
