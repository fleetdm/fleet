package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20260908192830(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	expiresAt := time.Now().Add(100 * 365 * 24 * time.Hour)

	// Insert a token with no team (unassigned, global_or_team_id = 0)
	_, err := db.Exec(`
		INSERT INTO android_zero_touch_tokens (team_id, global_or_team_id, token_name, token_value, embedded_enroll_secret, expires_at)
		VALUES (NULL, 0, 'enterprises/LC00test/enrollmentTokens/abc123', 'tokenvalue123', 'secret1', ?)`,
		expiresAt,
	)
	require.NoError(t, err)

	// Unique key on global_or_team_id prevents a second unassigned token
	_, err = db.Exec(`
		INSERT INTO android_zero_touch_tokens (team_id, global_or_team_id, token_name, token_value, embedded_enroll_secret, expires_at)
		VALUES (NULL, 0, 'enterprises/LC00test/enrollmentTokens/dup', 'dup', 'secret', ?)`,
		expiresAt,
	)
	require.Error(t, err, "unique key should prevent duplicate unassigned token")

	// Insert a token for a specific team
	_, err = db.Exec(`
		INSERT INTO android_zero_touch_tokens (team_id, global_or_team_id, token_name, token_value, embedded_enroll_secret, expires_at)
		VALUES (1, 1, 'enterprises/LC00test/enrollmentTokens/ghi789', 'tokenvalue789', 'secret3', ?)`,
		expiresAt,
	)
	require.NoError(t, err)

	// Unique key prevents a second token for the same team
	_, err = db.Exec(`
		INSERT INTO android_zero_touch_tokens (team_id, global_or_team_id, token_name, token_value, embedded_enroll_secret, expires_at)
		VALUES (1, 1, 'enterprises/LC00test/enrollmentTokens/dup', 'dup', 'secret', ?)`,
		expiresAt,
	)
	require.Error(t, err, "unique key should prevent duplicate team_id")

	// Read back and verify values
	var (
		tokenName  string
		tokenValue string
		secret     string
	)
	err = db.QueryRow(`
		SELECT token_name, token_value, embedded_enroll_secret
		FROM android_zero_touch_tokens WHERE global_or_team_id = 0`).Scan(&tokenName, &tokenValue, &secret)
	require.NoError(t, err)
	require.Equal(t, "enterprises/LC00test/enrollmentTokens/abc123", tokenName)
	require.Equal(t, "tokenvalue123", tokenValue)
	require.Equal(t, "secret1", secret)
}
