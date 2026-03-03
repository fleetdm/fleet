package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/recoverykeypassword"
	"github.com/google/uuid"
)

const RecoveryLockJobName = "recovery_lock"

type recoveryLockTask string

const verifyRecoveryLockTask recoveryLockTask = "verify_recovery_lock"

// RecoveryLock is a worker job that handles recovery lock verification.
type RecoveryLock struct {
	Datastore    fleet.Datastore
	RKPDatastore recoverykeypassword.Datastore
	Commander    recoverykeypassword.MDMCommander
	Log          *slog.Logger
}

func (r *RecoveryLock) Name() string {
	return RecoveryLockJobName
}

type recoveryLockArgs struct {
	Task     recoveryLockTask `json:"task"`
	HostUUID string           `json:"host_uuid"`
	HostID   uint             `json:"host_id"`
}

func (r *RecoveryLock) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args recoveryLockArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	switch args.Task {
	case verifyRecoveryLockTask:
		err := r.verifyRecoveryLock(ctx, args.HostUUID, args.HostID)
		return ctxerr.Wrap(ctx, err, "running verify recovery lock task")

	default:
		return ctxerr.Errorf(ctx, "unknown task: %v", args.Task)
	}
}

func (r *RecoveryLock) verifyRecoveryLock(ctx context.Context, hostUUID string, hostID uint) error {
	r.Log.DebugContext(ctx, "verifying recovery lock", "host_uuid", hostUUID, "host_id", hostID)

	// Get the stored password for the host
	rkp, err := r.RKPDatastore.GetHostRecoveryKeyPassword(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get host recovery key password")
	}

	// Generate a verification command UUID with the prefix
	cmdUUID := recoverykeypassword.VerifyRecoveryLockCommandPrefix + uuid.NewString()

	// Send VerifyRecoveryLock command
	rawCmd := recoverykeypassword.VerifyRecoveryLockCommand(cmdUUID, rkp.Password)
	if err := r.Commander.EnqueueCommand(ctx, []string{hostUUID}, string(rawCmd)); err != nil {
		return ctxerr.Wrap(ctx, err, "send VerifyRecoveryLock command")
	}

	// Update status to verifying with the verify command UUID
	if err := r.RKPDatastore.SetRecoveryLockVerifying(ctx, hostID, cmdUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "set recovery lock verifying")
	}

	r.Log.DebugContext(ctx, "sent VerifyRecoveryLock command",
		"host_id", hostID,
		"host_uuid", hostUUID,
		"command_uuid", cmdUUID,
	)

	return nil
}

// QueueRecoveryLockVerificationJob queues a job to send the VerifyRecoveryLock command.
func QueueRecoveryLockVerificationJob(ctx context.Context, ds fleet.Datastore, logger *slog.Logger, requestDelay time.Duration, hostUUID string, hostID uint) error {
	args := &recoveryLockArgs{
		Task:     verifyRecoveryLockTask,
		HostUUID: hostUUID,
		HostID:   hostID,
	}

	job, err := QueueJobWithDelay(ctx, ds, RecoveryLockJobName, args, requestDelay)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}

	logger.DebugContext(ctx, "queued recovery lock verification job", "job_id", job.ID, "job_name", RecoveryLockJobName, "task", args.Task)
	return nil
}
