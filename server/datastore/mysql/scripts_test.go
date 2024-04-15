package mysql

import (
	"context"
	_ "embed"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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
		{"ListScripts", testListScripts},
		{"GetHostScriptDetails", testGetHostScriptDetails},
		{"BatchSetScripts", testBatchSetScripts},
		{"TestLockHostViaScript", testLockHostViaScript},
		{"TestUnlockHostViaScript", testUnlockHostViaScript},
		{"TestLockUnlockWipeViaScripts", testLockUnlockWipeViaScripts},
		{"TestLockUnlockManually", testLockUnlockManually},
		{"TestInsertScriptContents", testInsertScriptContents},
		{"TestCleanupUnusedScriptContents", testCleanupUnusedScriptContents},
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
	pending, err := ds.ListPendingHostScriptExecutions(ctx, 1)
	require.NoError(t, err)
	require.Empty(t, pending)

	_, err = ds.GetHostScriptExecutionResult(ctx, "abc")
	require.Error(t, err)
	var nfe *notFoundError
	require.ErrorAs(t, err, &nfe)

	// create a createdScript execution request (with a user)
	u := test.NewUser(t, ds, "Bob", "bob@example.com", true)
	createdScript, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         1,
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
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 1)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, createdScript.ID, pending[0].ID)

	// record a result for this execution
	_, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      1,
		ExecutionID: createdScript.ExecutionID,
		Output:      "foo",
		Runtime:     2,
		ExitCode:    0,
	})
	require.NoError(t, err)

	// record a duplicate result for this execution, will be ignored
	hsr, err := ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      1,
		ExecutionID: createdScript.ExecutionID,
		Output:      "foobarbaz",
		Runtime:     22,
		ExitCode:    1,
	})
	require.NoError(t, err)
	require.Nil(t, hsr)

	// it is not pending anymore
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 1)
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

	// create another script execution request (null user id this time)
	createdScript, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         1,
		ScriptContents: "echo2",
	})
	require.NoError(t, err)
	require.NotZero(t, createdScript.ID)
	require.NotEmpty(t, createdScript.ExecutionID)
	require.Nil(t, createdScript.UserID)
	require.False(t, createdScript.SyncRequest)

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

	_, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
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

	// create an async execution request
	createdScript, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         1,
		ScriptContents: "echo 3",
		UserID:         &u.ID,
		SyncRequest:    false,
	})
	require.NoError(t, err)
	require.NotZero(t, createdScript.ID)
	require.NotEmpty(t, createdScript.ExecutionID)
	require.Equal(t, uint(1), createdScript.HostID)
	require.NotEmpty(t, createdScript.ExecutionID)
	require.Equal(t, "echo 3", createdScript.ScriptContents)
	require.Nil(t, createdScript.ExitCode)
	require.Empty(t, createdScript.Output)
	require.NotNil(t, createdScript.UserID)
	require.Equal(t, u.ID, *createdScript.UserID)
	require.False(t, createdScript.SyncRequest)

	// the script execution is now listed as pending for this host
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 1)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, createdScript.ID, pending[0].ID)

	// modify the timestamp of the script to simulate an script that has
	// been pending for a long time
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "UPDATE host_script_results SET created_at = ? WHERE id = ?", time.Now().Add(-24*time.Hour), createdScript.ID)
		return err
	})

	// the script execution still shows as pending
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 1)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, createdScript.ID, pending[0].ID)

	// modify the script to be a sync script that has
	// been pending for a long time
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "UPDATE host_script_results SET sync_request = 1 WHERE id = ?", createdScript.ID)
		return err
	})

	// the script is not pending anymore
	pending, err = ds.ListPendingHostScriptExecutions(ctx, 1)
	require.NoError(t, err)
	require.Empty(t, pending, 0)

	// check that scripts with large unsigned error codes get
	// converted to signed error codes
	createdUnsignedScript, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         1,
		ScriptContents: "echo",
		UserID:         &u.ID,
		SyncRequest:    true,
	})
	require.NoError(t, err)

	unsignedScriptResult, err := ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      1,
		ExecutionID: createdUnsignedScript.ExecutionID,
		Output:      "foo",
		Runtime:     1,
		ExitCode:    math.MaxUint32,
	})
	require.NoError(t, err)
	require.EqualValues(t, -1, *unsignedScriptResult.ExitCode)
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
		stmt := `
INSERT INTO
	host_script_results (%s host_id, created_at, execution_id, exit_code, output)
VALUES
	(%s ?,?,?,?,?)`

		args := []interface{}{}
		if script.ID == 0 {
			stmt = fmt.Sprintf(stmt, "", "")
		} else {
			stmt = fmt.Sprintf(stmt, "script_id,", "?,")
			args = append(args, script.ID)
		}
		args = append(args, hostID, createdAt, execID, exitCode, "")

		ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, stmt, args...)
			return err
		})
	}

	now := time.Now().UTC().Truncate(time.Second)

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

	// add some results for an ad-hoc, non-saved script, should not be included in results
	insertResults(t, 42, &fleet.Script{Name: "script-6", ScriptContents: "echo script-6"}, now.Add(-1*time.Minute), "execution-6-1", ptr.Int64(0))

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
				require.Nil(t, r.LastExecution)
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
		require.Len(t, r, 1)
	})
}

