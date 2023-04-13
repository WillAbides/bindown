package main

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v3/internal/bindown"
)

func Test_addChecksumsCmd(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var servers [5]*httptest.Server
		var urls [5]string
		for i := range servers {
			servers[i] = serveFile(t, testdataPath("downloadables/foo.tar.gz"), "/foo/foo.tar.gz", "")
			urls[i] = servers[i].URL + "/foo/foo.tar.gz"
		}
		runner := newCmdRunner(t)
		runner.writeConfig(&bindown.Config{
			Dependencies: map[string]*bindown.Dependency{
				"d1": {
					URL: &urls[0],
					Overrides: []bindown.DependencyOverride{
						{
							Dependency:      bindown.Dependency{URL: &urls[1]},
							OverrideMatcher: map[string][]string{"os": {"darwin"}},
						},
						{
							Dependency:      bindown.Dependency{URL: &urls[4]},
							OverrideMatcher: map[string][]string{"os": {"windows"}},
						},
					},
				},
				"d2": {
					URL: &urls[2],
					Overrides: []bindown.DependencyOverride{
						{
							Dependency:      bindown.Dependency{URL: &urls[3]},
							OverrideMatcher: map[string][]string{"os": {"darwin"}},
						},
					},
				},
			},
		})
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
		runner.writeConfig(&bindown.Config{
			Dependencies: map[string]*bindown.Dependency{
				"d1": {URL: &server.URL},
			},
		})
		result := runner.run("checksums", "add", "--system", "darwin/amd64")
		result.assertState(resultState{
			stderr: "cmd: error: failed downloading",
			exit:   1,
		})
	})

	t.Run("dependency does not exist", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfig(&bindown.Config{
			Dependencies: map[string]*bindown.Dependency{
				"d1": {URL: ptr("fake")},
			},
		})
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
		runner.writeConfig(&bindown.Config{
			URLChecksums: map[string]string{
				"foo": "bar",
				"baz": "qux",
			},
			Dependencies: map[string]*bindown.Dependency{
				"d1": {URL: ptr("foo")},
			},
		})
		result := runner.run("checksums", "prune")
		result.assertState(resultState{})
		want := map[string]string{
			"foo": "bar",
		}
		require.Equal(t, want, runner.getConfigFile().URLChecksums)
	})
}
