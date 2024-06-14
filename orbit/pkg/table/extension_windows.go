//go:build windows

package table

import (
	cisaudit "github.com/fleetdm/fleet/v4/orbit/pkg/table/cis_audit"
	mdmbridge "github.com/fleetdm/fleet/v4/orbit/pkg/table/mdm"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/windowsupdatetable"
	"golang.org/x/sys/windows/registry"

	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func PlatformTables(_ PluginOpts) []osquery.OsqueryPlugin {
	plugins := []osquery.OsqueryPlugin{
		// Fleet tables
		table.NewPlugin("cis_audit", cisaudit.Columns(), cisaudit.Generate),

		windowsupdatetable.TablePlugin(windowsupdatetable.UpdatesTable, osqueryLogger), // table name is "windows_updates"
	}

	if !IsWindowsServer() {
		plugins = append(plugins, table.NewPlugin("mdm_bridge", mdmbridge.Columns(), mdmbridge.Generate))
	}

	return plugins
}

func IsWindowsServer() bool {
	// If the registry can't be read, it's safer to assume we're a
	// server and not load the broken table
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return true
	}
	defer k.Close()

	s, _, err := k.GetStringValue("InstallationType")
	if err != nil {
		return true
	}

	if s == "Server" {
		return true
	}

	return false
}
