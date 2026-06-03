package tables

import (
	"crypto/md5" // nolint:gosec // used only to hash for efficient comparisons
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260615115235(t *testing.T) {
	db := applyUpToPrev(t)

	// This test recreates the original md5()-based generated columns to simulate an
	// upgrading instance, which requires the SQL MD5() function. MySQL 9.6/9.7
	// removed it, so skip there — the production migration itself uses no MD5().
	if _, err := db.Exec(`SELECT MD5('fleet')`); err != nil {
		t.Skipf("SQL MD5() not available on this MySQL version, skipping generated-column simulation: %v", err)
	}

	// After the retconned ADD migrations, these columns already exist as plain
	// BINARY(16), so on a fresh install this migration is a no-op. To exercise the
	// real generated-to-plain conversion an upgrading instance hits, recreate the
	// STORED generated columns exactly as the original (non-retconned) migrations
	// defined them, so MySQL computes the checksums/token for us.
	for _, stmt := range []string{
		`ALTER TABLE mdm_apple_declarations MODIFY COLUMN token BINARY(16) GENERATED ALWAYS AS (UNHEX(MD5(CONCAT(raw_json, IFNULL(secrets_updated_at, ''))))) STORED`,
		`ALTER TABLE mdm_windows_configuration_profiles MODIFY COLUMN checksum BINARY(16) GENERATED ALWAYS AS (UNHEX(MD5(syncml))) STORED`,
		`ALTER TABLE mdm_android_configuration_profiles MODIFY COLUMN checksum BINARY(16) GENERATED ALWAYS AS (UNHEX(MD5(CAST(raw_json AS CHAR CHARSET utf8mb4)))) STORED`,
	} {
		_, err := db.Exec(stmt)
		require.NoError(t, err)
	}

	// Insert rows while the columns are STORED generated columns; MySQL computes
	// the md5 checksum/token for us here.
	declRawJSON := `{"Type":"com.apple.configuration.test","Identifier":"id1"}`
	_, err := db.Exec(`INSERT INTO mdm_apple_declarations (declaration_uuid, team_id, identifier, name, raw_json) VALUES (?, ?, ?, ?, ?)`,
		"d1", 0, "id1", "decl1", declRawJSON)
	require.NoError(t, err)

	syncml := `<Replace></Replace>`
	_, err = db.Exec(`INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml) VALUES (?, ?, ?, ?)`,
		"w1", 0, "win1", syncml)
	require.NoError(t, err)

	androidRawJSON := `{"b":1,"a":2}`
	_, err = db.Exec(`INSERT INTO mdm_android_configuration_profiles (profile_uuid, team_id, name, raw_json) VALUES (?, ?, ?, ?)`,
		"a1", 0, "and1", androidRawJSON)
	require.NoError(t, err)

	// Capture the generated values before the migration.
	var declTokenBefore, winChecksumBefore, androidChecksumBefore []byte
	require.NoError(t, db.QueryRow(`SELECT token FROM mdm_apple_declarations WHERE declaration_uuid = ?`, "d1").Scan(&declTokenBefore))
	require.NoError(t, db.QueryRow(`SELECT checksum FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`, "w1").Scan(&winChecksumBefore))
	require.NoError(t, db.QueryRow(`SELECT checksum FROM mdm_android_configuration_profiles WHERE profile_uuid = ?`, "a1").Scan(&androidChecksumBefore))
	require.NotEmpty(t, declTokenBefore)
	require.NotEmpty(t, winChecksumBefore)
	require.NotEmpty(t, androidChecksumBefore)

	// Apply the migration that drops the GENERATED expressions.
	applyNext(t, db)

	// The stored bytes must be preserved exactly (no re-delivery / re-sync churn).
	var declTokenAfter, winChecksumAfter, androidChecksumAfter []byte
	require.NoError(t, db.QueryRow(`SELECT token FROM mdm_apple_declarations WHERE declaration_uuid = ?`, "d1").Scan(&declTokenAfter))
	require.NoError(t, db.QueryRow(`SELECT checksum FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`, "w1").Scan(&winChecksumAfter))
	require.NoError(t, db.QueryRow(`SELECT checksum FROM mdm_android_configuration_profiles WHERE profile_uuid = ?`, "a1").Scan(&androidChecksumAfter))

	require.Equal(t, declTokenBefore, declTokenAfter)
	require.Equal(t, winChecksumBefore, winChecksumAfter)
	require.Equal(t, androidChecksumBefore, androidChecksumAfter)

	// Sanity: the windows checksum equals md5(syncml).
	require.Equal(t, fmt.Sprintf("%x", md5.Sum([]byte(syncml))), fmt.Sprintf("%x", winChecksumAfter)) // nolint:gosec

	// The columns must now be plain (writable) — this would fail on a generated column.
	_, err = db.Exec(`UPDATE mdm_apple_declarations SET token = ? WHERE declaration_uuid = ?`, []byte("0123456789abcdef"), "d1")
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE mdm_windows_configuration_profiles SET checksum = ? WHERE profile_uuid = ?`, []byte("0123456789abcdef"), "w1")
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE mdm_android_configuration_profiles SET checksum = ? WHERE profile_uuid = ?`, []byte("0123456789abcdef"), "a1")
	require.NoError(t, err)
}
