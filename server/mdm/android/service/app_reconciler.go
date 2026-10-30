package service

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"maps"
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

// androidAppList is the app list built for a host whose apps or Fleet agent
// changed, with the installs to save once it's sent.
type androidAppList struct {
	host                         *fleet.Host
	applicationsWithoutAgent     []*androidmanagement.ApplicationPolicy
	installsToUpsert             []fleet.HostAndroidSoftwareInstall
	installsToRemove             []fleet.HostAndroidSoftwareInstall
	setupAppIDsWithChangedConfig []string
	retriesFailedRequest         bool
}

func (r *profileReconciler) buildAppListsToSend(
	ctx context.Context,
	profileHostUUIDs map[string]struct{},
	batchSize int,
	policyBuiltAt time.Time,
) (map[string]*androidAppList, error) {
	hosts, err := r.DS.ListAndroidHostsOnHostPolicy(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list android hosts")
	}
	hostIDs := make([]uint, 0, len(hosts))
	for _, host := range hosts {
		hostIDs = append(hostIDs, host.ID)
	}
	appsInScopeByHostID, err := r.DS.ListAndroidAppsInScopeForHosts(ctx, hostIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list android apps in scope for hosts")
	}
	setupAppsByHostID, err := r.DS.ListHostMDMAndroidVPPAppsFromSetupExperience(ctx, hostIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list android setup experience apps")
	}
	installsByHostID, err := r.DS.ListHostAndroidSoftwareInstalls(ctx, hostIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list host android software installs")
	}

	// Read the versions from the primary, after policyBuiltAt, so an edit a
	// replica doesn't have yet isn't saved as sent
	primaryCtx := ctxdb.RequirePrimary(ctx, true)
	// Read every fleet's versions with their configurations, so a host moved
	// from another fleet will get a send only when the configuration differs
	versionByID, err := r.DS.GetAndroidAppStoreAppVersionsByID(primaryCtx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get android app store app versions")
	}

	appListsByHostUUID := make(map[string]*androidAppList)
	var appList *androidAppList
	var otherHostListCount int
	for _, host := range hosts {
		_, hasProfileChanges := profileHostUUIDs[host.UUID]
		// Build the lists of hosts with profile changes, which go in their profile patch, and at most batchSize minus
		// that many other lists, so a large backlog will be spread over runs instead of using the AMAPI quota at once
		if !hasProfileChanges && batchSize > 0 && len(profileHostUUIDs)+otherHostListCount >= batchSize {
			continue
		}
		appList, err = r.prepareHostAppList(ctx, host, appsInScopeByHostID[host.ID], installsByHostID[host.ID], setupAppsByHostID[host.ID],
			versionByID, policyBuiltAt)
		if err != nil {
			r.Logger.ErrorContext(ctx, "failed to prepare android app list for host", "host_id", host.ID, "err", err)
			continue
		}
		if appList == nil {
			continue
		}
		appListsByHostUUID[host.UUID] = appList
		if !hasProfileChanges {
			otherHostListCount++
		}
	}
	return appListsByHostUUID, nil
}

