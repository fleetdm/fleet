package mysql

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestScripts(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"HostScriptResult", testHostScriptResult},
		{"Scripts", testScripts},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testHostScriptResult(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// no script saved yet
	pending, err := ds.ListPendingHostScriptExecutions(ctx, 1, time.Second)
	require.NoError(t, err)
	require.Empty(t, pending)

	_, err = ds.GetHostScriptExecutionResult(ctx, "abc")
	require.Error(t, err)
	var nfe *notFoundError
	require.ErrorAs(t, err, &nfe)

	// create a createdScript execution request
	createdScript, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         1,
		ScriptContents: "echo",
	})
	require.NoError(t, err)
	require.NotZero(t, createdScript.ID)
	require.NotEmpty(t, createdScript.ExecutionID)
	require.Equal(t, uint(1), createdScript.HostID)
	require.NotEmpty(t, createdScript.ExecutionID)
	require.Equal(t, "echo", createdScript.ScriptContents)
	require.Nil(t, createdScript.ExitCode)
	require.Empty(t, createdScript.Output)

	// the script execution is now listed as pending for this host
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 1, 10*time.Second)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, createdScript.ID, pending[0].ID)

	// waiting for a second and an ignore of 0s ignores this script
	time.Sleep(time.Second)
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 1, 0)
	require.NoError(t, err)
	require.Empty(t, pending)

	// record a result for this execution
	err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      1,
		ExecutionID: createdScript.ExecutionID,
		Output:      "foo",
		Runtime:     2,
		ExitCode:    0,
	})
	require.NoError(t, err)

	// it is not pending anymore
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 1, 10*time.Second)
	require.NoError(t, err)
	require.Empty(t, pending)

	// the script result can be retrieved
	script, err := ds.GetHostScriptExecutionResult(ctx, createdScript.ExecutionID)
	require.NoError(t, err)
	expectScript := *createdScript
	expectScript.Output = "foo"
	expectScript.Runtime = 2
	expectScript.ExitCode = ptr.Int64(0)
	require.Equal(t, &expectScript, script)

	// create another script execution request
	createdScript, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         1,
		ScriptContents: "echo2",
	})
	require.NoError(t, err)
	require.NotZero(t, createdScript.ID)
	require.NotEmpty(t, createdScript.ExecutionID)

	// the script result can be retrieved even if it has no result yet
	script, err = ds.GetHostScriptExecutionResult(ctx, createdScript.ExecutionID)
	require.NoError(t, err)
	require.Equal(t, createdScript, script)

	// record a result for this execution, with an output that is too large
	largeOutput := strings.Repeat("a", 1000) +
		strings.Repeat("b", 1000) +
		strings.Repeat("c", 1000) +
		strings.Repeat("d", 1000) +
		strings.Repeat("e", 1000) +
		strings.Repeat("f", 1000) +
		strings.Repeat("g", 1000) +
		strings.Repeat("h", 1000) +
		strings.Repeat("i", 1000) +
		strings.Repeat("j", 1000) +
		strings.Repeat("k", 1000)
	expectedOutput := strings.Repeat("b", 1000) +
		strings.Repeat("c", 1000) +
		strings.Repeat("d", 1000) +
		strings.Repeat("e", 1000) +
		strings.Repeat("f", 1000) +
		strings.Repeat("g", 1000) +
		strings.Repeat("h", 1000) +
		strings.Repeat("i", 1000) +
		strings.Repeat("j", 1000) +
		strings.Repeat("k", 1000)

	err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      1,
		ExecutionID: createdScript.ExecutionID,
		Output:      largeOutput,
		Runtime:     10,
		ExitCode:    1,
	})
	require.NoError(t, err)

	// the script result can be retrieved
	script, err = ds.GetHostScriptExecutionResult(ctx, createdScript.ExecutionID)
	require.NoError(t, err)
	require.Equal(t, expectedOutput, script.Output)
}

func testScripts(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// get unknown script
	_, err := ds.Script(ctx, 123)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	// create global scriptGlobal
	scriptGlobal, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "a",
		ScriptContents: "echo",
	})
	require.NoError(t, err)
	require.NotZero(t, scriptGlobal.ID)
	require.Nil(t, scriptGlobal.TeamID)
	require.Equal(t, "a", scriptGlobal.Name)
	require.Empty(t, scriptGlobal.ScriptContents) // we don't return the contents

	// create team script but team does not exist
	_, err = ds.NewScript(ctx, &fleet.Script{
		Name:           "a",
		TeamID:         ptr.Uint(123),
		ScriptContents: "echo",
	})
	require.Error(t, err)
	var fkErr fleet.ForeignKeyError
	require.ErrorAs(t, err, &fkErr)

	// create a team and a script for that team with the same name as global
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	scriptTeam, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "a",
		TeamID:         &tm.ID,
		ScriptContents: "echo",
	})
	require.NoError(t, err)
	require.NotEqual(t, scriptGlobal.ID, scriptTeam.ID)
	require.NotNil(t, scriptTeam.TeamID)
	require.Equal(t, tm.ID, *scriptTeam.TeamID)

	// try to create another team script with the same name
	_, err = ds.NewScript(ctx, &fleet.Script{
		Name:           "a",
		TeamID:         &tm.ID,
		ScriptContents: "echo",
	})
	require.Error(t, err)
	var existsErr fleet.AlreadyExistsError
	require.ErrorAs(t, err, &existsErr)

	// same for a global script
	_, err = ds.NewScript(ctx, &fleet.Script{
		Name:           "a",
		ScriptContents: "echo",
	})
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)

	// create a script with a different name for the team works
	_, err = ds.NewScript(ctx, &fleet.Script{
		Name:           "b",
		TeamID:         &tm.ID,
		ScriptContents: "echo",
	})
	require.NoError(t, err)

	// deleting script "a for the team, then we can re-create it
	err = ds.DeleteScript(ctx, scriptTeam.ID)
	require.NoError(t, err)
	scriptTeam2, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "a",
		TeamID:         &tm.ID,
		ScriptContents: "echo",
	})
	require.NoError(t, err)
	require.NotEqual(t, scriptTeam.ID, scriptTeam2.ID)
}
