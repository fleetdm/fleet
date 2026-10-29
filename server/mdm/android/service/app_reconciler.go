package service

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"maps"
	"math"
	"net/http"
	"slices"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/fleetdm/fleet/v4/server/mdm/profiles"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"google.golang.org/api/androidmanagement/v1"
)

// androidAppList is the app list built for a host whose apps or Fleet agent changed, with the installs to save once
// it's sent.
type androidAppList struct {
	host                                        *fleet.Host
	applicationsWithoutAgent                    []*androidmanagement.ApplicationPolicy
	installsToUpsert                            []fleet.HostAndroidSoftwareInstall
	installsToRemove                            []fleet.HostAndroidSoftwareInstall
	setupExperienceAppStoreIDsWithChangedConfig []string
	retriesFailedRequest                        bool
}

func (r *profileReconciler) prepareAppLists(ctx context.Context, batchSize int, policyBuiltAt time.Time) (map[string]*androidAppList, error) {
	// Check at most batchSize hosts per run, each costs at most one AMAPI request, so a large backlog is spread over
	// runs instead of using the shared quota at once. A batch size of 0 checks every host.
	limit := batchSize
	if limit <= 0 {
		limit = math.MaxInt32
	}
	hosts, err := r.DS.ListAndroidHostsOnHostPolicyAfterID(ctx, r.appCursor, limit)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list android hosts")
	}

	// Continue after this batch on the next run, or start from the first host again after the last batch
	r.appNextCursor = 0
	if batchSize > 0 && len(hosts) == batchSize {
		r.appNextCursor = hosts[len(hosts)-1].ID
	}

	hostIDs := make([]uint, 0, len(hosts))
	for _, host := range hosts {
		hostIDs = append(hostIDs, host.ID)
	}

	setupApps, err := r.DS.ListHostMDMAndroidVPPAppsFromSetupExperience(ctx, hostIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list android setup experience apps")
	}
	setupAppsByHostID := make(map[uint][]*fleet.HostAndroidVPPSoftwareInstall, len(hosts))
	for _, setupApp := range setupApps {
		setupAppsByHostID[setupApp.HostID] = append(setupAppsByHostID[setupApp.HostID], setupApp)
	}
	installs, err := r.DS.ListHostAndroidSoftwareInstalls(ctx, hostIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list host android software installs")
	}
	installByAppStoreIDByHostID := make(map[uint]map[string]*fleet.HostAndroidSoftwareInstall, len(hosts))
	for _, install := range installs {
		if installByAppStoreIDByHostID[install.HostID] == nil {
			installByAppStoreIDByHostID[install.HostID] = make(map[string]*fleet.HostAndroidSoftwareInstall)
		}
		installByAppStoreIDByHostID[install.HostID][install.AdamID] = install
	}

	// Read every fleet's versions with their configurations, so a host moved from another fleet gets a send only when the
	// configuration differs. Read them from the primary, after policyBuiltAt, so an edit a replica doesn't have yet isn't
	// saved as sent.
	versionsByTitleIDByFleetID, err := r.DS.GetAppStoreAppVersionsByFleetAndTitle(ctxdb.RequirePrimary(ctx, true), []fleet.InstallableDevicePlatform{fleet.AndroidPlatform})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get app store app versions")
	}
	versionByID := make(map[uint]*fleet.VPPAppStoreApp)
	for _, versionsByTitleID := range versionsByTitleIDByFleetID {
		for _, versions := range versionsByTitleID {
			for _, version := range versions {
				versionByID[version.VPPAppsTeamsID] = version
			}
		}
	}

	appListsByHostUUID := make(map[string]*androidAppList)
	var appList *androidAppList
	for _, host := range hosts {
		appList, err = r.prepareHostAppList(ctx, host, installByAppStoreIDByHostID[host.ID], setupAppsByHostID[host.ID], versionByID, policyBuiltAt)
		if err != nil {
			r.Logger.ErrorContext(ctx, "failed to prepare android app list for host", "host_id", host.ID, "err", err)
			continue
		}
		if appList != nil {
			appListsByHostUUID[host.UUID] = appList
		}
	}
	return appListsByHostUUID, nil
}