func (r *profileReconciler) prepareHostAppList(
	ctx context.Context,
	host *fleet.Host,
	appsInList []fleet.VPPAppTeam,
	installByAppID map[string]*fleet.HostAndroidSoftwareInstall,
	setupApps []*fleet.HostAndroidVPPSoftwareInstall,
	versionByID map[uint]*fleet.VPPAppStoreApp,
	policyBuiltAt time.Time,
) (*androidAppList, error) {
	installTypeByAppID := make(map[string]string, len(appsInList)+len(setupApps))
	for _, app := range appsInList {
		installTypeByAppID[app.AdamID] = android.AppStatusAvailable
	}
	// Add the apps the setup experience installs as PREINSTALLED, also while the
	// device installs them and when their labels filter the host out, so
	// replacing the list doesn't stop their install or uninstall them. The setup
	// experience doesn't check labels yet.
	for _, setupApp := range setupApps {
		if _, inList := installTypeByAppID[setupApp.AdamID]; !inList {
			appsInList = append(appsInList, fleet.VPPAppTeam{AdamID: setupApp.AdamID, AppTeamID: setupApp.VPPAppTeamID})
		}
		installTypeByAppID[setupApp.AdamID] = android.AppStatusPreinstalled
	}
	// Count the Fleet agent as in scope, because every list includes it
	agentPackage := r.AndroidAgentConfig.Package
	if agentPackage != "" {
		installTypeByAppID[agentPackage] = android.AppStatusForceInstalled
	}

	var err error
	var needsSend bool
	var needsRetry bool
	var lastFailedAt time.Time
	var installsToRemove []fleet.HostAndroidSoftwareInstall
	for appID, install := range installByAppID {
		// Wait before retrying a failed request
		if install.Status == fleet.AndroidSoftwareInstallStatusRequestFailed {
			needsRetry = true
			if install.UpdatedAt.After(lastFailedAt) {
				lastFailedAt = install.UpdatedAt
			}
		}
		if _, inList := installTypeByAppID[appID]; inList {
			continue
		}
		installsToRemove = append(installsToRemove, *install)
		// Send the removal right away, unless its last request failed and the retry below will send it
		if install.Status != fleet.AndroidSoftwareInstallStatusRequestFailed {
			needsSend = true
		}
	}
	var installsWithNewVersion []fleet.HostAndroidSoftwareInstall
	changedAppIDs := make(map[string]struct{})
	var configChangedAppIDs []string
	// Compare each app with its last sent install, to find the changed apps and whether the list needs a send
	for _, app := range appsInList {
		install, sent := installByAppID[app.AdamID]
		version := versionByID[app.AppTeamID]
		switch {
		case !sent:
			needsSend = true
		case install.NeedsResend, version != nil && version.ConfigurationUpdatedAt != nil && install.PolicyBuiltAt.Before(*version.ConfigurationUpdatedAt):
			needsSend = true
			configChangedAppIDs = append(configChangedAppIDs, app.AdamID)
		case !ptr.Equal(install.VPPAppTeamID, &app.AppTeamID):
			var sentVersion *fleet.VPPAppStoreApp
			if install.VPPAppTeamID != nil {
				sentVersion = versionByID[*install.VPPAppTeamID]
			}
			// Update only the install's version when the host moves to a version with the same configuration,
			// because the app's entry in the policy doesn't change
			if sentVersion != nil && version != nil && bytes.Equal(sentVersion.Configuration, version.Configuration) &&
				(sentVersion.ConfigurationUpdatedAt == nil || sentVersion.ConfigurationUpdatedAt.Before(install.PolicyBuiltAt)) {
				installWithNewVersion := *install
				installWithNewVersion.VPPAppTeamID = new(app.AppTeamID)
				installsWithNewVersion = append(installsWithNewVersion, installWithNewVersion)
			} else {
				needsSend = true
				configChangedAppIDs = append(configChangedAppIDs, app.AdamID)
			}
		case install.InstallType != installTypeByAppID[app.AdamID]:
			needsSend = true
		case install.Status == fleet.AndroidSoftwareInstallStatusRequestFailed:
			// Send it with the retry below
		default:
			continue
		}
		changedAppIDs[app.AdamID] = struct{}{}
	}
	if agentPackage != "" {
		agentInstall, agentSent := installByAppID[agentPackage]
		// Send a list to a host whose installs don't include the agent. It never got a list from Fleet,
		// so its policy can still have apps that older jobs didn't remove.
		if !agentSent {
			needsSend = true
		}
		// Send a list when the agent's configuration changed
		if agentSent && agentInstall.NeedsResend {
			needsSend = true
		}
	}

	var retriesFailedRequest bool
	// Retry a failed request after 1 hour, like the App Store app configuration reconciler
	if !needsSend && needsRetry && time.Since(lastFailedAt) >= fleet.VPPAutomaticInstallWaitSeconds*time.Second {
		var retries int
		// Count retries per host, because a request sends the host's whole app list
		if r.RetryCounter != nil {
			retries, err = r.RetryCounter.CountAttempts(ctx, host.ID, 0)
			if err != nil {
				r.Logger.WarnContext(ctx, "failed to count android app list retries", "host_id", host.ID, "err", err)
			}
		}
		// Retry at most twice until the count expires
		if retries < fleet.MaxVPPAutomaticInstallRetries {
			needsSend = true
			retriesFailedRequest = true
		}
	}
	if !needsSend {
		r.appInstallsToUpsert = append(r.appInstallsToUpsert, installsWithNewVersion...)
		return nil, nil
	}

	substitutionHost := profiles.AndroidAppConfigSubstitutionHost{
		HostID:         host.ID,
		UUID:           host.UUID,
		HardwareSerial: host.HardwareSerial,
		Platform:       host.Platform,
	}
	configsByAppID := make(map[string][]byte, len(appsInList))
	var availableAppIDs []string
	var preinstalledAppIDs []string
	var installsToUpsert []fleet.HostAndroidSoftwareInstall
	var substituted []byte
	for _, app := range appsInList {
		version := versionByID[app.AppTeamID]
		// Check the host again on the next run, because its app version was added after the versions were read
		if version == nil {
			return nil, nil
		}
		_, changed := changedAppIDs[app.AdamID]
		install := fleet.HostAndroidSoftwareInstall{HostID: host.ID, AdamID: app.AdamID, VPPAppTeamID: new(app.AppTeamID), InstallType: installTypeByAppID[app.AdamID],
			Status: fleet.AndroidSoftwareInstallStatusPending, PolicyBuiltAt: policyBuiltAt}
		if len(version.Configuration) > 0 {
			substituted, err = profiles.SubstituteFleetVarsAndVitalsInAndroidAppConfig(ctx, r.DS, version.Configuration, substitutionHost)
			if err != nil {
				if !profiles.IsUnresolvableAndroidAppConfigForHost(err) {
					return nil, ctxerr.Wrapf(ctx, err, "substitute fleet vars in android app config for app %s", app.AdamID)
				}
				r.Logger.WarnContext(ctx, "not sending android policy to host that can't supply a value referenced by an app config",
					"host_id", host.ID, "application_id", app.AdamID, "err", err)
				// Save the changed app as failed, so the next run will only try again once a change to that value flags it
				if changed {
					install.Status = fleet.AndroidSoftwareInstallStatusFailed
					r.appInstallsToUpsert = append(r.appInstallsToUpsert, install)
				}
				// Don't send the host's policy until it has every value its app configurations use, like for profiles
				return nil, nil
			}
			configsByAppID[app.AdamID] = substituted
		}
		// Save only the apps that changed. An unchanged app's entry in the new policy is the same as the one last sent.
		if changed {
			installsToUpsert = append(installsToUpsert, install)
		}
		if installTypeByAppID[app.AdamID] == android.AppStatusPreinstalled {
			preinstalledAppIDs = append(preinstalledAppIDs, app.AdamID)
			continue
		}
		availableAppIDs = append(availableAppIDs, app.AdamID)
	}
	appPolicies, err := profiles.BuildAndroidApplicationPolicies(ctx, availableAppIDs, configsByAppID, android.AppStatusAvailable)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building application policies with config")
	}
	preinstalledAppPolicies, err := profiles.BuildAndroidApplicationPolicies(ctx, preinstalledAppIDs, configsByAppID, android.AppStatusPreinstalled)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building preinstalled application policies with config")
	}

	if agentPackage != "" {
		installsToUpsert = append(installsToUpsert, fleet.HostAndroidSoftwareInstall{HostID: host.ID, AdamID: agentPackage, InstallType: android.AppStatusForceInstalled,
			Status: fleet.AndroidSoftwareInstallStatusPending, PolicyBuiltAt: policyBuiltAt})
	}
	var setupAppIDsWithChangedConfig []string
	for _, appID := range configChangedAppIDs {
		if installTypeByAppID[appID] == android.AppStatusPreinstalled {
			setupAppIDsWithChangedConfig = append(setupAppIDsWithChangedConfig, appID)
		}
	}

	return &androidAppList{
		host:                         host,
		applicationsWithoutAgent:     append(appPolicies, preinstalledAppPolicies...),
		installsToUpsert:             installsToUpsert,
		installsToRemove:             installsToRemove,
		setupAppIDsWithChangedConfig: setupAppIDsWithChangedConfig,
		retriesFailedRequest:         retriesFailedRequest,
	}, nil
}

