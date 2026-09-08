package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func insertABMToken(t *testing.T, db *sqlx.DB, orgName string) int64 {
	return execNoErrLastID(t, db,
		`INSERT INTO abm_tokens (organization_name, apple_id, renew_at, token, enrollment_url_token) VALUES (?, ?, NOW(), ?, ?)`,
		orgName, orgName+"-apple-id", orgName+"-token", orgName+"-enrollment-url-token-fake-token-long",
	)
}

func TestUp_20260908113251(t *testing.T) {
	t.Run("no tokens does not error", func(t *testing.T) {
		db := applyUpToPrev(t)
		applyNext(t, db)
	})
	t.Run("single token becomes default", func(t *testing.T) {
		db := applyUpToPrev(t)
		insertABMToken(t, db, "token1")

		applyNext(t, db)

		var isDefault bool
		err := db.Get(&isDefault, `SELECT is_default FROM abm_tokens WHERE organization_name = ?`, "token1")
		require.NoError(t, err)
		require.True(t, isDefault)
	})

	t.Run("multiple tokens leaves none default", func(t *testing.T) {
		db := applyUpToPrev(t)
		insertABMToken(t, db, "token1")
		insertABMToken(t, db, "token2")

		applyNext(t, db)

		var count int
		err := db.Get(&count, `SELECT COUNT(*) FROM abm_tokens WHERE is_default = 1`)
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})
}
