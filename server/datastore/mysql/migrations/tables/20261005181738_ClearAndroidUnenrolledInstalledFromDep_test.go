package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261005181738(t *testing.T) {
	db := applyUpToPrev(t)

	insertHost := func(uuid, platform string) uint {
		return uint(execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?)`, //nolint:gosec // dismiss G115
			"oh-"+uuid, "nk-"+uuid, uuid+".local", uuid, platform))
	}
	insertHostMDM := func(hostID uint, enrolled, fromDEP, personal bool) {
		execNoErr(t, db, `INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_personal_enrollment) VALUES (?, ?, '', ?, ?)`,
			hostID, enrolled, fromDEP, personal)
	}

	androidCOBOUnenrolled := insertHost("android-cobo-unenrolled", "android")
	insertHostMDM(androidCOBOUnenrolled, false, true, false)

	androidCOBOEnrolled := insertHost("android-cobo-enrolled", "android")
	insertHostMDM(androidCOBOEnrolled, true, true, false)

	androidPersonalUnenrolled := insertHost("android-personal-unenrolled", "android")
	insertHostMDM(androidPersonalUnenrolled, false, false, true)

	macPending := insertHost("mac-pending", "darwin")
	insertHostMDM(macPending, false, true, false)

	iosPending := insertHost("ios-pending", "ios")
	insertHostMDM(iosPending, false, true, false)

	applyNext(t, db)

	cases := []struct {
		name        string
		hostID      uint
		wantFromDEP bool
		wantStatus  string
	}{
		{"unenrolled android company-owned", androidCOBOUnenrolled, false, "Off"},
		{"enrolled android company-owned", androidCOBOEnrolled, true, "On (automatic)"},
		{"unenrolled android personal", androidPersonalUnenrolled, false, "Off"},
		{"pending macOS", macPending, true, "Pending"},
		{"pending iOS", iosPending, true, "Pending"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var row struct {
				FromDEP bool   `db:"installed_from_dep"`
				Status  string `db:"enrollment_status"`
			}
			require.NoError(t, db.Get(&row, `SELECT installed_from_dep, enrollment_status FROM host_mdm WHERE host_id = ?`, c.hostID))
			require.Equal(t, c.wantFromDEP, row.FromDEP)
			require.Equal(t, c.wantStatus, row.Status)
		})
	}
}
