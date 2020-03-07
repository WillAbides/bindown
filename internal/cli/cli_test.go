package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v2/internal/testutil"
	"github.com/willabides/bindown/v2/internal/util"
)

func TestRun(t *testing.T) {
	dir := testutil.TmpDir(t)
	configFile := filepath.Join(dir, "bindown.yml")
	require.NoError(t, util.CopyFile(testutil.ProjectPath("testdata", "configs", "ex1.yaml"), configFile, nil))
	require.NoError(t, os.Setenv("BINDOWN_CONFIG_FILE", configFile))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("BINDOWN_CONFIG_FILE"))
	})
	Run([]string{"version"}, kong.Exit(func(i int) {
		fmt.Printf("exited %d\n", i)
	}))
}
