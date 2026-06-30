package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/googleworkspace"
	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	chart_api "github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/cron"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	acme_api "github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/apple_apps"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
	"github.com/fleetdm/fleet/v4/server/service/redis_key_value"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
)

// cronSchedulesDeps carries the runServeCmd-scoped dependencies that the cron
// schedule registrations close over. Registration is grouped by domain in
// startCronSchedules; the call order is identical to the previous inline
// sequence in runServeCmd.
type cronSchedulesDeps struct {
	instanceID             string
	config                 *config.FleetConfig
	license                *fleet.LicenseInfo
	logger                 *slog.Logger
	cronSchedules          *fleet.CronSchedules
	ds                     fleet.Datastore
	svc                    fleet.Service
	carveStore             fleet.CarveStore
	enrollHostLimiter      fleet.EnrollHostLimiter
	liveQueryStore         fleet.LiveQueryStore
	failingPolicySet       fleet.FailingPolicySet
	redisPool              fleet.RedisPool
	commander              *apple_mdm.MDMAppleCommander
	depStorage             *mysql.NanoDEPStorage
	softwareInstallStore   fleet.SoftwareInstallerStore
	bootstrapPackageStore  fleet.MDMBootstrapPackageStore
	softwareTitleIconStore fleet.SoftwareTitleIconStore
	androidSvc             android.Service
	activitySvc            activity_api.Service
	acmeSvc                acme_api.Service
	chartSvc               chart_api.Service
	auditLogger            fleet.JSONLogger
	distributedLock        fleet.Lock
	initFatal              func(err error, msg string)
}

// register starts a single cron schedule and routes a registration failure
// through initFatal with the given message. It removes the repetitive
// error-check boilerplate from each registration site.
func (deps cronSchedulesDeps) register(failMsg string, newSchedule func() (fleet.CronSchedule, error)) {
	if err := deps.cronSchedules.StartCronSchedule(newSchedule); err != nil {
		deps.initFatal(err, failMsg)
	}
}

// startCronSchedules registers every cron schedule the server runs, grouped by
// domain. The registration order is preserved exactly from the previous inline
// sequence in runServeCmd.
func startCronSchedules(ctx context.Context, deps cronSchedulesDeps) {
	registerCleanupAndMaintenanceCrons(ctx, deps)
	registerVulnerabilityCrons(ctx, deps)
	registerWorkerCrons(ctx, deps)
	registerMDMCrons(ctx, deps)
	registerPremiumCrons(ctx, deps)
	registerMiscCrons(ctx, deps)

	deps.logger.InfoContext(ctx, fmt.Sprintf("started cron schedules: %s", strings.Join(deps.cronSchedules.ScheduleNames(), ", ")))
}

