package token

import (
	"errors"
	"fmt"
	"os"

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
		// OK
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

	err = os.WriteFile(rw.Path, []byte(id.String()), constant.DefaultSystemdUnitMode)
	if err != nil {
		return fmt.Errorf("write identifier file %q: %w", rw.Path, err)
	}

	return nil
}
