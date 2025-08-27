package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/gitops-migrate/log"
)

func main() {
	// Init the application context.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer cancel()

	// Parse command-line inputs.
	args := parseArgs()

	// If we received the ['--debug', '-d'] flag, tune the log config.
	if args.Debug {
		log.Options.SetWithCaller()
		log.Options.SetWithLevel()
		log.Level = log.LevelDebug
	}

	// Execute the command.
	err := cmdExec(ctx, args)
	if err != nil {
		log.Fatal(
			"Failed to execute command.",
			"Command", args.Commands,
			"Error", err,
		)
	}
}
