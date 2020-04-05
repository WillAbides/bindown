package bindown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v2/internal/testutil"
	"github.com/willabides/bindown/v2/internal/util"
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
			Checksum:    testutil.FooChecksum,
			tmplApplied: true,
		}
		require.NoError(t, util.CopyFile(testutil.DownloadablesPath("foo.tar.gz"), filepath.Join(dir, "foo.tar.gz"), nil))
		err := d.validateChecksum(dir, nil)
		assert.NoError(t, err)
		assert.True(t, fileExists(filepath.Join(dir, "foo.tar.gz")))
	})

	t.Run("missing file", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		d := &Downloader{
			URL:         "foo/foo.tar.gz",
			Checksum:    testutil.FooChecksum,
			tmplApplied: true,
		}

		err := d.validateChecksum(dir, nil)
		assert.Error(t, err)
	})

	t.Run("mismatch", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		d := &Downloader{
			URL:         "foo/foo.tar.gz",
			Checksum:    "deadbeef",
			tmplApplied: true,
		}
		require.NoError(t, util.CopyFile(testutil.DownloadablesPath("foo.tar.gz"), filepath.Join(dir, "foo.tar.gz"), nil))
		err := d.validateChecksum(dir, nil)
		assert.Error(t, err)
		assert.False(t, fileExists(filepath.Join(dir, "foo.tar.gz")))
	})
}

func TestDownloader_extract(t *testing.T) {
	dir := testutil.TmpDir(t)
	d := &Downloader{
		URL:         "foo/foo.tar.gz",
		Checksum:    testutil.FooChecksum,
		tmplApplied: true,
	}
	downloadDir := filepath.Join(dir, "download")
	extractDir := filepath.Join(dir, "extract")
	require.NoError(t, os.MkdirAll(downloadDir, 0750))
	err := util.CopyFile(testutil.DownloadablesPath("foo.tar.gz"), filepath.Join(downloadDir, "foo.tar.gz"), nil)
	require.NoError(t, err)
	err = d.extract(downloadDir, extractDir)
	assert.NoError(t, err)
}

func TestDownloader_Install(t *testing.T) {
	t.Run("raw file", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		servePath := testutil.DownloadablesPath("rawfile/foo")
		ts := testutil.ServeFile(t, servePath, "/foo/foo", "")
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
		dir := testutil.TmpDir(t)
		servePath := testutil.DownloadablesPath("fooinroot.tar.gz")
		ts := testutil.ServeFile(t, servePath, "/foo/fooinroot.tar.gz", "")
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

	t.Run("invalid url", func(t *testing.T) {
		d := &Downloader{
			URL: "://foo.com",
		}
		err := d.Install(InstallOpts{})
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "parse"))
	})

	t.Run("invalid target dir", func(t *testing.T) {
		dir := testutil.TmpDir(t) + "/" + string(byte(0)) + "/"
		d := &Downloader{}
		err := d.Install(InstallOpts{
			TargetDir: dir,
		})
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "mkdir"))
	})

	t.Run("move", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "foo=bar")
		d := &Downloader{
			URL:         ts.URL + "/foo/foo.tar.gz?foo=bar",
			Checksum:    testutil.FooChecksum,
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
		require.FileExists(t, filepath.Join(dir, "foo.txt"))
	})

	t.Run("wrong checksum", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "foo=bar")
		d := &Downloader{
			URL:         ts.URL + "/foo/foo.tar.gz?foo=bar",
			Checksum:    "0000000000000000000000000000000000000000000000000000000000000000",
			BinName:     "foo.txt",
			ArchivePath: "bin/foo.txt",
			Arch:        "amd64",
			OS:          "darwin",
		}
		err := d.Install(InstallOpts{
			TargetDir: dir,
		})
		require.Error(t, err)
		require.False(t, fileExists(filepath.Join(dir, "foo.txt")))
	})

	t.Run("tar file exists", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		d := &Downloader{
			URL:         "http://invalid/foo/foo.tar.gz?foo=bar",
			Checksum:    testutil.FooChecksum,
			BinName:     "foo.txt",
			ArchivePath: "bin/foo.txt",
			Arch:        "amd64",
			OS:          "darwin",
		}
		downloadsDir := filepath.Join(dir, ".bindown", "downloads", d.downloadsSubName())
		err := os.MkdirAll(downloadsDir, 0750)
		require.NoError(t, err)
		require.NoError(t, util.CopyFile(testutil.DownloadablesPath("foo.tar.gz"), filepath.Join(downloadsDir, "foo.tar.gz"), nil))
		err = d.Install(InstallOpts{
			TargetDir: dir,
		})
		require.NoError(t, err)
	})

	t.Run("link", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "foo=bar")
		d := &Downloader{
			URL:         ts.URL + "/foo/foo.tar.gz?foo=bar",
			Checksum:    testutil.FooChecksum,
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

func TestDownloader_UpdateChecksum(t *testing.T) {
	ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "foo=bar")
	d := &Downloader{
		URL:      ts.URL + "/foo/foo.tar.gz?foo=bar",
		Checksum: "wrongchecksum",
	}
	err := d.UpdateChecksum(UpdateChecksumOpts{})
	require.NoError(t, err)
	require.Equal(t, testutil.FooChecksum, d.Checksum)
}

func TestDownloader_Validate(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "foo=bar")
		d := &Downloader{
			URL:      ts.URL + "/foo/foo.tar.gz?foo=bar",
			Checksum: "wrongchecksum",
			OS:       "darwin",
			Arch:     "amd64",
		}
		err := d.Validate(ValidateOpts{
			DownloaderName: "foo",
		})
		assert.Error(t, err)
		assert.True(t, strings.HasPrefix(err.Error(), "could not validate downloader"))
	})

	t.Run("valid", func(t *testing.T) {
		ts := testutil.ServeFile(t, testutil.DownloadablesPath("foo.tar.gz"), "/foo/foo.tar.gz", "foo=bar")
		d := &Downloader{
			URL:      ts.URL + "/foo/foo.tar.gz?foo=bar",
			Checksum: testutil.FooChecksum,
			OS:       "darwin",
			Arch:     "amd64",
		}
		err := d.Validate(ValidateOpts{
			DownloaderName: "bin/foo.txt",
		})
		assert.NoError(t, err)
	})
}
