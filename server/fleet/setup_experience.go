package fleet

import (
	"fmt"
)

type SetupExperienceResultType uint16

// Type values.
const (
	// TypeBootstrapPackage is the 'bootstrap-package' type.
	TypeBootstrapPackage SetupExperienceResultType = 1
	// TypeSoftwareInstall is the 'software-install' type.
	TypeSoftwareInstall SetupExperienceResultType = 2
	// TypePostInstallScript is the 'post-install-script' type.
	TypePostInstallScript SetupExperienceResultType = 3
)

// String satisfies the [fmt.Stringer] interface.
func (t SetupExperienceResultType) String() string {
	switch t {
	case TypeBootstrapPackage:
		return "bootstrap-package"
	case TypeSoftwareInstall:
		return "software-install"
	case TypePostInstallScript:
		return "post-install-script"
	}
	return fmt.Sprintf("Type(%d)", t)
}

type SetupExperienceStatusResultStatus uint16

// Status values.
const (
	// StatusPendingInstall is the 'pending_install' status.
	StatusPendingInstall SetupExperienceStatusResultStatus = 1
	// StatusFailedInstall is the 'failed_install' status.
	StatusFailedInstall SetupExperienceStatusResultStatus = 2
	// StatusInstalled is the 'installed' status.
	StatusInstalled SetupExperienceStatusResultStatus = 3
	// StatusPendingUninstall is the 'pending_uninstall' status.
	StatusPendingUninstall SetupExperienceStatusResultStatus = 4
	// StatusFailedUninstall is the 'failed_uninstall' status.
	StatusFailedUninstall SetupExperienceStatusResultStatus = 5
)

// String satisfies the [fmt.Stringer] interface.
func (s SetupExperienceStatusResultStatus) String() string {
	switch s {
	case StatusPendingInstall:
		return "pending_install"
	case StatusFailedInstall:
		return "failed_install"
	case StatusInstalled:
		return "installed"
	case StatusPendingUninstall:
		return "pending_uninstall"
	case StatusFailedUninstall:
		return "failed_uninstall"
	}
	return fmt.Sprintf("Status(%d)", s)
}

type SetupExperienceStatusResult struct {
	ID                     uint                              `db:"id" `                        // id
	HostUUID               string                            `db:"host_uuid" `                 // host_uuid
	Type                   SetupExperienceResultType         `db:"type" `                      // type
	Name                   string                            `db:"name" `                      // name
	Status                 SetupExperienceStatusResultStatus `db:"status" `                    // status
	HostSoftwareInstallsID *uint                             `db:"host_software_installs_id" ` // host_software_installs_id
	NanoCommandUUID        *string                           `db:"nano_command_uuid" `         // nano_command_uuid
	ScriptExecutionID      *string                           `db:"script_execution_id" `       // script_execution_id
	Error                  *string                           `db:"error" `                     // error
}
