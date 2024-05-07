package fleet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"
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

// SoftwareInstallDetailsResult contains all of the information
// required for a client to pull in and install software from the fleet server
type SoftwareInstallDetails struct {
	// HostID is used for authentication on the backend and should not
	// be passed to the client
	HostID uint `json:"-" db:"host_id"`
	// ExecutionID is a unique identifier for this installation
	ExecutionID string `json:"install_id" db:"execution_id"`
	// InstallerID is the unique identifier for the software package metadata in Fleet.
	InstallerID uint `json:"installer_id" db:"installer_id"`
	// PreInstallCondition is the query to run as a condition to installing the software package.
	PreInstallCondition string `json:"pre_install_condition" db:"pre_install_condition"`
	// InstallScript is the script to run to install the software package.
	InstallScript string `json:"install_script" db:"install_script"`
	// PostInstallScript is the script to run after installing the software package.
	PostInstallScript string `json:"post_install_script" db:"post_install_script"`
}

// SoftwareInstaller represents a software installer package that can be used to install software on
// hosts in Fleet.
type SoftwareInstaller struct {
	// TeamID is the ID of the team. A value of nil means it is scoped to hosts that are assigned to
	// no team.
	TeamID *uint `json:"team_id" db:"team_id"`
	// TitleID is the id of the software title associated with the software installer.
	TitleID *uint `json:"-" db:"title_id"`
	// Name is the name of the software package.
	Name string `json:"name" db:"filename"`
	// Version is the version of the software package.
	Version string `json:"version" db:"version"`
	// UploadedAt is the time the software package was uploaded.
	UploadedAt time.Time `json:"uploaded_at" db:"uploaded_at"`
	// InstallerID is the unique identifier for the software package metadata in Fleet.
	InstallerID uint `json:"installer_id" db:"id"`
	// InstallScript is the script to run to install the software package.
	InstallScript string `json:"install_script" db:"-"`
	// InstallScriptContentID is the ID of the install script content.
	InstallScriptContentID uint `json:"-" db:"install_script_content_id"`
	// PreInstallQuery is the query to run as a condition to installing the software package.
	PreInstallQuery string `json:"pre_install_query" db:"pre_install_query"`
	// PostInstallScript is the script to run after installing the software package.
	PostInstallScript string `json:"post_install_script" db:"-"`
	// PostInstallScriptContentID is the ID of the post-install script content.
	PostInstallScriptContentID *uint `json:"-" db:"post_install_script_content_id"`
	// StorageID is the unique identifier for the software package in the software installer store.
	StorageID string `json:"-" db:"storage_id"`
	// SoftwareTitle is the title of the software pointed installed by this installer.
	SoftwareTitle string `json:"-" db:"software_title"`
}

// AuthzType implements authz.AuthzTyper.
func (s *SoftwareInstaller) AuthzType() string {
	return "software_installer"
}

// SoftwareInstallerStatusSummary represents aggregated status metrics for a software installer package.
type SoftwareInstallerStatusSummary struct {
	// Installed is the number of hosts that have the software package installed.
	Installed uint `json:"installed" db:"installed"`
	// Pending is the number of hosts that have the software package pending installation.
	Pending uint `json:"pending" db:"pending"`
	// Failed is the number of hosts that have the software package installation failed.
	Failed uint `json:"failed" db:"failed"`
}

// SoftwareInstallerStatus represents the status of a software installer package on a host.
type SoftwareInstallerStatus string

var (
	SoftwareInstallerPending   SoftwareInstallerStatus = "pending"
	SoftwareInstallerFailed    SoftwareInstallerStatus = "failed"
	SoftwareInstallerInstalled SoftwareInstallerStatus = "installed"
)

// HostSoftwareInstaller represents a software installer package that has been installed on a host.
type HostSoftwareInstallerResult struct {
	// InstallUUID is the unique identifier for the software install operation associated with the host.
	InstallUUID string `json:"install_uuid" db:"execution_id"`
	// SoftwareTitle is the title of the software.
	SoftwareTitle string `json:"software_title" db:"software_title"`
	// SoftwareVersion is the version of the software.
	SoftwareTitleID uint `json:"software_title_id" db:"software_title_id"`
	// SoftwarePackage is the name of the software installer package.
	SoftwarePackage string `json:"software_package" db:"software_package"`
	// HostID is the ID of the host.
	HostID uint `json:"host_id" db:"host_id"`
	// HostDisplayName is the display name of the host.
	HostDisplayName string `json:"host_display_name" db:"host_display_name"`
	// Status is the status of the software installer package on the host.
	Status SoftwareInstallerStatus `json:"status" db:"status"`
	// Detail is the detail of the software installer package on the host. TODO: does this field
	// have specific values that should be used? If so, how are they calculated?
	Detail string `json:"detail" db:"detail"`
	// Output is the output of the software installer package on the host.
	Output string `json:"output" db:"install_script_output"`
	// PreInstallQueryOutput is the output of the pre-install query on the host.
	PreInstallQueryOutput string `json:"pre_install_query_output" db:"pre_install_query_output"`
	// PostInstallScriptOutput is the output of the post-install script on the host.
	PostInstallScriptOutput string `json:"post_install_script_output" db:"post_install_script_output"`
	// HostTeamID is the team ID of the host on which this software install was attempted. This
	// field is not sent in the response, it is only used for internal authorization.
	HostTeamID *uint `json:"-" db:"host_team_id"`
	// UserID is the user ID that requested the software installation on that host.
	UserID *uint `json:"-" db:"user_id"`
}

