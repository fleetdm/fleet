package table

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/cryptoinfotable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/dev_table_tooling"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/firefox_preferences"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/tdebug"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/zfs"

	"github.com/go-kit/kit/log"
	osquery "github.com/osquery/osquery-go"
)

// PlatformTables returns all tables for the launcher build platform.
func PlatformTables(client *osquery.ExtensionManagerClient, logger log.Logger, currentOsquerydBinaryPath string) []osquery.OsqueryPlugin {
	// Common tables to all platforms
	tables := []osquery.OsqueryPlugin{
		BestPractices(client),
		ChromeLoginDataEmails(client, logger),
		ChromeUserProfiles(client, logger),
		EmailAddresses(client, logger),
		KeyInfo(client, logger),
		OnePasswordAccounts(client, logger),
		SlackConfig(client, logger),
		SshKeys(client, logger),
		cryptoinfotable.TablePlugin(logger),
		dev_table_tooling.TablePlugin(logger),
		firefox_preferences.TablePlugin(logger),
		dataflattentable.TablePluginExec(client, logger,
			"kolide_zerotier_info", dataflattentable.JsonType, zerotierCli("info")),
		dataflattentable.TablePluginExec(client, logger,
			"kolide_zerotier_networks", dataflattentable.JsonType, zerotierCli("listnetworks")),
		dataflattentable.TablePluginExec(client, logger,
			"kolide_zerotier_peers", dataflattentable.JsonType, zerotierCli("listpeers")),
		tdebug.LauncherGcInfo(client, logger),
		zfs.ZfsPropertiesPlugin(client, logger),
		zfs.ZpoolPropertiesPlugin(client, logger),
	}

	// The dataflatten tables
	tables = append(tables, dataflattentable.AllTablePlugins(client, logger)...)

	// add in the platform specific ones (as denoted by build tags)
	tables = append(tables, platformTables(client, logger, currentOsquerydBinaryPath)...)

	return tables
}
