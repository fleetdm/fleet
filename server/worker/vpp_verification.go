package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
)

const VPPVerificationJobName = "vpp_verification"

type VPPVerificationTask string

const VerifyVPPTask VPPVerificationTask = "verify_vpp_installs"

type VPPVerification struct {
	Datastore fleet.Datastore
	Commander *apple_mdm.MDMAppleCommander
	Log       kitlog.Logger
}

func (v *VPPVerification) Name() string {
	return VPPVerificationJobName
}

type vppVerificationArgs struct {
	Task                    VPPVerificationTask `json:"task"`
	HostUUID                string              `json:"host_uuid"`
	VerificationCommandUUID string              `json:"verification_command_uuid"`
}

func (v *VPPVerification) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args vppVerificationArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	switch args.Task {
	case VerifyVPPTask:
		err := v.verifyVPPInstalls(ctx, args.HostUUID, args.VerificationCommandUUID)
		return ctxerr.Wrap(ctx, err, "running migrate VPP token task")

	default:
		return ctxerr.Errorf(ctx, "unknown task: %v", args.Task)
	}
}

func (v *VPPVerification) verifyVPPInstalls(ctx context.Context, hostUUID, verificationCommandUUID string) error {
	pendingCmds, err := v.Datastore.GetAcknowledgedMDMCommandsByHost(ctx, hostUUID, "InstalledApplicationList")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get pending mdm commands by host")
	}
	// Only send a new list command if none are in flight. If there's one in
	// flight, the install will be verified by that one.
	if len(pendingCmds) == 0 {
		newListCmdUUID := fleet.RefetchVPPAppInstallsCommandUUIDPrefix + uuid.NewString()
		if err := v.Datastore.UpdateVPPInstallVerificationCommandByVerifyUUID(ctx, verificationCommandUUID, newListCmdUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "update install record")
		}
		err := v.Commander.InstalledApplicationList(ctx, []string{hostUUID}, newListCmdUUID, true)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "sending installed application list command in verify")
		}
		level.Debug(v.Log).Log("msg", "new installed application list command sent", "uuid", newListCmdUUID)
	}

	return nil
}

func QueueVPPInstallVerificationJob(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, task VPPVerificationTask, requestDelay time.Duration, hostUUID, verificationCommandUUID string) error {
	args := &vppVerificationArgs{
		Task:                    task,
		HostUUID:                hostUUID,
		VerificationCommandUUID: verificationCommandUUID,
	}

	job, err := QueueJobWithDelay(ctx, ds, VPPVerificationJobName, args, requestDelay)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}

	level.Debug(logger).Log("job_id", job.ID)
	return nil
}
