package fleet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

// MaxSoftwareInstallerSize is the maximum size allowed for software
// installers. This is enforced by the endpoints that upload installers.
const MaxSoftwareInstallerSize = 3000 * units.MiB

// SoftwareInstallerStore is the interface to store and retrieve software
// installer files. Fleet supports storing to the local filesystem and to an
// S3 bucket.
type SoftwareInstallerStore interface {
	Get(ctx context.Context, installerID string) (io.ReadCloser, int64, error)
	Put(ctx context.Context, installerID string, content io.ReadSeeker) error
	Exists(ctx context.Context, installerID string) (bool, error)
	Cleanup(ctx context.Context, usedInstallerIDs []string, removeCreatedBefore time.Time) (int, error)
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

func (FailingSoftwareInstallerStore) Cleanup(ctx context.Context, usedInstallerIDs []string, removeCreatedBefore time.Time) (int, error) {
	// do not fail for the failing store's cleanup, as unlike the other store
	// methods, this will be called even if software installers are otherwise not
	// used (by the cron job).
	return 0, nil
}

// SoftwareInstallDetails contains all of the information
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
	// UninstallScript is the script to run to uninstall the software package.
	UninstallScript string `json:"uninstall_script" db:"uninstall_script"`
	// PostInstallScript is the script to run after installing the software package.
	PostInstallScript string `json:"post_install_script" db:"post_install_script"`
	// SelfService indicates the install was initiated by the device user
	SelfService bool `json:"self_service" db:"self_service"`
}

// SoftwareInstaller represents a software installer package that can be used to install software on
// hosts in Fleet.
type SoftwareInstaller struct {
	// TeamID is the ID of the team. A value of nil means it is scoped to hosts that are assigned to
	// no team.
	TeamID *uint `json:"team_id" db:"team_id"`
	// TitleID is the id of the software title associated with the software installer.
	TitleID *uint `json:"title_id" db:"title_id"`
	// Name is the name of the software package.
	Name string `json:"name" db:"filename"`
	// Extension is the file extension of the software package, inferred from package contents.
	Extension string `json:"-" db:"extension"`
	// Version is the version of the software package.
	Version string `json:"version" db:"version"`
	// Platform can be "darwin" (for pkgs), "windows" (for exes/msis) or "linux" (for debs).
	Platform string `json:"platform" db:"platform"`
	// PackageIDList is a comma-separated list of packages extracted from the installer
	PackageIDList string `json:"-" db:"package_ids"`
	// UploadedAt is the time the software package was uploaded.
	UploadedAt time.Time `json:"uploaded_at" db:"uploaded_at"`
	// InstallerID is the unique identifier for the software package metadata in Fleet.
	InstallerID uint `json:"installer_id" db:"id"`
	// InstallScript is the script to run to install the software package.
	InstallScript string `json:"install_script" db:"install_script"`
	// InstallScriptContentID is the ID of the install script content.
	InstallScriptContentID uint `json:"-" db:"install_script_content_id"`
	// UninstallScriptContentID is the ID of the uninstall script content.
	UninstallScriptContentID uint `json:"-" db:"uninstall_script_content_id"`
	// PreInstallQuery is the query to run as a condition to installing the software package.
	PreInstallQuery string `json:"pre_install_query" db:"pre_install_query"`
	// PostInstallScript is the script to run after installing the software package.
	PostInstallScript string `json:"post_install_script" db:"post_install_script"`
	// UninstallScript is the script to run to uninstall the software package.
	UninstallScript string `json:"uninstall_script" db:"uninstall_script"`
	// PostInstallScriptContentID is the ID of the post-install script content.
	PostInstallScriptContentID *uint `json:"-" db:"post_install_script_content_id"`
	// StorageID is the unique identifier for the software package in the software installer store.
	StorageID string `json:"-" db:"storage_id"`
	// Status is the status of the software installer package.
	Status *SoftwareInstallerStatusSummary `json:"status,omitempty" db:"-"`
	// SoftwareTitle is the title of the software pointed installed by this installer.
	SoftwareTitle string `json:"-" db:"software_title"`
	// SelfService indicates that the software can be installed by the
	// end user without admin intervention
	SelfService bool `json:"self_service" db:"self_service"`
	// URL is the source URL for this installer (set when uploading via batch/gitops).
	URL string `json:"url" db:"url"`
	// FleetLibraryAppID is the related Fleet-maintained app for this installer (if not nil).
	FleetLibraryAppID *uint `json:"-" db:"fleet_library_app_id"`
	// AutomaticInstallPolicies is the list of policies that trigger automatic
	// installation of this software.
	AutomaticInstallPolicies []AutomaticInstallPolicy `json:"automatic_install_policies" db:"-"`
	// LablesIncludeAny is the list of "include any" labels for this software installer (if not nil).
	LabelsIncludeAny []SoftwareScopeLabel `json:"labels_include_any" db:"labels_include_any"`
	// LabelsExcludeAny is the list of "exclude any" labels for this software installer (if not nil).
	LabelsExcludeAny []SoftwareScopeLabel `json:"labels_exclude_any" db:"labels_exclude_any"`
}

