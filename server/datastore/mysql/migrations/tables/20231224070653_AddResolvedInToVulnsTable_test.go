package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20231224070653(t *testing.T) {
	db := applyUpToPrev(t)

	insertStmt := `
		INSERT INTO operating_system_vulnerabilities 
		(host_id, operating_system_id, cve, source) 
		VALUES (?, ?, ?, ?)
		`

	_, err := db.Exec(insertStmt, 1, 1, "cve-1", 0)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	selectStmt := `
		SELECT host_id, operating_system_id, cve, source, resolved_in_version, updated_at, created_at
		FROM operating_system_vulnerabilities
		WHERE operating_system_id = ?
	`
	var osv struct {
		HostID            uint    `db:"host_id"`
		OperatingSystemID uint    `db:"operating_system_id"`
		CVE               string  `db:"cve"`
		Source            int     `db:"source"`
		ResolvedIn        *string `db:"resolved_in_version"`
		UpdatedAt         string  `db:"updated_at"`
		CreatedAt         string  `db:"created_at"`
	}
	err = db.Get(&osv, selectStmt, 1)
	require.NoError(t, err)
	require.Equal(t, uint(1), osv.HostID)
	require.Equal(t, uint(1), osv.OperatingSystemID)
	require.Equal(t, "cve-1", osv.CVE)
	require.Equal(t, 0, osv.Source)
	require.Nil(t, osv.ResolvedIn)
	require.NotEmpty(t, osv.UpdatedAt)

	// Insert a new row.
	newInsertStmt := `
		INSERT INTO operating_system_vulnerabilities
		(host_id, operating_system_id, cve, source, resolved_in_version)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err = db.Exec(newInsertStmt, 2, 2, "cve-2", 0, "1.2.3")
	require.NoError(t, err)

	err = db.Get(&osv, selectStmt, 2)
	require.NoError(t, err)
	require.Equal(t, uint(2), osv.HostID)
	require.Equal(t, uint(2), osv.OperatingSystemID)
	require.Equal(t, "cve-2", osv.CVE)
	require.Equal(t, 0, osv.Source)
	require.Equal(t, "1.2.3", *osv.ResolvedIn)
	require.NotEmpty(t, osv.UpdatedAt)

	updateStmt := `
		UPDATE operating_system_vulnerabilities
		SET resolved_in_version = ?
		WHERE operating_system_id = ? AND cve = ? AND host_id = ?
	`
	_, err = db.Exec(updateStmt, "1.2.4", 2, "cve-2", 2)
	require.NoError(t, err)

	err = db.Get(&osv, selectStmt, 2)
	require.NoError(t, err)
	require.Equal(t, uint(2), osv.HostID)
	require.Equal(t, uint(2), osv.OperatingSystemID)
	require.Equal(t, "cve-2", osv.CVE)
	require.Equal(t, 0, osv.Source)
	require.Equal(t, "1.2.4", *osv.ResolvedIn)
	require.NotEmpty(t, osv.UpdatedAt)
}
