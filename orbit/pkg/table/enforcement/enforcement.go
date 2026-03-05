package enforcement

import (
	"context"
	"strconv"

	enf "github.com/fleetdm/fleet/v4/orbit/pkg/enforcement"
	"github.com/osquery/osquery-go/plugin/table"
)

// cache is a package-level reference to the compliance cache, set at
// registration time.
var cache *enf.ComplianceCache

// SetCache sets the compliance cache that this table reads from.
func SetCache(c *enf.ComplianceCache) {
	cache = c
}

// Columns defines the schema for the fleet_windows_enforcement table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("setting_name"),
		table.TextColumn("category"),
		table.TextColumn("policy_name"),
		table.TextColumn("cis_ref"),
		table.TextColumn("desired_value"),
		table.TextColumn("current_value"),
		table.IntegerColumn("compliant"),
		table.TextColumn("last_checked"),
	}
}

// Generate is called by osquery to produce rows for the table.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	if cache == nil {
		return nil, nil
	}

	records := cache.Records()
	rows := make([]map[string]string, 0, len(records))

	for _, r := range records {
		compliant := 0
		if r.Compliant {
			compliant = 1
		}

		rows = append(rows, map[string]string{
			"setting_name":  r.SettingName,
			"category":      r.Category,
			"policy_name":   r.PolicyName,
			"cis_ref":       r.CISRef,
			"desired_value": r.DesiredValue,
			"current_value": r.CurrentValue,
			"compliant":     strconv.Itoa(compliant),
			"last_checked":  r.LastChecked.Format("2006-01-02T15:04:05Z"),
		})
	}

	return rows, nil
}
