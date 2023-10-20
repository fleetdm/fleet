//go:build windows

package table

import (
	cisaudit "github.com/fleetdm/fleet/v4/orbit/pkg/table/cis_audit"
	mdmbridge "github.com/fleetdm/fleet/v4/orbit/pkg/table/mdm"

	// Kolide tables
	"github.com/kolide/launcher/pkg/osquery/tables/dsim_default_associations"

	// TODO: Fix build erros
	//"github.com/kolide/launcher/pkg/osquery/tables/gsettings"
	//"github.com/kolide/launcher/pkg/osquery/tables/mdmclient"

	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func PlatformTables() []osquery.OsqueryPlugin {
	return []osquery.OsqueryPlugin{
		// Fleet tables
		table.NewPlugin("mdm_bridge", mdmbridge.Columns(), mdmbridge.Generate),
		table.NewPlugin("cis_audit", cisaudit.Columns(), cisaudit.Generate),

		// Kolide extensions.
		dsim_default_associations.TablePlugin(kolideLogger),
		// gsettings.Settings(kolideLogger),    // TODO: Fix build error
		// mdmclient.TablePlugin(kolideLogger), // TODO: Fix build error
	}
}
