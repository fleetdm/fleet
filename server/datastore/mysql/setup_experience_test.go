package mysql

import (
	"context"
	"database/sql"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestSetupExperience(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"SetupExperienceScriptCRUD", testSetupExperienceScriptCRUD},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testSetupExperienceScriptCRUD(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	// create a script for team1
	wantScript1 := &fleet.Script{
		Name:           "script",
		TeamID:         &team1.ID,
		ScriptContents: "echo foo",
	}

	err = ds.SetSetupExperienceScript(ctx, wantScript1)
	require.NoError(t, err)

	// get the script for team1
	gotScript1, err := ds.GetSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err)
	require.NotNil(t, gotScript1)
	require.Equal(t, wantScript1.Name, gotScript1.Name)
	require.Equal(t, wantScript1.TeamID, gotScript1.TeamID)
	require.NotZero(t, gotScript1.ScriptContentID)

	b, err := ds.GetAnyScriptContents(ctx, gotScript1.ScriptContentID)
	require.NoError(t, err)
	require.Equal(t, wantScript1.ScriptContents, string(b))

	// create a script for team2
	wantScript2 := &fleet.Script{
		Name:           "script",
		TeamID:         &team2.ID,
		ScriptContents: "echo bar",
	}

	err = ds.SetSetupExperienceScript(ctx, wantScript2)
	require.NoError(t, err)

	// get the script for team2
	gotScript2, err := ds.GetSetupExperienceScript(ctx, &team2.ID)
	require.NoError(t, err)
	require.NotNil(t, gotScript2)
	require.Equal(t, wantScript2.Name, gotScript2.Name)
	require.Equal(t, wantScript2.TeamID, gotScript2.TeamID)
	require.NotZero(t, gotScript2.ScriptContentID)
	require.NotEqual(t, gotScript1.ScriptContentID, gotScript2.ScriptContentID)

	b, err = ds.GetAnyScriptContents(ctx, gotScript2.ScriptContentID)
	require.NoError(t, err)
	require.Equal(t, wantScript2.ScriptContents, string(b))

	// create a script with no team id
	wantScriptNoTeam := &fleet.Script{
		Name:           "script",
		ScriptContents: "echo bar",
	}

	err = ds.SetSetupExperienceScript(ctx, wantScriptNoTeam)
	require.NoError(t, err)

	// get the script nil team id is equivalent to team id 0
	gotScriptNoTeam, err := ds.GetSetupExperienceScript(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, gotScriptNoTeam)
	require.Equal(t, wantScriptNoTeam.Name, gotScriptNoTeam.Name)
	require.Nil(t, gotScriptNoTeam.TeamID)
	require.NotZero(t, gotScriptNoTeam.ScriptContentID)
	require.Equal(t, gotScript2.ScriptContentID, gotScriptNoTeam.ScriptContentID) // should be the same as team2

	b, err = ds.GetAnyScriptContents(ctx, gotScriptNoTeam.ScriptContentID)
	require.NoError(t, err)
	require.Equal(t, wantScriptNoTeam.ScriptContents, string(b))

	// try to create another with name "script" and no team id
	var existsErr fleet.AlreadyExistsError
	err = ds.SetSetupExperienceScript(ctx, &fleet.Script{Name: "script", ScriptContents: "echo baz"})
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)

	// try to create another script with no team id and a different name
	err = ds.SetSetupExperienceScript(ctx, &fleet.Script{Name: "script2", ScriptContents: "echo baz"})
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)

	// try to add a script for a team that doesn't exist
	var fkErr fleet.ForeignKeyError
	err = ds.SetSetupExperienceScript(ctx, &fleet.Script{TeamID: ptr.Uint(42), Name: "script", ScriptContents: "echo baz"})
	require.Error(t, err)
	require.ErrorAs(t, err, &fkErr)

	// delete the script for team1
	err = ds.DeleteSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err)

	// get the script for team1
	_, err = ds.GetSetupExperienceScript(ctx, &team1.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// try to delete script for team1 again
	err = ds.DeleteSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err) // TODO: confirm if we want to return not found on deletes

	// try to delete script for team that doesn't exist
	err = ds.DeleteSetupExperienceScript(ctx, ptr.Uint(42))
	require.NoError(t, err) // TODO: confirm if we want to return not found on deletes

	// add same script for team1 again
	err = ds.SetSetupExperienceScript(ctx, wantScript1)
	require.NoError(t, err)

	// get the script for team1
	oldScript1 := gotScript1
	newScript1, err := ds.GetSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err)
	require.NotNil(t, newScript1)
	require.Equal(t, wantScript1.Name, newScript1.Name)
	require.Equal(t, wantScript1.TeamID, newScript1.TeamID)
	require.NotZero(t, newScript1.ScriptContentID)
	// script contents are deleted by CleanupUnusedScriptContents not by DeleteSetupExperienceScript
	// so the content id should be the same as the old
	require.Equal(t, oldScript1.ScriptContentID, newScript1.ScriptContentID)
}
