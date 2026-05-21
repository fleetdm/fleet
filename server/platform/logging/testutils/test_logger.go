package testutils

import (
	"log/slog"
	"testing"
)

// TestLogger returns a *slog.Logger that routes output through t.Log.
// Logs are only printed when a test fails (or with -v), keeping passing test output clean.
// In parallel tests, logs stay grouped with their test instead of interleaving on stdout.
//
// Example usage:
//
//	func TestSomething(t *testing.T) {
//		logger := testutils.TestLogger(t)
//		svc := mypackage.NewService(logger)
//		// ... test svc; log output only appears if the test fails
//	}
func TestLogger(t testing.TB) *slog.Logger {
	return slog.New(slog.NewTextHandler(tLogWriter{t}, nil))
}

// tLogWriter adapts testing.TB to io.Writer so slog output is captured by t.Log
type tLogWriter struct{ t testing.TB }

func (w tLogWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}
