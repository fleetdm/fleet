package worker

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
)

type BatchScripts struct {
	Datastore fleet.Datastore
	Log       kitlog.Logger
}

func (b *BatchScripts) Name() string {
	return fleet.BatchActivityScriptsJobName
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
