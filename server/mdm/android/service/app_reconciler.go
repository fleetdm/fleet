package service

import (
	"bytes"
	"cmp"
	"context"
	"maps"
	"math"
	"slices"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/fleetdm/fleet/v4/server/mdm/profiles"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"google.golang.org/api/androidmanagement/v1"
)

// androidAppList is the app list built for a host whose apps changed, with what's needed to record the result of
// sending it.
type androidAppList struct {
	host                       *fleet.Host
	appPolicies                []*androidmanagement.ApplicationPolicy
	appsInScope                []fleet.VPPAppTeam
	appsToSend                 []fleet.VPPAppTeam
	appsWithUnresolvableValues []fleet.VPPAppTeam
	changedAppStoreIDs         map[string]struct{}
	installTypeByAppStoreID    map[string]string
	installByAppStoreID        map[string]*fleet.HostAndroidSoftwareInstall
	policyBuiltAt              time.Time
	// sentWithProfiles is set when the list went out in the same policy patch as the host's profiles
	sentWithProfiles bool
}

func (r *profileReconciler) prepareAppLists(ctx context.Context, batchSize int) (map[string]*androidAppList, error) {
	// Start from the first host when the cursor can't be read, checking hosts again only costs reads
	cursor, err := r.DS.GetAndroidAppReconcileCursor(ctx)
	if err != nil {
		r.Logger.WarnContext(ctx, "failed to read android app reconcile cursor, starting from the first host", "err", err)
		cursor = 0
	}
	// Check at most batchSize hosts per run, each costs at most one AMAPI request, so a large backlog is spread over
	// runs instead of using the shared quota at once. A batch size of 0 checks every host.
	limit := batchSize
	if limit <= 0 {
		limit = math.MaxInt32
	}
	hosts, err := r.DS.ListAndroidHostsOnHostPolicyAfterID(ctx, cursor, limit)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list android hosts")
	}
	hostIDs := make([]uint, 0, len(hosts))
	for _, host := range hosts {
		hostIDs = append(hostIDs, host.ID)
	}

	// Continue after this batch on the next run, or start from the first host again after the last batch
	r.appNextCursor = 0
	if batchSize > 0 && len(hosts) == batchSize {
		r.appNextCursor = hosts[len(hosts)-1].ID
	}

	hostIDsWithPendingSetupInstall, err := r.DS.GetAndroidHostIDsWithPendingSetupExperienceInstall(ctx, hostIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get android hosts with pending setup experience install")
	}
	setupApps, err := r.DS.ListHostMDMAndroidVPPAppsInstalledDuringSetup(ctx, hostIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list android apps installed during setup")
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

	// Read every fleet's versions, so a host moved from another fleet gets a send only when the configuration differs
	versionsByTitleIDByFleetID, err := r.DS.GetAppStoreAppVersionsByFleetAndTitle(ctx, []fleet.InstallableDevicePlatform{fleet.AndroidPlatform})
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
		// Leave the host to the setup experience until its setup installs are verified or failed, a replace would send
		// them as AVAILABLE before the device installs them
		if _, pending := hostIDsWithPendingSetupInstall[host.ID]; pending {
			continue
		}

		appList, err = r.prepareHostAppList(ctx, host, installByAppStoreIDByHostID[host.ID], setupAppsByHostID[host.ID], versionByID)
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

func (r *profileReconciler) prepareHostAppList(ctx context.Context, host *fleet.Host, installByAppStoreID map[string]*fleet.HostAndroidSoftwareInstall, setupApps []*fleet.HostAndroidVPPSoftwareInstall, versionByID map[uint]*fleet.VPPAppStoreApp) (*androidAppList, error) {
	appsInScope, err := r.DS.GetAndroidAppsInScopeForHost(ctx, host.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get android apps in scope for host")
	}

	// Keep the apps the setup experience installed as PREINSTALLED, also when their labels leave the host out of scope,
	// a replace without them would uninstall them. The setup experience doesn't check labels yet.
	installTypeByAppStoreID := make(map[string]string, len(appsInScope)+len(setupApps))
	for _, app := range appsInScope {
		installTypeByAppStoreID[app.AdamID] = android.AppStatusAvailable
	}
	for _, setupApp := range setupApps {
		if _, inScope := installTypeByAppStoreID[setupApp.AdamID]; !inScope {
			appsInScope = append(appsInScope, fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: setupApp.AdamID}, AppTeamID: setupApp.VPPAppTeamID})
		}
		installTypeByAppStoreID[setupApp.AdamID] = android.AppStatusPreinstalled
	}

	// Retry a failed request after a wait, and send any other change right away
	var needsSend bool
	var needsRetry bool
	var lastFailedAt time.Time
	newVersionIDByAppStoreID := make(map[string]uint)
	changedAppStoreIDs := make(map[string]struct{})
	for _, install := range installByAppStoreID {
		if install.Status == fleet.AndroidSoftwareInstallStatusFailed && install.NeedsResend {
			needsRetry = true
			if install.UpdatedAt.After(lastFailedAt) {
				lastFailedAt = install.UpdatedAt
			}
		}
	}
	for _, app := range appsInScope {
		install, sent := installByAppStoreID[app.AdamID]
		if !sent {
			needsSend = true
			changedAppStoreIDs[app.AdamID] = struct{}{}
			continue
		}
		version := versionByID[app.AppTeamID]
		switch {
		case install.Status == fleet.AndroidSoftwareInstallStatusFailed && install.NeedsResend:
			// Retried below
			changedAppStoreIDs[app.AdamID] = struct{}{}
		case install.InstallType != installTypeByAppStoreID[app.AdamID]:
			needsSend = true
			changedAppStoreIDs[app.AdamID] = struct{}{}
		case !ptr.Equal(install.VPPAppTeamID, &app.AppTeamID):
			changedAppStoreIDs[app.AdamID] = struct{}{}
			// Update only the row's version when the host moves to a version with the same configuration, the app entry
			// in the policy is the same
			if install.VPPAppTeamID != nil && version != nil {
				sentVersion, found := versionByID[*install.VPPAppTeamID]
				if found && !install.NeedsResend && bytes.Equal(sentVersion.Configuration, version.Configuration) &&
					(sentVersion.ConfigurationUpdatedAt == nil || sentVersion.ConfigurationUpdatedAt.Before(install.PolicyBuiltAt)) {
					newVersionIDByAppStoreID[app.AdamID] = app.AppTeamID
					continue
				}
			}
			needsSend = true
		case install.NeedsResend:
			needsSend = true
			changedAppStoreIDs[app.AdamID] = struct{}{}
		case version != nil && version.ConfigurationUpdatedAt != nil && install.PolicyBuiltAt.Before(*version.ConfigurationUpdatedAt):
			needsSend = true
			changedAppStoreIDs[app.AdamID] = struct{}{}
		}
	}
	for appStoreID, install := range installByAppStoreID {
		if _, inScope := installTypeByAppStoreID[appStoreID]; inScope {
			continue
		}
		if install.Status != fleet.AndroidSoftwareInstallStatusFailed || !install.NeedsResend {
			needsSend = true
		}
	}
	// TODO(JK): Account for the Fleet agent app. Every replace sends it, but it has no row, so:
	//  - A host with no apps in scope has no rows before or after a send, so it never gets an app list. If an older
	//    job put an app in its policy, for example before a label change took it out of the app's scope, that app is
	//    never removed from the policy or the device.
	//  - The agent's configuration is also sent with an add call at enrollment and when its certificate templates
	//    change. Those sends write no row, so the table can't show which agent configuration the host has.
	// The app lists need to account for the agent because every replace includes the agent with its configuration
	// built at that moment, so each send can change the agent's configuration on the device, and a replace without the
	// agent would remove it from the device.

	// Retry a failed request after 1 hour, at most MaxVPPAutomaticInstallRetries times until the count expires, like the
	// App Store app configuration reconciler. Count retries per host, a request sends the host's whole app list.
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
			if r.RetryCounter != nil {
				_, err = r.RetryCounter.RecordAttempt(ctx, host.ID, 0, fleet.VPPAutomaticInstallRetryTimeout)
				if err != nil {
					r.Logger.WarnContext(ctx, "failed to record android app list retry", "host_id", host.ID, "err", err)
				}
			}
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

	// Read the time before the configurations, so an edit made while building the policy is sent on the next run
	policyBuiltAt := time.Now()
	vppAppTeamIDs := make([]uint, 0, len(appsInScope))
	for _, app := range appsInScope {
		vppAppTeamIDs = append(vppAppTeamIDs, app.AppTeamID)
	}
	configsByAppID, err := r.DS.BulkGetAndroidAppConfigurations(ctx, vppAppTeamIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "bulk get android app configurations")
	}

	// Leave out an app whose configuration uses a value the host doesn't have, it's sent once a change to that value flags
	// it for resend
	substitutionHost := profiles.AndroidAppConfigSubstitutionHost{
		HostID:         host.ID,
		UUID:           host.UUID,
		HardwareSerial: host.HardwareSerial,
		Platform:       host.Platform,
	}
	var appsToSend []fleet.VPPAppTeam
	var appsWithUnresolvableValues []fleet.VPPAppTeam
	var substituted map[string][]byte
	for _, app := range appsInScope {
		config, configured := configsByAppID[app.AdamID]
		if !configured {
			appsToSend = append(appsToSend, app)
			continue
		}
		substituted, err = profiles.SubstituteFleetVarsInAndroidAppConfigs(ctx, r.DS, map[string][]byte{app.AdamID: config}, substitutionHost)
		if err != nil {
			if profiles.IsUnresolvableAndroidAppConfigForHost(err) {
				r.Logger.WarnContext(ctx, "skipping android app for host that can't supply a value referenced by its config",
					"host_id", host.ID, "application_id", app.AdamID, "err", err)
				appsWithUnresolvableValues = append(appsWithUnresolvableValues, app)
				continue
			}
			return nil, ctxerr.Wrap(ctx, err, "substitute fleet vars in app configs")
		}
		configsByAppID[app.AdamID] = substituted[app.AdamID]
		appsToSend = append(appsToSend, app)
	}

	var availableAppStoreIDs []string
	var preinstalledAppStoreIDs []string
	for _, app := range appsToSend {
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
	appPolicies = append(appPolicies, preinstalledAppPolicies...)

	// Fail rather than replace the list without the Fleet Agent, as replacing would remove it
	fleetAgentPolicy, err := r.AndroidSvc.BuildFleetAgentApplicationPolicy(ctx, host.UUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build fleet agent application policy")
	}
	if fleetAgentPolicy != nil {
		appPolicies = append(appPolicies, fleetAgentPolicy)
	}

	return &androidAppList{
		host:                       host,
		appPolicies:                appPolicies,
		appsInScope:                appsInScope,
		appsToSend:                 appsToSend,
		appsWithUnresolvableValues: appsWithUnresolvableValues,
		changedAppStoreIDs:         changedAppStoreIDs,
		installTypeByAppStoreID:    installTypeByAppStoreID,
		installByAppStoreID:        installByAppStoreID,
		policyBuiltAt:              policyBuiltAt,
	}, nil
}

