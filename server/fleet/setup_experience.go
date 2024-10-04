package fleet

type SetupExperienceStatusResultStatus string

// Status values.
const (
	SetupExperienceStatusPending SetupExperienceStatusResultStatus = "pending"
	SetupExperienceStatusRunning SetupExperienceStatusResultStatus = "running"
	SetupExperienceStatusSuccess SetupExperienceStatusResultStatus = "success"
	SetupExperienceStatusFailure SetupExperienceStatusResultStatus = "failure"
)

type SetupExperienceStatusResult struct {
	ID                      uint                              `db:"id" json:"id,omitempty" `
	HostUUID                string                            `db:"host_uuid" json:"host_uuid,omitempty" `
	Name                    string                            `db:"name" json:"name,omitempty" `
	Status                  SetupExperienceStatusResultStatus `db:"status" json:"status,omitempty" `
	SoftwareInstallerID     *uint                             `db:"software_installer_id" json:"software_installer_id,omitempty" `
	HostSoftwareInstallsID  *uint                             `db:"host_software_installs_id" json:"host_software_installs_id,omitempty" `
	VPPAppTeamID            *uint                             `db:"vpp_app_team_id" json:"vpp_app_team_id,omitempty" `
	NanoCommandUUID         *string                           `db:"nano_command_uuid" json:"nano_command_uuid,omitempty" `
	SetupExperienceScriptID *uint                             `db:"setup_experience_script_id" json:"setup_experience_script_id,omitempty" `
	ScriptExecutionID       *string                           `db:"script_execution_id" json:"script_execution_id,omitempty" `
	Error                   *string                           `db:"error" json:"error,omitempty" `
}

func (s *SetupExperienceStatusResult) IsScript() bool {
	return s.SetupExperienceScriptID != nil
}

func (s *SetupExperienceStatusResult) IsSoftware() bool {
	return s.VPPAppTeamID != nil || s.SoftwareInstallerID != nil
}

type SetupExperienceStatusPayload struct {
	Script   *SetupExperienceStatusResult   `json:"script,omitempty"`
	Software []*SetupExperienceStatusResult `json:"software,omitempty"`
}
