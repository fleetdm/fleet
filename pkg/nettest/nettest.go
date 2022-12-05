// Package nettest provides functionality to run tests that access the public network.
package nettest

import (
	"errors"
	"net"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/retry"
)

func lock(lockFilePath string) {
	for {
		outFile, err := os.OpenFile(lockFilePath, os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			outFile.Close()
			return
		}
		time.Sleep(5 * time.Second)
	}
}

func unlock(lockFilePath string) {
	os.Remove(lockFilePath)
}

func runSerial(t *testing.T, lockFilePath string) {
	lock(lockFilePath)
	t.Logf("network test start: %s", t.Name())

	t.Cleanup(func() {
		t.Logf("network test done: %s", t.Name())
		unlock(lockFilePath)
	})
}

// Run can be used by test that access the public network.
func Run(t *testing.T) {
	if os.Getenv("NETWORK_TEST") == "" {
		t.Skip("set environment variable NETWORK_TEST=1 to run")
		return
	}

	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return
	}

	if lockFilePath := os.Getenv("TEST_LOCK_FILE_PATH"); lockFilePath != "" {
		runSerial(t, lockFilePath)
	}
}

// Retryable returns whether the error warrants a retry.
func Retryable(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Temporary() || netErr.Timeout() {
			return true
		}
	}
	// Using the exact same error check used in:
	// https://github.com/golang/go/blob/a5d61be040ed20b5774bff1b6b578c6d393ab332/src/net/http/serve_test.go#L1417
	//
	// Also we use raw string matching because this method is used on raw error strings (returned by commands).
	if errStr := err.Error(); (strings.Contains(errStr, "timeout") && strings.Contains(errStr, "TLS handshake")) ||
		strings.Contains(errStr, "unexpected EOF") || strings.Contains(errStr, "connection reset by peer") ||
		strings.Contains(errStr, "unexpected http response") || strings.Contains(errStr, "context deadline exceeded") {
		return true
	}
	return false
}

// RunWithNetRetry runs the given function and retries in case of network errors (see Retryable).
func RunWithNetRetry(t *testing.T, fn func() error) error {
	var err error
	// Do cannot return a non-nil error here.
	_ = retry.Do(func() error {
		err = fn()
		if err != nil && Retryable(err) {
			t.Logf("%s: retrying error: %s", t.Name(), err)
			return err
		}
		return nil
	}, retry.WithInterval(5*time.Second))
	return err
}
