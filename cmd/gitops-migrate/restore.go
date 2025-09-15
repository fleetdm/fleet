package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/cmd/gitops-migrate/log"
)

// restore restores a Fleet GitOps backup from the provided tarball.
//
// 'from' must be a path to a gzipped tarball.
//
// If the 'to' path is not provided, the current working directory will be used.
func restore(ctx context.Context, from string, to string) error {
	// Set a default 'to' path, if necessary.
	if to == "" {
		log.Debug(
			"Found no 'to' path for restore operation, defaulting to " +
				"current working directory.",
		)
		to = "."
	}
	log.Info(
		"Performing Fleet GitOps restore.",
		"Archive Path", from,
		"Restore Path", to,
	)

	// Create the output directory, if necessary.
	err := os.MkdirAll(to, fileModeUserRWX)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf(
			"failed to create restore output directory: %w",
			err,
		)
	}

	// Get a readable handle to the archive.
	//
	// NOTE: We handle closure of all streams in a deferred function ~30-lines
	// down.
	f, err := os.Open(from)
	if err != nil {
		return fmt.Errorf(
			"failed to get a readable handle to restore archive(%s): %w",
			from, err,
		)
	}

	// Wrap the '*os.File' in a 'LimitReader' with an upper bound of 1GB to
	// mitigate potential zip bombs.
	limitReader := io.LimitReader(f, 1<<30)

	// Wrap the file stream in a gzip reader.
	//
	// NOTE: We handle closure of all streams in a deferred function ~15-lines
	// down.
	gz, err := gzip.NewReader(limitReader)
	if err != nil {
		return fmt.Errorf(
			"failed to create gzip reader from restore archive file stream(%s): %w",
			from, err,
		)
	}

	// Wrap the gzip reader in a tar reader.
	tr := tar.NewReader(gz)

	// Defer closure of all readers*, in reverse order.
	//
	// * Except the tar reader, it's not a 'ReadCloser'.
	defer func() {
		var errs error

		// Close the gzip reader.
		err := gz.Close()
		if err != nil {
			log.Error(
				"Failed to close restore archive gzip reader.",
				"Error", err,
			)
		}
		errs = errors.Join(errs, err)

		// Close the restore archive file stream.
		err = f.Close()
		if err != nil {
			log.Error(
				"Failed to close restore archive file stream.",
				"Error", err,
			)
		}
		errs = errors.Join(errs, err)

		if errs != nil {
			log.Fatal("Errors encountered in restore archive stream close, exiting.")
		}
	}()

	// Read and extract all files from the tarball.
	for {
		// Get the next compressed file.
		header, err := tr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// We'll catch an 'io.EOF' when we hit the end of the archive.
				return nil
			}
			// Otherwise, something has gone wrong.
			return fmt.Errorf(
				"failed to read next file from the restore archive stream: %w",
				err,
			)
		}

		// Construct the output path for this item.
		output := filepath.Join(to, filepath.Clean(header.Name))
		log.Debugf("Decompressing restore archive item: %s.", output)

		// Handle the fs op based on the header type.
		switch header.Typeflag {
		case tar.TypeDir:
			// Simply create the directory.
			err := os.MkdirAll(output, fileModeUserRWX)
			if err != nil && !errors.Is(err, os.ErrExist) {
				return fmt.Errorf("failed to create output directory(%s): %w", output, err)
			}

		case tar.TypeReg:
			// Get a writable handle to the restore target.
			decompressed, err := os.Create(output)
			if err != nil {
				return fmt.Errorf(
					"failed to open writable stream to output file(%s): %w",
					output, err,
				)
			}

			// Decompress the file to disk.
			//
			//nolint:gosec,G110 // Above, the '*os.File' is wrapped in a 'LimitReader'.
			n, err := io.Copy(decompressed, tr)
			if err != nil {
				return fmt.Errorf(
					"failed to decompress file(%s) during backup restoration: %w",
					output, err,
				)
			}
			// Make sure we wrote the expected content length.
			if n != header.Size {
				return fmt.Errorf(
					"encountered no error in restore archive file(%s) decompression, "+
						"but the archive's file size(%d) does not match what we wrote "+
						"to disk(%d)",
					output, header.Size, n,
				)
			}

			// Close the output file.
			err = decompressed.Close()
			if err != nil {
				return fmt.Errorf(
					"failed to close output file(%s) stream during restore: %w",
					output, err,
				)
			}
		}
	}
}
