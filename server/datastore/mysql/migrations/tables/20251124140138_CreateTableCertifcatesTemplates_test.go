package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20251124140138(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	query := `
		INSERT INTO certificate_authorities (type, name, url)
		VALUES (?, ?, ?)
	`
	caID := execNoErrLastID(t, db, query, "digicert", "Test CA", "https://fleetdm.com")

	query = `
		INSERT INTO teams (name, description)
		VALUES (?, ?)
	`
	teamID := execNoErrLastID(t, db, query, "Test Team", "Description")

	query = `
		INSERT INTO certificate_templates (team_id, certificate_authority_id, name, subject_name)
		VALUES (?, ?, ?, ?)
	`
	_, err := db.Exec(query, teamID, caID, "wifi-certificate", "CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME/OU=$FLEET_VAR_HOST_UUID/ST=$FLEET_VAR_HOST_HARDWARE_SERIAL")
	require.NoError(t, err)

	type CertificateTemplateResult struct {
		ID                     int64
		TeamID                 int64
		CertificateAuthorityID int
		Name                   string
		SubjectName            string
	}
	var result CertificateTemplateResult
	err = db.QueryRow(`
		SELECT 
			id, 
			team_id, 
			certificate_authority_id, 
			name, 
			subject_name
		FROM certificate_templates
	`).Scan(&result.ID, &result.TeamID, &result.CertificateAuthorityID, &result.Name, &result.SubjectName)
	require.NoError(t, err)
	require.Equal(t, teamID, result.TeamID)
	require.Equal(t, int(caID), result.CertificateAuthorityID)
	require.Equal(t, "wifi-certificate", result.Name)
	require.Equal(t, "CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME/OU=$FLEET_VAR_HOST_UUID/ST=$FLEET_VAR_HOST_HARDWARE_SERIAL", result.SubjectName)

	// unique constraint on (team_id, name)
	query = `
		INSERT INTO certificate_templates (team_id, certificate_authority_id, name, subject_name)
		VALUES (?, ?, ?, ?)
	`
	_, err = db.Exec(query, teamID, caID, "wifi-certificate", "CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME/OU=$FLEET_VAR_HOST_UUID/ST=$FLEET_VAR_HOST_HARDWARE_SERIAL")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Duplicate entry")

	// Should allow same name with different team_id
	query = `
		INSERT INTO teams (name, description)
		VALUES (?, ?)
	`
	teamID2 := execNoErrLastID(t, db, query, "Test Team 2", "Second team")

	query = `
		INSERT INTO certificate_templates (team_id, certificate_authority_id, name, subject_name)
		VALUES (?, ?, ?, ?)
	`
	_, err = db.Exec(query, teamID2, caID, "wifi-certificate", "CN=Template 1 Team 2")
	require.NoError(t, err)

	// Should allow different name with same team_id
	query = `
		INSERT INTO certificate_templates (team_id, certificate_authority_id, name, subject_name)
		VALUES (?, ?, ?, ?)
	`
	_, err = db.Exec(query, teamID, caID, "Template 2", "CN=Template 2")
	require.NoError(t, err)
}