func (r *profileReconciler) sendHostApps(ctx context.Context, appListsByHostUUID map[string]*androidAppList, batchSize int, profileHostCount int) {
	appLists := slices.SortedFunc(maps.Values(appListsByHostUUID), func(a *androidAppList, b *androidAppList) int {
		return cmp.Compare(a.host.ID, b.host.ID)
	})
	for i, appList := range appLists {
		// Stop once the run's policy patches reach batchSize, and set the cursor before this host so the next run sends
		// the rest
		if batchSize > 0 && profileHostCount+i >= batchSize {
			r.appNextCursor = appList.host.ID - 1
			return
		}

		hostUUID := appList.host.UUID
		policyRequests, err := r.AndroidSvc.SetAppsForAndroidPolicy(ctx, r.Enterprise.Name(), appList.appPolicies, map[string]string{hostUUID: hostUUID})
		if err != nil {
			// Stop on a quota error without moving the cursor, the next run checks this batch again
			if androidmgmt.IsTooManyRequestsError(err) {
				r.Logger.WarnContext(ctx, "stopped sending android app lists on a quota error", "host_id", appList.host.ID, "err", err)
				r.appSendsStoppedByQuota = true
				return
			}
			r.Logger.ErrorContext(ctx, "failed to send android app list for host", "host_id", appList.host.ID, "err", err)
			r.recordAppListResult(appList, nil, true)
			continue
		}
		r.recordAppListResult(appList, policyRequests[hostUUID], false)
	}
}

