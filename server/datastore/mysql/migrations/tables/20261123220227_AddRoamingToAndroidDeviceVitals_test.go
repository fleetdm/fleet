package tables

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261123220227(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `INSERT INTO host_mdm_android_device_vitals (host_uuid, manufacturer) VALUES ('existing-host', 'Google')`)

	applyNext(t, db)

	var row struct {
		Manufacturer     string       `db:"manufacturer"`
		Roaming          sql.NullBool `db:"roaming"`
		RoamingUpdatedAt sql.NullTime `db:"roaming_updated_at"`
	}
	require.NoError(t, db.Get(&row, `SELECT manufacturer, roaming, roaming_updated_at FROM host_mdm_android_device_vitals WHERE host_uuid = 'existing-host'`))
	require.Equal(t, "Google", row.Manufacturer)
	require.False(t, row.Roaming.Valid, "existing hosts have no roaming value until the agent reports one")
	require.False(t, row.RoamingUpdatedAt.Valid)

	execNoErr(t, db, `UPDATE host_mdm_android_device_vitals SET roaming = 1, roaming_updated_at = NOW(6) WHERE host_uuid = 'existing-host'`)
	require.NoError(t, db.Get(&row, `SELECT manufacturer, roaming, roaming_updated_at FROM host_mdm_android_device_vitals WHERE host_uuid = 'existing-host'`))
	require.True(t, row.Roaming.Valid)
	require.True(t, row.Roaming.Bool)
	require.True(t, row.RoamingUpdatedAt.Valid)
}
