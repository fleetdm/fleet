package file

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/pkg/errors"
)

// Copy copies the file from srcPath to dstPath, using the provided permissions.
//
// Note that on Windows the permissions support is limited in Go's file functions.
func Copy(srcPath, dstPath string, perm os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return errors.Wrap(err, "open src for copy")
	}
	defer src.Close()

	if err := secure.MkdirAll(filepath.Dir(dstPath), os.ModeDir|perm); err != nil {
		return errors.Wrap(err, "create dst dir for copy")
	}

	dst, err := secure.OpenFile(dstPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return errors.Wrap(err, "open dst for copy")
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return errors.Wrap(err, "copy src to dst")
	}
	if err := dst.Sync(); err != nil {
		return errors.Wrap(err, "sync dst after copy")
	}

	return nil
}

// Copy copies the file from srcPath to dstPath, using the permissions of the original file.
//
// Note that on Windows the permissions support is limited in Go's file functions.
func CopyWithPerms(srcPath, dstPath string) error {
	stat, err := os.Stat(srcPath)
	if err != nil {
		return errors.Wrap(err, "get permissions for copy")
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
		return false, errors.Wrap(err, "check file exists")
	}

	return info.Mode().IsRegular(), nil
}
