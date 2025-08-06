package mysql

import (
	"context"
	_ "embed"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScripts(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"HostScriptResult", testHostScriptResult},
		{"DEPRestoredHost", testListPendingScriptDEPRestoration},
		{"Scripts", testScripts},
		{"ListScripts", testListScripts},
		{"GetHostScriptDetails", testGetHostScriptDetails},
		{"BatchSetScripts", testBatchSetScripts},
		{"TestLockHostViaScript", testLockHostViaScript},
		{"TestUnlockHostViaScript", testUnlockHostViaScript},
		{"TestLockUnlockWipeViaScripts", testLockUnlockWipeViaScripts},
		{"TestLockUnlockManually", testLockUnlockManually},
		{"TestInsertScriptContents", testInsertScriptContents},
		{"TestCleanupUnusedScriptContents", testCleanupUnusedScriptContents},
		{"TestGetAnyScriptContents", testGetAnyScriptContents},
		{"TestDeleteScriptsAssignedToPolicy", testDeleteScriptsAssignedToPolicy},
		{"TestDeletePendingHostScriptExecutionsForPolicy", testDeletePendingHostScriptExecutionsForPolicy},
		{"UpdateScriptContents", testUpdateScriptContents},
		{"UpdateDeletingUpcomingScriptExecutions", testUpdateDeletingUpcomingScriptExecutions},
		{"BatchExecute", testBatchExecuteWithStatus},
		{"DeleteScriptActivatesNextActivity", testDeleteScriptActivatesNextActivity},
		{"BatchSetScriptActivatesNextActivity", testBatchSetScriptActivatesNextActivity},
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
	pending, err := ds.ListPendingHostScriptExecutions(ctx, 1, false)
	require.NoError(t, err)
	require.Empty(t, pending)

	_, err = ds.GetHostScriptExecutionResult(ctx, "abc")
	require.Error(t, err)
	var nfe *common_mysql.NotFoundError
	require.ErrorAs(t, err, &nfe)

	// create a createdScript execution request (with a user)
	u := test.NewUser(t, ds, "Bob", "bob@example.com", true)
	createdScript1, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         1,
		ScriptContents: "echo",
		UserID:         &u.ID,
		SyncRequest:    true,
	})
	require.NoError(t, err)
	require.NotZero(t, createdScript1.ID)
	require.NotEmpty(t, createdScript1.ExecutionID)
	require.Equal(t, uint(1), createdScript1.HostID)
	require.NotEmpty(t, createdScript1.ExecutionID)
	require.Equal(t, "echo", createdScript1.ScriptContents)
	require.Nil(t, createdScript1.ExitCode)
	require.Empty(t, createdScript1.Output)
	require.NotNil(t, createdScript1.UserID)
	require.Equal(t, u.ID, *createdScript1.UserID)
	require.True(t, createdScript1.SyncRequest)
	// createdScript1 is now activated, as the queue was empty

	// the script execution is now listed as pending for this host
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 1, false)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, createdScript1.ID, pending[0].ID)

	// the script execution isn't visible when looking at internal-only scripts
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 1, true)
	require.NoError(t, err)
	require.Empty(t, pending)

	// record a result for this execution
	hsr, action, err := ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      1,
		ExecutionID: createdScript1.ExecutionID,
		Output:      "foo",
		Runtime:     2,
		ExitCode:    0,
		Timeout:     300,
	})
	require.NoError(t, err)
	assert.Empty(t, action)
	assert.NotNil(t, hsr)

	// record a duplicate result for this execution, will be ignored
	hsr, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      1,
		ExecutionID: createdScript1.ExecutionID,
		Output:      "foobarbaz",
		Runtime:     22,
		ExitCode:    1,
		Timeout:     360,
	})
	require.NoError(t, err)
	require.Nil(t, hsr)

	// it is not pending anymore
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 1, false)
	require.NoError(t, err)
	require.Empty(t, pending)

	// the script result can be retrieved
	script, err := ds.GetHostScriptExecutionResult(ctx, createdScript1.ExecutionID)
	require.NoError(t, err)
	expectScript := *createdScript1
	expectScript.Output = "foo"
	expectScript.Runtime = 2
	expectScript.ExitCode = ptr.Int64(0)
	expectScript.Timeout = ptr.Int(300)
	expectScript.CreatedAt, script.CreatedAt = time.Time{}, time.Time{}
	require.Equal(t, &expectScript, script)

	// create another script execution request (null user id this time)
	time.Sleep(time.Millisecond) // ensure a different timestamp
	createdScript2, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         1,
		ScriptContents: "echo2",
	})
	require.NoError(t, err)
	require.NotZero(t, createdScript2.ID)
	require.NotEmpty(t, createdScript2.ExecutionID)
	require.Nil(t, createdScript2.UserID)
	require.False(t, createdScript2.SyncRequest)
	// createdScript2 is now activated as the queue was empty

	// the script execution is now listed as pending for this host
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 1, false)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, createdScript2.ID, pending[0].ID)

	// the script result can be retrieved even if it has no result yet
	script, err = ds.GetHostScriptExecutionResult(ctx, createdScript2.ExecutionID)
	require.NoError(t, err)
	expectedScript := *createdScript2
	expectedScript.CreatedAt, script.CreatedAt = time.Time{}, time.Time{}
	require.Equal(t, &expectedScript, script)

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
	// Note that the expectation is that the "a"s get truncated
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

	_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      1,
		ExecutionID: createdScript2.ExecutionID,
		Output:      largeOutput,
		Runtime:     10,
		ExitCode:    1,
		Timeout:     300,
	})
	require.NoError(t, err)

	// the script result can be retrieved
	script, err = ds.GetHostScriptExecutionResult(ctx, createdScript2.ExecutionID)
	require.NoError(t, err)
	require.Equal(t, expectedOutput, script.Output)

	// create an async execution request
	time.Sleep(time.Millisecond) // ensure a different timestamp
	createdScript3, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         1,
		ScriptContents: "echo 3",
		UserID:         &u.ID,
		SyncRequest:    false,
	})
	require.NoError(t, err)
	require.NotZero(t, createdScript3.ID)
	require.NotEmpty(t, createdScript3.ExecutionID)
	require.Equal(t, uint(1), createdScript3.HostID)
	require.NotEmpty(t, createdScript3.ExecutionID)
	require.Equal(t, "echo 3", createdScript3.ScriptContents)
	require.Nil(t, createdScript3.ExitCode)
	require.Empty(t, createdScript3.Output)
	require.NotNil(t, createdScript3.UserID)
	require.Equal(t, u.ID, *createdScript3.UserID)
	require.False(t, createdScript3.SyncRequest)
	// createdScript3 is now activated as the queue was empty

	// the script execution is now listed as pending for this host
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 1, false)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, createdScript3.ID, pending[0].ID)

	// modify the upcoming script to be a sync script that has
	// been pending for a long time doesn't change result
	// https://github.com/fleetdm/fleet/issues/22866#issuecomment-2575961141
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "UPDATE upcoming_activities SET created_at = ?, payload = JSON_SET(payload, '$.sync_request', ?) WHERE id = ?",
			time.Now().Add(-24*time.Hour), true, createdScript3.ID)
		return err
	})

	// the script is still pending
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 1, false)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, createdScript3.ExecutionID, pending[0].ExecutionID)

	// check that scripts with large unsigned error codes get
	// converted to signed error codes
	createdUnsignedScript, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         1,
		ScriptContents: "echo",
		UserID:         &u.ID,
		SyncRequest:    true,
	})
	require.NoError(t, err)

	// record a result for createdScript3 so that the unsigned script gets activated
	_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      1,
		ExecutionID: createdScript3.ExecutionID,
		Output:      "foo",
		Runtime:     1,
		ExitCode:    0,
		Timeout:     0,
	})
	require.NoError(t, err)
	// createdUnsignedScript is now activated, record its result

	unsignedScriptResult, _, err := ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      1,
		ExecutionID: createdUnsignedScript.ExecutionID,
		Output:      "foo",
		Runtime:     1,
		ExitCode:    math.MaxUint32,
		Timeout:     300,
	})
	require.NoError(t, err)
	require.EqualValues(t, -1, *unsignedScriptResult.ExitCode)
}

func testListPendingScriptDEPRestoration(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host := test.NewHost(t, ds, "host", "10.0.0.1", "1", "uuid1", time.Now())

	// no script saved yet
	pending, err := ds.ListPendingHostScriptExecutions(ctx, host.ID, false)
	require.NoError(t, err)
	require.Empty(t, pending)

	// create a createdScript execution request (with a user)
	u := test.NewUser(t, ds, "Bob", "bob@example.com", true)
	createdScript, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         host.ID,
		ScriptContents: "echo",
		UserID:         &u.ID,
		SyncRequest:    true,
	})
	require.NoError(t, err)
	require.NotZero(t, createdScript.ID)
	require.NotEmpty(t, createdScript.ExecutionID)
	require.Equal(t, uint(1), createdScript.HostID)
	require.NotEmpty(t, createdScript.ExecutionID)
	require.Equal(t, "echo", createdScript.ScriptContents)
	require.Nil(t, createdScript.ExitCode)
	require.Empty(t, createdScript.Output)
	require.NotNil(t, createdScript.UserID)
	require.Equal(t, u.ID, *createdScript.UserID)
	require.True(t, createdScript.SyncRequest)

	// the script execution is now listed as pending for this host
	pending, err = ds.ListPendingHostScriptExecutions(ctx, host.ID, false)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, createdScript.ID, pending[0].ID)

	// Set LastEnrolledAt before deleting the host (simulating a DEP enrolled host)
	host.LastEnrolledAt = time.Now()

	err = ds.DeleteHost(ctx, host.ID)
	require.NoError(t, err)

	err = ds.RestoreMDMApplePendingDEPHost(ctx, host)
	require.NoError(t, err)

	// the script execution is no longer listed as pending for this host
	pending, err = ds.ListPendingHostScriptExecutions(ctx, host.ID, false)
	require.NoError(t, err)
	require.Len(t, pending, 0)
}

