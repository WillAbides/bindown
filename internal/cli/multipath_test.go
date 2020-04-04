package cli

import (
	"reflect"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_multipathMapper(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		configFile := createConfigFile(t, "ex1.yaml")
		decodeCtx := &kong.DecodeContext{
			Scan: kong.Scan("invalidfile|" + configFile),
		}
		var got string
		err := multipathMapper(decodeCtx, reflect.ValueOf(&got).Elem())
		require.NoError(t, err)
		assert.Equal(t, configFile, got)
	})

	t.Run("nil context", func(t *testing.T) {
		var got string
		err := multipathMapper(nil, reflect.ValueOf(&got).Elem())
		require.Error(t, err)
		assert.Equal(t, "", got)
	})

	t.Run("no existing file", func(t *testing.T) {
		decodeCtx := &kong.DecodeContext{
			Scan: kong.Scan("invalidfile|alsoinvalid"),
		}
		var got string
		err := multipathMapper(decodeCtx, reflect.ValueOf(&got).Elem())
		require.Error(t, err)
		assert.Empty(t, got)
	})

	t.Run("no token to scan", func(t *testing.T) {
		decodeCtx := &kong.DecodeContext{
			Scan: kong.Scan(),
		}
		var got string
		err := multipathMapper(decodeCtx, reflect.ValueOf(&got).Elem())
		require.Error(t, err)
		assert.Equal(t, "", got)
	})
}
