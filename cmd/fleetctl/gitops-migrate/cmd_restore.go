package main

import (
	"context"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/gitops-migrate/log"
)

const cmdRestore = "restore"

func cmdRestoreExec(ctx context.Context, args Args) error {
	if len(args.Commands) < 2 {
		showUsageAndExit(
			1,
			"expected two positional args to command 'restore': the path to the "+
				"archive, and the directory we want to restore the archive to",
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
