package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
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
	"google.golang.org/api/googleapi"
)

const softwareWorkerJobName = "software_worker"

type SoftwareWorkerTask string

type SoftwareWorker struct {
	Datastore        fleet.Datastore
	AndroidModule    android.Service
	Log              *slog.Logger
	AndroidBatchSize int
	VPPInstaller     fleet.AppleMDMVPPInstaller
}

func (v *SoftwareWorker) Name() string {
	return softwareWorkerJobName
}

const (
	makeAndroidAppsAvailableForHostTask     SoftwareWorkerTask = "make_android_apps_available_for_host" // deprecated
	makeAndroidAppAvailableTask             SoftwareWorkerTask = "make_android_app_available"
	makeAndroidAppAvailableBatchTask        SoftwareWorkerTask = "make_android_app_available_batch"
	makeAndroidAppUnavailableTask           SoftwareWorkerTask = "make_android_app_unavailable"
	runAndroidSetupExperienceTask           SoftwareWorkerTask = "run_android_setup_experience"
	bulkSetAndroidAppsAvailableForHostTask  SoftwareWorkerTask = "bulk_set_android_apps_available_for_host"
	bulkSetAndroidAppsAvailableForHostsTask SoftwareWorkerTask = "bulk_set_android_apps_available_for_hosts"
	resendVPPAppConfigurationTask           SoftwareWorkerTask = "resend_vpp_app_configuration"
	resendVPPAppConfigurationBatchTask      SoftwareWorkerTask = "resend_vpp_app_configuration_batch"
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

	FleetID                 uint                            `json:"fleet_id,omitempty"`
	Platform                fleet.InstallableDevicePlatform `json:"platform,omitempty"`
	ConfigChangedAppTeamIDs []uint                          `json:"config_changed_app_team_ids,omitempty"` //nolint:apiparamcheck // "team" is from the vpp_apps_teams table, not a fleet
	VersionLabelsChanged    bool                            `json:"version_labels_changed,omitempty"`
	// HostIDs is empty for every host of the fleet.
	HostIDs []uint `json:"host_ids,omitempty"`
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

	case makeAndroidAppAvailableTask:
		return ctxerr.Wrapf(
			ctx,
			v.makeAndroidAppAvailable(ctx, args.ApplicationID, args.AppTeamID, args.EnterpriseName, args.AppConfigChanged, args.VersionLabelsChanged),
			"running %s task",
			makeAndroidAppAvailableTask,
		)

	case makeAndroidAppAvailableBatchTask:
		return ctxerr.Wrapf(
			ctx,
			v.makeAndroidAppAvailableBatch(ctx, args.ApplicationID, args.AppTeamID, args.HostUUIDToPolicyID, args.EnterpriseName, args.AppConfigChanged),
			"running %s task",
			makeAndroidAppAvailableBatchTask,
		)

	case makeAndroidAppUnavailableTask:
		return ctxerr.Wrapf(
			ctx,
			v.makeAndroidAppUnavailable(ctx, args.ApplicationID, args.HostUUIDToPolicyID, args.EnterpriseName),
			"running %s task",
			makeAndroidAppUnavailableTask,
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

	case resendVPPAppConfigurationTask:
		return ctxerr.Wrapf(ctx, v.resendVPPAppConfiguration(ctx, args), "running %s task", resendVPPAppConfigurationTask)

	case resendVPPAppConfigurationBatchTask:
		return ctxerr.Wrapf(ctx, v.resendVPPAppConfigurationBatch(ctx, args), "running %s task", resendVPPAppConfigurationBatchTask)

	default:
		return ctxerr.Errorf(ctx, "unknown task: %v", args.Task)

	}
}

