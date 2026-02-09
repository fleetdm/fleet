package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20260106000000(t *testing.T) {
	db := applyUpToPrev(t)

	// Create a team
	teamRes, err := db.Exec(`INSERT INTO teams (name) VALUES (?)`, "TestTeam")
	require.NoError(t, err)
	teamID, _ := teamRes.LastInsertId()

	// Create a certificate authority
	caRes, err := db.Exec(`
		INSERT INTO certificate_authorities (name, type, url)
		VALUES (?, ?, ?)
	`, "TestCA", "custom_scep_proxy", "http://localhost:8080/scep")
	require.NoError(t, err)
	caID, _ := caRes.LastInsertId()

	// Create a certificate template
	ctRes, err := db.Exec(`
		INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name)
		VALUES (?, ?, ?, ?)
	`, "TestTemplate", teamID, caID, "CN=Test")
	require.NoError(t, err)
	ctID, _ := ctRes.LastInsertId()

	// Insert host_certificate_templates record (before migration, no validity columns)
	_, err = db.Exec(`
		INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, operation_type, name)
		VALUES (?, ?, ?, ?, ?)
	`, "host-uuid-1", ctID, "verified", "install", "TestTemplate")
	require.NoError(t, err)

	// Apply current migration
	applyNext(t, db)

	// Verify the columns exist and are NULL for existing rows
	var row struct {
		HostUUID       string     `db:"host_uuid"`
		NotValidBefore *time.Time `db:"not_valid_before"`
		NotValidAfter  *time.Time `db:"not_valid_after"`
		Serial         *string    `db:"serial"`
	}
	err = db.Get(&row, `
		SELECT host_uuid, not_valid_before, not_valid_after, serial
		FROM host_certificate_templates
		WHERE host_uuid = ?
	`, "host-uuid-1")
	require.NoError(t, err)
	require.Equal(t, "host-uuid-1", row.HostUUID)
	require.Nil(t, row.NotValidBefore, "existing row should have NULL not_valid_before")
	require.Nil(t, row.NotValidAfter, "existing row should have NULL not_valid_after")
	require.Nil(t, row.Serial, "existing row should have NULL serial")

	// Verify we can insert new rows with validity columns populated
	notValidBefore := time.Now().UTC().Truncate(time.Microsecond)
	notValidAfter := notValidBefore.AddDate(1, 0, 0) // 1 year from now
	serial := "ABC123DEF456"

	_, err = db.Exec(`
		INSERT INTO host_certificate_templates
			(host_uuid, certificate_template_id, status, operation_type, name, not_valid_before, not_valid_after, serial)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "host-uuid-2", ctID, "verified", "install", "TestTemplate", notValidBefore, notValidAfter, serial)
	require.NoError(t, err)

	// Verify the new row has the values
	var newRow struct {
		NotValidBefore *time.Time `db:"not_valid_before"`
		NotValidAfter  *time.Time `db:"not_valid_after"`
		Serial         *string    `db:"serial"`
	}
	err = db.Get(&newRow, `
		SELECT not_valid_before, not_valid_after, serial
		FROM host_certificate_templates
		WHERE host_uuid = ?
	`, "host-uuid-2")
	require.NoError(t, err)
	require.NotNil(t, newRow.NotValidBefore)
	require.NotNil(t, newRow.NotValidAfter)
	require.NotNil(t, newRow.Serial)
	require.WithinDuration(t, notValidBefore, *newRow.NotValidBefore, time.Second)
	require.WithinDuration(t, notValidAfter, *newRow.NotValidAfter, time.Second)
	require.Equal(t, serial, *newRow.Serial)

	// Verify we can update existing rows with validity data
	updateNotValidBefore := time.Now().UTC().Truncate(time.Microsecond)
	updateNotValidAfter := updateNotValidBefore.AddDate(0, 6, 0) // 6 months from now
	updateSerial := "XYZ789"

	_, err = db.Exec(`
		UPDATE host_certificate_templates
		SET not_valid_before = ?, not_valid_after = ?, serial = ?
		WHERE host_uuid = ?
	`, updateNotValidBefore, updateNotValidAfter, updateSerial, "host-uuid-1")
	require.NoError(t, err)

	// Verify the update
	var updatedRow struct {
		NotValidBefore *time.Time `db:"not_valid_before"`
		NotValidAfter  *time.Time `db:"not_valid_after"`
		Serial         *string    `db:"serial"`
	}
	err = db.Get(&updatedRow, `
		SELECT not_valid_before, not_valid_after, serial
		FROM host_certificate_templates
		WHERE host_uuid = ?
	`, "host-uuid-1")
	require.NoError(t, err)
	require.NotNil(t, updatedRow.NotValidBefore)
	require.NotNil(t, updatedRow.NotValidAfter)
	require.NotNil(t, updatedRow.Serial)
	require.WithinDuration(t, updateNotValidBefore, *updatedRow.NotValidBefore, time.Second)
	require.WithinDuration(t, updateNotValidAfter, *updatedRow.NotValidAfter, time.Second)
	require.Equal(t, updateSerial, *updatedRow.Serial)

	// Verify index exists and can be used for renewal queries
	// This query simulates finding certificates expiring within 30 days
	var expiringRows []struct {
		HostUUID      string    `db:"host_uuid"`
		NotValidAfter time.Time `db:"not_valid_after"`
	}
	err = db.Select(&expiringRows, `
		SELECT host_uuid, not_valid_after
		FROM host_certificate_templates
		WHERE not_valid_after IS NOT NULL
		  AND not_valid_after < DATE_ADD(NOW(), INTERVAL 30 DAY)
		  AND status IN ('delivered', 'verified')
		  AND operation_type = 'install'
	`)
	require.NoError(t, err)
	// host-uuid-1 has cert expiring in 6 months, host-uuid-2 in 1 year - neither should match
	require.Len(t, expiringRows, 0)

	// Insert a certificate that's expiring soon
	soonExpiring := time.Now().UTC().Add(7 * 24 * time.Hour) // 7 days from now
	_, err = db.Exec(`
		INSERT INTO host_certificate_templates
			(host_uuid, certificate_template_id, status, operation_type, name, not_valid_before, not_valid_after, serial)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "host-uuid-3", ctID, "verified", "install", "TestTemplate", time.Now().UTC(), soonExpiring, "EXPIRING123")
	require.NoError(t, err)

	// Now the renewal query should find the expiring certificate
	err = db.Select(&expiringRows, `
		SELECT host_uuid, not_valid_after
		FROM host_certificate_templates
		WHERE not_valid_after IS NOT NULL
		  AND not_valid_after < DATE_ADD(NOW(), INTERVAL 30 DAY)
		  AND status IN ('delivered', 'verified')
		  AND operation_type = 'install'
	`)
	require.NoError(t, err)
	require.Len(t, expiringRows, 1)
	require.Equal(t, "host-uuid-3", expiringRows[0].HostUUID)
}
