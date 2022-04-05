package mysql

import (
	"context"

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
    error
)
VALUES (?, ?, ?, ?, ?)
`
	result, err := ds.writer.ExecContext(ctx, query, job.Name, job.Args, job.State, job.Retries, job.Error)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	job.ID = uint(id)

	return job, nil
}

func (ds *Datastore) GetQueuedJobs(ctx context.Context, maxNumJobs int) ([]*fleet.Job, error) {
	query := `
SELECT
    id, created_at, updated_at, name, args, state, retries, error
FROM
    jobs
WHERE
    state = ?
ORDER BY
    created_at asc
LIMIT ?
`

	var jobs []*fleet.Job
	err := sqlx.SelectContext(ctx, ds.reader, &jobs, query, fleet.JobStateQueued, maxNumJobs)
	if err != nil {
		return nil, err
	}

	return jobs, nil
}

func (ds *Datastore) UpdateJob(ctx context.Context, id uint, job *fleet.Job) (*fleet.Job, error) {
	query := `
UPDATE jobs
SET
    state = ?,
    retries = ?,
    error = ?
WHERE
    id = ?
`
	_, err := ds.writer.ExecContext(ctx, query, job.State, job.Retries, job.Error, job.ID)
	if err != nil {
		return nil, err
	}

	return job, nil
}
