package fleet

import (
	"context"
	"errors"
	"io"
)

// SoftwareInstallerStore is the interface to store and retrieve software
// installer files. Fleet supports storing to the local filesystem and to an
// S3 bucket.
type SoftwareInstallerStore interface {
	Get(ctx context.Context, installerID string) (io.ReadCloser, int64, error)
	Put(ctx context.Context, installerID string, content io.ReadSeeker) error
	Exists(ctx context.Context, installerID string) (bool, error)
}

// FailingSoftwareInstallerStore is an implementation of SoftwareInstallerStore
// that fails all operations. It is used when S3 is not configured and the
// local filesystem store could not be setup.
type FailingSoftwareInstallerStore struct{}

func (FailingSoftwareInstallerStore) Get(ctx context.Context, installerID string) (io.ReadCloser, int64, error) {
	return nil, 0, errors.New("software installer store not properly configured")
}

func (FailingSoftwareInstallerStore) Put(ctx context.Context, installerID string, content io.ReadSeeker) error {
	return errors.New("software installer store not properly configured")
}

func (FailingSoftwareInstallerStore) Exists(ctx context.Context, installerID string) (bool, error) {
	return false, errors.New("software installer store not properly configured")
}

// SoftwareInstaller represents a software installer package that can be used to install software on
// hosts in Fleet.
type SoftwareInstaller struct {
	// TeamID is the ID of the team. A value of nil means it is scoped to hosts that are assigned to
	// no team.
	TeamID *uint `json:"team_id"`
}

// AuthzType implements authz.AuthzTyper.
func (s *SoftwareInstaller) AuthzType() string {
	return "software_installer"
}
