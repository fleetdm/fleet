package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20261016044236(t *testing.T) {
	db := applyUpToPrev(t)

	deliveredAt := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	insertHost := func(uuid string, personalType *string, rights *int) {
		hostID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, 'ipados')`,
			"oh-"+uuid, "nk-"+uuid, uuid+".local", uuid)
		execNoErr(t, db, `INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_personal_enrollment, personal_enrollment_type) VALUES (?, 1, 'https://fleet.local', 0, ?, ?)`,
			hostID, personalType != nil, personalType)
		if rights != nil {
			execNoErr(t, db, `INSERT INTO host_mdm_apple_enrollment_permissions (host_uuid, access_rights, delivered_at) VALUES (?, ?, ?)`,
				uuid, *rights, deliveredAt)
		}
	}

	accountDriven, manualProfile := "account_driven", "manual_profile"
	all, narrowed := 8191, 8179
	insertHost("adue-full", &accountDriven, &all)
	insertHost("adue-narrowed", &accountDriven, &narrowed)
	insertHost("adue-no-row", &accountDriven, nil)
	// Only account-driven rows are narrowed, even when another personal row is
	// stored with full rights.
	insertHost("manual-byod", &manualProfile, &all)
	insertHost("company", nil, &all)

	applyNext(t, db)

	for uuid, want := range map[string]int{
		"adue-full":     narrowed,
		"adue-narrowed": narrowed,
		"manual-byod":   all,
		"company":       all,
	} {
		var row struct {
			AccessRights int       `db:"access_rights"`
			DeliveredAt  time.Time `db:"delivered_at"`
		}
		require.NoError(t, db.Get(&row, `SELECT access_rights, delivered_at FROM host_mdm_apple_enrollment_permissions WHERE host_uuid = ?`, uuid))
		require.Equal(t, want, row.AccessRights, uuid)
		require.True(t, deliveredAt.Equal(row.DeliveredAt), uuid)
	}

	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM host_mdm_apple_enrollment_permissions WHERE host_uuid = 'adue-no-row'`))
	require.Zero(t, count)
}
