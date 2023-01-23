//go:build darwin
// +build darwin

package table

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/airport"
	appicons "github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/app-icons"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/apple_silicon_security_policy"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/execparsers/remotectl"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/filevault"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/firmwarepasswd"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/ioreg"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/macos_software_update"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/mdmclient"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/munki"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/osquery_user_exec_table"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/profiles"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/pwpolicy"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/systemprofiler"
	"github.com/go-kit/kit/log"
	"github.com/knightsc/system_policy/osquery/table/kextpolicy"
	"github.com/knightsc/system_policy/osquery/table/legacyexec"
	_ "github.com/mattn/go-sqlite3"
	osquery "github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

const (
	keychainAclsQuery  = "select * from keychain_acls"
	keychainItemsQuery = "select * from keychain_items"
	screenlockQuery    = "select enabled, grace_period from screenlock"
)

func platformTables(client *osquery.ExtensionManagerClient, logger log.Logger, currentOsquerydBinaryPath string) []osquery.OsqueryPlugin {
	munki := munki.New()

	// This table uses undocumented APIs, There is some discussion at the
	// PR adding the table. See
	// https://github.com/osquery/osquery/pull/6243
	screenlockTable := osquery_user_exec_table.TablePlugin(
		client, logger, "kolide_screenlock",
		currentOsquerydBinaryPath, screenlockQuery,
		[]table.ColumnDefinition{
			table.IntegerColumn("enabled"),
			table.IntegerColumn("grace_period"),
		})

	keychainAclsTable := osquery_user_exec_table.TablePlugin(
		client, logger, "kolide_keychain_acls",
		currentOsquerydBinaryPath, keychainItemsQuery,
		[]table.ColumnDefinition{
			table.TextColumn("keychain_path"),
			table.TextColumn("authorizations"),
			table.TextColumn("path"),
			table.TextColumn("description"),
			table.TextColumn("label"),
		})

	keychainItemsTable := osquery_user_exec_table.TablePlugin(
		client, logger, "kolide_keychain_items",
		currentOsquerydBinaryPath, keychainAclsQuery,
		[]table.ColumnDefinition{
			table.TextColumn("label"),
			table.TextColumn("description"),
			table.TextColumn("comment"),
			table.TextColumn("created"),
			table.TextColumn("modified"),
			table.TextColumn("type"),
			table.TextColumn("path"),
		})

	return []osquery.OsqueryPlugin{
		keychainAclsTable,
		keychainItemsTable,
		Airdrop(client),
		appicons.AppIcons(),
		ChromeLoginKeychainInfo(client, logger),
		firmwarepasswd.TablePlugin(client, logger),
		GDriveSyncConfig(client, logger),
		GDriveSyncHistoryInfo(client, logger),
		KolideVulnerabilities(client, logger),
		MDMInfo(logger),
		macos_software_update.MacOSUpdate(client),
		macos_software_update.RecommendedUpdates(logger),
		macos_software_update.AvailableProducts(logger),
		MachoInfo(),
		Spotlight(),
		TouchIDUserConfig(client, logger),
		TouchIDSystemConfig(client, logger),
		UserAvatar(logger),
		ioreg.TablePlugin(client, logger),
		profiles.TablePlugin(client, logger),
		airport.TablePlugin(client, logger),
		kextpolicy.TablePlugin(),
		filevault.TablePlugin(client, logger),
		mdmclient.TablePlugin(client, logger),
		apple_silicon_security_policy.TablePlugin(logger),
		legacyexec.TablePlugin(),
		dataflattentable.TablePluginExec(client, logger,
			"kolide_diskutil_list", dataflattentable.PlistType, []string{"/usr/sbin/diskutil", "list", "-plist"}),
		dataflattentable.TablePluginExec(client, logger,
			"kolide_falconctl_stats", dataflattentable.PlistType, []string{"/Applications/Falcon.app/Contents/Resources/falconctl", "stats", "-p"}),
		dataflattentable.TablePluginExec(client, logger,
			"kolide_apfs_list", dataflattentable.PlistType, []string{"/usr/sbin/diskutil", "apfs", "list", "-plist"}),
		dataflattentable.TablePluginExec(client, logger,
			"kolide_apfs_users", dataflattentable.PlistType, []string{"/usr/sbin/diskutil", "apfs", "listUsers", "/", "-plist"}),
		dataflattentable.TablePluginExec(client, logger,
			"kolide_tmutil_destinationinfo", dataflattentable.PlistType, []string{"/usr/bin/tmutil", "destinationinfo", "-X"}),
		dataflattentable.TablePluginExec(client, logger,
			"kolide_powermetrics", dataflattentable.PlistType, []string{"/usr/bin/powermetrics", "-n", "1", "-f", "plist"}),
		screenlockTable,
		pwpolicy.TablePlugin(client, logger),
		systemprofiler.TablePlugin(client, logger),
		munki.ManagedInstalls(client, logger),
		munki.MunkiReport(client, logger),
		dataflattentable.NewExecAndParseTable(logger, "kolide_remotectl", remotectl.Parser, []string{`/usr/libexec/remotectl`, `dumpstate`}),
	}
}
