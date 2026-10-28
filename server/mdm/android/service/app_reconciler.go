package service

import (
	"bytes"
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/fleetdm/fleet/v4/server/mdm/profiles"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

type appReconciler struct {
	DS           fleet.Datastore
	AndroidSvc   android.Service
	RetryCounter fleet.SoftwareInstallAttemptCounter
	Logger       *slog.Logger
}

func ReconcileApps(ctx context.Context, ds fleet.Datastore, androidSvc android.Service, retryCounter fleet.SoftwareInstallAttemptCounter,
	logger *slog.Logger, batchSize int,
) error {
	ctx = androidmgmt.WithoutRetry(ctx)

	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get app config")
	}
	if !appConfig.MDM.AndroidEnabledAndConfigured {
		return nil
	}
	enterprise, err := ds.GetEnterprise(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get android enterprise")
	}

	// Start from the first host when the cursor can't be read, checking hosts again only costs reads
	cursor, err := ds.GetAndroidAppReconcileCursor(ctx)
	if err != nil {
		logger.WarnContext(ctx, "failed to read android app reconcile cursor, starting from the first host", "err", err)
		cursor = 0
	}
	// Check at most batchSize hosts per run, each costs at most one AMAPI request, so a large backlog is spread over
	// runs instead of using the shared quota at once. A batch size of 0 checks every host.
	limit := batchSize
	if limit <= 0 {
		limit = math.MaxInt32
	}
	hosts, err := ds.ListAndroidHostsOnHostPolicyAfterID(ctx, cursor, limit)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list android hosts")
	}
	hostIDs := make([]uint, 0, len(hosts))
	for _, host := range hosts {
		hostIDs = append(hostIDs, host.ID)
	}

	hostIDsWithPendingSetupInstall, err := ds.GetAndroidHostIDsWithPendingSetupExperienceInstall(ctx, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get android hosts with pending setup experience install")
	}
	installs, err := ds.ListHostAndroidSoftwareInstalls(ctx, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list host android software installs")
	}
	installByAppStoreIDByHostID := make(map[uint]map[string]*fleet.HostAndroidSoftwareInstall, len(hosts))
	for _, install := range installs {
		if installByAppStoreIDByHostID[install.HostID] == nil {
			installByAppStoreIDByHostID[install.HostID] = make(map[string]*fleet.HostAndroidSoftwareInstall)
		}
		installByAppStoreIDByHostID[install.HostID][install.AdamID] = install
	}

	// Read every fleet's versions, so a host moved from another fleet gets a send only when the configuration differs
	versionsByTitleIDByFleetID, err := ds.GetAppStoreAppVersionsByFleetAndTitle(ctx, []fleet.InstallableDevicePlatform{fleet.AndroidPlatform})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get app store app versions")
	}
	versionByID := make(map[uint]*fleet.VPPAppStoreApp)
	for _, versionsByTitleID := range versionsByTitleIDByFleetID {
		for _, versions := range versionsByTitleID {
			for _, version := range versions {
				versionByID[version.VPPAppsTeamsID] = version
			}
		}
	}

	reconciler := &appReconciler{DS: ds, AndroidSvc: androidSvc, RetryCounter: retryCounter, Logger: logger}
	for _, host := range hosts {
		// Leave the host to the setup experience until its setup installs are verified or failed, a replace would send
		// them as AVAILABLE before the device installs them
		if _, pending := hostIDsWithPendingSetupInstall[host.ID]; pending {
			continue
		}

		err = reconciler.reconcileHostApps(ctx, enterprise.Name(), host, installByAppStoreIDByHostID[host.ID], versionByID)
		if err != nil {
			// Stop on a quota error without moving the cursor, the next run starts this batch again
			if androidmgmt.IsTooManyRequestsError(err) {
				return ctxerr.Wrap(ctx, err, "reconcile android apps")
			}
			logger.ErrorContext(ctx, "failed to reconcile android apps for host", "host_id", host.ID, "err", err)
		}
	}

	// Skip saving the cursor when the run timed out, so the next run checks this batch again
	if ctx.Err() != nil {
		return ctxerr.Wrap(ctx, ctx.Err(), "reconcile android apps")
	}

	// Continue after this batch on the next run, or start from the first host again after the last batch
	var nextCursor uint
	if batchSize > 0 && len(hosts) == batchSize {
		nextCursor = hosts[len(hosts)-1].ID
	}
	err = ds.SetAndroidAppReconcileCursor(ctx, nextCursor)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "set android app reconcile cursor")
	}
	return nil
}