func (r *profileReconciler) prepareHostAppList(ctx context.Context, host *fleet.Host, installByAppStoreID map[string]*fleet.HostAndroidSoftwareInstall, setupApps []*fleet.HostAndroidVPPSoftwareInstall, versionByID map[uint]*fleet.VPPAppStoreApp, policyBuiltAt time.Time) (*androidAppList, error) {
	appsInList, err := r.DS.GetAndroidAppsInScopeForHost(ctx, host.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get android apps in scope for host")
	}

	// Keep the apps the setup experience installs as PREINSTALLED, also while the device installs them and when their
	// labels leave the host out of scope, so a replace doesn't stop their install or uninstall them. The setup experience
	// doesn't check labels yet.
	installTypeByAppStoreID := make(map[string]string, len(appsInList)+len(setupApps))
	for _, app := range appsInList {
		installTypeByAppStoreID[app.AdamID] = android.AppStatusAvailable
	}
	for _, setupApp := range setupApps {
		if _, inList := installTypeByAppStoreID[setupApp.AdamID]; !inList {
			appsInList = append(appsInList, fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: setupApp.AdamID}, AppTeamID: setupApp.VPPAppTeamID})
		}
		installTypeByAppStoreID[setupApp.AdamID] = android.AppStatusPreinstalled
	}
	// Count the Fleet agent as in scope, every list includes it
	agentPackage := r.AndroidAgentConfig.Package
	if agentPackage != "" {
		installTypeByAppStoreID[agentPackage] = android.AppStatusForceInstalled
	}

	// Retry a failed request after a wait, and send any other change right away
	var needsSend bool
	var needsRetry bool
	var lastFailedAt time.Time
	var installsToRemove []fleet.HostAndroidSoftwareInstall
	for appStoreID, install := range installByAppStoreID {
		if install.Status == fleet.AndroidSoftwareInstallStatusRequestFailed {
			needsRetry = true
			if install.UpdatedAt.After(lastFailedAt) {
				lastFailedAt = install.UpdatedAt
			}
		}
		if _, inList := installTypeByAppStoreID[appStoreID]; inList {
			continue
		}
		installsToRemove = append(installsToRemove, *install)
		if install.Status != fleet.AndroidSoftwareInstallStatusRequestFailed {
			needsSend = true
		}
	}
	newVersionIDByAppStoreID := make(map[string]uint)
	changedAppStoreIDs := make(map[string]struct{})
	configChangedAppStoreIDs := make(map[string]struct{})
	for _, app := range appsInList {
		install, sent := installByAppStoreID[app.AdamID]
		version := versionByID[app.AppTeamID]
		switch {
		case !sent:
			needsSend = true
		case install.NeedsResend, version != nil && version.ConfigurationUpdatedAt != nil && install.PolicyBuiltAt.Before(*version.ConfigurationUpdatedAt):
			needsSend = true
			configChangedAppStoreIDs[app.AdamID] = struct{}{}
		case !ptr.Equal(install.VPPAppTeamID, &app.AppTeamID):
			// Update only the row's version when the host moves to a version with the same configuration, the app entry
			// in the policy is the same
			var sentVersion *fleet.VPPAppStoreApp
			if install.VPPAppTeamID != nil {
				sentVersion = versionByID[*install.VPPAppTeamID]
			}
			if sentVersion != nil && version != nil && bytes.Equal(sentVersion.Configuration, version.Configuration) &&
				(sentVersion.ConfigurationUpdatedAt == nil || sentVersion.ConfigurationUpdatedAt.Before(install.PolicyBuiltAt)) {
				newVersionIDByAppStoreID[app.AdamID] = app.AppTeamID
			} else {
				needsSend = true
				configChangedAppStoreIDs[app.AdamID] = struct{}{}
			}
		case install.InstallType != installTypeByAppStoreID[app.AdamID]:
			needsSend = true
		case install.Status == fleet.AndroidSoftwareInstallStatusRequestFailed:
			// Sent with the retry below
		default:
			continue
		}
		changedAppStoreIDs[app.AdamID] = struct{}{}
	}
	// Send a list to a host whose installs don't include the agent, it never got a list from Fleet, so its policy can
	// still have apps that older jobs didn't remove. Send it also when the agent's configuration changed.
	agentInstall, agentSent := installByAppStoreID[agentPackage]
	if agentPackage != "" && (!agentSent || agentInstall.NeedsResend) {
		needsSend = true
	}

	// Retry a failed request after 1 hour, at most MaxVPPAutomaticInstallRetries times until the count expires, like the
	// App Store app configuration reconciler. Count retries per host, a request sends the host's whole app list.
	var retriesFailedRequest bool
	if !needsSend && needsRetry && time.Since(lastFailedAt) >= fleet.VPPAutomaticInstallWaitSeconds*time.Second {
		var retries int
		if r.RetryCounter != nil {
			retries, err = r.RetryCounter.CountAttempts(ctx, host.ID, 0)
			if err != nil {
				r.Logger.WarnContext(ctx, "failed to count android app list retries", "host_id", host.ID, "err", err)
			}
		}
		if retries < fleet.MaxVPPAutomaticInstallRetries {
			needsSend = true
			retriesFailedRequest = true
		}
	}
	if !needsSend {
		for appStoreID, newVersionID := range newVersionIDByAppStoreID {
			updatedInstall := *installByAppStoreID[appStoreID]
			updatedInstall.VPPAppTeamID = &newVersionID
			r.appInstallsToUpsert = append(r.appInstallsToUpsert, updatedInstall)
		}
		return nil, nil
	}

	configsByAppID := make(map[string][]byte, len(appsInList))
	for _, app := range appsInList {
		version := versionByID[app.AppTeamID]
		if version == nil {
			// Check the host again on the next run, its app version was added after the versions were read
			return nil, nil
		}
		if len(version.Configuration) > 0 {
			configsByAppID[app.AdamID] = version.Configuration
		}
	}

	// Leave out an app whose configuration uses a value the host doesn't have, it's sent once a change to that value flags
	// it for resend
	substitutionHost := profiles.AndroidAppConfigSubstitutionHost{
		HostID:         host.ID,
		UUID:           host.UUID,
		HardwareSerial: host.HardwareSerial,
		Platform:       host.Platform,
	}
	var availableAppStoreIDs []string
	var preinstalledAppStoreIDs []string
	// Save only the apps that changed, an unchanged app's entry in the new policy is the same as in the policy of its
	// existing install
	var installsToUpsert []fleet.HostAndroidSoftwareInstall
	var failedInstalls []fleet.HostAndroidSoftwareInstall
	var substituted []byte
	for _, app := range appsInList {
		_, changed := changedAppStoreIDs[app.AdamID]
		install := fleet.HostAndroidSoftwareInstall{HostID: host.ID, AdamID: app.AdamID, VPPAppTeamID: new(app.AppTeamID), InstallType: installTypeByAppStoreID[app.AdamID],
			Status: fleet.AndroidSoftwareInstallStatusPending, PolicyBuiltAt: policyBuiltAt}
		config, configured := configsByAppID[app.AdamID]
		if configured {
			substituted, err = profiles.SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, r.DS, config, substitutionHost)
			if err != nil {
				if !profiles.IsUnresolvableAndroidAppConfigForHost(err) {
					return nil, ctxerr.Wrapf(ctx, err, "substitute fleet vars in android app config for app %s", app.AdamID)
				}
				r.Logger.WarnContext(ctx, "skipping android app for host that can't supply a value referenced by its config",
					"host_id", host.ID, "application_id", app.AdamID, "err", err)
				if changed {
					install.Status = fleet.AndroidSoftwareInstallStatusFailed
					failedInstalls = append(failedInstalls, install)
				}
				continue
			}
			configsByAppID[app.AdamID] = substituted
		}
		if changed {
			installsToUpsert = append(installsToUpsert, install)
		}
		if installTypeByAppStoreID[app.AdamID] == android.AppStatusPreinstalled {
			preinstalledAppStoreIDs = append(preinstalledAppStoreIDs, app.AdamID)
			continue
		}
		availableAppStoreIDs = append(availableAppStoreIDs, app.AdamID)
	}
	appPolicies, err := profiles.BuildAndroidApplicationPolicies(ctx, availableAppStoreIDs, configsByAppID, android.AppStatusAvailable)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building application policies with config")
	}
	preinstalledAppPolicies, err := profiles.BuildAndroidApplicationPolicies(ctx, preinstalledAppStoreIDs, configsByAppID, android.AppStatusPreinstalled)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building preinstalled application policies with config")
	}

	installsToUpsert = append(installsToUpsert, failedInstalls...)
	if agentPackage != "" {
		installsToUpsert = append(installsToUpsert, fleet.HostAndroidSoftwareInstall{HostID: host.ID, AdamID: agentPackage, InstallType: android.AppStatusForceInstalled,
			Status: fleet.AndroidSoftwareInstallStatusPending, PolicyBuiltAt: policyBuiltAt})
	}
	var setupExperienceAppStoreIDsWithChangedConfig []string
	for appStoreID := range configChangedAppStoreIDs {
		if installTypeByAppStoreID[appStoreID] == android.AppStatusPreinstalled {
			setupExperienceAppStoreIDsWithChangedConfig = append(setupExperienceAppStoreIDsWithChangedConfig, appStoreID)
		}
	}

	return &androidAppList{
		host:                     host,
		applicationsWithoutAgent: append(appPolicies, preinstalledAppPolicies...),
		installsToUpsert:         installsToUpsert,
		installsToRemove:         installsToRemove,
		setupExperienceAppStoreIDsWithChangedConfig: setupExperienceAppStoreIDsWithChangedConfig,
		retriesFailedRequest:                        retriesFailedRequest,
	}, nil
}

