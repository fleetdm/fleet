package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260429154601(t *testing.T) {
	db := applyUpToPrev(t)

	// Pre-migration row to confirm the new column lands as NULL on
	// existing data rather than rejecting or auto-filling.
	execNoErr(t, db, `INSERT INTO vpp_tokens
			(organization_name, location, renew_at, token)
		 VALUES (?, ?, NOW(), ?)`,
		"Pre-Migration Org", "https://example.com/mdm/apple/mdm", []byte("pre-migration-token"))

	applyNext(t, db)

	assertCountryCode := func(table string) {
		t.Helper()
		var (
			dataType   string
			maxLen     int
			isNullable string
			defaultVal *string
			collation  *string
		)
		err := db.QueryRow(`
			SELECT DATA_TYPE, CHARACTER_MAXIMUM_LENGTH, IS_NULLABLE, COLUMN_DEFAULT, COLLATION_NAME
			FROM information_schema.columns
			WHERE table_schema = DATABASE()
			  AND table_name = ?
			  AND column_name = 'country_code'`, table).
			Scan(&dataType, &maxLen, &isNullable, &defaultVal, &collation)
		require.NoError(t, err, "country_code column missing on %s", table)
		require.Equal(t, "varchar", dataType, "expected varchar on %s.country_code", table)
		require.Equal(t, 4, maxLen, "expected length 4 on %s.country_code", table)
		require.Equal(t, "YES", isNullable, "expected nullable %s.country_code", table)
		require.Nil(t, defaultVal, "expected NULL default on %s.country_code", table)
		require.NotNil(t, collation, "expected explicit collation on %s.country_code", table)
		require.Equal(t, "utf8mb4_unicode_ci", *collation, "wrong collation on %s.country_code", table)
	}

	assertCountryCode("vpp_tokens")
	assertCountryCode("vpp_apps")

	var preCountry *string
	err := db.QueryRow(`SELECT country_code FROM vpp_tokens WHERE organization_name = ?`,
		"Pre-Migration Org").Scan(&preCountry)
	require.NoError(t, err)
	require.Nil(t, preCountry, "expected NULL country_code on pre-migration row")
}
