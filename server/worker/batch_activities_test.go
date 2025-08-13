package worker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

func TestBatchActivitiesRun(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)

	var recvExecID string
	ds.RunScheduledBatchActivityFunc = func(ctx context.Context, executionID string) error {
		recvExecID = executionID
		return nil
	}

	batchWorker := &BatchScripts{
		Datastore: ds,
	}

	batchWorker.Run(ctx, json.RawMessage(`{"execution_id": "abc"}`))

	require.Equal(t, "abc", recvExecID)
}
