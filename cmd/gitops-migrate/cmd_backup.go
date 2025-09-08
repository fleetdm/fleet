package main

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/cmd/gitops-migrate/log"
)

const cmdBackup = "backup"

func cmdBackupExec(ctx context.Context, args Args) error {
	// Grab the backup "source" path and the archive output path.
	if len(args.Commands) < 2 {
		return errors.New(
			"please specify the path to your GitOps files for backup",
		)
	}
	from := args.Commands[0]
	to := args.Commands[1]

	// Perform the backup.
	archivePath, err := backup(ctx, from, to)
	if err != nil {
		return err
	}
	log.Info(
		"Fleet GitOps backup completed successfully.",
		"Archive Path", archivePath,
	)

	return nil
}
