package bindown

import (
	"crypto/sha256"
	"hash/fnv"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecuteTemplate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		vars := map[string]string{
			"version": "1.2.3",
		}
		tmpl := `whatever-{{.version}}/mybin-{{.os}}-{{.arch}}`
		got, err := executeTemplate(tmpl, "Linux", "arm", vars)
		require.NoError(t, err)
		require.Equal(t, "whatever-1.2.3/mybin-Linux-arm", got)
	})

	t.Run("nil vars", func(t *testing.T) {
		tmpl := `whatever/mybin-{{.os}}-{{.arch}}`
		got, err := executeTemplate(tmpl, "Linux", "arm", nil)
		require.NoError(t, err)
		require.Equal(t, "whatever/mybin-Linux-arm", got)
	})
}

func Test_fileExistsWithChecksum(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "myfile")
		require.NoError(t, copyFile(filepath.Join("testdata", "downloadables", "foo.tar.gz"), file, nil))
		got, err := fileExistsWithChecksum(file, fooChecksum)
		require.NoError(t, err)
		require.True(t, got)
	})

	t.Run("wrong checksum", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "myfile")
		checksum := "0000000000000000000000000000000000000000000000000000000000000000"
		require.NoError(t, copyFile(filepath.Join("testdata", "downloadables", "foo.tar.gz"), file, nil))
		got, err := fileExistsWithChecksum(file, checksum)
		require.NoError(t, err)
		require.False(t, got)
	})

	t.Run("doesn't exist", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "myfile")
		got, err := fileExistsWithChecksum(file, fooChecksum)
		require.NoError(t, err)
		require.False(t, got)
	})
}

func Test_fileChecksum(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "myfile")
		require.NoError(t, copyFile(filepath.Join("testdata", "downloadables", "foo.tar.gz"), file, nil))
		got, err := fileChecksum(file)
		require.NoError(t, err)
		require.Equal(t, fooChecksum, got)
	})

	t.Run("doesn't exist", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "myfile")
		got, err := fileChecksum(file)
		require.Error(t, err)
		require.Empty(t, got)
	})
}

func Test_hexHash(t *testing.T) {
	require.Equal(t, "dcb27518fed9d577", hexHash(fnv.New64a(), []byte("foo")))
	require.Equal(t, "85944171f73967e8", hexHash(fnv.New64a(), []byte("foo"), []byte("bar")))
	content, err := os.ReadFile(filepath.Join("testdata", "downloadables", "foo.tar.gz"))
	require.NoError(t, err)
	require.Equal(t, fooChecksum, hexHash(sha256.New(), content))
}

func Test_copyFile(t *testing.T) {
	t.Run("doesn't exist", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		err := copyFile(src, dst, nil)
		require.Error(t, err)
	})

	t.Run("directory", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		err := os.Mkdir(src, 0o750)
		require.NoError(t, err)
		err = copyFile(src, dst, nil)
		require.Error(t, err)
	})

	t.Run("copies", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		content := []byte("foo")
		require.NoError(t, os.WriteFile(src, content, 0o600))
		err := copyFile(src, dst, nil)
		require.NoError(t, err)

		got, err := os.ReadFile(dst)
		require.NoError(t, err)
		require.Equal(t, content, got)
	})

	t.Run("dst directory doesn't exist", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "dst", "file2")
		content := []byte("foo")
		require.NoError(t, os.WriteFile(src, content, 0o600))
		err := copyFile(src, dst, nil)
		require.Error(t, err)
	})

	t.Run("overwrite", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		content := []byte("foo")
		require.NoError(t, os.WriteFile(src, content, 0o600))
		require.NoError(t, os.WriteFile(dst, []byte("bar"), 0o600))
		err := copyFile(src, dst, nil)
		require.NoError(t, err)
		got, err := os.ReadFile(dst)
		require.NoError(t, err)
		require.Equal(t, content, got)
	})
}