// this is called when a new app is added to Fleet and when an existing app is updated
// (either its scope of affected hosts changed due to labels conditions, or its
// configuration changed).
func (v *SoftwareWorker) makeAndroidAppAvailable(ctx context.Context, applicationID string, appTeamID uint, enterpriseName string, appConfigChanged bool, versionLabelsChanged bool) error {
	versionIDs, err := v.Datastore.GetAppStoreAppVersionIDsFromSpecificVersion(ctx, appTeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "add app store app: getting versions of the app")
	}

	// Queue staggered batch jobs. The phase-2 handler handles per-host
	// variable substitution within each batch, so we always chunk the same way.
	batchSize := v.AndroidBatchSize
	if batchSize <= 0 {
		batchSize = defaultAndroidBatchSize
	}
	queuedHosts := make(map[string]struct{})
	var batchIndex int
	var hosts map[string]string
	for _, versionID := range versionIDs {
		hosts, err = v.Datastore.GetIncludedHostUUIDMapForAppStoreApp(ctx, versionID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "add app store app: getting android hosts in scope")
		}

		// Give each host the first-added version it is in scope for, versions are ordered by id
		versionHosts := make(map[string]string, len(hosts))
		for hostUUID, policyID := range hosts {
			if _, ok := queuedHosts[hostUUID]; ok {
				continue
			}
			queuedHosts[hostUUID] = struct{}{}
			versionHosts[hostUUID] = policyID
		}
		if len(versionHosts) == 0 {
			continue
		}
		// Push the hosts of every version when the edited version's labels changed, they can move hosts between versions
		if versionID != appTeamID && !versionLabelsChanged {
			continue
		}

		for _, batch := range splitHostMap(versionHosts, batchSize) {
			delay := time.Duration(batchIndex) * androidSoftwareInstallStaggerInterval
			err = queueMakeAndroidAppAvailableBatch(ctx, v.Datastore, applicationID, versionID, batch, enterpriseName, appConfigChanged, delay)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "queue batch for make android app available")
			}
			batchIndex++
		}
	}

	return nil
}

func (v *SoftwareWorker) makeAndroidAppAvailableBatch(ctx context.Context, applicationID string, appTeamID uint, hostUUIDToPolicyID map[string]string, enterpriseName string, appConfigChanged bool) error {
	config, err := v.Datastore.GetAndroidAppConfigurationByAppTeamID(ctx, appTeamID)
	if err != nil && !fleet.IsNotFound(err) {
		return ctxerr.Wrap(ctx, err, "get android app configuration")
	}
	var configByAppID map[string][]byte
	if config != nil {
		configByAppID = map[string][]byte{applicationID: config}
	}

	needsPerHostSubstitution := config != nil && profiles.ContainsFleetVarOrCustomHostVital(config)

	if needsPerHostSubstitution {
		hostUUIDs := make([]string, 0, len(hostUUIDToPolicyID))
		for uuid := range hostUUIDToPolicyID {
			hostUUIDs = append(hostUUIDs, uuid)
		}
		filter := fleet.TeamFilter{User: &fleet.User{GlobalRole: new("admin")}}
		hostDetails, err := v.Datastore.ListHostsLiteByUUIDs(ctx, filter, hostUUIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "batch fetch host details for variable substitution")
		}
		hostByUUID := make(map[string]*fleet.Host, len(hostDetails))
		for _, h := range hostDetails {
			hostByUUID[h.UUID] = h
		}

		for hostUUID := range hostUUIDToPolicyID {
			h, ok := hostByUUID[hostUUID]
			if !ok {
				continue // host deleted since the job was queued
			}

			subHost := profiles.AndroidAppConfigSubstitutionHost{
				HostID:         h.ID,
				UUID:           h.UUID,
				HardwareSerial: h.HardwareSerial,
				Platform:       h.Platform,
			}
			substituted, err := v.substituteFleetVarsInConfigs(ctx, configByAppID, subHost)
			if err != nil {
				// One host that can't supply a referenced value must not block the
				// rest of the batch, nor burn the job's retries on something no
				// retry can fix. Whichever value is missing has its own resend
				// trigger for when it lands.
				if isUnresolvableAndroidAppConfigForHost(err) {
					v.Log.WarnContext(ctx, "skipping android app config for host that can't supply a referenced value",
						"host_uuid", hostUUID, "application_id", applicationID, "err", err)
					continue
				}
				return ctxerr.Wrapf(ctx, err, "substitute fleet vars for host %s", hostUUID)
			}

			appPolicies, err := buildApplicationPolicyWithConfig(ctx, []string{applicationID}, substituted, "AVAILABLE")
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "building application policies with config for host %s", hostUUID)
			}

			singleHost := map[string]string{hostUUID: hostUUIDToPolicyID[hostUUID]}
			policyRequestsByHost, err := v.AndroidModule.AddAppsToAndroidPolicy(ctx, enterpriseName, appPolicies, singleHost)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "add app to android policy for host %s", hostUUID)
			}

			if appConfigChanged {
				for uuid, policyRequest := range policyRequestsByHost {
					if err := v.Datastore.SetAndroidAppInstallPendingApplyConfig(ctx, uuid, applicationID, policyRequest.PolicyVersion.V); err != nil {
						return ctxerr.Wrapf(ctx, err, "set android app install pending apply config for host %s and app %s", uuid, applicationID)
					}
				}
			}
		}
	} else {
		appPolicies, err := buildApplicationPolicyWithConfig(ctx, []string{applicationID}, configByAppID, "AVAILABLE")
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building application policies with config")
		}

		policyRequestsByHost, err := v.AndroidModule.AddAppsToAndroidPolicy(ctx, enterpriseName, appPolicies, hostUUIDToPolicyID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "add app store app: add app to android policy")
		}

		if appConfigChanged {
			for hostUUID, policyRequest := range policyRequestsByHost {
				if err := v.Datastore.SetAndroidAppInstallPendingApplyConfig(ctx, hostUUID, applicationID, policyRequest.PolicyVersion.V); err != nil {
					return ctxerr.Wrapf(ctx, err, "set android app install pending apply config for host %s and app %s", hostUUID, applicationID)
				}
			}
		}
	}

	return nil
}