func testScripts(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// get unknown script
	_, err := ds.Script(ctx, 123)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	// get unknown script contents
	_, err = ds.GetScriptContents(ctx, 123)
	require.ErrorAs(t, err, &nfe)
	_, err = ds.GetAnyScriptContents(ctx, 123)
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

	// get the global script
	script, err := ds.Script(ctx, scriptGlobal.ID)
	require.NoError(t, err)
	require.Equal(t, scriptGlobal, script)

	// get the global script contents
	contents, err := ds.GetScriptContents(ctx, scriptGlobal.ID)
	require.NoError(t, err)
	require.Equal(t, "echo", string(contents))
	contents, err = ds.GetAnyScriptContents(ctx, scriptGlobal.ID)
	require.NoError(t, err)
	require.Equal(t, "echo", string(contents))

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
		ScriptContents: "echo 'team'",
	})
	require.NoError(t, err)
	require.NotEqual(t, scriptGlobal.ID, scriptTeam.ID)
	require.NotNil(t, scriptTeam.TeamID)
	require.Equal(t, tm.ID, *scriptTeam.TeamID)

	// get the team script
	script, err = ds.Script(ctx, scriptTeam.ID)
	require.NoError(t, err)
	require.Equal(t, scriptTeam, script)

	// get the team script contents
	contents, err = ds.GetScriptContents(ctx, scriptTeam.ID)
	require.NoError(t, err)
	require.Equal(t, "echo 'team'", string(contents))
	contents, err = ds.GetAnyScriptContents(ctx, scriptTeam.ID)
	require.NoError(t, err)
	require.Equal(t, "echo 'team'", string(contents))

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

func testListScripts(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create three teams
	tm1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	tm2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	tm3, err := ds.NewTeam(ctx, &fleet.Team{Name: "team3"})
	require.NoError(t, err)

	// create 5 scripts for no team and team 1
	for i := 0; i < 5; i++ {
		_, err = ds.NewScript(ctx, &fleet.Script{
			Name:           string('a' + byte(i)), // i.e. "a", "b", "c", ...
			ScriptContents: "echo",
		})
		require.NoError(t, err)
		_, err = ds.NewScript(ctx, &fleet.Script{Name: string('a' + byte(i)), TeamID: &tm1.ID, ScriptContents: "echo"})
		require.NoError(t, err)
	}

	// create a single script for team 2
	_, err = ds.NewScript(ctx, &fleet.Script{Name: "a", TeamID: &tm2.ID, ScriptContents: "echo"})
	require.NoError(t, err)

	cases := []struct {
		opts      fleet.ListOptions
		teamID    *uint
		wantNames []string
		wantMeta  *fleet.PaginationMetadata
	}{
		{
			opts:      fleet.ListOptions{},
			wantNames: []string{"a", "b", "c", "d", "e"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
		{
			opts:      fleet.ListOptions{PerPage: 2},
			wantNames: []string{"a", "b"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
		},
		{
			opts:      fleet.ListOptions{Page: 1, PerPage: 2},
			wantNames: []string{"c", "d"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true},
		},
		{
			opts:      fleet.ListOptions{Page: 2, PerPage: 2},
			wantNames: []string{"e"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			opts:      fleet.ListOptions{PerPage: 3},
			teamID:    &tm1.ID,
			wantNames: []string{"a", "b", "c"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
		},
		{
			opts:      fleet.ListOptions{Page: 1, PerPage: 3},
			teamID:    &tm1.ID,
			wantNames: []string{"d", "e"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			opts:      fleet.ListOptions{Page: 2, PerPage: 3},
			teamID:    &tm1.ID,
			wantNames: nil,
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			opts:      fleet.ListOptions{PerPage: 3},
			teamID:    &tm2.ID,
			wantNames: []string{"a"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
		{
			opts:      fleet.ListOptions{Page: 0, PerPage: 2},
			teamID:    &tm3.ID,
			wantNames: nil,
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%v: %#v", c.teamID, c.opts), func(t *testing.T) {
			// always include metadata
			c.opts.IncludeMetadata = true
			scripts, meta, err := ds.ListScripts(ctx, c.teamID, c.opts)
			require.NoError(t, err)

			require.Equal(t, len(c.wantNames), len(scripts))
			require.Equal(t, c.wantMeta, meta)

			var gotNames []string
			if len(scripts) > 0 {
				gotNames = make([]string, len(scripts))
				for i, s := range scripts {
					gotNames[i] = s.Name
					require.Equal(t, c.teamID, s.TeamID)
				}
			}
			require.Equal(t, c.wantNames, gotNames)
		})
	}
}

func testGetHostScriptDetails(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	t.Cleanup(func() { ds.testActivateSpecificNextActivities = nil })

	names := []string{"script-1.sh", "script-2.sh", "script-3.sh", "script-4.sh", "script-5.sh"}
	for _, r := range append(names[1:], names[0]) {
		_, err := ds.NewScript(ctx, &fleet.Script{
			Name:           r,
			ScriptContents: "echo " + r,
		})
		require.NoError(t, err)
	}

	// create a windows script as well
	_, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "script-6.ps1",
		ScriptContents: `Write-Host "Hello, World!"`,
	})
	require.NoError(t, err)

	scripts, _, err := ds.ListScripts(ctx, nil, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, scripts, 6)

	insertResults := func(t *testing.T, hostID uint, script *fleet.Script, createdAt time.Time, execID string, exitCode *int64) {
		var scriptID *uint
		if script.ID != 0 {
			scriptID = &script.ID
		}
		hsr, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
			HostID:   hostID,
			ScriptID: scriptID,
		})
		require.NoError(t, err)
		ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, `UPDATE upcoming_activities SET execution_id = ?, created_at = ? WHERE execution_id = ?`,
				execID, createdAt, hsr.ExecutionID)
			return err
		})
		if exitCode != nil {
			ds.testActivateSpecificNextActivities = []string{execID}
			act, err := ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), hostID, "")
			require.NoError(t, err)
			require.ElementsMatch(t, act, ds.testActivateSpecificNextActivities)
			ds.testActivateSpecificNextActivities = nil

			_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
				HostID:      hostID,
				ExecutionID: execID,
				ExitCode:    int(*exitCode),
			})
			require.NoError(t, err)

			// force the test timestamp
			ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
				_, err := tx.ExecContext(ctx, `UPDATE host_script_results SET created_at = ? WHERE execution_id = ?`,
					createdAt, execID)
				return err
			})
		}
	}

	now := time.Now().UTC().Truncate(time.Second)

	// add some results for an ad-hoc, non-saved script, should not be included in results
	// create it first so that this one gets activated, and the other ones are never
	// activated automatically.
	_, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         42,
		ScriptContents: "echo script-6",
	})
	require.NoError(t, err)

	// add some results for script-1
	insertResults(t, 42, scripts[0], now.Add(-3*time.Minute), "execution-1-1", nil)
	insertResults(t, 42, scripts[0], now.Add(-1*time.Minute), "execution-1-2", nil) // last execution for script-1, status "pending"
	insertResults(t, 42, scripts[0], now.Add(-2*time.Minute), "execution-1-3", nil)

	// add some results for script-2
	insertResults(t, 42, scripts[1], now.Add(-3*time.Minute), "execution-2-1", ptr.Int64(0))
	insertResults(t, 42, scripts[1], now.Add(-1*time.Minute), "execution-2-2", ptr.Int64(1)) // last execution for script-2, status "error"

	// add some results for script-3
	insertResults(t, 42, scripts[2], now.Add(-1*time.Minute), "execution-3-1", ptr.Int64(0))
	insertResults(t, 42, scripts[2], now.Add(-1*time.Minute), "execution-3-2", ptr.Int64(0)) // last execution for script-3, status "ran"
	insertResults(t, 42, scripts[2], now.Add(-2*time.Minute), "execution-3-3", ptr.Int64(0))

	// add some results for script-4
	insertResults(t, 42, scripts[3], now.Add(-1*time.Minute), "execution-4-1", ptr.Int64(-2)) // last execution for script-4, status "error"

	// add a pending and a completed script execution for script-5
	insertResults(t, 42, scripts[4], now.Add(-2*time.Minute), "execution-5-1", ptr.Int64(0))
	insertResults(t, 42, scripts[4], now.Add(-3*time.Minute), "execution-5-2", nil) // upcoming is always latest, regardless of timestamp

	t.Run("results match expected formatting and filtering", func(t *testing.T) {
		res, _, err := ds.GetHostScriptDetails(ctx, 42, nil, fleet.ListOptions{}, "")
		require.NoError(t, err)
		require.Len(t, res, 6)
		for _, r := range res {
			switch r.ScriptID {
			case scripts[0].ID:
				require.Equal(t, scripts[0].Name, r.Name)
				require.NotNil(t, r.LastExecution)
				require.Equal(t, now.Add(-1*time.Minute), r.LastExecution.ExecutedAt)
				require.Equal(t, "execution-1-2", r.LastExecution.ExecutionID)
				require.Equal(t, "pending", r.LastExecution.Status)
			case scripts[1].ID:
				require.Equal(t, scripts[1].Name, r.Name)
				require.NotNil(t, r.LastExecution)
				require.Equal(t, now.Add(-1*time.Minute), r.LastExecution.ExecutedAt)
				require.Equal(t, "execution-2-2", r.LastExecution.ExecutionID)
				require.Equal(t, "error", r.LastExecution.Status)
			case scripts[2].ID:
				require.Equal(t, scripts[2].Name, r.Name)
				require.NotNil(t, r.LastExecution)
				require.Equal(t, now.Add(-1*time.Minute), r.LastExecution.ExecutedAt)
				require.Equal(t, "execution-3-2", r.LastExecution.ExecutionID)
				require.Equal(t, "ran", r.LastExecution.Status)
			case scripts[3].ID:
				require.Equal(t, scripts[3].Name, r.Name)
				require.NotNil(t, r.LastExecution)
				require.Equal(t, now.Add(-1*time.Minute), r.LastExecution.ExecutedAt)
				require.Equal(t, "execution-4-1", r.LastExecution.ExecutionID)
				require.Equal(t, "error", r.LastExecution.Status)
			case scripts[4].ID:
				require.Equal(t, scripts[4].Name, r.Name)
				require.NotNil(t, r.LastExecution)
				// require.Equal(t, now.Add(-3*time.Minute), r.LastExecution.ExecutedAt)
				require.Equal(t, "execution-5-2", r.LastExecution.ExecutionID)
				require.Equal(t, "pending", r.LastExecution.Status)
			case scripts[5].ID:
				require.Equal(t, scripts[5].Name, r.Name)
				require.Nil(t, r.LastExecution)
			default:
				t.Errorf("unexpected script id: %d", r.ScriptID)
			}
		}
	})

	t.Run("empty slice returned if no scripts", func(t *testing.T) {
		res, _, err := ds.GetHostScriptDetails(ctx, 42, ptr.Uint(1), fleet.ListOptions{}, "") // team 1 has no scripts
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res, 0)
	})

	t.Run("list options are supported", func(t *testing.T) {
		cases := []struct {
			opts      fleet.ListOptions
			teamID    *uint
			wantNames []string
			wantMeta  *fleet.PaginationMetadata
		}{
			{
				opts:      fleet.ListOptions{},
				wantNames: names,
				wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
			},
			{
				opts:      fleet.ListOptions{PerPage: 2},
				wantNames: names[:2],
				wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
			},
			{
				opts:      fleet.ListOptions{Page: 1, PerPage: 2},
				wantNames: names[2:4],
				wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true},
			},
			{
				opts:      fleet.ListOptions{Page: 2, PerPage: 2},
				wantNames: names[4:],
				wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
			},
		}
		for _, c := range cases {
			t.Run(fmt.Sprintf("%#v", c.opts), func(t *testing.T) {
				// always include metadata
				c.opts.IncludeMetadata = true
				// custom ordering is not supported, always by name
				c.opts.OrderKey = "name"
				results, meta, err := ds.GetHostScriptDetails(ctx, 42, nil, c.opts, "darwin")
				require.NoError(t, err)

				require.Equal(t, len(c.wantNames), len(results))
				require.Equal(t, c.wantMeta, meta)

				var gotNames []string
				if len(results) > 0 {
					gotNames = make([]string, len(results))
					for i, r := range results {
						gotNames[i] = r.Name
					}
				}
				require.Equal(t, c.wantNames, gotNames)
			})
		}
	})

	t.Run("windows ps1 scripts are supported", func(t *testing.T) {
		res, _, err := ds.GetHostScriptDetails(ctx, 42, nil, fleet.ListOptions{}, "windows")
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res, 1)
		require.Equal(t, "script-6.ps1", res[0].Name)
	})

	t.Run("can check if pending host script results exist", func(t *testing.T) {
		insertResults(t, 42, scripts[2], now.Add(-2*time.Minute), "execution-3-4", nil)
		r, err := ds.IsExecutionPendingForHost(ctx, 42, scripts[2].ID)
		require.NoError(t, err)
		require.True(t, r)
	})

	t.Run("script deletion cancels pending script runs", func(t *testing.T) {
		insertResults(t, 43, scripts[3], now.Add(-2*time.Minute), "execution-4-4", nil)
		pending, err := ds.ListPendingHostScriptExecutions(ctx, 43, false)
		require.NoError(t, err)
		require.Len(t, pending, 1)

		err = ds.DeleteScript(ctx, scripts[3].ID)
		require.NoError(t, err)

		pending, err = ds.ListPendingHostScriptExecutions(ctx, 43, false)
		require.NoError(t, err)
		require.Len(t, pending, 0)
	})
}

