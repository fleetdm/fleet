package fleet

import (
	"errors"
	"fmt"
)

type SetupExperienceStatusResultStatus string

const (
	SetupExperienceStatusPending   SetupExperienceStatusResultStatus = "pending"
	SetupExperienceStatusRunning   SetupExperienceStatusResultStatus = "running"
	SetupExperienceStatusSuccess   SetupExperienceStatusResultStatus = "success"
	SetupExperienceStatusFailure   SetupExperienceStatusResultStatus = "failure"
	SetupExperienceStatusCancelled SetupExperienceStatusResultStatus = "cancelled"
)

func (s SetupExperienceStatusResultStatus) IsValid() bool {
	switch s {
	case SetupExperienceStatusPending, SetupExperienceStatusRunning, SetupExperienceStatusSuccess, SetupExperienceStatusFailure:
		return true
	default:
		return false
	}
}

func (s SetupExperienceStatusResultStatus) IsTerminalStatus() bool {
	switch s {
	case SetupExperienceStatusSuccess, SetupExperienceStatusFailure:
		return true
	default:
		return false
	}
}

// SetupExperienceStatusResult represents the status of a particular step in the macOS setup
// experience process for a particular host. These steps can either be a software installer
// installation, a VPP app installation, or a script execution.
type SetupExperienceStatusResult struct {
	ID                              uint                              `db:"id" json:"-" `
	HostUUID                        string                            `db:"host_uuid" json:"-" `
	Name                            string                            `db:"name" json:"name,omitempty" `
	Status                          SetupExperienceStatusResultStatus `db:"status" json:"status,omitempty" `
	SoftwareInstallerID             *uint                             `db:"software_installer_id" json:"-" `
	HostSoftwareInstallsExecutionID *string                           `db:"host_software_installs_execution_id" json:"-" `
	VPPAppTeamID                    *uint                             `db:"vpp_app_team_id" json:"-" `
	VPPAppAdamID                    *string                           `db:"vpp_app_adam_id" json:"-"`
	VPPAppPlatform                  *string                           `db:"vpp_app_platform" json:"-"`
	NanoCommandUUID                 *string                           `db:"nano_command_uuid" json:"-" `
	SetupExperienceScriptID         *uint                             `db:"setup_experience_script_id" json:"-" `
	ScriptContentID                 *uint                             `db:"script_content_id" json:"-"`
	ScriptExecutionID               *string                           `db:"script_execution_id" json:"execution_id,omitempty" `
	Error                           *string                           `db:"error" json:"error" `
	// SoftwareTitleID must be filled through a JOIN
	SoftwareTitleID *uint `json:"software_title_id,omitempty" db:"software_title_id"`
}

func (s *SetupExperienceStatusResult) IsValid() error {
	var colsSet uint
	if s.SoftwareInstallerID != nil {
		colsSet++
		if s.NanoCommandUUID != nil || s.ScriptExecutionID != nil {
			return fmt.Errorf("invalid setup experience status row, software_installer_id set with incorrect secondary value column: %d", s.ID)
		}
	}
	if s.VPPAppTeamID != nil {
		colsSet++
		if s.HostSoftwareInstallsExecutionID != nil || s.ScriptExecutionID != nil {
			return fmt.Errorf("invalid setup experience status row, vpp_app_team set with incorrect secondary value column: %d", s.ID)
		}
	}
	if s.SetupExperienceScriptID != nil {
		colsSet++
		if s.HostSoftwareInstallsExecutionID != nil || s.NanoCommandUUID != nil {
			return fmt.Errorf("invalid setup experience status row, setip_experience_script_id set with incorrect secondary value column: %d", s.ID)
		}
	}
	if colsSet > 1 {
		return fmt.Errorf("invalid setup experience status row, multiple underlying value columns set: %d", s.ID)
	}
	if colsSet == 0 {
		return fmt.Errorf("invalid setup experience status row, no underlying value colunm set: %d", s.ID)
	}

	return nil
}

func (s *SetupExperienceStatusResult) VPPAppID() (*VPPAppID, error) {
	if s.VPPAppAdamID == nil || s.VPPAppPlatform == nil {
		return nil, errors.New("not a VPP app")
	}

	return &VPPAppID{
		AdamID:   *s.VPPAppAdamID,
		Platform: AppleDevicePlatform(*s.VPPAppPlatform),
	}, nil
}

// IsForScript indicates if this result is for a setup experience script step.
func (s *SetupExperienceStatusResult) IsForScript() bool {
	return s.SetupExperienceScriptID != nil
}

// IsForSoftware indicates if this result is for a setup experience software step: either a software
// installer or a VPP app.
func (s *SetupExperienceStatusResult) IsForSoftware() bool {
	return s.VPPAppTeamID != nil || s.SoftwareInstallerID != nil
}

// IsForSoftwarePackage indicates if this result is for a setup experience software installer step.
func (s *SetupExperienceStatusResult) IsForSoftwarePackage() bool {
	return s.SoftwareInstallerID != nil
}

type SetupExperienceBootstrapPackageResult struct {
	Name   string                    `json:"name"`
	Status MDMBootstrapPackageStatus `json:"status"`
}

