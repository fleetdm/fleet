package token

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/google/uuid"
)

type ReadWriter struct {
	*Reader
}

func NewReadWriter(path string) *ReadWriter {
	return &ReadWriter{
		Reader: &Reader{Path: path},
	}
}

// LoadOrGeneratre tries to read a token file from disk,
// and if it doesn't exist, generates a new one.
func (rw *ReadWriter) LoadOrGenerate() error {
	_, err := rw.Read()
	switch {
	case err == nil:
		// ensure the file is readable by other processes, old versions of Orbit
		// used to chmod this file with 0o600
		if err := rw.setChmod(); err != nil {
			return fmt.Errorf("loading token file, chmod %q: %w", rw.Path, err)
		}
	case errors.Is(err, os.ErrNotExist):
		if err := rw.Rotate(); err != nil {
			return fmt.Errorf("rotating token on generation: %w", err)
		}
	default:
		return fmt.Errorf("load identifier file %q: %w", rw.Path, err)
	}
	return nil
}

// Rotate assigns a new value to the token and writes it to disk.
func (rw *ReadWriter) Rotate() error {
	id, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("generate identifier: %w", err)
	}

	err = os.WriteFile(rw.Path, []byte(id.String()), constant.DefaultWorldReadableFileMode)
	if err != nil {
		return fmt.Errorf("write identifier file %q: %w", rw.Path, err)
	}

	// ensure the file is readable by other processes, os.WriteFile does not
	// modify permissions if the file already exists
	if err := rw.setChmod(); err != nil {
		return fmt.Errorf("write identifier file, chmod %q: %w", rw.Path, err)
	}

	// ensure the `mtime` is updated, we have seen tests fail in some versions of
	// Ubuntu because this value is not update when the file is written
	if err = os.Chtimes(rw.Path, time.Now(), time.Now()); err != nil {
		return fmt.Errorf("set mtime of identifier file %q: %w", rw.Path, err)
	}

	// make sure we can read the token, and cache the value
	if _, err = rw.Read(); err != nil {
		return fmt.Errorf("read identifier file %q: %w", rw.Path, err)
	}

	return nil
}

func (rw *ReadWriter) setChmod() error {
	return os.Chmod(rw.Path, constant.DefaultWorldReadableFileMode)
}
