package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260805011013(t *testing.T) {
	db := applyUpToPrev(t)

	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('ManagedKeyTest')`)

	// Pre-existing policies with Fleet-maintained display names stay user-owned
	// until GitOps/API sets fleet_managed_key explicitly.
	globalNamed := execNoErrLastID(t, db, `
		INSERT INTO policies (name, query, description, platforms, checksum, updated_at)
		VALUES ('Operating system up to date (macOS)', 'SELECT 1', '', 'darwin', UNHEX(MD5(CONCAT_WS(CHAR(0), '', 'Operating system up to date (macOS)'))), '2020-01-01 00:00:00')`)
	dogfoodAlias := execNoErrLastID(t, db, `
		INSERT INTO policies (name, query, description, platforms, team_id, checksum)
		VALUES ('macOS - Operating system up to date', 'SELECT 1', '', 'darwin', ?, UNHEX(MD5(CONCAT_WS(CHAR(0), ?, 'macOS - Operating system up to date'))))`, teamID, teamID)

	applyNext(t, db)

	var updatedAtUnchanged bool
	require.NoError(t, db.QueryRow(`SELECT updated_at = '2020-01-01 00:00:00' FROM policies WHERE id = ?`, globalNamed).Scan(&updatedAtUnchanged))
	require.True(t, updatedAtUnchanged, "migration must not modify existing policy timestamps")

	expectedIndexes := []struct {
		name      string
		column    string
		nonUnique int
	}{
		{"idx_policies_fleet_managed_key", "fleet_managed_key", 1},
		{"idx_policies_fleet_managed_team_key", "fleet_managed_team_key", 0},
	}
	for _, expected := range expectedIndexes {
		var column string
		var nonUnique int
		require.NoError(t, db.QueryRow(`
			SELECT column_name, non_unique
			FROM information_schema.statistics
			WHERE table_schema = DATABASE() AND table_name = 'policies' AND index_name = ?
			ORDER BY seq_in_index
			LIMIT 1`, expected.name).Scan(&column, &nonUnique))
		require.Equal(t, expected.column, column)
		require.Equal(t, expected.nonUnique, nonUnique)
	}

	var key *string
	err := db.QueryRow(`SELECT fleet_managed_key FROM policies WHERE id = ?`, globalNamed).Scan(&key)
	require.NoError(t, err)
	require.Nil(t, key, "migration must not claim policies by name")

	err = db.QueryRow(`SELECT fleet_managed_key FROM policies WHERE id = ?`, dogfoodAlias).Scan(&key)
	require.NoError(t, err)
	require.Nil(t, key, "migration must not claim policies by name")

	// Explicit key works; unique fleet_managed_team_key rejects a second global claim.
	execNoErr(t, db, `
		UPDATE policies SET fleet_managed_key = 'macos_os_up_to_date' WHERE id = ?`, globalNamed)

	_, err = db.Exec(`
		INSERT INTO policies (name, query, description, platforms, fleet_managed_key, checksum)
		VALUES ('other', 'SELECT 1', '', 'darwin', 'macos_os_up_to_date', UNHEX(MD5(CONCAT_WS(CHAR(0), '', 'other'))))`)
	require.Error(t, err)

	// Same key on a different team is allowed.
	_, err = db.Exec(`
		INSERT INTO policies (name, query, description, platforms, team_id, fleet_managed_key, checksum)
		VALUES ('team up to date', 'SELECT 1', '', 'darwin', ?, 'macos_os_up_to_date', UNHEX(MD5(CONCAT_WS(CHAR(0), ?, 'team up to date'))))`, teamID, teamID)
	require.NoError(t, err)
}
