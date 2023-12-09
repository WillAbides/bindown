package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v4/internal/cache"
	"github.com/willabides/bindown/v4/internal/testutil"
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
		wantStderr := `no such file or directory`
		if runtime.GOOS == "windows" {
			wantStderr = `The system cannot find the path specified.`
		}
		result.assertState(resultState{
			exit:   1,
			stderr: wantStderr,
		})
	})
}

func Test_extractCmd(t *testing.T) {
	servePath := testdataPath("downloadables/fooinroot.tar.gz")
	successServer := testutil.ServeFile(t, servePath, "/foo/fooinroot.tar.gz", "")
	depURL := successServer.URL + "/foo/fooinroot.tar.gz"

	assertExtractSuccess := func(t *testing.T, result *runCmdResult) {
		t.Helper()
		prefix := "extracted foo to "
		result.assertState(resultState{
			stdout: prefix,
		})
		extractDir := result.getExtractDir()
		wantFile := filepath.Join(extractDir, "foo")
		require.FileExists(t, wantFile)
		// make sure there are no extra files
		dirFiles, err := os.ReadDir(extractDir)
		require.NoError(t, err)
		require.Len(t, dirFiles, 1)
	}

	t.Run("success", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  foo:
    url: %s
url_checksums:
  %s: 27dcce60d1ed72920a84dd4bc01e0bbd013e5a841660e9ee2e964e53fb83c0b3
`, depURL, depURL))
		result := runner.run("extract", "foo")
		assertExtractSuccess(t, result)
	})

	t.Run("invalid cache", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  foo:
    url: %s
url_checksums:
  %s: 27dcce60d1ed72920a84dd4bc01e0bbd013e5a841660e9ee2e964e53fb83c0b3
`, depURL, depURL))
		result := runner.run("extract", "foo")
		assertExtractSuccess(t, result)

		// corrupt the cache
		extractDir := result.getExtractDir()
		exCache := &cache.Cache{
			Root: filepath.Dir(extractDir),
		}
		exCacheKey := filepath.Base(extractDir)
		require.NoError(t, exCache.Evict(exCacheKey))
		_, unlock, err := exCache.Dir(exCacheKey, nil, func(d string) error {
			return os.WriteFile(filepath.Join(d, "foo"), []byte("foo"), 0o666)
		})
		require.NoError(t, err)
		require.NoError(t, unlock())

		result = runner.run("extract", "foo")
		assertExtractSuccess(t, result)
	})

	t.Run("empty cache", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  foo:
    url: %s
url_checksums:
  %s: 27dcce60d1ed72920a84dd4bc01e0bbd013e5a841660e9ee2e964e53fb83c0b3
`, depURL, depURL))
		result := runner.run("extract", "foo")
		assertExtractSuccess(t, result)
	})

	t.Run("valid cache", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  foo:
    url: %s
url_checksums:
  %s: 27dcce60d1ed72920a84dd4bc01e0bbd013e5a841660e9ee2e964e53fb83c0b3
`, depURL, depURL))
		result := runner.run("extract", "foo")
		assertExtractSuccess(t, result)
		result = runner.run("extract", "foo")
		assertExtractSuccess(t, result)
	})

	t.Run("does not overwrite invalid cache", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  foo:
    url: %s
url_checksums:
  %s: 27dcce60d1ed72920a84dd4bc01e0bbd013e5a841660e9ee2e964e53fb83c0b3
`, depURL, depURL))
		result := runner.run("extract", "foo")
		assertExtractSuccess(t, result)

		// corrupt the cache
		extractDir := result.getExtractDir()
		exCache := &cache.Cache{
			Root: filepath.Dir(extractDir),
		}
		exCacheKey := filepath.Base(extractDir)
		require.NoError(t, exCache.Evict(exCacheKey))
		_, unlock, err := exCache.Dir(exCacheKey, nil, func(d string) error {
			return os.WriteFile(filepath.Join(d, "foo"), []byte("foo"), 0o666)
		})
		require.NoError(t, err)
		require.NoError(t, unlock())

		result = runner.run("extract", "foo")
		result.assertState(resultState{
			stdout: "extracted foo to ",
		})
		// make sure the file was not overwritten
		got, err := os.ReadFile(filepath.Join(extractDir, "foo"))
		require.NoError(t, err)
		assert.Equal(t, "foo", string(got))
	})
}

func Test_downloadCmd(t *testing.T) {
	servePath := testdataPath("downloadables/fooinroot.tar.gz")
	successServer := testutil.ServeFile(t, servePath, "/foo/fooinroot.tar.gz", "")
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
		runner.writeConfigYaml(fmt.Sprintf(`
url_checksums:
  %s: 27dcce60d1ed72920a84dd4bc01e0bbd013e5a841660e9ee2e964e53fb83c0b3
dependencies:
  foo:
    url: %s
`, depURL, depURL))
		result := runner.run("download", "foo")
		assertDownloadSuccess(t, result)
	})

	t.Run("no url", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`
dependencies:
  foo: {}
`)
		result := runner.run("download", "foo")
		result.assertState(resultState{
			stderr: `cmd: error: dependency "foo" has no URL`,
			exit:   1,
		})
	})

	t.Run("missing var", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`
dependencies:
  foo:
    url: https://localhost/{{ .MISSING_VAR }}
`)
		result := runner.run("download", "foo")
		result.assertState(resultState{
			stderr: `cmd: error: error applying template`,
			exit:   1,
		})
	})

	t.Run("missing checksum", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  foo:
    url: %s
`, depURL))
		result := runner.run("download", "foo")
		result.assertState(resultState{
			stderr: `cmd: error: no checksum configured for foo`,
			exit:   1,
		})
	})

	t.Run("--allow-missing-checksum", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  foo:
    url: %s
`, depURL))
		result := runner.run("download", "foo", "--allow-missing-checksum")
		assertDownloadSuccess(t, result)
	})

	t.Run("--allow-missing-checksum with dl error", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  foo:
    url: %s
`, errURL))
		result := runner.run("download", "foo", "--allow-missing-checksum")
		result.assertState(resultState{
			stderr: `cmd: error: failed downloading`,
			exit:   1,
		})
	})

	t.Run("already exists", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  foo:
    url: %s
url_checksums:
  %s: 27dcce60d1ed72920a84dd4bc01e0bbd013e5a841660e9ee2e964e53fb83c0b3
`, depURL, depURL))
		// download to put it in the cache
		result := runner.run("download", "foo")
		assertDownloadSuccess(t, result)
		// download again
		result = runner.run("download", "foo")
		assertDownloadSuccess(t, result)
	})

	t.Run("already exists with --allow-missing-checksum", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  foo:
    url: %s
`, depURL))
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
		servePath := testdataPath("downloadables/rawfile/foo")
		ts := testutil.ServeFile(t, servePath, "/foo/foo", "")
		depURL := ts.URL + "/foo/foo"
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  foo:
    url: %s
