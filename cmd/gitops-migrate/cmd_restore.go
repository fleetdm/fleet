package main

import (
	"context"

	"github.com/fleetdm/fleet/v4/cmd/gitops-migrate/log"
)

const cmdRestore = "restore"

func cmdRestoreExec(ctx context.Context, args Args) error {
	if len(args.Commands) < 2 {
		showUsageAndExit(
			1,
			"please enter the path to the archive to restore, "+
				"followed by the directory you want to restore the archive to",
		)
	}
	from, to := args.Commands[0], args.Commands[1]
	err := restore(ctx, from, to)
	if err != nil {
		return err
	}
	log.Info(
		"Fleet GitOps restore completed successfully.",
		"Restored To", to,
	)

	return nil
}