func testBatchSetScripts(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	applyAndExpect := func(newSet []*fleet.Script, tmID *uint, want []*fleet.Script) map[string]uint {
		err := ds.BatchSetScripts(ctx, tmID, newSet)
		require.NoError(t, err)

		if tmID == nil {
			tmID = ptr.Uint(0)
		}
		got, _, err := ds.ListScripts(ctx, tmID, fleet.ListOptions{})
		require.NoError(t, err)

		// compare only the fields we care about
		m := make(map[string]uint)
		for _, gotScript := range got {
			m[gotScript.Name] = gotScript.ID
			if gotScript.TeamID != nil && *gotScript.TeamID == 0 {
				gotScript.TeamID = nil
			}
			gotScript.ID = 0
			gotScript.CreatedAt = time.Time{}
			gotScript.UpdatedAt = time.Time{}
		}
		// order is not guaranteed
		require.ElementsMatch(t, want, got)

		return m
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

	// apply single script set for no-team
	sNoTm := applyAndExpect([]*fleet.Script{
		{Name: "N1", ScriptContents: "C1"},
	}, nil, []*fleet.Script{
		{Name: "N1", TeamID: nil},
	})

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

	// apply edited (by contents only) script set for no-team
	sNoTmb := applyAndExpect([]*fleet.Script{
		{Name: "N1", ScriptContents: "C1-changed"},
	}, nil, []*fleet.Script{
		{Name: "N1", TeamID: nil},
	})
	require.Equal(t, sNoTm["I1"], sNoTmb["I1"])

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

	// apply only new scripts to no-team
	applyAndExpect([]*fleet.Script{
		{Name: "N4", ScriptContents: "C4"},
		{Name: "N5", ScriptContents: "C5"},
	}, nil, []*fleet.Script{
		{Name: "N4", TeamID: nil},
		{Name: "N5", TeamID: nil},
	})

	// clear scripts for tm1
	applyAndExpect(nil, ptr.Uint(1), nil)
}

func testLockHostViaScript(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	// no script saved yet
	pending, err := ds.ListPendingHostScriptExecutions(ctx, 1)
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

	s := status.LockScript
	require.Equal(t, script, s.ScriptContents)
	require.Equal(t, windowsHostID, s.HostID)
	require.False(t, s.SyncRequest)
	require.Equal(t, &user.ID, s.UserID)

	require.True(t, status.IsPendingLock())

	// simulate a successful result for the lock script execution
	_, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      s.HostID,
		ExecutionID: s.ExecutionID,
		ExitCode:    0,
	})
	require.NoError(t, err)

	status, err = ds.GetHostLockWipeStatus(ctx, &fleet.Host{ID: windowsHostID, Platform: "windows", UUID: "uuid"})
	require.NoError(t, err)
	require.True(t, status.IsLocked())
	require.False(t, status.IsPendingLock())
	require.False(t, status.IsUnlocked())
}

func testUnlockHostViaScript(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	// no script saved yet
	pending, err := ds.ListPendingHostScriptExecutions(ctx, 1)
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
	_, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      s.HostID,
		ExecutionID: s.ExecutionID,
		ExitCode:    0,
	})
	require.NoError(t, err)

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
		hostID := uint(i + 1)

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
			_, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
				HostID:      hostID,
				ExecutionID: status.LockScript.ExecutionID,
				ExitCode:    0,
			})
			require.NoError(t, err)

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
			_, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
				HostID:      hostID,
				ExecutionID: status.UnlockScript.ExecutionID,
				ExitCode:    -1,
			})
			require.NoError(t, err)

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
			_, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
				HostID:      hostID,
				ExecutionID: status.UnlockScript.ExecutionID,
				ExitCode:    0,
			})
			require.NoError(t, err)

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
			_, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
				HostID:      hostID,
				ExecutionID: status.LockScript.ExecutionID,
				ExitCode:    2,
			})
			require.NoError(t, err)

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
				_, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
					HostID:      hostID,
					ExecutionID: status.WipeScript.ExecutionID,
					ExitCode:    1,
				})
				require.NoError(t, err)

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
				_, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
					HostID:      hostID,
					ExecutionID: status.WipeScript.ExecutionID,
					ExitCode:    0,
				})
				require.NoError(t, err)

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
	res, err := insertScriptContents(ctx, contents, ds.writer(ctx))
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	require.Equal(t, int64(1), id)
	expectedCS := md5ChecksumScriptContent(contents)

	// insert same contents again, verify that the checksum and ID stayed the same
	res, err = insertScriptContents(ctx, contents, ds.writer(ctx))
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
	require.Equal(t, uint(id), sc[0].ID)
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

	// create a sync script execution
	res, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{ScriptContents: "echo something_else", SyncRequest: true})
	require.NoError(t, err)

	// delete our saved script without ever executing it
	require.NoError(t, ds.DeleteScript(ctx, s.ID))

	// validate that script contents still exist
	var sc []scriptContents
	stmt := `SELECT id, HEX(md5_checksum) as md5_checksum FROM script_contents`
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &sc, stmt)
	require.NoError(t, err)
	require.Len(t, sc, 2)

	// this should only remove the script_contents of the saved script, since the sync script is
	// still "in use" by the script execution
	require.NoError(t, ds.CleanupUnusedScriptContents(ctx))

	sc = []scriptContents{}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &sc, stmt)
	require.NoError(t, err)
	require.Len(t, sc, 1)
	require.Equal(t, md5ChecksumScriptContent(res.ScriptContents), sc[0].Checksum)
}
