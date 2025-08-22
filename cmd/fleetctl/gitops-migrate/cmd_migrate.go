package main

import (
	"context"
	"os"
)

const (
	cmdMigrate = "migrate"
)

func cmdMigrateExec(ctx context.Context, path string) error {
	// Backup the provided migration target.
	//
	// NOTE: Calling 'backup' with path as both 'from' and 'to' args looks a
	// little weird but the function creates a timestamped file name so there's
	// no worries of file collision or overwrite. The purpose of this
	// implementation is to make it convenient to expose the output path to the
	// user eventually if we want.
	archivePath, err := backup(ctx, path, path)
	if err != nil {
		return err
	}
	_ = archivePath

	// Enumerate all files in the provided path.
	for filePath, err := range fsEnum(path) {
		if err != nil {
			return err
		}

		// Read the file.
		content, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		_ = content
		// Unmarshal the file content.
	}

	return nil
}
