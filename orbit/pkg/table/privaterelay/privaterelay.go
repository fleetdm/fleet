//go:build darwin
// +build darwin

package privaterelay

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.IntegerColumn("status"),
	}
}

// Generate is called to return the results for the table at query time.
//
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	uid, gid, err := getConsoleUidGid()
	if err != nil {
		return nil, fmt.Errorf("failed to get console user: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(
		ctx,
		"bash", "-c",
		// This seems a bit brittle, but it works as of now. When it breaks we might want to look
		// into finding a more resilient mechanism for getting the status. This method discovered by
		// @sharvilshah.
		`defaults export com.apple.networkserviceproxy - | plutil -extract NSPServiceStatusManagerInfo raw - -o - | base64 -D | plutil -convert xml1 - -o - | plutil -p - | grep '"PrivacyProxyServiceStatus" =>' | head -1`,
	)
	// Run as the current console user (otherwise we get empty results for the root user)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: uid, Gid: gid},
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	switch s := strings.TrimSpace(string(out)); s {
	case `"PrivacyProxyServiceStatus" => 0`:
		return []map[string]string{{"status": "0"}}, nil

	case `"PrivacyProxyServiceStatus" => 1`:
		return []map[string]string{{"status": "1"}}, nil

	default:
		return nil, fmt.Errorf("failed to parse: '%s'", s)
	}
}

// getActiveUserGroup gets the uid and gid of the current (or more accurately, most recently logged
// in) *console* user. In most scenarios this should be the currently logged in user on the system.
// Note that getting the current user of the Orbit process is typically going to return root and we
// need the underlying user.
func getConsoleUidGid() (uid uint32, gid uint32, err error) {
	info, err := os.Stat("/dev/console")
	if err != nil {
		return 0, 0, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, fmt.Errorf("unexpected type %T", info.Sys())
	}
	return stat.Uid, stat.Gid, nil
}