func (r *profileReconciler) sendHostApps(ctx context.Context, appListsByHostUUID map[string]*androidAppList) error {
	appLists := slices.SortedFunc(maps.Values(appListsByHostUUID), func(a *androidAppList, b *androidAppList) int {
		return cmp.Compare(a.host.ID, b.host.ID)
	})
	for _, appList := range appLists {
		if r.appSendsStoppedByQuota {
			return nil
		}

		applications, err := r.buildApplicationsWithAgent(ctx, appList)
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

func (r *profileReconciler) buildApplicationsWithAgent(
	ctx context.Context,
	appList *androidAppList,
) ([]*androidmanagement.ApplicationPolicy, error) {
	agentSvc := &Service{
		logger:             r.Logger,
		ds:                 r.DS,
		fleetDS:            r.DS,
		androidAPIClient:   r.Client,
		androidAgentConfig: r.AndroidAgentConfig,
	}
	agentPolicy, err := agentSvc.BuildFleetAgentApplicationPolicy(ctx, appList.host.UUID)
	// Fail instead of building the list without the Fleet agent, because replacing the list would remove the agent
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build fleet agent application policy")
	}
	applications := slices.Clone(appList.applicationsWithoutAgent)
	if agentPolicy != nil {
		applications = append(applications, agentPolicy)
	}
	return applications, nil
}

func (r *profileReconciler) recordAppListRequest(
	ctx context.Context,
	appList *androidAppList,
	policyRequest *android.MDMAndroidPolicyRequest,
	skip bool,
) error {
	host := appList.host

	// Stop sending app lists on a quota error and don't save this list, so the
	// next run will send it again without waiting to retry
	if policyRequest.StatusCode == http.StatusTooManyRequests {
		r.appSendsStoppedByQuota = true
		return nil
	}

	// Count a retry only once its request is sent, because the batch size can delay the host to the next run
	if appList.retriesFailedRequest && r.RetryCounter != nil {
		_, err := r.RetryCounter.RecordAttempt(ctx, host.ID, 0, fleet.VPPAutomaticInstallRetryTimeout)
		if err != nil {
			r.Logger.WarnContext(ctx, "failed to record android app list retry", "host_id", host.ID, "err", err)
		}
	}

	if !skip && policyRequest.StatusCode != http.StatusOK {
		// Set the changed apps and the agent to request failed, so the next run will wait before sending again
		for _, install := range appList.installsToUpsert {
			if install.Status == fleet.AndroidSoftwareInstallStatusPending {
				install.Status = fleet.AndroidSoftwareInstallStatusRequestFailed
			}
			r.appInstallsToUpsert = append(r.appInstallsToUpsert, install)
		}
		// Set the removed apps to request failed instead of deleting them, because their removal wasn't sent either
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

	// Set the setup experience install of an app whose configuration changed back to pending in the new policy
	// version, so the next status report will verify it with the new configuration applied
	if policyRequest.PolicyVersion.Valid {
		for _, appID := range appList.setupAppIDsWithChangedConfig {
			err := r.DS.SetAndroidAppInstallPendingApplyConfig(ctx, host.UUID, appID, policyRequest.PolicyVersion.V)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "set android setup experience install pending apply config")
			}
		}
	}
	return nil
}
