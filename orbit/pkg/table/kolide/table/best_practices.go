package table

import (
	"context"
	"fmt"

	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

// bestPracticesSimpleColumns is a map of the best practices columns that are
// "simple" to generate. The keys are the column names, and the values are the
// associated queries. Any practice that can be defined by a query returning a
// single row with an integer "1" for compliant, or "0" for non-compliant can
// be added to this map and automatically included in the best practices table.
// This should be treated as const.
var bestPracticesSimpleColumns = map[string]string{
	"sip_enabled":        "SELECT enabled AS compliant FROM sip_config WHERE config_flag='sip'",
	"gatekeeper_enabled": "SELECT assessments_enabled AS compliant FROM gatekeeper",
	"filevault_enabled":  "SELECT de.encrypted AS compliant FROM mounts m join disk_encryption de ON m.device_alias = de.name WHERE m.path = '/'",
	"firewall_enabled":   "SELECT global_state AS compliant FROM alf",
	// Sharing prefs
	"screen_sharing_disabled":      "SELECT screen_sharing = 0 AS compliant FROM sharing_preferences",
	"file_sharing_disabled":        "SELECT file_sharing = 0 AS compliant FROM sharing_preferences",
	"printer_sharing_disabled":     "SELECT printer_sharing = 0 AS compliant FROM sharing_preferences",
	"remote_login_disabled":        "SELECT remote_login = 0 AS compliant FROM sharing_preferences",
	"remote_management_disabled":   "SELECT remote_management = 0 AS compliant FROM sharing_preferences",
	"remote_apple_events_disabled": "SELECT remote_apple_events = 0 AS compliant FROM sharing_preferences",
	"internet_sharing_disabled":    "SELECT internet_sharing = 0 AS compliant FROM sharing_preferences",
	"bluetooth_sharing_disabled":   "SELECT bluetooth_sharing = 0 AS compliant FROM sharing_preferences",
	"disc_sharing_disabled":        "SELECT disc_sharing = 0 AS compliant FROM sharing_preferences",
}

func BestPractices(client *osquery.ExtensionManagerClient) *table.Plugin {
	columns := []table.ColumnDefinition{}
	for col := range bestPracticesSimpleColumns {
		columns = append(columns, table.IntegerColumn(col))
	}

	return table.NewPlugin("kolide_best_practices", columns, generateBestPractices(client))
}

func generateBestPractices(client *osquery.ExtensionManagerClient) table.GenerateFunc {
	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		res := map[string]string{}
		// Add all of the "simple" columns
		for col, query := range bestPracticesSimpleColumns {
			row, err := client.QueryRow(query)
			if err != nil {
				return nil, fmt.Errorf("query %s: %w", col, err)
			}
			val, ok := row["compliant"]
			if !ok {
				return nil, fmt.Errorf("query %s did not have 'compliant' column", col)
			}
			res[col] = val
		}

		return []map[string]string{res}, nil
	}
}
