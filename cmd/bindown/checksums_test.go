package main

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v4/internal/testutil"
)

func Test_addChecksumsCmd(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var servers [5]*httptest.Server
		var urls [5]string
		for i := range servers {
			servers[i] = testutil.ServeFile(t, testdataPath("downloadables/foo.tar.gz"), "/foo/foo.tar.gz", "")
			urls[i] = servers[i].URL + "/foo/foo.tar.gz"
		}
		runner := newCmdRunner(t)
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  d1:
    url: %q
    overrides:
      - dependency:
          url: %q
        matcher:
         os: [ darwin ]
      - dependency:
           url: %q
        matcher:
            os: [ windows ]
  d2:
    url: %q
    overrides:
      - dependency:
          url: %q
        matcher:
          os: [ darwin ]
`, urls[0], urls[1], urls[2], urls[2], urls[3]))

		result := runner.run("checksums", "add", "--system", "darwin/amd64", "--system", "linux/amd64")
		result.assertState(resultState{})
		want := map[string]string{
			urls[0]: fooChecksum,
			urls[1]: fooChecksum,
			urls[2]: fooChecksum,
			urls[3]: fooChecksum,
		}
		require.Equal(t, want, runner.getConfigFile().URLChecksums)
	})

	t.Run("400", func(t *testing.T) {
		server := serveErr(t, 400)
		runner := newCmdRunner(t)
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  d1:
    url: %q
`, server.URL))
		result := runner.run("checksums", "add", "--system", "darwin/amd64")
		result.assertState(resultState{
			stderr: "cmd: error: failed downloading",
			exit:   1,
		})
	})

	t.Run("dependency does not exist", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`
dependencies:
  d1:
    url: fake
`)
		result := runner.run("checksums", "add", "--system", "darwin/amd64", "--dependency", "d2")
		result.assertState(resultState{
			stderr: `cmd: error: no dependency configured with the name "d2"`,
			exit:   1,
		})
	})
}

func Test_pruneChecksumsCmd(t *testing.T) {
	t.Run("prunes", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`
url_checksums:
  foo: bar
  baz: qux
dependencies:
  d1:
    url: foo
`)
		result := runner.run("checksums", "prune")
		result.assertState(resultState{})
		want := map[string]string{
			"foo": "bar",
		}
		require.Equal(t, want, runner.getConfigFile().URLChecksums)
	})
}
