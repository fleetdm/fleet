package fleet

type SetupExperienceStatusResultStatus string

// Status values.
const (
	StatusPending SetupExperienceStatusResultStatus = "pending"
	StatusRunning SetupExperienceStatusResultStatus = "running"
	StatusSuccess SetupExperienceStatusResultStatus = "success"
	StatusFailure SetupExperienceStatusResultStatus = "failure"
)

type SetupExperienceStatusResult struct {
	ID                      uint                              `db:"id"`
	HostUUID                string                            `db:"host_uuid"`
	Name                    string                            `db:"name"`
	Status                  SetupExperienceStatusResultStatus `db:"status"`
	SoftwareInstallerID     *uint                             `db:"software_installer_id"`
	HostSoftwareInstallsID  *uint                             `db:"host_software_installs_id"`
	VPPAppTeamID            *uint                             `db:"vpp_app_team_id"`
	NanoCommandUUID         *string                           `db:"nano_command_uuid"`
	SetupExperienceScriptID *uint                             `db:"setup_experience_script_id"`
	ScriptExecutionID       *string                           `db:"script_execution_id"`
	Error                   *string                           `db:"error"`
}
