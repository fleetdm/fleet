package worker

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
)

const batchScriptsName = "batch_scripts"

type BatchScripts struct {
	Datastore fleet.Datastore
	Log       kitlog.Logger
}

// Name returns the name of the job.
func (b *BatchScripts) Name() string {
	return batchScriptsName
}

func (b *BatchScripts) Run(ctx context.Context, jobArgs json.RawMessage) error {
	ds := b.Datastore
	var args fleet.BatchActivityScriptJobArgs

	if err := json.Unmarshal(jobArgs, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal json")
	}

	activity, err := ds.GetBatchActivity(ctx, args.ExecutionID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting batch activity")
	}

	if err := ds.RunScheduledBatchActivity(ctx, activity.BatchExecutionID); err != nil {
		return ctxerr.Wrap(ctx, err, "running scheduled batch script")
	}

	return nil
}
