//go:build darwin
// +build darwin

package nvram_info

import (
	"context"
	tbl_common "github.com/fleetdm/fleet/v4/orbit/pkg/table/common"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.IntegerColumn("amfi_enabled"),
		table.IntegerColumn("ssv_enabled"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	SSVEnabled, err := getSSVEnabled(ctx)
	return []map[string]string{
		{"ssv_enabled": SSVEnabled},
	}, err
}

func getSSVEnabled(ctx context.Context) (amfiEnabled string, err error) {
	res, err := runCommand(ctx, "/usr/sbin/nvram", "-p")
	amfiEnabled = ""
	if err == nil {
		amfiEnabled = "0"
		if !strings.Contains(res, "amfi_get_out_of_my_way=1") {
			amfiEnabled = "1"
		}
	}
	return amfiEnabled, err
}

func runCommand(ctx context.Context, name string, arg ...string) (res string, err error) {
	uid, gid, err := tbl_common.GetConsoleUidGid()
	if err != nil {
		log.Debug().Err(err).Msg("failed to get console user")
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, arg...)

	// Run as the current console user (otherwise we get empty results for the root user)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: uid, Gid: gid},
	}

	out, err := cmd.Output()
	if err != nil {
		log.Debug().Err(err).Msg("failed while generating nvram table")
		return "", err
	}
	return string(out), nil
}
