package tables

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20250318122638(t *testing.T) {
	db := applyUpToPrev(t)

	devID := uuid.NewString()
	execNoErr(t, db, `INSERT INTO nano_devices (id, authenticate)
		VALUES (?, ?)`, devID, "auth")

	tmID := execNoErrLastID(t, db, `INSERT INTO teams (name)
		VALUES (?)`, uuid.NewString())

	applyNext(t, db)

	// existing device is still there without a platform and a null team id
	var rows []struct {
		ID           string `db:"id"`
		Platform     string `db:"platform"`
		EnrollTeamID *uint  `db:"enroll_team_id"`
	}
	require.NoError(t, db.Select(&rows, `SELECT id, platform, enroll_team_id FROM nano_devices`))
	require.Len(t, rows, 1)
	require.Empty(t, rows[0].Platform)
	require.Nil(t, rows[0].EnrollTeamID)

	// set the enroll team to the existing team and the platform to iphone
	execNoErr(t, db, `UPDATE nano_devices SET enroll_team_id = ?, platform = ? WHERE id = ?`, tmID, "iphone", devID)
	require.NoError(t, db.Select(&rows, `SELECT id, platform, enroll_team_id FROM nano_devices`))
	require.Len(t, rows, 1)
	require.Equal(t, "iphone", rows[0].Platform)
	require.NotNil(t, rows[0].EnrollTeamID)
	require.EqualValues(t, tmID, *rows[0].EnrollTeamID)

	// deleting the team nulls the enroll team id field
	execNoErr(t, db, `DELETE FROM teams WHERE id = ?`, tmID)
	require.NoError(t, db.Select(&rows, `SELECT id, platform, enroll_team_id FROM nano_devices`))
	require.Len(t, rows, 1)
	require.Equal(t, "iphone", rows[0].Platform)
	require.Nil(t, rows[0].EnrollTeamID)
}