func (r *profileReconciler) sendHostApps(ctx context.Context, appListsByHostUUID map[string]*androidAppList, batchSize int, profileHostCount int) error {
	appLists := slices.SortedFunc(maps.Values(appListsByHostUUID), func(a *androidAppList, b *androidAppList) int {
		return cmp.Compare(a.host.ID, b.host.ID)
	})
	for i, appList := range appLists {
		if r.appSendsStoppedByQuota {
			return nil
		}
		// Stop once the run's policy patches reach batchSize, and move the cursor back before this host so the next run
		// sends the rest
		if batchSize > 0 && profileHostCount+i >= batchSize {
			if r.appNextCursor == 0 || appList.host.ID-1 < r.appNextCursor {
				r.appNextCursor = appList.host.ID - 1
			}
			return nil
		}

		applications, err := r.appListApplications(ctx, appList)
		if err != nil {
			r.Logger.ErrorContext(ctx, "failed to build android app list for host", "host_id", appList.host.ID, "err", err)
			continue
		}
		hostUUID := appList.host.UUID
		policyName := fmt.Sprintf("%s/policies/%s", r.Enterprise.Name(), hostUUID)
		policy := &androidmanagement.Policy{Applications: applications}
		policyReq, skip, err := r.patchPolicy(ctx, hostUUID, policyName, policy, nil, androidmgmt.PoliciesPatchOpts{OnlyUpdateApps: true})
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "patch app list for host %s", hostUUID)
		}
		err = r.recordAppListRequest(ctx, appList, policyReq, skip)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *profileReconciler) appListApplications(ctx context.Context, appList *androidAppList) ([]*androidmanagement.ApplicationPolicy, error) {
	// Fail rather than replace the list without the Fleet agent, as replacing would remove it
	agentSvc := &Service{
		logger:             r.Logger,
		ds:                 r.DS,
		fleetDS:            r.DS,
		androidAPIClient:   r.Client,
		androidAgentConfig: r.AndroidAgentConfig,
	}
	agentPolicy, err := agentSvc.BuildFleetAgentApplicationPolicy(ctx, appList.host.UUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build fleet agent application policy")
	}
	applications := slices.Clone(appList.applicationsWithoutAgent)
	if agentPolicy != nil {
		applications = append(applications, agentPolicy)
	}
	return applications, nil
}

