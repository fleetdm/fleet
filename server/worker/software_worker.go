package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/fleetdm/fleet/v4/server/mdm/profiles"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"google.golang.org/api/androidmanagement/v1"
)

const softwareWorkerJobName = "software_worker"

type SoftwareWorkerTask string

type SoftwareWorker struct {
	Datastore        fleet.Datastore
	AndroidModule    android.Service
	Log              *slog.Logger
	AndroidBatchSize int
}

func (v *SoftwareWorker) Name() string {
	return softwareWorkerJobName
}

const (
	makeAndroidAppsAvailableForHostTask     SoftwareWorkerTask = "make_android_apps_available_for_host" // deprecated
	makeAndroidAppAvailableTask             SoftwareWorkerTask = "make_android_app_available"           // deprecated
	makeAndroidAppAvailableBatchTask        SoftwareWorkerTask = "make_android_app_available_batch"     // deprecated
	makeAndroidAppUnavailableTask           SoftwareWorkerTask = "make_android_app_unavailable"         // deprecated
	runAndroidSetupExperienceTask           SoftwareWorkerTask = "run_android_setup_experience"
	bulkSetAndroidAppsAvailableForHostTask  SoftwareWorkerTask = "bulk_set_android_apps_available_for_host" // deprecated
	bulkSetAndroidAppsAvailableForHostsTask SoftwareWorkerTask = "bulk_set_android_apps_available_for_hosts"
	resendVPPAppConfigurationTask           SoftwareWorkerTask = "resend_vpp_app_configuration"       // deprecated
	resendVPPAppConfigurationBatchTask      SoftwareWorkerTask = "resend_vpp_app_configuration_batch" // deprecated
)

type softwareWorkerArgs struct {
	Task           SoftwareWorkerTask `json:"task"`
	HostUUID       string             `json:"host_uuid,omitempty"`
	ApplicationID  string             `json:"application_id,omitempty"`
	ApplicationIDs []string           `json:"application_ids,omitempty"`
	EnterpriseName string             `json:"enterprise_name,omitempty"`
	// AppTeamID is *not* a team ID, it is the vpp_apps_teams.id value. This is a bit confusing
	// as a name, but that is what is expected in this field.
	AppTeamID uint `json:"app_team_id,omitempty"` //nolint:apiparamcheck
	HostID    uint `json:"host_id,omitempty"`

	// HostEnrollTeamID is the team ID associated with the host at the time
	// of enrollment, which is the one used to run the setup experience.
	// A value of 0 is used to represent "no team".
	HostEnrollTeamID uint `json:"host_enroll_team_id,omitempty"` //nolint:apiparamcheck // not user-facing

	// PolicyID is the Android Management API Policy ID associated with the host, *not*
	// a Fleet policy ID.
	PolicyID string `json:"policy_id,omitempty"`

	// AppConfigChanged indicates if the android app configuration changed as part
	// of the action that triggered this task.
	AppConfigChanged bool            `json:"app_config_changed,omitempty"`
	UUIDsToIDs       map[string]uint `json:"uuids_to_ids,omitempty"`

	// HostUUIDToPolicyID is a map of host UUID as key to policy ID as value
	// for which the app to make unavailable applies.
	HostUUIDToPolicyID map[string]string `json:"host_uuid_to_policy_id,omitempty"`

	VersionLabelsChanged bool `json:"version_labels_changed,omitempty"`
}

