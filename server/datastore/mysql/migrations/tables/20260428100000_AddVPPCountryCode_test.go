package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260428100000(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	// Verify country_code column was added to vpp_tokens.
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = DATABASE()
		  AND table_name = 'vpp_tokens'
		  AND column_name = 'country_code'`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count, "expected country_code column on vpp_tokens")

	// Verify country_code column was added to vpp_apps.
	err = db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = DATABASE()
		  AND table_name = 'vpp_apps'
		  AND column_name = 'country_code'`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count, "expected country_code column on vpp_apps")
}
