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
	TeamID *uint `json:"team_id" db:"team_id"`
	// Name is the name of the software package.
	Name string `json:"name" db:"name"`
	// Version is the version of the software package.
	Version string `json:"version" db:"version"`
	// UploadedAt is the time the software package was uploaded.
	UploadedAt string `json:"uploaded_at" db:"uploaded_at"`
	// InstallerID is the unique identifier for the software package.
	InstallerID string `json:"-" db:"installer_id"`
	// InstallScript is the script to run to install the software package.
	InstallScript string `json:"install_script" db:"install_script"`
	// PreInstallQuery is the query to run as a condition to installing the software package.
	PreInstallQuery string `json:"pre_install_query" db:"pre_install_query"`
	// PostInstallScript is the script to run after installing the software package.
	PostInstallScript string `json:"post_install_script"`
}

// AuthzType implements authz.AuthzTyper.
func (s *SoftwareInstaller) AuthzType() string {
	return "software_installer"
}

// SoftwareInstallerStatus represents the status of a software installer package.
type SoftwareInstallerStatus struct {
	// Installed is the number of hosts that have the software package installed.
	Installed int `json:"installed" db:"installed"`
	// Pending is the number of hosts that have the software package pending installation.
	Pending int `json:"pending" db:"pending"`
	// Failed is the number of hosts that have the software package installation failed.
	Failed int `json:"failed" db:"failed"`
}
