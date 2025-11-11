package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"google.golang.org/api/androidmanagement/v1"
)

const SoftwareWorkerJobName = "software_worker"

type SoftwareWorkerTask string

type SoftwareWorker struct {
	Datastore     fleet.Datastore
	AndroidModule android.Service
	Log           kitlog.Logger
}

func (v *SoftwareWorker) Name() string {
	return SoftwareWorkerJobName
}

const MakeAndroidAppsAvailableForHostTask SoftwareWorkerTask = "make_android_apps_available_for_host"
const MakeAndroidAppAvailableTask SoftwareWorkerTask = "make_android_app_available"

type softwareWorkerArgs struct {
	Task           SoftwareWorkerTask `json:"task"`
	HostUUID       string             `json:"host_uuid"`
	ApplicationID  string             `json:"application_id"`
	EnterpriseName string             `json:"enterprise_name"`
	AppTeamID      uint               `json:"app_team_id"`
	HostID         uint               `json:"host_id"`
	PolicyID       string             `json:"policy_id"`
}

func (v *SoftwareWorker) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args softwareWorkerArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	switch args.Task {

	case MakeAndroidAppsAvailableForHostTask:
		return ctxerr.Wrapf(
			ctx,
			v.makeAndroidAppsAvailableForHost(ctx, args.HostUUID, args.HostID, args.EnterpriseName, args.PolicyID),
			"running %s task",
			MakeAndroidAppsAvailableForHostTask,
		)

	case MakeAndroidAppAvailableTask:
		return ctxerr.Wrapf(
			ctx,
			v.makeAndroidAppAvailable(ctx, args.ApplicationID, args.AppTeamID, args.EnterpriseName),
			"running %s task",
			MakeAndroidAppAvailableTask,
		)

	default:
		return ctxerr.Errorf(ctx, "unknown task: %v", args.Task)
	}
}

func (v *SoftwareWorker) makeAndroidAppAvailable(ctx context.Context, applicationID string, appTeamID uint, enterpriseName string) error {
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
	args := &softwareWorkerArgs{
		Task:           MakeAndroidAppAvailableTask,
		ApplicationID:  applicationID,
		AppTeamID:      appTeamID,
		EnterpriseName: enterpriseName,
	}

	job, err := QueueJob(ctx, ds, SoftwareWorkerJobName, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}

	level.Debug(logger).Log("job_id", job.ID, "job_name", SoftwareWorkerJobName, "task", MakeAndroidAppAvailableTask)
	return nil
}

func (v *SoftwareWorker) makeAndroidAppsAvailableForHost(ctx context.Context, hostUUID string, hostID uint, enterpriseName, policyID string) error {

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
	args := &softwareWorkerArgs{
		Task:           MakeAndroidAppsAvailableForHostTask,
		HostUUID:       hostUUID,
		HostID:         hostID,
		EnterpriseName: enterpriseName,
		PolicyID:       policyID,
	}

	job, err := QueueJob(ctx, ds, SoftwareWorkerJobName, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}

	level.Debug(logger).Log("job_id", job.ID, "job_name", SoftwareWorkerJobName, "task", MakeAndroidAppsAvailableForHostTask)
	return nil
}
