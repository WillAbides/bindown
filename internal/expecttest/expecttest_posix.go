//go:build !windows

package expecttest

import (
	"sync"
	"testing"

	"github.com/Netflix/go-expect"
	pseudotty "github.com/creack/pty"
	"github.com/hinshun/vt10x"
	"github.com/stretchr/testify/assert"
)

func run(
	t testing.TB,
	expectFunc func(*expect.Console),
	testFunc func(*expect.Console),
	opts ...Option,
) bool {
	o := option{}
	for _, opt := range opts {
		opt(&o)
	}
	pty, tty, err := pseudotty.Open()
	if !assert.NoError(t, err) {
		return false
	}
	consoleOpts := append([]expect.ConsoleOpt{
		expect.WithStdout(vt10x.New(vt10x.WithWriter(tty))),
		expect.WithStdin(pty),
		expect.WithCloser(pty, tty),
	}, o.consoleOpts...)
	console, err := expect.NewConsole(consoleOpts...)
	if !assert.NoError(t, err) {
		return false
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		expectFunc(console)
	}()
	testFunc(console)
	err = console.Close()
	if !assert.NoError(t, err) {
		return false
	}
	wg.Wait()
	return true
}
