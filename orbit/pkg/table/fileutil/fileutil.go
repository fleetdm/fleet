//go:build darwin
// +build darwin

// Package fileutil implements an extension osquery table to get information about a macOS file
package fileutil

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
)

const (
	colPath    = "path"
	colBinHash = "binary_sha256"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		// added here
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
			switch constraint.Operator {
			case table.OperatorLike:
				path = constraint.Expression
				wildcard = true
			case table.OperatorEquals:
				path = constraint.Expression
				wildcard = false
			default:
				return results, errors.New("invalid comparison for column 'path': supported comparisons are `=` and `LIKE`")
			}
		}
	} else {
		return results, errors.New("missing `path` constraint: provide a `path` in the query's `WHERE` clause")
	}

	processed, err := processFile(path, wildcard)
	if err != nil {
		return results, err
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
			return nil, err
		}
		for _, file := range files {
			// TODO: confirm file is valid for hash, e.g. not a directory?

			hash, err := computeFileSHA256(file)
			if err != nil {
				return nil, err
			}
			output = append(output, fileInfo{Path: file, BinSha256: hash})
		}
	} else {
		hash, err := computeFileSHA256(path)
		if err != nil {
			return nil, err
		}
		output = append(output, fileInfo{Path: path, BinSha256: hash})
	}

	return output, nil
}

func computeFileSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
