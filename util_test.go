package bindown

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_must(t *testing.T) {
	require.Panics(t, func() {
		must(assert.AnError)
	})
}

func Test_logCloseErr(t *testing.T) {
	oldLogWriter := log.Writer()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(oldLogWriter)
	})
	logCloseErr(&dummyCloser{})
	require.Empty(t, buf.String())
	logCloseErr(&dummyCloser{
		err: assert.AnError,
	})
	require.True(t, strings.Contains(buf.String(), assert.AnError.Error()))
}

type dummyCloser struct {
	err error
}

func (d *dummyCloser) Close() error {
	return d.err
}