func testBatchSetScripts(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	applyAndExpect := func(newSet []*fleet.Script, tmID *uint, want []*fleet.Script) map[string]uint {
		responseFromSet, err := ds.BatchSetScripts(ctx, tmID, newSet)
		require.NoError(t, err)

		if tmID == nil {
			tmID = ptr.Uint(0)
		}
		got, _, err := ds.ListScripts(ctx, tmID, fleet.ListOptions{})
		require.NoError(t, err)

		// compare only the fields we care about
		fromGetByScriptName := make(map[string]uint)
		fromSetByScriptName := make(map[string]uint)
		for _, gotScript := range responseFromSet {
			fromSetByScriptName[gotScript.Name] = gotScript.ID
		}
		for _, gotScript := range got {
			fromGetByScriptName[gotScript.Name] = gotScript.ID
			if gotScript.TeamID != nil && *gotScript.TeamID == 0 {
				gotScript.TeamID = nil
			}

			require.Equal(t, fromGetByScriptName[gotScript.Name], gotScript.ID)
			gotScript.ID = 0
			gotScript.CreatedAt = time.Time{}
			gotScript.UpdatedAt = time.Time{}
		}
		// order is not guaranteed
		require.ElementsMatch(t, want, got)

		return fromGetByScriptName
	}

	// apply empty set for no-team
	applyAndExpect(nil, nil, nil)

	// create a team
	tm1, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_tm1"})
	require.NoError(t, err)

	// apply single script set for tm1
	sTm1 := applyAndExpect([]*fleet.Script{
		{Name: "N1", ScriptContents: "C1"},
	}, ptr.Uint(tm1.ID), []*fleet.Script{
		{Name: "N1", TeamID: ptr.Uint(tm1.ID)},
	})
	n1WithTeamID := sTm1["N1"]

	teamPolicy, err := ds.NewTeamPolicy(ctx, tm1.ID, nil, fleet.PolicyPayload{
		Name:     "Team One Policy",
		Query:    "SELECT 1",
		Platform: "darwin",
		ScriptID: &n1WithTeamID,
	})
	require.NoError(t, err)

	// apply single script set for no-team
	sNoTm := applyAndExpect([]*fleet.Script{
		{Name: "N1", ScriptContents: "C1"},
	}, nil, []*fleet.Script{
		{Name: "N1", TeamID: nil},
	})
	n1WithNoTeamId := sNoTm["N1"]

	noTeamPolicy, err := ds.NewTeamPolicy(ctx, fleet.PolicyNoTeamID, nil, fleet.PolicyPayload{
		Name:     "No Team Policy",
		Query:    "SELECT 1",
		Platform: "darwin",
		ScriptID: &n1WithNoTeamId,
	})
	require.NoError(t, err)

	// apply new script set for tm1
	sTm1b := applyAndExpect([]*fleet.Script{
		{Name: "N1", ScriptContents: "C1"},
		{Name: "N2", ScriptContents: "C2"},
	}, ptr.Uint(tm1.ID), []*fleet.Script{
		{Name: "N1", TeamID: ptr.Uint(tm1.ID)},
		{Name: "N2", TeamID: ptr.Uint(tm1.ID)},
	})
	// name for N1-I1 is unchanged
	require.Equal(t, sTm1["I1"], sTm1b["I1"])

	// policy still has script associated
	teamPolicy, err = ds.Policy(ctx, teamPolicy.ID)
	require.NoError(t, err)
	require.Equal(t, n1WithTeamID, *teamPolicy.ScriptID)

	// apply edited (by contents only) script set for no-team
	sNoTmb := applyAndExpect([]*fleet.Script{
		{Name: "N1", ScriptContents: "C1-changed"},
	}, nil, []*fleet.Script{
		{Name: "N1", TeamID: nil},
	})
	require.Equal(t, sNoTm["I1"], sNoTmb["I1"])

	// policy still has script associated
	noTeamPolicy, err = ds.Policy(ctx, noTeamPolicy.ID)
	require.NoError(t, err)
	require.Equal(t, n1WithNoTeamId, *noTeamPolicy.ScriptID)

	// apply edited script (by content only), unchanged script and new
	// script for tm1
	sTm1c := applyAndExpect([]*fleet.Script{
		{Name: "N1", ScriptContents: "C1-updated"}, // content updated
		{Name: "N2", ScriptContents: "C2"},         // unchanged
		{Name: "N3", ScriptContents: "C3"},         // new
	}, ptr.Uint(tm1.ID), []*fleet.Script{
		{Name: "N1", TeamID: ptr.Uint(tm1.ID)}, // content updated
		{Name: "N2", TeamID: ptr.Uint(tm1.ID)}, // unchanged
		{Name: "N3", TeamID: ptr.Uint(tm1.ID)}, // new
	})
	// name for N1-I1 is unchanged
	require.Equal(t, sTm1b["I1"], sTm1c["I1"])
	// identifier for N2-I2 is unchanged
	require.Equal(t, sTm1b["I2"], sTm1c["I2"])

	// policy still has script associated
	teamPolicy, err = ds.Policy(ctx, teamPolicy.ID)
	require.NoError(t, err)
	require.Equal(t, n1WithTeamID, *teamPolicy.ScriptID)

	// add pending scripts on team and no-team and confirm they're shown as pending
	_, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:   44,
		ScriptID: &n1WithTeamID,
	})
	require.NoError(t, err)
	_, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:   45,
		ScriptID: &n1WithNoTeamId,
	})
	require.NoError(t, err)

	pending, err := ds.ListPendingHostScriptExecutions(ctx, 44, false)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 45, false)
	require.NoError(t, err)
	require.Len(t, pending, 1)

	// clear scripts for tm1
	applyAndExpect(nil, ptr.Uint(1), nil)

	// policy on team should not have script assigned
	teamPolicy, err = ds.Policy(ctx, teamPolicy.ID)
	require.NoError(t, err)
	require.Nil(t, teamPolicy.ScriptID)

	// no-team policy still has script associated
	noTeamPolicy, err = ds.Policy(ctx, noTeamPolicy.ID)
	require.NoError(t, err)
	require.Equal(t, n1WithNoTeamId, *noTeamPolicy.ScriptID)

	// team script should no longer be pending, no-team script should still be pending
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 44, false)
	require.NoError(t, err)
	require.Len(t, pending, 0)
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 45, false)
	require.NoError(t, err)
	require.Len(t, pending, 1)

	// apply only new scripts to no-team
	applyAndExpect([]*fleet.Script{
		{Name: "N4", ScriptContents: "C4"},
		{Name: "N5", ScriptContents: "C5"},
	}, nil, []*fleet.Script{
		{Name: "N4", TeamID: nil},
		{Name: "N5", TeamID: nil},
	})

	// policy on team should not have script assigned
	teamPolicy, err = ds.Policy(ctx, teamPolicy.ID)
	require.NoError(t, err)
	require.Nil(t, teamPolicy.ScriptID)

	// no-team policy should not have script associated
	noTeamPolicy, err = ds.Policy(ctx, noTeamPolicy.ID)
	require.NoError(t, err)
	require.Nil(t, noTeamPolicy.ScriptID)

	// no-team script should no longer be pending
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 45, false)
	require.NoError(t, err)
	require.Len(t, pending, 0)
}

