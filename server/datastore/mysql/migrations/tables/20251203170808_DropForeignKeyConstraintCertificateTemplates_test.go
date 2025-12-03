package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20251203170808(t *testing.T) {
	db := applyUpToPrev(t)

	// Create a team
	team1ID := uint(execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('team1');`)) //nolint:gosec // dismiss G115

	// Create a certificate authority
	caID := uint(execNoErrLastID(t, db, //nolint:gosec // dismiss G115
		`INSERT INTO certificate_authorities (name, type, url)
		VALUES ('Test CA', 'custom_scep_proxy', 'https://test-ca.example.com');`,
	))

	// team 1 certificate template
	cert1Team1 := uint(execNoErrLastID(t, db, //nolint:gosec // dismiss G115
		`INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name)
		VALUES ('Team1 Cert', ?, ?, 'CN=test1');`,
		team1ID, caID,
	))

	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM certificate_templates WHERE team_id = ?;`, team1ID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	applyNext(t, db)

	// Team id 0
	certNoTeam := uint(execNoErrLastID(t, db, //nolint:gosec // dismiss G115
		`INSERT INTO certificate_templates (name, team_id, certificate_authority_id, subject_name)
		VALUES ('No Team Cert', 0, ?, 'CN=noteam');`,
		caID,
	))

	var noTeamCert struct {
		ID                     uint   `db:"id"`
		Name                   string `db:"name"`
		TeamID                 uint   `db:"team_id"`
		CertificateAuthorityID uint   `db:"certificate_authority_id"`
		SubjectName            string `db:"subject_name"`
	}
	err = db.Get(&noTeamCert, `SELECT id, name, team_id, certificate_authority_id, subject_name FROM certificate_templates WHERE id = ?;`, certNoTeam)
	require.NoError(t, err)
	require.Equal(t, certNoTeam, noTeamCert.ID)
	require.Equal(t, "No Team Cert", noTeamCert.Name)
	require.Equal(t, uint(0), noTeamCert.TeamID)
	require.Equal(t, caID, noTeamCert.CertificateAuthorityID)

	execNoErr(t, db, `DELETE FROM teams WHERE id = ?;`, team1ID)

	var teamCerts []struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}
	err = db.Select(&teamCerts, `SELECT id, name FROM certificate_templates WHERE id = ?;`, cert1Team1)
	require.NoError(t, err)
	require.Len(t, teamCerts, 1)

	err = db.Get(&count, `SELECT COUNT(*) FROM certificate_templates WHERE team_id = 0;`)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
