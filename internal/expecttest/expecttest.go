package expecttest

import (
	"testing"

	"github.com/Netflix/go-expect"
)

type option struct {
	consoleOpts []expect.ConsoleOpt
}

type Option func(*option)

func WithConsoleOpt(opt ...expect.ConsoleOpt) Option {
	return func(o *option) {
		o.consoleOpts = append(o.consoleOpts, opt...)
	}
}

func Run(
	t testing.TB,
	expectFunc func(*expect.Console),
	testFunc func(*expect.Console),
	opts ...Option,
) bool {
	return run(t, expectFunc, testFunc, opts...)
}
