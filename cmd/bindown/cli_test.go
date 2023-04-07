package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v3"
)

func Test_fmtCmd(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`{"systems": [ "darwin/amd64", "linux/386" ]}`)
		result := runner.run("format")
		result.assertState(resultState{})
		runner.assertConfigYaml(`systems:
- darwin/amd64
- linux/386
`)
	})

	t.Run("error loading config", func(t *testing.T) {
		runner := newCmdRunner(t)
		// invalid -- missing final "}"
		runner.writeConfigYaml(`{"systems": [ "darwin/amd64", "linux/386" ]`)
		result := runner.run("format")
		result.assertState(resultState{
			stderr: "cmd: error: config is not valid yaml (or json)",
			exit:   1,
		})
	})

	t.Run("json output", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`systems:
    - darwin/amd64
    - linux/386
`)
		result := runner.run("format", "--json")
		result.assertState(resultState{})
		runner.assertConfigYaml(`{
  "systems": [
    "darwin/amd64",
    "linux/386"
  ]
}`)
	})
}

func Test_initCmd(t *testing.T) {
	t.Run("default file", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.cache = ""
		runner.configFile = ""
		testInDir(t, runner.tmpDir)
		result := runner.run("init")
		result.assertState(resultState{})
		content, err := os.ReadFile(".bindown.yaml")
		require.NoError(t, err)
		require.Equal(t, "{}\n", string(content))
	})

	t.Run("default file already exists", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.cache = ""
		runner.configFile = ""
		testInDir(t, runner.tmpDir)
		err := os.WriteFile(".bindown.yaml", []byte("foo"), 0o600)
		require.NoError(t, err)
		result := runner.run("init")
		result.assertState(resultState{
			stderr: "cmd: error: .bindown.yaml already exists",
			exit:   1,
		})
	})

	t.Run("default file when bindown.yml already exixts", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.cache = ""
		runner.configFile = ""
		testInDir(t, runner.tmpDir)
		err := os.WriteFile("bindown.yml", []byte("foo"), 0o600)
		require.NoError(t, err)
		result := runner.run("init")
		result.assertState(resultState{
			stderr: "cmd: error: bindown.yml already exists",
			exit:   1,
		})
	})

	t.Run("custom file", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.cache = ""
		runner.configFile = ""
		testInDir(t, runner.tmpDir)
		result := runner.run("init", "--configfile", "foo.yaml")
		result.assertState(resultState{})
		content, err := os.ReadFile("foo.yaml")
		require.NoError(t, err)
		require.Equal(t, "{}\n", string(content))
	})

	t.Run("custom file in sub directory", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.cache = ""
		runner.configFile = ""
		testInDir(t, runner.tmpDir)
		err := os.Mkdir("foo", 0o700)
		require.NoError(t, err)
		result := runner.run("init", "--configfile", "foo/bar.yaml")
		result.assertState(resultState{})
		content, err := os.ReadFile("foo/bar.yaml")
		require.NoError(t, err)
		require.Equal(t, "{}\n", string(content))
	})

	t.Run("custom file in sub directory that does not exist", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.cache = ""
		runner.configFile = ""
		testInDir(t, runner.tmpDir)
		result := runner.run("init", "--configfile", "foo/bar.yaml")
		result.assertState(resultState{
			exit:   1,
			stderr: `cmd: error: open .+: no such file or directory`,
		})
	})
}

