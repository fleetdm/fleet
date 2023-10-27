//go:build linux

package table

import (
	"github.com/kolide/launcher/pkg/osquery/tables/crowdstrike/falcon_kernel_check"
	"github.com/kolide/launcher/pkg/osquery/tables/crowdstrike/falconctl"
	"github.com/kolide/launcher/pkg/osquery/tables/cryptsetup"
	"github.com/kolide/launcher/pkg/osquery/tables/gsettings"
	"github.com/kolide/launcher/pkg/osquery/tables/xfconf"
	"github.com/kolide/launcher/pkg/osquery/tables/xrdb"
	"github.com/kolide/launcher/pkg/osquery/tables/zfs"
	"github.com/osquery/osquery-go"
)

func PlatformTables() []osquery.OsqueryPlugin {
	return []osquery.OsqueryPlugin{
		// Kolide extensions.
		cryptsetup.TablePlugin(kolideLogger),
		// Linux Gnome settings and metadata
		gsettings.Settings(kolideLogger),
		gsettings.Metadata(kolideLogger),
		// Wrapper for /usr/bin/xrdb command.
		xrdb.TablePlugin(kolideLogger),
		xfconf.TablePlugin(kolideLogger), // table name is "xfconf"
		falconctl.NewFalconctlOptionTable(kolideLogger),
		falcon_kernel_check.TablePlugin(kolideLogger),
		zfs.ZfsPropertiesPlugin(kolideLogger),   // table name is "zfs_properties"
		zfs.ZpoolPropertiesPlugin(kolideLogger), // table name is "zpool_properties"
	}
}