func testLockHostViaScript(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	// no script saved yet
	pending, err := ds.ListPendingHostScriptExecutions(ctx, 1, false)
	require.NoError(t, err)
	require.Empty(t, pending)

	user := test.NewUser(t, ds, "Bob", "bob@example.com", true)

	windowsHostID := uint(1)

	script := "lock"

	err = ds.LockHostViaScript(ctx, &fleet.HostScriptRequestPayload{
		HostID:         windowsHostID,
		ScriptContents: script,
		UserID:         &user.ID,
		SyncRequest:    false,
	}, "windows")

	require.NoError(t, err)

	// verify that we have created entries in host_mdm_actions and host_script_results
	status, err := ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: windowsHostID, Platform: "windows", UUID: "uuid"})
	require.NoError(t, err)
	require.Equal(t, "windows", status.HostFleetPlatform)
	require.NotNil(t, status.LockScript)
	assert.Nil(t, status.UnlockScript)

	s := status.LockScript
	require.Equal(t, script, s.ScriptContents)
	require.Equal(t, windowsHostID, s.HostID)
	require.False(t, s.SyncRequest)
	require.Equal(t, &user.ID, s.UserID)

	require.True(t, status.IsPendingLock())

	// simulate a successful result for the lock script execution
	_, action, err := ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      s.HostID,
		ExecutionID: s.ExecutionID,
		ExitCode:    0,
	})
	require.NoError(t, err)
	assert.Equal(t, "lock_ref", action)

	status, err = ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: windowsHostID, Platform: "windows", UUID: "uuid"})
	require.NoError(t, err)
	require.True(t, status.IsLocked())
	require.False(t, status.IsPendingLock())
	require.False(t, status.IsUnlocked())
}

func testUnlockHostViaScript(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	// no script saved yet
	pending, err := ds.ListPendingHostScriptExecutions(ctx, 1, false)
	require.NoError(t, err)
	require.Empty(t, pending)

	user := test.NewUser(t, ds, "Bob", "bob@example.com", true)

	hostID := uint(1)

	script := "unlock"

	err = ds.UnlockHostViaScript(ctx, &fleet.HostScriptRequestPayload{
		HostID:         hostID,
		ScriptContents: script,
		UserID:         &user.ID,
		SyncRequest:    false,
	}, "windows")

	require.NoError(t, err)

	// verify that we have created entries in host_mdm_actions and host_script_results
	status, err := ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: hostID, Platform: "windows", UUID: "uuid"})
	require.NoError(t, err)
	require.Equal(t, "windows", status.HostFleetPlatform)
	require.NotNil(t, status.UnlockScript)

	s := status.UnlockScript
	require.Equal(t, script, s.ScriptContents)
	require.Equal(t, hostID, s.HostID)
	require.False(t, s.SyncRequest)
	require.Equal(t, &user.ID, s.UserID)

	require.True(t, status.IsPendingUnlock())

	// simulate a successful result for the unlock script execution
	_, action, err := ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      s.HostID,
		ExecutionID: s.ExecutionID,
		ExitCode:    0,
	})
	require.NoError(t, err)
	assert.Equal(t, "unlock_ref", action)

	status, err = ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: hostID, Platform: "windows", UUID: "uuid"})
	require.NoError(t, err)
	require.True(t, status.IsUnlocked())
	require.False(t, status.IsPendingUnlock())
	require.False(t, status.IsLocked())
}

func testLockUnlockWipeViaScripts(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user := test.NewUser(t, ds, "Bob", "bob@example.com", true)

	for i, platform := range []string{"windows", "linux"} {
		hostID := uint(i + 1) //nolint:gosec // dismiss G115

		t.Run(platform, func(t *testing.T) {
			status, err := ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: hostID, Platform: platform, UUID: "uuid"})
			require.NoError(t, err)

			// default state
			checkLockWipeState(t, status, true, false, false, false, false, false)

			// record a request to lock the host
			err = ds.LockHostViaScript(ctx, &fleet.HostScriptRequestPayload{
				HostID:         hostID,
				ScriptContents: "lock",
				UserID:         &user.ID,
				SyncRequest:    false,
			}, platform)
			require.NoError(t, err)

			status, err = ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: hostID, Platform: platform, UUID: "uuid"})
			require.NoError(t, err)
			checkLockWipeState(t, status, true, false, false, false, true, false)

			// simulate a successful result for the lock script execution
			_, action, err := ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
				HostID:      hostID,
				ExecutionID: status.LockScript.ExecutionID,
				ExitCode:    0,
			})
			require.NoError(t, err)
			assert.Equal(t, "lock_ref", action)

			status, err = ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: hostID, Platform: platform, UUID: "uuid"})
			require.NoError(t, err)
			checkLockWipeState(t, status, false, true, false, false, false, false)

			// record a request to unlock the host
			err = ds.UnlockHostViaScript(ctx, &fleet.HostScriptRequestPayload{
				HostID:         hostID,
				ScriptContents: "unlock",
				UserID:         &user.ID,
				SyncRequest:    false,
			}, platform)
			require.NoError(t, err)

			status, err = ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: hostID, Platform: platform, UUID: "uuid"})
			require.NoError(t, err)
			checkLockWipeState(t, status, false, true, false, true, false, false)

			// simulate a failed result for the unlock script execution
			_, action, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
				HostID:      hostID,
				ExecutionID: status.UnlockScript.ExecutionID,
				ExitCode:    -1,
			})
			require.NoError(t, err)
			assert.Equal(t, "unlock_ref", action)

			// still locked
			status, err = ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: hostID, Platform: platform, UUID: "uuid"})
			require.NoError(t, err)
			checkLockWipeState(t, status, false, true, false, false, false, false)

			// record another request to unlock the host
			err = ds.UnlockHostViaScript(ctx, &fleet.HostScriptRequestPayload{
				HostID:         hostID,
				ScriptContents: "unlock",
				UserID:         &user.ID,
				SyncRequest:    false,
			}, platform)
			require.NoError(t, err)

			status, err = ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: hostID, Platform: platform, UUID: "uuid"})
			require.NoError(t, err)
			checkLockWipeState(t, status, false, true, false, true, false, false)

			// this time simulate a successful result for the unlock script execution
			_, action, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
				HostID:      hostID,
				ExecutionID: status.UnlockScript.ExecutionID,
				ExitCode:    0,
			})
			require.NoError(t, err)
			assert.Equal(t, "unlock_ref", action)

			// host is now unlocked
			status, err = ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: hostID, Platform: platform, UUID: "uuid"})
			require.NoError(t, err)
			checkLockWipeState(t, status, true, false, false, false, false, false)

			// record another request to lock the host
			err = ds.LockHostViaScript(ctx, &fleet.HostScriptRequestPayload{
				HostID:         hostID,
				ScriptContents: "lock",
				UserID:         &user.ID,
				SyncRequest:    false,
			}, platform)
			require.NoError(t, err)

			status, err = ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: hostID, Platform: platform, UUID: "uuid"})
			require.NoError(t, err)
			checkLockWipeState(t, status, true, false, false, false, true, false)

			// simulate a failed result for the lock script execution
			_, action, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
				HostID:      hostID,
				ExecutionID: status.LockScript.ExecutionID,
				ExitCode:    2,
			})
			require.NoError(t, err)
			assert.Equal(t, "lock_ref", action)

			status, err = ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: hostID, Platform: platform, UUID: "uuid"})
			require.NoError(t, err)
			checkLockWipeState(t, status, true, false, false, false, false, false)

			switch platform {
			case "windows":
				// need a real MDM-enrolled host for MDM commands
				h, err := ds.NewHost(ctx, &fleet.Host{
					Hostname:      "test-host-windows",
					OsqueryHostID: ptr.String("osquery-windows"),
					NodeKey:       ptr.String("nodekey-windows"),
					UUID:          "test-uuid-windows",
					Platform:      "windows",
				})
				require.NoError(t, err)
				windowsEnroll(t, ds, h)

				// record a request to wipe the host
				wipeCmdUUID := uuid.NewString()
				wipeCmd := &fleet.MDMWindowsCommand{
					CommandUUID:  wipeCmdUUID,
					RawCommand:   []byte(`<Exec></Exec>`),
					TargetLocURI: "./Device/Vendor/MSFT/RemoteWipe/doWipeProtected",
				}
				err = ds.WipeHostViaWindowsMDM(ctx, h, wipeCmd)
				require.NoError(t, err)

				status, err = ds.GetHostLockWipeStatus(ctx, h)
				require.NoError(t, err)
				checkLockWipeState(t, status, true, false, false, false, false, true)

				// TODO: we don't seem to have an easy way to simulate a Windows MDM
				// protocol response, and there are lots of validations happening so we
				// can't just send a simple XML. Will test the rest via integration
				// tests.

			case "linux":
				// record a request to wipe the host
				err = ds.WipeHostViaScript(ctx, &fleet.HostScriptRequestPayload{
					HostID:         hostID,
					ScriptContents: "wipe",
					UserID:         &user.ID,
					SyncRequest:    false,
				}, platform)
				require.NoError(t, err)

				status, err = ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: hostID, Platform: platform, UUID: "uuid"})
				require.NoError(t, err)
				checkLockWipeState(t, status, true, false, false, false, false, true)

				// simulate a failed result for the wipe script execution
				_, action, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
					HostID:      hostID,
					ExecutionID: status.WipeScript.ExecutionID,
					ExitCode:    1,
				})
				require.NoError(t, err)
				assert.Equal(t, "wipe_ref", action)

				status, err = ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: hostID, Platform: platform, UUID: "uuid"})
				require.NoError(t, err)
				checkLockWipeState(t, status, true, false, false, false, false, false)

				// record another request to wipe the host
				err = ds.WipeHostViaScript(ctx, &fleet.HostScriptRequestPayload{
					HostID:         hostID,
					ScriptContents: "wipe2",
					UserID:         &user.ID,
					SyncRequest:    false,
				}, platform)
				require.NoError(t, err)

				status, err = ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: hostID, Platform: platform, UUID: "uuid"})
				require.NoError(t, err)
				checkLockWipeState(t, status, true, false, false, false, false, true)

				// simulate a successful result for the wipe script execution
				_, action, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
					HostID:      hostID,
					ExecutionID: status.WipeScript.ExecutionID,
					ExitCode:    0,
				})
				require.NoError(t, err)
				assert.Equal(t, "wipe_ref", action)

				status, err = ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: hostID, Platform: platform, UUID: "uuid"})
				require.NoError(t, err)
				checkLockWipeState(t, status, false, false, true, false, false, false)
			}
		})
	}
}