func Test_extractCmd(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		runner := newCmdRunner(t)
		servePath := filepath.FromSlash("../../testdata/downloadables/fooinroot.tar.gz")
		ts := serveFile(t, servePath, "/foo/fooinroot.tar.gz", "")
		depURL := ts.URL + "/foo/fooinroot.tar.gz"
		runner.writeConfig(&bindown.Config{
			URLChecksums: map[string]string{
				depURL: "27dcce60d1ed72920a84dd4bc01e0bbd013e5a841660e9ee2e964e53fb83c0b3",
			},
			Dependencies: map[string]*bindown.Dependency{
				"foo": {
					URL: &depURL,
				},
			},
		})
		result := runner.run("extract", "foo")
		require.Equal(t, 0, result.exitVal)
		stdout := result.stdOut.String()
		wantPrefix := "extracted foo to "
		require.True(t, strings.HasPrefix(stdout, wantPrefix))
		extractDir := strings.TrimSpace(strings.TrimPrefix(stdout, wantPrefix))
		wantFile := filepath.Join(extractDir, "foo")
		require.FileExists(t, wantFile)
	})
}

func Test_downloadCmd(t *testing.T) {
	servePath := filepath.FromSlash("../../testdata/downloadables/fooinroot.tar.gz")
	successServer := serveFile(t, servePath, "/foo/fooinroot.tar.gz", "")
	depURL := successServer.URL + "/foo/fooinroot.tar.gz"

	errServer := serveErr(t, 400)
	errURL := errServer.URL + "/foo/fooinroot.tar.gz"

	assertDownloadSuccess := func(t *testing.T, result *runCmdResult) {
		t.Helper()
		prefix := "downloaded foo to "
		result.assertState(resultState{
			stdout: prefix,
		})
		stdout := result.stdOut.String()
		if !assert.True(t, strings.HasPrefix(stdout, prefix)) {
			return
		}
		dlPath := strings.TrimSpace(strings.TrimPrefix(stdout, prefix))
		assert.FileExists(t, dlPath)
	}

	t.Run("success", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfig(&bindown.Config{
			URLChecksums: map[string]string{
				depURL: "27dcce60d1ed72920a84dd4bc01e0bbd013e5a841660e9ee2e964e53fb83c0b3",
			},
			Dependencies: map[string]*bindown.Dependency{
				"foo": {
					URL: &depURL,
				},
			},
		})
		result := runner.run("download", "foo")
		assertDownloadSuccess(t, result)
	})

	t.Run("no url", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfig(&bindown.Config{
			Dependencies: map[string]*bindown.Dependency{
				"foo": {},
			},
		})
		result := runner.run("download", "foo")
		result.assertState(resultState{
			stderr: `cmd: error: no URL configured`,
			exit:   1,
		})
	})

	t.Run("missing var", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfig(&bindown.Config{
			Dependencies: map[string]*bindown.Dependency{
				"foo": {
					URL: ptr("https://localhost/{{ .MISSING_VAR }}"),
				},
			},
		})
		result := runner.run("download", "foo")
		result.assertState(resultState{
			stderr: `cmd: error: error applying template`,
			exit:   1,
		})
	})

	t.Run("missing checksum", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfig(&bindown.Config{
			Dependencies: map[string]*bindown.Dependency{
				"foo": {
					URL: &depURL,
				},
			},
		})
		result := runner.run("download", "foo")
		result.assertState(resultState{
			stderr: `cmd: error: no checksum for the url`,
			exit:   1,
		})
	})

	t.Run("--allow-missing-checksum", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfig(&bindown.Config{
			Dependencies: map[string]*bindown.Dependency{
				"foo": {
					URL: &depURL,
				},
			},
		})
		result := runner.run("download", "foo", "--allow-missing-checksum")
		assertDownloadSuccess(t, result)
	})

	t.Run("--allow-missing-checksum with dl error", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfig(&bindown.Config{
			Dependencies: map[string]*bindown.Dependency{
				"foo": {
					URL: &errURL,
				},
			},
		})
		result := runner.run("download", "foo", "--allow-missing-checksum")
		result.assertState(resultState{
			stderr: `cmd: error: failed downloading`,
			exit:   1,
		})
	})

	t.Run("already exists", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfig(&bindown.Config{
			URLChecksums: map[string]string{
				depURL: "27dcce60d1ed72920a84dd4bc01e0bbd013e5a841660e9ee2e964e53fb83c0b3",
			},
			Dependencies: map[string]*bindown.Dependency{
				"foo": {
					URL: &depURL,
				},
			},
		})
		// download to put it in the cache
		result := runner.run("download", "foo")
		assertDownloadSuccess(t, result)
		// download again
		result = runner.run("download", "foo")
		assertDownloadSuccess(t, result)
	})

	t.Run("already exists with --allow-missing-checksum", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfig(&bindown.Config{
			Dependencies: map[string]*bindown.Dependency{
				"foo": {
					URL: &depURL,
				},
			},
		})
		// download to put it in the cache
		result := runner.run("download", "foo", "--allow-missing-checksum")
		assertDownloadSuccess(t, result)
		// download again
		result = runner.run("download", "foo", "--allow-missing-checksum")
		assertDownloadSuccess(t, result)
	})
}

