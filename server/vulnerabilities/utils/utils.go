package utils

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// LatestFile returns the path of 'fileName' in 'dir' if the file exists, otherwise it will
// return the most recent file (based on the timestamp contained in 'fileName').
func LatestFile(fileName string, dir string) (string, error) {
	target := filepath.Join(dir, fileName)
	ext := filepath.Ext(target)

	switch _, err := os.Stat(target); {
	case err == nil:
		return target, nil
	case errors.Is(err, fs.ErrNotExist):
		files, err := os.ReadDir(dir)
		if err != nil {
			return "", err
		}

		prefix := strings.Split(fileName, "-")[0]
		var latest os.FileInfo
		for _, f := range files {
			if strings.HasPrefix(f.Name(), prefix) && strings.HasSuffix(f.Name(), ext) {
				info, err := f.Info()
				if err != nil {
					continue
				}
				if latest == nil || info.ModTime().After(latest.ModTime()) {
					latest = info
				}
			}
		}
		if latest == nil {
			return "", fmt.Errorf("file not found '%s' in '%s'", fileName, dir)
		}
		return filepath.Join(dir, latest.Name()), nil
	default:
		return "", fmt.Errorf("failed to stat %q: %w", target, err)
	}
}