// registerCleanupAndMaintenanceCrons covers chart data collection, cron_stats
// cleanup, software migrations, frequent cleanups, the cleanups-then-aggregation
// schedule, query results cleanup, upcoming activities maintenance, usage
// statistics, and batch activities.
func registerCleanupAndMaintenanceCrons(ctx context.Context, deps cronSchedulesDeps) {
	if os.Getenv("FLEET_SKIP_CHART_DATA_COLLECTION") == "" {
		deps.register("failed to register chart_data_collection schedule", func() (fleet.CronSchedule, error) {
			return newChartDataCollectionSchedule(ctx, deps.instanceID, deps.ds, deps.chartSvc, deps.logger)
		})
	} else {
		deps.logger.InfoContext(ctx, "skipping chart data collection cron (FLEET_SKIP_CHART_DATA_COLLECTION is set)")
	}

	// Perform a cleanup of cron_stats outside of the cronSchedules because the
	// schedule package uses cron_stats entries to decide whether a schedule will
	// run or not (see https://github.com/fleetdm/fleet/issues/9486).
	go func() {
		cleanupCronStats := func() {
			deps.logger.DebugContext(ctx, "cleaning up cron_stats")
			// Datastore.CleanupCronStats should be safe to run by multiple fleet
			// instances at the same time and it should not be an expensive operation.
			if err := deps.ds.CleanupCronStats(ctx); err != nil {
				deps.logger.InfoContext(ctx, "failed to clean up cron_stats", "err", err)
			}
		}

		cleanupCronStats()

		cleanUpCronStatsTick := time.NewTicker(1 * time.Hour)
		defer cleanUpCronStatsTick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-cleanUpCronStatsTick.C:
				cleanupCronStats()
			}
		}
	}()

	if deps.softwareInstallStore != nil {
		deps.register(fmt.Sprintf("failed to register %s", fleet.CronUninstallSoftwareMigration), func() (fleet.CronSchedule, error) {
			return cronUninstallSoftwareMigration(ctx, deps.instanceID, deps.ds, deps.softwareInstallStore, deps.logger)
		})

		deps.register(fmt.Sprintf("failed to register %s", fleet.CronUpgradeCodeSoftwareMigration), func() (fleet.CronSchedule, error) {
			return cronUpgradeCodeSoftwareMigration(ctx, deps.instanceID, deps.ds, deps.softwareInstallStore, deps.logger)
		})
	}

	if deps.config.Server.FrequentCleanupsEnabled {
		deps.register("failed to register frequent_cleanups schedule", func() (fleet.CronSchedule, error) {
			return newFrequentCleanupsSchedule(ctx, deps.instanceID, deps.ds, deps.liveQueryStore, deps.logger)
		})
	}

	deps.register("failed to register cleanups_then_aggregations schedule", func() (fleet.CronSchedule, error) {
		return newCleanupsAndAggregationSchedule(
			ctx, deps.instanceID, deps.ds, deps.carveStore, deps.svc, deps.logger, deps.enrollHostLimiter, deps.config, deps.commander, deps.softwareInstallStore, deps.bootstrapPackageStore, deps.softwareTitleIconStore, deps.androidSvc, deps.activitySvc, deps.acmeSvc, deps.chartSvc,
		)
	})

	deps.register("failed to register query_results_cleanup schedule", func() (fleet.CronSchedule, error) {
		return newQueryResultsCleanupSchedule(ctx, deps.instanceID, deps.ds, deps.liveQueryStore, deps.logger)
	})

	deps.register("failed to register upcoming_activities_maintenance schedule", func() (fleet.CronSchedule, error) {
		return newUpcomingActivitiesSchedule(ctx, deps.instanceID, deps.ds, deps.logger)
	})

	deps.register("failed to register stats schedule", func() (fleet.CronSchedule, error) {
		return newUsageStatisticsSchedule(ctx, deps.instanceID, deps.ds, *deps.config, deps.logger)
	})

	deps.register("failed to register batch activities schedule", func() (fleet.CronSchedule, error) {
		return newBatchActivitiesSchedule(ctx, deps.instanceID, deps.ds, deps.logger)
	})
}

// vulnerabilityProcessingDisabled reports whether this instance should skip
// running the vulnerabilities processing schedule. Processing is disabled
// either explicitly (disable_schedule) or when this instance opts out of
// current_instance_checks ("no" or the legacy "0"); in that case a remote
// trigger proxy is registered instead.
func vulnerabilityProcessingDisabled(cfg config.VulnerabilitiesConfig) bool {
	return cfg.DisableSchedule || cfg.IsDisabledByInstanceCheck()
}

// registerVulnerabilityCrons registers either the vulnerabilities processing
// schedule or, when processing is disabled on this instance, a remote trigger
// proxy so triggering still works when the schedule runs on a separate server.
func registerVulnerabilityCrons(ctx context.Context, deps cronSchedulesDeps) {
	// Log the specific reason(s) processing is disabled, if any.
	if deps.config.Vulnerabilities.DisableSchedule {
		deps.logger.InfoContext(ctx, "vulnerabilities schedule disabled via vulnerabilities.disable_schedule")
	}
	if deps.config.Vulnerabilities.IsDisabledByInstanceCheck() {
		deps.logger.InfoContext(ctx, "vulnerabilities schedule disabled via vulnerabilities.current_instance_checks")
	}
	if !vulnerabilityProcessingDisabled(deps.config.Vulnerabilities) {
		// vuln processing by default is run by internal cron mechanism
		deps.register("failed to register vulnerabilities schedule", func() (fleet.CronSchedule, error) {
			return newVulnerabilitiesSchedule(ctx, deps.instanceID, deps.ds, deps.logger, &deps.config.Vulnerabilities)
		})
	} else {
		// Register a remote trigger proxy so triggering still works
		// when the vulnerability schedule runs on a separate server.
		deps.register("failed to register remote vulnerability trigger", func() (fleet.CronSchedule, error) {
			return schedule.NewRemoteTriggerSchedule(string(fleet.CronVulnerabilities), deps.ds), nil
		})
	}
}

