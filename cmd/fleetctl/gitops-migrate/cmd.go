package main

import (
	"context"
	"strings"
)

func cmdExec(ctx context.Context, args Args) error {
	// Check for the ['--help', '-h'] flag.
	if args.Help {
		showUsageAndExit(0, "")
	}

	// Make sure we have a command past this point.
	if len(args.Commands) == 0 {
		showUsageAndExit(2, "Received no command.")
	}

	// Slice off the first command.
	//
	// If we decide to do nested subcommands in the future this will make it super
	// straightforward to implement the branching.
	cmd := args.Commands[0]
	args.Commands = args.Commands[1:]

	switch strings.ToLower(cmd) {
	case cmdUsage, cmdHelp:
		return cmdUsageExec(ctx, args)
	case cmdFormat:
		return cmdFormatExec(ctx, args)
	case cmdMigrate:
		return cmdMigrateExec(ctx, args)
	case cmdBackup:
		return cmdBackupExec(ctx, args)
	case cmdRestore:
		return cmdRestoreExec(ctx, args)
	default:
		showUsageAndExit(2, "Received unknown command: %s.", cmd)
		panic("impossible")
	}
}
