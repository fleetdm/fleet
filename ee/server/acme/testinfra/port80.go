// Package testinfra provides shared test infrastructure for the ACME bounded context.
package testinfra

import (
	"net"
	"testing"
	"time"
)

// ListenPort80 attempts to bind port 80 with retries. When multiple test
// packages run in parallel (go test ./...), they may contend on port 80
// for http-01 challenge servers. This function retries for up to 30 seconds
// to allow the other test to release the port.
//
// Returns the listener and true on success, or nil and false if port 80
// cannot be bound (e.g., insufficient privileges).
func ListenPort80(t *testing.T) (net.Listener, bool) {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	var lastErr error

	for time.Now().Before(deadline) {
		listener, err := net.Listen("tcp", "127.0.0.1:80")
		if err == nil {
			return listener, true
		}
		lastErr = err

		// If the error is permission-denied, retrying won't help
		if isPermissionError(err) {
			t.Logf("cannot bind port 80 (insufficient privileges): %v", err)
			return nil, false
		}

		// Port is likely in use by another test package — wait and retry
		time.Sleep(500 * time.Millisecond)
	}

	t.Logf("cannot bind port 80 after 30s retries: %v", lastErr)
	return nil, false
}

func isPermissionError(err error) bool {
	if opErr, ok := err.(*net.OpError); ok {
		return opErr.Err.Error() == "bind: permission denied"
	}
	return false
}
