//go:build darwin

package table

import (
	"context"

	"github.com/macadmins/osquery-extension/tables/unifiedlog"
	"github.com/osquery/osquery-go/plugin/table"
)

// unifiedLogGenerate patches macadmins_unified_log.
//
// Upstream switched `log show` from `--style json` to `--style ndjson`. The
// ndjson stream ends with a summary record ({"count":N,"finished":1}) that
// carries none of the log fields, and upstream decodes every record in the
// stream into a row, so each query returns one extra all-empty row.
//
// See https://github.com/macadmins/osquery-extension/issues/120
func unifiedLogGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	rows, err := unifiedlog.UnifiedLogGenerate(ctx, queryContext)
	if err != nil {
		return nil, err
	}
	return dropNDJSONSummaryRow(rows), nil
}

// dropNDJSONSummaryRow checks only the last row, and only for the absence of
// every log field, so a genuine entry is never dropped.
func dropNDJSONSummaryRow(rows []map[string]string) []map[string]string {
	if len(rows) == 0 {
		return rows
	}
	last := rows[len(rows)-1]
	if last["trace_id"] == "0" && last["event_type"] == "" && last["timestamp"] == "" {
		return rows[:len(rows)-1]
	}
	return rows
}
