package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20251121124239(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	hostUUID := "123e4567-e89b-12d3-a456-426614174000"
	query := `
		INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, fleet_challenge, status)
		VALUES (?, ?, ?, ?)
	`
	_, err := db.Exec(query, hostUUID, 1, "challenge-string", "pending")
	require.NoError(t, err)

	type HostCertificateTemplateResult struct {
		HostUUID              string
		CertificateTemplateID int
		FleetChallenge        string
		Status                string
	}

	var result HostCertificateTemplateResult
	row := db.QueryRow(`
		SELECT host_uuid, certificate_template_id, fleet_challenge, status
		FROM host_certificate_templates
		WHERE host_uuid = ?
	`, hostUUID)
	err = row.Scan(&result.HostUUID, &result.CertificateTemplateID, &result.FleetChallenge, &result.Status)
	require.NoError(t, err)

	require.Equal(t, hostUUID, result.HostUUID)
	require.Equal(t, 1, result.CertificateTemplateID)
	require.Equal(t, "challenge-string", result.FleetChallenge)
	require.Equal(t, "pending", result.Status)
}
