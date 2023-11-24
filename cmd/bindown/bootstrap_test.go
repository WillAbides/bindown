package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v4/internal/testutil"
)

func Test_bootstrapCmd(t *testing.T) {
	output := filepath.Join(t.TempDir(), "foo", "bootstrap.sh")
	runner := newCmdRunner(t)

	server := testutil.ServeFile(
		t,
		testdataPath("build-bootstrapper/checksums.txt"),
		"/WillAbides/bindown/releases/download/v4.8.0/checksums.txt",
		"",
	)
	want, err := os.ReadFile(testdataPath("build-bootstrapper/bootstrap-bindown.sh"))
	require.NoError(t, err)
	result := runner.run("bootstrap", "--output", output, "--tag", "4.8.0", "--base-url", server.URL)
	result.assertState(resultState{})
	got, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Equal(t, string(want), string(got))
}
