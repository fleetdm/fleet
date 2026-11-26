package tables

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261125230914(t *testing.T) {
	db := applyUpToPrev(t)

	fleetMDMID := execNoErrLastID(t, db,
		`INSERT INTO mobile_device_management_solutions (name, server_url) VALUES ('Fleet', 'https://fleet.example.com')`)
	intuneMDMID := execNoErrLastID(t, db,
		`INSERT INTO mobile_device_management_solutions (name, server_url) VALUES ('Intune', 'https://intune.example.com')`)

	var hostCount int
	newHost := func(uuid, platform string, osqueryIDIsUUID bool) uint {
		hostCount++
		osqueryID := fmt.Sprintf("osquery-%d", hostCount)
		if osqueryIDIsUUID {
			osqueryID = uuid
		}
		id := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?)`,
			osqueryID, fmt.Sprintf("node-key-%d", hostCount), fmt.Sprintf("host-%d", hostCount), uuid, platform)
		return uint(id) //nolint:gosec // dismiss G115
	}
	setMDM := func(hostID uint, enrolled bool, mdmID int64) {
		execNoErr(t, db, `INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, mdm_id, is_server) VALUES (?, ?, '', 0, ?, 0)`,
			hostID, enrolled, mdmID)
	}
	var enrollmentCount int
	enroll := func(hostUUID string) {
		enrollmentCount++
		execNoErr(t, db, `INSERT INTO mdm_windows_enrollments (mdm_device_id, mdm_hardware_id, device_state, device_type, device_name,
			enroll_type, enroll_user_id, enroll_proto_version, enroll_client_version, host_uuid)
			VALUES (?, ?, 'MDMDeviceEnrolledEnrolled', 'CIMClient_Windows', 'DESKTOP', 'ProgrammaticEnrollment', '', '5.0', '10.0', ?)`,
			fmt.Sprintf("device-%d", enrollmentCount), fmt.Sprintf("hardware-%d", enrollmentCount), hostUUID)
	}

	single := newHost("single", "windows", true)
	setMDM(single, true, fleetMDMID)
	enroll("single")

	enrolledOld := newHost("enrolled", "windows", false)
	setMDM(enrolledOld, true, fleetMDMID)
	setMDM(newHost("enrolled", "windows", true), false, fleetMDMID)
	enroll("enrolled")

	fleetOld := newHost("fleet", "windows", false)
	setMDM(fleetOld, true, fleetMDMID)
	setMDM(newHost("fleet", "windows", true), true, intuneMDMID)
	enroll("fleet")

	identifierOld := newHost("identifier", "windows", true)
	setMDM(identifierOld, true, fleetMDMID)
	setMDM(newHost("identifier", "windows", false), true, fleetMDMID)
	enroll("identifier")

	setMDM(newHost("newest", "windows", false), true, fleetMDMID)
	newest := newHost("newest", "windows", false)
	setMDM(newest, true, fleetMDMID)
	enroll("newest")

	// Hosts with no host_mdm row rank as not enrolled.
	noMDMRow := newHost("no-mdm-row", "windows", false)
	enroll("no-mdm-row")

	caseOld := newHost("case-uuid", "windows", false)
	setMDM(caseOld, true, fleetMDMID)
	caseNew := newHost("CASE-UUID", "windows", false)
	setMDM(caseNew, true, fleetMDMID)
	enroll("Case-Uuid")

	// A macOS host sharing a Windows device's UUID is never its Windows owner.
	setMDM(newHost("bootcamp", "darwin", true), true, fleetMDMID)
	bootcampWindows := newHost("bootcamp", "windows", false)
	setMDM(bootcampWindows, false, fleetMDMID)
	enroll("bootcamp")

	setMDM(newHost("mac-only", "darwin", true), true, fleetMDMID)
	enroll("mac-only")

	newHost("unlinked", "windows", true)
	enroll("")

	applyNext(t, db)

	var owners []struct {
		HostUUID    string `db:"host_uuid"`
		MDMPlatform string `db:"mdm_platform"`
		HostID      uint   `db:"host_id"`
		ClaimedBy   string `db:"claimed_by"`
	}
	require.NoError(t, db.Select(&owners, `SELECT host_uuid, mdm_platform, host_id, claimed_by FROM host_mdm_uuid_owners ORDER BY host_id`))
	got := make(map[uint]string, len(owners))
	for _, o := range owners {
		require.Equal(t, "windows", o.MDMPlatform)
		require.Equal(t, "migration", o.ClaimedBy)
		got[o.HostID] = o.HostUUID
	}
	require.Equal(t, map[uint]string{
		single:          "single",
		enrolledOld:     "enrolled",
		fleetOld:        "fleet",
		identifierOld:   "identifier",
		newest:          "newest",
		noMDMRow:        "no-mdm-row",
		caseNew:         "CASE-UUID",
		bootcampWindows: "bootcamp",
	}, got)

	// The primary key compares UUIDs case-insensitively, like hosts.uuid.
	_, err := db.Exec(
		`INSERT INTO host_mdm_uuid_owners (host_uuid, mdm_platform, host_id, claimed_by) VALUES ('case-uuid', 'windows', ?, 'test')`, caseOld)
	require.ErrorContains(t, err, "Duplicate entry")
}
