package util

import (
	"crypto/sha256"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v2/internal/testutil"
)

func TestCopyFile(t *testing.T) {
	t.Run("doesn't exist", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		err := CopyFile(src, dst, nil)
		require.Error(t, err)
	})

	t.Run("directory", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		err := os.Mkdir(src, 0750)
		require.NoError(t, err)
		err = CopyFile(src, dst, nil)
		require.Error(t, err)
	})

	t.Run("copies", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		content := []byte("foo")
		testutil.MustWriteFile(t, src, content)
		err := CopyFile(src, dst, nil)
		require.NoError(t, err)
		got := testutil.MustReadFile(t, dst)
		assert.Equal(t, content, got)
	})

	t.Run("dst directory doesn't exist", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "dst", "file2")
		content := []byte("foo")
		testutil.MustWriteFile(t, src, content)
		err := CopyFile(src, dst, nil)
		require.Error(t, err)
	})

	t.Run("overwrite", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		content := []byte("foo")
		testutil.MustWriteFile(t, src, content)
		testutil.MustWriteFile(t, dst, []byte("bar"))
		err := CopyFile(src, dst, nil)
		require.NoError(t, err)
		got := testutil.MustReadFile(t, dst)
		require.Equal(t, content, got)
	})
}

func TestExecuteTemplate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		vars := map[string]string{
			"version": "1.2.3",
		}
		tmpl := `whatever-{{.version}}/mybin-{{.os}}-{{.arch}}`
		got, err := ExecuteTemplate(tmpl, "Linux", "arm", vars)
		require.NoError(t, err)
		require.Equal(t, "whatever-1.2.3/mybin-Linux-arm", got)
	})

	t.Run("nil vars", func(t *testing.T) {
		tmpl := `whatever/mybin-{{.os}}-{{.arch}}`
		got, err := ExecuteTemplate(tmpl, "Linux", "arm", nil)
		require.NoError(t, err)
		require.Equal(t, "whatever/mybin-Linux-arm", got)
	})
}

func Test_fileExistsWithChecksum(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		file := filepath.Join(testutil.TmpDir(t), "myfile")
		require.NoError(t, CopyFile(testutil.DownloadablesPath("foo.tar.gz"), file, nil))
		got, err := FileExistsWithChecksum(file, testutil.FooChecksum)
		require.NoError(t, err)
		require.True(t, got)
	})

	t.Run("wrong checksum", func(t *testing.T) {
		file := filepath.Join(testutil.TmpDir(t), "myfile")
		checksum := "0000000000000000000000000000000000000000000000000000000000000000"
		require.NoError(t, CopyFile(testutil.DownloadablesPath("foo.tar.gz"), file, nil))
		got, err := FileExistsWithChecksum(file, checksum)
		require.NoError(t, err)
		require.False(t, got)
	})

	t.Run("doesn't exist", func(t *testing.T) {
		file := filepath.Join(testutil.TmpDir(t), "myfile")
		got, err := FileExistsWithChecksum(file, testutil.FooChecksum)
		require.NoError(t, err)
		require.False(t, got)
	})
}

func Test_fileChecksum(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		file := filepath.Join(testutil.TmpDir(t), "myfile")
		require.NoError(t, CopyFile(testutil.DownloadablesPath("foo.tar.gz"), file, nil))
		got, err := FileChecksum(file)
		require.NoError(t, err)
		require.Equal(t, testutil.FooChecksum, got)
	})

	t.Run("doesn't exist", func(t *testing.T) {
		file := filepath.Join(testutil.TmpDir(t), "myfile")
		got, err := FileChecksum(file)
		require.Error(t, err)
		require.Empty(t, got)
	})
}

func Test_hexHash(t *testing.T) {
	got, err := HexHash(fnv.New64a(), []byte("foo"))
	require.NoError(t, err)
	require.Equal(t, "dcb27518fed9d577", got)
	got, err = HexHash(fnv.New64a(), []byte("foo"), []byte("bar"))
	require.NoError(t, err)
	require.Equal(t, "85944171f73967e8", got)
	content := testutil.MustReadFile(t, testutil.DownloadablesPath("foo.tar.gz"))
	got, err = HexHash(sha256.New(), content)
	require.NoError(t, err)
	require.Equal(t, testutil.FooChecksum, got)
}

func TestWalk(t *testing.T) {
	inputDir := ".."
	got, err := DirectoryChecksum(inputDir)
	require.NoError(t, err)

	fmt.Println(got)
}

func TestLink(t *testing.T) {
	//foo := testutil.ProjectPath("tmp", "foo")
	foo := filepath.FromSlash("../../tmp/foo")
	fmt.Println(foo)
	info, err := os.Lstat(foo)
	require.NoError(t, err)
	rl, err := os.Readlink(foo)
	require.NoError(t, err)
	fmt.Println("rl", rl)
	fmt.Println(info.Mode()&os.ModeSymlink != 0)
	fmt.Println(info.Mode() & os.ModeType)
	fmt.Println(info.Mode().String())
	got, err := filepath.EvalSymlinks(foo)
	require.NoError(t, err)
	fmt.Println(got)
}

func TestLinkWalk(t *testing.T) {
	inputDir := filepath.FromSlash("../../tmp")
	err := filepath.Walk(filepath.FromSlash(inputDir), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		linfo, err := os.Lstat(path)
		if err != nil {
			return err
		}
		// if it's a symlink, just add the target path to the hash
		if linfo.Mode()&os.ModeSymlink != 0 {
			linkPath, err := os.Readlink(path)
			if err != nil {
				return err
			}

			fmt.Println(path, "->", linkPath)
		}
		return nil
	})
	require.NoError(t, err)
}
