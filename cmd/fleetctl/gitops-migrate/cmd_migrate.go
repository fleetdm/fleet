package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	cmdMigrate = "migrate"
)

func cmdMigrateExec(ctx context.Context, args Args) error {
	// Create a temp directory to which we'll write the backup archive.
	tmpDir, err := mkBackupDir()
	if err != nil {
		return err
	}

	// Backup the provided migration target.
	archivePath, err := backup(ctx, args.From, tmpDir)
	if err != nil {
		return err
	}
	// TODO: use a named return for 'error' and put this archive back if we fail
	// the migration.
	_ = archivePath

	// Enumerate all files in the provided path.
	for file, err := range fsEnum(args.From) {
		if err != nil {
			return err
		}

		// Get a readable handle to the input file.
		f, err := os.Open(file.Path)
		if err != nil {
			return fmt.Errorf(
				"failed to get readable handle to source file(%s): %w",
				file.Path, err,
			)
		}

		// Init a SHA-256 'hash.Hasher'.
		hasher := sha256.New()

		// Init a buffer to read the file to.
		buf := bytes.NewBuffer(make([]byte, 0, file.Stats.Size()))

		// Read & hash the file.
		w := io.MultiWriter(hasher, buf)
		n, err := io.Copy(w, f)
		if err != nil {
			return fmt.Errorf(
				"failed to read source file(%s): %w",
				file.Path, err,
			)
		}

		// Ensure we read the file size we expect.
		if n != file.Stats.Size() {
			return fmt.Errorf(
				"encountered no error in source file(%s) read, however the stat'd "+
					"file size(%d) didn't match the size we actually wrote(%d)",
				file.Path, file.Stats.Size(), n,
			)
		}

		// Close the input file stream.
		err = f.Close()
		if err != nil {
			return fmt.Errorf(
				"failed to close source file(%s) stream: %w",
				file.Path, err,
			)
		}

		// Unmarshal the file content.
		m := make(map[string]any)
		err = yaml.Unmarshal(buf.Bytes(), &m)
		if err != nil {
			return fmt.Errorf(
				"failed to unmarshal YAML input(%s): %w",
				file.Path, err,
			)
		}
	}

	return nil
}
