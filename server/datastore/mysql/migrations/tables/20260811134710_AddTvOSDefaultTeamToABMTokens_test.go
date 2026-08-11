package tables

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260811134710(t *testing.T) {
	db := applyUpToPrev(t)

	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('Apple TVs')`)
	tokenID := execNoErrLastID(t, db, `
		INSERT INTO abm_tokens (organization_name, apple_id, renew_at, token, enrollment_url_token)
		VALUES ('org', 'apple@example.com', NOW(), 'tok', ?)`,
		// abm_tokens_enroll_url_length requires more than 32 bytes.
		strings.Repeat("a", 40))

	applyNext(t, db)

	// Existing tokens default to "No team".
	var tvOSTeamID *uint
	require.NoError(t, db.Get(&tvOSTeamID,
		`SELECT tvos_default_team_id FROM abm_tokens WHERE id = ?`, tokenID))
	require.Nil(t, tvOSTeamID)

	execNoErr(t, db, `UPDATE abm_tokens SET tvos_default_team_id = ? WHERE id = ?`, teamID, tokenID)
	require.NoError(t, db.Get(&tvOSTeamID,
		`SELECT tvos_default_team_id FROM abm_tokens WHERE id = ?`, tokenID))
	require.NotNil(t, tvOSTeamID)
	require.EqualValues(t, teamID, *tvOSTeamID)

	// Deleting the team must not delete the token, just unset the default.
	execNoErr(t, db, `DELETE FROM teams WHERE id = ?`, teamID)
	require.NoError(t, db.Get(&tvOSTeamID,
		`SELECT tvos_default_team_id FROM abm_tokens WHERE id = ?`, tokenID))
	require.Nil(t, tvOSTeamID)
}
