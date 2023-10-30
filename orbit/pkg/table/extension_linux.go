//go:build linux

package table

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/crowdstrike/falcon_kernel_check"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/crowdstrike/falconctl"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/cryptsetup"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/gsettings"
	"github.com/kolide/launcher/pkg/osquery/tables/xfconf"
	"github.com/kolide/launcher/pkg/osquery/tables/xrdb"
	"github.com/kolide/launcher/pkg/osquery/tables/zfs"
	"github.com/osquery/osquery-go"
)

func PlatformTables() []osquery.OsqueryPlugin {
	return []osquery.OsqueryPlugin{
		// Kolide extensions.
		cryptsetup.TablePlugin(osqueryLogger),
		// Linux Gnome settings and metadata
		gsettings.Settings(osqueryLogger),
		gsettings.Metadata(osqueryLogger),
		// Wrapper for /usr/bin/xrdb command.
		xrdb.TablePlugin(osqueryLogger),
		xfconf.TablePlugin(osqueryLogger), // table name is "xfconf"
		falconctl.NewFalconctlOptionTable(osqueryLogger),
		falcon_kernel_check.TablePlugin(osqueryLogger),
		zfs.ZfsPropertiesPlugin(osqueryLogger),   // table name is "zfs_properties"
		zfs.ZpoolPropertiesPlugin(osqueryLogger), // table name is "zpool_properties"
	}
}
