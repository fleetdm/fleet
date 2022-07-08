package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20220526123327(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	query := `
INSERT INTO cve_meta (
    cve,
    cvss_score,
    epss_probability,
    cisa_known_exploit,
    published
)
VALUES (?, ?, ?, ?, ?)
`
	_, err := db.Exec(query, "CVE-2022-29464", 9.8, 0.63387, true, time.Now())
	require.NoError(t, err)
}
