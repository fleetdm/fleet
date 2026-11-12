package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
)

// existingOutput is a previously generated output file. Its version block is the pin for runs
// without -update.
type existingOutput struct {
	Version profilevalidation.Version
	Data    []byte
}

// readOutput returns the output at path, or nil if it's missing or empty. Only the version block is
// decoded, so an output from an older generator still pins.
func readOutput(path string) (*existingOutput, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && len(data) == 0) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var header struct {
		Version profilevalidation.Version `json:"version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, fmt.Errorf("read version from %s: %w", path, err)
	}
	return &existingOutput{Version: header.Version, Data: data}, nil
}

// writeOutput writes schema to path. generated_at moves only when the content does, so rerunning
// against the same pins is byte-identical. The document must pass profilevalidation.Parse first, so a
// reorganized upstream fails generation instead of shipping a shrunken schema; on any failure the
// existing file is left untouched.
func writeOutput(ctx context.Context, logger *slog.Logger, path string, schema *profilevalidation.ProfileValidationSchema, existing *existingOutput, now time.Time) error {
	schema.FormatVersion = profilevalidation.FormatVersion

	if existing != nil {
		schema.Version.GeneratedAt = existing.Version.GeneratedAt
		data, err := encodeSchema(schema)
		if err != nil {
			return err
		}
		if bytes.Equal(data, existing.Data) {
			if _, err := profilevalidation.Parse(data); err != nil {
				return fmt.Errorf("existing output fails the schema checks: %w", err)
			}
			logger.InfoContext(ctx, "output unchanged; keeping generated_at", "out", path, "generated_at", existing.Version.GeneratedAt)
			return nil
		}
		logger.InfoContext(ctx, "output content changed", "out", path)
	}

	schema.Version.GeneratedAt = now.UTC().Truncate(time.Second)
	data, err := encodeSchema(schema)
	if err != nil {
		return err
	}
	if _, err := profilevalidation.Parse(data); err != nil {
		return fmt.Errorf("generated schema fails its own checks, not writing %s: %w", path, err)
	}
	if err := writeFileAtomic(path, data); err != nil {
		return err
	}
	logger.InfoContext(ctx, "wrote output", "out", path, "bytes", len(data), "apple_release", schema.Version.Apple.Release,
		"apple_commit", schema.Version.Apple.Commit, "generated_at", schema.Version.GeneratedAt)
	return nil
}

// encodeSchema returns indented JSON with sorted map keys. HTML escaping is off so format regexes
// stay readable ("<" rather than "\u003c").
func encodeSchema(schema *profilevalidation.ProfileValidationSchema) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(schema); err != nil {
		return nil, fmt.Errorf("encode schema: %w", err)
	}
	return buf.Bytes(), nil
}

// writeFileAtomic writes via a temp file in the same directory and a rename, so readers and a
// failed run never see a partial file.
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op once renamed
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write %s: %w", tmp.Name(), err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
