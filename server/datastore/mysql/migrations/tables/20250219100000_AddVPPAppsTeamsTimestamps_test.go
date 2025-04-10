package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20250219100000(t *testing.T) {
	db := applyUpToPrev(t)

	appCreatedAt := time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC)
	appUpdatedAt := time.Date(2025, 2, 2, 1, 0, 0, 0, time.UTC)

	adamID := "a"
	execNoErr(
		t, db, `INSERT INTO vpp_apps (adam_id, platform, created_at, updated_at) VALUES (?,?,?,?),(?,?,?,?),(?,?,?,?)`,
		adamID, "darwin", appCreatedAt, appUpdatedAt,
		adamID, "ios", appCreatedAt, appUpdatedAt,
		adamID, "ipados", appCreatedAt, appUpdatedAt,
	)
	vppTokenID := execNoErrLastID(t, db, `
	INSERT INTO vpp_tokens (
		organization_name,
		location,
		renew_at,
		token
	) VALUES
		(?, ?, ?, ?)
	`,
		"org1", "loc1", "2030-01-01 10:10:10", "blob1",
	)
	execNoErr(t, db, `INSERT INTO teams (name, id) VALUES ("Foo", 1)`)
	execNoErr(
		t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, team_id, global_or_team_id, vpp_token_id)
			VALUES (?,?,?,?,?),(?,?,?,?,?),(?,?,?,?,?),(?,?,?,?,?)`,
		adamID, "darwin", nil, 0, vppTokenID,
		adamID, "darwin", 1, 1, vppTokenID,
		adamID, "ios", 1, 1, vppTokenID,
		adamID, "ipados", 1, 1, vppTokenID,
	)

	// Apply current migration.
	applyNext(t, db)

	var row struct {
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}

	require.NoError(t, db.Get(&row, `SELECT created_at, updated_at FROM vpp_apps_teams WHERE adam_id = ?`, adamID))
	assert.Equal(t, appCreatedAt, row.CreatedAt)
	assert.Equal(t, appUpdatedAt, row.UpdatedAt)

	// test activity hydration manual query
	execNoErr(t, db, `INSERT INTO activities (activity_type, details, created_at) VALUES
		("added_app_store_app", '{"app_store_id":"a","team_id":0,"platform":"darwin"}', "2025-02-03 00:00:01"),
		("added_app_store_app", '{"app_store_id":"a","team_id":0,"platform":"darwin"}', "2025-02-03 00:00:02"),
		("edited_app_store_app", '{"app_store_id":"a","team_id":0,"platform":"darwin"}', "2025-02-03 00:00:03"),
		("edited_app_store_app", '{"app_store_id":"a","team_id":0,"platform":"darwin"}', "2025-02-03 00:00:04"),
		("added_app_store_app", '{"app_store_id":"a","team_id":1,"platform":"darwin"}', "2025-02-03 00:00:05"),
		("edited_app_store_app", '{"app_store_id":"a","team_id":1,"platform":"darwin"}', "2025-02-03 00:00:06"),
		("added_app_store_app", '{"app_store_id":"a","team_id":1,"platform":"ipados"}', "2025-02-03 00:00:07")
	`)

	execNoErr(t, db, `UPDATE vpp_apps_teams vat
LEFT JOIN (SELECT MAX(created_at) added_at, details->>"$.app_store_id" adam_id, details->>"$.platform" platform, details->>"$.team_id" team_id
    	FROM activities WHERE activity_type = 'added_app_store_app' GROUP BY adam_id, platform, team_id) aa ON 
	vat.global_or_team_id = aa.team_id AND vat.adam_id = aa.adam_id AND vat.platform = aa.platform
LEFT JOIN (SELECT MAX(created_at) edited_at, details->>"$.app_store_id" adam_id, details->>"$.platform" platform, details->>"$.team_id" team_id
		FROM activities WHERE activity_type = 'edited_app_store_app' GROUP BY adam_id, platform, team_id) ae ON
	vat.global_or_team_id = ae.team_id AND vat.adam_id = ae.adam_id AND vat.platform = ae.platform
SET vat.created_at = COALESCE(added_at, vat.created_at), vat.updated_at = COALESCE(edited_at, added_at, vat.updated_at)`)

	var rows []struct {
		Platform  string    `db:"platform"`
		TeamID    *uint     `db:"team_id"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}

	require.NoError(t, db.Select(&rows, `SELECT platform, team_id, created_at, updated_at FROM vpp_apps_teams ORDER BY created_at, updated_at`))

	// no activities on iOS, so they keep the carry-over from the original query
	assert.Equal(t, "ios", rows[0].Platform)
	assert.Equal(t, appCreatedAt, rows[0].CreatedAt)
	assert.Equal(t, appUpdatedAt, rows[0].UpdatedAt)
	// no-team app with multiple events each
	assert.Nil(t, rows[1].TeamID)
	assert.Equal(t, time.Date(2025, 2, 3, 0, 0, 2, 0, time.UTC), rows[1].CreatedAt)
	assert.Equal(t, time.Date(2025, 2, 3, 0, 0, 4, 0, time.UTC), rows[1].UpdatedAt)
	// team 1 app with one event per type
	assert.Equal(t, uint(1), *rows[2].TeamID)
	assert.Equal(t, time.Date(2025, 2, 3, 0, 0, 5, 0, time.UTC), rows[2].CreatedAt)
	assert.Equal(t, time.Date(2025, 2, 3, 0, 0, 6, 0, time.UTC), rows[2].UpdatedAt)
	// team 1 app with only added event
	assert.Equal(t, "ipados", rows[3].Platform)
	assert.Equal(t, time.Date(2025, 2, 3, 0, 0, 7, 0, time.UTC), rows[3].CreatedAt)
	assert.Equal(t, time.Date(2025, 2, 3, 0, 0, 7, 0, time.UTC), rows[3].UpdatedAt)
}
