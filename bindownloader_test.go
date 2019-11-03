package bindownloader

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir, teardown := tmpDir(t)
		defer teardown()
		file := filepath.Join(dir, "buildtools.json")

		// language=json
		content := `
{
  "gobin": [
    {
      "os": "darwin",
	  "arch": "amd64",
      "url": "https://github.com/myitcv/gobin/releases/download/v0.0.10/darwin-amd64",
      "checksum": "84ed966949e06bebd7d006bc343caf9d736932fd8b37df5cb5b268a28d07bd30",
      "move-from": "darwin-amd64",
      "bin": "gobin"
    },
    {
      "os": "linux",
	  "arch": "amd64",
      "url": "https://github.com/myitcv/gobin/releases/download/v0.0.10/linux-amd64",
      "checksum": "415266d9af98578067051653f5057ea267c51ebf085408df48b118a8b978bac6",
      "move-from": "linux-amd64",
      "bin": "gobin"
    }
  ]
}
`
		err := ioutil.WriteFile(file, []byte(content), 0640)
		require.NoError(t, err)
		fileReader, err := os.Open(file)
		require.NoError(t, err)
		defer func() {
			require.NoError(t, fileReader.Close())
		}()
		d, err := LoadConfig(fileReader)
		assert.NoError(t, err)
		assert.Equal(t, "gobin", d["gobin"][0].BinName)
	})
}
