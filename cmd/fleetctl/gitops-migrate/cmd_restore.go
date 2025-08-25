package main

import (
	"context"
)

const cmdRestore = "restore"

func cmdRestoreExec(ctx context.Context, args Args) error {
	log := LoggerFromContext(ctx)

	err := restore(ctx, args.From, args.To)
	if err != nil {
		return err
	}
	log.Info(
		"Fleet GitOps restore completed successfully",
		"restored_path", args.To,
	)

	return nil
}
