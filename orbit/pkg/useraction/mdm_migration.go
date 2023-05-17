package useraction

import "github.com/fleetdm/fleet/v4/server/fleet"

type MDMMigrator interface {
	CanRun() bool
	SetProps(MDMMigratorProps)
	Show() error
	ShowInterval() error
	Exit()
}

type MDMMigratorProps struct {
	OrgInfo   fleet.DesktopOrgInfo
	Aggresive bool
}

type MDMMigratorHandler interface {
	NotifyRemote() error
	ShowInstructions()
}
