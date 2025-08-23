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
	"strings"
	"time"
)

const (
	fileModeUserRWX    = 0o700
	fileModeUserRW     = 0o600
	fileFlagsOverwrite = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
)

// backup creates a backup of the path provided via 'from', to a gzipped tarball
// in the directory specified by 'to'.
func backup(ctx context.Context, from string, to string) (string, error) {
	log := LoggerFromContext(ctx)

	// Construct the full output archive path.
	now := time.Now()
	output := filepath.Join(
		to, fmt.Sprintf(
			"fleet-gitops-backup-%d-%d-%d_%d-%d-%d.tar.gz",
			now.Month(), now.Day(), now.Year(), now.Hour(), now.Minute(), now.Second(),
		),
	)

	log.Info(
		"performing Fleet GitOps file backup",
		"source", from,
		"destination", output,
	)

	// Create any requisite parent directories if necessary.
	err := os.MkdirAll(filepath.Dir(to), fileModeUserRWX)
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf(
				"failed to create backup output directory(%s): %w",
				to, err,
			)
		}
	}

	// Get a writable handle to the output archive.
	//
	//nolint:gosec,G304 // 'output' is a trusted input.
	f, err := os.OpenFile(output, fileFlagsOverwrite, fileModeUserRW)
	if err != nil {
		return "", fmt.Errorf(
			"failed to get a writable handle to archive file(%s): %w",
			output, err,
		)
	}
	// Init the gzip writer.
	gz := gzip.NewWriter(f)
	// Init the tar writer.
	tw := tar.NewWriter(gz)
	// Defer stream closure, in reverse order.
	defer func() {
		var errs error

		// Close the output archive tar writer.
		err = tw.Close()
		if err != nil {
			log.Error(
				"failed to close the backup archive tar writer",
				"error", err,
			)
		}
		errs = errors.Join(errs, err)

		// Close the output archive gzip writer.
		err = gz.Close()
		if err != nil {
			log.Error(
				"failed to close the backup archive gzip writer",
				"error", err,
			)
		}
		errs = errors.Join(errs, err)

		// Close the output archive file stream.
		err := f.Close()
		if err != nil {
			log.Error(
				"failed to close the backup archive file stream",
				"error", err,
			)
		}
		errs = errors.Join(errs, err)

		// We want a good, clean backup before we proceed with hijacking all the
		// targeted GitOps files. So, if we encounter any errors at all, close up
		// shop.
		if errs != nil {
			log.Error("errors encountered in backup archive stream close, exiting")
			os.Exit(1)
		}
	}()

	// Walk the file system, writing files to the tarball.
	//
	//nolint:wrapcheck // Not an external package error.
	return output, filepath.Walk(from, func(path string, stats os.FileInfo, err error) error {
		// Init the tar header for this file.
		header, err := tar.FileInfoHeader(stats, path)
		if err != nil {
			return fmt.Errorf(
				"failed to create tar header for file(%s): %w",
				path, err,
			)
		}

		// Construct the relative file path.
		filePathRelative := strings.TrimPrefix(path, from)
		filePathRelative = strings.TrimPrefix(filePathRelative, string(os.PathSeparator))

		// Fix up the tar header.
		//
		// See 'tar.FileInfoHeader' docs for more on this.
		header.Name = filePathRelative
		// If the item is a directory, zero the size.
		if stats.IsDir() {
			header.Size = 0
		}

		// Write the tar header.
		err = tw.WriteHeader(header)
		if err != nil {
			return fmt.Errorf(
				"failed to write tar header to tar stream: %w", err,
			)
		}
		// If the item is a directory, no further to-do here.
		if stats.IsDir() {
			return nil
		}

		log.Info("compressing file", "to", filePathRelative)

		// Get a readable handle to the file.
		//
		//nolint:gosec,G304 // 'path' is a trusted input.
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf(
				"failed to get a readable handle to file [%s] while performing backup: %w",
				path, err,
			)
		}
		defer func() {
			err := f.Close()
			if err != nil {
				log.Error(
					"encountered error closing input file stream during backup",
					"file", path,
					"error", err,
				)
				os.Exit(1)
			}
		}()

		// Write the file content to the tar stream.
		n, err := io.Copy(tw, f)
		if err != nil {
			return fmt.Errorf(
				"failed to write file [%s] to tar stream during backup: %w",
				path, err,
			)
		}
		if n != stats.Size() {
			return fmt.Errorf(
				"encountered no error during backup, however the stat'd "+
					"file size(%d) didn't match the size we actually wrote(%d)",
				stats.Size(), n,
			)
		}

		return nil
	})
}

// restore restores a Fleet GitOps backup from the provided tarball.
//
// 'from' must be a path to a gzipped tarball.
//
// If the 'to' path is not provided, the current working directory will be used.
func restore(ctx context.Context, from string, to string) error {
	log := LoggerFromContext(ctx)

	// Set a default 'to' path, if necessary.
	if to == "" {
		log.Debug(
			"found no 'to' path for restore operation, defaulting to " +
				"current working directory",
		)
		to = "."
	}
	log.Info("performing Fleet GitOps restore", "from", from, "to", to)

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
	//nolint:gosec,G304 // 'from' is a trusted input.
	f, err := os.Open(from)
	if err != nil {
		return fmt.Errorf(
			"failed to get a readable handle to restore archive(%s): %w",
			from, err,
		)
	}
	// Wrap the file stream in a gzip reader.
	gz, err := gzip.NewReader(f)
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
				"failed to close restore archive gzip reader",
				"error", err,
			)
		}
		errs = errors.Join(errs, err)

		// Close the restore archive file stream.
		err = f.Close()
		if err != nil {
			log.Error(
				"failed to close restore archive file stream",
				"error", err,
			)
		}
		errs = errors.Join(errs, err)

		if errs != nil {
			log.Error("errors encountered in restore archive stream close, exiting")
			os.Exit(1)
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
		log.Info("decompressing restore archive item", "to", output)

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
			//
			//nolint:gosec,G304 // 'output' is sanitized above.
			decompressed, err := os.Create(output)
			if err != nil {
				return fmt.Errorf(
					"failed to open writable stream to output file(%s): %w",
					output, err,
				)
			}

			// Decompress the file to disk.
			//
			//nolint:gosec,G110 // These are archives [hopefully] we created.
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
						"but the archive's file size(%d) does not match what we wrote to disk(%d)",
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