func testLockUnlockManually(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	twoDaysAgo := time.Now().AddDate(0, 0, -2).UTC()
	today := time.Now().UTC()
	err := ds.UnlockHostManually(ctx, 1, "darwin", twoDaysAgo)
	require.NoError(t, err)

	status, err := ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: 1, Platform: "darwin", UUID: "uuid"})
	require.NoError(t, err)
	require.False(t, status.UnlockRequestedAt.IsZero())
	require.WithinDuration(t, twoDaysAgo, status.UnlockRequestedAt, 1*time.Second)

	// if the unlock request already exists, it is not overwritten by subsequent
	// requests
	err = ds.UnlockHostManually(ctx, 1, "darwin", today)
	require.NoError(t, err)
	status, err = ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: 1, Platform: "darwin", UUID: "uuid"})
	require.NoError(t, err)
	require.False(t, status.UnlockRequestedAt.IsZero())
	require.WithinDuration(t, twoDaysAgo, status.UnlockRequestedAt, 1*time.Second)

	// but for a new host, it will set it properly, even if that host already has a
	// host_mdm_actions entry
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO host_mdm_actions (host_id) VALUES (2)")
		return err
	})
	err = ds.UnlockHostManually(ctx, 2, "darwin", today)
	require.NoError(t, err)
	status, err = ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: 2, Platform: "darwin", UUID: "uuid"})
	require.NoError(t, err)
	require.False(t, status.UnlockRequestedAt.IsZero())
	require.WithinDuration(t, today, status.UnlockRequestedAt, 1*time.Second)
}

func checkLockWipeState(t *testing.T, status *fleet.HostLockWipeStatus, unlocked, locked, wiped, pendingUnlock, pendingLock, pendingWipe bool) {
	require.Equal(t, unlocked, status.IsUnlocked(), "unlocked")
	require.Equal(t, locked, status.IsLocked(), "locked")
	require.Equal(t, wiped, status.IsWiped(), "wiped")
	require.Equal(t, pendingLock, status.IsPendingLock(), "pending lock")
	require.Equal(t, pendingUnlock, status.IsPendingUnlock(), "pending unlock")
	require.Equal(t, pendingWipe, status.IsPendingWipe(), "pending wipe")
}

type scriptContents struct {
	ID       uint   `db:"id"`
	Checksum string `db:"md5_checksum"`
}

func testInsertScriptContents(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	contents := `echo foobar;`
	res, err := insertScriptContents(ctx, ds.writer(ctx), contents)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	require.Equal(t, int64(1), id)
	expectedCS := md5ChecksumScriptContent(contents)

	// insert same contents again, verify that the checksum and ID stayed the same
	res, err = insertScriptContents(ctx, ds.writer(ctx), contents)
	require.NoError(t, err)
	id, _ = res.LastInsertId()
	require.Equal(t, int64(1), id)

	stmt := `SELECT id, HEX(md5_checksum) as md5_checksum FROM script_contents WHERE id = ?`

	var sc []scriptContents
	err = sqlx.SelectContext(ctx, ds.reader(ctx),
		&sc, stmt,
		id,
	)
	require.NoError(t, err)

	require.Len(t, sc, 1)
	require.EqualValues(t, id, sc[0].ID)
	require.Equal(t, expectedCS, sc[0].Checksum)
}

func testCleanupUnusedScriptContents(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create a saved script
	s := &fleet.Script{
		ScriptContents: "echo foobar",
	}
	s, err := ds.NewScript(ctx, s)
	require.NoError(t, err)

	user1 := test.NewUser(t, ds, "Bob", "bob@example.com", true)

	// create a sync script execution
	res, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{ScriptContents: "echo something_else", SyncRequest: true})
	require.NoError(t, err)

	// create a software install that references scripts
	tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	swi, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "install-script",
		UninstallScript:   "uninstall-script",
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "post-install-script",
		InstallerFile:     tfr1,
		StorageID:         "storage1",
		Filename:          "file1",
		Title:             "file1",
		Version:           "1.0",
		Source:            "apps",
		UserID:            user1.ID,
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// delete our saved script without ever executing it
	require.NoError(t, ds.DeleteScript(ctx, s.ID))

	// validate that script contents still exist
	var sc []scriptContents
	stmt := `SELECT id, HEX(md5_checksum) as md5_checksum FROM script_contents`
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &sc, stmt)
	require.NoError(t, err)
	require.Len(t, sc, 5)

	// this should only remove the script_contents of the saved script, since the sync script is
	// still "in use" by the script execution
	require.NoError(t, ds.CleanupUnusedScriptContents(ctx))

	sc = []scriptContents{}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &sc, stmt)
	require.NoError(t, err)
	require.Len(t, sc, 4)
	require.ElementsMatch(t, []string{
		md5ChecksumScriptContent(res.ScriptContents),
		md5ChecksumScriptContent("install-script"),
		md5ChecksumScriptContent("post-install-script"),
		md5ChecksumScriptContent("uninstall-script"),
	}, []string{
		sc[0].Checksum,
		sc[1].Checksum,
		sc[2].Checksum,
		sc[3].Checksum,
	})

	// remove the software installer from the DB
	err = ds.DeleteSoftwareInstaller(ctx, swi)
	require.NoError(t, err)

	require.NoError(t, ds.CleanupUnusedScriptContents(ctx))

	// validate that script contents still exist
	sc = []scriptContents{}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &sc, stmt)
	require.NoError(t, err)
	require.Len(t, sc, 1)
	require.Equal(t, md5ChecksumScriptContent(res.ScriptContents), sc[0].Checksum)

	// create a software install without a post-install script
	tfr2, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	swi, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		PreInstallQuery: "SELECT 1",
		InstallScript:   "install-script",
		UninstallScript: "uninstall-script",
		InstallerFile:   tfr2,
		StorageID:       "storage1",
		Filename:        "file1",
		Title:           "file1",
		Version:         "1.0",
		Source:          "apps",
		UserID:          user1.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// run the cleanup function
	require.NoError(t, ds.CleanupUnusedScriptContents(ctx))
	sc = []scriptContents{}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &sc, stmt)
	require.NoError(t, err)
	require.Len(t, sc, 3)

	// remove the software installer from the DB
	err = ds.DeleteSoftwareInstaller(ctx, swi)
	require.NoError(t, err)
	require.NoError(t, ds.CleanupUnusedScriptContents(ctx))

	// validate that script contents still exist
	sc = []scriptContents{}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &sc, stmt)
	require.NoError(t, err)
	require.Len(t, sc, 1)
	require.Equal(t, md5ChecksumScriptContent(res.ScriptContents), sc[0].Checksum)
}

func testGetAnyScriptContents(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	contents := `echo foobar;`
	res, err := insertScriptContents(ctx, ds.writer(ctx), contents)
	require.NoError(t, err)
	id, _ := res.LastInsertId()

	result, err := ds.GetAnyScriptContents(ctx, uint(id)) //nolint:gosec // dismiss G115
	require.NoError(t, err)
	require.Equal(t, contents, string(result))
}

func testDeleteScriptsAssignedToPolicy(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	script, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "script.sh",
		TeamID:         &team1.ID,
		ScriptContents: "hello world",
	})
	require.NoError(t, err)

	p1, err := ds.NewTeamPolicy(ctx, team1.ID, nil, fleet.PolicyPayload{
		Name:     "p1",
		Query:    "SELECT 1;",
		ScriptID: &script.ID,
	})
	require.NoError(t, err)

	err = ds.DeleteScript(ctx, script.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, errDeleteScriptWithAssociatedPolicy)

	_, err = ds.DeleteTeamPolicies(ctx, team1.ID, []uint{p1.ID})
	require.NoError(t, err)

	err = ds.DeleteScript(ctx, script.ID)
	require.NoError(t, err)
}

