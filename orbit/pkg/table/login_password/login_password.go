//go:build darwin
// +build darwin

package login_password

import (
	"context"
	"fmt"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/common"
	"github.com/osquery/osquery-go/plugin/table"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.IntegerColumn("password_hint_enabled"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	uid, gid, err := common.GetConsoleUidGid()
	if err != nil {
		return nil, fmt.Errorf("failed to get console user: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "dscl", ".", "-list", "/Users", "hint")

	// Run as the current console user (otherwise we get empty results for the root user)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: uid, Gid: gid},
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	res := "0"
	if len(strings.TrimSpace(string(out))) > 0 {
		res = "1"
	}
	return []map[string]string{{"password_hint_enabled": res}}, nil
}
