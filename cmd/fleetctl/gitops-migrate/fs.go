package main

import (
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
)

const (
	fileModeUserRWX    = 0o700
	fileModeUserRW     = 0o600
	fileFlagsOverwrite = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
)

// We use this error to break the 'fsEnum' iterator if, for whatever reason, the
// caller breaks the iterator.
var (
	enumEOF = errors.New("EOF")
	sep     = string(os.PathSeparator)
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

func fsEnumRec(path string, yield func(File, error) bool) {
	stats, err := os.Lstat(path)
	switch {
	case err != nil:
		// Handle failed stats.
		yield(File{}, fmt.Errorf(
			"failed to stat path(%s) during directory enumeration: %w",
			path, err,
		))
		return

	case stats.IsDir():
		// Yield the directory.
		if !yield(File{Path: path, Stats: stats}, nil) {
			return
		}

		// Continue recursive directory enumeration.
		items, err := os.ReadDir(path)
		if err != nil {
			yield(File{}, fmt.Errorf(
				"failed to read directory(%s) during enumeration: %w",
				path, err,
			))
			return
		}

		// Iterate returned directory items, passing files through the yield fn and
		// recursing on directories.
		for _, item := range items {
			// Assemble the nested item's path.
			path := filepath.Join(path, item.Name())

			// Get the 'os.FileInfo' for this item.
			info, err := item.Info()
			if err != nil {
				yield(
					File{},
					fmt.Errorf(
						"failed to get file info for directory item(%s) during enumeration: %w",
						path, err,
					))
				return
			}

			if !item.IsDir() {
				if !yield(File{Path: path, Stats: info}, nil) {
					return
				}
			} else {
				fsEnumRec(path, yield)
			}
		}

	case !stats.IsDir():
		// Simply yield file items.
		if !yield(File{Path: path, Stats: stats}, nil) {
			return
		}

	default:
		panic("impossible")
	}
}
