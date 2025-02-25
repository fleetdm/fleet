package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	kitlog "github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBMigrationsVPPToken(t *testing.T) {
	ctx := context.Background()

	ds := mysql.CreateMySQLDS(t)
	// call TruncateTables immediately as a DB migration may have created jobs
	mysql.TruncateTables(t, ds)

	nopLog := kitlog.NewNopLogger()
	// use this to debug/verify details of calls
	// nopLog := kitlog.NewJSONLogger(os.Stdout)

	// create and register the worker
	processor := &DBMigration{
		Datastore: ds,
		Log:       nopLog,
	}
	w := NewWorker(ds, nopLog)
	w.Register(processor)

	// create the migrated token and enqueue the job
	expDate := time.Date(2024, 8, 27, 0, 0, 0, 0, time.UTC)
	tok, err := test.CreateVPPTokenEncodedAfterMigration(expDate, "test-org", "test-loc")
	require.NoError(t, err)
	encTok, err := mysql.EncryptWithPrivateKey(t, ds, tok)
	require.NoError(t, err)

	const insVPP = `
INSERT INTO vpp_tokens
	(
		organization_name,
		location,
		renew_at,
		token
	)
VALUES
	('', '', DATE('2000-01-01'), ?)
`

	const insJob = `
INSERT INTO jobs (
		name,
		args,
		state,
		error,
		not_before,
		created_at,
		updated_at
)
VALUES (?, ?, ?, '', ?, ?, ?)
`
	mysql.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, insVPP, encTok)
		if err != nil {
			return err
		}

		argsJSON, err := json.Marshal(dbMigrationArgs{Task: DBMigrateVPPTokenTask})
		if err != nil {
			return fmt.Errorf("failed to JSON marshal the job arguments: %w", err)
		}
		ts := time.Date(2024, 8, 26, 0, 0, 0, 0, time.UTC)
		if _, err := q.ExecContext(ctx, insJob, dbMigrationJobName, argsJSON, fleet.JobStateQueued, ts, ts, ts); err != nil {
			return err
		}
		return nil
	})

	// run the worker, should mark the job as done
	err = w.ProcessJobs(ctx)
	require.NoError(t, err)

	// nothing more to run
	jobs, err := ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute)) // look in the future to catch any delayed job
	require.NoError(t, err)
	if !assert.Empty(t, jobs) {
		t.Logf(">>> %#+v", jobs[0])
	}

	// token should've been updated
	vppTok, err := ds.GetVPPTokenByLocation(ctx, "test-loc")
	require.NoError(t, err)
	require.Equal(t, "test-org", vppTok.OrgName)
	require.Equal(t, "test-loc", vppTok.Location)
	require.Equal(t, expDate, vppTok.RenewDate)
	require.Contains(t, string(tok), `"token":"`+vppTok.Token+`"`) // the DB-stored token is the "token" JSON field in the raw tok
	require.NotNil(t, vppTok.Teams)
	require.Len(t, vppTok.Teams, 0)

	// empty-location token should not exist anymore
	_, err = ds.GetVPPTokenByLocation(ctx, "")
	require.Error(t, err)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	// enqueue a DB migration job with an unknown task
	mysql.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		argsJSON, err := json.Marshal(dbMigrationArgs{Task: DBMigrationTask("no-such-task")})
		if err != nil {
			return fmt.Errorf("failed to JSON marshal the job arguments: %w", err)
		}
		ts := time.Date(2024, 8, 26, 0, 0, 0, 0, time.UTC)
		if _, err := q.ExecContext(ctx, insJob, dbMigrationJobName, argsJSON, fleet.JobStateQueued, ts, ts, ts); err != nil {
			return err
		}
		return nil
	})

	// run the worker, will fail but still queued for a retry
	err = w.ProcessJobs(ctx)
	require.NoError(t, err)

	jobs, err = ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute)) // look in the future to catch any delayed job
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.Equal(t, fleet.JobStateQueued, jobs[0].State)
	require.Equal(t, 1, jobs[0].Retries)
	require.Contains(t, jobs[0].Error, "unknown task: no-such-task")
}
