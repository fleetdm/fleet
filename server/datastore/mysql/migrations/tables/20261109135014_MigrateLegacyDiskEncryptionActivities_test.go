package tables

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20261109135014(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `INSERT INTO users (id, name, email, password, salt) VALUES (1, 'Admin', 'admin@example.com', 'x', 'x')`)

	type activityRow struct {
		ID             int64          `db:"id"`
		CreatedAt      time.Time      `db:"created_at"`
		UserID         sql.NullInt64  `db:"user_id"`
		UserName       sql.NullString `db:"user_name"`
		UserEmail      string         `db:"user_email"`
		ActivityType   string         `db:"activity_type"`
		Details        sql.NullString `db:"details"`
		Streamed       bool           `db:"streamed"`
		FleetInitiated bool           `db:"fleet_initiated"`
		HostOnly       bool           `db:"host_only"`
	}
	const selectRows = `SELECT id, created_at, user_id, user_name, user_email, activity_type, details, streamed, fleet_initiated, host_only
		FROM activity_past ORDER BY id`

	// Vary the non-migrated columns so the test catches a migration that touches them.
	insert := func(activityType string, details *string, streamed bool) int64 {
		return execNoErrLastID(t, db, `
			INSERT INTO activity_past (created_at, user_id, user_name, user_email, activity_type, details, streamed)
			VALUES ('2025-01-02 03:04:05.123456', 1, 'Admin', 'admin@example.com', ?, ?, ?)`,
			activityType, details, streamed)
	}

	const (
		fleetDetails      = `{"fleet_id": 123, "fleet_name": "Workstations", "platform": null}`
		unassignedDetails = `{"fleet_id": null, "fleet_name": null, "platform": null}`
	)
	want := make(map[int64]string) // row id to expected details
	for _, activityType := range []string{"enabled_macos_disk_encryption", "disabled_macos_disk_encryption"} {
		for _, tc := range []struct {
			before string
			after  string
		}{
			// Fleet 4.29 to 4.81.
			{`{"team_id": 123, "team_name": "Workstations"}`, fleetDetails},
			{`{"team_id": null, "team_name": null}`, unassignedDetails},
			// Fleet 4.82 to 4.91.
			{`{"team_id": 123, "team_name": "Workstations", "fleet_id": 123, "fleet_name": "Workstations"}`, fleetDetails},
			{`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`, unassignedDetails},
			// Seeded by dibble.
			{`{"fleet_id": 123, "fleet_name": "Workstations"}`, fleetDetails},
			{`{"fleet_id": null, "fleet_name": null}`, unassignedDetails},
			{`{}`, unassignedDetails},
			// Never written, but must not fail the migration.
			{`null`, unassignedDetails},
			{`[]`, unassignedDetails},
		} {
			want[insert(activityType, &tc.before, true)] = tc.after
		}
		want[insert(activityType, nil, false)] = unassignedDetails
	}

	untouchedDetails := map[string]string{
		"edited_disk_encryption_settings": `{"fleet_id": 5, "fleet_name": "Desktops", "platform": "macos"}`,
		"edited_agent_options":            `{"team_id": 5, "team_name": "Desktops", "global": false}`,
	}
	for activityType, details := range untouchedDetails {
		insert(activityType, &details, true)
	}

	var before []activityRow
	require.NoError(t, db.Select(&before, selectRows))
	beforeByID := make(map[int64]activityRow, len(before))
	for _, row := range before {
		beforeByID[row.ID] = row
	}

	applyNext(t, db)

	var after []activityRow
	require.NoError(t, db.Select(&after, selectRows))
	require.Len(t, after, len(before))

	for _, row := range after {
		orig, ok := beforeByID[row.ID]
		require.True(t, ok, "row %d did not exist before the migration", row.ID)

		if wantDetails, migrated := want[row.ID]; migrated {
			assert.Equal(t, "edited_disk_encryption_settings", row.ActivityType)
			require.True(t, row.Details.Valid)
			assert.JSONEq(t, wantDetails, row.Details.String, "row %d was %s", row.ID, orig.Details.String)
		} else {
			assert.Equal(t, orig.ActivityType, row.ActivityType)
			assert.JSONEq(t, untouchedDetails[orig.ActivityType], row.Details.String)
		}

		// Clear the migrated columns, then everything else must match.
		orig.ActivityType, row.ActivityType = "", ""
		orig.Details, row.Details = sql.NullString{}, sql.NullString{}
		assert.Equal(t, orig, row)
	}

	var remaining int
	require.NoError(t, db.Get(&remaining,
		`SELECT COUNT(*) FROM activity_past WHERE activity_type IN ('enabled_macos_disk_encryption', 'disabled_macos_disk_encryption')`))
	assert.Zero(t, remaining)
}
