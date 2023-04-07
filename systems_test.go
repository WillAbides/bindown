package bindown

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSystemInfo_UnmarshalText(t *testing.T) {
	t.Run("current", func(t *testing.T) {
		s := &SystemInfo{}
		err := s.UnmarshalText([]byte("current"))
		require.NoError(t, err)
		require.Equal(t, runtime.GOOS, s.OS)
		require.Equal(t, runtime.GOARCH, s.Arch)
	})

	t.Run("os/arch", func(t *testing.T) {
		s := &SystemInfo{}
		err := s.UnmarshalText([]byte("os/arch"))
		require.NoError(t, err)
		require.Equal(t, "os", s.OS)
		require.Equal(t, "arch", s.Arch)
	})

	t.Run("invalid", func(t *testing.T) {
		s := &SystemInfo{}
		err := s.UnmarshalText([]byte("invalid"))
		require.Error(t, err)
	})
}
