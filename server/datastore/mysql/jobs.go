package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) NewJob(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
	query := `
INSERT INTO jobs (
    name,
    args,
    state,
    retries,
    error,
    not_before
)
VALUES (?, ?, ?, ?, ?, COALESCE(?, NOW()))
`
	var notBefore *time.Time
	if !job.NotBefore.IsZero() {
		notBefore = &job.NotBefore
	}
	result, err := ds.writer(ctx).ExecContext(ctx, query, job.Name, job.Args, job.State, job.Retries, job.Error, notBefore)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	job.ID = uint(id) //nolint:gosec // dismiss G115

	return job, nil
}

func (ds *Datastore) GetQueuedJobs(ctx context.Context, maxNumJobs int, now time.Time) ([]*fleet.Job, error) {
	return ds.GetFilteredQueuedJobs(ctx, maxNumJobs, now, nil)
}

func (ds *Datastore) GetFilteredQueuedJobs(ctx context.Context, maxNumJobs int, now time.Time, jobNames []string) ([]*fleet.Job, error) {
	query := `
SELECT
    id, created_at, updated_at, name, args, state, retries, error, not_before
FROM
    jobs
WHERE
    state = ? AND
    not_before <= ?
	%s
ORDER BY
    updated_at ASC
LIMIT ?
`

	if now.IsZero() {
		now = time.Now().UTC()
	}

	args := []interface{}{fleet.JobStateQueued, now}

	// Add job name filter if needed
	var nameClause string
	if len(jobNames) > 0 {
		clause, nameArgs, err := sqlx.In("AND name IN (?)", jobNames)
		if err != nil {
			return nil, err
		}
		nameClause = clause
		args = append(args, nameArgs...)
	}

	query = fmt.Sprintf(query, nameClause)
	args = append(args, maxNumJobs)
	var jobs []*fleet.Job
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &jobs, query, args...)
	if err != nil {
		return nil, err
	}

	return jobs, nil
}

func (ds *Datastore) updateJob(ctx context.Context, tx sqlx.ExtContext, id uint, job *fleet.Job) (*fleet.Job, error) {
	query := `
UPDATE jobs
SET
    state = ?,
    retries = ?,
    error = ?,
    not_before = COALESCE(?, NOW())
WHERE
    id = ?
`
	var notBefore *time.Time
	if !job.NotBefore.IsZero() {
		notBefore = &job.NotBefore
	}
	_, err := tx.ExecContext(ctx, query, job.State, job.Retries, job.Error, notBefore, id)
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (ds *Datastore) UpdateJob(ctx context.Context, id uint, job *fleet.Job) (*fleet.Job, error) {
	return ds.updateJob(ctx, ds.writer(ctx), id, job)
}

func (ds *Datastore) CleanupWorkerJobs(ctx context.Context, failedSince, completedSince time.Duration) (int64, error) {
	// using not_before instead of created_at/updated_at to be able to use the
	// existing index, and the difference between those timestamps will be
	// minimal (max 5 retries for failed jobs, with a few hours difference).
	const stmt = `
	DELETE FROM
		jobs
	WHERE
		(state = ? AND not_before < ?) OR
		(state = ? AND not_before < ?)
`

	now := time.Now().UTC()
	failedBefore := now.Add(-failedSince)
	completedBefore := now.Add(-completedSince)

	res, err := ds.writer(ctx).ExecContext(ctx, stmt,
		fleet.JobStateFailure, failedBefore,
		fleet.JobStateSuccess, completedBefore)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "cleanup worker jobs")
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// HasQueuedJobWithArgs determines whether a job with the given name and args
// is currently queued. Can be used to deduplicate jobs that would be redundant
// (or harmful) to enqueue multiple times.
//
// Uses the primary to avoid replica lag, since one of the primary use cases
// is handling thrash from rapid user action such as quickly disabling
// and re-enabling a chart dataset multiple times.
func (ds *Datastore) HasQueuedJobWithArgs(ctx context.Context, name string, args json.RawMessage) (bool, error) {
	const query = `
SELECT 1 FROM jobs
WHERE name = ? AND state = ? AND args = CAST(? AS JSON)
LIMIT 1`
	var found int
	ctx = ctxdb.RequirePrimary(ctx, true)
	err := sqlx.GetContext(ctx, ds.reader(ctx), &found, query, name, fleet.JobStateQueued, []byte(args))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, ctxerr.Wrap(ctx, err, "check queued job with args")
	}
	return true, nil
}

func (ds *Datastore) GetJob(ctx context.Context, jobID uint) (*fleet.Job, error) {
	query := `
		SELECT
			id,
			created_at,
			updated_at,
			name,
			args,
			state,
			retries,
			error,
			not_before
		FROM
			jobs
		WHERE
			id = ?`

	job := &fleet.Job{}

	if err := sqlx.GetContext(ctx, ds.reader(ctx), job, query, jobID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get job")
	}

	return job, nil
}
