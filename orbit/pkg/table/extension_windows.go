//go:build windows

package table

import (
	cisaudit "github.com/fleetdm/fleet/v4/orbit/pkg/table/cis_audit"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/dsim_default_associations"
	mdmbridge "github.com/fleetdm/fleet/v4/orbit/pkg/table/mdm"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/secedit"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/windowsupdatetable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/wmitable"

	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func PlatformTables() []osquery.OsqueryPlugin {
	return []osquery.OsqueryPlugin{
		// Fleet tables
		table.NewPlugin("mdm_bridge", mdmbridge.Columns(), mdmbridge.Generate),
		table.NewPlugin("cis_audit", cisaudit.Columns(), cisaudit.Generate),

		dsim_default_associations.TablePlugin(osqueryLogger),                           // table name is "dsim_default_associations"
		secedit.TablePlugin(osqueryLogger),                                             // table name is "secedit"
		wmitable.TablePlugin(osqueryLogger),                                            // table name is "wmi"
		windowsupdatetable.TablePlugin(windowsupdatetable.UpdatesTable, osqueryLogger), // table name is "windows_updates"
	}
}
