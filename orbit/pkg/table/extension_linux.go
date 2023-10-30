//go:build linux

package table

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/crowdstrike/falcon_kernel_check"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/crowdstrike/falconctl"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/cryptsetup"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/gsettings"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/xfconf"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/xrdb"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/zfs"

	"github.com/osquery/osquery-go"
)

func PlatformTables() []osquery.OsqueryPlugin {
	return []osquery.OsqueryPlugin{
		cryptsetup.TablePlugin(osqueryLogger),            // table name is "cryptsetup_status"
		falconctl.NewFalconctlOptionTable(osqueryLogger), // table name is "falconctl_option"
		falcon_kernel_check.TablePlugin(osqueryLogger),   // table name is "falcon_kernel_check"
		xfconf.TablePlugin(osqueryLogger),                // table name is "xfconf"
		zfs.ZfsPropertiesPlugin(osqueryLogger),           // table name is "zfs_properties"
		zfs.ZpoolPropertiesPlugin(osqueryLogger),         // table name is "zpool_properties"

		// Linux Gnome settings and metadata
		gsettings.Settings(osqueryLogger), // table name is "gsettings"
		gsettings.Metadata(osqueryLogger), // table name is "gsettings_metadata"

		// Wrapper for /usr/bin/xrdb command.
		xrdb.TablePlugin(osqueryLogger), // table name is "xrdb"
	}
}