func queueMakeAndroidAppAvailableBatch(ctx context.Context, ds fleet.Datastore, applicationID string, appTeamID uint, hostUUIDToPolicyID map[string]string, enterpriseName string, appConfigChanged bool, delay time.Duration) error {
	args := &softwareWorkerArgs{
		Task:               makeAndroidAppAvailableBatchTask,
		ApplicationID:      applicationID,
		AppTeamID:          appTeamID,
		HostUUIDToPolicyID: hostUUIDToPolicyID,
		EnterpriseName:     enterpriseName,
		AppConfigChanged:   appConfigChanged,
	}
	_, err := QueueJobWithDelay(ctx, ds, softwareWorkerJobName, args, delay)
	return err
}

// this is called when an app is removed from Fleet.
func (v *SoftwareWorker) makeAndroidAppUnavailable(ctx context.Context, applicationID string, hostUUIDToPolicyID map[string]string, enterpriseName string) error {
	// Update Android MDM policy to remove the app from the hosts
	_, err := v.AndroidModule.RemoveAppsFromAndroidPolicy(ctx, enterpriseName, []string{applicationID}, hostUUIDToPolicyID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "add app store app: add app to android policy")
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

	appsInScope, err := v.Datastore.GetAndroidAppsInScopeForHost(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get android apps in scope for host")
	}

	if len(appsInScope) == 0 {
		return nil
	}

	appIDs := make([]string, 0, len(appsInScope))
	vppAppTeamIDs := make([]uint, 0, len(appsInScope))
	for _, app := range appsInScope {
		appIDs = append(appIDs, app.AdamID)
		vppAppTeamIDs = append(vppAppTeamIDs, app.AppTeamID)
	}

	configsByAppID, err := v.Datastore.BulkGetAndroidAppConfigurations(ctx, vppAppTeamIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "bulk get android app configurations")
	}

	configsByAppID, err = v.substituteFleetVarsInConfigs(ctx, configsByAppID, androidHostToSubstitutionHost(androidHost))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "substitute fleet vars in app configs")
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
		configsByAppID, err := v.Datastore.BulkGetAndroidAppConfigurations(ctx, vppAppTeamIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "bulk get android app configurations")
		}

		configsByAppID, err = v.substituteFleetVarsInConfigs(ctx, configsByAppID, androidHostToSubstitutionHost(host))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "substitute fleet vars in app configs")
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
	}

	return nil
}

