package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220503134048(t *testing.T) {
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