// SoftwarePackageResponse is the response type used when applying software by batch.
type SoftwarePackageResponse struct {
	// TeamID is the ID of the team.
	// A value of nil means it is scoped to hosts that are assigned to "No team".
	TeamID *uint `json:"team_id" db:"team_id"`
	// TitleID is the id of the software title associated with the software installer.
	TitleID *uint `json:"title_id" db:"title_id"`
	// URL is the source URL for this installer (set when uploading via batch/gitops).
	URL string `json:"url" db:"url"`
}

// AuthzType implements authz.AuthzTyper.
func (s *SoftwareInstaller) AuthzType() string {
	return "installable_entity"
}

// PackageIDs turns the comma-separated string from the database into a list (potentially zero-length) of string package IDs
func (s *SoftwareInstaller) PackageIDs() []string {
	if s.PackageIDList == "" {
		return []string{}
	}

	return strings.Split(s.PackageIDList, ",")
}

// SoftwareInstallerStatusSummary represents aggregated status metrics for a software installer package.
type SoftwareInstallerStatusSummary struct {
	// Installed is the number of hosts that have the software package installed.
	Installed uint `json:"installed" db:"installed"`
	// PendingInstall is the number of hosts that have the software package pending installation.
	PendingInstall uint `json:"pending_install" db:"pending_install"`
	// FailedInstall is the number of hosts that have the software package installation failed.
	FailedInstall uint `json:"failed_install" db:"failed_install"`
	// PendingUninstall is the number of hosts that have the software package pending installation.
	PendingUninstall uint `json:"pending_uninstall" db:"pending_uninstall"`
	// FailedInstall is the number of hosts that have the software package installation failed.
	FailedUninstall uint `json:"failed_uninstall" db:"failed_uninstall"`
}

// SoftwareInstallerStatus represents the status of a software installer package on a host.
type SoftwareInstallerStatus string

const (
	SoftwareInstallPending   SoftwareInstallerStatus = "pending_install"
	SoftwareInstallFailed    SoftwareInstallerStatus = "failed_install"
	SoftwareInstalled        SoftwareInstallerStatus = "installed"
	SoftwareUninstallPending SoftwareInstallerStatus = "pending_uninstall"
	SoftwareUninstallFailed  SoftwareInstallerStatus = "failed_uninstall"
	// SoftwarePending and SoftwareFailed statuses are only used as filters in the API and are not stored in the database.
	SoftwarePending SoftwareInstallerStatus = "pending" // either pending_install or pending_uninstall
	SoftwareFailed  SoftwareInstallerStatus = "failed"  // either failed_install or failed_uninstall
)

func (s SoftwareInstallerStatus) IsValid() bool {
	switch s {
	case
		SoftwarePending,
		SoftwareFailed,
		SoftwareUninstallPending,
		SoftwareUninstallFailed,
		SoftwareInstallFailed,
		SoftwareInstalled,
		SoftwareInstallPending:
		return true
	default:
		return false
	}
}

// HostLastInstallData contains data for the last installation of a package on a host.
type HostLastInstallData struct {
	// ExecutionID is the installation ID of the package on the host.
	ExecutionID string `db:"execution_id"`
	// Status is the status of the installation on the host.
	Status *SoftwareInstallerStatus `db:"status"`
}

