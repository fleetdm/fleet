package filesystem

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

const fleetdInstallersPrefix = "fleetd-installers"

type fleetdInstallerNotFoundError struct{}

func (p fleetdInstallerNotFoundError) Error() string {
	return "fleetd installer not found"
}

func (p fleetdInstallerNotFoundError) IsNotFound() bool {
	return true
}

// FleetdInstallerStore implements fleet.FleetdInstallerStore using the local filesystem.
type FleetdInstallerStore struct {
	rootDir string
}

// NewFleetdInstallerStore creates a fleetd installer store using the local
// filesystem rooted at the provided rootDir.
func NewFleetdInstallerStore(rootDir string) (*FleetdInstallerStore, error) {
	dir := filepath.Join(rootDir, fleetdInstallersPrefix)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &FleetdInstallerStore{rootDir}, nil
}

// Get retrieves the requested fleetd installer from the local filesystem.
// The caller must close the returned reader.
func (s *FleetdInstallerStore) Get(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	path := s.pathForKey(key)
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, fleetdInstallerNotFoundError{}
		}
		return nil, 0, ctxerr.Wrap(ctx, err, "retrieving fleetd installer from filesystem store")
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "opening fleetd installer file from filesystem store")
	}
	return f, st.Size(), nil
}

// Put stores a fleetd installer in the local filesystem.
func (s *FleetdInstallerStore) Put(ctx context.Context, key string, content io.ReadSeeker) error {
	path := s.pathForKey(key)

	f, err := os.Create(path)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating fleetd installer file in filesystem store")
	}
	defer f.Close()

	if _, err := io.Copy(f, content); err != nil {
		return ctxerr.Wrap(ctx, err, "writing fleetd installer file in filesystem store")
	}
	if err := f.Close(); err != nil {
		return ctxerr.Wrap(ctx, err, "closing fleetd installer file in filesystem store")
	}
	return nil
}

// Exists checks if a fleetd installer exists in the filesystem for the key.
func (s *FleetdInstallerStore) Exists(ctx context.Context, key string) (bool, error) {
	path := s.pathForKey(key)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, ctxerr.Wrap(ctx, err, "looking up fleetd installer in filesystem store")
	}
	return true, nil
}

func (s *FleetdInstallerStore) pathForKey(key string) string {
	return filepath.Join(s.rootDir, fleetdInstallersPrefix, key)
}