type HostSoftwareInstallerResultAuthz struct {
	HostTeamID *uint `json:"host_team_id"`
}

// AuthzType implements authz.AuthzTyper.
func (s *HostSoftwareInstallerResultAuthz) AuthzType() string {
	return "host_software_installer_result"
}

type UploadSoftwareInstallerPayload struct {
	TeamID            *uint
	InstallScript     string
	PreInstallQuery   string
	PostInstallScript string
	InstallerFile     io.ReadSeeker // TODO: maybe pull this out of the payload and only pass it to methods that need it (e.g., won't be needed when storing metadata in the database)
	StorageID         string
	Filename          string
	Title             string
	Version           string
	Source            string
}

// DownloadSoftwareInstallerPayload is the payload for downloading a software installer.
type DownloadSoftwareInstallerPayload struct {
	Filename  string
	Installer io.ReadCloser
	Size      int64
}

func SofwareInstallerSourceFromFilename(filename string) (string, error) {
	switch ext := filepath.Ext(filename); ext {
	case ".deb":
		return "deb_packages", nil
	case ".exe", ".msi":
		return "programs", nil
	case ".pkg":
		return "pkg_packages", nil
	default:
		return "", fmt.Errorf("unsupported file type: %s", filename)
	}
}

// HostSoftwareWithInstaller represents the list of software installed on a
// host with installer information if a matching installer exists. This is the
// payload returned by the "Get host's (device's) software" endpoints.
type HostSoftwareWithInstaller struct {
	ID                uint                            `json:"id" db:"id"`
	Name              string                          `json:"name" db:"name"`
	Source            string                          `json:"source" db:"source"`
	Status            *SoftwareInstallerStatus        `json:"status" db:"status"`
	LastInstall       *HostSoftwareInstall            `json:"last_install"`
	InstalledVersions []*HostSoftwareInstalledVersion `json:"installed_versions"`

	// PackageAvailableForInstall is only present for the user-authenticated
	// endpoint, not the device-authenticated one. I.e. when
	// available-but-not-installed software are part of the response.
	PackageAvailableForInstall *string `json:"package_available_for_install,omitempty" db:"package_available_for_install"`
}

// HostSoftwareInstall represents installation of software on a host from a
// Fleet software installer.
type HostSoftwareInstall struct {
	InstallUUID string    `json:"install_uuid" db:"install_id"`
	InstalledAt time.Time `json:"installed_at" db:"installed_at"`
}

// HostSoftwareInstalledVersion represents a version of software installed on a
// host.
type HostSoftwareInstalledVersion struct {
	SoftwareID      uint       `json:"-" db:"software_id"`
	SoftwareTitleID uint       `json:"-" db:"software_title_id"`
	Version         string     `json:"version" db:"version"`
	LastOpenedAt    *time.Time `json:"last_opened_at" db:"last_opened_at"`
	Vulnerabilities []string   `json:"vulnerabilities" db:"vulnerabilities"`
	InstalledPaths  []string   `json:"installed_paths" db:"installed_paths"`
}

// HostSoftwareInstallResultPayload is the payload provided by fleetd to record
// the results of a software installation attempt.
type HostSoftwareInstallResultPayload struct {
	HostID      uint   `json:"host_id"`
	InstallUUID string `json:"install_uuid"`

	// the following fields are nil-able because the corresponding steps may not
	// have been executed (optional step, or executed conditionally to a previous
	// step).
	PreInstallConditionOutput *string `json:"pre_install_condition_output"`
	InstallScriptExitCode     *int    `json:"install_script_exit_code"`
	InstallScriptOutput       *string `json:"install_script_output"`
	PostInstallScriptExitCode *int    `json:"post_install_script_exit_code"`
	PostInstallScriptOutput   *string `json:"post_install_script_output"`
}

// Status returns the status computed from the result payload. It should match the logic
// found in the database-computed status (see
// softwareInstallerHostStatusNamedQuery in mysql/software.go).
func (h *HostSoftwareInstallResultPayload) Status() SoftwareInstallerStatus {
	switch {
	case h.PostInstallScriptExitCode != nil && *h.PostInstallScriptExitCode == 0:
		return SoftwareInstallerInstalled
	case h.PostInstallScriptExitCode != nil && *h.PostInstallScriptExitCode != 0:
		return SoftwareInstallerFailed
	case h.InstallScriptExitCode != nil && *h.InstallScriptExitCode == 0:
		return SoftwareInstallerInstalled
	case h.InstallScriptExitCode != nil && *h.InstallScriptExitCode != 0:
		return SoftwareInstallerFailed
	case h.PreInstallConditionOutput != nil && *h.PreInstallConditionOutput == "":
		return SoftwareInstallerFailed
	default:
		return SoftwareInstallerPending
	}
}
