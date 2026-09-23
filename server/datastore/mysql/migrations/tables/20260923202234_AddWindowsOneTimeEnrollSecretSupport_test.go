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

	applyNext(t, db)

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

	// The name the reconciler upserts is now free in every team.
	var taken int
	require.NoError(t, db.Get(&taken, `
		SELECT COUNT(*) FROM mdm_windows_configuration_profiles
		WHERE name = 'Fleetd enroll secret'`))
	require.Zero(t, taken)
}
