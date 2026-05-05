package tables

import (
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"
)

func TestUp_20250430112622_NoProfile(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	assertRowCount(t, db, "mdm_configuration_profile_variables", 0)
}

type profileVarTuple struct {
	ProfileUUID string `db:"apple_profile_uuid"`
	VarID       uint   `db:"fleet_variable_id"`
}

func TestUp_20250430112622_SingleProfileWithVar(t *testing.T) {
	db := applyUpToPrev(t)

	prof1 := insertAppleConfigProfile(t, db, "N1", "I1", "FLEET_VAR_HOST_END_USER_IDP_USERNAME")
	var varID uint
	err := db.Get(&varID, `SELECT id FROM fleet_variables WHERE name = 'FLEET_VAR_HOST_END_USER_IDP_USERNAME'`)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	var profs []profileVarTuple
	err = db.Select(&profs, `SELECT apple_profile_uuid, fleet_variable_id FROM mdm_configuration_profile_variables`)
	require.NoError(t, err)
	require.Len(t, profs, 1)
	require.Equal(t, profileVarTuple{prof1, varID}, profs[0])
}

func TestUp_20250430112622_MultipleProfilesWithVar(t *testing.T) {
	runWithNProfiles(t, 10)
}

func TestUp_20250430112622_ProfilesWithoutVariable(t *testing.T) {
	db := applyUpToPrev(t)

	insertAppleConfigProfile(t, db, "N1", "I1")
	insertAppleConfigProfile(t, db, "N2", "I2")
	insertAppleConfigProfile(t, db, "N3", "I3")

	// Apply current migration.
	applyNext(t, db)

	var profs []profileVarTuple
	err := db.Select(&profs, `SELECT apple_profile_uuid, fleet_variable_id FROM mdm_configuration_profile_variables`)
	require.NoError(t, err)
	require.Len(t, profs, 0)
}

func TestUp_20250430112622_ProfilesUnknownVariable(t *testing.T) {
	db := applyUpToPrev(t)

	insertAppleConfigProfile(t, db, "N1", "I1", "FLEET_VAR_NO_SUCH_VARIABLE")
	insertAppleConfigProfile(t, db, "N2", "I2", "FLEET_VAR_WHAT_I_CANT_EVEN")

	// Apply current migration.
	applyNext(t, db)

	var profs []profileVarTuple
	err := db.Select(&profs, `SELECT apple_profile_uuid, fleet_variable_id FROM mdm_configuration_profile_variables`)
	require.NoError(t, err)
	require.Len(t, profs, 0)
}

func TestUp_20250430112622_ExactBatch(t *testing.T) {
	runWithNProfiles(t, 100)
}

func TestUp_20250430112622_OverBatch(t *testing.T) {
	runWithNProfiles(t, 101)
}

func TestUp_20250430112622_LoadTest(t *testing.T) {
	runWithNProfiles(t, 1011)
}

func runWithNProfiles(t *testing.T, n int) {
	db := applyUpToPrev(t)

	var defs []varDef
	err := db.Select(&defs, `SELECT id, name, is_prefix FROM fleet_variables`)
	require.NoError(t, err)

	nano := uint64(time.Now().UnixNano()) // nolint:gosec
	t.Logf("random seed: %d", nano)
	randSeed := rand.New(rand.NewSource(nano))
	expectedProfs := createProfilesWithRandomVars(t, db, randSeed, n, defs)

	// Apply current migration.
	applyNext(t, db)

	var profs []profileVarTuple
	err = db.Select(&profs, `SELECT apple_profile_uuid, fleet_variable_id FROM mdm_configuration_profile_variables`)
	require.NoError(t, err)
	require.Len(t, profs, len(expectedProfs))
	require.ElementsMatch(t, expectedProfs, profs)
}

func createProfilesWithRandomVars(t *testing.T, db *sqlx.DB, randSeed *rand.Rand, n int, vars []varDef) []profileVarTuple {
	profs := make([]profileVarTuple, 0, n)
	for i := range n {
		// select a random number of variables to assign to the profile
		numVars := randSeed.Intn(len(vars) + 1) // +1 because 0 means no var
		// shuffle the vars so that the first ones are not over-represented
		randSeed.Shuffle(len(vars), func(i, j int) {
			vars[i], vars[j] = vars[j], vars[i]
		})

		addVars := vars[:numVars]
		varNames := make([]string, 0, numVars)
		varIDs := make([]uint, 0, numVars)
		for _, v := range addVars {
			if v.IsPrefix {
				varNames = append(varNames, v.Name+"ABC")
			} else {
				varNames = append(varNames, v.Name)
			}
			varIDs = append(varIDs, v.ID)
		}
		profUUID := insertAppleConfigProfile(t, db, "N"+fmt.Sprint(i), "I"+fmt.Sprint(i), varNames...)
		for _, id := range varIDs {
			profs = append(profs, profileVarTuple{profUUID, id})
		}
	}

	return profs
}
