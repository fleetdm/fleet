package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
)

const softwareWorkerJobName = "software_worker"

type SoftwareWorkerTask string

type SoftwareWorker struct {
	Datastore     fleet.Datastore
	AndroidModule android.Service
	Log           kitlog.Logger
}

func (v *SoftwareWorker) Name() string {
	return softwareWorkerJobName
}

const (
	makeAndroidAppsAvailableForHostTask     SoftwareWorkerTask = "make_android_apps_available_for_host"
	makeAndroidAppAvailableTask             SoftwareWorkerTask = "make_android_app_available"
	runAndroidSetupExperienceTask           SoftwareWorkerTask = "run_android_setup_experience"
	bulkSetAndroidAppsAvailableForHostTask  SoftwareWorkerTask = "bulk_set_android_apps_available_for_host"
	bulkSetAndroidAppsAvailableForHostsTask SoftwareWorkerTask = "bulk_set_android_apps_available_for_hosts"
)

type softwareWorkerArgs struct {
	Task           SoftwareWorkerTask `json:"task"`
	HostUUID       string             `json:"host_uuid,omitempty"`
	ApplicationID  string             `json:"application_id,omitempty"`
	ApplicationIDs []string           `json:"application_ids,omitempty"`
	EnterpriseName string             `json:"enterprise_name,omitempty"`
	// AppTeamID is *not* a team ID, it is the vpp_apps_teams.id value. This is a bit confusing
	// as a name, but that is what is expected in this field.
	AppTeamID uint `json:"app_team_id,omitempty"`
	HostID    uint `json:"host_id,omitempty"`

	// HostEnrollTeamID is the team ID associated with the host at the time
	// of enrollment, which is the one used to run the setup experience.
	// A value of 0 is used to represent "no team".
	HostEnrollTeamID uint `json:"host_enroll_team_id,omitempty"`

	// PolicyID is the Android Management API Policy ID associated with the host, *not*
	// a Fleet policy ID.
	PolicyID string `json:"policy_id,omitempty"`

	// AppConfigChanged indicates if the android app configuration changed as part
	// of the action that triggered this task.
	AppConfigChanged bool            `json:"app_config_changed,omitempty"`
	UUIDsToIDs       map[string]uint `json:"uuids_to_ids,omitempty"`
}

func (v *SoftwareWorker) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args softwareWorkerArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	switch args.Task {
	case makeAndroidAppsAvailableForHostTask:
		// this task is deprecated (its logic is part of the run setup experience task), but must
		// be kept here in case some pending jobs are still in the queue.
		return ctxerr.Wrapf(
			ctx,
			v.makeAndroidAppsAvailableForHost(ctx, args.HostUUID, args.HostID, args.EnterpriseName, args.PolicyID),
			"running %s task",
			makeAndroidAppsAvailableForHostTask,
		)

	case makeAndroidAppAvailableTask:
		return ctxerr.Wrapf(
			ctx,
			v.makeAndroidAppAvailable(ctx, args.ApplicationID, args.AppTeamID, args.EnterpriseName, args.AppConfigChanged),
			"running %s task",
			makeAndroidAppAvailableTask,
		)

	case runAndroidSetupExperienceTask:
		return ctxerr.Wrapf(
			ctx,
			v.runAndroidSetupExperience(ctx, args.HostUUID, args.HostEnrollTeamID, args.EnterpriseName),
			"running %s task",
			runAndroidSetupExperienceTask,
		)

	case bulkSetAndroidAppsAvailableForHostTask:
		return ctxerr.Wrapf(ctx, v.bulkMakeAndroidAppsAvailableForHost(
			ctx,
			args.HostUUID,
			args.PolicyID,
			args.ApplicationIDs,
			args.EnterpriseName,
		), "running %s task",
			bulkSetAndroidAppsAvailableForHostTask)

	case bulkSetAndroidAppsAvailableForHostsTask:
		return ctxerr.Wrapf(ctx, v.bulkSetAndroidAppsAvailableForHosts(
			ctx,
			args.UUIDsToIDs,
			args.EnterpriseName,
		), "running %s task", bulkSetAndroidAppsAvailableForHostsTask)

	default:
		return ctxerr.Errorf(ctx, "unknown task: %v", args.Task)

	}
}

