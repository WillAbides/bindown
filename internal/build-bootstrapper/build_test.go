package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	// This doesn't work on windows, and for now I don't care because it is only used in CI
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	origConfig, err := os.ReadFile(filepath.FromSlash("../../bindown.yml"))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(
			t,
			os.WriteFile(filepath.FromSlash("../../bindown.yml"), origConfig, 0o644),
		)
	})
	got, err := build("v4.0.0", filepath.FromSlash("../../"))
	require.NoError(t, err)
	want, err := os.ReadFile(filepath.FromSlash("testdata/want.txt"))
	require.NoError(t, err)
	require.Equal(t, string(want), got)
}
