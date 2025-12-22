package filesystem

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

const softwareTitleIconsPrefix = "software-title-icons"

type iconNotFoundError struct{}

var _ fleet.NotFoundError = (*iconNotFoundError)(nil)

func (p iconNotFoundError) Error() string {
	return "icon not found"
}

func (p iconNotFoundError) IsNotFound() bool {
	return true
}

type SoftwareTitleIconStore struct {
	rootDir string
}

// NewSoftwareTitleIconStore creates a software title icon store using the
// local filesystem rooted at the provided rootDir.
func NewSoftwareTitleIconStore(rootDir string) (*SoftwareTitleIconStore, error) {
	dir := filepath.Join(rootDir, softwareTitleIconsPrefix)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	return &SoftwareTitleIconStore{rootDir}, nil
}

// Get retrieves the requested software title icon from the local filesystem.
// It is important that the caller closes the reader when done.
func (s *SoftwareTitleIconStore) Get(ctx context.Context, iconID string) (io.ReadCloser, int64, error) {
	path := s.pathForIcon(iconID)
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, int64(0), iconNotFoundError{}
		}
		return nil, 0, ctxerr.Wrap(ctx, err, "retrieving software title icon from filesystem store")
	}

	sz := st.Size()
	f, err := os.Open(path)
	if err != nil {
		return nil, sz, ctxerr.Wrap(ctx, err, "opening software title icon file from filesystem store")
	}
	return f, sz, nil
}

// Put stores a software title icon in the local filesystem.
func (s *SoftwareTitleIconStore) Put(ctx context.Context, iconID string, content io.ReadSeeker) error {
	path := s.pathForIcon(iconID)

	f, err := os.Create(path)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating software title icon file in filesystem store")
	}
	defer f.Close()

	if _, err := io.Copy(f, content); err != nil {
		return ctxerr.Wrap(ctx, err, "writing software title icon file in filesystem store")
	}
	if err := f.Close(); err != nil {
		return ctxerr.Wrap(ctx, err, "closing software title icon file in filesystem store")
	}
	return nil
}

// Exists checks if a software title icon exists in the filesystem for the ID.
func (s *SoftwareTitleIconStore) Exists(ctx context.Context, iconID string) (bool, error) {
	path := s.pathForIcon(iconID)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, ctxerr.Wrap(ctx, err, "looking up software title icon in filesystem store")
	}
	return true, nil
}

func (s *SoftwareTitleIconStore) Cleanup(ctx context.Context, usedIconIDs []string, removeCreatedBefore time.Time) (int, error) {
	usedSet := make(map[string]struct{}, len(usedIconIDs))
	for _, id := range usedIconIDs {
		usedSet[id] = struct{}{}
	}

	baseDir := filepath.Join(s.rootDir, softwareTitleIconsPrefix)
	dirEnts, err := os.ReadDir(baseDir)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "listing software title icons in filesystem store")
	}

	var errs []error
	var count int
	for _, de := range dirEnts {
		if !de.Type().IsRegular() {
			continue
		}
		if _, isUsed := usedSet[de.Name()]; isUsed {
			continue
		}

		info, err := de.Info()
		if err != nil {
			return 0, ctxerr.Wrap(ctx, err, "get software title icon modtime in filesystem store")
		}
		if info.ModTime().After(removeCreatedBefore) {
			continue
		}
		if err := os.Remove(filepath.Join(baseDir, de.Name())); err != nil {
			errs = append(errs, err)
		} else {
			count++
		}
	}
	return count, ctxerr.Wrap(ctx, errors.Join(errs...), "delete unused software title icons")
}

func (s *SoftwareTitleIconStore) Sign(ctx context.Context, _ string) (string, error) {
	return "", ctxerr.New(ctx, "signing not supported for software title icons in filesystem store")
}

func (s *SoftwareTitleIconStore) pathForIcon(iconID string) string {
	return filepath.Join(s.rootDir, softwareTitleIconsPrefix, iconID)
}
