//go:build linux
// +build linux

package table

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/crowdstrike/falcon_kernel_check"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/crowdstrike/falconctl"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/cryptsetup"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/execparsers/simple_array"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/fscrypt_info"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/gsettings"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/secureboot"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/xfconf"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/xrdb"
	"github.com/go-kit/kit/log"
	osquery "github.com/osquery/osquery-go"
)

func platformTables(client *osquery.ExtensionManagerClient, logger log.Logger, currentOsquerydBinaryPath string) []osquery.OsqueryPlugin {
	return []osquery.OsqueryPlugin{
		cryptsetup.TablePlugin(client, logger),
		gsettings.Settings(client, logger),
		gsettings.Metadata(client, logger),
		secureboot.TablePlugin(client, logger),
		xrdb.TablePlugin(client, logger),
		fscrypt_info.TablePlugin(logger),
		falcon_kernel_check.TablePlugin(logger),
		falconctl.NewFalconctlOptionTable(logger),
		xfconf.TablePlugin(logger),

		dataflattentable.TablePluginExec(client, logger,
			"kolide_nmcli_wifi", dataflattentable.KeyValueType,
			[]string{"/usr/bin/nmcli", "--mode=multiline", "--fields=all", "device", "wifi", "list"},
			dataflattentable.WithKVSeparator(":")),
		dataflattentable.TablePluginExec(client, logger, "kolide_lsblk", dataflattentable.JsonType,
			[]string{"lsblk", "-J"},
			dataflattentable.WithBinDirs("/usr/bin", "/bin"),
		),
		dataflattentable.NewExecAndParseTable(logger, "kolide_falconctl_systags", simple_array.New("systags"), []string{"/opt/CrowdStrike/falconctl", "-g", "--systags"}),
	}
}
