//go:build darwin
// +build darwin

package file_system_permissions

import (
	"context"
	"fmt"
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
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	uid, gid, err := tbl_common.GetConsoleUidGid()
	if err != nil {
		log.Debug().Err(err).Msg("failed to get console user")
		return nil, fmt.Errorf("failed to get console user: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/sbin/nvram", "-p")

	// Run as the current console user (otherwise we get empty results for the root user)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: uid, Gid: gid},
	}

	out, err := cmd.Output()
	if err != nil {
		log.Debug().Err(err).Msg("Running nvram failed")
		return nil, fmt.Errorf("running nvram failed: %w", err)
	}

	outstr := string(out)
	res := 0
	if !strings.Contains(outstr, "amfi_get_out_of_my_way=1") {
		res = 1
	}

	return []map[string]string{
		{"amfi_enabled": res},
	}, nil
}