func testDeletePendingHostScriptExecutionsForPolicy(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team1, _ := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})

	script1, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "script1.sh",
		TeamID:         &team1.ID,
		ScriptContents: "hello world",
	})
	require.NoError(t, err)
	script2, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "script2.sh",
		TeamID:         &team1.ID,
		ScriptContents: "hello world",
	})
	require.NoError(t, err)

	p1, err := ds.NewTeamPolicy(ctx, team1.ID, nil, fleet.PolicyPayload{
		Name:     "p1",
		Query:    "SELECT 1;",
		ScriptID: &script1.ID,
	})
	require.NoError(t, err)

	p2, err := ds.NewTeamPolicy(ctx, team1.ID, nil, fleet.PolicyPayload{
		Name:     "p2",
		Query:    "SELECT 2;",
		ScriptID: &script2.ID,
	})
	require.NoError(t, err)

	// pending host script execution for correct policy
	_, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         1,
		ScriptContents: "echo",
		UserID:         &user.ID,
		PolicyID:       &p1.ID,
		SyncRequest:    true,
		ScriptID:       &script1.ID,
	})
	require.NoError(t, err)

	pending, err := ds.ListPendingHostScriptExecutions(ctx, 1, false)
	require.NoError(t, err)
	require.Equal(t, 1, len(pending))

	err = ds.deletePendingHostScriptExecutionsForPolicy(ctx, &team1.ID, p1.ID)
	require.NoError(t, err)

	pending, err = ds.ListPendingHostScriptExecutions(ctx, 1, false)
	require.NoError(t, err)
	require.Equal(t, 0, len(pending))

	// test pending host script execution for incorrect policy
	hsr, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         1,
		ScriptContents: "echo",
		UserID:         &user.ID,
		PolicyID:       &p2.ID,
		SyncRequest:    true,
		ScriptID:       &script2.ID,
	})
	require.NoError(t, err)

	pending, err = ds.ListPendingHostScriptExecutions(ctx, 1, false)
	require.NoError(t, err)
	require.Equal(t, 1, len(pending))

	err = ds.deletePendingHostScriptExecutionsForPolicy(ctx, &team1.ID, p1.ID)
	require.NoError(t, err)

	pending, err = ds.ListPendingHostScriptExecutions(ctx, 1, false)
	require.NoError(t, err)
	require.Equal(t, 1, len(pending))

	// test not pending host script execution for correct policy
	scriptExecution, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         1,
		ScriptContents: "echo",
		UserID:         &user.ID,
		PolicyID:       &p1.ID,
		SyncRequest:    true,
		ScriptID:       &script1.ID,
	})
	require.NoError(t, err)

	// record a result for the previous pending script
	_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      1,
		ExecutionID: hsr.ExecutionID,
		Output:      "foo",
		ExitCode:    0,
	})
	require.NoError(t, err)

	// record a failed result for the current pending script
	_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      1,
		ExecutionID: scriptExecution.ExecutionID,
		Output:      "foo",
		ExitCode:    1,
	})
	require.NoError(t, err)

	err = ds.deletePendingHostScriptExecutionsForPolicy(ctx, &team1.ID, p1.ID)
	require.NoError(t, err)

	var count int
	err = sqlx.GetContext(
		ctx,
		ds.reader(ctx),
		&count,
		"SELECT count(1) FROM host_script_results WHERE id = ?",
		scriptExecution.ID,
	)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func testUpdateScriptContents(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	originalScript, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "script1",
		ScriptContents: "hello world",
	})
	require.NoError(t, err)

	originalContents, err := ds.GetScriptContents(ctx, originalScript.ScriptContentID)
	require.NoError(t, err)
	require.Equal(t, "hello world", string(originalContents))

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE scripts SET updated_at = ? WHERE id = ?", time.Now().Add(-2*time.Minute), originalScript.ID)
		if err != nil {
			return err
		}
		return nil
	})

	// Make sure updated_at was changed correctly, but the script is the same
	oldScript, err := ds.Script(ctx, originalScript.ID)
	require.Equal(t, originalScript.ScriptContentID, oldScript.ScriptContentID)
	require.NoError(t, err)
	require.NotEqual(t, originalScript.UpdatedAt, oldScript.UpdatedAt)

	// Modify the script
	updatedScript, err := ds.UpdateScriptContents(ctx, originalScript.ID, "updated script")
	require.NoError(t, err)
	require.Equal(t, originalScript.ID, updatedScript.ID)
	require.Equal(t, originalScript.ScriptContentID, updatedScript.ScriptContentID)

	updatedContents, err := ds.GetScriptContents(ctx, originalScript.ScriptContentID)
	require.NoError(t, err)
	require.Equal(t, "updated script", string(updatedContents))
	require.NotEqual(t, oldScript.UpdatedAt, updatedScript.UpdatedAt)
}

func testUpdateDeletingUpcomingScriptExecutions(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user := test.NewUser(t, ds, "User", "user@example.com", true)
	host1 := test.NewHost(t, ds, "host1", "10.0.0.1", "host1Key", "host1UUID", time.Now())
	host2 := test.NewHost(t, ds, "host2", "10.0.0.2", "host2Key", "host2UUID", time.Now())

	script1, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "script1",
		ScriptContents: "contents1",
	})
	require.NoError(t, err)

	script2, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "script2",
		ScriptContents: "contents2",
	})
	require.NoError(t, err)

	script3, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "script3",
		ScriptContents: "contents3",
	})
	require.NoError(t, err)

	// Queue script executions
	_, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:   host1.ID,
		ScriptID: &script1.ID,
		UserID:   &user.ID,
	})
	require.NoError(t, err)

	_, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:   host1.ID,
		ScriptID: &script2.ID,
		UserID:   &user.ID,
	})
	require.NoError(t, err)

	_, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:   host2.ID,
		ScriptID: &script2.ID,
		UserID:   &user.ID,
	})
	require.NoError(t, err)

	_, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:   host2.ID,
		ScriptID: &script1.ID,
		UserID:   &user.ID,
	})
	require.NoError(t, err)

	upcoming1, err := ds.listUpcomingHostScriptExecutions(ctx, host1.ID, false, false)
	require.NoError(t, err)
	require.Len(t, upcoming1, 2)

	upcoming2, err := ds.listUpcomingHostScriptExecutions(ctx, host2.ID, false, false)
	require.NoError(t, err)
	require.Len(t, upcoming2, 2)

	// Updating the "pending/upcoming" script will cancel the activity and stop it from running
	_, err = ds.UpdateScriptContents(ctx, script1.ID, "new contents1")
	require.NoError(t, err)

	upcoming1, err = ds.listUpcomingHostScriptExecutions(ctx, host1.ID, false, false)
	require.NoError(t, err)
	require.Len(t, upcoming1, 1)
	require.Equal(t, script2.ID, *upcoming1[0].ScriptID)

	upcoming2, err = ds.listUpcomingHostScriptExecutions(ctx, host2.ID, false, false)
	require.NoError(t, err)
	require.Len(t, upcoming2, 1)
	require.Equal(t, script2.ID, *upcoming2[0].ScriptID)

	// Updating a script with no upcoming activities shouldn't affect anything
	_, err = ds.UpdateScriptContents(ctx, script3.ID, "new contents")
	require.NoError(t, err)

	upcoming1, err = ds.listUpcomingHostScriptExecutions(ctx, host1.ID, false, false)
	require.NoError(t, err)
	require.Len(t, upcoming1, 1)
	require.Equal(t, script2.ID, *upcoming1[0].ScriptID)

	upcoming2, err = ds.listUpcomingHostScriptExecutions(ctx, host2.ID, false, false)
	require.NoError(t, err)
	require.Len(t, upcoming2, 1)
	require.Equal(t, script2.ID, *upcoming2[0].ScriptID)
}

func testBatchExecute(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	user := test.NewUser(t, ds, "user1", "user@example.com", true)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	hostNoScripts := test.NewHost(t, ds, "hostNoScripts", "10.0.0.1", "hostnoscripts", "hostnoscriptsuuid", time.Now())
	hostWindows := test.NewHost(t, ds, "hostWin", "10.0.0.2", "hostWinKey", "hostWinUuid", time.Now(), test.WithPlatform("windows"))
	host1 := test.NewHost(t, ds, "host1", "10.0.0.3", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "10.0.0.4", "host2key", "host2uuid", time.Now())
	host3 := test.NewHost(t, ds, "host3", "10.0.0.4", "host3key", "host3uuid", time.Now())
	hostTeam1 := test.NewHost(t, ds, "hostTeam1", "10.0.0.5", "hostTeam1key", "hostTeam1uuid", time.Now(), test.WithTeamID(team1.ID))

	test.SetOrbitEnrollment(t, hostWindows, ds)
	test.SetOrbitEnrollment(t, host1, ds)
	test.SetOrbitEnrollment(t, host2, ds)
	test.SetOrbitEnrollment(t, host3, ds)
	test.SetOrbitEnrollment(t, hostTeam1, ds)

	script, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "script1.sh",
		ScriptContents: "echo hi",
	})
	require.NoError(t, err)

	// Hosts all have to be on the same team as the script
	execID, err := ds.BatchExecuteScript(ctx, &user.ID, script.ID, []uint{hostNoScripts.ID, hostTeam1.ID})
	require.Empty(t, execID)
	require.ErrorContains(t, err, "same team")

	// Actual good execution
	execID, err = ds.BatchExecuteScript(ctx, &user.ID, script.ID, []uint{hostNoScripts.ID, hostWindows.ID, host1.ID, host2.ID, host3.ID})
	require.NoError(t, err)

	summary, err := ds.BatchExecuteSummary(ctx, execID)
	require.NoError(t, err)
	require.Equal(t, script.ID, summary.ScriptID)
	require.Equal(t, script.Name, summary.ScriptName)
	require.Equal(t, uint(0), *summary.TeamID)
	require.NotNil(t, summary.CreatedAt)

	// The summary should have two pending hosts and two errored ones, because
	// the script is not compatible with the hostNoScripts and hostWindows.
	require.Equal(t, summary.NumPending, uint(3))
	require.Equal(t, summary.NumErrored, uint(2))
	require.Equal(t, summary.NumRan, uint(0))
	require.Equal(t, summary.NumCanceled, uint(0))
	// Host 1 should have an upcoming execution
	host1Upcoming, err := ds.listUpcomingHostScriptExecutions(ctx, host1.ID, false, false)
	require.NoError(t, err)
	require.Len(t, host1Upcoming, 1)
	require.Equal(t, &summary.ScriptID, host1Upcoming[0].ScriptID)
	// Host 2 should have an upcoming execution
	host2Upcoming, err := ds.listUpcomingHostScriptExecutions(ctx, host2.ID, false, false)
	require.NoError(t, err)
	require.Len(t, host2Upcoming, 1)
	require.Equal(t, &summary.ScriptID, host2Upcoming[0].ScriptID)
	// Host 3 should have an upcoming execution
	host3Upcoming, err := ds.listUpcomingHostScriptExecutions(ctx, host3.ID, false, false)
	require.NoError(t, err)
	require.Len(t, host3Upcoming, 1)
	require.Equal(t, &summary.ScriptID, host3Upcoming[0].ScriptID)
	// Host Windows should not have an upcoming execution
	hostWindowsUpcoming, err := ds.listUpcomingHostScriptExecutions(ctx, hostWindows.ID, false, false)
	require.NoError(t, err)
	require.Len(t, hostWindowsUpcoming, 0)
	// Host No Scripts should not have an upcoming execution
	hostNoScriptsUpcoming, err := ds.listUpcomingHostScriptExecutions(ctx, hostNoScripts.ID, false, false)
	require.NoError(t, err)
	require.Len(t, hostNoScriptsUpcoming, 0)
	// Host Windows should have an error in its `batch_activity_host_results` row
	var exec_error string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		db := q.(*sqlx.DB)
		err := db.Get(&exec_error, "SELECT error FROM batch_activity_host_results WHERE host_id = ? AND batch_execution_id = ?", hostWindows.ID, execID)
		require.NoError(t, err)
		return nil
	})
	require.Equal(t, fleet.BatchExecuteIncompatiblePlatform, exec_error)
	// Host No Scripts should have an error in its `batch_activity_host_results` row
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		db := q.(*sqlx.DB)
		err := db.Get(&exec_error, "SELECT error FROM batch_activity_host_results WHERE host_id = ? AND batch_execution_id = ?", hostNoScripts.ID, execID)
		require.NoError(t, err)
		return nil
	})
	require.Equal(t, fleet.BatchExecuteIncompatibleFleetd, exec_error)

	// Set host 1 to have a successful script result
	_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      host1.ID,
		ExecutionID: host1Upcoming[0].ExecutionID,
		Output:      "foo",
		ExitCode:    0,
	})
	require.NoError(t, err)
	// Get the summary again
	summary, err = ds.BatchExecuteSummary(ctx, execID)
	require.NoError(t, err)
	// The summary should have one pending host, one run host and two errored ones.
	require.Equal(t, summary.NumPending, uint(2))
	require.Equal(t, summary.NumErrored, uint(2))
	require.Equal(t, summary.NumRan, uint(1))
	require.Equal(t, summary.NumCanceled, uint(0))

	// Set host 1 to have a failed script result
	_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      host2.ID,
		ExecutionID: host2Upcoming[0].ExecutionID,
		Output:      "bar",
		ExitCode:    1,
	})
	require.NoError(t, err)

	// Get the summary again
	summary, err = ds.BatchExecuteSummary(ctx, execID)
	require.NoError(t, err)
	// The summary should have one pending host, one run host and two errored ones.
	require.Equal(t, summary.NumPending, uint(1))
	require.Equal(t, summary.NumErrored, uint(3))
	require.Equal(t, summary.NumRan, uint(1))
	require.Equal(t, summary.NumCanceled, uint(0))

	// Cancel the execution
	_, err = ds.CancelHostUpcomingActivity(ctx, host3.ID, host3Upcoming[0].ExecutionID)
	require.NoError(t, err)
	// Get the summary again
	summary, err = ds.BatchExecuteSummary(ctx, execID)
	require.NoError(t, err)
	// The summary should have no pending hosts, one run host, three errored ones and one canceled.
	require.Equal(t, summary.NumPending, uint(0))
	require.Equal(t, summary.NumErrored, uint(3))
	require.Equal(t, summary.NumRan, uint(1))
	require.Equal(t, summary.NumCanceled, uint(1))
}