func (v *SoftwareWorker) Run(ctx context.Context, argsJSON json.RawMessage) error {
	// Jobs run one at a time on a worker shared with other job types, and failed jobs are retried by the
	// worker, so fail fast on AMAPI quota errors instead of waiting them out here.
	ctx = androidmgmt.WithoutRetry(ctx)

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

	case makeAndroidAppAvailableTask, makeAndroidAppAvailableBatchTask, makeAndroidAppUnavailableTask, bulkSetAndroidAppsAvailableForHostTask:
		// these tasks are deprecated, the Android app reconciler cron sends the app list, but they must be kept here in
		// case some pending jobs are still in the queue.
		return nil

	case runAndroidSetupExperienceTask:
		return ctxerr.Wrapf(
			ctx,
			v.runAndroidSetupExperience(ctx, args.HostUUID, args.HostEnrollTeamID, args.EnterpriseName),
			"running %s task",
			runAndroidSetupExperienceTask,
		)

	case bulkSetAndroidAppsAvailableForHostsTask:
		return ctxerr.Wrapf(ctx, v.updateAndroidHostsAfterFleetTransfer(ctx, args.UUIDsToIDs), "running %s task", bulkSetAndroidAppsAvailableForHostsTask)

	case resendVPPAppConfigurationTask, resendVPPAppConfigurationBatchTask:
		// these tasks are deprecated, the App Store app configurations cron sends configurations, but they must be
		// kept here in case some pending jobs are still in the queue.
		return nil

	default:
		return ctxerr.Errorf(ctx, "unknown task: %v", args.Task)

	}
}

// this is called when a new app is added to Fleet and when an existing app is updated
// (either its scope of affected hosts changed due to labels conditions, or its
// configuration changed).
// this is called when an app is removed from Fleet.
// prepareHostSpecificPolicy sets the settings of the host-specific policy before the device is moved to it.
func (v *SoftwareWorker) prepareHostSpecificPolicy(ctx context.Context, hostUUID string, enterpriseName string) error {
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

	// That patch replaces every setting of the host policy, which a re-enrolled host
	// reuses and the profile reconciler may already have filled for this enrollment,
	// so have the reconciler send the host's profiles again.
	if err := v.Datastore.ResetMDMAndroidHostProfilesForRedelivery(ctx, hostUUID); err != nil {
		return ctxerr.Wrapf(ctx, err, "reset android profiles for redelivery for host %s", hostUUID)
	}

	err = v.AndroidModule.BuildAndSendFleetAgentConfig(ctx, enterpriseName, []string{hostUUID}, false)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "build and send fleet agent config for host %s", hostUUID)
	}
	return nil
}

