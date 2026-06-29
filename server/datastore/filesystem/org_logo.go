package filesystem

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

const orgLogosPrefix = "org-logos"

type orgLogoNotFoundError struct{}

var _ fleet.NotFoundError = (*orgLogoNotFoundError)(nil)

func (orgLogoNotFoundError) Error() string    { return "org logo not found" }
func (orgLogoNotFoundError) IsNotFound() bool { return true }

type OrgLogoStore struct {
	rootDir string
}

func NewOrgLogoStore(rootDir string) (*OrgLogoStore, error) {
	dir := filepath.Join(rootDir, orgLogosPrefix)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &OrgLogoStore{rootDir: rootDir}, nil
}

func (s *OrgLogoStore) pathFor(mode fleet.OrgLogoMode) string {
	return filepath.Join(s.rootDir, orgLogosPrefix, string(mode))
}

func (s *OrgLogoStore) Get(ctx context.Context, mode fleet.OrgLogoMode) (io.ReadCloser, int64, error) {
	p := s.pathFor(mode)
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, orgLogoNotFoundError{}
		}
		return nil, 0, ctxerr.Wrap(ctx, err, "open org logo")
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, 0, ctxerr.Wrap(ctx, err, "stat org logo")
	}
	return f, st.Size(), nil
}

func (s *OrgLogoStore) Put(ctx context.Context, mode fleet.OrgLogoMode, content io.ReadSeeker) error {
	p := s.pathFor(mode)
	f, err := os.Create(p)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "create org logo file")
	}
	defer f.Close()
	if _, err := io.Copy(f, content); err != nil {
		return ctxerr.Wrap(ctx, err, "write org logo file")
	}
	if err := f.Close(); err != nil {
		return ctxerr.Wrap(ctx, err, "close org logo file")
	}
	return nil
}

func (s *OrgLogoStore) Delete(ctx context.Context, mode fleet.OrgLogoMode) error {
	p := s.pathFor(mode)
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return ctxerr.Wrap(ctx, err, "remove org logo file")
	}
	return nil
}

func (s *OrgLogoStore) Exists(ctx context.Context, mode fleet.OrgLogoMode) (bool, error) {
	p := s.pathFor(mode)
	if _, err := os.Stat(p); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, ctxerr.Wrap(ctx, err, "stat org logo")
	}
	return true, nil
}
