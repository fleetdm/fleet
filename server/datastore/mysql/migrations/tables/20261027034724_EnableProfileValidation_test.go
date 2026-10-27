package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261027034724(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(
		JSON_REMOVE(json_value, '$.mdm.enable_profile_validation'),
		'$.mdm.apple_require_hardware_attestation', true
	) WHERE id = 1`)

	applyNext(t, db)

	assertStored := func(t *testing.T) {
		t.Helper()
		var validation, sibling string
		require.NoError(t, db.QueryRow(`SELECT
			JSON_TYPE(JSON_EXTRACT(json_value, '$.mdm.enable_profile_validation')),
			JSON_EXTRACT(json_value, '$.mdm.apple_require_hardware_attestation')
			FROM app_config_json WHERE id = 1`).Scan(&validation, &sibling))
		require.Equal(t, "BOOLEAN", validation)
		require.Equal(t, "true", sibling)

		var isTrue bool
		require.NoError(t, db.QueryRow(`SELECT JSON_EXTRACT(json_value, '$.mdm.enable_profile_validation') = CAST('true' AS JSON)
			FROM app_config_json WHERE id = 1`).Scan(&isTrue))
		require.True(t, isTrue)
	}
	assertStored(t)

	// re-running is safe
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, Up_20261026092108(tx))
	require.NoError(t, tx.Commit())
	assertStored(t)
}

func TestUp_20261027034724_NoMDMObject(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_REMOVE(json_value, '$.mdm') WHERE id = 1`)

	applyNext(t, db)

	var isTrue bool
	require.NoError(t, db.QueryRow(`SELECT JSON_EXTRACT(json_value, '$.mdm.enable_profile_validation') = CAST('true' AS JSON)
		FROM app_config_json WHERE id = 1`).Scan(&isTrue))
	require.True(t, isTrue)
}
