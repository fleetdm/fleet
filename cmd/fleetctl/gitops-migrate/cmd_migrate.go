package main

import (
	"context"
	"os"
)

const (
	cmdMigrate = "migrate"
)

func cmdMigrateExec(ctx context.Context, path string) error {
	// Create a temp directory to which we'll write the backup archive.
	tmpDir, err := mkBackupDir()
	if err != nil {
		return err
	}

	// Backup the provided migration target.
	archivePath, err := backup(ctx, path, tmpDir)
	if err != nil {
		return err
	}
	_ = archivePath

	// Enumerate all files in the provided path.
	for file, err := range fsEnum(path) {
		if err != nil {
			return err
		}

		// Read the file.
		content, err := os.ReadFile(file.Path)
		if err != nil {
			return err
		}

		_ = content
		// Unmarshal the file content.
	}

	return nil
}
