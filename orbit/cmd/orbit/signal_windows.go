package main

import (
	"context"
	"os"

	"github.com/oklog/run"
)

func signalHandler(ctx context.Context) (execute func() error, interrupt func(error)) {
	return run.SignalHandler(ctx, os.Interrupt, os.Kill)
}

func sigusrListener(rootDir string) {
}
