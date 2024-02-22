package tables

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20231206142340(t *testing.T) {
	db := applyUpToPrev(t)

	insertStmt := `
INSERT INTO host_mdm (
	host_id,
	enrolled,
	server_url
) VALUES (?, ?, ?)`

	execNoErr(t, db, insertStmt, 1, 1, "https://example.com")

	applyNext(t, db)

	// verify that the new column is present
	var hmdm struct {
		ID               uint   `db:"id"`
		HostID           uint   `db:"host_id"`
		Enrolled         bool   `db:"enrolled"`
		ServerURL        string `db:"server_url"`
		InstalledFromDEP bool   `db:"installed_from_dep"`
		MDMID            *uint  `db:"mdm_id"`
		IsServer         *bool  `db:"is_server"`
		FleetEnrollRef   string `db:"fleet_enroll_ref"`
	}
	err := db.Get(&hmdm, "SELECT * FROM host_mdm WHERE host_id = ?", 1)
	require.NoError(t, err)
	require.Equal(t, uint(1), hmdm.HostID)
	require.Equal(t, true, hmdm.Enrolled)
	require.Equal(t, "https://example.com", hmdm.ServerURL)
	require.Equal(t, false, hmdm.InstalledFromDEP)
	require.Nil(t, hmdm.MDMID)
	require.Nil(t, hmdm.IsServer)
	require.Equal(t, "", hmdm.FleetEnrollRef)

	insertStmt = `
INSERT INTO host_mdm (
	host_id,
	enrolled,
	server_url,
	fleet_enroll_ref
) VALUES (?, ?, ?, ?)`

	ref := uuid.NewString()
	execNoErr(t, db, insertStmt, 2, 1, "https://example.com", ref)

	err = db.Get(&hmdm, "SELECT * FROM host_mdm WHERE host_id = ?", 2)
	require.NoError(t, err)
	require.Equal(t, uint(2), hmdm.HostID)
	require.Equal(t, true, hmdm.Enrolled)
	require.Equal(t, "https://example.com", hmdm.ServerURL)
	require.Equal(t, false, hmdm.InstalledFromDEP)
	require.Nil(t, hmdm.MDMID)
	require.Nil(t, hmdm.IsServer)
	require.Equal(t, ref, hmdm.FleetEnrollRef)
}
