package filesystem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// Put stores a software title icon atomically: contents are streamed to a
// temp file, fsync'd, then renamed into place. A crash mid-write leaves the
// temp file for Cleanup to reap rather than a truncated final file.
func (s *SoftwareTitleIconStore) Put(ctx context.Context, iconID string, content io.ReadSeeker) error {
	finalPath := s.pathForIcon(iconID)

	tmp, err := os.CreateTemp(filepath.Dir(finalPath), ".tmp-icon-*")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating temp software title icon file in filesystem store")
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once Rename has consumed it

	if _, err := io.Copy(tmp, content); err != nil {
		_ = tmp.Close()
		return ctxerr.Wrap(ctx, err, "writing software title icon file in filesystem store")
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return ctxerr.Wrap(ctx, err, "syncing software title icon file in filesystem store")
	}
	if err := tmp.Close(); err != nil {
		return ctxerr.Wrap(ctx, err, "closing software title icon file in filesystem store")
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return ctxerr.Wrap(ctx, err, "renaming software title icon file in filesystem store")
	}
	return nil
}

// Exists reports whether the icon for iconID is present and intact. A file
// that exists but is empty or whose SHA-256 doesn't match iconID is treated
// as not-present, so callers fall through to a fresh upload rather than
// trusting corrupted bytes.
func (s *SoftwareTitleIconStore) Exists(ctx context.Context, iconID string) (bool, error) {
	path := s.pathForIcon(iconID)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, ctxerr.Wrap(ctx, err, "looking up software title icon in filesystem store")
	}
	if info.Size() == 0 {
		return false, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "opening software title icon for hash verification")
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, ctxerr.Wrap(ctx, err, "hashing software title icon for verification")
	}
	if hex.EncodeToString(h.Sum(nil)) != iconID {
		return false, nil
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

func (s *SoftwareTitleIconStore) Sign(ctx context.Context, _ string, _ time.Duration) (string, error) {
	return "", ctxerr.New(ctx, "signing not supported for software title icons in filesystem store")
}

func (s *SoftwareTitleIconStore) pathForIcon(iconID string) string {
	return filepath.Join(s.rootDir, softwareTitleIconsPrefix, iconID)
}
