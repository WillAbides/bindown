package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v2/internal/testutil"
	"github.com/willabides/bindown/v2/internal/util"
)

type runCmdResult struct {
	stdOut  bytes.Buffer
	stdErr  bytes.Buffer
	exited  bool
	exitVal int
}

func (r runCmdResult) assertStdOut(t *testing.T, want string) {
	t.Helper()
	assert.Equal(t, want, strings.TrimSpace(r.stdOut.String()))
}

func (r runCmdResult) assertStdErr(t *testing.T, want string) {
	t.Helper()
	assert.Equal(t, want, strings.TrimSpace(r.stdErr.String()))
}

func (r runCmdResult) assertError(t *testing.T, msg string) {
	t.Helper()
	assert.NotZero(t, r.exitVal)
	wantPrefix := fmt.Sprintf("cmdname: error: %s", msg)
	stdErr := r.stdErr.String()
	hasPrefix := strings.HasPrefix(stdErr, wantPrefix)
	assert.True(t, hasPrefix, "stdErr should start with %q\ninstead stdErr was %q", wantPrefix, stdErr)
}

func runCmd(commandLine ...string) runCmdResult {
	result := runCmdResult{}
	Run(commandLine,
		kong.Name("cmdname"),
		kong.Writers(&result.stdOut, &result.stdErr),
		kong.Exit(func(i int) {
			result.exited = true
			result.exitVal = i
		}),
	)
	return result
}

func createConfigFile(t *testing.T, sourceFile string) string {
	t.Helper()
	sourceFile = testutil.ProjectPath("testdata", "configs", sourceFile)
	dir := testutil.TmpDir(t)
	dest := filepath.Join(dir, "bindown.config")
	err := util.CopyFile(sourceFile, dest, nil)
	require.NoError(t, err)
	return dest
}

func setConfigFileEnvVar(t *testing.T, file string) {
	t.Helper()
	err := os.Setenv("BINDOWN_CONFIG_FILE", file)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("BINDOWN_CONFIG_FILE"))
	})
}

func TestRun(t *testing.T) {
	t.Run("version", func(t *testing.T) {
		result := runCmd("version")
		result.assertStdOut(t, "cmdname: version unknown")
		result.assertStdErr(t, "")
	})

	t.Run("config format", func(t *testing.T) {
		setConfigFileEnvVar(t, createConfigFile(t, "ex1.yaml"))
		result := runCmd("config", "format")
		result.assertStdErr(t, "")
		result.assertStdOut(t, "")
		assert.Zero(t, result.exitVal)
	})

	t.Run("config format no config file", func(t *testing.T) {
		result := runCmd("config", "format")
		assert.NotZero(t, result.exitVal)
		result.assertStdOut(t, "")
		result.assertError(t, "could not load config file")
	})
}
