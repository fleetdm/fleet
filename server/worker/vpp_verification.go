package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"google.golang.org/api/androidmanagement/v1"
)

const AppleSoftwareJobName = "apple_software"

type AppleSoftwareTask string

const VerifyVPPTask AppleSoftwareTask = "verify_vpp_installs"
const MakeAndroidAppAvailableTask AppleSoftwareTask = "make_android_apps_available"
const MakeAndroidAppsAvailableForHostTask AppleSoftwareTask = "make_android_app_available_for_host"

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
	HostID                  uint              `json:"host_id"`
	PolicyID                string            `json:"policy_id"`
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

	case MakeAndroidAppAvailableTask:
		return ctxerr.Wrapf(
			ctx,
			v.makeAndroidAppAvailable(ctx, args.ApplicationID, args.AppTeamID, args.EnterpriseName),
			"running %s task",
			MakeAndroidAppAvailableTask,
		)

	case MakeAndroidAppsAvailableForHostTask:
		return ctxerr.Wrapf(
			ctx,
			v.makeAndroidAppsAvailableForHost(ctx, args.HostUUID, args.HostID, args.EnterpriseName, args.PolicyID),
			"running %s task",
			MakeAndroidAppsAvailableForHostTask,
		)

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

func (v *AppleSoftware) makeAndroidAppAvailable(ctx context.Context, applicationID string, appTeamID uint, enterpriseName string) error {
	hosts, err := v.Datastore.GetIncludedHostUUIDMapForAppStoreApp(ctx, appTeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "add app store app: getting android hosts in scope")
	}

	// Update Android MDM policy to include the app in self service
	err = v.AndroidModule.AddAppToAndroidPolicy(ctx, enterpriseName, []string{applicationID}, hosts)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "add app store app: add app to android policy")
	}

	return nil
}

func QueueMakeAndroidAppAvailableJob(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, applicationID string, appTeamID uint, enterpriseName string) error {
	args := &appleSoftwareArgs{
		Task:           MakeAndroidAppAvailableTask,
		ApplicationID:  applicationID,
		AppTeamID:      appTeamID,
		EnterpriseName: enterpriseName,
	}

	job, err := QueueJob(ctx, ds, AppleSoftwareJobName, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}

	level.Debug(logger).Log("job_id", job.ID, "job_name", appleMDMJobName, "task", MakeAndroidAppAvailableTask)
	return nil
}

func (v *AppleSoftware) makeAndroidAppsAvailableForHost(ctx context.Context, hostUUID string, hostID uint, enterpriseName, policyID string) error {

	if policyID == "1" {
		var policy androidmanagement.Policy

		policy.StatusReportingSettings = &androidmanagement.StatusReportingSettings{
			DeviceSettingsEnabled:        true,
			MemoryInfoEnabled:            true,
			NetworkInfoEnabled:           true,
			DisplayInfoEnabled:           true,
			PowerManagementEventsEnabled: true,
			HardwareStatusEnabled:        true,
			SystemPropertiesEnabled:      true,
			SoftwareInfoEnabled:          true,
			CommonCriteriaModeEnabled:    true,
			ApplicationReportsEnabled:    true,
			ApplicationReportingSettings: nil, // only option is "includeRemovedApps", which I opted not to enable (we can diff apps to see removals)
		}

		policyName := fmt.Sprintf("%s/policies/%s", enterpriseName, hostUUID)
		_, err := v.AndroidModule.PatchPolicy(ctx, hostUUID, policyName, &policy, nil)
		if err != nil {
			return err
		}
		device := &androidmanagement.Device{
			PolicyName: policyName,
			// State must be specified when updating a device, otherwise it fails with
			// "Illegal state transition from ACTIVE to DEVICE_STATE_UNSPECIFIED"
			//
			// > Note that when calling enterprises.devices.patch, ACTIVE and
			// > DISABLED are the only allowable values.

			// TODO(ap): should we send whatever the previous state was? If it was DISABLED,
			// we probably don't want to re-enable it by accident. Those are the only
			// 2 valid states when patching a device.
			State: "ACTIVE",
		}
		androidHost, err := v.Datastore.AndroidHostLiteByHostUUID(ctx, hostUUID)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "get android host by host UUID %s", hostUUID)
		}
		deviceName := fmt.Sprintf("%s/devices/%s", enterpriseName, androidHost.DeviceID)
		_, err = v.AndroidModule.PatchDevice(ctx, hostUUID, deviceName, device)
		if err != nil {
			return err
		}
	}

	appIDs, err := v.Datastore.GetAndroidAppsInScopeForHost(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get android apps in scope for host")
	}

	if len(appIDs) == 0 {
		return nil
	}

	err = v.AndroidModule.AddAppToAndroidPolicy(ctx, enterpriseName, appIDs, map[string]string{hostUUID: hostUUID})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "add app store app: add app to android policy")
	}

	return nil
}

func QueueMakeAndroidAppsAvailableForHostJob(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, hostUUID string, hostID uint, enterpriseName, policyID string) error {
	args := &appleSoftwareArgs{
		Task:           MakeAndroidAppsAvailableForHostTask,
		HostUUID:       hostUUID,
		HostID:         hostID,
		EnterpriseName: enterpriseName,
		PolicyID:       policyID,
	}

	job, err := QueueJob(ctx, ds, AppleSoftwareJobName, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}

	level.Debug(logger).Log("job_id", job.ID, "job_name", appleMDMJobName, "task", MakeAndroidAppsAvailableForHostTask)
	return nil
}
