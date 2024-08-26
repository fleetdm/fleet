package useraction

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

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
	// MigrationInProgress checks if the MDM migration is still in progress (i.e. the host is not
	// yet fully enrolled in Fleet MDM). It returns the type of migration that is in progress, if any.
	MigrationInProgress() (string, error)
	// MarkMigrationCompleted marks the migration as completed. This is currently done by removing
	// the migration file.
	MarkMigrationCompleted() error
}

// MDMMigratorProps are props required to display the dialog. It's akin to the
// concept of props in UI frameworks like React.
type MDMMigratorProps struct {
	OrgInfo     fleet.DesktopOrgInfo
	IsUnmanaged bool
	// DisableTakeover is used to disable the blur and always on top features of the dialog.
	DisableTakeover bool
}

// MDMMigratorHandler handles remote actions/callbacks that the migrator calls.
type MDMMigratorHandler interface {
	NotifyRemote() error
	ShowInstructions() error
}

type MDMOfflineWatcher interface {
	ShowIfOffline(ctx context.Context) bool
}
