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
	//
	// This is a verbose chunk of code, but it's important to check each error
	// here as the 'gzip.Writer' and 'tar.Writer' closures actually write critical
	// trailer data to the file stream.
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
		filePathRelative = strings.TrimPrefix(filePathRelative, string(os.PathSeparator))

		// Fix up the tar header.
		//
		// See 'tar.FileInfoHeader' docs for more on this.
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

		log.Info("compressing file", "to", filePathRelative)

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

func mkBackupDir() (string, error) {
	path, err := os.MkdirTemp(os.TempDir(), "fleet-gitops-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp backup directory: %w", err)
	}
	return path, nil
}
