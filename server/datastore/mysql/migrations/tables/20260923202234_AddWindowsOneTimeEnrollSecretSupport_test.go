package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260923202234(t *testing.T) {
	db := applyUpToPrev(t)

	// Seeded before the migration runs, because the rename half of it acts on profiles that already exist.
	insertProfile := func(uuid string, teamID uint, name, syncml string) {
		execNoErr(t, db, `
			INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml, uploaded_at)
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP())`, uuid, teamID, name, syncml)
		execNoErr(t, db, `
			INSERT INTO host_mdm_windows_profiles (host_uuid, profile_uuid, profile_name, command_uuid, status, operation_type)
			VALUES (?, ?, ?, ?, 'verified', 'install')`, "host-"+uuid, uuid, name, "cmd-"+uuid)
	}

	// The profile whose name this release claims, authored by a customer before the name was reserved.
	conflicting := "w11111111-1111-1111-1111-111111111111"
	insertProfile(conflicting, 0, "Fleetd enroll secret", "<Replace><Item><Target><LocURI>./Device/Custom</LocURI></Target></Item></Replace>")

	// The same name in another team has to move too, since the reconciler claims it per team.
	conflictingTeam := "w22222222-2222-2222-2222-222222222222"
	insertProfile(conflictingTeam, 1, "Fleetd enroll secret", "<Add><Item><Target><LocURI>./Device/Other</LocURI></Target></Item></Add>")

	// A profile that merely looks similar keeps its name.
	untouched := "w33333333-3333-3333-3333-333333333333"
	insertProfile(untouched, 0, "Fleetd enroll secret backup", "<Replace><Item><Target><LocURI>./Device/Keep</LocURI></Target></Item></Replace>")

	// Fleet's own profile, identified by its payload rather than its name, must survive a re-run untouched.
	fleetOwned := "w44444444-4444-4444-4444-444444444444"
	insertProfile(fleetOwned, 2, "Fleetd enroll secret",
		"<Add><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/FleetdEnrollSecret/Policy/FleetdEnrollSecretAdmx</LocURI></Target></Item></Add>")

	applyNext(t, db)

	// --- the enrollment binding ---

	newEnrollment := func(hardwareID string) int64 {
		return execNoErrLastID(t, db, `INSERT INTO mdm_windows_enrollments
			(mdm_device_id, mdm_hardware_id, device_state, device_type, device_name,
			 enroll_type, enroll_user_id, enroll_proto_version, enroll_client_version)
			VALUES (?, ?, 'state', 'CIMClient_Windows', 'DESKTOP-1', 'Device', 'upn@example.com', '4.0', '10.0')`,
			"device-"+hardwareID, hardwareID)
	}

	enrollmentID := newEnrollment("hw-1")

	// A Windows secret is minted before any hosts row exists, so it binds to the enrollment and leaves host_id NULL.
	execNoErr(t, db, `INSERT INTO host_one_time_enroll_secrets
		(secret, host_id, mdm_windows_enrollment_id, platform, hardware_uuid, hardware_serial)
		VALUES ('win-secret', NULL, ?, 'windows', '', '')`, enrollmentID)

	var bound int
	require.NoError(t, db.Get(&bound, `SELECT COUNT(*) FROM host_one_time_enroll_secrets
		WHERE mdm_windows_enrollment_id = ? AND host_id IS NULL`, enrollmentID))
	require.Equal(t, 1, bound)

	// The Apple path keeps working: no enrollment binding, host_id set.
	execNoErr(t, db, `INSERT INTO host_one_time_enroll_secrets
		(secret, host_id, platform, hardware_uuid, hardware_serial)
		VALUES ('mac-secret', 1, 'darwin', 'UUID-1', 'SERIAL-1')`)

	// The binding must reference a real enrollment.
	_, err := db.Exec(`INSERT INTO host_one_time_enroll_secrets
		(secret, mdm_windows_enrollment_id, platform, hardware_uuid, hardware_serial)
		VALUES ('orphan', 999999, 'windows', '', '')`)
	require.Error(t, err, "a secret must not bind to a nonexistent enrollment")

	// Re-enrollment deletes the enrollment row, and that is what invalidates the secret minted for it. The Apple row,
	// which has no binding, must survive.
	execNoErr(t, db, `DELETE FROM mdm_windows_enrollments WHERE id = ?`, enrollmentID)

	var remaining []string
	require.NoError(t, db.Select(&remaining, `SELECT secret FROM host_one_time_enroll_secrets ORDER BY secret`))
	require.Equal(t, []string{"mac-secret"}, remaining, "the Windows secret must cascade away with its enrollment")

	// --- the profile rename ---

	name := func(uuid string) string {
		var n string
		require.NoError(t, db.Get(&n, `SELECT name FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`, uuid))
		return n
	}
	hostName := func(uuid string) string {
		var n string
		require.NoError(t, db.Get(&n, `SELECT profile_name FROM host_mdm_windows_profiles WHERE profile_uuid = ?`, uuid))
		return n
	}

	// Renamed out of the way, and the denormalized copy on the host row moved with it. Leaving those out of step would show
	// the old name everywhere the host's profiles are listed.
	require.Equal(t, "Fleetd enroll secret (renamed w11111111-1111-1111-1111-111111111111)", name(conflicting))
	require.Equal(t, "Fleetd enroll secret (renamed w11111111-1111-1111-1111-111111111111)", hostName(conflicting))
	require.Equal(t, "Fleetd enroll secret (renamed w22222222-2222-2222-2222-222222222222)", name(conflictingTeam))
	require.Equal(t, "Fleetd enroll secret (renamed w22222222-2222-2222-2222-222222222222)", hostName(conflictingTeam))

	require.Equal(t, "Fleetd enroll secret backup", name(untouched))
	require.Equal(t, "Fleetd enroll secret backup", hostName(untouched))

	require.Equal(t, "Fleetd enroll secret", name(fleetOwned))
	require.Equal(t, "Fleetd enroll secret", hostName(fleetOwned))

	// The name the reconciler upserts is now free in every team that had a conflict.
	var taken int
	require.NoError(t, db.Get(&taken, `
		SELECT COUNT(*) FROM mdm_windows_configuration_profiles
		WHERE name = 'Fleetd enroll secret' AND team_id IN (0, 1)`))
	require.Zero(t, taken)
}
