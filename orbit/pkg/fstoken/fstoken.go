package fstoken

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/google/uuid"
)

type Token struct {
	Path        string
	token       string
	cachedMtime time.Time
}

func New(path string) *Token {
	return &Token{filepath.Join(path, "identifier"), "", time.Unix(0, 0)}
}

func (u *Token) Get() (string, error) {
	info, err := os.Stat(u.Path)
	if err != nil {
		return "", err
	}

	mtime := info.ModTime()

	switch {
	case u.token == "":
	case mtime.Unix() != u.cachedMtime.Unix():
		if err = u.Refetch(); err != nil {
			return "", err
		}
	}

	return u.token, nil
}

func (u *Token) Refetch() error {
	f, err := os.ReadFile(u.Path)
	if err != nil {
		return err
	}
	u.token = string(f)
	return nil
}

func (u *Token) Generate() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("generate identifier: %w", err)
	}
	if err := os.WriteFile(u.Path, []byte(id.String()), constant.DefaultFileMode); err != nil {
		return "", fmt.Errorf("write identifier file %q: %w", u.Path, err)
	}
	return id.String(), nil
}
