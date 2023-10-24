//go:build darwin

package table

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/authdb"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/csrutil_info"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/diskutil/apfs"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/diskutil/corestorage"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/dscl"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/filevault_prk"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/find_cmd"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/firmware_eficheck_integrity_check"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/nvram_info"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/pmset"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/privaterelay"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/pwd_policy"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/software_update"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/sudo_info"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/user_login_settings"

	// Kolide tables
	"github.com/kolide/launcher/pkg/osquery/tables/airport"
	"github.com/kolide/launcher/pkg/osquery/tables/apple_silicon_security_policy"
	"github.com/kolide/launcher/pkg/osquery/tables/dataflattentable"
	"github.com/kolide/launcher/pkg/osquery/tables/filevault"
	"github.com/kolide/launcher/pkg/osquery/tables/firmwarepasswd"
	"github.com/kolide/launcher/pkg/osquery/tables/ioreg"
	"github.com/kolide/launcher/pkg/osquery/tables/mdmclient"
	kolidemunki "github.com/kolide/launcher/pkg/osquery/tables/munki"
	"github.com/kolide/launcher/pkg/osquery/tables/profiles"
	"github.com/kolide/launcher/pkg/osquery/tables/pwpolicy"

	// TODO: This Kolide table requires more complicated coding
	"github.com/kolide/launcher/pkg/osquery/tables/osquery_user_exec_table"

	"github.com/macadmins/osquery-extension/tables/filevaultusers"
	"github.com/macadmins/osquery-extension/tables/macos_profiles"
	"github.com/macadmins/osquery-extension/tables/macosrsr"
	"github.com/macadmins/osquery-extension/tables/mdm"
	"github.com/macadmins/osquery-extension/tables/munki"
	"github.com/macadmins/osquery-extension/tables/unifiedlog"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

const (
	currentOsquerydBinaryPath = "/opt/orbit/bin/osqueryd/macos-app/stable/osquery.app/Contents/MacOS/osqueryd"
	keychainAclsQuery         = "select * from keychain_acls"
	keychainItemsQuery        = "select * from keychain_items"
	screenlockQuery           = "select enabled, grace_period from screenlock"
)

