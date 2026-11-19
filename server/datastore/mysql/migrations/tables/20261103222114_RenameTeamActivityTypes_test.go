package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20261103222114(t *testing.T) {
	db := applyUpToPrev(t)

	insert := func(activityType string) int64 {
		return execNoErrLastID(t, db,
			`INSERT INTO activity_past (activity_type, details, streamed, user_email) VALUES (?, '{"team_id": 7}', true, 'a@example.com')`,
			activityType)
	}
	// Written out, not read from the migration, so a typo in the migration fails the test.
	renamed := map[string]string{
		"created_team":           "created_fleet",
		"deleted_team":           "deleted_fleet",
		"applied_spec_team":      "applied_spec_fleet",
		"changed_user_team_role": "changed_user_fleet_role",
		"deleted_user_team_role": "deleted_user_fleet_role",
	}
	want := make(map[int64]string) // row id to expected type
	for oldType, newType := range renamed {
		want[insert(oldType)] = newType
	}
	// Types that must not change, including one that already has the new name.
	for _, activityType := range []string{"created_fleet", "edited_policy", "changed_user_global_role"} {
		want[insert(activityType)] = activityType
	}

	applyNext(t, db)

	var rows []struct {
		ID           int64  `db:"id"`
		ActivityType string `db:"activity_type"`
		Details      string `db:"details"`
		Streamed     bool   `db:"streamed"`
	}
	require.NoError(t, db.Select(&rows, `SELECT id, activity_type, details, streamed FROM activity_past`))
	require.Len(t, rows, len(want))
	for _, row := range rows {
		assert.Equal(t, want[row.ID], row.ActivityType)
		assert.JSONEq(t, `{"team_id": 7}`, row.Details, "details are not touched")
		assert.True(t, row.Streamed, "streamed is not touched")
	}
}
