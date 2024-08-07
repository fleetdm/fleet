package migration

import (
	"errors"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/rs/zerolog/log"
)

type Reader struct {
	Path string
}

func (r *Reader) read() (string, error) {
	data, err := os.ReadFile(r.Path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (r *Reader) GetMigrationType() (string, error) {
	data, err := r.read()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
	}

	return data, nil
}

func (r *Reader) FileExists() (bool, error) {
	_, err := os.Stat(r.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Debug().Msg("JVE_LOG: migration file not found")
			return false, nil
		}

		return false, err
	}

	return true, nil
}

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

func (rw *ReadWriter) setChmod() error {
	return os.Chmod(rw.Path, constant.DefaultWorldReadableFileMode)
}
