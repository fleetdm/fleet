package fleet

type SetupExperienceResultType string

// Type values.
const (
	TypeBootstrapPackage  SetupExperienceResultType = "bootstrap-package"
	TypeSoftwareInstall   SetupExperienceResultType = "software-install"
	TypePostInstallScript SetupExperienceResultType = "post-install-script"
)

type SetupExperienceStatusResultStatus string

// Status values.
const (
	StatusPending SetupExperienceStatusResultStatus = "pending"
	StatusRunning SetupExperienceStatusResultStatus = "running"
	StatusSuccess SetupExperienceStatusResultStatus = "success"
	StatusFailure SetupExperienceStatusResultStatus = "failure"
)

type SetupExperienceStatusResult struct {
	ID                     uint                              `db:"id" `
	HostUUID               string                            `db:"host_uuid" `
	Type                   SetupExperienceResultType         `db:"type" `
	Name                   string                            `db:"name" `
	Status                 SetupExperienceStatusResultStatus `db:"status" `
	HostSoftwareInstallsID *uint                             `db:"host_software_installs_id" `
	NanoCommandUUID        *string                           `db:"nano_command_uuid" `
	ScriptExecutionID      *string                           `db:"script_execution_id" `
	Error                  *string                           `db:"error" `
}
