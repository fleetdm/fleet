// Package automationtest provides cross-package test helpers for code that
// interacts with the policy-automation recording lifecycle (the methods on
// fleet.Datastore that maintain the policy_runs and
// policy_automation_executions tables).
package automationtest

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/google/uuid"
)

func StubNoopRecording(ds *mock.Store) {
	ds.RecordPolicyTransitionsFunc = func(ctx context.Context, hostID uint, policyResults map[uint]*bool, newFailing, newPassing []uint) (map[uint]uint, error) {
		out := make(map[uint]uint, len(newFailing))
		for i, pid := range newFailing {
			out[pid] = uint(i + 1)
		}
		return out, nil
	}
	ds.GetFailingPolicyRunsFunc = func(ctx context.Context, policyIDs, hostIDs []uint) ([]fleet.PolicyRunRef, error) {
		out := make([]fleet.PolicyRunRef, 0, len(policyIDs)*len(hostIDs))
		idx := uint(1)
		for _, pid := range policyIDs {
			for _, hid := range hostIDs {
				out = append(out, fleet.PolicyRunRef{PolicyID: pid, HostID: hid, RunID: idx})
				idx++
			}
		}
		return out, nil
	}
	ds.CreatePolicyAutomationExecutionsFunc = func(ctx context.Context, typ fleet.PolicyAutomationType, executions []fleet.PolicyRunRef) (uuid.UUID, error) {
		if len(executions) == 0 {
			return uuid.Nil, nil
		}
		return uuid.New(), nil
	}
	ds.UpdatePolicyAutomationExecutionsFunc = func(ctx context.Context, batchID uuid.UUID, outcomeErr error) error {
		return nil
	}
}
