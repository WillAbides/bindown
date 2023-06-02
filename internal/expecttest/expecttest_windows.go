//go:build windows

package expecttest

import (
	"testing"

	"github.com/Netflix/go-expect"
)

func run(
	t testing.TB,
	expectFunc func(*expect.Console),
	testFunc func(*expect.Console),
	opts ...Option,
) bool {
	t.Skip("expecttest is not supported on Windows")
	return false
}
