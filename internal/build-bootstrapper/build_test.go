package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	got, err := build("v3.15.5", filepath.FromSlash("../../"))
	require.NoError(t, err)
	want, err := os.ReadFile(filepath.FromSlash("testdata/want.txt"))
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(string(want), got))
}