func (v *SoftwareWorker) bulkMakeAndroidAppsAvailableForHost(ctx context.Context, hostUUID, policyID string, applicationIDs []string, enterpriseName string) error {
	host, err := v.Datastore.AndroidHostLiteByHostUUID(ctx, hostUUID)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "getting android host lite by uuid %s", hostUUID)
	}

	appsInScope, err := v.Datastore.GetAndroidAppsInScopeForHost(ctx, host.Host.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get android apps in scope for host")
	}
	var vppAppTeamIDs []uint
	for _, app := range appsInScope {
		if slices.Contains(applicationIDs, app.AdamID) {
			vppAppTeamIDs = append(vppAppTeamIDs, app.AppTeamID)
		}
	}

	configsByAppID, err := v.Datastore.BulkGetAndroidAppConfigurations(ctx, vppAppTeamIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "bulk get android app configurations")
	}

	configsByAppID, err = v.substituteFleetVarsInConfigs(ctx, configsByAppID, androidHostToSubstitutionHost(host))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "substitute fleet vars in app configs")
	}

	appPolicies, err := buildApplicationPolicyWithConfig(ctx, applicationIDs, configsByAppID, "AVAILABLE")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building application policies with config")
	}

	// Include the Fleet Agent in the app list so it's not removed when we replace the apps.
	fleetAgentPolicy, err := v.AndroidModule.BuildFleetAgentApplicationPolicy(ctx, hostUUID)
	if err != nil {
		v.Log.ErrorContext(ctx, "failed to build Fleet Agent policy, Fleet Agent may be removed", "host_uuid", hostUUID, "err", err)
	} else if fleetAgentPolicy != nil {
		appPolicies = append(appPolicies, fleetAgentPolicy)
	}

	// Update Android MDM policy to include the apps in self service
	err = v.AndroidModule.SetAppsForAndroidPolicy(ctx, enterpriseName, appPolicies, map[string]string{hostUUID: policyID})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "make android apps available")
	}

	return nil
}

func buildApplicationPolicyWithConfig(ctx context.Context, appIDs []string,
	configsByAppID map[string][]byte, installType string,
) ([]*androidmanagement.ApplicationPolicy, error) {
	appPolicies := make([]*androidmanagement.ApplicationPolicy, 0, len(appIDs))
	for _, appID := range appIDs {
		appPolicy := &androidmanagement.ApplicationPolicy{}
		if config := configsByAppID[appID]; config != nil {
			if err := json.Unmarshal(config, appPolicy); err != nil {
				// should never happen, as it is stored as json in the db and is pre-validated
				return nil, ctxerr.Wrap(ctx, err, "unmarshal android app configuration")
			}
		} else {
			// if there is no config for this app, we must make sure we clear any previously-applied
			// config.
			appPolicy.ManagedConfiguration = googleapi.RawMessage{}
			appPolicy.WorkProfileWidgets = "WORK_PROFILE_WIDGETS_UNSPECIFIED"
			appPolicy.CredentialProviderPolicy = "CREDENTIAL_PROVIDER_POLICY_UNSPECIFIED"
		}
		// Set after unmarshalling so a stored config can never override them.
		appPolicy.PackageName = appID
		appPolicy.InstallType = installType
		appPolicies = append(appPolicies, appPolicy)
	}
	return appPolicies, nil
}

