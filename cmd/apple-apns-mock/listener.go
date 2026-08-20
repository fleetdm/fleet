package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"sync/atomic"
	"time"
)

// resilientListener keeps http.Server.Serve alive across accept failures.
//
// net/http retries only accept errors that report Temporary() -- EINTR,
// EMFILE, ENFILE, EAGAIN, ETIMEDOUT -- and hands everything else back to
// Serve, which returns it to main. ENOBUFS and ENOMEM are the ones that matter
// at scale: the kernel can refuse an accept for reasons that clear a moment
// later, and ending the process over a transient refusal drops every open SSE
// stream and loses the whole load test.
//
// Failures are counted rather than only logged, because a process that cannot
// accept looks identical to an idle one from the outside: flat CPU, flat
// memory, and existing streams still being served.
type resilientListener struct {
	net.Listener

	logger *slog.Logger
	errs   *atomic.Int64
}

func (l *resilientListener) Accept() (net.Conn, error) {
	// Same bounds net/http uses for the errors it retries itself.
	const (
		minDelay = 5 * time.Millisecond
		maxDelay = time.Second
	)

	var delay time.Duration
	for {
		conn, err := l.Listener.Accept()
		if err == nil {
			return conn, nil
		}
		// A closed listener has to propagate: retrying it spins forever and
		// Serve could never return.
		if errors.Is(err, net.ErrClosed) {
			return nil, err
		}

		total := l.errs.Add(1)
		if delay == 0 {
			delay = minDelay
		} else {
			delay = min(delay*2, maxDelay)
		}
		// Deliberately no file descriptor count here: this path can fire
		// thousands of times a second at the shortest backoff, and counting
		// 300k descriptors per attempt would itself become the problem. The
		// periodic stats line carries that number.
		l.logger.ErrorContext(context.Background(), "accept failed, retrying",
			"error", err, "retry_in", delay, "accept_errors_total", total)
		time.Sleep(delay)
	}
}
