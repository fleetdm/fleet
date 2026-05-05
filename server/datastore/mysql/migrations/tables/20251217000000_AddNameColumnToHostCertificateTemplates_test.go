package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20251217000000(t *testing.T) {
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

	// Create certificate templates
	ctRes1, err := db.Exec(`
		INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name)
		VALUES (?, ?, ?, ?)
	`, "TemplateName1", teamID, caID, "CN=Test1")
	require.NoError(t, err)
	ctID1, _ := ctRes1.LastInsertId()

	ctRes2, err := db.Exec(`
		INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name)
		VALUES (?, ?, ?, ?)
	`, "TemplateName2", teamID, caID, "CN=Test2")
	require.NoError(t, err)
	ctID2, _ := ctRes2.LastInsertId()

	// Insert host_certificate_templates records (before migration, no name column)
	_, err = db.Exec(`
		INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, operation_type)
		VALUES (?, ?, ?, ?), (?, ?, ?, ?)
	`,
		"host-uuid-1", ctID1, "pending", "install",
		"host-uuid-2", ctID2, "delivered", "install",
	)
	require.NoError(t, err)

	// Apply current migration
	applyNext(t, db)

	// Verify the name column exists and was populated from certificate_templates
	var rows []struct {
		HostUUID              string `db:"host_uuid"`
		CertificateTemplateID int64  `db:"certificate_template_id"`
		Name                  string `db:"name"`
	}
	err = db.Select(&rows, `
		SELECT host_uuid, certificate_template_id, name
		FROM host_certificate_templates
		ORDER BY host_uuid
	`)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	// Verify names were populated from the certificate_templates table
	require.Equal(t, "host-uuid-1", rows[0].HostUUID)
	require.Equal(t, ctID1, rows[0].CertificateTemplateID)
	require.Equal(t, "TemplateName1", rows[0].Name)

	require.Equal(t, "host-uuid-2", rows[1].HostUUID)
	require.Equal(t, ctID2, rows[1].CertificateTemplateID)
	require.Equal(t, "TemplateName2", rows[1].Name)

	// Verify we can insert new rows with the name column
	_, err = db.Exec(`
		INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, operation_type, name)
		VALUES (?, ?, ?, ?, ?)
	`, "host-uuid-3", ctID1, "pending", "install", "ManualName")
	require.NoError(t, err)

	// Verify the new row
	var newRow struct {
		Name string `db:"name"`
	}
	err = db.Get(&newRow, `SELECT name FROM host_certificate_templates WHERE host_uuid = ?`, "host-uuid-3")
	require.NoError(t, err)
	require.Equal(t, "ManualName", newRow.Name)
}
