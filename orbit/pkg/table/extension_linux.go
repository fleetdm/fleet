//go:build linux

package table

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/containerd_containers"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/crowdstrike/falcon_kernel_check"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/crowdstrike/falconctl"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/cryptsetup"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/cryptsetup_luks_salt"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/dconf_read"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

func PlatformTables(_ PluginOpts) ([]osquery.OsqueryPlugin, error) {
	return []osquery.OsqueryPlugin{
		cryptsetup.TablePlugin(log.Logger),            // table name is "cryptsetup_status"
		falconctl.NewFalconctlOptionTable(log.Logger), // table name is "falconctl_option"
		falcon_kernel_check.TablePlugin(log.Logger),   // table name is "falcon_kernel_check"
		dataflattentable.TablePluginExec(log.Logger, "nftables", dataflattentable.JsonType, []string{"nft", "-jat", "list", "ruleset"}, dataflattentable.WithBinDirs("/usr/bin", "/usr/sbin")), // -j (json) -a (show object handles) -t (terse, omit set contents)
		table.NewPlugin("dconf_read", dconf_read.Columns(), dconf_read.Generate),
		table.NewPlugin("containerd_containers", containerd_containers.Columns(), containerd_containers.Generate),

		dataflattentable.TablePluginExec(
			log.Logger,
			"lsblk",
			dataflattentable.JsonType,
			[]string{"lsblk", "-n", "-O", "--json"}, // -n (no header) -O (all vars) --json (output in json)
			dataflattentable.WithBinDirs("/usr/bin", "/usr/sbin"),
		),

		table.NewPlugin(
			cryptsetup_luks_salt.TblName,
			cryptsetup_luks_salt.Columns(),
			cryptsetup_luks_salt.Generate,
		),
	}, nil
}