func PlatformTables() []osquery.OsqueryPlugin {
	plugins := []osquery.OsqueryPlugin{
		// Fleet tables
		table.NewPlugin("icloud_private_relay", privaterelay.Columns(), privaterelay.Generate),
		table.NewPlugin("user_login_settings", user_login_settings.Columns(), user_login_settings.Generate),
		table.NewPlugin("pwd_policy", pwd_policy.Columns(), pwd_policy.Generate),
		table.NewPlugin("csrutil_info", csrutil_info.Columns(), csrutil_info.Generate),
		table.NewPlugin("nvram_info", nvram_info.Columns(), nvram_info.Generate),
		table.NewPlugin("authdb", authdb.Columns(), authdb.Generate),
		table.NewPlugin("pmset", pmset.Columns(), pmset.Generate),
		table.NewPlugin("sudo_info", sudo_info.Columns(), sudo_info.Generate),
		table.NewPlugin("software_update", software_update.Columns(), software_update.Generate),
		table.NewPlugin("firmware_eficheck_integrity_check", firmware_eficheck_integrity_check.Columns(), firmware_eficheck_integrity_check.Generate),
		table.NewPlugin("dscl", dscl.Columns(), dscl.Generate),
		table.NewPlugin("apfs_volumes", apfs.VolumesColumns(), apfs.VolumesGenerate),
		table.NewPlugin("apfs_physical_stores", apfs.PhysicalStoresColumns(), apfs.PhysicalStoresGenerate),
		table.NewPlugin("corestorage_logical_volumes", corestorage.LogicalVolumesColumns(), corestorage.LogicalVolumesGenerate),
		table.NewPlugin("corestorage_logical_volume_families", corestorage.LogicalVolumeFamiliesColumns(), corestorage.LogicalVolumeFamiliesGenerate),
		table.NewPlugin("filevault_prk", filevault_prk.Columns(), filevault_prk.Generate),
		table.NewPlugin("find_cmd", find_cmd.Columns(), find_cmd.Generate),

		// Macadmins extension tables
		table.NewPlugin("filevault_users", filevaultusers.FileVaultUsersColumns(), filevaultusers.FileVaultUsersGenerate),
		table.NewPlugin("macos_profiles", macos_profiles.MacOSProfilesColumns(), macos_profiles.MacOSProfilesGenerate),
		table.NewPlugin("mdm", mdm.MDMInfoColumns(), mdm.MDMInfoGenerate),
		table.NewPlugin("munki_info", munki.MunkiInfoColumns(), munki.MunkiInfoGenerate),
		table.NewPlugin("munki_installs", munki.MunkiInstallsColumns(), munki.MunkiInstallsGenerate),
		table.NewPlugin("macos_rsr", macosrsr.MacOSRsrColumns(), macosrsr.MacOSRsrGenerate),
		// osquery version 5.5.0 and up ships a unified_log table in core
		// we are renaming the one from the macadmins extension to avoid collision
		table.NewPlugin("macadmins_unified_log", unifiedlog.UnifiedLogColumns(), unifiedlog.UnifiedLogGenerate),

		// Kolide tables
		filevault.TablePlugin(kolideLogger),
		// kolide_firmwarepasswd table. Only returns valid data on a Mac with an Intel processor. Background: https://support.apple.com/en-us/HT204455
		firmwarepasswd.TablePlugin(kolideLogger),
		ioreg.TablePlugin(kolideLogger),
		profiles.TablePlugin(kolideLogger),
		pwpolicy.TablePlugin(kolideLogger),
		airport.TablePlugin(kolideLogger),
		firmwarepasswd.TablePlugin(kolideLogger),
		apple_silicon_security_policy.TablePlugin(kolideLogger),
		mdmclient.TablePlugin(kolideLogger),
		kolidemunki.New().ManagedInstalls(kolideLogger),
		kolidemunki.New().MunkiReport(kolideLogger),
		// Tables for parsing Apple Property List files, which are typically stored in ~/Library/Preferences/
		dataflattentable.TablePlugin(kolideLogger, dataflattentable.JsonType),  // table name is "kolide_json"
		dataflattentable.TablePlugin(kolideLogger, dataflattentable.JsonlType), // table name is "kolide_jsonl"
		dataflattentable.TablePlugin(kolideLogger, dataflattentable.XmlType),   // table name is "kolide_xml"
		dataflattentable.TablePlugin(kolideLogger, dataflattentable.IniType),   // table name is "kolide_ini"
		dataflattentable.TablePlugin(kolideLogger, dataflattentable.PlistType), // table name is "kolide_plist"

		osquery_user_exec_table.TablePlugin(
			kolideLogger, "kolide_keychain_items",
			currentOsquerydBinaryPath, keychainAclsQuery,
			[]table.ColumnDefinition{
				table.TextColumn("label"),
				table.TextColumn("description"),
				table.TextColumn("comment"),
				table.TextColumn("created"),
				table.TextColumn("modified"),
				table.TextColumn("type"),
				table.TextColumn("path"),
			}),

		osquery_user_exec_table.TablePlugin(
			kolideLogger, "kolide_keychain_acls",
			currentOsquerydBinaryPath, keychainItemsQuery,
			[]table.ColumnDefinition{
				table.TextColumn("keychain_path"),
				table.TextColumn("authorizations"),
				table.TextColumn("path"),
				table.TextColumn("description"),
				table.TextColumn("label"),
			}),

		osquery_user_exec_table.TablePlugin(
			kolideLogger, "kolide_screenlock",
			currentOsquerydBinaryPath, screenlockQuery,
			[]table.ColumnDefinition{
				table.IntegerColumn("enabled"),
				table.IntegerColumn("grace_period"),
			}),
	}

	// append platform specific tables
	plugins = appendTables(plugins)

	return plugins
}
