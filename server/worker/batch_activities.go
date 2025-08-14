package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
)

const BatchScriptsName = "batch_scripts"

type BatchScripts struct {
	Datastore fleet.Datastore
	Log       kitlog.Logger
}

func (b *BatchScripts) Name() string {
	return BatchScriptsName
}

func (b *BatchScripts) Run(ctx context.Context, jobArgs json.RawMessage) error {
	var args fleet.BatchActivityScriptJobArgs

	if err := json.Unmarshal(jobArgs, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal json")
	}

	activity, err := b.Datastore.GetBatchActivity(ctx, args.ExecutionID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "could not find batch activity")
	}

	// The job was already started, close the job
	if activity.Status != fleet.ScheduledBatchExecutionScheduled {
		return nil
	}

	// The activity was canceled, close the job
	if activity.Canceled {
		return nil
	}

	if err := b.Datastore.RunScheduledBatchActivity(ctx, args.ExecutionID); err != nil {
		return ctxerr.Wrap(ctx, err, "running scheduled batch script")
	}

	return nil
}

func QueueBatchScriptJob(ctx context.Context, ds fleet.Datastore, tx sqlx.ExtContext, notBefore time.Time, args fleet.BatchActivityScriptJobArgs) (*fleet.Job, error) {
	argBytes, err := json.Marshal(args)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshaling args")
	}

	job, err := ds.NewJobTx(ctx, tx, &fleet.Job{
		Name:      BatchScriptsName,
		Args:      (*json.RawMessage)(&argBytes),
		State:     fleet.JobStateQueued,
		NotBefore: notBefore.UTC(),
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "queueing job")
	}

	return job, nil
}