func (v *SoftwareWorker) substituteFleetVarsInConfigs(
	ctx context.Context,
	configsByAppID map[string][]byte,
	host profiles.AndroidAppConfigSubstitutionHost,
) (map[string][]byte, error) {
	if len(configsByAppID) == 0 {
		return configsByAppID, nil
	}

	hasVars := false
	for _, cfg := range configsByAppID {
		if profiles.ContainsFleetVarOrCustomHostVital(cfg) {
			hasVars = true
			break
		}
	}
	if !hasVars {
		return configsByAppID, nil
	}

	result := make(map[string][]byte, len(configsByAppID))
	for appID, cfg := range configsByAppID {
		substituted, err := profiles.SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, v.Datastore, cfg, host)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "substitute fleet vars in android app config for app %s", appID)
		}
		result[appID] = substituted
	}
	return result, nil
}

// isUnresolvableAndroidAppConfigForHost separates "this host can't supply a
// referenced value" — a custom host vital with no value set for this host, or
// an unresolvable $FLEET_VAR_* — from a real failure. The former is host state
// no retry can fix, so it's a per-host skip rather than a batch failure.
func isUnresolvableAndroidAppConfigForHost(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := errors.AsType[*fleet.MissingCustomHostVitalValueError](err); ok {
		return true
	}
	return errors.Is(err, profiles.ErrUnresolvableAndroidAppConfigVar)
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

func QueueMakeAndroidAppAvailableJob(ctx context.Context, ds fleet.Datastore, logger *slog.Logger, applicationID string, appTeamID uint, enterpriseName string, appConfigChanged bool, versionLabelsChanged bool) error {
	args := &softwareWorkerArgs{
		Task:                 makeAndroidAppAvailableTask,
		ApplicationID:        applicationID,
		AppTeamID:            appTeamID,
		EnterpriseName:       enterpriseName,
		AppConfigChanged:     appConfigChanged,
		VersionLabelsChanged: versionLabelsChanged,
	}

	job, err := QueueJob(ctx, ds, softwareWorkerJobName, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}

	logger.DebugContext(ctx, "queued software worker job", "job_id", job.ID, "job_name", softwareWorkerJobName, "task", args.Task)
	return nil
}

func QueueMakeAndroidAppUnavailableJob(ctx context.Context, ds fleet.Datastore, logger *slog.Logger, applicationID string, hostsUUIDToPolicyID map[string]string, enterpriseName string, batchSize int) error {
	if batchSize <= 0 || len(hostsUUIDToPolicyID) <= batchSize {
		return queueMakeAndroidAppUnavailableJob(ctx, ds, logger, applicationID, hostsUUIDToPolicyID, enterpriseName, 0)
	}

	batch := make(map[string]string, batchSize)
	batchIdx := 0
	for uuid, policyID := range hostsUUIDToPolicyID {
		batch[uuid] = policyID
		if len(batch) >= batchSize {
			delay := time.Duration(batchIdx) * androidSoftwareInstallStaggerInterval
			if err := queueMakeAndroidAppUnavailableJob(ctx, ds, logger, applicationID, batch, enterpriseName, delay); err != nil {
				return err
			}
			batch = make(map[string]string, batchSize)
			batchIdx++
		}
	}
	if len(batch) > 0 {
		delay := time.Duration(batchIdx) * androidSoftwareInstallStaggerInterval
		if err := queueMakeAndroidAppUnavailableJob(ctx, ds, logger, applicationID, batch, enterpriseName, delay); err != nil {
			return err
		}
	}
	return nil
}

func queueMakeAndroidAppUnavailableJob(ctx context.Context, ds fleet.Datastore, logger *slog.Logger, applicationID string, hostsUUIDToPolicyID map[string]string, enterpriseName string, delay time.Duration) error {
	args := &softwareWorkerArgs{
		Task:               makeAndroidAppUnavailableTask,
		ApplicationID:      applicationID,
		HostUUIDToPolicyID: hostsUUIDToPolicyID,
		EnterpriseName:     enterpriseName,
	}

	job, err := QueueJobWithDelay(ctx, ds, softwareWorkerJobName, args, delay)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}

	logger.DebugContext(ctx, "queued software worker job", "job_id", job.ID, "job_name", softwareWorkerJobName, "task", args.Task, "delay", delay, "hosts", len(hostsUUIDToPolicyID))
	return nil
}