// registerWorkerCrons covers the automations schedule and the worker
// integrations schedule.
func registerWorkerCrons(ctx context.Context, deps cronSchedulesDeps) {
	deps.register("failed to register automations schedule", func() (fleet.CronSchedule, error) {
		return newAutomationsSchedule(ctx, deps.instanceID, deps.ds, deps.logger, 5*time.Minute, deps.failingPolicySet, deps.activitySvc)
	})

	deps.register("failed to register worker integrations schedule", func() (fleet.CronSchedule, error) {
		return newWorkerIntegrationsSchedule(ctx, deps.instanceID, deps.ds, deps.logger, deps.depStorage, deps.commander, deps.androidSvc, deps.chartSvc, deps.config.MDM.AndroidBatchSize, deps.activitySvc)
	})
}

// registerMDMCrons covers the Apple MDM worker, DEP profile assigner, service
// discovery, the Apple/Windows/Android profile managers, the Android device
// reconciler, the Android default-policy and per-host policy migrations, and
// the APNs pusher.
func registerMDMCrons(ctx context.Context, deps cronSchedulesDeps) {
	deps.register("failed to register apple_mdm_worker schedule", func() (fleet.CronSchedule, error) {
		vppInstaller := deps.svc.(fleet.AppleMDMVPPInstaller)
		return newAppleMDMWorkerSchedule(ctx, deps.instanceID, deps.ds, deps.logger, deps.commander, deps.bootstrapPackageStore, vppInstaller, deps.svc.NewActivity)
	})

	deps.register("failed to register apple_mdm_dep_profile_assigner schedule", func() (fleet.CronSchedule, error) {
		return newAppleMDMDEPProfileAssigner(ctx, deps.instanceID, deps.config.MDM.AppleDEPSyncPeriodicity, deps.ds, deps.depStorage, deps.logger)
	})

	deps.register("failed to register mdm_apple_service_discovery schedule", func() (fleet.CronSchedule, error) {
		return newMDMAppleServiceDiscoverySchedule(ctx, deps.instanceID, deps.ds, deps.depStorage, deps.logger, deps.config.Server.URLPrefix)
	})

	deps.register("failed to register mdm_apple_profile_manager schedule", func() (fleet.CronSchedule, error) {
		return newAppleMDMProfileManagerSchedule(
			ctx,
			deps.instanceID,
			deps.ds,
			deps.commander,
			redis_key_value.New(deps.redisPool),
			deps.logger,
			deps.config.MDM.CertificateProfilesLimit,
		)
	})

	deps.register("failed to register mdm_windows_profile_manager schedule", func() (fleet.CronSchedule, error) {
		return newWindowsMDMProfileManagerSchedule(
			ctx,
			deps.instanceID,
			deps.ds,
			deps.logger,
		)
	})

	deps.register("failed to register mdm_android_profile_manager schedule", func() (fleet.CronSchedule, error) {
		return newAndroidMDMProfileManagerSchedule(
			ctx,
			deps.instanceID,
			deps.ds,
			deps.logger,
			deps.config.License.Key, // NOTE: this requires the license key, not the parsed *LicenseInfo available in the ctx
			deps.config.MDM.AndroidAgent,
			deps.config.MDM.AndroidBatchSize,
		)
	})

	// Register Android MDM Device Reconciler schedule (same interval as Android profile manager)
	deps.register("failed to register mdm_android_device_reconciler schedule", func() (fleet.CronSchedule, error) {
		return newAndroidMDMDeviceReconcilerSchedule(
			ctx,
			deps.instanceID,
			deps.ds,
			deps.logger,
			deps.config.License.Key,
			deps.svc.NewActivity,
		)
	})

	deps.register("failed to register enable_android_app_reports_on_default_policy cron", func() (fleet.CronSchedule, error) {
		return cronEnableAndroidAppReportsOnDefaultPolicy(ctx, deps.instanceID, deps.ds, deps.logger, deps.androidSvc)
	})

	deps.register("failed to register migrate_to_per_host_policy cron", func() (fleet.CronSchedule, error) {
		return cronMigrateToPerHostPolicy(ctx, deps.instanceID, deps.ds, deps.logger, deps.androidSvc)
	})

	deps.register("failed to register APNs pusher schedule", func() (fleet.CronSchedule, error) {
		return newMDMAPNsPusher(
			ctx,
			deps.instanceID,
			deps.ds,
			deps.commander,
			deps.logger,
		)
	})
}

