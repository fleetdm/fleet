//go:build windows

package table

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/cis_audit"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/mdm"
	"github.com/kolide/launcher/pkg/osquery/tables/secedit"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func PlatformTables() []osquery.OsqueryPlugin {
	return []osquery.OsqueryPlugin{
		// Fleet tables
		table.NewPlugin("mdm_bridge", mdmbridge.Columns(), mdmbridge.Generate),
		table.NewPlugin("cis_audit", cisaudit.Columns(), cisaudit.Generate),

		// Kolide tables
		secedit.TablePlugin(serverClient, kolideLogger), // table name is "kolide_secedit"
	}
}