func QueueBulkSetAndroidAppsAvailableForHost(
	ctx context.Context,
	ds fleet.Datastore,
	logger *slog.Logger,
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

	logger.DebugContext(ctx, "queued software worker job", "job_id", job.ID, "job_name", softwareWorkerJobName, "task", args.Task)
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

		teamID := ptr.ValOrZero(androidHost.TeamID)

		// Update certificate templates for team transfer:
		// 1. Mark old templates as pending removal
		// 2. Create new pending templates for the new team
		// This must happen before building the managed config, which includes certificate template IDs.
		if err := v.Datastore.SetHostCertificateTemplatesToPendingRemoveForHost(ctx, uuid); err != nil {
			return ctxerr.Wrap(ctx, err, "set host certificate templates to pending remove for host")
		}
		if _, err := v.Datastore.CreatePendingCertificateTemplatesForNewHost(ctx, uuid, teamID); err != nil {
			return ctxerr.Wrap(ctx, err, "create pending certificate templates for new host")
		}

		appsInScope, err := v.Datastore.GetAndroidAppsInScopeForHost(ctx, hostID)
		if err != nil {
			return ctxerr.WrapWithData(ctx, err, "get android apps in scope for host", map[string]any{"host_id": hostID})
		}

		appIDs := make([]string, 0, len(appsInScope))
		vppAppTeamIDs := make([]uint, 0, len(appsInScope))
		for _, app := range appsInScope {
			appIDs = append(appIDs, app.AdamID)
			vppAppTeamIDs = append(vppAppTeamIDs, app.AppTeamID)
		}

		configsByAppID, err := v.Datastore.BulkGetAndroidAppConfigurations(ctx, vppAppTeamIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "bulk get android app configurations")
		}

		configsByAppID, err = v.substituteFleetVarsInConfigs(ctx, configsByAppID, androidHostToSubstitutionHost(androidHost))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "substitute fleet vars in app configs")
		}

		appPolicies, err := buildApplicationPolicyWithConfig(ctx, appIDs, configsByAppID, "AVAILABLE")
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building application policies with config")
		}

		// Include the Fleet Agent in the app list so it's not removed when we replace the apps.
		fleetAgentPolicy, err := v.AndroidModule.BuildFleetAgentApplicationPolicy(ctx, uuid)
		if err != nil {
			v.Log.ErrorContext(ctx, "failed to build Fleet Agent policy, Fleet Agent may be removed", "host_uuid", uuid, "err", err)
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

func splitHostMap(hosts map[string]string, batchSize int) []map[string]string {
	if batchSize <= 0 || len(hosts) <= batchSize {
		return []map[string]string{hosts}
	}
	var batches []map[string]string
	batch := make(map[string]string, batchSize)
	for k, v := range hosts {
		batch[k] = v
		if len(batch) >= batchSize {
			batches = append(batches, batch)
			batch = make(map[string]string, batchSize)
		}
	}
	if len(batch) > 0 {
		batches = append(batches, batch)
	}
	return batches
}

func QueueMakeAndroidAppAvailableForHostsJob(ctx context.Context, ds fleet.Datastore, applicationID string, appTeamID uint, hostUUIDToPolicyID map[string]string, enterpriseName string, batchSize int) error {
	if batchSize <= 0 {
		batchSize = defaultAndroidBatchSize
	}
	for batchIndex, batch := range splitHostMap(hostUUIDToPolicyID, batchSize) {
		delay := time.Duration(batchIndex) * androidSoftwareInstallStaggerInterval
		err := queueMakeAndroidAppAvailableBatch(ctx, ds, applicationID, appTeamID, batch, enterpriseName, true, delay)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "queue batch for make android app available for hosts")
		}
	}
	return nil
}