// HostSoftwareInstaller represents a software installer package that has been installed on a host.
type HostSoftwareInstallerResult struct {
	// ID is the unique numerical ID of the result assigned by the datastore.
	ID uint `json:"-" db:"id"`
	// InstallUUID is the unique identifier for the software install operation associated with the host.
	InstallUUID string `json:"install_uuid" db:"execution_id"`
	// SoftwareTitle is the title of the software.
	SoftwareTitle string `json:"software_title" db:"software_title"`
	// SoftwareTitleID is the unique numerical ID of the software title assigned by the datastore.
	SoftwareTitleID *uint `json:"software_title_id" db:"software_title_id"`
	// SoftwareInstallerID is the unique numerical ID of the software installer assigned by the datastore.
	SoftwareInstallerID *uint `json:"-" db:"software_installer_id"`
	// SoftwarePackage is the name of the software installer package.
	SoftwarePackage string `json:"software_package" db:"software_package"`
	// HostID is the ID of the host.
	HostID uint `json:"host_id" db:"host_id"`
	// Status is the status of the software installer package on the host.
	Status SoftwareInstallerStatus `json:"status" db:"status"`
	// Output is the output of the software installer package on the host.
	Output *string `json:"output" db:"install_script_output"`
	// PreInstallQueryOutput is the output of the pre-install query on the host.
	PreInstallQueryOutput *string `json:"pre_install_query_output" db:"pre_install_query_output"`
	// PostInstallScriptOutput is the output of the post-install script on the host.
	PostInstallScriptOutput *string `json:"post_install_script_output" db:"post_install_script_output"`
	// CreatedAt is the time the software installer request was triggered.
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	// UpdatedAt is the time the software installer request was last updated.
	UpdatedAt *time.Time `json:"updated_at" db:"updated_at"`
	// UserID is the user ID that requested the software installation on that host.
	UserID *uint `json:"-" db:"user_id"`
	// InstallScriptExitCode is used internally to determine the output displayed to the user.
	InstallScriptExitCode *int `json:"-" db:"install_script_exit_code"`
	// PostInstallScriptExitCode is used internally to determine the output displayed to the user.
	PostInstallScriptExitCode *int `json:"-" db:"post_install_script_exit_code"`
	// SelfService indicates that the installation was queued by the
	// end user and not an administrator
	SelfService bool `json:"self_service" db:"self_service"`
	// HostDeletedAt indicates if the data is associated with a
	// deleted host
	HostDeletedAt *time.Time `json:"-" db:"host_deleted_at"`
	// PolicyID is the id of the policy that triggered the install, or
	// nil if the install was not triggered by a policy failure
	PolicyID *uint `json:"policy_id" db:"policy_id"`
}

const (
	SoftwareInstallerQueryFailCopy          = "Query didn't return result or failed\nInstall stopped"
	SoftwareInstallerQuerySuccessCopy       = "Query returned result\nProceeding to install..."
	SoftwareInstallerScriptsDisabledCopy    = "Installing software...\nError: Scripts are disabled for this host. To run scripts, deploy the fleetd agent with --scripts-enabled."
	SoftwareInstallerInstallFailCopy        = "Installing software...\nFailed\n%s"
	SoftwareInstallerInstallSuccessCopy     = "Installing software...\nSuccess\n%s"
	SoftwareInstallerPostInstallSuccessCopy = "Running script...\nExit code: 0 (Success)\n%s"
	SoftwareInstallerPostInstallFailCopy    = `Running script...
Exit code: %d (Failed)
%s
`
)

// EnhanceOutputDetails is used to add extra boilerplate/information to the
// output fields so they're easier to consume by users.
func (h *HostSoftwareInstallerResult) EnhanceOutputDetails() {
	if h.Status == SoftwareInstallPending {
		return
	}

	if h.PreInstallQueryOutput != nil {
		if *h.PreInstallQueryOutput == "" {
			*h.PreInstallQueryOutput = SoftwareInstallerQueryFailCopy
			return
		}
		*h.PreInstallQueryOutput = SoftwareInstallerQuerySuccessCopy
	}

	if h.Output == nil || h.InstallScriptExitCode == nil {
		return
	}
	if *h.InstallScriptExitCode == -2 {
		*h.Output = SoftwareInstallerScriptsDisabledCopy
		return
	} else if *h.InstallScriptExitCode != 0 {
		h.Output = ptr.String(fmt.Sprintf(SoftwareInstallerInstallFailCopy, *h.Output))
		return
	}
	h.Output = ptr.String(fmt.Sprintf(SoftwareInstallerInstallSuccessCopy, *h.Output))

	if h.PostInstallScriptExitCode == nil || h.PostInstallScriptOutput == nil {
		return
	}
	if *h.PostInstallScriptExitCode != 0 {
		h.PostInstallScriptOutput = ptr.String(fmt.Sprintf(SoftwareInstallerPostInstallFailCopy, *h.PostInstallScriptExitCode, *h.PostInstallScriptOutput))
		return
	}

	h.PostInstallScriptOutput = ptr.String(fmt.Sprintf(SoftwareInstallerPostInstallSuccessCopy, *h.PostInstallScriptOutput))
}

