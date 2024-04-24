package file

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/pkg/secure"
)

// ExtractInstallerMetadata extracts the software name and version from the
// installer file. The format of the installer is determined based on the
// extension of the filename.
func ExtractInstallerMetadata(filename string, b []byte) (name, version string, err error) {
	switch ext := filepath.Ext(filename); ext {
	case ".deb":
		return ExtractDebMetadata(b)
	case ".exe":
		return ExtractPEMetadata(b)
	case ".pkg":
		return ExtractXARMetadata(b)
	default:
		return "", "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

// Copy copies the file from srcPath to dstPath, using the provided permissions.
//
// Note that on Windows the permissions support is limited in Go's file functions.
func Copy(srcPath, dstPath string, perm os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open src for copy: %w", err)
	}
	defer src.Close()

	if err := secure.MkdirAll(filepath.Dir(dstPath), os.ModeDir|perm); err != nil {
		return fmt.Errorf("create dst dir for copy: %w", err)
	}

	dst, err := secure.OpenFile(dstPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("open dst for copy: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy src to dst: %w", err)
	}
	if err := dst.Sync(); err != nil {
		return fmt.Errorf("sync dst after copy: %w", err)
	}

	return nil
}

// Copy copies the file from srcPath to dstPath, using the permissions of the original file.
//
// Note that on Windows the permissions support is limited in Go's file functions.
func CopyWithPerms(srcPath, dstPath string) error {
	stat, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("get permissions for copy: %w", err)
	}

	return Copy(srcPath, dstPath, stat.Mode().Perm())
}

// Exists returns whether the file exists and is a regular file.
func Exists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("check file exists: %w", err)
	}

	return info.Mode().IsRegular(), nil
}
