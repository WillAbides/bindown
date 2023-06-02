//go:build windows

package expecttest

func run(
	t testing.TB,
	expectFunc func(*expect.Console),
	testFunc func(*expect.Console),
	opts ...Option,
) bool {
	t.Skip("expecttest is not supported on Windows")
	return false
}