func QueueResendVPPAppConfigurationJob(ctx context.Context, ds fleet.Datastore, logger *slog.Logger, appID fleet.VPPAppID, fleetID uint, configChangedAppTeamIDs []uint, versionLabelsChanged bool, hostIDs []uint) error {
	args := &softwareWorkerArgs{
		Task:                    resendVPPAppConfigurationTask,
		ApplicationID:           appID.AdamID,
		Platform:                appID.Platform,
		FleetID:                 fleetID,
		ConfigChangedAppTeamIDs: configChangedAppTeamIDs,
		VersionLabelsChanged:    versionLabelsChanged,
		HostIDs:                 hostIDs,
	}
	job, err := QueueJob(ctx, ds, softwareWorkerJobName, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}

	logger.DebugContext(ctx, "queued software worker job", "job_id", job.ID, "job_name", softwareWorkerJobName, "task", args.Task)
	return nil
}

func (v *SoftwareWorker) resendVPPAppConfiguration(ctx context.Context, args softwareWorkerArgs) error {
	// This task only queues installs. Leave hosts alone when every version of the app is deleted or the host is in
	// scope for no version, a configuration re-send never removes the app from a host.
	app, err := v.Datastore.GetVPPAppMetadataByAdamIDPlatformTeamID(ctx, args.ApplicationID, args.Platform, &args.FleetID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil
		}
		return ctxerr.Wrap(ctx, err, "get vpp app for configuration resend")
	}
	versions, err := v.Datastore.GetAppStoreAppVersionsByTeamAndTitleID(ctx, args.FleetID, app.TitleID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil
		}
		return ctxerr.Wrap(ctx, err, "get vpp app versions for configuration resend")
	}
	installedVersionByHostID, err := v.Datastore.ListHostAppStoreAppInstallVersions(ctx, app.VPPAppID, args.FleetID, args.HostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list hosts with app store app installs")
	}

	installedHostIDs := slices.Collect(maps.Keys(installedVersionByHostID))

	// Give each host the first version that includes it, versions are ordered by id. The batches install the version picked here.
	hostIDsByVersionID := make(map[uint][]uint, len(versions))
	assignedHostIDs := make(map[uint]struct{}, len(installedVersionByHostID))
	var includedHostIDs map[uint]struct{}
	for _, version := range versions {
		includedHostIDs, err = v.Datastore.GetIncludedHostIDMapForVPPAppHosts(ctx, version.VPPAppsTeamsID, installedHostIDs)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "get hosts in scope of vpp app version %d", version.VPPAppsTeamsID)
		}
		for hostID, installedVersionID := range installedVersionByHostID {
			_, assigned := assignedHostIDs[hostID]
			if assigned {
				continue
			}
			_, included := includedHostIDs[hostID]
			if !included {
				continue
			}
			assignedHostIDs[hostID] = struct{}{}

			switch {
			// Re-send to hosts whose version's configuration changed
			case slices.Contains(args.ConfigChangedAppTeamIDs, version.VPPAppsTeamsID):
			// Re-send to hosts whose label scoping now gives them a different version than the one installed
			case args.VersionLabelsChanged && !ptr.Equal(installedVersionID, &version.VPPAppsTeamsID):
			// Re-send to hosts whose installed version was deleted and whose label scoping gives them another version
			case len(args.ConfigChangedAppTeamIDs) == 0 && installedVersionID == nil:
			default:
				continue
			}
			hostIDsByVersionID[version.VPPAppsTeamsID] = append(hostIDsByVersionID[version.VPPAppsTeamsID], hostID)
		}
	}

	// Queue staggered batch jobs per version so each job installs one version on a bounded number of hosts
	var batchIndex int
	for _, version := range versions {
		hostIDs := hostIDsByVersionID[version.VPPAppsTeamsID]
		slices.Sort(hostIDs)
		for batchHostIDs := range slices.Chunk(hostIDs, defaultAndroidBatchSize) {
			batchArgs := args
			batchArgs.Task = resendVPPAppConfigurationBatchTask
			batchArgs.AppTeamID = version.VPPAppsTeamsID
			batchArgs.HostIDs = batchHostIDs
			delay := time.Duration(batchIndex) * androidSoftwareInstallStaggerInterval
			_, err = QueueJobWithDelay(ctx, v.Datastore, softwareWorkerJobName, &batchArgs, delay)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "queue batch for resend vpp app configuration")
			}
			batchIndex++
		}
	}
	return nil
}

