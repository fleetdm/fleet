package useraction

import "github.com/fleetdm/fleet/v4/server/fleet"

// MDMMigrator represents the minimum set of methods a migration must implement
// in order to be used by Fleet Desktop.
type MDMMigrator interface {
	// CanRun indicates if the migrator is able to run, for example, for macOS it
	// checks if the swiftDialog executable is present.
	CanRun() bool
	// SetProps sets/updates the props.
	SetProps(MDMMigratorProps)
	// Show displays the dialog if there's no other dialog running.
	Show() error
	// ShowInterval is used to display dialogs at an interval. It displays
	// the dialog if there's no other dialog running and a given interval
	// (defined by the migrator itself) has passed since the last time the
	// dialog was shown.
	ShowInterval() error
	// Exit tries to stop any processes started by the migrator.
	Exit()
	// MigrationFileExists checks if the MDM migration file exists. If the file exists, then an MDM
	// migration was started but hasn't completed.
	MigrationFileExists() (bool, error)
	// RemoveMigrationFile removes the migration file once the MDM migration is complete.
	RemoveMigrationFile() error
}

// MDMMigratorProps are props required to display the dialog. It's akin to the
// concept of props in UI frameworks like React.
type MDMMigratorProps struct {
	OrgInfo     fleet.DesktopOrgInfo
	IsUnmanaged bool
}

// MDMMigratorHandler handles remote actions/callbacks that the migrator calls.
type MDMMigratorHandler interface {
	NotifyRemote() error
	ShowInstructions() error
}
