//go:build !windows
// +build !windows

package augeas

import (
	"embed"
	"io"
	"os"
	"path/filepath"
)

//go:embed lenses
var lenses embed.FS

func CopyLenses(installPath string) (string, error) {
	outPath := filepath.Join(installPath, "lenses")

	err := os.RemoveAll(outPath)
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(outPath, 0o755)
	if err != nil {
		return "", err
	}
	entries, err := lenses.ReadDir("lenses")
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		src, err := lenses.Open(filepath.Join("lenses", entry.Name()))
		if err != nil {
			return "", err
		}
		dest, err := os.OpenFile(filepath.Join(outPath, entry.Name()), os.O_CREATE|os.O_WRONLY, 0o644) // nolint:gosec // G302
		if err != nil {
			return "", err
		}
		_, err = io.Copy(dest, src)
		if err != nil {
			return "", err
		}
		err = src.Close()
		if err != nil {
			return "", err
		}
		err = dest.Close()
		if err != nil {
			return "", err
		}
	}

	return outPath, nil
}