func testBatchExecuteWithStatus(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	user := test.NewUser(t, ds, "user1", "user@example.com", true)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	hostNoScripts := test.NewHost(t, ds, "hostNoScripts", "10.0.0.1", "hostnoscripts", "hostnoscriptsuuid", time.Now())
	hostWindows := test.NewHost(t, ds, "hostWin", "10.0.0.2", "hostWinKey", "hostWinUuid", time.Now(), test.WithPlatform("windows"))
	host1 := test.NewHost(t, ds, "host1", "10.0.0.3", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "10.0.0.4", "host2key", "host2uuid", time.Now())
	host3 := test.NewHost(t, ds, "host3", "10.0.0.4", "host3key", "host3uuid", time.Now())
	hostTeam1 := test.NewHost(t, ds, "hostTeam1", "10.0.0.5", "hostTeam1key", "hostTeam1uuid", time.Now(), test.WithTeamID(team1.ID))

	test.SetOrbitEnrollment(t, hostWindows, ds)
	test.SetOrbitEnrollment(t, host1, ds)
	test.SetOrbitEnrollment(t, host2, ds)
	test.SetOrbitEnrollment(t, host3, ds)
	test.SetOrbitEnrollment(t, hostTeam1, ds)

	script, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "script1.sh",
		ScriptContents: "echo hi",
	})
	require.NoError(t, err)

	// Hosts all have to be on the same team as the script
	execID, err := ds.BatchExecuteScript(ctx, &user.ID, script.ID, []uint{hostNoScripts.ID, hostTeam1.ID})
	require.Empty(t, execID)
	require.ErrorContains(t, err, "same team")

	// Actual good execution
	execID, err = ds.BatchExecuteScript(ctx, &user.ID, script.ID, []uint{hostNoScripts.ID, hostWindows.ID, host1.ID, host2.ID, host3.ID})
	require.NoError(t, err)

	// Update the batch to have a pending status
	// TODO -- remove this when status is set automatically
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "UPDATE batch_activities SET status = 'scheduled' WHERE execution_id = ?", execID)
		return err
	})

	summaryList, err := ds.BatchExecuteStatus(ctx, fleet.BatchExecutionStatusFilter{
		ExecutionID: &execID,
	})
	require.NoError(t, err)
	require.Len(t, *summaryList, 1)
	summary := (*summaryList)[0]
	require.Equal(t, script.ID, summary.ScriptID)
	require.Equal(t, script.Name, summary.ScriptName)
	require.Equal(t, uint(0), *summary.TeamID)
	require.NotNil(t, summary.CreatedAt)

	// The summary should have two pending hosts and two errored ones, because
	// the script is not compatible with the hostNoScripts and hostWindows.
	require.Equal(t, summary.NumTargeted, uint(5))
	require.Equal(t, summary.NumPending, uint(3))
	require.Equal(t, summary.NumIncompatible, uint(2))
	require.Equal(t, summary.NumErrored, uint(0))
	require.Equal(t, summary.NumRan, uint(0))
	require.Equal(t, summary.NumCanceled, uint(0))
	// Host 1 should have an upcoming execution
	host1Upcoming, err := ds.listUpcomingHostScriptExecutions(ctx, host1.ID, false, false)
	require.NoError(t, err)
	require.Len(t, host1Upcoming, 1)
	require.Equal(t, &summary.ScriptID, host1Upcoming[0].ScriptID)
	// Host 2 should have an upcoming execution
	host2Upcoming, err := ds.listUpcomingHostScriptExecutions(ctx, host2.ID, false, false)
	require.NoError(t, err)
	require.Len(t, host2Upcoming, 1)
	require.Equal(t, &summary.ScriptID, host2Upcoming[0].ScriptID)
	// Host 3 should have an upcoming execution
	host3Upcoming, err := ds.listUpcomingHostScriptExecutions(ctx, host3.ID, false, false)
	require.NoError(t, err)
	require.Len(t, host3Upcoming, 1)
	require.Equal(t, &summary.ScriptID, host3Upcoming[0].ScriptID)
	// Host Windows should not have an upcoming execution
	hostWindowsUpcoming, err := ds.listUpcomingHostScriptExecutions(ctx, hostWindows.ID, false, false)
	require.NoError(t, err)
	require.Len(t, hostWindowsUpcoming, 0)
	// Host No Scripts should not have an upcoming execution
	hostNoScriptsUpcoming, err := ds.listUpcomingHostScriptExecutions(ctx, hostNoScripts.ID, false, false)
	require.NoError(t, err)
	require.Len(t, hostNoScriptsUpcoming, 0)
	// Host Windows should have an error in its `batch_activity_host_results` row
	var exec_error string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		db := q.(*sqlx.DB)
		err := db.Get(&exec_error, "SELECT error FROM batch_activity_host_results WHERE host_id = ? AND batch_execution_id = ?", hostWindows.ID, execID)
		require.NoError(t, err)
		return nil
	})
	require.Equal(t, fleet.BatchExecuteIncompatiblePlatform, exec_error)
	// Host No Scripts should have an error in its `batch_activity_host_results` row
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		db := q.(*sqlx.DB)
		err := db.Get(&exec_error, "SELECT error FROM batch_activity_host_results WHERE host_id = ? AND batch_execution_id = ?", hostNoScripts.ID, execID)
		require.NoError(t, err)
		return nil
	})
	require.Equal(t, fleet.BatchExecuteIncompatibleFleetd, exec_error)

	// Set host 1 to have a successful script result
	_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      host1.ID,
		ExecutionID: host1Upcoming[0].ExecutionID,
		Output:      "foo",
		ExitCode:    0,
	})
	require.NoError(t, err)

	// Get the summary again
	summaryList, err = ds.BatchExecuteStatus(ctx, fleet.BatchExecutionStatusFilter{
		ExecutionID: &execID,
	})
	require.NoError(t, err)
	require.Len(t, *summaryList, 1)
	summary = (*summaryList)[0]
	// The summary should have one pending host, one run host and two errored ones.
	require.Equal(t, summary.NumTargeted, uint(5))
	require.Equal(t, summary.NumPending, uint(2))
	require.Equal(t, summary.NumIncompatible, uint(2))
	require.Equal(t, summary.NumErrored, uint(0))
	require.Equal(t, summary.NumRan, uint(1))
	require.Equal(t, summary.NumCanceled, uint(0))

	// Set host 1 to have a failed script result
	_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      host2.ID,
		ExecutionID: host2Upcoming[0].ExecutionID,
		Output:      "bar",
		ExitCode:    1,
	})
	require.NoError(t, err)

	// Get the summary again
	summaryList, err = ds.BatchExecuteStatus(ctx, fleet.BatchExecutionStatusFilter{
		ExecutionID: &execID,
	})
	require.NoError(t, err)
	require.Len(t, *summaryList, 1)
	summary = (*summaryList)[0] // The summary should have one pending host, one run host and two errored ones.
	require.Equal(t, summary.NumTargeted, uint(5))
	require.Equal(t, summary.NumPending, uint(1))
	require.Equal(t, summary.NumIncompatible, uint(2))
	require.Equal(t, summary.NumErrored, uint(1))
	require.Equal(t, summary.NumRan, uint(1))
	require.Equal(t, summary.NumCanceled, uint(0))

	// Cancel the execution
	_, err = ds.CancelHostUpcomingActivity(ctx, host3.ID, host3Upcoming[0].ExecutionID)
	require.NoError(t, err)
	// Get the summary again
	summaryList, err = ds.BatchExecuteStatus(ctx, fleet.BatchExecutionStatusFilter{
		ExecutionID: &execID,
	})
	require.NoError(t, err)
	require.Len(t, *summaryList, 1)
	summary = (*summaryList)[0]
	// The summary should have no pending hosts, one run host, three errored ones and one canceled.
	require.Equal(t, summary.NumPending, uint(0))
	require.Equal(t, summary.NumIncompatible, uint(2))
	require.Equal(t, summary.NumErrored, uint(1))
	require.Equal(t, summary.NumRan, uint(1))
	require.Equal(t, summary.NumCanceled, uint(1))

	// The summary should be returned when filtering by status "scheduled".
	summaryList, err = ds.BatchExecuteStatus(ctx, fleet.BatchExecutionStatusFilter{
		Status: ptr.String("scheduled"),
	})
	require.NoError(t, err)
	require.Len(t, *summaryList, 1)
	summary = (*summaryList)[0]
	// The summary should have no pending hosts, one run host, three errored ones and one canceled.
	require.Equal(t, summary.NumPending, uint(0))
	require.Equal(t, summary.NumIncompatible, uint(2))
	require.Equal(t, summary.NumErrored, uint(1))
	require.Equal(t, summary.NumRan, uint(1))
	require.Equal(t, summary.NumCanceled, uint(1))
}

