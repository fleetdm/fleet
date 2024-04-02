package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240119091637(t *testing.T) {
	db := applyUpToPrev(t)

	stmt := `
		INSERT INTO operating_system_vulnerabilities (host_id, operating_system_id, cve, source, resolved_in_version)
		VALUES (1, 1, 'cve-1', 0, '1.0.0')
	`
	_, err := db.Exec(stmt)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// ensure table is truncated
	stmt = `
		SELECT COUNT(*) FROM operating_system_vulnerabilities
	`
	var count int
	err = db.QueryRow(stmt).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// check new unique index
	stmt = `
		INSERT INTO operating_system_vulnerabilities (operating_system_id, cve, source, resolved_in_version)
		VALUES (1, 'cve-1', 0, '1.0.0')
	`
	_, err = db.Exec(stmt)
	require.NoError(t, err)

	stmt = `
		INSERT INTO operating_system_vulnerabilities (operating_system_id, cve, source, resolved_in_version)
		VALUES (1, 'cve-1', 0, '1.0.0')
	`
	_, err = db.Exec(stmt)
	require.Error(t, err)
}
