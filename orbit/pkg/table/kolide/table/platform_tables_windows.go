//go:build windows
// +build windows

package table

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/dsim_default_associations"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/execparsers/dsregcmd"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/secedit"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/wifi_networks"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/windowsupdatetable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/wmitable"

	"github.com/go-kit/kit/log"
	osquery "github.com/osquery/osquery-go"
)

func platformTables(client *osquery.ExtensionManagerClient, logger log.Logger, currentOsquerydBinaryPath string) []osquery.OsqueryPlugin {
	return []osquery.OsqueryPlugin{
		ProgramIcons(),
		dsim_default_associations.TablePlugin(client, logger),
		secedit.TablePlugin(client, logger),
		wifi_networks.TablePlugin(client, logger),
		windowsupdatetable.TablePlugin(windowsupdatetable.UpdatesTable, client, logger),
		windowsupdatetable.TablePlugin(windowsupdatetable.HistoryTable, client, logger),
		wmitable.TablePlugin(client, logger),
		dataflattentable.NewExecAndParseTable(logger, "kolide_dsregcmd", dsregcmd.Parser, []string{`/Windows/System32/dsregcmd.exe`, `/status`}),
	}
}
