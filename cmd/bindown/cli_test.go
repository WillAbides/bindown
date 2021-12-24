package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v3"
	"github.com/willabides/bindown/v3/cmd/bindown/mocks"
)

type runCmdResult struct {
	stdOut  bytes.Buffer
	stdErr  bytes.Buffer
	exited  bool
	exitVal int
}

func (r *runCmdResult) assertStdOut(t *testing.T, want string) {
	t.Helper()
	require.Equal(t, want, strings.TrimSpace(r.stdOut.String()))
}

func (r *runCmdResult) assertStdErr(t *testing.T, want string) {
	t.Helper()
	require.Equal(t, want, strings.TrimSpace(r.stdErr.String()))
}

func (r *runCmdResult) assertState(t *testing.T, wantStdout, wantStderr string, wantExited bool, wantExitVal int) {
	t.Helper()
	r.assertStdOut(t, wantStdout)
	r.assertStdErr(t, wantStderr)
	require.Equal(t, wantExited, r.exited)
	require.Equal(t, wantExitVal, r.exitVal)
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

type cmdRunner func(commandLine ...string) runCmdResult

func setupMocks(t *testing.T) (cmdRunner, *mocks.MockConfigLoader, *mocks.MockConfigFile) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})
	mockConfigLoader := mocks.NewMockConfigLoader(ctrl)
	mockConfigFile := mocks.NewMockConfigFile(ctrl)

	runner := func(commandLine ...string) runCmdResult {
		oldCfgLoader := configLoader
		t.Cleanup(func() {
			configLoader = oldCfgLoader
		})
		configLoader = mockConfigLoader
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

	return runner, mockConfigLoader, mockConfigFile
}

func wantStderr(msg string) string {
	return fmt.Sprintf("cmdname: error: %s", msg)
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	pwd, err := os.Getwd()
	require.NoError(t, err)
	return pwd
}

func wdPath(t *testing.T, pth string) string {
	wd := mustGetwd(t)
	return filepath.Join(wd, filepath.FromSlash(pth))
}

func testEnvVar(t *testing.T, name, value string) {
	t.Helper()
	existing, ok := os.LookupEnv(name)
	if ok {
		t.Cleanup(func() {
			require.NoError(t, os.Setenv(name, existing))
		})
	} else {
		t.Cleanup(func() {
			require.NoError(t, os.Unsetenv(name))
		})
	}
	require.NoError(t, os.Setenv(name, value))
}

func TestFormat(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		runner, mockConfigLoader, mockConfigFile := setupMocks(t)
		mockConfigLoader.EXPECT().Load(filepath.FromSlash("/omg"), true).Return(mockConfigFile, nil)
		mockConfigFile.EXPECT().Write(false)
		result := runner("format", "--configfile", "/omg")
		result.assertState(t, "", "", false, 0)
	})

	t.Run("with config from environment", func(t *testing.T) {
		runner, mockConfigLoader, mockConfigFile := setupMocks(t)
		mockConfigLoader.EXPECT().Load(wdPath(t, "omg"), true).Return(mockConfigFile, nil)
		mockConfigFile.EXPECT().Write(false)
		testEnvVar(t, "BINDOWN_CONFIG_FILE", "omg")
		result := runner("format")
		result.assertState(t, "", "", false, 0)
	})

	t.Run("error loading config", func(t *testing.T) {
		runner, mockConfigLoader, _ := setupMocks(t)
		mockConfigLoader.EXPECT().Load(wdPath(t, "bindown.yml"), true).Return(nil, assert.AnError)
		result := runner("format")
		result.assertState(t, "", wantStderr(assert.AnError.Error()), true, 1)
	})

	t.Run("json output", func(t *testing.T) {
		runner, mockConfigLoader, mockConfigFile := setupMocks(t)
		mockConfigLoader.EXPECT().Load(wdPath(t, "bindown.yml"), true).Return(mockConfigFile, nil)
		mockConfigFile.EXPECT().Write(true)
		result := runner("format", "--json")
		result.assertState(t, "", "", false, 0)
	})

	t.Run("write error", func(t *testing.T) {
		runner, mockConfigLoader, mockConfigFile := setupMocks(t)
		mockConfigLoader.EXPECT().Load(wdPath(t, "bindown.yml"), true).Return(mockConfigFile, nil)
		mockConfigFile.EXPECT().Write(false).Return(assert.AnError)
		result := runner("format")
		result.assertState(t, "", wantStderr(assert.AnError.Error()), true, 1)
	})
}