func (v *SoftwareWorker) moveDeviceToHostSpecificPolicy(ctx context.Context, androidHost *fleet.AndroidHost, enterpriseName string) error {
	device := &androidmanagement.Device{
		PolicyName: fmt.Sprintf("%s/policies/%s", enterpriseName, androidHost.Host.UUID),
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
	deviceName := fmt.Sprintf("%s/devices/%s", enterpriseName, androidHost.DeviceID)
	_, err := v.AndroidModule.PatchDevice(ctx, androidHost.Host.UUID, deviceName, device)
	return err
}

func (v *SoftwareWorker) makeAndroidAppsAvailableForHost(ctx context.Context, hostUUID string, hostID uint, enterpriseName, policyID string) error {
	onDefaultPolicy := policyID == fmt.Sprint(android.DefaultAndroidPolicyID)
	if onDefaultPolicy {
		if err := v.prepareHostSpecificPolicy(ctx, hostUUID, enterpriseName); err != nil {
			return ctxerr.Wrapf(ctx, err, "preparing host-specific policy for host %s", hostUUID)
		}
	}

	androidHost, err := v.Datastore.AndroidHostLiteByHostUUID(ctx, hostUUID)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "get android host by host UUID %s", hostUUID)
	}

	appsInScope, err := v.Datastore.GetAndroidAppsInScopeForHost(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get android apps in scope for host")
	}

	policyBuiltAt := time.Now()
	vppAppTeamIDs := make([]uint, 0, len(appsInScope))
	for _, app := range appsInScope {
		vppAppTeamIDs = append(vppAppTeamIDs, app.AppTeamID)
	}

	configsByAppID, err := v.Datastore.BulkGetAndroidAppConfigurations(ctx, vppAppTeamIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "bulk get android app configurations")
	}

	// Substitute per app, so that an app this host can't supply a referenced value for
	// doesn't keep the rest from replacing the previous enrollment's apps, nor the device
	// from moving to its host policy. Whichever value is missing has its own resend trigger.
	substitutionHost := androidHostToSubstitutionHost(androidHost)
	availableAppIDs := make([]string, 0, len(appsInScope))
	installs := make([]fleet.HostAndroidSoftwareInstall, 0, len(appsInScope)+1)
	var failedInstalls []fleet.HostAndroidSoftwareInstall
	var substituted []byte
	for _, app := range appsInScope {
		appID := app.AdamID
		install := fleet.HostAndroidSoftwareInstall{HostID: hostID, AdamID: appID, VPPAppTeamID: new(app.AppTeamID), InstallType: android.AppStatusAvailable,
			Status: fleet.AndroidSoftwareInstallStatusPending, PolicyBuiltAt: policyBuiltAt}
		config, ok := configsByAppID[appID]
		if !ok {
			availableAppIDs = append(availableAppIDs, appID)
			installs = append(installs, install)
			continue
		}
		substituted, err = profiles.SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, v.Datastore, config, substitutionHost)
		if err != nil {
			if profiles.IsUnresolvableAndroidAppConfigForHost(err) {
				v.Log.WarnContext(ctx, "skipping android app for host that can't supply a value referenced by its config",
					"host_uuid", hostUUID, "application_id", appID, "err", err)
				install.Status = fleet.AndroidSoftwareInstallStatusFailed
				failedInstalls = append(failedInstalls, install)
				continue
			}
			return ctxerr.Wrapf(ctx, err, "substitute fleet vars in android app config for app %s", appID)
		}
		configsByAppID[appID] = substituted
		availableAppIDs = append(availableAppIDs, appID)
		installs = append(installs, install)
	}

	appPolicies, err := profiles.BuildAndroidApplicationPolicies(ctx, availableAppIDs, configsByAppID, android.AppStatusAvailable)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building application policies with config")
	}

	// A re-enrolled host reuses its host policy, which still lists the apps of its
	// previous enrollment, so replace the list instead of adding to it, before the
	// device is moved to that policy. Fail rather than continue without the Fleet
	// Agent, as replacing would remove it.
	fleetAgentPolicy, err := v.AndroidModule.BuildFleetAgentApplicationPolicy(ctx, hostUUID)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "build fleet agent application policy for host %s", hostUUID)
	}
	if fleetAgentPolicy != nil {
		appPolicies = append(appPolicies, fleetAgentPolicy)
	}

	policyRequests, err := v.AndroidModule.SetAppsForAndroidPolicy(ctx, enterpriseName, appPolicies, map[string]string{hostUUID: hostUUID})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "set apps for android policy")
	}

	var inPolicyVersion *int64
	policyRequest := policyRequests[hostUUID]
	if policyRequest != nil && policyRequest.PolicyVersion.Valid {
		inPolicyVersion = &policyRequest.PolicyVersion.V
	}
	for i := range installs {
		installs[i].InPolicyVersion = inPolicyVersion
	}
	installs = append(installs, failedInstalls...)
	if fleetAgentPolicy != nil {
		installs = append(installs, fleet.HostAndroidSoftwareInstall{HostID: hostID, AdamID: fleetAgentPolicy.PackageName, InstallType: android.AppStatusForceInstalled,
			Status: fleet.AndroidSoftwareInstallStatusPending, InPolicyVersion: inPolicyVersion, PolicyBuiltAt: policyBuiltAt})
	}
	err = v.Datastore.ReplaceHostAndroidSoftwareInstalls(ctx, hostID, installs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "replace host android software installs")
	}

	if onDefaultPolicy {
		if err := v.moveDeviceToHostSpecificPolicy(ctx, androidHost, enterpriseName); err != nil {
			return ctxerr.Wrapf(ctx, err, "moving device to host-specific policy for host %s", hostUUID)
		}
	}

	return nil
}

