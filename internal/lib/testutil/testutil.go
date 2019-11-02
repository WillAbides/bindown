package testutil

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// ProjectPath exchanges a path relative to the project root for an absolute path
func ProjectPath(path string) string {
	return filepath.Join(ProjectRoot(), path)
}

// ProjectRoot returns the absolute path of the project root
func ProjectRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), filepath.FromSlash("../../.."))
}

// TmpDir returns the path to a newly created tmp dir and a function for deleting that dir
func TmpDir(t *testing.T) (string, func()) {
	t.Helper()
	projectTmp := ProjectPath("tmp")
	err := os.MkdirAll(projectTmp, 0750)
	require.NoError(t, err)
	tmpdir, err := ioutil.TempDir(projectTmp, "")
	require.NoError(t, err)
	return tmpdir, func() {
		require.NoError(t, os.RemoveAll(tmpdir))
	}
}
