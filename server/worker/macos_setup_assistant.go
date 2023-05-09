package worker

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// Name of the macos setup assistant job as registered in the worker. Note that
// although it is a single job, it processes a number of different-but-related
// tasks, identified by the Task field in the job's payload.
const macosSetupAssistantJobName = "macos_setup_assistant"

type MacosSetupAssistantTask string

// List of supported tasks.
const (
	MacosSetupAssistantProfileChanged    MacosSetupAssistantTask = "profile_changed"
	MacosSetupAssistantProfileDeleted    MacosSetupAssistantTask = "profile_deleted"
	MacosSetupAssistantTeamDeleted       MacosSetupAssistantTask = "team_deleted"
	MacosSetupAssistantHostsTransferred  MacosSetupAssistantTask = "hosts_transferred"
	MacosSetupAssistantUpdateAllProfiles MacosSetupAssistantTask = "update_all_profiles"
)

// MacosSetupAssistant is the job processor for the macos_setup_assistant job.
type MacosSetupAssistant struct {
	Datastore fleet.Datastore
	Log       kitlog.Logger
}

// Name returns the name of the job.
func (m *MacosSetupAssistant) Name() string {
	return macosSetupAssistantJobName
}

// macosSetupAssistantArgs is the payload for the macos setup assistant job.
type macosSetupAssistantArgs struct {
	Task   MacosSetupAssistantTask `json:"task"`
	TeamID *uint                   `json:"team_id,omitempty"`
	// Note that only DEP-enrolled hosts in Fleet MDM should be provided.
	HostSerialNumbers []string `json:"host_serial_numbers,omitempty"`
}

// Run executes the macos_setup_assistant job.
func (m *MacosSetupAssistant) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args macosSetupAssistantArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	switch args.Task {
	case MacosSetupAssistantProfileChanged:
	case MacosSetupAssistantProfileDeleted:
	case MacosSetupAssistantTeamDeleted:
	case MacosSetupAssistantHostsTransferred:
	case MacosSetupAssistantUpdateAllProfiles:
	default:
		return ctxerr.Errorf(ctx, "unknown task: %v", args.Task)
	}
	panic("unimplemented")
}

// QueueMacosSetupAssistantJob queues a macos_setup_assistant job for one of
// the supported tasks, to be processed asynchronously via the worker.
func QueueMacosSetupAssistantJob(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	task MacosSetupAssistantTask,
	teamID *uint,
	serialNumbers ...string,
) error {
	attrs := []interface{}{
		"enabled", "true",
		macosSetupAssistantJobName, task,
		"hosts_count", len(serialNumbers),
	}
	if teamID != nil {
		attrs = append(attrs, "team_id", *teamID)
	}
	level.Info(logger).Log(attrs...)

	args := &macosSetupAssistantArgs{
		Task:              task,
		TeamID:            teamID,
		HostSerialNumbers: serialNumbers,
	}
	job, err := QueueJob(ctx, ds, macosSetupAssistantJobName, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}
	level.Debug(logger).Log("job_id", job.ID)
	return nil
}
