//go:build darwin
// +build darwin

// Package fileutil implements an extension osquery table to get information about a macOS file
package fileutil

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

const (
	colPath    = "path"
	colBinHash = "binary_sha256"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn(colPath),
		table.TextColumn(colBinHash),
	}
}

// Generate is called to return the results for the table at query time.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	path := ""
	wildcard := false

	var results []map[string]string

	if constraintList, present := queryContext.Constraints[colPath]; present {
		// 'path' is in the where clause
		for _, constraint := range constraintList.Constraints {
			path = constraint.Expression

			switch constraint.Operator {
			case table.OperatorLike:
				path = constraint.Expression
				wildcard = true
			case table.OperatorEquals:
				path = constraint.Expression
				wildcard = false
			}
		}
	} else {
		return results, errors.New("missing `path` constraint: provide a `path` in the query's `WHERE` clause")
	}

	processed, err := processFile(path, wildcard)
	if err != nil {
		return nil, err
	}

	for _, res := range processed {
		results = append(results, map[string]string{
			colPath:    res.Path,
			colBinHash: res.BinSha256,
		})
	}

	return results, nil
}

type fileInfo struct {
	Path      string
	BinSha256 string
}

func processFile(path string, wildcard bool) ([]fileInfo, error) {
	var output []fileInfo

	if wildcard {
		replacedPath := strings.ReplaceAll(path, "%", "*")

		files, err := filepath.Glob(replacedPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve filepaths for incoming path: %w", err)
		}
		for _, f := range files {
			binPath := getExecutablePath(context.Background(), f)

			hash, err := computeFileSHA256(binPath)
			if err != nil {
				return nil, fmt.Errorf("computing bin sha256 from wildcard path: %w", err)
			}

			output = append(output, fileInfo{Path: binPath, BinSha256: hash})
		}
	} else {
		binPath := getExecutablePath(context.Background(), path)

		hash, err := computeFileSHA256(binPath)
		if err != nil {
			return nil, fmt.Errorf("computing bin sha256 from specific path: %w", err)
		}
		output = append(output, fileInfo{Path: path, BinSha256: hash})
	}
	return output, nil
}

func computeFileSHA256(filePath string) (string, error) {
	if filePath == "" {
		log.Warn().Msg("empty path provided, returning empty hash")
		return "", nil
	}
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("couldn't open filepath: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("computing hash: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func getExecutablePath(ctx context.Context, path string) string {
	if strings.HasSuffix(path, ".app") {
		// Use defaults to read CFBundleExecutable from Info.plist
		infoPlistPath := path + "/Contents/Info.plist"
		output, err := exec.CommandContext(ctx, "/usr/bin/defaults", "read", infoPlistPath, "CFBundleExecutable").Output()
		if err != nil {
			// lots of helper .app bundles nested within parent .apps seem to have invalid Info.plists - warn and continue
			log.Warn().Err(err).Str("path", path).Msg("failed to read CFBundleExecutable from Info.plist, returning empty executable path")
			return ""
		}

		executableName := strings.TrimSpace(string(output))
		if executableName == "" {
			return ""
		}

		return path + "/Contents/MacOS/" + executableName
	}

	// For non-app paths, check if it's a regular file (binary)
	info, err := os.Stat(path)
	if err != nil {
		log.Warn().Err(err).Str("path", path).Msg("couldn't get FileInfo")
		return ""
	}

	// Only return the path if it's a regular file (not a directory)
	if info.Mode().IsRegular() {
		return path
	}

	log.Warn().Err(err).Str("path", path).Msg("path is not a regular file nor a .app bundle")
	return ""
}
