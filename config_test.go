package bindown

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v2/internal/testhelper"
)

func TestNewConfigFile(t *testing.T) {
	t.Run("current format", func(t *testing.T) {
		filename := testhelper.ProjectPath(filepath.FromSlash("testdata/config/ex1.json"))
		configFile, err := NewConfigFile(filename)
		assert.NoError(t, err)
		assert.Equal(t, "darwin-amd64", configFile.Downloaders["gobin"][0].ArchivePath)
		assert.True(t, configFile.Downloaders["gobin"][0].Link)
	})

	t.Run("current format", func(t *testing.T) {
		filename := testhelper.ProjectPath(filepath.FromSlash("testdata/config/oldformat.json"))
		configFile, err := NewConfigFile(filename)
		assert.NoError(t, err)
		assert.Equal(t, "darwin-amd64", configFile.Downloaders["gobin"][0].ArchivePath)
		assert.True(t, configFile.Downloaders["gobin"][0].Link)
	})

	t.Run("unknown field", func(t *testing.T) {
		filename := testhelper.ProjectPath(filepath.FromSlash("testdata/config/unknownfield.json"))
		_, err := NewConfigFile(filename)
		assert.Error(t, err)
	})

	t.Run("file does not exist", func(t *testing.T) {
		filename := testhelper.ProjectPath(filepath.FromSlash("testdata/config/fakefile.json"))
		_, err := NewConfigFile(filename)
		assert.Error(t, err)
	})
}

func TestConfig_Downloader(t *testing.T) {
	configFile, err := NewConfigFile(testhelper.ProjectPath(filepath.FromSlash("testdata/config/ex1.json")))
	require.NoError(t, err)
	config := configFile.Config

	t.Run("success", func(t *testing.T) {
		dl := config.Downloader("gobin", "linux", "amd64")
		assert.NotNil(t, dl)
		assert.Equal(t, "415266d9af98578067051653f5057ea267c51ebf085408df48b118a8b978bac6", dl.Checksum)
	})

	t.Run("no mapped bin", func(t *testing.T) {
		dl := config.Downloader("foo", "darwin", "amd64")
		assert.Nil(t, dl)
	})

	t.Run("missing os", func(t *testing.T) {
		dl := config.Downloader("gobin", "windows", "amd64")
		assert.Nil(t, dl)
	})

	t.Run("missing arch", func(t *testing.T) {
		dl := config.Downloader("gobin", "darwin", "x86")
		assert.Nil(t, dl)
	})
}

func TestConfigFile_WriteFile(t *testing.T) {
	filename := testhelper.ProjectPath(filepath.FromSlash("testdata/config/ex1.json"))
	wantBytes, err := ioutil.ReadFile(filename)
	require.NoError(t, err)
	configFile, err := NewConfigFile(filename)
	require.NoError(t, err)
	tmpDir, teardown := testhelper.TmpDir(t)
	defer teardown()
	newFile := filepath.Join(tmpDir, "config.json")
	configFile.filename = newFile
	err = configFile.WriteFile()
	assert.NoError(t, err)
	gotBytes, err := ioutil.ReadFile(newFile)
	require.NoError(t, err)
	assert.JSONEq(t, string(wantBytes), string(gotBytes))
}
