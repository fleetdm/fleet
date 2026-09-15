package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260901200000(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('ots-team')`)

	insert := func(secret string, team any) error {
		_, err := db.Exec(`INSERT INTO host_one_time_enroll_secrets
			(secret, host_id, team_id, platform, hardware_uuid, hardware_serial)
			VALUES (?, 1, ?, 'darwin', 'UUID-1', 'SERIAL-1')`, secret, team)
		return err
	}

	require.NoError(t, insert("secret-a", teamID))

	// secret lookups are case-sensitive: an enroll request presenting the
	// wrong case must not match
	var matches int
	require.NoError(t, db.Get(&matches, `SELECT COUNT(*) FROM host_one_time_enroll_secrets WHERE secret = ?`, "SECRET-A"))
	require.Equal(t, 0, matches)
	require.NoError(t, db.Get(&matches, `SELECT COUNT(*) FROM host_one_time_enroll_secrets WHERE secret = ?`, "secret-a"))
	require.Equal(t, 1, matches)

	var collation string
	require.NoError(t, db.Get(&collation, `SELECT COLLATION_NAME FROM information_schema.columns
		WHERE TABLE_SCHEMA = (SELECT DATABASE()) AND TABLE_NAME = 'host_one_time_enroll_secrets' AND COLUMN_NAME = 'secret'`))
	require.Equal(t, "utf8mb4_bin", collation)

	// secret is unique, and uniqueness is case-sensitive too
	require.Error(t, insert("secret-a", teamID))
	require.NoError(t, insert("SECRET-A", teamID))

	// use markers default to NULL
	var nulls int
	require.NoError(t, db.Get(&nulls, `SELECT COUNT(*) FROM host_one_time_enroll_secrets
		WHERE consumed_at IS NULL AND orbit_used_at IS NULL AND osquery_used_at IS NULL`))
	require.Equal(t, 2, nulls)

	// deleting the team keeps the rows but clears team_id
	execNoErr(t, db, `DELETE FROM teams WHERE id = ?`, teamID)
	var withTeam, total int
	require.NoError(t, db.Get(&total, `SELECT COUNT(*) FROM host_one_time_enroll_secrets`))
	require.NoError(t, db.Get(&withTeam, `SELECT COUNT(*) FROM host_one_time_enroll_secrets WHERE team_id IS NOT NULL`))
	require.Equal(t, 2, total)
	require.Equal(t, 0, withTeam)
}