type HostSoftwareInstallerResultAuthz struct {
	HostTeamID *uint `json:"host_team_id"`
}

// AuthzType implements authz.AuthzTyper.
func (s *HostSoftwareInstallerResultAuthz) AuthzType() string {
	return "host_software_installer_result"
}

type UploadSoftwareInstallerPayload struct {
	TeamID             *uint
	InstallScript      string
	PreInstallQuery    string
	PostInstallScript  string
	InstallerFile      *TempFileReader // TODO: maybe pull this out of the payload and only pass it to methods that need it (e.g., won't be needed when storing metadata in the database)
	StorageID          string
	Filename           string
	Title              string
	Version            string
	Source             string
	Platform           string
	BundleIdentifier   string
	SelfService        bool
	UserID             uint
	URL                string
	FleetLibraryAppID  *uint
	PackageIDs         []string
	UninstallScript    string
	Extension          string
	InstallDuringSetup *bool    // keep saved value if nil, otherwise set as indicated
	LabelsIncludeAny   []string // names of "include any" labels
	LabelsExcludeAny   []string // names of "exclude any" labels
	// ValidatedLabels is a struct that contains the validated labels for the software installer. It
	// is nil if the labels have not been validated.
	ValidatedLabels  *LabelIdentsWithScope
	AutomaticInstall bool
}

type UpdateSoftwareInstallerPayload struct {
	// find the installer via these fields
	TitleID     uint
	TeamID      *uint
	InstallerID uint
	// used for authorization and persisted as author
	UserID uint
	// optional; used for pulling metadata + persisting new installer package to file system
	InstallerFile *TempFileReader
	// update the installer with these fields (*not* PATCH semantics at that point; while the
	// associated endpoint is a PATCH, the entire row will be updated to these values, including
	// blanks, so make sure they're set from either user input or the existing installer row
	// before saving)
	InstallScript     *string
	PreInstallQuery   *string
	PostInstallScript *string
	SelfService       *bool
	UninstallScript   *string
	StorageID         string
	Filename          string
	Version           string
	PackageIDs        []string
	LabelsIncludeAny  []string // names of "include any" labels
	LabelsExcludeAny  []string // names of "exclude any" labels
	// ValidatedLabels is a struct that contains the validated labels for the software installer. It
	// can be nil if the labels have not been validated or if the labels are not being updated.
	ValidatedLabels *LabelIdentsWithScope
}

// DownloadSoftwareInstallerPayload is the payload for downloading a software installer.
type DownloadSoftwareInstallerPayload struct {
	Filename  string
	Installer io.ReadCloser
	Size      int64
}

