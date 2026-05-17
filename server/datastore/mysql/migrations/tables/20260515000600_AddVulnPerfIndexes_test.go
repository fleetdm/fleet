package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260515000600(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	expected := []struct {
		table     string
		indexName string
		columns   []string
	}{
		{"cve_meta", "idx_cve_meta_exploit", []string{"cisa_known_exploit", "cve"}},
		{"cve_meta", "idx_cve_meta_cvss_score", []string{"cvss_score", "cve"}},
		{"vulnerability_host_counts", "idx_vhc_scope_cve", []string{"global_stats", "team_id", "host_count", "cve"}},
	}

	for _, e := range expected {
		rows, err := db.Query(
			"SELECT seq_in_index, column_name FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ? ORDER BY seq_in_index",
			e.table, e.indexName,
		)
		require.NoError(t, err)

		var actualColumns []string
		for rows.Next() {
			var seqInIndex int
			var columnName string
			err := rows.Scan(&seqInIndex, &columnName)
			require.NoError(t, err)
			actualColumns = append(actualColumns, columnName)
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close()) //nolint:sqlclosecheck // a defer per-iteration would leak until the loop ends; explicit close is fine in this test

		require.Equalf(
			t,
			e.columns,
			actualColumns,
			"expected index %s on %s to have columns %v in order, got %v",
			e.indexName,
			e.table,
			e.columns,
			actualColumns,
		)
	}
}
