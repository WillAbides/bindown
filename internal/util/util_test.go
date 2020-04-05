package util

import (
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
