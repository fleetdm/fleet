//go:build windows

package table

import (
	cisaudit "github.com/fleetdm/fleet/v4/orbit/pkg/table/cis_audit"
	mdmbridge "github.com/fleetdm/fleet/v4/orbit/pkg/table/mdm"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/windowsupdatetable"

	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func PlatformTables(_ PluginOpts) []osquery.OsqueryPlugin {
	return []osquery.OsqueryPlugin{
		// Fleet tables
		table.NewPlugin("mdm_bridge", mdmbridge.Columns(), mdmbridge.Generate),
		table.NewPlugin("cis_audit", cisaudit.Columns(), cisaudit.Generate),

		windowsupdatetable.TablePlugin(windowsupdatetable.UpdatesTable, osqueryLogger), // table name is "windows_updates"
	}
}
