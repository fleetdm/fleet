package main

import "context"

const cmdBackup = "backup"

func cmdBackupExec(ctx context.Context, args Args) error {
	log := LoggerFromContext(ctx)

	archivePath, err := backup(ctx, args.From, args.To)
	if err != nil {
		return err
	}
	log.Info("Fleet GitOps backup completed successfully", "archive", archivePath)

	return nil
}
