package bindown

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// fooChecksum is the checksum of downloadablesPath("foo.tar.gz")
const fooChecksum = "f7fa712caea646575c920af17de3462fe9d08d7fe062b9a17010117d5fa4ed88"

func mustConfigFromYAML(t *testing.T, yml string) *Config {
	t.Helper()
	got, err := ConfigFromYAML(context.Background(), []byte(yml))
	require.NoError(t, err)
	return got
}

func ptr[T any](val T) *T {
	return &val
}
