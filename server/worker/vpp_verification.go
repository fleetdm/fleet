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
)

const AppleSoftwareJobName = "apple_software"

type AppleSoftwareTask string

const verifyVPPTask AppleSoftwareTask = "verify_vpp_installs"

type AppleSoftware struct {
	Datastore fleet.Datastore
	Commander *apple_mdm.MDMAppleCommander
	Log       kitlog.Logger
}

func (v *AppleSoftware) Name() string {
	return AppleSoftwareJobName
}

type appleSoftwareArgs struct {
	Task                    AppleSoftwareTask `json:"task"`
	HostUUID                string            `json:"host_uuid"`
	VerificationCommandUUID string            `json:"verification_command_uuid"`
	DisableManagedOnlyApps  bool              `json:"disable_managed_only_apps,omitempty"`
}

func (v *AppleSoftware) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args appleSoftwareArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	switch args.Task {
	case verifyVPPTask:
		err := v.verifyVPPInstalls(ctx, args.HostUUID, args.VerificationCommandUUID, args.DisableManagedOnlyApps)
		return ctxerr.Wrap(ctx, err, "running migrate VPP token task")

	default:
		return ctxerr.Errorf(ctx, "unknown task: %v", args.Task)
	}
}

func (v *AppleSoftware) verifyVPPInstalls(ctx context.Context, hostUUID, verificationCommandUUID string, disableManagedOnlyApps bool) error {
	level.Debug(v.Log).Log("msg", "verifying VPP installs", "host_uuid", hostUUID, "verification_command_uuid", verificationCommandUUID)
	newListCmdUUID := fleet.VerifySoftwareInstallCommandUUID()
	// for app verification, we always request only managed apps except
	// if disableManagedOnlyApps is true
	err := v.Commander.InstalledApplicationList(ctx, []string{hostUUID}, newListCmdUUID, !disableManagedOnlyApps)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "sending installed application list command in verify")
	}

	if err := v.Datastore.ReplaceVPPInstallVerificationUUID(ctx, verificationCommandUUID, newListCmdUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "update vpp install record")
	}

	if err := v.Datastore.ReplaceInHouseAppInstallVerificationUUID(ctx, verificationCommandUUID, newListCmdUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "update in-house app install record")
	}

	level.Debug(v.Log).Log("msg", "new installed application list command sent", "uuid", newListCmdUUID)

	return nil
}

func QueueVPPInstallVerificationJob(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, requestDelay time.Duration, hostUUID, verificationCommandUUID string, disableManagedOnly bool) error {
	args := &appleSoftwareArgs{
		Task:                    verifyVPPTask,
		HostUUID:                hostUUID,
		VerificationCommandUUID: verificationCommandUUID,
		DisableManagedOnlyApps:  disableManagedOnly,
	}

	job, err := QueueJobWithDelay(ctx, ds, AppleSoftwareJobName, args, requestDelay)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}

	level.Debug(logger).Log("job_id", job.ID, "job_name", appleMDMJobName, "task", args.Task)
	return nil
}
