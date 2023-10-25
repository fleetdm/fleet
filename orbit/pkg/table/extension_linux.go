//go:build linux

package table

import (
	"github.com/kolide/launcher/pkg/osquery/tables/crowdstrike/falcon_kernel_check"
	"github.com/kolide/launcher/pkg/osquery/tables/crowdstrike/falconctl"
	"github.com/kolide/launcher/pkg/osquery/tables/cryptsetup"
	"github.com/kolide/launcher/pkg/osquery/tables/gsettings"
	"github.com/osquery/osquery-go"
)

func PlatformTables() []osquery.OsqueryPlugin {
	return []osquery.OsqueryPlugin{
		// Kolide extensions.
		cryptsetup.TablePlugin(kolideLogger),
		// Linux Gnome settings and metadata
		gsettings.Settings(kolideLogger),
		gsettings.Metadata(kolideLogger),

		// Not tested:
		falconctl.NewFalconctlOptionTable(kolideLogger),
		falcon_kernel_check.TablePlugin(kolideLogger),
	}
}