func testDeleteScriptActivatesNextActivity(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	u := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	// create a couple of scripts
	scriptA, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "a",
		ScriptContents: "echo 'a'",
	})
	require.NoError(t, err)
	scriptB, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "b",
		ScriptContents: "echo 'b'",
	})
	require.NoError(t, err)

	// create some hosts
	hosts := make([]*fleet.Host, 4)
	for i := range hosts {
		host, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         ptr.String(fmt.Sprint(i)),
			UUID:            fmt.Sprint(i),
			Hostname:        fmt.Sprintf("%d-foo.local", i),
			PrimaryIP:       fmt.Sprintf("192.168.1.%d", i),
			PrimaryMac:      fmt.Sprintf("30-65-EC-6F-C4-5%d", i),
		})
		require.NoError(t, err)
		hosts[i] = host
	}

	// enqueue scripts executions:
	// * hosts[0]: a, b
	// * hosts[1]: a, b
	// * hosts[2]: b, a
	// * hosts[3]: b
	execHost0ScriptA, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:      hosts[0].ID,
		ScriptID:    &scriptA.ID,
		UserID:      &u.ID,
		SyncRequest: true,
	})
	require.NoError(t, err)
	execHost0ScriptB, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:      hosts[0].ID,
		ScriptID:    &scriptB.ID,
		UserID:      &u.ID,
		SyncRequest: true,
	})
	require.NoError(t, err)
	execHost1ScriptA, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:      hosts[1].ID,
		ScriptID:    &scriptA.ID,
		UserID:      &u.ID,
		SyncRequest: true,
	})
	require.NoError(t, err)
	execHost1ScriptB, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:      hosts[1].ID,
		ScriptID:    &scriptB.ID,
		UserID:      &u.ID,
		SyncRequest: true,
	})
	require.NoError(t, err)
	execHost2ScriptB, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:      hosts[2].ID,
		ScriptID:    &scriptB.ID,
		UserID:      &u.ID,
		SyncRequest: true,
	})
	require.NoError(t, err)
	execHost2ScriptA, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:      hosts[2].ID,
		ScriptID:    &scriptA.ID,
		UserID:      &u.ID,
		SyncRequest: true,
	})
	require.NoError(t, err)
	execHost3ScriptB, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:      hosts[3].ID,
		ScriptID:    &scriptB.ID,
		UserID:      &u.ID,
		SyncRequest: true,
	})
	require.NoError(t, err)

	checkUpcomingActivities(t, ds, hosts[0], execHost0ScriptA.ExecutionID, execHost0ScriptB.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[1], execHost1ScriptA.ExecutionID, execHost1ScriptB.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[2], execHost2ScriptB.ExecutionID, execHost2ScriptA.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[3], execHost3ScriptB.ExecutionID)

	// delete scriptA removes pending upcoming activity and activates next activity
	err = ds.DeleteScript(ctx, scriptA.ID)
	require.NoError(t, err)

	checkUpcomingActivities(t, ds, hosts[0], execHost0ScriptB.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[1], execHost1ScriptB.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[2], execHost2ScriptB.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[3], execHost3ScriptB.ExecutionID)
}

func testBatchSetScriptActivatesNextActivity(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// batch-set some scripts
	scripts, err := ds.BatchSetScripts(ctx, nil, []*fleet.Script{
		{Name: "A", ScriptContents: "C1"},
		{Name: "B", ScriptContents: "C2"},
		{Name: "C", ScriptContents: "C3"},
	})
	require.NoError(t, err)

	// index scripts by name
	scriptByName := make(map[string]uint)
	for _, s := range scripts {
		scriptByName[s.Name] = s.ID
	}

	// create some hosts
	hosts := make([]*fleet.Host, 4)
	for i := range hosts {
		host, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         ptr.String(fmt.Sprint(i)),
			UUID:            fmt.Sprint(i),
			Hostname:        fmt.Sprintf("%d-foo.local", i),
			PrimaryIP:       fmt.Sprintf("192.168.1.%d", i),
			PrimaryMac:      fmt.Sprintf("30-65-EC-6F-C4-5%d", i),
		})
		require.NoError(t, err)
		hosts[i] = host
	}

	// enqeue script executions:
	// * hosts[0]: A, C, A, B
	// * hosts[1]: B, B, C
	// * hosts[2]: C, A
	// * hosts[3]: A, B, C
	execHost0ScriptA, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: hosts[0].ID, ScriptID: ptr.Uint(scriptByName["A"]), SyncRequest: true, ScriptContents: "C1",
	})
	require.NoError(t, err)
	execHost0ScriptC, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: hosts[0].ID, ScriptID: ptr.Uint(scriptByName["C"]), SyncRequest: true, ScriptContents: "C3",
	})
	require.NoError(t, err)
	execHost0ScriptA2, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: hosts[0].ID, ScriptID: ptr.Uint(scriptByName["A"]), SyncRequest: true, ScriptContents: "C1",
	})
	require.NoError(t, err)
	execHost0ScriptB, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: hosts[0].ID, ScriptID: ptr.Uint(scriptByName["B"]), SyncRequest: true, ScriptContents: "C2",
	})
	require.NoError(t, err)
	execHost1ScriptB, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: hosts[1].ID, ScriptID: ptr.Uint(scriptByName["B"]), SyncRequest: true, ScriptContents: "C2",
	})
	require.NoError(t, err)
	execHost1ScriptB2, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: hosts[1].ID, ScriptID: ptr.Uint(scriptByName["B"]), SyncRequest: true, ScriptContents: "C2",
	})
	require.NoError(t, err)
	execHost1ScriptC, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: hosts[1].ID, ScriptID: ptr.Uint(scriptByName["C"]), SyncRequest: true, ScriptContents: "C3",
	})
	require.NoError(t, err)
	execHost2ScriptC, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: hosts[2].ID, ScriptID: ptr.Uint(scriptByName["C"]), SyncRequest: true, ScriptContents: "C3",
	})
	require.NoError(t, err)
	execHost2ScriptA, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: hosts[2].ID, ScriptID: ptr.Uint(scriptByName["A"]), SyncRequest: true, ScriptContents: "C1",
	})
	require.NoError(t, err)
	execHost3ScriptA, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: hosts[3].ID, ScriptID: ptr.Uint(scriptByName["A"]), SyncRequest: true, ScriptContents: "C1",
	})
	require.NoError(t, err)
	execHost3ScriptB, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: hosts[3].ID, ScriptID: ptr.Uint(scriptByName["B"]), SyncRequest: true, ScriptContents: "C2",
	})
	require.NoError(t, err)
	execHost3ScriptC, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: hosts[3].ID, ScriptID: ptr.Uint(scriptByName["C"]), SyncRequest: true, ScriptContents: "C3",
	})
	require.NoError(t, err)

	checkUpcomingActivities(t, ds, hosts[0], execHost0ScriptA.ExecutionID, execHost0ScriptC.ExecutionID, execHost0ScriptA2.ExecutionID, execHost0ScriptB.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[1], execHost1ScriptB.ExecutionID, execHost1ScriptB2.ExecutionID, execHost1ScriptC.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[2], execHost2ScriptC.ExecutionID, execHost2ScriptA.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[3], execHost3ScriptA.ExecutionID, execHost3ScriptB.ExecutionID, execHost3ScriptC.ExecutionID)

	// no change
	_, err = ds.BatchSetScripts(ctx, nil, []*fleet.Script{
		{Name: "A", ScriptContents: "C1"},
		{Name: "B", ScriptContents: "C2"},
		{Name: "C", ScriptContents: "C3"},
	})
	require.NoError(t, err)

	checkUpcomingActivities(t, ds, hosts[0], execHost0ScriptA.ExecutionID, execHost0ScriptC.ExecutionID, execHost0ScriptA2.ExecutionID, execHost0ScriptB.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[1], execHost1ScriptB.ExecutionID, execHost1ScriptB2.ExecutionID, execHost1ScriptC.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[2], execHost2ScriptC.ExecutionID, execHost2ScriptA.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[3], execHost3ScriptA.ExecutionID, execHost3ScriptB.ExecutionID, execHost3ScriptC.ExecutionID)

	// batch-set removes A, updates B and creates D, cancelling any pending A and B executions
	_, err = ds.BatchSetScripts(ctx, nil, []*fleet.Script{
		{Name: "B", ScriptContents: "C2updated"},
		{Name: "C", ScriptContents: "C3"},
		{Name: "D", ScriptContents: "C4"},
	})
	require.NoError(t, err)

	checkUpcomingActivities(t, ds, hosts[0], execHost0ScriptC.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[1], execHost1ScriptC.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[2], execHost2ScriptC.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[3], execHost3ScriptC.ExecutionID)

	// batch-set remove all
	_, err = ds.BatchSetScripts(ctx, nil, []*fleet.Script{})
	require.NoError(t, err)

	checkUpcomingActivities(t, ds, hosts[0])
	checkUpcomingActivities(t, ds, hosts[1])
	checkUpcomingActivities(t, ds, hosts[2])
	checkUpcomingActivities(t, ds, hosts[3])
}