func (r *profileReconciler) recordAppListRequest(ctx context.Context, appList *androidAppList, policyRequest *android.MDMAndroidPolicyRequest, skip bool) error {
	host := appList.host

	// Leave the list unrecorded and stop the app list sends on a quota error, the next run sends it again without waiting
	// to retry
	if policyRequest.StatusCode == http.StatusTooManyRequests {
		r.appSendsStoppedByQuota = true
		return nil
	}

	// Count a retry only once its request is sent, the batch size can leave the host for the next run
	if appList.retriesFailedRequest && r.RetryCounter != nil {
		_, err := r.RetryCounter.RecordAttempt(ctx, host.ID, 0, fleet.VPPAutomaticInstallRetryTimeout)
		if err != nil {
			r.Logger.WarnContext(ctx, "failed to record android app list retry", "host_id", host.ID, "err", err)
		}
	}

	if !skip && policyRequest.StatusCode != http.StatusOK {
		// Set the changed apps and the agent to request failed, so the next run waits before sending again. Keep the apps
		// that were being removed, their removal wasn't sent either.
		for _, install := range appList.installsToUpsert {
			if install.Status == fleet.AndroidSoftwareInstallStatusPending {
				install.Status = fleet.AndroidSoftwareInstallStatusRequestFailed
			}
			r.appInstallsToUpsert = append(r.appInstallsToUpsert, install)
		}
		for _, install := range appList.installsToRemove {
			install.Status = fleet.AndroidSoftwareInstallStatusRequestFailed
			r.appInstallsToUpsert = append(r.appInstallsToUpsert, install)
		}
		return nil
	}

	for _, install := range appList.installsToUpsert {
		if install.Status == fleet.AndroidSoftwareInstallStatusPending && policyRequest.PolicyVersion.Valid {
			install.InPolicyVersion = &policyRequest.PolicyVersion.V
		}
		r.appInstallsToUpsert = append(r.appInstallsToUpsert, install)
	}
	r.appInstallsToRemove = append(r.appInstallsToRemove, appList.installsToRemove...)
	r.appSentHostIDs = append(r.appSentHostIDs, host.ID)

	// Set the setup experience install of an app whose configuration changed back to pending in the new policy version,
	// so the next status report verifies it with the new configuration applied
	if policyRequest.PolicyVersion.Valid {
		for _, appStoreID := range appList.setupExperienceAppStoreIDsWithChangedConfig {
			err := r.DS.SetAndroidAppInstallPendingApplyConfig(ctx, host.UUID, appStoreID, policyRequest.PolicyVersion.V)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "set android setup experience install pending apply config")
			}
		}
	}
	return nil
}