// this is called when a new app is added to Fleet and when an existing app is updated
// (either its scope of affected hosts changed due to labels conditions, or its
// configuration changed).
func (v *SoftwareWorker) makeAndroidAppAvailable(ctx context.Context, applicationID string, appTeamID uint, enterpriseName string, appConfigChanged bool) error {
	hosts, err := v.Datastore.GetIncludedHostUUIDMapForAppStoreApp(ctx, appTeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "add app store app: getting android hosts in scope")
	}

	config, err := v.Datastore.GetAndroidAppConfigurationByAppTeamID(ctx, appTeamID)
	if err != nil && !fleet.IsNotFound(err) {
		return ctxerr.Wrap(ctx, err, "get android app configuration")
	}
	var configByAppID map[string]json.RawMessage
	if config != nil {
		configByAppID = map[string]json.RawMessage{
			applicationID: *config,
		}
	}

	appPolicies, err := buildApplicationPolicyWithConfig(ctx, []string{applicationID}, configByAppID, "AVAILABLE")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building application policies with config")
	}

	// Update Android MDM policy to include the app in self service
	policyRequestsByHost, err := v.AndroidModule.AddAppsToAndroidPolicy(ctx, enterpriseName, appPolicies, hosts)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "add app store app: add app to android policy")
	}

	// if this is called from an UPDATE (config changed), mark existing installs
	// as "pending" (unless already "failed") and with the correct policy version to verify
	// (currently temporarily stored as a string in associated_event_id, to revisit
	// when we implement full Android apps support).
	if appConfigChanged {
		for hostUUID, policyRequest := range policyRequestsByHost {
			err := v.Datastore.SetAndroidAppInstallPendingApplyConfig(ctx, hostUUID, applicationID, policyRequest.PolicyVersion.V)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "set android app install pending apply config for host %s and app %s", hostUUID, applicationID)
			}
		}
	}

	return nil
}

func (v *SoftwareWorker) ensureHostSpecificPolicyIsApplied(ctx context.Context, hostUUID string, enterpriseName, policyID string) error {
	if policyID == fmt.Sprint(android.DefaultAndroidPolicyID) {
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

		err = v.AndroidModule.BuildAndSendFleetAgentConfig(ctx, enterpriseName, []string{hostUUID}, false)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "build and send fleet agent config for host %s", hostUUID)
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
	return nil
}

func (v *SoftwareWorker) makeAndroidAppsAvailableForHost(ctx context.Context, hostUUID string, hostID uint, enterpriseName, policyID string) error {
	if err := v.ensureHostSpecificPolicyIsApplied(ctx, hostUUID, enterpriseName, policyID); err != nil {
		return ctxerr.Wrapf(ctx, err, "ensuring host-specific policy is applied for host %s", hostUUID)
	}

	androidHost, err := v.Datastore.AndroidHostLiteByHostUUID(ctx, hostUUID)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "get android host by host UUID %s", hostUUID)
	}

	appIDs, err := v.Datastore.GetAndroidAppsInScopeForHost(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get android apps in scope for host")
	}

	if len(appIDs) == 0 {
		return nil
	}

	configsByAppID, err := v.Datastore.BulkGetAndroidAppConfigurations(ctx, appIDs, ptr.ValOrZero(androidHost.TeamID))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "bulk get android app configurations")
	}

	appPolicies, err := buildApplicationPolicyWithConfig(ctx, appIDs, configsByAppID, "AVAILABLE")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building application policies with config")
	}

	_, err = v.AndroidModule.AddAppsToAndroidPolicy(ctx, enterpriseName, appPolicies, map[string]string{hostUUID: hostUUID})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "add app store app: add app to android policy")
	}

	return nil
}

