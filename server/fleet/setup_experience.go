package fleet

import "errors"

type SetupExperienceStatusResultStatus string

const (
	SetupExperienceStatusPending SetupExperienceStatusResultStatus = "pending"
	SetupExperienceStatusRunning SetupExperienceStatusResultStatus = "running"
	SetupExperienceStatusSuccess SetupExperienceStatusResultStatus = "success"
	SetupExperienceStatusFailure SetupExperienceStatusResultStatus = "failure"
)

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
	ScriptExecutionID               *string                           `db:"script_execution_id" json:"execution_id,omitempty" `
	Error                           *string                           `db:"error" json:"-" `
	// SoftwareTitleID must be filled through a JOIN
	SoftwareTitleID *uint `json:"software_title_id,omitempty" db:"software_title_id"`
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

// SetupExperienceStatusPayload is the payload we send to Orbit to tell it what the current status
// of the setup experience is for that host.
type SetupExperienceStatusPayload struct {
	Script                *SetupExperienceStatusResult                 `json:"script,omitempty"`
	Software              []*SetupExperienceStatusResult               `json:"software,omitempty"`
	BootstrapPackage      *SetupExperienceBootstrapPackageResult       `json:"bootstrap_package,omitempty"`
	ConfigurationProfiles []*SetupExperienceConfigurationProfileResult `json:"configuration_profiles,omitempty"`
	AccountConfiguration  *SetupExperienceAccountConfigurationResult   `json:"account_configuration,omitempty"`
}