func (v *SoftwareWorker) resendVPPAppConfigurationBatch(ctx context.Context, args softwareWorkerArgs) error {
	if v.VPPInstaller == nil {
		// should be unreachable
		return errors.New("VPP installer not configured")
	}

	// Skip the batch when its version was deleted after it was queued, a configuration re-send never removes the app
	app, err := v.Datastore.GetVPPAppMetadataByAdamIDPlatformTeamID(ctx, args.ApplicationID, args.Platform, &args.FleetID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil
		}
		return ctxerr.Wrap(ctx, err, "get vpp app for configuration resend")
	}
	vppApp, err := v.Datastore.GetVPPAppByTeamAndTitleID(ctx, &args.FleetID, app.TitleID, args.AppTeamID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil
		}
		return ctxerr.Wrapf(ctx, err, "get vpp app version %d", args.AppTeamID)
	}

	// Skip hosts with an install of this version that hasn't activated yet, it reads the configuration when it activates.
	// A retry of this batch skips the hosts it already queued, unless their install activated in between, which only
	// happens when activity.fleet_initiated_release_per_minute is 0.
	hostIDsWithUnactivatedInstall, err := v.Datastore.GetHostIDsWithUnactivatedVPPAppInstall(ctx, args.AppTeamID, args.HostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get hosts with unactivated vpp app install")
	}

	var retryHostIDs []uint
	var host *fleet.Host
	var token string
	for _, hostID := range args.HostIDs {
		_, installUnactivated := hostIDsWithUnactivatedInstall[hostID]
		if installUnactivated {
			continue
		}
		host, err = v.Datastore.Host(ctx, hostID)
		if err != nil {
			if fleet.IsNotFound(err) {
				continue
			}
			return ctxerr.Wrapf(ctx, err, "get host %d", hostID)
		}
		// Skip hosts that left the fleet after the batch was queued
		if ptr.ValOrZero(host.TeamID) != args.FleetID {
			continue
		}

		token, err = v.VPPInstaller.GetVPPTokenIfCanInstallVPPApps(ctx, true, host)
		if err == nil {
			_, err = v.VPPInstaller.InstallVPPAppPostValidation(ctx, host, vppApp, token, fleet.HostSoftwareInstallOptions{
				ForConfigurationResend: true,
			})
		}
		if err != nil {
			// Skip hosts that can't take the install, such as hosts without MDM or a license, and retry the batch for
			// other failures, such as an error from Apple
			var clientErr fleet.ErrWithIsClientError
			if errors.As(err, &clientErr) && clientErr.IsClientError() {
				v.Log.WarnContext(ctx, "skipping vpp app configuration resend for host", "host_id", hostID, "adam_id", app.AdamID, "err", err)
				continue
			}
			v.Log.ErrorContext(ctx, "resend vpp app configuration", "host_id", hostID, "adam_id", app.AdamID, "err", err)
			retryHostIDs = append(retryHostIDs, hostID)
		}
	}
	if len(retryHostIDs) > 0 {
		return ctxerr.Errorf(ctx, "resend vpp app configuration failed for hosts %v", retryHostIDs)
	}
	return nil
}
