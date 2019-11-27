package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v2/internal/testhelper"
	"github.com/willabides/bindown/v2/pkg/config/internal/mocks"
	"go.uber.org/multierr"
)

func TestConfig_Validate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cellarDir := "cellar"
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockDownloader := mocks.NewMockDownloader(ctrl)
		mockDownloader.EXPECT().Validate(cellarDir)
		mockDownloader2 := mocks.NewMockDownloader(ctrl)
		mockDownloader2.EXPECT().Validate(cellarDir)
		downloaders := map[string][]Downloader{
			"foo": {mockDownloader, mockDownloader2},
		}
		cfg := &Config{
			Downloaders: downloaders,
		}
		err := cfg.Validate("foo", cellarDir)
		assert.NoError(t, err)
	})

	t.Run("errs", func(t *testing.T) {
		err1 := fmt.Errorf("err1")
		err2 := fmt.Errorf("err2")
		cellarDir := "cellar"
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockDownloader := mocks.NewMockDownloader(ctrl)
		mockDownloader.EXPECT().Validate(cellarDir).Return(err1)
		mockDownloader.EXPECT().ErrString("foo").Return("errmsg1")
		mockDownloader2 := mocks.NewMockDownloader(ctrl)
		mockDownloader2.EXPECT().Validate(cellarDir)
		mockDownloader3 := mocks.NewMockDownloader(ctrl)
		mockDownloader3.EXPECT().Validate(cellarDir).Return(err2)
		mockDownloader3.EXPECT().ErrString("foo").Return("errmsg2")
		downloaders := map[string][]Downloader{
			"foo": {mockDownloader, mockDownloader2, mockDownloader3},
		}
		cfg := &Config{
			Downloaders: downloaders,
		}
		err := cfg.Validate("foo", cellarDir)
		assert.Error(t, err)
		errs := multierr.Errors(err)
		assert.Len(t, errs, 2)
		assert.Equal(t, "error validating errmsg1: err1", errs[0].Error())
		assert.Equal(t, "error validating errmsg2: err2", errs[1].Error())
	})
}

func TestUpdateChecksums(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockDownloader := mocks.NewMockDownloader(ctrl)
		mockDownloader.EXPECT().UpdateChecksum(gomock.Any())
		mockDownloader2 := mocks.NewMockDownloader(ctrl)
		mockDownloader2.EXPECT().UpdateChecksum(gomock.Any())
		downloaders := map[string][]Downloader{
			"foo": {mockDownloader, mockDownloader2},
		}
		cfg := &Config{
			Downloaders: downloaders,
		}
		err := cfg.UpdateChecksums("foo", "")
		assert.NoError(t, err)
	})

	t.Run("errs", func(t *testing.T) {
		cellarDir := "cellar"
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockDownloader := mocks.NewMockDownloader(ctrl)
		mockDownloader.EXPECT().UpdateChecksum(cellarDir)
		mockDownloader2 := mocks.NewMockDownloader(ctrl)
		mockDownloader2.EXPECT().UpdateChecksum(cellarDir)
		mockDownloader3 := mocks.NewMockDownloader(ctrl)
		mockDownloader3.EXPECT().UpdateChecksum(cellarDir).Return(assert.AnError)
		downloaders := map[string][]Downloader{
			"foo": {mockDownloader, mockDownloader2, mockDownloader3},
		}
		cfg := &Config{
			Downloaders: downloaders,
		}
		err := cfg.UpdateChecksums("foo", cellarDir)
		assert.Error(t, err)
		assert.Equal(t, assert.AnError, err)
	})
}

func TestNewConfigFile(t *testing.T) {
	t.Run("current format", func(t *testing.T) {
		filename := testhelper.ProjectPath(filepath.FromSlash("testdata/config/ex1.json"))
		configFile, err := NewConfigFile(filename)
		assert.NoError(t, err)
		assert.True(t, configFile.Downloaders["gobin"][0].
			HasChecksum("84ed966949e06bebd7d006bc343caf9d736932fd8b37df5cb5b268a28d07bd30"))
	})

	t.Run("old format", func(t *testing.T) {
		filename := testhelper.ProjectPath(filepath.FromSlash("testdata/config/oldformat.json"))
		configFile, err := NewConfigFile(filename)
		assert.NoError(t, err)
		assert.True(t, configFile.Downloaders["gobin"][0].
			HasChecksum("84ed966949e06bebd7d006bc343caf9d736932fd8b37df5cb5b268a28d07bd30"))
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
		assert.True(t, dl.HasChecksum("415266d9af98578067051653f5057ea267c51ebf085408df48b118a8b978bac6"))
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