func (r *appReconciler) reconcileHostApps(ctx context.Context, enterpriseName string, host *fleet.Host,
	installByAppStoreID map[string]*fleet.HostAndroidSoftwareInstall, versionByID map[uint]*fleet.VPPAppStoreApp,
) error {
	appsInScope, err := r.DS.GetAndroidAppsInScopeForHost(ctx, host.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get android apps in scope for host")
	}

	// Retry a failed request after a wait, and send any other change right away
	var needsSend bool
	var needsRetry bool
	var lastFailedAt time.Time
	var versionOnlyInstalls []*fleet.HostAndroidSoftwareInstall
	appStoreIDsInScope := make(map[string]struct{}, len(appsInScope))
	for _, install := range installByAppStoreID {
		if install.Status == fleet.AndroidSoftwareInstallStatusFailed && install.NeedsResend {
			needsRetry = true
			if install.UpdatedAt.After(lastFailedAt) {
				lastFailedAt = install.UpdatedAt
			}
		}
	}
	for _, app := range appsInScope {
		appStoreIDsInScope[app.AdamID] = struct{}{}
		install, sent := installByAppStoreID[app.AdamID]
		if !sent {
			needsSend = true
			continue
		}
		version := versionByID[app.AppTeamID]
		switch {
		case install.Status == fleet.AndroidSoftwareInstallStatusFailed && install.NeedsResend:
			// Retried below
		case !ptr.Equal(install.VPPAppTeamID, &app.AppTeamID):
			// Update only the row's version when the host moves to a version with the same configuration, the app entry
			// in the policy is the same
			if install.VPPAppTeamID != nil && version != nil {
				sentVersion, found := versionByID[*install.VPPAppTeamID]
				if found && !install.NeedsResend && bytes.Equal(sentVersion.Configuration, version.Configuration) &&
					(sentVersion.ConfigurationUpdatedAt == nil || sentVersion.ConfigurationUpdatedAt.Before(install.PolicyBuiltAt)) {
					versionOnlyInstall := *install
					versionOnlyInstall.VPPAppTeamID = &app.AppTeamID
					versionOnlyInstalls = append(versionOnlyInstalls, &versionOnlyInstall)
					continue
				}
			}
			needsSend = true
		case install.NeedsResend:
			needsSend = true
		case version != nil && version.ConfigurationUpdatedAt != nil && install.PolicyBuiltAt.Before(*version.ConfigurationUpdatedAt):
			needsSend = true
		}
	}
	for appStoreID, install := range installByAppStoreID {
		if _, inScope := appStoreIDsInScope[appStoreID]; inScope {
			continue
		}
		if install.Status != fleet.AndroidSoftwareInstallStatusFailed || !install.NeedsResend {
			needsSend = true
		}
	}
	// TODO(JK): Account for the Fleet agent app. Every replace sends it, but it has no row, so:
	//  - A host with no apps in scope has no rows before or after a send, so this cron never sends it a list. If an
	//    older job put an app in its policy, for example before a label change took it out of the app's scope, the cron
	//    never removes that app from the policy or the device.
	//  - The agent's configuration is also sent with an add call at enrollment and when its certificate templates
	//    change. Those sends write no row, so the table can't show which agent configuration the host has.
	// This cron needs to know about the agent because every replace it sends includes the agent with its configuration
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
		if len(versionOnlyInstalls) == 0 {
			return nil
		}
		err = r.DS.UpdateHostAndroidSoftwareInstallVersions(ctx, host.ID, versionOnlyInstalls)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update host android software install versions")
		}
		return nil
	}

	// Read the time before the configurations, so an edit made while building the policy is sent on the next run
	policyBuiltAt := time.Now()
	vppAppTeamIDs := make([]uint, 0, len(appsInScope))
	for _, app := range appsInScope {
		vppAppTeamIDs = append(vppAppTeamIDs, app.AppTeamID)
	}
	configsByAppID, err := r.DS.BulkGetAndroidAppConfigurations(ctx, vppAppTeamIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "bulk get android app configurations")
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
			return ctxerr.Wrap(ctx, err, "substitute fleet vars in app configs")
		}
		configsByAppID[app.AdamID] = substituted[app.AdamID]
		appsToSend = append(appsToSend, app)
	}

	appStoreIDsToSend := make([]string, 0, len(appsToSend))
	for _, app := range appsToSend {
		appStoreIDsToSend = append(appStoreIDsToSend, app.AdamID)
	}
	appPolicies, err := profiles.BuildAndroidApplicationPolicies(ctx, appStoreIDsToSend, configsByAppID, "AVAILABLE")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building application policies with config")
	}

	// Fail rather than replace the list without the Fleet Agent, as replacing would remove it
	fleetAgentPolicy, err := r.AndroidSvc.BuildFleetAgentApplicationPolicy(ctx, host.UUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build fleet agent application policy")
	}
	if fleetAgentPolicy != nil {
		appPolicies = append(appPolicies, fleetAgentPolicy)
	}

	policyRequests, err := r.AndroidSvc.SetAppsForAndroidPolicy(ctx, enterpriseName, appPolicies, map[string]string{host.UUID: host.UUID})
	if err != nil {
		if androidmgmt.IsTooManyRequestsError(err) {
			return ctxerr.Wrap(ctx, err, "set apps for android policy")
		}
		// Set the host's rows to failed, so the next run waits before sending again
		failedInstalls := make([]*fleet.HostAndroidSoftwareInstall, 0, len(appsInScope))
		for _, app := range appsInScope {
			failedInstalls = append(failedInstalls, &fleet.HostAndroidSoftwareInstall{
				HostID:        host.ID,
				AdamID:        app.AdamID,
				VPPAppTeamID:  &app.AppTeamID,
				PolicyBuiltAt: policyBuiltAt,
			})
		}
		recordErr := r.DS.SetHostAndroidSoftwareInstallsRequestFailed(ctx, host.ID, failedInstalls)
		if recordErr != nil {
			r.Logger.ErrorContext(ctx, "failed to record failed android app request", "host_id", host.ID, "err", recordErr)
		}
		return ctxerr.Wrap(ctx, err, "set apps for android policy")
	}

	installs := fleet.MergeHostAndroidInstallsWithPolicyDetails(appsToSend, appsWithUnresolvableValues, policyRequests[host.UUID], policyBuiltAt)
	err = r.DS.ReplaceHostAndroidSoftwareInstalls(ctx, host.ID, installs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "replace host android software installs")
	}
	return nil
}