func (v *SoftwareWorker) runAndroidSetupExperience(ctx context.Context,
	hostUUID string, hostEnrollTeamID uint, enterpriseName string) error {
	host, err := v.Datastore.AndroidHostLiteByHostUUID(ctx, hostUUID)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "getting android host lite by uuid %s", hostUUID)
	}

	// first step is to make the apps available for self-service available on the host
	// we do this first because it also takes care of assigning the host-specific policy
	// to the host if necessary.
	policyID := fmt.Sprint(android.DefaultAndroidPolicyID)
	if host.AppliedPolicyID != nil {
		policyID = *host.AppliedPolicyID
	}

	// TODO(mna): obviously it would be ideal to define a single policy at enroll time with
	// everything it needs at once (instead of that call to add self-service app, and the subsequent
	// one to install setup experience apps). I'll keep this as a follow-up optimization if we
	// have a bit of time at the end of this story - it will require a somewhat significant refactor.
	// Also, this may be more API-efficient, but less portable to our ordered, unified queue framework
	// that eventually Android apps will have to fit into
	// (see https://github.com/fleetdm/fleet/issues/33761#issuecomment-3553434984).
	if err := v.makeAndroidAppsAvailableForHost(ctx, hostUUID, host.Host.ID, enterpriseName, policyID); err != nil {
		return ctxerr.Wrapf(ctx, err, "making android apps available for host %s", hostUUID)
	}

	// TODO(mna): if the host has been transferred to another team before it had a chance to install
	// the enrollment team's setup experience software, do we still run those installs?
	// my guess is yes (because we don't _uninstall_ on team transfers, so it should be
	// expected that the original team's software gets installed despite being transferred).
	appIDs, err := v.Datastore.GetVPPAppsToInstallDuringSetupExperience(ctx, &hostEnrollTeamID, string(fleet.AndroidPlatform))
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "getting vpp apps to install during setup experience for team %d", hostEnrollTeamID)
	}

	if len(appIDs) > 0 {
		// NOTE: from my tests, we do need to re-apply the app configs when installing apps,
		// even if they were already applied when making the apps available for self-service.
		// However, once installed, if the app config changes it is applied automatically by the
		// policy change (no need to re-install).
		configsByAppID, err := v.Datastore.BulkGetAndroidAppConfigurations(ctx, appIDs, hostEnrollTeamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "bulk get android app configurations")
		}

		appPolicies, err := buildApplicationPolicyWithConfig(ctx, appIDs, configsByAppID, "PREINSTALLED")
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building application policies with config")
		}

		// assign those apps to the host's Android policy
		hostToPolicyRequest, err := v.AndroidModule.AddAppsToAndroidPolicy(ctx, enterpriseName, appPolicies, map[string]string{hostUUID: hostUUID})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "add app store app: add app to android policy")
		}

		// if it succeeded, it's guaranteed that only one entry exists in that map (as there's only one host)
		var policyRequest *android.MDMAndroidPolicyRequest
		for _, req := range hostToPolicyRequest {
			policyRequest = req
		}
		for _, appID := range appIDs {
			// NOTE: there is a unique index on the command uuid, so we cannot use the
			// Android request's UUID for this, as we currently add many apps in the same request
			// per host. For the moment, this is fine as we don't store any response of the request
			// so there's nothing to show in the UI in the details of the install. When we work
			// on "standard" Android app install support, we'll have to revisit this and either
			// make one API request per app per host (which we may have to do anyway to support the
			// unified queue), or make some DB changes.
			//
			// So in the meantime we use a random uuid in this place.
			err := v.Datastore.InsertAndroidSetupExperienceSoftwareInstall(ctx, &fleet.HostAndroidVPPSoftwareInstall{
				HostID:            host.Host.ID,
				AdamID:            appID,
				CommandUUID:       uuid.NewString(),
				AssociatedEventID: fmt.Sprint(policyRequest.PolicyVersion.V),
			})
			if err != nil {
				return ctxerr.Wrap(ctx, err, "inserting android setup experience install request")
			}
		}
	}

	return nil
}

func (v *SoftwareWorker) bulkMakeAndroidAppsAvailableForHost(ctx context.Context, hostUUID, policyID string, applicationIDs []string, enterpriseName string) error {
	host, err := v.Datastore.AndroidHostLiteByHostUUID(ctx, hostUUID)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "getting android host lite by uuid %s", hostUUID)
	}

	configsByAppID, err := v.Datastore.BulkGetAndroidAppConfigurations(ctx, applicationIDs, ptr.ValOrZero(host.Host.TeamID))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "bulk get android app configurations")
	}

	appPolicies, err := buildApplicationPolicyWithConfig(ctx, applicationIDs, configsByAppID, "AVAILABLE")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building application policies with config")
	}

	// Update Android MDM policy to include the apps in self service
	err = v.AndroidModule.SetAppsForAndroidPolicy(ctx, enterpriseName, appPolicies, map[string]string{hostUUID: policyID})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "make android apps available")
	}

	return nil
}

func buildApplicationPolicyWithConfig(ctx context.Context, appIDs []string,
	configsByAppID map[string]json.RawMessage, installType string) ([]*androidmanagement.ApplicationPolicy, error) {

	appPolicies := make([]*androidmanagement.ApplicationPolicy, 0, len(appIDs))
	for _, appID := range appIDs {
		var androidAppConfig struct {
			ManagedConfiguration json.RawMessage `json:"managedConfiguration"`
			WorkProfileWidgets   string          `json:"workProfileWidgets"`
		}
		if config := configsByAppID[appID]; config != nil {
			if err := json.Unmarshal(config, &androidAppConfig); err != nil {
				// should never happen, as it is stored as json in the db and is pre-validated
				return nil, ctxerr.Wrap(ctx, err, "unmarshal android app configuration")
			}
		} else {
			// if there is no config for this app, we must make sure we clear any previously-applied
			// config.
			androidAppConfig.ManagedConfiguration = json.RawMessage{}
			androidAppConfig.WorkProfileWidgets = "WORK_PROFILE_WIDGETS_UNSPECIFIED"
		}
		appPolicies = append(appPolicies, &androidmanagement.ApplicationPolicy{
			PackageName:          appID,
			InstallType:          installType,
			ManagedConfiguration: googleapi.RawMessage(androidAppConfig.ManagedConfiguration),
			WorkProfileWidgets:   androidAppConfig.WorkProfileWidgets,
		})
	}
	return appPolicies, nil
}