url_checksums:
  %s: f044ff8b6007c74bcc1b5a5c92776e5d49d6014f5ff2d551fab115c17f48ac41
`, depURL, depURL))
		result := runner.run("install", "foo")
		result.assertState(resultState{
			stdout: `installed foo to`,
		})
		wantBin := filepath.Join(runner.tmpDir, "bin", "foo")
		testutil.AssertFile(t, wantBin, true, false)
	})

	t.Run("link raw file", func(t *testing.T) {
		runner := newCmdRunner(t)
		servePath := testdataPath("downloadables/rawfile/foo")
		ts := testutil.ServeFile(t, servePath, "/foo/foo", "")
		depURL := ts.URL + "/foo/foo"
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  foo:
    url: %s
    link: true
url_checksums:
  %s: f044ff8b6007c74bcc1b5a5c92776e5d49d6014f5ff2d551fab115c17f48ac41
`, depURL, depURL))
		result := runner.run("install", "foo")
		result.assertState(resultState{
			stdout: `installed foo to`,
		})
		wantBin := filepath.Join(runner.tmpDir, "bin", "foo")
		testutil.AssertFile(t, wantBin, true, true)
	})

	t.Run("bin in root", func(t *testing.T) {
		runner := newCmdRunner(t)
		servePath := testdataPath("downloadables/fooinroot.tar.gz")
		ts := testutil.ServeFile(t, servePath, "/foo/fooinroot.tar.gz", "")
		depURL := ts.URL + "/foo/fooinroot.tar.gz"
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  foo:
    url: %s
url_checksums:
  %s: 27dcce60d1ed72920a84dd4bc01e0bbd013e5a841660e9ee2e964e53fb83c0b3
`, depURL, depURL))
		result := runner.run("install", "foo")
		result.assertState(resultState{
			stdout: `installed foo to`,
		})
		wantBin := filepath.Join(runner.tmpDir, "bin", "foo")
		testutil.AssertFile(t, wantBin, true, false)
	})

	t.Run("wrong checksum", func(t *testing.T) {
		runner := newCmdRunner(t)
		servePath := testdataPath("downloadables/fooinroot.tar.gz")
		ts := testutil.ServeFile(t, servePath, "/foo/fooinroot.tar.gz", "")
		depURL := ts.URL + "/foo/fooinroot.tar.gz"
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  foo:
    url: %q
url_checksums:
  %q: "0000000000000000000000000000000000000000000000000000000000000000"
`, depURL, depURL))
		result := runner.run("install", "foo")
		require.Equal(t, 1, result.exitVal)
		require.True(t, strings.HasPrefix(result.stdErr.String(), `cmd: error: checksum mismatch in downloaded file`))
		require.NoFileExists(t, filepath.Join(runner.tmpDir, "bin", "foo"))
	})
}

func Test_wrapCmd(t *testing.T) {
	t.Run("bindown path", func(t *testing.T) {
		runner := newCmdRunner(t)
		servePath := testdataPath("downloadables/runnable.tar.gz")
		ts := testutil.ServeFile(t, servePath, "/runnable/runnable.tar.gz", "")
		depURL := ts.URL + "/runnable/runnable.tar.gz"
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  runnable:
    archive_path: bin/runnable.sh
    url: %s
url_checksums:
    %s: fb2fe41a34b77ee180def0cb9a222d8776a6e581106009b64f35983da291ab6e
`, depURL, depURL))
		outputDir := filepath.Join(runner.tmpDir, "output")
		runnable := filepath.Join(outputDir, "runnable")
		result := runner.run("wrap", "runnable", "--bindown", testutil.BindownBin(), "--output", runnable)
		result.assertState(resultState{stdout: runnable})
		testutil.AssertFile(t, runnable, true, false)
		testutil.CheckGoldenDir(t, outputDir, filepath.FromSlash("testdata/golden/wrap/bindown-path"))

		// make sure it runs
		cmd := exec.Command("sh", "-c", filepath.ToSlash(runnable))
		out, err := cmd.Output()
		require.NoError(t, err)
		require.Equal(t, "Hello world", strings.TrimSpace(string(out)))
	})
}
