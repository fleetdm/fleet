package main

import (
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
)

// We use this error to break the 'fsEnum' iterator if, for whatever reason, the
// caller breaks the iterator.
var enumEOF = errors.New("EOF")
var sep = string(os.PathSeparator)

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
		// Stat 'root', we need to know if it's a directory or a file.
		stats, err := os.Stat(root)
		if err != nil {
			yield("", fmt.Errorf("failed to stat root path(%s): %w", root, err))
			return
		}

		// If 'root' is a file, yield once and return.
		if !stats.IsDir() {
			yield(root, nil)
			return
		}

		// Otherwise, recursively enumerate 'root'.
		err = filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf(
					"error encountered in directory enumeration(%s): %w",
					path, err,
				)
			}

			// Skip the 'root' (we only want to yield files anyway).
			if path == root {
				return nil
			}

			// Skip directories.
			if info.IsDir() {
				return nil
			}

			// Yield all other items.
			//
			// TODO: Do we want to filter down to '.yaml'/'.yml' files only?

			// Construct the relative path.
			// filePathRelative := strings.TrimPrefix(path, root)
			// filePathRelative = strings.TrimPrefix(
			// 	filePathRelative,
			// 	string(os.PathSeparator),
			// )

			// Yield the file path.
			if !yield(path, nil) {
				return enumEOF
			}

			return nil
		})

		// Address any Walk errors.
		if err != nil {
			// If we caught an 'enumEOF' return cleanly.
			if errors.Is(err, enumEOF) {
				return
			}
			// Otherwise, produce the error.
			yield("", err)
		}
	}
}
