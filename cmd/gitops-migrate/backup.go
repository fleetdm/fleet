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

	"github.com/fleetdm/fleet/v4/cmd/gitops-migrate/log"
)

// backup creates a backup of the path provided via 'from', to a gzipped tarball
// in the directory specified by 'to'.
func backup(ctx context.Context, from string, to string) (string, error) {
	// Resolve and validate the backup archive output path.
	output, err := resolveBackupTarget(to)
	if err != nil {
		return "", err
	}

	log.Info(
		"Performing Fleet GitOps file backup.",
		"Source", from,
		"Destination", output,
	)

	// Get a writable handle to the output archive file.
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
	//
	// This is a verbose chunk of code, but it's important to check each error
	// here as the 'gzip.Writer' and 'tar.Writer' closures actually write critical
	// trailer data to the file stream.
	defer func() {
		var errs error

		// Close the output archive tar writer.
		err = tw.Close()
		if err != nil {
			log.Errorf("Failed to close the backup archive tar writer: %s.", err)
		}
		errs = errors.Join(errs, err)

		// Close the output archive gzip writer.
		err = gz.Close()
		if err != nil {
			log.Errorf("Failed to close the backup archive gzip writer: %s.", err)
		}
		errs = errors.Join(errs, err)

		// Close the output archive file stream.
		err := f.Close()
		if err != nil {
			log.Errorf("Failed to close the backup archive file stream: %s.", err)
		}
		errs = errors.Join(errs, err)

		// We want a good, clean backup before we proceed with hijacking all the
		// targeted GitOps files. So, if we encounter any errors at all, close up
		// shop.
		if errs != nil {
			log.Fatal("Errors encountered in backup archive stream close, exiting.")
		}
	}()

	// Enumerate the file system, writing files to the tarball.
	//
	//nolint:wrapcheck // Not an external package error.
	for file, err := range fsEnum(from) {
		if err != nil {
			return "", fmt.Errorf(
				"encountered erorr in directory enumeration: %w",
				err,
			)
		}

		// Init the tar header for this file.
		header, err := tar.FileInfoHeader(file.Stats, file.Path)
		if err != nil {
			return "", fmt.Errorf(
				"failed to create tar header for file(%s): %w",
				file.Path, err,
			)
		}

		// Construct the relative file path.
		filePathRelative := strings.TrimPrefix(file.Path, from)
		filePathRelative = strings.TrimPrefix(
			filePathRelative,
			string(os.PathSeparator),
		)

		// Fix up the tar header.
		//
		// See 'tar.FileInfoHeader' docs for more on why we need to set this twice.
		header.Name = filePathRelative
		// If the item is a directory, zero the size.
		if file.Stats.IsDir() {
			header.Size = 0
		}

		// Write the tar header.
		err = tw.WriteHeader(header)
		if err != nil {
			return "", fmt.Errorf(
				"failed to write tar header to tar stream: %w", err,
			)
		}
		// If the item is a directory, no further to-do here.
		if file.Stats.IsDir() {
			continue
		}

		log.Debug(
			"Compressing file.",
			"File Path", file.Path,
			"Archive Path", filePathRelative,
		)

		// Get a readable handle to the file.
		//
		//nolint:gosec,G304 // 'path' is a trusted input.
		f, err := os.Open(file.Path)
		if err != nil {
			return "", fmt.Errorf(
				"failed to get a readable handle to file [%s] while performing backup: %w",
				file.Path, err,
			)
		}

		// Write the file content to the tar stream.
		n, err := io.Copy(tw, f)
		if err != nil {
			return "", fmt.Errorf(
				"failed to write file [%s] to tar stream during backup: %w",
				file.Path, err,
			)
		}

		// Ensure we wrote the content length we expect.
		if n != file.Stats.Size() {
			return "", fmt.Errorf(
				"encountered no error during backup, however the stat'd "+
					"file size(%d) didn't match the size we actually wrote(%d)",
				file.Stats.Size(), n,
			)
		}

		// Close the file stream.
		err = f.Close()
		if err != nil {
			return "", fmt.Errorf(
				"failed to close input file stream(%s) during backup: %w",
				file.Path, err,
			)
		}
	}

	return output, nil
}

// resolveBackupTarget evaluates the type of path 'path' points to.
//
// If 'path' is a FILE, the parent directory is created if it doesn't exist and
// the path is returned unchanged.
//
// If 'path' is a DIRECTORY, the directory is created if it doesn't exist, a
// random file name is generated and concatenated to the end of 'path',
// finally returning the result.
func resolveBackupTarget(path string) (string, error) {
	// Resolve the absolute file path.
	path, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf(
			"failed to identify absolute file path from [%s]: %w",
			path, err,
		)
	}

	// Attempt to identify if 'path' is a file or directory path by the presence
	// of a file extension.
	if filepath.Ext(path) == "" {
		// 'path' is a directory.
		return resolveBackupDirPath(path)
	}

	// 'path' is a file.
	return resolveBackupFilePath(path)
}

func resolveBackupDirPath(path string) (string, error) {
	// Create 'path' if it doesn't exist.
	err := os.MkdirAll(path, fileModeUserRWX)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return "", fmt.Errorf(
			"failed to create all or part of the provided path(%s): %w",
			path, err,
		)
	}

	now := time.Now()
	// Generate a timestamped file name.
	fileName := fmt.Sprintf(
		"fleet-gitops-backup-%d-%d-%d_%d-%d-%d.tar.gz",
		now.Month(), now.Day(), now.Year(), now.Hour(), now.Minute(), now.Second(),
	)

	// Concatenate the file name to the directory path and return.
	return filepath.Join(path, fileName), nil
}

func resolveBackupFilePath(path string) (string, error) {
	// Create 'path' if it doesn't exist.
	err := os.MkdirAll(filepath.Dir(path), fileModeUserRWX)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return "", fmt.Errorf(
			"failed to create all or part of the provided path(%s): %w",
			path, err,
		)
	}

	// Return the file path, unchanged.
	return path, nil
}

func mkBackupDir() (string, error) {
	path, err := os.MkdirTemp(os.TempDir(), "fleet-gitops-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp backup directory: %w", err)
	}
	return path, nil
}
