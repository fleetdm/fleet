package worker

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// Name of the Apple MDM job as registered in the worker. Note that although it
// is a single job, it can process a number of different-but-related tasks,
// identified by the Task field in the job's payload.
const appleMDMJobName = "apple_mdm"

type AppleMDMTask string

// List of supported tasks.
const (
	AppleMDMPostDEPEnrollmentTask AppleMDMTask = "post_dep_enrollment"
)

// AppleMDM is the job processor for the apple_mdm job.
type AppleMDM struct {
	Datastore fleet.Datastore
	Log       kitlog.Logger
	Commander *apple_mdm.MDMAppleCommander
}

// Name returns the name of the job.
func (a *AppleMDM) Name() string {
	return appleMDMJobName
}

// appleMDMArgs is the payload for the Apple MDM job.
type appleMDMArgs struct {
	Task            AppleMDMTask `json:"task"`
	HostUUID        string       `json:"host_uuid"`
	TeamID          *uint        `json:"team_id,omitempty"`
	EnrollReference string       `json:"enroll_reference,omitempty"`
}

// Run executes the apple_mdm job.
func (a *AppleMDM) Run(ctx context.Context, argsJSON json.RawMessage) error {
	// if Commander is nil, then mdm is not enabled, so just return without
	// error so we clean up any pending jobs.
	if a.Commander == nil {
		return nil
	}

	var args appleMDMArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	switch args.Task {
	case AppleMDMPostDEPEnrollmentTask:
		return a.runPostDEPEnrollment(ctx, args)
	default:
		return ctxerr.Errorf(ctx, "unknown task: %v", args.Task)
	}
}

func (a *AppleMDM) runPostDEPEnrollment(ctx context.Context, args appleMDMArgs) error {
	panic("unimplemented")
}

// QueueAppleMDMJob queues a apple_mdm job for one of the supported tasks, to
// be processed asynchronously via the worker.
func QueueAppleMDMJob(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	task AppleMDMTask,
	hostUUID string,
	teamID *uint,
	enrollReference string,
) error {
	attrs := []interface{}{
		"enabled", "true",
		appleMDMJobName, task,
		"host_uuid", hostUUID,
		"with_enroll_reference", enrollReference != "",
	}
	if teamID != nil {
		attrs = append(attrs, "team_id", *teamID)
	}
	level.Info(logger).Log(attrs...)

	args := &appleMDMArgs{
		Task:            task,
		HostUUID:        hostUUID,
		TeamID:          teamID,
		EnrollReference: enrollReference,
	}
	job, err := QueueJob(ctx, ds, appleMDMJobName, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}
	level.Debug(logger).Log("job_id", job.ID)
	return nil
}
