//go:build darwin
// +build darwin

package pwd_policy

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	tbl_common "github.com/fleetdm/fleet/v4/orbit/pkg/table/common"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.IntegerColumn("max_failed_attempts"),
		table.IntegerColumn("expires_every_n_days"),
		table.IntegerColumn("days_to_expiration"),
		table.IntegerColumn("history_depth"),
		table.IntegerColumn("min_mixed_case_characters"),
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
	cmd := exec.CommandContext(ctx, "/usr/bin/pwpolicy", "-getaccountpolicies")

	// Run as the current console user (otherwise we get empty results for the root user)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: uid, Gid: gid},
	}

	out, err := cmd.Output()
	if err != nil {
		log.Debug().Err(err).Msg("Running pwpolicy failed")
		return nil, fmt.Errorf("running pwpolicy failed: %w", err)
	}

	pwpolicyXMLData := string(out)
	maxFailedAttempts, err := tbl_common.GetValFromXMLWithTags(pwpolicyXMLData, "dict", "key", "policyAttributeMaximumFailedAuthentications", "integer")
	if err != nil {
		maxFailedAttempts = ""
		log.Debug().Err(err).Msg("get policyAttributeMaximumFailedAuthentications failed")
	}
	expiresEveryNDays, err := tbl_common.GetValFromXMLWithTags(pwpolicyXMLData, "dict", "key", "policyAttributeExpiresEveryNDays", "integer")
	if err != nil {
		expiresEveryNDays = ""
		log.Debug().Err(err).Msg("get policyAttributeExpiresEveryNDays failed")
	}
	daysToExpiration, err := tbl_common.GetValFromXMLWithTags(pwpolicyXMLData, "dict", "key", "policyAttributeDaysUntilExpiration", "integer")
	if err != nil {
		daysToExpiration = ""
		log.Debug().Err(err).Msg("get policyAttributeDaysUntilExpiration failed")
	}
	historyDepth, err := tbl_common.GetValFromXMLWithTags(pwpolicyXMLData, "dict", "key", "policyAttributePasswordHistoryDepth", "integer")
	if err != nil {
		historyDepth = ""
		log.Debug().Err(err).Msg("get policyAttributePasswordHistoryDepth failed")
	}
	minMixedCaseCharacters, err := tbl_common.GetValFromXMLWithTags(pwpolicyXMLData, "dict", "key", "minimumMixedCaseCharacters", "integer")
	if err != nil {
		minMixedCaseCharacters = ""
		log.Debug().Err(err).Msg("get minimumMixedCaseCharacters failed")
	}

	return []map[string]string{
		{
			"max_failed_attempts":       maxFailedAttempts,
			"expires_every_n_days":      expiresEveryNDays,
			"days_to_expiration":        daysToExpiration,
			"history_depth":             historyDepth,
			"min_mixed_case_characters": minMixedCaseCharacters,
		},
	}, nil
}
