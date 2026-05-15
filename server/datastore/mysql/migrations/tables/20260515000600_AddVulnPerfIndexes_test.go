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
	}{
		{"cve_meta", "idx_cve_meta_exploit"},
		{"cve_meta", "idx_cve_meta_cvss_score"},
		{"vulnerability_host_counts", "idx_vhc_scope_cve"},
	}

	for _, e := range expected {
		var count int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?",
			e.table, e.indexName,
		).Scan(&count)
		require.NoError(t, err)
		require.Greater(t, count, 0, "expected index %s on %s to exist", e.indexName, e.table)
	}
}
