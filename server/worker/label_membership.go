package worker

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Persisted in the jobs table, so changing it orphans queued jobs.
const LabelMembershipCleanupJobName = "label_membership_cleanup"

type LabelMembershipCleanup struct {
	Datastore fleet.Datastore
}

func (l *LabelMembershipCleanup) Name() string { return LabelMembershipCleanupJobName }

func (l *LabelMembershipCleanup) Run(ctx context.Context, _ json.RawMessage) error {
	// The job can run before a replica has applied the label's deletion, and would then find nothing to clean up.
	ctx = ctxdb.RequirePrimary(ctx, true)
	if err := l.Datastore.CleanupOrphanedLabelMembership(ctx); err != nil {
		return ctxerr.Wrap(ctx, err, "cleanup orphaned label membership")
	}
	return nil
}

func QueueLabelMembershipCleanupJob(ctx context.Context, ds fleet.Datastore) error {
	if _, err := QueueJob(ctx, ds, LabelMembershipCleanupJobName, struct{}{}); err != nil {
		return ctxerr.Wrap(ctx, err, "queue label membership cleanup job")
	}
	return nil
}
