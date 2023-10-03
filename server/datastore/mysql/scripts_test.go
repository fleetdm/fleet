package mysql

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
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
		{"GetHostScriptDetailss", testGetHostScriptDetails},
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

	names := []string{"script-1", "script-2", "script-3", "script-4", "script-5"}
	for _, r := range append(names[1:], names[0]) {
		_, err := ds.NewScript(ctx, &fleet.Script{
			Name:           r,
			ScriptContents: "echo " + r,
		})
		require.NoError(t, err)
	}

	scripts, _, err := ds.ListScripts(ctx, nil, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, scripts, 5)

	insertResults := func(t *testing.T, hostID uint, script *fleet.Script, createdAt time.Time, execID string, exitCode *int64) {
		stmt := `
INSERT INTO 
	host_script_results (%s host_id, created_at, execution_id, exit_code, script_contents, output) 
VALUES 
	(%s ?,?,?,?,?,?)`

		args := []interface{}{}
		if script.ID == 0 {
			stmt = fmt.Sprintf(stmt, "", "")
		} else {
			stmt = fmt.Sprintf(stmt, "script_id,", "?,")
			args = append(args, script.ID)
		}
		args = append(args, hostID, createdAt, execID, exitCode, script.ScriptContents, "")

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
		res, _, err := ds.GetHostScriptDetails(ctx, 42, nil, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, res, 5)
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
			default:
				t.Errorf("unexpected script id: %d", r.ScriptID)
			}
		}
	})

	t.Run("empty slice returned if no scripts", func(t *testing.T) {
		res, _, err := ds.GetHostScriptDetails(ctx, 42, ptr.Uint(1), fleet.ListOptions{}) // team 1 has no scripts
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
				results, meta, err := ds.GetHostScriptDetails(ctx, 42, nil, c.opts)
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
}
