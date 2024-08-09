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

func (rw *ReadWriter) GetMigrationType() (string, error) {
	data, err := rw.read()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
	}

	return data, nil
}

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

func MigrationFileDir() (string, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user's home directory: %w", err)
	}

	return filepath.Join(homedir, "Library/Caches/com.fleetdm.orbit"), nil
}
