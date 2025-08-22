package main

import (
	"context"
	"fmt"
	"strings"
)

func cmdExec(ctx context.Context, args Args) error {
	if len(args.Commands) == 0 {
		return fmt.Errorf("expected command 'migrate' or 'restore'")
	}

	// Slice off the first command.
	//
	// If we decide to do nested subcommands in the future this will make it super
	// straightforward to implement the branching.
	cmd := args.Commands[0]
	args.Commands = args.Commands[1:]

	switch strings.ToLower(cmd) {
	case cmdMigrate:
		return cmdMigrateExec(ctx, args.From)
	case cmdBackup:
		return cmdBackupExec(ctx, args)
	case cmdRestore:
		return cmdRestoreExec(ctx, args)
	default:
		panic("NYI")
	}
}
