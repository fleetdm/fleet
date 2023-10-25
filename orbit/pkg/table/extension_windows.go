//go:build windows

package table

import (
	cisaudit "github.com/fleetdm/fleet/v4/orbit/pkg/table/cis_audit"
	mdmbridge "github.com/fleetdm/fleet/v4/orbit/pkg/table/mdm"

	// Kolide tables
	"github.com/kolide/launcher/pkg/osquery/tables/dsim_default_associations"
	"github.com/kolide/launcher/pkg/osquery/tables/secedit"
	"github.com/kolide/launcher/pkg/osquery/tables/wifi_networks"
	"github.com/kolide/launcher/pkg/osquery/tables/wmitable"

	// TODO: Fix build erros
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
		dsim_default_associations.TablePlugin(kolideLogger), // table name is "kolide_dsim_default_associations"
		secedit.TablePlugin(kolideLogger),                   // table name is "kolide_secedit"
		wifi_networks.TablePlugin(kolideLogger),             // table name is "kolide_wifi_networks"
		wmitable.TablePlugin(kolideLogger),                  // table name is "kolide_wmi"

		// TODO: Fix build error
		// mdmclient.TablePlugin(kolideLogger),
	}
}