func (v *SoftwareWorker) runAndroidSetupExperience(ctx context.Context,
	hostUUID string, hostEnrollTeamID uint, enterpriseName string,
) error {
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
	flaggedVersions, err := v.Datastore.GetVPPAppsToInstallDuringSetupExperience(ctx, &hostEnrollTeamID, string(fleet.AndroidPlatform))
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "getting vpp apps to install during setup experience for team %d", hostEnrollTeamID)
	}
	// Install the first-added flagged version of each app, versions are ordered by id
	var setupApps []fleet.VPPAppTeam
	appIDs := make([]string, 0, len(flaggedVersions))
	vppAppTeamIDs := make([]uint, 0, len(flaggedVersions))
	for _, app := range flaggedVersions {
		if slices.Contains(appIDs, app.AdamID) {
			continue
		}
		setupApps = append(setupApps, app)
		appIDs = append(appIDs, app.AdamID)
		vppAppTeamIDs = append(vppAppTeamIDs, app.AppTeamID)
	}

	if len(appIDs) > 0 {
		// NOTE: from my tests, we do need to re-apply the app configs when installing apps,
		// even if they were already applied when making the apps available for self-service.
		// However, once installed, if the app config changes it is applied automatically by the
		// policy change (no need to re-install).
		policyBuiltAt := time.Now()
		configsByAppID, err := v.Datastore.BulkGetAndroidAppConfigurations(ctx, vppAppTeamIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "bulk get android app configurations")
		}

		configsByAppID, err = profiles.SubstituteFleetVarsInAndroidAppConfigs(ctx, v.Datastore, configsByAppID, androidHostToSubstitutionHost(host))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "substitute fleet vars in app configs")
		}

		appPolicies, err := profiles.BuildAndroidApplicationPolicies(ctx, appIDs, configsByAppID, android.AppStatusPreinstalled)
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
		for _, app := range setupApps {
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
				AdamID:            app.AdamID,
				CommandUUID:       uuid.NewString(),
				AssociatedEventID: fmt.Sprint(policyRequest.PolicyVersion.V),
				VPPAppTeamID:      app.AppTeamID,
			})
			if err != nil {
				return ctxerr.Wrap(ctx, err, "inserting android setup experience install request")
			}
		}

		var inPolicyVersion *int64
		if policyRequest != nil && policyRequest.PolicyVersion.Valid {
			inPolicyVersion = &policyRequest.PolicyVersion.V
		}
		installs := make([]fleet.HostAndroidSoftwareInstall, 0, len(setupApps))
		for _, app := range setupApps {
			installs = append(installs, fleet.HostAndroidSoftwareInstall{
				HostID:          host.Host.ID,
				AdamID:          app.AdamID,
				VPPAppTeamID:    &app.AppTeamID,
				InstallType:     android.AppStatusPreinstalled,
				Status:          fleet.AndroidSoftwareInstallStatusPending,
				InPolicyVersion: inPolicyVersion,
				PolicyBuiltAt:   policyBuiltAt,
			})
		}
		err = v.Datastore.BulkUpsertHostAndroidSoftwareInstalls(ctx, installs, nil)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "upsert host android software installs for setup experience")
		}
	}

	return nil
}

func androidHostToSubstitutionHost(h *fleet.AndroidHost) profiles.AndroidAppConfigSubstitutionHost {
	return profiles.AndroidAppConfigSubstitutionHost{
		HostID:         h.Host.ID,
		UUID:           h.Host.UUID,
		HardwareSerial: h.Host.HardwareSerial,
		Platform:       h.Host.Platform,
	}
}

func QueueRunAndroidSetupExperience(ctx context.Context, ds fleet.Datastore, logger *slog.Logger,
	hostUUID string, hostEnrollTeamID *uint, enterpriseName string,
) error {
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

	logger.DebugContext(ctx, "queued android setup experience job", "job_id", job.ID, "job_name", softwareWorkerJobName, "task", args.Task)
	return nil
}

