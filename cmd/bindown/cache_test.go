package main

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_cacheClearCmd(t *testing.T) {
	servePath := testdataPath("downloadables/fooinroot.tar.gz")
	successServer := serveFile(t, servePath, "/foo/fooinroot.tar.gz", "")
	depURL := successServer.URL + "/foo/fooinroot.tar.gz"

	t.Run("removes populated cache", func(t *testing.T) {
		runner := newCmdRunner(t)
		// extract something to populate the cache
		runner.writeConfigYaml(fmt.Sprintf(`
dependencies:
  foo:
    url: %s
url_checksums:
  %s: 27dcce60d1ed72920a84dd4bc01e0bbd013e5a841660e9ee2e964e53fb83c0b3
`, depURL, depURL))
		result := runner.run("extract", "foo")
		extractDir := result.getExtractDir()
		assert.FileExists(t, filepath.Join(extractDir, "foo"))
		result = runner.run("cache", "clear")
		result.assertState(resultState{})
		assert.NoDirExists(t, extractDir)
	})

	t.Run("does nothing if cache is empty", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`{}`)
		result := runner.run("cache", "clear")
		result.assertState(resultState{})
	})

	t.Run("errors on missing config", func(t *testing.T) {
		runner := newCmdRunner(t)
		result := runner.run("cache", "clear")
		result.assertState(resultState{
			exit:   1,
			stderr: `no such file or directory`,
		})
	})
}
