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

const softwareInstallersPrefix = "software-installers"

type installerNotFoundError struct{}

var _ fleet.NotFoundError = (*installerNotFoundError)(nil)

func (p installerNotFoundError) Error() string {
	return "installer not found"
}

func (p installerNotFoundError) IsNotFound() bool {
	return true
}

type SoftwareInstallerStore struct {
	rootDir string
}

// NewSoftwareInstallerStore creates a software installer store using the
// local filesystem rooted at the provided rootDir.
func NewSoftwareInstallerStore(rootDir string) (*SoftwareInstallerStore, error) {
	// ensure the directories exist (the provided rootDir and the
	// softwareInstallersPrefix we create inside it).
	dir := filepath.Join(rootDir, softwareInstallersPrefix)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return &SoftwareInstallerStore{rootDir}, nil
}

// Get retrieves the requested software installer from the local filesystem.
// It is important that the caller closes the reader when done.
func (i *SoftwareInstallerStore) Get(ctx context.Context, installerID string) (io.ReadCloser, int64, error) {
	path := i.pathForInstaller(installerID)
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, int64(0), installerNotFoundError{}
		}
		return nil, 0, ctxerr.Wrap(ctx, err, "retrieving software installer from filesystem store")
	}

	sz := st.Size()
	f, err := os.Open(path)
	if err != nil {
		return nil, sz, ctxerr.Wrap(ctx, err, "opening software installer file from filesystem store")
	}
	return f, sz, nil
}

// Put stores a software installer in the local filesystem.
func (i *SoftwareInstallerStore) Put(ctx context.Context, installerID string, content io.ReadSeeker) error {
	path := i.pathForInstaller(installerID)

	f, err := os.Create(path)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating software installer file in filesystem store")
	}
	defer f.Close()

	if _, err := io.Copy(f, content); err != nil {
		return ctxerr.Wrap(ctx, err, "writing software installer file in filesystem store")
	}
	if err := f.Close(); err != nil {
		return ctxerr.Wrap(ctx, err, "closing software installer file in filesystem store")
	}
	return nil
}

// Exists checks if a software installer exists in the filesystem for the ID.
func (i *SoftwareInstallerStore) Exists(ctx context.Context, installerID string) (bool, error) {
	path := i.pathForInstaller(installerID)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, ctxerr.Wrap(ctx, err, "looking up software installer in filesystem store")
	}
	return true, nil
}

func (i *SoftwareInstallerStore) Cleanup(ctx context.Context, usedInstallerIDs []string, removeCreatedBefore time.Time) (int, error) {
	usedSet := make(map[string]struct{}, len(usedInstallerIDs))
	for _, id := range usedInstallerIDs {
		usedSet[id] = struct{}{}
	}

	baseDir := filepath.Join(i.rootDir, softwareInstallersPrefix)
	dirEnts, err := os.ReadDir(baseDir)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "listing software installers in filesystem store")
	}

	// collect deletion errors so that it keeps going if possible
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
			return 0, ctxerr.Wrap(ctx, err, "get software installer modtime in filesystem store")
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
	return count, ctxerr.Wrap(ctx, errors.Join(errs...), "delete unused software installers")
}

func (i *SoftwareInstallerStore) Sign(ctx context.Context, _ string) (string, error) {
	return "", ctxerr.New(ctx, "signing not supported for software installers in filesystem store")
}

// pathForInstaller builds local filesystem path to identify the software
// installer.
func (i *SoftwareInstallerStore) pathForInstaller(installerID string) string {
	return filepath.Join(i.rootDir, softwareInstallersPrefix, installerID)
}
