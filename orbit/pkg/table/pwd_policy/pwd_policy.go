//go:build darwin
// +build darwin

package pwd_policy

import (
	"context"
	"fmt"
	tbl_common "github.com/fleetdm/fleet/v4/orbit/pkg/table/common"
	"github.com/osquery/osquery-go/plugin/table"
	"os/exec"
	"syscall"
	"time"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.IntegerColumn("maxFailedAttempts"),
		table.IntegerColumn("expiresEveryNDays"),
		table.IntegerColumn("daysToExpiration"),
		table.IntegerColumn("historyDepth"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	uid, gid, err := tbl_common.GetConsoleUidGid()
	if err != nil {
		return nil, fmt.Errorf("failed to get console user: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pwpolicy", "-getaccountpolicies")

	// Run as the current console user (otherwise we get empty results for the root user)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: uid, Gid: gid},
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	pwpolicyXMLData := string(out)
	maxFailedAttempts, err := tbl_common.GetValFromXMLWithTags(pwpolicyXMLData, "dict", "key", "policyAttributeMaximumFailedAuthentications", "integer")
	expiresEveryNDays, err := tbl_common.GetValFromXMLWithTags(pwpolicyXMLData, "dict", "key", "policyAttributeExpiresEveryNDays", "integer")
	daysToExpiration, err := tbl_common.GetValFromXMLWithTags(pwpolicyXMLData, "dict", "key", "policyAttributeDaysUntilExpiration", "integer")
	historyDepth, err := tbl_common.GetValFromXMLWithTags(pwpolicyXMLData, "dict", "key", "policyAttributePasswordHistoryDepth", "integer")

	return []map[string]string{
		{"maxFailedAttempts": maxFailedAttempts,
			"expiresEveryNDays": expiresEveryNDays,
			"daysToExpiration":  daysToExpiration,
			"historyDepth":      historyDepth},
	}, nil
}
