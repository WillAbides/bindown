package expecttest

import (
	"fmt"
	"github.com/Netflix/go-expect"
	pseudotty "github.com/creack/pty"
	"github.com/hinshun/vt10x"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

type Test struct {
	console    *expect.Console
	expectFunc func(*expect.Console)
	wg         sync.WaitGroup
	started    bool
}

func (x *Test) Start() error {
	if x.started {
		return fmt.Errorf("already started")
	}
	x.started = true
	x.wg.Add(1)
	go func() {
		defer x.wg.Done()
		x.expectFunc(x.console)
	}()
	return nil
}

func (x *Test) Wait() {
	x.wg.Wait()
}

func (x *Test) Close() error {
	return x.console.Close()
}

func New(expectFunc func(*expect.Console), opt ...expect.ConsoleOpt) (*Test, error) {
	pty, tty, err := pseudotty.Open()
	if err != nil {
		return nil, err
	}
	opt = append([]expect.ConsoleOpt{
		expect.WithStdout(vt10x.New(vt10x.WithWriter(tty))),
		expect.WithStdin(pty),
		expect.WithCloser(pty, tty),
	}, opt...)
	console, err := expect.NewConsole(opt...)
	if err != nil {
		return nil, err
	}
	return &Test{
		console:    console,
		expectFunc: expectFunc,
	}, nil
}

func MustNew(expectFunc func(*expect.Console), opt ...expect.ConsoleOpt) *Test {
	x, err := New(expectFunc, opt...)
	if err != nil {
		panic(err)
	}
	return x
}

func (x *Test) Run(t testing.TB, fn func(*expect.Console)) bool {
	err := x.Start()
	if !assert.NoError(t, err) {
		return false
	}
	fn(x.console)
	err = x.Close()
	if !assert.NoError(t, err) {
		return false
	}
	x.Wait()
	return true
}
