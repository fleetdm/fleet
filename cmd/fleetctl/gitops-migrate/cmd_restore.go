package main

import (
	"context"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/gitops-migrate/log"
)

const cmdRestore = "restore"

func cmdRestoreExec(ctx context.Context, args Args) error {
	err := restore(ctx, args.From, args.To)
	if err != nil {
		return err
	}
	log.Info(
		"Fleet GitOps restore completed successfully.",
		"Restored To", args.To,
	)

	return nil
}
