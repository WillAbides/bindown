package testutil

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func updateGoldenDir(t *testing.T, resultDir, goldenDir string) {
	t.Helper()
	if os.Getenv("UPDATE_GOLDEN") == "" {
		return
	}
	require.NoError(t, os.RemoveAll(goldenDir))
	err := filepath.WalkDir(resultDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		fmt.Println(path)
		relName := mustRel(t, resultDir, path)
		return copyFile(path, filepath.Join(goldenDir, relName))
	})
	require.NoError(t, err)
}

func copyFile(src, dst string) (errOut error) {
	err := os.MkdirAll(filepath.Dir(dst), 0o777)
	if err != nil {
		return err
	}
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		e := dstFile.Close()
		if errOut == nil {
			errOut = e
		}
	}()
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		e := srcFile.Close()
		if errOut == nil {
			errOut = e
		}
	}()
	_, err = io.Copy(dstFile, srcFile)
	return err
}

func CheckGoldenDir(t *testing.T, resultDir, goldenDir string) {
	t.Helper()
	golden := true
	t.Cleanup(func() {
		t.Helper()
		if !golden {
			t.Log("To regenerate golden files run `UPDATE_GOLDEN=1 script/test`")
		}
	})
	updateGoldenDir(t, resultDir, goldenDir)
	checked := map[string]bool{}
	_, err := os.Stat(goldenDir)
	if err == nil {
		assert.NoError(t, filepath.WalkDir(goldenDir, func(wantPath string, info fs.DirEntry, err error) error {
			relPath := mustRel(t, goldenDir, wantPath)
			if err != nil || info.IsDir() {
				return err
			}
			if !assertEqualFiles(t, wantPath, filepath.Join(resultDir, relPath)) {
				golden = false
			}
			checked[relPath] = true
			return nil
		}))
	}
	assert.NoError(t, filepath.Walk(resultDir, func(resultPath string, info fs.FileInfo, err error) error {
		relPath := mustRel(t, resultDir, resultPath)
		if err != nil || info.IsDir() || checked[relPath] {
			return err
		}
		golden = false
		return fmt.Errorf("found unexpected file:\n%s", relPath)
	}))
}

func mustRel(t *testing.T, base, target string) string {
	t.Helper()
	rel, err := filepath.Rel(base, target)
	require.NoError(t, err)
	return rel
}

func assertEqualFiles(t *testing.T, want, got string) bool {
	t.Helper()
	wantBytes, err := os.ReadFile(want)
	if !assert.NoError(t, err) {
		return false
	}
	wantBytes = bytes.ReplaceAll(wantBytes, []byte("\r\n"), []byte("\n"))
	gotBytes, err := os.ReadFile(got)
	if !assert.NoError(t, err) {
		return false
	}
	gotBytes = bytes.ReplaceAll(gotBytes, []byte("\r\n"), []byte("\n"))
	return assert.Equal(t, string(wantBytes), string(gotBytes))
}
