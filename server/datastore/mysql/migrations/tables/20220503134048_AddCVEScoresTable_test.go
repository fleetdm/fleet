package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220503134048(t *testing.T) {
	// skipping old migration tests as migrations don't change and we're getting
	// timeouts in CI
	t.Skip("old migration test, not longer required to run")
	db := applyUpToPrev(t)

	applyNext(t, db)

	query := `
INSERT INTO cve_scores (
    cve,
    cvss_score,
    epss_probability,
    cisa_known_exploit
)
VALUES (?, ?, ?, ?)
`
	_, err := db.Exec(query, "CVE-2022-29464", 9.8, 0.63387, true)
	require.NoError(t, err)
}
