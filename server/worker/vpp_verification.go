package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

const AppleSoftwareJobName = "apple_software"

type AppleSoftwareTask string

const VerifyVPPTask AppleSoftwareTask = "verify_vpp_installs"
const MakeAndroidAppsAvailableTask AppleSoftwareTask = "make_android_apps_available"

type AppleSoftware struct {
	Datastore     fleet.Datastore
	Commander     *apple_mdm.MDMAppleCommander
	AndroidModule android.Service
	Log           kitlog.Logger
}

func (v *AppleSoftware) Name() string {
	return AppleSoftwareJobName
}

type appleSoftwareArgs struct {
	Task                    AppleSoftwareTask `json:"task"`
	HostUUID                string            `json:"host_uuid"`
	VerificationCommandUUID string            `json:"verification_command_uuid"`
	ApplicationID           string            `json:"application_id"`
	AppTeamID               uint              `json:"app_team_id"`
	EnterpriseName          string            `json:"enterprise_name"`
}

func (v *AppleSoftware) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args appleSoftwareArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	switch args.Task {
	case VerifyVPPTask:
		err := v.verifyVPPInstalls(ctx, args.HostUUID, args.VerificationCommandUUID)
		return ctxerr.Wrap(ctx, err, "running migrate VPP token task")

	case MakeAndroidAppsAvailableTask:
		return ctxerr.Wrap(ctx, v.makeAndroidAppsAvailable(ctx, args.ApplicationID, args.AppTeamID, args.EnterpriseName))

	default:
		return ctxerr.Errorf(ctx, "unknown task: %v", args.Task)
	}
}

func (v *AppleSoftware) verifyVPPInstalls(ctx context.Context, hostUUID, verificationCommandUUID string) error {
	level.Debug(v.Log).Log("msg", "verifying VPP installs", "host_uuid", hostUUID, "verification_command_uuid", verificationCommandUUID)
	newListCmdUUID := fleet.VerifySoftwareInstallCommandUUID()
	err := v.Commander.InstalledApplicationList(ctx, []string{hostUUID}, newListCmdUUID, true)
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

func QueueVPPInstallVerificationJob(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, task AppleSoftwareTask, requestDelay time.Duration, hostUUID, verificationCommandUUID string) error {
	args := &appleSoftwareArgs{
		Task:                    task,
		HostUUID:                hostUUID,
		VerificationCommandUUID: verificationCommandUUID,
	}

	job, err := QueueJobWithDelay(ctx, ds, AppleSoftwareJobName, args, requestDelay)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}

	level.Debug(logger).Log("job_id", job.ID, "job_name", appleMDMJobName, "task", task)
	return nil
}

func (v *AppleSoftware) makeAndroidAppsAvailable(ctx context.Context, applicationID string, appTeamID uint, enterpriseName string) error {
	hosts, err := v.Datastore.GetIncludedHostUUIDMapForAppStoreApp(ctx, appTeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "add app store app: getting android hosts in scope")
	}

	// Update Android MDM policy to include the app in self service
	err = v.AndroidModule.AddAppToAndroidPolicy(ctx, enterpriseName, applicationID, hosts)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "add app store app: add app to android policy")
	}

	return nil
}

func QueueMakeAndroidAppAvailableJob(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, applicationID string, appTeamID uint, enterpriseName string) error {
	args := &appleSoftwareArgs{
		Task:           MakeAndroidAppsAvailableTask,
		ApplicationID:  applicationID,
		AppTeamID:      appTeamID,
		EnterpriseName: enterpriseName,
	}

	job, err := QueueJob(ctx, ds, AppleSoftwareJobName, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}

	level.Debug(logger).Log("job_id", job.ID, "job_name", appleMDMJobName, "task", MakeAndroidAppsAvailableTask)
	return nil
}
