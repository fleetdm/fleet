//go:build windows

package table

import (
	"fmt"

	cisaudit "github.com/fleetdm/fleet/v4/orbit/pkg/table/cis_audit"
	mdmbridge "github.com/fleetdm/fleet/v4/orbit/pkg/table/mdm"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/windowsupdatetable"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows/registry"

	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func PlatformTables(_ PluginOpts) ([]osquery.OsqueryPlugin, error) {
	plugins := []osquery.OsqueryPlugin{
		// Fleet tables
		table.NewPlugin("cis_audit", cisaudit.Columns(), cisaudit.Generate),

		windowsupdatetable.TablePlugin(windowsupdatetable.UpdatesTable, log.Logger), // table name is "windows_updates"
	}

	windowsServer, err := IsWindowsServer()
	if err != nil {
		return nil, err
	}

	if !windowsServer {
		plugins = append(plugins, table.NewPlugin("mdm_bridge", mdmbridge.Columns(), mdmbridge.Generate))
	}

	return plugins, nil
}

func IsWindowsServer() (bool, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return false, fmt.Errorf("windows server check: %w", err)
	}
	defer k.Close()

	s, _, err := k.GetStringValue("InstallationType")
	if err != nil {
		return false, fmt.Errorf("windows server check: %w", err)
	}

	if s == "Server" {
		return true, nil
	}

	return false, nil
}
