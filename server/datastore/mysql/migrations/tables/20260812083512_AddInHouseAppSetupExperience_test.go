package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260812083512(t *testing.T) {
	db := applyUpToPrev(t)

	iosAppID := execNoErrLastID(t, db,
		`INSERT INTO in_house_apps (global_or_team_id, storage_id, platform, filename) VALUES (?, ?, ?, ?)`,
		0, "storage-acme", "ios", "acme.ipa")
	ipadosAppID := execNoErrLastID(t, db,
		`INSERT INTO in_house_apps (global_or_team_id, storage_id, platform, filename) VALUES (?, ?, ?, ?)`,
		0, "storage-acme", "ipados", "acme.ipa")

	applyNext(t, db)

	var installDuringSetup []bool
	err := db.Select(&installDuringSetup,
		`SELECT install_during_setup FROM in_house_apps WHERE id IN (?, ?) ORDER BY id`, iosAppID, ipadosAppID)
	require.NoError(t, err)
	require.Equal(t, []bool{false, false}, installDuringSetup)

	execNoErr(t, db,
		`INSERT INTO setup_experience_status_results (host_uuid, name, status, in_house_app_id) VALUES (?, ?, ?, ?)`,
		"host-uuid-1", "Acme", "pending", iosAppID)
	execNoErr(t, db,
		`INSERT INTO setup_experience_status_results (host_uuid, name, status, in_house_app_id) VALUES (?, ?, ?, ?)`,
		"host-uuid-1", "Acme", "pending", ipadosAppID)

	execNoErr(t, db, `DELETE FROM in_house_apps WHERE id = ?`, iosAppID)

	var remaining []int64
	err = db.Select(&remaining,
		`SELECT in_house_app_id FROM setup_experience_status_results WHERE host_uuid = ?`, "host-uuid-1")
	require.NoError(t, err)
	require.Equal(t, []int64{ipadosAppID}, remaining)
}
