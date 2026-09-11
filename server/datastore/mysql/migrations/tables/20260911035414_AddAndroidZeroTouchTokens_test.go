package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20260911035414(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	expiresAt := time.Now().Add(100 * 365 * 24 * time.Hour)

	// Insert a token with no team (unassigned, global_or_team_id = 0)
	_, err := db.Exec(`
		INSERT INTO android_zero_touch_tokens (team_id, global_or_team_id, token_name, token_value, expires_at)
		VALUES (NULL, 0, 'enterprises/LC00test/enrollmentTokens/abc123', 'tokenvalue123', ?)`,
		expiresAt,
	)
	require.NoError(t, err)

	// Unique key on global_or_team_id prevents a second unassigned token
	_, err = db.Exec(`
		INSERT INTO android_zero_touch_tokens (team_id, global_or_team_id, token_name, token_value, expires_at)
		VALUES (NULL, 0, 'enterprises/LC00test/enrollmentTokens/dup', 'dup', ?)`,
		expiresAt,
	)
	require.Error(t, err, "unique key should prevent duplicate unassigned token")

	// Insert a token for a specific team
	_, err = db.Exec(`
		INSERT INTO android_zero_touch_tokens (team_id, global_or_team_id, token_name, token_value, expires_at)
		VALUES (1, 1, 'enterprises/LC00test/enrollmentTokens/ghi789', 'tokenvalue789', ?)`,
		expiresAt,
	)
	require.NoError(t, err)

	// Unique key prevents a second token for the same team
	_, err = db.Exec(`
		INSERT INTO android_zero_touch_tokens (team_id, global_or_team_id, token_name, token_value, expires_at)
		VALUES (1, 1, 'enterprises/LC00test/enrollmentTokens/dup', 'dup', ?)`,
		expiresAt,
	)
	require.Error(t, err, "unique key should prevent duplicate team_id")

	// Read back and verify values
	var (
		tokenName  string
		tokenValue string
	)
	err = db.QueryRow(`
		SELECT token_name, token_value
		FROM android_zero_touch_tokens WHERE global_or_team_id = 0`).Scan(&tokenName, &tokenValue)
	require.NoError(t, err)
	require.Equal(t, "enterprises/LC00test/enrollmentTokens/abc123", tokenName)
	require.Equal(t, "tokenvalue123", tokenValue)
}
