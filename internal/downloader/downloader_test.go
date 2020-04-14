package downloader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v3/internal/testutil"
	"github.com/willabides/bindown/v3/internal/util"
)

func Test_downloadFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
		err := downloadFile(filepath.Join(dir, "bar.tar.gz"), ts.URL+"/foo/foo.tar.gz")
		assert.NoError(t, err)
		testutil.AssertEqualFiles(t, testutil.DownloadablesPath("foo.tar.gz"), filepath.Join(dir, "bar.tar.gz"))
	})

	t.Run("404", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
		err := downloadFile(filepath.Join(dir, "bar.tar.gz"), ts.URL+"/wrongpath")
		assert.Error(t, err)
	})

	t.Run("bad url", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		err := downloadFile(filepath.Join(dir, "bar.tar.gz"), "https://bad/url")
		assert.Error(t, err)
	})

	t.Run("bad target", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "")
		err := downloadFile(filepath.Join(dir, "notreal", "bar.tar.gz"), ts.URL+"/foo/foo.tar.gz")
		assert.Error(t, err)
	})
}

func Test_Downloader_validateChecksum(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		d := &Downloader{
			URL:         "foo/foo.tar.gz",
			tmplApplied: true,
		}
		require.NoError(t, util.CopyFile(testutil.DownloadablesPath("foo.tar.gz"), filepath.Join(dir, "foo.tar.gz"), nil))
		err := d.validateChecksum(dir, testutil.FooChecksum)
		assert.NoError(t, err)
		assert.True(t, util.FileExists(filepath.Join(dir, "foo.tar.gz")))
	})

	t.Run("missing file", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		d := &Downloader{
			URL:         "foo/foo.tar.gz",
			tmplApplied: true,
		}

		err := d.validateChecksum(dir, testutil.FooChecksum)
		assert.Error(t, err)
	})

	t.Run("mismatch", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		d := &Downloader{
			URL:         "foo/foo.tar.gz",
			tmplApplied: true,
		}
		require.NoError(t, util.CopyFile(testutil.DownloadablesPath("foo.tar.gz"), filepath.Join(dir, "foo.tar.gz"), nil))
		err := d.validateChecksum(dir, "deadbeef")
		assert.Error(t, err)
		assert.False(t, util.FileExists(filepath.Join(dir, "foo.tar.gz")))
	})
}

func TestDownloader_extract(t *testing.T) {
	dir := testutil.TmpDir(t)
	d := &Downloader{
		URL:         "foo/foo.tar.gz",
		tmplApplied: true,
	}
	downloadDir := filepath.Join(dir, "download")
	extractDir := filepath.Join(dir, "extract")
	require.NoError(t, os.MkdirAll(downloadDir, 0750))
	err := util.CopyFile(testutil.DownloadablesPath("foo.tar.gz"), filepath.Join(downloadDir, "foo.tar.gz"), nil)
	require.NoError(t, err)
	err = d.Extract(downloadDir, extractDir)
	assert.NoError(t, err)
}