func Test_installCmd(t *testing.T) {
	t.Run("raw file", func(t *testing.T) {
		runner := newCmdRunner(t)
		servePath := filepath.FromSlash("../../testdata/downloadables/rawfile/foo")
		ts := serveFile(t, servePath, "/foo/foo", "")
		depURL := ts.URL + "/foo/foo"
		runner.writeConfig(&bindown.Config{
			URLChecksums: map[string]string{
				depURL: "f044ff8b6007c74bcc1b5a5c92776e5d49d6014f5ff2d551fab115c17f48ac41",
			},
			Dependencies: map[string]*bindown.Dependency{
				"foo": {
					URL: &depURL,
				},
			},
		})
		result := runner.run("install", "foo")
		require.Equal(t, 0, result.exitVal)
		wantBin := filepath.Join(runner.tmpDir, "bin", "foo")
		require.FileExists(t, wantBin)
		stat, err := os.Stat(wantBin)
		require.NoError(t, err)
		require.EqualValues(t, os.FileMode(0o750), stat.Mode().Perm()&0o750)
	})

	t.Run("bin in root", func(t *testing.T) {
		runner := newCmdRunner(t)
		servePath := filepath.FromSlash("../../testdata/downloadables/fooinroot.tar.gz")
		ts := serveFile(t, servePath, "/foo/fooinroot.tar.gz", "")
		depURL := ts.URL + "/foo/fooinroot.tar.gz"
		runner.writeConfig(&bindown.Config{
			URLChecksums: map[string]string{
				depURL: "27dcce60d1ed72920a84dd4bc01e0bbd013e5a841660e9ee2e964e53fb83c0b3",
			},
			Dependencies: map[string]*bindown.Dependency{
				"foo": {
					URL: &depURL,
				},
			},
		})
		result := runner.run("install", "foo")
		require.Equal(t, 0, result.exitVal)
		wantBin := filepath.Join(runner.tmpDir, "bin", "foo")
		require.FileExists(t, wantBin)
		stat, err := os.Stat(wantBin)
		require.NoError(t, err)
		require.EqualValues(t, os.FileMode(0o750), stat.Mode().Perm()&0o750)
	})

	t.Run("wrong checksum", func(t *testing.T) {
		runner := newCmdRunner(t)
		servePath := filepath.FromSlash("../../testdata/downloadables/fooinroot.tar.gz")
		ts := serveFile(t, servePath, "/foo/fooinroot.tar.gz", "")
		depURL := ts.URL + "/foo/fooinroot.tar.gz"
		runner.writeConfig(&bindown.Config{
			URLChecksums: map[string]string{
				depURL: "0000000000000000000000000000000000000000000000000000000000000000",
			},
			Dependencies: map[string]*bindown.Dependency{
				"foo": {URL: &depURL},
			},
		})
		result := runner.run("install", "foo")
		fmt.Println(result.stdErr.String())
		require.Equal(t, 1, result.exitVal)
		require.True(t, strings.HasPrefix(result.stdErr.String(), `cmd: error: checksum mismatch in downloaded file`))
		require.NoFileExists(t, filepath.Join(runner.tmpDir, "bin", "foo"))
	})
}