type SetupExperienceConfigurationProfileResult struct {
	ProfileUUID string            `json:"profile_uuid"`
	Name        string            `json:"name"`
	Status      MDMDeliveryStatus `json:"status"`
}

type SetupExperienceAccountConfigurationResult struct {
	CommandUUID string `json:"command_uuid"`
	Status      string `json:"status"`
}

type SetupExperienceVPPInstallResult struct {
	HostUUID      string
	CommandUUID   string
	CommandStatus string
}

func (r SetupExperienceVPPInstallResult) SetupExperienceStatus() SetupExperienceStatusResultStatus {
	switch r.CommandStatus {
	case MDMAppleStatusAcknowledged:
		return SetupExperienceStatusSuccess
	case MDMAppleStatusError, MDMAppleStatusCommandFormatError:
		return SetupExperienceStatusFailure
	default:
		// TODO: is this what we want as the default, what about other possible statuses?
		return SetupExperienceStatusPending
	}
}

type SetupExperienceSoftwareInstallResult struct {
	HostUUID        string
	ExecutionID     string
	InstallerStatus SoftwareInstallerStatus
}

func (r SetupExperienceSoftwareInstallResult) SetupExperienceStatus() SetupExperienceStatusResultStatus {
	switch r.InstallerStatus {
	case SoftwareInstalled:
		return SetupExperienceStatusSuccess
	case SoftwareFailed, SoftwareInstallFailed:
		return SetupExperienceStatusFailure
	default:
		// TODO: is this what we want as the default, what about other possible statuses (uninstall)?
		return SetupExperienceStatusPending
	}
}

type SetupExperienceScriptResult struct {
	HostUUID    string
	ExecutionID string
	ExitCode    int
}

func (r SetupExperienceScriptResult) SetupExperienceStatus() SetupExperienceStatusResultStatus {
	if r.ExitCode == 0 {
		return SetupExperienceStatusSuccess
	}
	// TODO: what about other possible script statuses? seems like pending/running is never a
	// possibility here (exit code can't be null)?
	return SetupExperienceStatusFailure
}

// SetupExperienceStatusPayload is the payload we send to Orbit to tell it what the current status
// of the setup experience is for that host.
type SetupExperienceStatusPayload struct {
	Script                *SetupExperienceStatusResult                 `json:"script,omitempty"`
	Software              []*SetupExperienceStatusResult               `json:"software,omitempty"`
	BootstrapPackage      *SetupExperienceBootstrapPackageResult       `json:"bootstrap_package,omitempty"`
	ConfigurationProfiles []*SetupExperienceConfigurationProfileResult `json:"configuration_profiles,omitempty"`
	AccountConfiguration  *SetupExperienceAccountConfigurationResult   `json:"account_configuration,omitempty"`
	OrgLogoURL            string                                       `json:"org_logo_url"`
	RequireAllSoftware    bool                                         `json:"require_all_software"`
}

// IsSetupExperienceSupported returns whether "Setup experience" is supported for the host's platform.
// TODO: Setup Experience supports a wide range of platforms now but has a feature matrix where not all
// platforms support all features. May be worth refactoring to check for supported features instead
func IsSetupExperienceSupported(hostPlatform string) bool {
	return hostPlatform == "darwin" || hostPlatform == "ios" || hostPlatform == "ipados" || hostPlatform == "windows" || IsLinux(hostPlatform)
}

// DeviceSetupExperienceStatusPayload holds the status of the "Setup experience" for a device.
type DeviceSetupExperienceStatusPayload struct {
	// Software holds the status of the software to install on the device.
	Software []*SetupExperienceStatusResult `json:"software,omitempty"`
	// Scripts holds the status of the scripts to run on the device.
	Scripts []*SetupExperienceStatusResult `json:"scripts,omitempty"`
}

// HostUUIDForSetupExperience returns the host "UUID" to use during the "Setup experience"
// for a non-Apple host.
//
// The setup_experience_status_results uses the host's "UUID" as the host identifier because the table
// was created to implement "Setup experience" for macOS devices.
//
// On Windows/Linux devices there might be issues with duplicate hardware UUIDs, so for that reason we will instead
// use the host.OsqueryHostID as UUID. For Windows/Linux devices, the "Setup experience" will be triggered after orbit
// and osquery enrollment, thus host.OsqueryHostID will always be set and unique.
func HostUUIDForSetupExperience(host *Host) (string, error) {
	if host.Platform == string(MacOSPlatform) || host.Platform == string(IOSPlatform) || host.Platform == string(IPadOSPlatform) {
		return host.UUID, nil
	}
	// Currently it seems this field is always set when orbit or osquery enroll,
	// to be safe we return an error when that's the case (instead of panicking).
	if host.OsqueryHostID == nil {
		return "", errors.New("missing osquery_host_id")
	}
	return *host.OsqueryHostID, nil
}

type SetupExperienceCount struct {
	Installers uint `db:"installers"`
	Scripts    uint `db:"scripts"`
	VPP        uint `db:"vpp"`
}
