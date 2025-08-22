package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
)

func main() {
	// Init the application context.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer cancel()

	// Parse command-line inputs.
	args := parseArgs()

	// Setup logging, burning the logger into the context.
	//
	// See log.go/LoggerFromContext+LoggerIntoContext for more details.
	ctx = setupLogging(ctx, args)

	// Execute the command.
	err := cmdExec(ctx, args)
	if err != nil {
		LoggerFromContext(ctx).Error(
			"failed to execute command",
			"command", args.Commands,
			"error", err,
		)
	}
}

func setupLogging(ctx context.Context, args Args) context.Context {
	// Log with short file name ('main.go:20').
	log.SetFlags(log.Lshortfile)

	// Init a default slog-to-stderr text handler.
	handlerOpts := slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	// If we got the '--debug' flag, set the log level to debug.
	if args.Debug {
		handlerOpts.Level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stderr, nil)

	// Init and assign the default slogger.
	slogger := slog.New(handler)
	slog.SetDefault(slogger)

	// Burn the logger into the context.
	return LoggerIntoContext(ctx, slogger)
}
