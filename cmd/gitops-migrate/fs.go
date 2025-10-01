package main

import (
	"fmt"
	"iter"
	"os"
	"path/filepath"
)

const (
	fileModeUserRW     = 0o600
	fileModeUserRWX    = 0o700
	fileFlagsOverwrite = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	fileFlagsReadWrite = os.O_RDWR
)

type File struct {
	Path  string
	Hash  string
	Stats os.FileInfo
}

// fsEnum is a very specific file system iterator for the Fleet GitOps migration
// tooling which yields all _files_ under 'root' in relative-path-form (relative
// to 'root').
//
// If the provided 'root' value is a file path, a single yield will occur
// before the iterator returns.
//
// If the provided 'root' value is a directory path, the directory will be
// recursively enumerated and produced.
func fsEnum(root string) iter.Seq2[File, error] {
	return func(yield func(File, error) bool) {
		// Send the path and yield fn through the recursive enumerator.
		fsEnumRec(root, yield)
	}
}

func fsEnumRec(path string, yield func(File, error) bool) bool {
	stats, err := os.Lstat(path)
	switch {
	case err != nil:
		// Handle failed stats.
		yield(File{}, fmt.Errorf(
			"failed to stat path(%s) during directory enumeration: %w",
			path, err,
		))
		return false

	case stats.IsDir():
		// Yield the directory.
		if !yield(File{Path: path, Stats: stats}, nil) {
			return false
		}

		// Continue recursive directory enumeration.
		items, err := os.ReadDir(path)
		if err != nil {
			yield(File{}, fmt.Errorf(
				"failed to read directory(%s) during enumeration: %w",
				path, err,
			))
			return false
		}

		// Iterate returned directory items, recursing on each.
		for _, item := range items {
			// Assemble the nested item's path.
			path := filepath.Join(path, item.Name())
			if !fsEnumRec(path, yield) {
				return false
			}
		}

	case !stats.IsDir():
		// Simply yield file items.
		if !yield(File{Path: path, Stats: stats}, nil) {
			return false
		}

	default:
		panic("impossible")
	}

	return true
}
