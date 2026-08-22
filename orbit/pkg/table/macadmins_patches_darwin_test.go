//go:build darwin

package table

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDropNDJSONSummaryRow(t *testing.T) {
	logRow := func(traceID, eventType, timestamp string) map[string]string {
		return map[string]string{
			"trace_id":   traceID,
			"event_type": eventType,
			"timestamp":  timestamp,
			"last":       "1h",
		}
	}
	// "last" comes from the query context, so it is set even on the summary row.
	summaryRow := logRow("0", "", "")

	t.Run("drops the trailing summary row", func(t *testing.T) {
		rows := []map[string]string{
			logRow("100", "logEvent", "2026-08-19 07:48:33.536689-0400"),
			logRow("200", "activityCreateEvent", "2026-08-19 07:48:34.536689-0400"),
			summaryRow,
		}

		got := dropNDJSONSummaryRow(rows)

		require.Len(t, got, 2)
		require.Equal(t, "100", got[0]["trace_id"])
		require.Equal(t, "200", got[1]["trace_id"])
	})

	t.Run("leaves rows alone when there is no summary row", func(t *testing.T) {
		rows := []map[string]string{
			logRow("100", "logEvent", "2026-08-19 07:48:33.536689-0400"),
		}

		require.Len(t, dropNDJSONSummaryRow(rows), 1)
	})

	t.Run("only the last row is eligible", func(t *testing.T) {
		rows := []map[string]string{
			summaryRow,
			logRow("100", "logEvent", "2026-08-19 07:48:33.536689-0400"),
		}

		require.Len(t, dropNDJSONSummaryRow(rows), 2)
	})

	t.Run("handles an empty result", func(t *testing.T) {
		require.Empty(t, dropNDJSONSummaryRow(nil))
	})
}