// registerPremiumCrons covers the Fleet Premium schedules: iPhone/iPad
// refetcher and reviver, maintained apps, VPP app version refresh (and the
// one-shot VPP country backfill), recovery lock passwords, managed local
// account rotation, activities streaming, and the calendar schedule.
func registerPremiumCrons(ctx context.Context, deps cronSchedulesDeps) {
	if !deps.license.IsPremium() {
		return
	}

	deps.register("failed to register apple_mdm_iphone_ipad_refetcher schedule", func() (fleet.CronSchedule, error) {
		return newIPhoneIPadRefetcher(ctx, deps.instanceID, 10*time.Minute, deps.ds, deps.commander, deps.logger, deps.svc.NewActivity)
	})

	deps.register("failed to register apple_mdm_iphone_ipad_reviver schedule", func() (fleet.CronSchedule, error) {
		return newIPhoneIPadReviver(ctx, deps.instanceID, deps.ds, deps.commander, deps.logger)
	})

	deps.register("failed to register maintained apps schedule", func() (fleet.CronSchedule, error) {
		return newMaintainedAppSchedule(ctx, deps.instanceID, deps.ds, deps.logger)
	})

	deps.register("failed to register maintained apps auto-update schedule", func() (fleet.CronSchedule, error) {
		return newMaintainedAppsAutoUpdateSchedule(ctx, deps.instanceID, deps.ds, deps.softwareInstallStore, deps.logger)
	})

	deps.register("failed to register refresh vpp app versions schedule", func() (fleet.CronSchedule, error) {
		return newRefreshVPPAppVersionsSchedule(ctx, deps.instanceID, deps.ds, deps.logger, apple_apps.Configure(ctx, deps.ds, deps.config.License.Key, deps.config.MDM.AppleConnectJWT))
	})

	// One-shot backfill for VPP token and app country codes that
	// predate the country_code column. Fire-and-forget is safe because
	// the work is idempotent and ctx cancels on shutdown.
	go vpp.BackfillLegacyCountries(ctx, deps.ds, deps.logger)

	deps.register("failed to register recovery lock password schedule", func() (fleet.CronSchedule, error) {
		return newRecoveryLockPasswordSchedule(ctx, deps.instanceID, deps.ds, deps.commander, deps.logger, deps.svc.NewActivity)
	})

	deps.register("failed to register managed local account rotation schedule", func() (fleet.CronSchedule, error) {
		return newManagedLocalAccountRotationSchedule(ctx, deps.instanceID, deps.ds, deps.commander, deps.logger, deps.svc.NewActivity)
	})

	deps.register("failed to register cleanup expired ADUE challenges schedule", func() (fleet.CronSchedule, error) {
		return newCleanupExpiredADUEChallengesSchedule(ctx, deps.instanceID, deps.ds, deps.logger)
	})

	if deps.config.Activity.EnableAuditLog {
		deps.register("failed to register activities streaming schedule", func() (fleet.CronSchedule, error) {
			return newActivitiesStreamingSchedule(ctx, deps.instanceID, deps.activitySvc, deps.ds, deps.logger, deps.auditLogger)
		})
	}

	deps.register("failed to register calendar schedule", func() (fleet.CronSchedule, error) {
		if deps.config.Calendar.Periodicity > 0 {
			deps.config.Calendar.SetAlwaysReloadEvent(true)
		} else {
			deps.config.Calendar.Periodicity = 5 * time.Minute
		}
		return cron.NewCalendarSchedule(ctx, deps.instanceID, deps.ds, deps.distributedLock, deps.config.Calendar, deps.logger, deps.activitySvc)
	})

	deps.register("failed to register google workspace sync schedule", func() (fleet.CronSchedule, error) {
		return cron.NewGoogleWorkspaceSchedule(ctx, deps.instanceID, deps.ds, googleworkspace.NewDirectory, deps.logger)
	})
}

// registerMiscCrons covers the host vitals label membership schedule and the
// batch activity completion checker.
func registerMiscCrons(ctx context.Context, deps cronSchedulesDeps) {
	// Start the service that calculates and updates host vitals label membership.
	deps.register("failed to register host vitals label membership schedule", func() (fleet.CronSchedule, error) {
		return newHostVitalsLabelMembershipSchedule(ctx, deps.instanceID, deps.ds, deps.logger)
	})

	// Start the service that marks activities as completed.
	deps.register("failed to register batch activity completion checker schedule", func() (fleet.CronSchedule, error) {
		return newBatchActivityCompletionCheckerSchedule(ctx, deps.instanceID, deps.ds, deps.logger)
	})
}
