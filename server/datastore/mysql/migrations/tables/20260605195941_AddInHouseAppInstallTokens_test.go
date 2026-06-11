package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20260605195941(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	// Verify the new table accepts inserts with the expected shape.
	expires := time.Now().UTC().Add(6 * time.Hour)
	execNoErr(t, db,
		`INSERT INTO in_house_app_install_tokens
			(token, software_title_id, team_id, host_id, expires_at)
		 VALUES (?, ?, ?, ?, ?)`,
		"00000000-0000-0000-0000-000000000001", 1, 0, 1, expires,
	)

	// Verify the primary key constraint rejects duplicates.
	_, err := db.Exec(
		`INSERT INTO in_house_app_install_tokens
			(token, software_title_id, team_id, host_id, expires_at)
		 VALUES (?, ?, ?, ?, ?)`,
		"00000000-0000-0000-0000-000000000001", 2, 1, 2, expires,
	)
	require.Error(t, err)

	// Verify we can read the row back.
	var got struct {
		Token           string    `db:"token"`
		SoftwareTitleID uint      `db:"software_title_id"`
		TeamID          uint      `db:"team_id"`
		HostID          uint      `db:"host_id"`
		ExpiresAt       time.Time `db:"expires_at"`
	}
	require.NoError(t, db.Get(&got,
		`SELECT token, software_title_id, team_id, host_id, expires_at
		 FROM in_house_app_install_tokens WHERE token = ?`,
		"00000000-0000-0000-0000-000000000001",
	))
	require.EqualValues(t, 1, got.SoftwareTitleID)
	require.EqualValues(t, 0, got.TeamID)
	require.EqualValues(t, 1, got.HostID)
	require.WithinDuration(t, expires, got.ExpiresAt, time.Second)
}