func SofwareInstallerSourceFromExtensionAndName(ext, name string) (string, error) {
	ext = strings.TrimPrefix(ext, ".")
	switch ext {
	case "deb":
		return "deb_packages", nil
	case "rpm":
		return "rpm_packages", nil
	case "exe", "msi":
		return "programs", nil
	case "pkg":
		if filepath.Ext(name) == ".app" {
			return "apps", nil
		}
		return "pkg_packages", nil
	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

func SofwareInstallerPlatformFromExtension(ext string) (string, error) {
	ext = strings.TrimPrefix(ext, ".")
	switch ext {
	case "deb", "rpm":
		return "linux", nil
	case "exe", "msi":
		return "windows", nil
	case "pkg":
		return "darwin", nil
	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
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
	InstalledVersions []*HostSoftwareInstalledVersion `json:"installed_versions"`

	// SoftwarePackage provides software installer package information, it is
	// only present if a software installer is available for the software title.
	SoftwarePackage *SoftwarePackageOrApp `json:"software_package"`

	// AppStoreApp provides VPP app information, it is only present if a VPP app
	// is available for the software title.
	AppStoreApp *SoftwarePackageOrApp `json:"app_store_app"`
}

type AutomaticInstallPolicy struct {
	ID      uint   `json:"id" db:"id"`
	Name    string `json:"name" db:"name"`
	TitleID uint   `json:"-" db:"software_title_id"`
}

// SoftwarePackageOrApp provides information about a software installer
// package or a VPP app.
type SoftwarePackageOrApp struct {
	// AppStoreID is only present for VPP apps.
	AppStoreID string `json:"app_store_id,omitempty"`
	// Name is only present for software installer packages.
	Name string `json:"name,omitempty"`
	// AutomaticInstallPolicies is present for Fleet maintained apps and custom packages
	// installed automatically with a policy.
	AutomaticInstallPolicies []AutomaticInstallPolicy `json:"automatic_install_policies"`

	Version       string                 `json:"version"`
	SelfService   *bool                  `json:"self_service,omitempty"`
	IconURL       *string                `json:"icon_url"`
	LastInstall   *HostSoftwareInstall   `json:"last_install"`
	LastUninstall *HostSoftwareUninstall `json:"last_uninstall"`
	PackageURL    *string                `json:"package_url"`
	// InstallDuringSetup is a boolean that indicates if the package
	// will be installed during the macos setup experience.
	InstallDuringSetup *bool `json:"install_during_setup,omitempty" db:"install_during_setup"`
}

type SoftwarePackageSpec struct {
	URL               string                `json:"url"`
	SelfService       bool                  `json:"self_service"`
	PreInstallQuery   TeamSpecSoftwareAsset `json:"pre_install_query"`
	InstallScript     TeamSpecSoftwareAsset `json:"install_script"`
	PostInstallScript TeamSpecSoftwareAsset `json:"post_install_script"`
	UninstallScript   TeamSpecSoftwareAsset `json:"uninstall_script"`
	LabelsIncludeAny  []string              `json:"labels_include_any"`
	LabelsExcludeAny  []string              `json:"labels_exclude_any"`

	// ReferencedYamlPath is the resolved path of the file used to fill the
	// software package. Only present after parsing a GitOps file on the fleetctl
	// side of processing. This is required to match a macos_setup.software to
	// its corresponding software package, as we do this matching by yaml path.
	//
	// It must be JSON-marshaled because it gets set during gitops file processing,
	// which is then re-marshaled to JSON from this struct and later re-unmarshaled
	// during ApplyGroup...
	ReferencedYamlPath string `json:"referenced_yaml_path"`
}

type SoftwareSpec struct {
	Packages     optjson.Slice[SoftwarePackageSpec] `json:"packages,omitempty"`
	AppStoreApps optjson.Slice[TeamSpecAppStoreApp] `json:"app_store_apps,omitempty"`
}

// HostSoftwareInstall represents installation of software on a host from a
// Fleet software installer.
type HostSoftwareInstall struct {
	// InstallUUID is the the UUID of the script execution issued to install the related software. This
	// field is only used if the install we're describing was for an uploaded software installer.
	// Empty if the install was for an App Store app.
	InstallUUID string `json:"install_uuid,omitempty"`

	// CommandUUID is the UUID of the MDM command issued to install the related software. This field
	// is only used if the install we're describing was for an App Store app.
	// Empty if the install was for an uploaded software installer.
	CommandUUID string    `json:"command_uuid,omitempty"`
	InstalledAt time.Time `json:"installed_at"`
}

// HostSoftwareUninstall represents uninstallation of software from a host with a
// Fleet software installer.
type HostSoftwareUninstall struct {
	// ExecutionID is the UUID of the script execution that uninstalled the software.
	ExecutionID   string    `json:"script_execution_id,omitempty"`
	UninstalledAt time.Time `json:"uninstalled_at"`
}

// HostSoftwareInstalledVersion represents a version of software installed on a host.
type HostSoftwareInstalledVersion struct {
	SoftwareID       uint       `json:"-" db:"software_id"`
	SoftwareTitleID  uint       `json:"-" db:"software_title_id"`
	Source           string     `json:"-" db:"source"`
	Version          string     `json:"version" db:"version"`
	BundleIdentifier string     `json:"bundle_identifier,omitempty" db:"bundle_identifier"`
	LastOpenedAt     *time.Time `json:"last_opened_at" db:"last_opened_at"`

	Vulnerabilities      []string                   `json:"vulnerabilities" db:"vulnerabilities"`
	InstalledPaths       []string                   `json:"installed_paths"`
	SignatureInformation []PathSignatureInformation `json:"signature_information,omitempty"`
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
		return SoftwareInstalled
	case h.PostInstallScriptExitCode != nil && *h.PostInstallScriptExitCode != 0:
		return SoftwareInstallFailed
	case h.InstallScriptExitCode != nil && *h.InstallScriptExitCode == 0:
		return SoftwareInstalled
	case h.InstallScriptExitCode != nil && *h.InstallScriptExitCode != 0:
		return SoftwareInstallFailed
	case h.PreInstallConditionOutput != nil && *h.PreInstallConditionOutput == "":
		return SoftwareInstallFailed
	default:
		return SoftwareInstallPending
	}
}

// SoftwareInstallerTokenMetadata is the metadata stored in Redis for a software installer token.
type SoftwareInstallerTokenMetadata struct {
	TitleID uint `json:"title_id"`
	TeamID  uint `json:"team_id"`
}

const SoftwareInstallerURLMaxLength = 4000

// TempFileReader is an io.Reader with all extra io interfaces supported by a
// file on disk reader (e.g. io.ReaderAt, io.Seeker, etc.). When created with
// NewTempFileReader, it is backed by a temporary file on disk, and that file
// is deleted when Close is called.
type TempFileReader struct {
	*os.File
	keepFile bool
}

// Rewind seeks to the beginning of the file so the next read will read from
// the start of the bytes.
func (r *TempFileReader) Rewind() error {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return err
	}
	return nil
}

// Close closes the TempFileReader and deletes the underlying temp file unless
// it was instructed not to do so at creation time.
func (r *TempFileReader) Close() error {
	cerr := r.File.Close()
	var rerr error
	if !r.keepFile {
		rerr = os.Remove(r.File.Name())
	}
	if cerr != nil {
		return cerr
	}
	return rerr
}

// NewKeepFileReader creates a TempFileReader from a file path and keeps the
// file on Close, instead of deleting it.
func NewKeepFileReader(filename string) (*TempFileReader, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return &TempFileReader{File: f, keepFile: true}, nil
}

// NewTempFileReader creates a temp file to store the data from the provided
// reader and returns a TempFileReader that reads from that temp file, deleting
// it on close.
func NewTempFileReader(from io.Reader, tempDirFn func() string) (*TempFileReader, error) {
	if tempDirFn == nil {
		tempDirFn = os.TempDir
	}

	tempFile, err := os.CreateTemp(tempDirFn(), "fleet-temp-file-*")
	if err != nil {
		return nil, err
	}
	tfr := &TempFileReader{File: tempFile}

	if _, err := io.Copy(tempFile, from); err != nil {
		_ = tfr.Close() // best-effort close/delete
		return nil, err
	}
	if err := tfr.Rewind(); err != nil {
		_ = tfr.Close() // best-effort close/delete
		return nil, err
	}
	return tfr, nil
}

// SoftwareScopeLabel represents the many-to-many relationship between
// software titles and labels.
//
// NOTE: json representation of the fields is a bit awkward to match the
// required API response, as this struct is returned within software title details.
//
// NOTE: depending on how/where this struct is used, fields MAY BE
// UNRELIABLE insofar as they represent default, empty values.
type SoftwareScopeLabel struct {
	LabelName string `db:"label_name" json:"name"`
	LabelID   uint   `db:"label_id" json:"id"` // label id in database, which may be the empty value in some cases where id is not known in advance (e.g., if labels are created during gitops processing)
	Exclude   bool   `db:"exclude" json:"-"`   // not rendered in JSON, used when processing LabelsIncludeAny and LabelsExcludeAny on parent title (may be the empty value in some cases)
	TitleID   uint   `db:"title_id" json:"-"`  // not rendered in JSON, used to store the associated title ID (may be the empty value in some cases)
}
