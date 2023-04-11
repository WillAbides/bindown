package bindown

import (
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
		require.NoError(t, copyFile(filepath.Join("testdata", "downloadables", "foo.tar.gz"), file))
		got, err := fileExistsWithChecksum(file, fooChecksum)
		require.NoError(t, err)
		require.True(t, got)
	})

	t.Run("wrong checksum", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "myfile")
		checksum := "0000000000000000000000000000000000000000000000000000000000000000"
		require.NoError(t, copyFile(filepath.Join("testdata", "downloadables", "foo.tar.gz"), file))
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

func Test_directoryChecksum(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		got, err := directoryChecksum(filepath.Join("testdata", "directoryChecksum"))
		require.NoError(t, err)
		// This should only change when the contents of testdata/directoryChecksum change.
		require.Equal(t, "0eb72a7b3c1e286a", got)
	})
}

func Test_copyFile(t *testing.T) {
	t.Run("doesn't exist", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		err := copyFile(src, dst)
		require.Error(t, err)
	})

	t.Run("directory", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		err := os.Mkdir(src, 0o750)
		require.NoError(t, err)
		err = copyFile(src, dst)
		require.Error(t, err)
	})

	t.Run("copies", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		content := []byte("foo")
		require.NoError(t, os.WriteFile(src, content, 0o600))
		err := copyFile(src, dst)
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
		err := copyFile(src, dst)
		require.Error(t, err)
	})

	t.Run("overwrite", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		content := []byte("foo")
		require.NoError(t, os.WriteFile(src, content, 0o600))
		require.NoError(t, os.WriteFile(dst, []byte("bar"), 0o600))
		err := copyFile(src, dst)
		require.NoError(t, err)
		got, err := os.ReadFile(dst)
		require.NoError(t, err)
		require.Equal(t, content, got)
	})
}
