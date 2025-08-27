package main

import (
	"context"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/gitops-migrate/log"
)

const cmdBackup = "backup"

func cmdBackupExec(ctx context.Context, args Args) error {
	archivePath, err := backup(ctx, args.From, args.To)
	if err != nil {
		return err
	}
	log.Info(
		"Fleet GitOps backup completed successfully.",
		"Archive Path", archivePath,
	)

	return nil
}