func (v *SoftwareWorker) updateAndroidHostsAfterFleetTransfer(ctx context.Context, uuidsToIDs map[string]uint) error {
	hostIDs := make([]uint, 0, len(uuidsToIDs))
	for uuid, hostID := range uuidsToIDs {
		androidHost, err := v.Datastore.AndroidHostLiteByHostUUID(ctx, uuid)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "get android host by host UUID %s", uuid)
		}

		teamID := ptr.ValOrZero(androidHost.TeamID)

		// Update certificate templates for team transfer:
		// 1. Mark old templates as pending removal
		// 2. Create new pending templates for the new team
		if err := v.Datastore.SetHostCertificateTemplatesToPendingRemoveForHost(ctx, uuid); err != nil {
			return ctxerr.Wrap(ctx, err, "set host certificate templates to pending remove for host")
		}
		if _, err := v.Datastore.CreatePendingCertificateTemplatesForNewHost(ctx, uuid, teamID); err != nil {
			return ctxerr.Wrap(ctx, err, "create pending certificate templates for new host")
		}
		hostIDs = append(hostIDs, hostID)
	}

	// Flag the hosts' apps for resend, so the Android profile reconciler sends their new fleet's apps and agent configuration
	err := v.Datastore.SetHostAndroidSoftwareInstallsNeedsResend(ctx, hostIDs, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "flag android software installs for resend after fleet transfer")
	}
	return nil
}

const (
	androidSoftwareInstallStaggerInterval = 60 * time.Second
	defaultAndroidBatchSize               = 100
)

func QueueBulkSetAndroidAppsAvailableForHosts(
	ctx context.Context,
	ds fleet.Datastore,
	logger *slog.Logger,
	uuidsToIDs map[string]uint,
	enterpriseName string,
	batchSize int,
) error {
	if batchSize <= 0 || len(uuidsToIDs) <= batchSize {
		// No batching needed.
		return queueBulkSetAndroidAppsAvailableForHostsJob(ctx, ds, logger, uuidsToIDs, enterpriseName, 0)
	}

	// Chunk the host map into batches and stagger with not_before.
	batch := make(map[string]uint, batchSize)
	batchIdx := 0
	for uuid, id := range uuidsToIDs {
		batch[uuid] = id
		if len(batch) >= batchSize {
			delay := time.Duration(batchIdx) * androidSoftwareInstallStaggerInterval
			if err := queueBulkSetAndroidAppsAvailableForHostsJob(ctx, ds, logger, batch, enterpriseName, delay); err != nil {
				return err
			}
			batch = make(map[string]uint, batchSize)
			batchIdx++
		}
	}
	// Queue remaining hosts.
	if len(batch) > 0 {
		delay := time.Duration(batchIdx) * androidSoftwareInstallStaggerInterval
		if err := queueBulkSetAndroidAppsAvailableForHostsJob(ctx, ds, logger, batch, enterpriseName, delay); err != nil {
			return err
		}
	}
	return nil
}

func queueBulkSetAndroidAppsAvailableForHostsJob(
	ctx context.Context,
	ds fleet.Datastore,
	logger *slog.Logger,
	uuidsToIDs map[string]uint,
	enterpriseName string,
	delay time.Duration,
) error {
	args := &softwareWorkerArgs{
		Task:           bulkSetAndroidAppsAvailableForHostsTask,
		UUIDsToIDs:     uuidsToIDs,
		EnterpriseName: enterpriseName,
	}

	job, err := QueueJobWithDelay(ctx, ds, softwareWorkerJobName, args, delay)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}

	logger.DebugContext(ctx, "queued software worker job", "job_id", job.ID, "job_name", softwareWorkerJobName, "task", args.Task, "delay", delay, "hosts", len(uuidsToIDs))
	return nil
}
