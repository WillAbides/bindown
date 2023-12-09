package main

import (
	"path/filepath"
	"testing"

	"github.com/willabides/bindown/v4/internal/testutil"
)

func Test_bootstrapCmd(t *testing.T) {
	targetDir := filepath.Join(t.TempDir(), "target")
	output := filepath.Join(targetDir, "bootstrap.sh")
	runner := newCmdRunner(t)

	server := testutil.ServeFile(
		t,
		"testdata/bootstrap/checksums.txt",
		"/WillAbides/bindown/releases/download/v4.8.0/checksums.txt",
		"",
	)
	result := runner.run("bootstrap", "--output", output, "--tag", "4.8.0", "--base-url", server.URL)
	result.assertState(resultState{})

	testutil.CheckGoldenDir(t, targetDir, filepath.FromSlash("testdata/golden/bootstrap"))
}
