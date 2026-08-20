package main

import (
	"io"
	"log/slog"
	"net"
	"os"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubListener replays a scripted sequence of Accept results so a test can
// drive resilientListener through failures that a real listener only produces
// under load.
type stubListener struct {
	results []error // nil entries mean "hand back a connection"
	accepts int
}

func (l *stubListener) Accept() (net.Conn, error) {
	l.accepts++
	if len(l.results) > 0 {
		err := l.results[0]
		l.results = l.results[1:]
		if err != nil {
			return nil, err
		}
	}
	client, server := net.Pipe()
	_ = client.Close()
	return server, nil
}

func (l *stubListener) Close() error   { return nil }
func (l *stubListener) Addr() net.Addr { return &net.TCPAddr{} }

func newTestResilientListener(t *testing.T, results ...error) (*resilientListener, *stubListener, *atomic.Int64) {
	t.Helper()
	stub := &stubListener{results: results}
	var errs atomic.Int64
	return &resilientListener{
		Listener: stub,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		errs:     &errs,
	}, stub, &errs
}

// opErr builds the shape a real accept failure arrives in, since
// resilientListener has to survive whatever the kernel returns rather than
// only the errors net/http already retries.
func opErr(errno syscall.Errno) error {
	return &net.OpError{Op: "accept", Net: "tcp", Err: os.NewSyscallError("accept", errno)}
}

// ENOBUFS and ENOMEM are the reason this type exists: net/http does not
// consider them Temporary, so Serve used to return them and end the process,
// dropping every live stream over a condition that clears by itself.
func TestResilientListenerRetriesNonTemporaryErrors(t *testing.T) {
	for name, errno := range map[string]syscall.Errno{
		"ENOBUFS": syscall.ENOBUFS,
		"ENOMEM":  syscall.ENOMEM,
		"EMFILE":  syscall.EMFILE,
	} {
		t.Run(name, func(t *testing.T) {
			lst, stub, errs := newTestResilientListener(t, opErr(errno), opErr(errno), nil)

			conn, err := lst.Accept()
			require.NoError(t, err, "accept must recover instead of surfacing the error")
			require.NotNil(t, conn)
			_ = conn.Close()

			assert.Equal(t, 3, stub.accepts, "should have retried twice then succeeded")
			assert.Equal(t, int64(2), errs.Load(), "both failures should be counted for /stats")
		})
	}
}

// A closed listener is the one error that must propagate: retrying it would
// spin forever and Serve could never return.
func TestResilientListenerPropagatesClosed(t *testing.T) {
	lst, stub, errs := newTestResilientListener(t, net.ErrClosed)

	_, err := lst.Accept()
	require.ErrorIs(t, err, net.ErrClosed)
	assert.Equal(t, 1, stub.accepts)
	assert.Zero(t, errs.Load(), "shutdown is not an accept failure")
}

func TestResilientListenerPassesThroughSuccess(t *testing.T) {
	lst, stub, errs := newTestResilientListener(t)

	conn, err := lst.Accept()
	require.NoError(t, err)
	require.NotNil(t, conn)
	_ = conn.Close()

	assert.Equal(t, 1, stub.accepts, "a healthy accept must not retry")
	assert.Zero(t, errs.Load())
}
