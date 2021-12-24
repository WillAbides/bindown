package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v3/internal/testutil"
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
		err := os.Mkdir(src, 0o750)
		require.NoError(t, err)
		err = CopyFile(src, dst, nil)
		require.Error(t, err)
	})

	t.Run("copies", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		content := []byte("foo")
		require.NoError(t, os.WriteFile(src, content, 0o600))
		err := CopyFile(src, dst, nil)
		require.NoError(t, err)

		got, err := os.ReadFile(dst)
		require.NoError(t, err)
		require.Equal(t, content, got)
	})

	t.Run("dst directory doesn't exist", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "dst", "file2")
		content := []byte("foo")
		require.NoError(t, os.WriteFile(src, content, 0o600))
		err := CopyFile(src, dst, nil)
		require.Error(t, err)
	})

	t.Run("overwrite", func(t *testing.T) {
		dir := testutil.TmpDir(t)
		src := filepath.Join(dir, "file1")
		dst := filepath.Join(dir, "file2")
		content := []byte("foo")
		require.NoError(t, os.WriteFile(src, content, 0o600))
		require.NoError(t, os.WriteFile(dst, []byte("bar"), 0o600))
		err := CopyFile(src, dst, nil)
		require.NoError(t, err)
		got, err := os.ReadFile(dst)
		require.NoError(t, err)
		require.Equal(t, content, got)
	})
}