func (r *profileReconciler) recordAppListResult(appList *androidAppList, policyRequest *android.MDMAndroidPolicyRequest, requestFailed bool) {
	host := appList.host
	if requestFailed {
		// Set the host's installs to failed and flagged for resend, so the next run waits before sending again. Keep the
		// apps that were being removed, their removal wasn't sent either.
		for _, app := range appList.appsInScope {
			r.appInstallsToUpsert = append(r.appInstallsToUpsert, fleet.HostAndroidSoftwareInstall{
				HostID:        host.ID,
				AdamID:        app.AdamID,
				VPPAppTeamID:  &app.AppTeamID,
				InstallType:   appList.installTypeByAppStoreID[app.AdamID],
				Status:        fleet.AndroidSoftwareInstallStatusFailed,
				NeedsResend:   true,
				PolicyBuiltAt: appList.policyBuiltAt,
			})
		}
		for appStoreID, install := range appList.installByAppStoreID {
			if _, inScope := appList.installTypeByAppStoreID[appStoreID]; inScope {
				continue
			}
			failedInstall := *install
			failedInstall.Status = fleet.AndroidSoftwareInstallStatusFailed
			failedInstall.NeedsResend = true
			r.appInstallsToUpsert = append(r.appInstallsToUpsert, failedInstall)
		}
		return
	}

	// Upsert only the apps that changed, an unchanged app's entry in the new policy is the same as in the policy of its
	// existing install
	var changedAppsToSend []fleet.VPPAppTeam
	var changedAppsWithUnresolvableValues []fleet.VPPAppTeam
	for _, app := range appList.appsToSend {
		if _, changed := appList.changedAppStoreIDs[app.AdamID]; changed {
			changedAppsToSend = append(changedAppsToSend, app)
		}
	}
	for _, app := range appList.appsWithUnresolvableValues {
		if _, changed := appList.changedAppStoreIDs[app.AdamID]; changed {
			changedAppsWithUnresolvableValues = append(changedAppsWithUnresolvableValues, app)
		}
	}
	installs := fleet.MergeHostAndroidInstallsWithPolicyDetails(host.ID, changedAppsToSend, changedAppsWithUnresolvableValues, policyRequest, appList.policyBuiltAt)
	for i := range installs {
		installs[i].InstallType = appList.installTypeByAppStoreID[installs[i].AdamID]
	}
	r.appInstallsToUpsert = append(r.appInstallsToUpsert, installs...)

	for appStoreID, install := range appList.installByAppStoreID {
		if _, inScope := appList.installTypeByAppStoreID[appStoreID]; inScope {
			continue
		}
		r.appInstallsToRemove = append(r.appInstallsToRemove, *install)
	}
}
