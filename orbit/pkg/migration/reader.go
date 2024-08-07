package migration

import (
	"errors"
	"os"
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
			return false, nil
		}

		return false, err
	}

	return true, nil
}
