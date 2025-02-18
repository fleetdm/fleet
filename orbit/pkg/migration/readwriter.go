package migration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
)

type ReadWriter struct {
	Path     string
	FileName string
}

func NewReadWriter(path, filename string) *ReadWriter {
	return &ReadWriter{
		Path:     path,
		FileName: filepath.Join(path, filename),
	}
}

// SetMigrationFile sets `typ` in the file used to track MDM migration type. This overwrites the
// file if it exists.
func (rw *ReadWriter) SetMigrationFile(typ string) error {
	_, err := rw.read()
	switch {
	case err == nil:
		// ensure the file is readable by other processes
		if err := rw.setChmod(); err != nil {
			return fmt.Errorf("loading migration file, chmod %q: %w", rw.Path, err)
		}
	case errors.Is(err, os.ErrNotExist):
		if err := os.MkdirAll(rw.Path, constant.DefaultDirMode); err != nil {
			return fmt.Errorf("creating directory for migration file: %w", err)
		}
		if err := os.WriteFile(rw.FileName, []byte(typ), constant.DefaultWorldReadableFileMode); err != nil {
			return fmt.Errorf("writing migration file: %w", err)
		}

	default:
		return fmt.Errorf("load migration file %q: %w", rw.Path, err)
	}
	return nil
}

// RemoveFile removes the file used for tracking the MDM migration type.
func (rw *ReadWriter) RemoveFile() error {
	if err := os.Remove(rw.FileName); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// that's ok, noop
			return nil
		}

		return fmt.Errorf("removing migration file: %w", err)
	}

	return nil
}

// GetMigrationType returns the contents of the MDM migration file. The contents say what type of
// migration it is.
func (rw *ReadWriter) GetMigrationType() (string, error) {
	data, err := rw.read()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}

		return "", err
	}

	return data, nil
}

// FileExists returns whether or not the MDM migration file exists on this host.
func (rw *ReadWriter) FileExists() (bool, error) {
	_, err := os.Stat(rw.FileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// DirExists returns whether or not the directory where the MDM migration file is stored exists.
func (rw *ReadWriter) DirExists() (bool, error) {
	_, err := os.Stat(rw.FileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (rw *ReadWriter) read() (string, error) {
	data, err := os.ReadFile(rw.FileName)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (rw *ReadWriter) setChmod() error {
	return os.Chmod(rw.FileName, constant.DefaultWorldReadableFileMode)
}

func (rw *ReadWriter) NewFileWatcher() FileWatcher {
	return &fileWatcher{rw: rw}
}

type FileWatcher interface {
	GetMigrationType() (string, error)
	FileExists() (bool, error)
	DirExists() (bool, error)
}

type fileWatcher struct {
	rw *ReadWriter
}

// GetMigrationType returns the contents of the MDM migration file which indicate what type of
// migration it is.
func (r *fileWatcher) GetMigrationType() (string, error) {
	return r.rw.GetMigrationType()
}

// FileExists returns whether or not the MDM migration file exists on this host.
func (r *fileWatcher) FileExists() (bool, error) {
	return r.rw.FileExists()
}

// DirExists returns whether or not the directory where the MDM migration file is stored exists.
func (r *fileWatcher) DirExists() (bool, error) {
	return r.rw.DirExists()
}

// Dir returns the path to the directory where the MDM migration file is stored. This path should be
// ~/Library/Caches/com.fleetdm.orbit
func Dir() (string, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user's home directory: %w", err)
	}

	return filepath.Join(homedir, "Library/Caches/com.fleetdm.orbit"), nil
}
