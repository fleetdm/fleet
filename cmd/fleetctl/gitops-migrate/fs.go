package main

import (
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
)

// We use this error to break the 'fsEnum' iterator if, for whatever reason, the
// caller breaks the iterator.
var (
	enumEOF = errors.New("EOF")
	sep     = string(os.PathSeparator)
)

// fsEnum is a very specific file system iterator for the Fleet GitOps migration
// tooling which yields all _files_ under 'root' in relative-path-form (relative
// to 'root').
//
// If the provided 'root' value is a file path, a single yield will occur
// before the iterator returns.
//
// If the provided 'root' value is a directory path, the directory will be
// recursively enumerated and produced.
func fsEnum(root string) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		// Send the path and yield fn through the recursive enumerator.
		fsEnumRec(root, yield)
	}
}

func fsEnumRec(path string, yield func(string, error) bool) {
	stats, err := os.Stat(path)
	switch {
	case err != nil:
		// Handle failed stats.
		yield("", fmt.Errorf(
			"failed to stat path(%s) during directory enumeration: %w",
			path, err,
		))
		return

	case stats.IsDir():
		// Continue recursive directory enumeration.
		items, err := os.ReadDir(path)
		if err != nil {
			yield("", fmt.Errorf(
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
			if !item.IsDir() {
				if !yield(path, nil) {
					return
				}
			} else {
				fsEnumRec(path, yield)
			}
		}

	case !stats.IsDir():
		// Simply yield file items.
		if !yield(path, nil) {
			return
		}

	default:
		panic("impossible")
	}
}
