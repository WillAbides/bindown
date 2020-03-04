package bindown

import (
	"crypto/sha256"
	"hash/fnv"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_fileExistsWithChecksum(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		file := filepath.Join(tmpDir(t), "myfile")
		mustCopyFile(t, downloadablesPath("foo.tar.gz"), file)
		got, err := fileExistsWithChecksum(file, fooChecksum)
		require.NoError(t, err)
		require.True(t, got)
	})

	t.Run("wrong checksum", func(t *testing.T) {
		file := filepath.Join(tmpDir(t), "myfile")
		checksum := "0000000000000000000000000000000000000000000000000000000000000000"
		mustCopyFile(t, downloadablesPath("foo.tar.gz"), file)
		got, err := fileExistsWithChecksum(file, checksum)
		require.NoError(t, err)
		require.False(t, got)
	})

	t.Run("doesn't exist", func(t *testing.T) {
		file := filepath.Join(tmpDir(t), "myfile")
		got, err := fileExistsWithChecksum(file, fooChecksum)
		require.NoError(t, err)
		require.False(t, got)
	})
}

func Test_fileChecksum(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		file := filepath.Join(tmpDir(t), "myfile")
		mustCopyFile(t, downloadablesPath("foo.tar.gz"), file)
		got, err := fileChecksum(file)
		require.NoError(t, err)
		require.Equal(t, fooChecksum, got)
	})

	t.Run("doesn't exist", func(t *testing.T) {
		file := filepath.Join(tmpDir(t), "myfile")
		got, err := fileChecksum(file)
		require.Error(t, err)
		require.Empty(t, got)
	})
}

func Test_hexHash(t *testing.T) {
	got, err := hexHash(fnv.New64a(), []byte("foo"))
	require.NoError(t, err)
	require.Equal(t, "dcb27518fed9d577", got)
	got, err = hexHash(fnv.New64a(), []byte("foo"), []byte("bar"))
	require.NoError(t, err)
	require.Equal(t, "85944171f73967e8", got)
	content := mustReadFile(t, downloadablesPath("foo.tar.gz"))
	got, err = hexHash(sha256.New(), content)
	require.NoError(t, err)
	require.Equal(t, fooChecksum, got)
}

func Test_copyFile(t *testing.T) {
	t.Run("doesn't exist", func(t *testing.T) {
		dir := tmpDir(t)
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		err := copyFile(src, dst)
		require.Error(t, err)
	})

	t.Run("directory", func(t *testing.T) {
		dir := tmpDir(t)
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		err := os.Mkdir(src, 0750)
		require.NoError(t, err)
		err = copyFile(src, dst)
		require.Error(t, err)
	})

	t.Run("copies", func(t *testing.T) {
		dir := tmpDir(t)
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		content := []byte("foo")
		mustWriteFile(t, src, content)
		err := copyFile(src, dst)
		require.NoError(t, err)
		got := mustReadFile(t, dst)
		assert.Equal(t, content, got)
	})

	t.Run("dst directory doesn't exist", func(t *testing.T) {
		dir := tmpDir(t)
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "dst", "file2")
		content := []byte("foo")
		mustWriteFile(t, src, content)
		err := copyFile(src, dst)
		require.Error(t, err)
	})

	t.Run("overwrite", func(t *testing.T) {
		dir := tmpDir(t)
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		content := []byte("foo")
		mustWriteFile(t, src, content)
		mustWriteFile(t, dst, []byte("bar"))
		err := copyFile(src, dst)
		require.NoError(t, err)
		got := mustReadFile(t, dst)
		require.Equal(t, content, got)
	})
}
