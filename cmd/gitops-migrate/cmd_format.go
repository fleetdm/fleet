package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/cmd/gitops-migrate/limit"
	"github.com/fleetdm/fleet/v4/cmd/gitops-migrate/log"
	"gopkg.in/yaml.v3"
)

const cmdFormat = "format"

func cmdFormatExec(ctx context.Context, args Args) error {
	// Expect the 'input' path (root to begin formatting _from_) as the first
	// positional arg.
	if len(args.Commands) == 0 {
		return errors.New("received no path to 'format' command")
	}
	fmtPath := args.Commands[0]

	log.Info("Formatting GitOps YAML files.")

	// Init a limiter with a concurrency allowance equal to number of host machine
	// logical processors.
	//
	//nolint:gosec,G115 // Not until we have 2147483648-core CPUs!
	l := limit.New(int32(runtime.NumCPU()))

	// Enumerate the file system, format all YAML files.
	pass := int32(0)
	fail := int32(0)
	for file, err := range fsEnum(fmtPath) {
		// Handle errors.
		if err != nil {
			return fmt.Errorf("encountered error in file system enumeration: %w", err)
		}

		// Skip directories.
		if file.Stats.IsDir() {
			log.Debugf("Skipping [%s]: item is a directory, not file.", file.Path)
			continue
		}

		// Ignore non-YAML files.
		lowerPath := strings.ToLower(file.Path)
		if !strings.HasSuffix(lowerPath, ".yml") &&
			!strings.HasSuffix(lowerPath, ".yaml") {
			log.Debugf("Skipping [%s]: file is not YAML.", file.Path)
			continue
		}

		l.Go(func() {
			log.Infof("Formatting file: %s.", file.Path)
			err := formatFile(file.Path)
			if err != nil {
				log.Error(
					"Failed to format file.",
					"File", file.Path,
					"Error", err,
				)
				atomic.AddInt32(&fail, 1)
			} else {
				atomic.AddInt32(&pass, 1)
			}
		})
	}

	// Wait for formatting to complete with a 10-second timeout.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := l.WaitContext(ctx); err != nil {
		return errors.New("hung Goroutine in limiter")
	}

	log.Info("Format run complete.", "Successful", pass, "Failed", fail)
	if fail > 0 {
		return errors.New("encountered format job failures")
	}

	return nil
}

func formatFile(path string) error {
	// Get a read-writable handle to the file.
	f, err := os.OpenFile(path, fileFlagsReadWrite, 0)
	if err != nil {
		return fmt.Errorf(
			"failed to get a read-writable handle to file(%s): %w",
			path, err,
		)
	}
	defer func() { _ = f.Close() }()

	// Deserialize the content.
	//
	// We have some YAML files in which the root data structure is an object and
	// some which are arrays. To accommodate for this, we first attempt a decode
	// to a map. If this fails, we swap in a slice and try again*.
	//
	// * This is also why we wrap the map into an interface before we attempt the
	// decode.
	m := any(make(map[string]any))
	err = yaml.NewDecoder(f).Decode(m)
	if err != nil {
		// Reset read position.
		_, err := f.Seek(0, io.SeekStart)
		if err != nil {
			return fmt.Errorf(
				"failed to seek to file start for second decode attempt(%s): %w",
				path, err,
			)
		}

		// Init the slice, wrap its address into an interface.
		slice := []any{}
		m = &slice

		// Re-attempt the decode.
		err = yaml.NewDecoder(f).Decode(m)
		if err != nil {
			return fmt.Errorf("failed to decode YAML file(%s): %w", path, err)
		}
	}

	// Reset the '*os.File' read position.
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to start of file(%s): %w", path, err)
	}

	// Re-serialize the content.
	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	err = enc.Encode(m)
	if err != nil {
		return fmt.Errorf("failed to re-encode the YAML file(%s): %w", path, err)
	}

	// Identify our current position in the file.
	n, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf(
			"failed to determine YAML file content length(%s): %w",
			path, err,
		)
	}

	// Truncate the file at the current position.
	err = f.Truncate(n)
	if err != nil {
		return fmt.Errorf(
			"failed to truncate the formatted file(%s): %w",
			path, err,
		)
	}

	return nil
}
