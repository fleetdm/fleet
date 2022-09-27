//go:build unix

package main

import (
	"context"
	"os"
	"syscall"

	"github.com/oklog/run"
)

func signalHandler(ctx context.Context) (execute func() error, interrupt func(error)) {
	return run.SignalHandler(ctx, os.Interrupt, os.Kill, syscall.SIGTERM)
}
