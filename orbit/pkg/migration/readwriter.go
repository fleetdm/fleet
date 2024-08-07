package migration

import (
	"errors"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
)

type ReadWriter struct {
	*Reader
}

func NewReadWriter(path string) *ReadWriter {
	return &ReadWriter{
		Reader: &Reader{Path: path},
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
		if err := os.WriteFile(rw.Path, []byte(typ), constant.DefaultWorldReadableFileMode); err != nil {
			return fmt.Errorf("writing migration file: %w", err)
		}

	default:
		return fmt.Errorf("load migration file %q: %w", rw.Path, err)
	}
	return nil
}

func (rw *ReadWriter) RemoveFile() error {
	if err := os.Remove(rw.Path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// that's ok, noop
			return nil
		}

		return fmt.Errorf("removing migration file: %w", err)
	}

	return nil
}

func (rw *ReadWriter) setChmod() error {
	return os.Chmod(rw.Path, constant.DefaultWorldReadableFileMode)
}
