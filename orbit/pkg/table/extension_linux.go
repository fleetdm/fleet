//go:build linux

package table

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/crowdstrike/falcon_kernel_check"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/crowdstrike/falconctl"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/cryptsetup"

	"github.com/osquery/osquery-go"
)

func PlatformTables(_ PluginOpts) ([]osquery.OsqueryPlugin, error) {
	return []osquery.OsqueryPlugin{
		cryptsetup.TablePlugin(osqueryLogger),            // table name is "cryptsetup_status"
		falconctl.NewFalconctlOptionTable(osqueryLogger), // table name is "falconctl_option"
		falcon_kernel_check.TablePlugin(osqueryLogger),   // table name is "falcon_kernel_check"
	}, nil
}
