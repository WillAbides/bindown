package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	// don't test on Windows
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}
	got, err := build("v3.15.5", filepath.FromSlash("../../"))
	require.NoError(t, err)
	want, err := os.ReadFile(filepath.FromSlash("testdata/want.txt"))
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(string(want), got))
}
