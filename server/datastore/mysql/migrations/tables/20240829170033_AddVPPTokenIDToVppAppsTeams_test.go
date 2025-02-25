package tables

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20240829170033_Existing(t *testing.T) {
	db := applyUpToPrev(t)

	// insert a vpp token
	vppTokenID := execNoErrLastID(t, db, "INSERT INTO vpp_tokens (organization_name, location, renew_at, token) VALUES (?, ?, ?, ?)", "org", "location", time.Now(), "token")

	// create a couple teams
	tm1 := execNoErrLastID(t, db, "INSERT INTO teams (name) VALUES ('team1')")
	tm2 := execNoErrLastID(t, db, "INSERT INTO teams (name) VALUES ('team2')")

	// create a couple of vpp apps
	adamID1 := "123"
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES (?, "iOS")`, adamID1)
	adamID2 := "456"
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES (?, "iOS")`, adamID2)

	// insert some teams with vpp apps
	execNoErr(t, db, `INSERT INTO vpp_apps_teams (adam_id, team_id, global_or_team_id, platform, self_service) VALUES (?, ?, ?, ?, ?)`, adamID1, tm1, 0, "iOS", 0)
	execNoErr(t, db, `INSERT INTO vpp_apps_teams (adam_id, team_id, global_or_team_id, platform, self_service) VALUES (?, ?, ?, ?, ?)`, adamID2, tm2, 0, "iOS", 0)

	// apply current migration
	applyNext(t, db)

	// ensure vpp_token_id is set for all teams
	var vppTokenIDs []int
	err := sqlx.Select(db, &vppTokenIDs, `SELECT vpp_token_id FROM vpp_apps_teams`)
	require.NoError(t, err)
	require.Len(t, vppTokenIDs, 2)
	for _, tokenID := range vppTokenIDs {
		require.Equal(t, int(vppTokenID), tokenID)
	}
}

func TestUp_20240829170033_NoTokens(t *testing.T) {
	db := applyUpToPrev(t)

	// create a couple teams
	tm1 := execNoErrLastID(t, db, "INSERT INTO teams (name) VALUES ('team1')")
	tm2 := execNoErrLastID(t, db, "INSERT INTO teams (name) VALUES ('team2')")

	// create a couple of vpp apps
	adamID1 := "123"
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES (?, "iOS")`, adamID1)
	adamID2 := "456"
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES (?, "iOS")`, adamID2)

	// insert some teams with vpp apps
	execNoErr(t, db, `INSERT INTO vpp_apps_teams (adam_id, team_id, global_or_team_id, platform, self_service) VALUES (?, ?, ?, ?, ?)`, adamID1, tm1, 0, "iOS", 0)
	execNoErr(t, db, `INSERT INTO vpp_apps_teams (adam_id, team_id, global_or_team_id, platform, self_service) VALUES (?, ?, ?, ?, ?)`, adamID2, tm2, 0, "iOS", 0)

	// apply current migration
	applyNext(t, db)

	// ensure no rows are left in vpp_apps_teams (since there are no tokens)
	var count int
	err := sqlx.Get(db, &count, `SELECT COUNT(*) FROM vpp_apps_teams`)
	require.NoError(t, err)
	require.Zero(t, count)

}