func QueueRunAndroidSetupExperience(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger,
	hostUUID string, hostEnrollTeamID *uint, enterpriseName string) error {

	var enrollTeamID uint
	if hostEnrollTeamID != nil {
		enrollTeamID = *hostEnrollTeamID
	}
	args := &softwareWorkerArgs{
		Task:             runAndroidSetupExperienceTask,
		HostUUID:         hostUUID,
		EnterpriseName:   enterpriseName,
		HostEnrollTeamID: enrollTeamID,
	}

	job, err := QueueJob(ctx, ds, softwareWorkerJobName, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}

	level.Debug(logger).Log("job_id", job.ID, "job_name", softwareWorkerJobName, "task", args.Task)
	return nil
}

func QueueMakeAndroidAppAvailableJob(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, applicationID string, appTeamID uint, enterpriseName string, appConfigChanged bool) error {
	args := &softwareWorkerArgs{
		Task:             makeAndroidAppAvailableTask,
		ApplicationID:    applicationID,
		AppTeamID:        appTeamID,
		EnterpriseName:   enterpriseName,
		AppConfigChanged: appConfigChanged,
	}

	job, err := QueueJob(ctx, ds, softwareWorkerJobName, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}

	level.Debug(logger).Log("job_id", job.ID, "job_name", softwareWorkerJobName, "task", args.Task)
	return nil
}

func QueueBulkSetAndroidAppsAvailableForHost(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	hostUUID string,
	policyID string,
	applicationIDs []string,
	enterpriseName string,
) error {

	args := &softwareWorkerArgs{
		Task:           bulkSetAndroidAppsAvailableForHostTask,
		HostUUID:       hostUUID,
		PolicyID:       policyID,
		EnterpriseName: enterpriseName,
		ApplicationIDs: applicationIDs,
	}

	job, err := QueueJob(ctx, ds, softwareWorkerJobName, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}

	level.Debug(logger).Log("job_id", job.ID, "job_name", softwareWorkerJobName, "task", args.Task)
	return nil
}

func (v *SoftwareWorker) bulkSetAndroidAppsAvailableForHosts(ctx context.Context, uuidsToIDs map[string]uint, enterpriseName string) error {
	// for each host
	// get the set of self-service apps that are in scope for it
	for uuid, hostID := range uuidsToIDs {
		androidHost, err := v.Datastore.AndroidHostLiteByHostUUID(ctx, uuid)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "get android host by host UUID %s", uuid)
		}

		appIDs, err := v.Datastore.GetAndroidAppsInScopeForHost(ctx, hostID)
		if err != nil {
			return ctxerr.WrapWithData(ctx, err, "get android apps in scope for host", map[string]any{"host_id": hostID})
		}

		configsByAppID, err := v.Datastore.BulkGetAndroidAppConfigurations(ctx, appIDs, ptr.ValOrZero(androidHost.TeamID))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "bulk get android app configurations")
		}

		appPolicies, err := buildApplicationPolicyWithConfig(ctx, appIDs, configsByAppID, "AVAILABLE")
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building application policies with config")
		}

		// Include the Fleet Agent in the app list so it's not removed when we replace the apps.
		fleetAgentPolicy, err := v.AndroidModule.BuildFleetAgentApplicationPolicy(ctx, uuid)
		if err != nil {
			level.Error(v.Log).Log("msg", "failed to build Fleet Agent policy, Fleet Agent may be removed", "host_uuid", uuid, "err", err)
		} else if fleetAgentPolicy != nil {
			appPolicies = append(appPolicies, fleetAgentPolicy)
		}

		err = v.AndroidModule.SetAppsForAndroidPolicy(ctx, enterpriseName, appPolicies, map[string]string{uuid: uuid})

		if err != nil {
			return ctxerr.WrapWithData(ctx, err, "set apps for android policy", map[string]any{"host_id": hostID})
		}

	}

	return nil

}

func QueueBulkSetAndroidAppsAvailableForHosts(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	uuidsToIDs map[string]uint,
	enterpriseName string) error {

	args := &softwareWorkerArgs{
		Task:           bulkSetAndroidAppsAvailableForHostsTask,
		UUIDsToIDs:     uuidsToIDs,
		EnterpriseName: enterpriseName,
	}

	job, err := QueueJob(ctx, ds, softwareWorkerJobName, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}

	level.Debug(logger).Log("job_id", job.ID, "job_name", softwareWorkerJobName, "task", args.Task)
	return nil
}
