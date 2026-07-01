package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20260701135344(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert a row before the migration to verify existing rows get the correct default.
	enrollmentToken, err := fleet.GenerateRandom32ByteEntropyURLSafeToken()
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO abm_tokens (organization_name, apple_id, renew_at, token, enrollment_url_token) VALUES ('test-org', 'test@apple.com', NOW(), 'token', ?)`, enrollmentToken)
	require.NoError(t, err)

	applyNext(t, db)

	// Verify existing row was backfilled with default value of 0.
	var tokenInvalid int
	err = db.QueryRow(`SELECT token_invalid FROM abm_tokens WHERE organization_name = 'test-org'`).Scan(&tokenInvalid)
	require.NoError(t, err)
	require.Equal(t, 0, tokenInvalid)

	// Verify column structure.
	var colName, colType, isNullable, colDefault string
	err = db.QueryRow(`
		SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_DEFAULT
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'abm_tokens'
		  AND COLUMN_NAME = 'token_invalid'
	`).Scan(&colName, &colType, &isNullable, &colDefault)
	require.NoError(t, err)
	require.Equal(t, "token_invalid", colName)
	require.Equal(t, "tinyint(1)", colType)
	require.Equal(t, "NO", isNullable)
	require.Equal(t, "0", colDefault)
}