func TestAddChecksums(t *testing.T) {
	runner, mockConfigLoader, mockConfigFile := setupMocks(t)
	mockConfigLoader.EXPECT().Load(wdPath(t, "bindown.yml"), true).Return(mockConfigFile, nil)
	mockConfigFile.EXPECT().AddChecksums([]string{"foo"}, nil)
	mockConfigFile.EXPECT().Write(true)
	result := runner("add-checksums", "--dependency", "foo", "--json")
	result.assertState(t, "", "", false, 0)
}

func TestValidate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		runner, mockConfigLoader, mockConfigFile := setupMocks(t)
		mockConfigLoader.EXPECT().Load(wdPath(t, "bindown.yml"), false).Return(mockConfigFile, nil)
		mockConfigFile.EXPECT().Validate([]string{"foo"}, nil)
		result := runner("validate", "foo")
		result.assertState(t, "", "", false, 0)
	})

	t.Run("error loading config", func(t *testing.T) {
		runner, mockConfigLoader, _ := setupMocks(t)
		mockConfigLoader.EXPECT().Load(wdPath(t, "bindown.yml"), false).Return(nil, assert.AnError)
		result := runner("validate", "foo")
		result.assertState(t, "", wantStderr(assert.AnError.Error()), true, 1)
	})

	t.Run("multiple systems", func(t *testing.T) {
		runner, mockConfigLoader, mockConfigFile := setupMocks(t)
		mockConfigLoader.EXPECT().Load(wdPath(t, "bindown.yml"), false).Return(mockConfigFile, nil)
		mockConfigFile.EXPECT().Validate([]string{"foo"}, []bindown.SystemInfo{
			{OS: "foo", Arch: "bar"},
			{OS: "baz", Arch: "qux"},
		})
		result := runner("validate", "foo", "--system", "foo/bar", "--system=baz/qux")
		result.assertState(t, "", "", false, 0)
	})
}

func TestExtract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		runner, mockConfigLoader, mockConfigFile := setupMocks(t)
		mockConfigLoader.EXPECT().Load(wdPath(t, "bindown.yml"), false).Return(mockConfigFile, nil)
		mockConfigFile.EXPECT().ExtractDependency("foo", bindown.SystemInfo{
			Arch: runtime.GOARCH,
			OS:   runtime.GOOS,
		}, &bindown.ConfigExtractDependencyOpts{}).Return("wherever", nil)
		result := runner("extract", "foo")
		result.assertState(t, "extracted foo to wherever", "", false, 0)
	})
}

func TestDownload(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		runner, mockConfigLoader, mockConfigFile := setupMocks(t)
		mockConfigLoader.EXPECT().Load(wdPath(t, "bindown.yml"), false).Return(mockConfigFile, nil)
		mockConfigFile.EXPECT().DownloadDependency("foo", bindown.SystemInfo{
			Arch: runtime.GOARCH,
			OS:   runtime.GOOS,
		}, &bindown.ConfigDownloadDependencyOpts{}).Return("wherever", nil)
		result := runner("download", "foo")
		result.assertState(t, "downloaded foo to wherever", "", false, 0)
	})
}

func TestInstall(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		runner, mockConfigLoader, mockConfigFile := setupMocks(t)
		mockConfigLoader.EXPECT().Load(wdPath(t, "bindown.yml"), false).Return(mockConfigFile, nil)
		mockConfigFile.EXPECT().InstallDependency("foo", bindown.SystemInfo{
			Arch: runtime.GOARCH,
			OS:   runtime.GOOS,
		}, &bindown.ConfigInstallDependencyOpts{}).Return("wherever", nil)
		result := runner("install", "foo")
		result.assertState(t, "installed foo to wherever", "", false, 0)
	})
}

func createConfigFile(t *testing.T, sourceFile string) string {
	t.Helper()
	sourceFile = filepath.Join("..", "..", "testdata", "configs", sourceFile)
	dir := t.TempDir()
	dest := filepath.Join(dir, "bindown.config")
	copyFile(t, sourceFile, dest)
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

func TestVersion(t *testing.T) {
	result := runCmd("version")
	result.assertStdOut(t, "cmdname: version unknown")
	result.assertStdErr(t, "")
}

func copyFile(t *testing.T, sourceFile, destFile string) {
	t.Helper()
	source, err := os.Open(sourceFile)
	require.NoError(t, err)
	dest, err := os.Create(destFile)
	require.NoError(t, err)
	_, err = io.Copy(dest, source)
	require.NoError(t, err)
	require.NoError(t, source.Close())
	require.NoError(t, dest.Close())
}
