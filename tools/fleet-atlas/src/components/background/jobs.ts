export type CronTopic =
  | "Housekeeping"
  | "Automations & integrations"
  | "Batch activities"
  | "Vulnerabilities"
  | "Software catalog (FMAs & VPP)"
  | "MDM: Apple"
  | "MDM: Microsoft"
  | "MDM: Android"
  | "Calendar & Workspace"
  | "Reports (query results)"
  | "One-shot migrations";

export const TOPIC_ORDER: CronTopic[] = [
  "Housekeeping",
  "Automations & integrations",
  "Batch activities",
  "Vulnerabilities",
  "Software catalog (FMAs & VPP)",
  "MDM: Apple",
  "MDM: Microsoft",
  "MDM: Android",
  "Calendar & Workspace",
  "Reports (query results)",
  "One-shot migrations",
];

export const TOPIC_COLORS: Record<CronTopic, string> = {
  Housekeeping:
    "bg-zinc-100 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300",
  "Automations & integrations":
    "bg-sky-100 text-sky-800 dark:bg-sky-900/60 dark:text-sky-200",
  "Batch activities":
    "bg-indigo-100 text-indigo-800 dark:bg-indigo-900/60 dark:text-indigo-200",
  Vulnerabilities:
    "bg-fuchsia-100 text-fuchsia-800 dark:bg-fuchsia-900/60 dark:text-fuchsia-200",
  "Software catalog (FMAs & VPP)":
    "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/60 dark:text-emerald-200",
  "MDM: Apple":
    "bg-pink-100 text-pink-800 dark:bg-pink-900/60 dark:text-pink-200",
  "MDM: Microsoft":
    "bg-cyan-100 text-cyan-800 dark:bg-cyan-900/60 dark:text-cyan-200",
  "MDM: Android":
    "bg-lime-100 text-lime-800 dark:bg-lime-900/60 dark:text-lime-200",
  "Calendar & Workspace":
    "bg-amber-100 text-amber-800 dark:bg-amber-900/60 dark:text-amber-200",
  "Reports (query results)":
    "bg-violet-100 text-violet-800 dark:bg-violet-900/60 dark:text-violet-200",
  "One-shot migrations":
    "bg-rose-100 text-rose-800 dark:bg-rose-900/60 dark:text-rose-200",
};

export type CronJob = {
  name: string;
  intervalMinutes: number;
  purpose: string;
  file: string;
  topic: CronTopic;
};

// Pulled from server/fleet/cron_schedules.go + cmd/fleet/cron.go
// Intervals are typical defaults; some are configurable.
export const CRONS: CronJob[] = [
  {
    name: "cleanups_then_aggregation",
    intervalMinutes: 60,
    purpose:
      "Hourly janitor: deletes stale sessions, old carves, orphan rows; recomputes host counts and dashboard aggregates.",
    file: "cmd/fleet/cron.go",
    topic: "Housekeeping",
  },
  {
    name: "frequent_cleanups",
    intervalMinutes: 15,
    purpose:
      "Short-interval housekeeping: expire pending enrolls, retire finished live-query campaigns.",
    file: "cmd/fleet/cron.go",
    topic: "Housekeeping",
  },
  {
    name: "upcoming_activities_maintenance",
    intervalMinutes: 10,
    purpose:
      "Expire stale upcoming_activities rows and reconcile queue depth.",
    file: "cmd/fleet/cron.go",
    topic: "Housekeeping",
  },
  {
    name: "host_vitals_label_membership",
    intervalMinutes: 5,
    purpose:
      "Recompute label membership for custom-host-vital-based labels.",
    file: "cmd/fleet/cron.go",
    topic: "Housekeeping",
  },
  {
    name: "usage_statistics",
    intervalMinutes: 60,
    purpose:
      "Daily anonymous usage telemetry upload if the admin opted in.",
    file: "cmd/fleet/cron.go",
    topic: "Housekeeping",
  },
  {
    name: "automations",
    intervalMinutes: 10,
    purpose:
      "Fire failing-policy webhooks, failing-policy integrations (Jira/Zendesk) by enqueueing worker jobs.",
    file: "cmd/fleet/cron.go",
    topic: "Automations & integrations",
  },
  {
    name: "integrations",
    intervalMinutes: 1,
    purpose:
      "Drain the worker queue (Jira, Zendesk, software install, Apple MDM, VPP verification).",
    file: "cmd/fleet/cron.go",
    topic: "Automations & integrations",
  },
  {
    name: "activities_streaming",
    intervalMinutes: 5,
    purpose:
      "Push activity-feed rows to a configured streaming endpoint (SIEM / webhook).",
    file: "cmd/fleet/cron.go",
    topic: "Automations & integrations",
  },
  {
    name: "fleet_initiated_activities_release",
    intervalMinutes: 1,
    purpose:
      "Promote queued fleet-initiated activities (install, run script) to the host when the window opens.",
    file: "cmd/fleet/cron.go",
    topic: "Batch activities",
  },
  {
    name: "batch_activity_completion_checker",
    intervalMinutes: 5,
    purpose:
      "Mark batch activities complete when all per-host children have finished.",
    file: "cmd/fleet/cron.go",
    topic: "Batch activities",
  },
  {
    name: "scheduled_batch_activities",
    intervalMinutes: 2,
    purpose:
      "Fire batch activities that were scheduled for a future time.",
    file: "cmd/fleet/cron.go",
    topic: "Batch activities",
  },
  {
    name: "vulnerabilities",
    intervalMinutes: 60,
    purpose:
      "Download NVD/OSV/OVAL/MSRC/Office feeds, match CPEs against installed software, write host_software_vulnerabilities.",
    file: "server/vulnerabilities/",
    topic: "Vulnerabilities",
  },
  {
    name: "maintained_apps",
    intervalMinutes: 60,
    purpose:
      "Rebuild the Fleet-maintained apps (FMA) catalog from Homebrew + winget nightly.",
    file: "ee/server/maintainedapps/",
    topic: "Software catalog (FMAs & VPP)",
  },
  {
    name: "windows_maintained_app_titles",
    intervalMinutes: 60,
    purpose:
      "Sync Windows FMA software_titles rows.",
    file: "ee/server/maintainedapps/",
    topic: "Software catalog (FMAs & VPP)",
  },
  {
    name: "maintained_apps_auto_update",
    intervalMinutes: 60,
    purpose:
      "Push version bumps for FMAs with auto-update enabled.",
    file: "ee/server/maintainedapps/",
    topic: "Software catalog (FMAs & VPP)",
  },
  {
    name: "refresh_vpp_app_versions",
    intervalMinutes: 60,
    purpose:
      "Poll Apple VPP API for latest versions of licensed apps.",
    file: "cmd/fleet/cron.go",
    topic: "Software catalog (FMAs & VPP)",
  },
  {
    name: "mdm_apple_profile_manager",
    intervalMinutes: 0.5,
    purpose:
      "Reconcile desired vs. installed Apple MDM profiles per host; queue install/remove commands. Runs every 30 seconds.",
    file: "server/service/mdm.go",
    topic: "MDM: Apple",
  },
  {
    name: "apple_mdm_apns_sweep",
    intervalMinutes: 1,
    purpose:
      "Modern APNs push: re-wakes Apple MDM enrollments that have been silent for more than a day. Replaced the legacy per-minute pusher. Default tick 1m.",
    file: "cmd/fleet/cron.go",
    topic: "MDM: Apple",
  },
  {
    name: "apple_mdm_apns_pusher (legacy)",
    intervalMinutes: 1,
    purpose:
      "LEGACY per-minute APNs pusher. Only registered if FLEET_MDM_APPLE_LEGACY_APNS_PUSHER_INTERVAL env var is set. Replaced by apple_mdm_apns_sweep.",
    file: "cmd/fleet/cron.go",
    topic: "MDM: Apple",
  },
  {
    name: "apple_mdm_worker",
    intervalMinutes: 1 / 6,
    purpose:
      "Fast drain of the Apple MDM worker job queue (~10 seconds). Picks up queued MDM command enqueues from service-layer actions.",
    file: "cmd/fleet/cron.go",
    topic: "MDM: Apple",
  },
  {
    name: "apple_mdm_dep_profile_assigner",
    intervalMinutes: 1,
    purpose:
      "Push DEP enrollment profile assignments to Apple Business Manager.",
    file: "cmd/fleet/cron.go",
    topic: "MDM: Apple",
  },
  {
    name: "apple_mdm_iphone_ipad_refetcher",
    intervalMinutes: 10,
    purpose:
      "Refresh iPhone/iPad inventory via MDM (osquery doesn't run on iOS). Every 10 minutes.",
    file: "cmd/fleet/cron.go",
    topic: "MDM: Apple",
  },
  {
    name: "apple_mdm_iphone_ipad_reviver",
    intervalMinutes: 60,
    purpose:
      "Attempt to re-contact dormant iPhones/iPads (hourly, not daily).",
    file: "cmd/fleet/cron.go",
    topic: "MDM: Apple",
  },
  {
    name: "mdm_windows_profile_manager",
    intervalMinutes: 0.5,
    purpose:
      "Reconcile Windows SyncML profiles; emit outbound commands into windows_mdm_command_queue.",
    file: "server/service/mdm_windows.go",
    topic: "MDM: Microsoft",
  },
  {
    name: "microsoft_autopilot_sync",
    intervalMinutes: 5,
    purpose:
      "Pull Autopilot device registrations from Microsoft Graph.",
    file: "server/cron/microsoft_autopilot_cron.go",
    topic: "MDM: Microsoft",
  },
  {
    name: "mdm_android_profile_manager",
    intervalMinutes: 0.5,
    purpose:
      "Diff Android policy against Google's view; call AMAPI to update.",
    file: "server/mdm/android/",
    topic: "MDM: Android",
  },
  {
    name: "mdm_android_device_reconciler",
    intervalMinutes: 60,
    purpose:
      "Reconcile Android device state Google reports vs. Fleet's record.",
    file: "server/mdm/android/",
    topic: "MDM: Android",
  },
  {
    name: "enable_android_app_reports_on_default_policy",
    intervalMinutes: 60 * 24,
    purpose:
      "One-shot backfill cron turning on app reports on default Android policies.",
    file: "cmd/fleet/cron.go",
    topic: "MDM: Android",
  },
  {
    name: "calendar",
    intervalMinutes: 10,
    purpose:
      "Create/update calendar events on end-user calendars for policy automations.",
    file: "server/cron/calendar_cron.go",
    topic: "Calendar & Workspace",
  },
  {
    name: "google_workspace_sync",
    intervalMinutes: 5,
    purpose:
      "Sync users/groups from Google Workspace for policy automation targeting.",
    file: "server/cron/google_workspace_cron.go",
    topic: "Calendar & Workspace",
  },
  {
    name: "chart_data_collection",
    intervalMinutes: 60,
    purpose:
      "Compute dashboard chart datasets (host counts over time, policy pass rates, etc.) on an hourly tick.",
    file: "cmd/fleet/cron.go",
    topic: "Housekeeping",
  },
  {
    name: "query_results_cleanup",
    intervalMinutes: 1,
    purpose:
      "Delete query_results rows past the configured retention window so the table doesn't grow unbounded.",
    file: "cmd/fleet/cron.go",
    topic: "Reports (query results)",
  },
  {
    name: "end_user_notifications",
    intervalMinutes: 1,
    purpose:
      "Deliver end-user-facing notifications (install prompts, update nudges) to Fleet Desktop every minute.",
    file: "cmd/fleet/cron.go",
    topic: "MDM: Apple",
  },
  {
    name: "apple_mdm_os_updates",
    intervalMinutes: 60,
    purpose:
      "Enforce the configured macOS / iOS / iPadOS minimum version per fleet; nudge devices toward the deadline.",
    file: "cmd/fleet/cron.go",
    topic: "MDM: Apple",
  },
  {
    name: "app_store_app_configuration_reconciler",
    intervalMinutes: 1,
    purpose:
      "Reconcile Managed App Config (post-install JSON config for VPP apps) against desired state.",
    file: "cmd/fleet/cron.go",
    topic: "MDM: Apple",
  },
  {
    name: "send_recovery_lock_commands",
    intervalMinutes: 0.5,
    purpose:
      "Push Recovery Lock set/clear MDM commands to Apple Silicon Macs every 30 seconds.",
    file: "cmd/fleet/cron.go",
    topic: "MDM: Apple",
  },
  {
    name: "send_managed_local_account_rotation_commands",
    intervalMinutes: 5,
    purpose:
      "Rotate the password for the managed local account on macOS hosts that are due.",
    file: "cmd/fleet/cron.go",
    topic: "MDM: Apple",
  },
  {
    name: "reencrypt_windows_disk_encryption_keys",
    intervalMinutes: 60,
    purpose:
      "Re-encrypt escrowed BitLocker recovery keys using the current org key; runs hourly.",
    file: "cmd/fleet/cron.go",
    topic: "MDM: Microsoft",
  },
  {
    name: "mdm_android_command_reconciler",
    intervalMinutes: 60 * 24,
    purpose:
      "Daily reconciliation of outstanding Android MDM commands against AMAPI reported state.",
    file: "cmd/fleet/cron.go",
    topic: "MDM: Android",
  },
  {
    name: "cleanup_expired_adue_challenges",
    intervalMinutes: 60 * 24,
    purpose:
      "Delete expired account-driven user-enrollment (ADUE) challenge rows daily.",
    file: "cmd/fleet/cron.go",
    topic: "Housekeeping",
  },
  {
    name: "migrate_to_per_host_policy",
    intervalMinutes: 60 * 24,
    purpose:
      "One-shot migration cron: back-fills the per-host policy model after the schema change. Idempotent; runs daily until done.",
    file: "cmd/fleet/cron.go",
    topic: "One-shot migrations",
  },
  {
    name: "uninstall_software_migration",
    intervalMinutes: 60 * 24,
    purpose:
      "One-shot migration cron for a software-uninstall data model change.",
    file: "cmd/fleet/cron.go",
    topic: "One-shot migrations",
  },
  {
    name: "upgrade_code_software_migration",
    intervalMinutes: 60 * 24,
    purpose:
      "One-shot migration cron for the upgrade-code column backfill.",
    file: "cmd/fleet/cron.go",
    topic: "One-shot migrations",
  },
];

export type WorkerJob = {
  name: string;
  purpose: string;
  file: string;
};

// Pulled from server/worker/
export const WORKERS: WorkerJob[] = [
  {
    name: "jira",
    purpose:
      "Create Jira issues for policy failures, software vulns, or failed scripts, retrying on API errors.",
    file: "server/worker/jira.go",
  },
  {
    name: "zendesk",
    purpose:
      "Same shape as Jira but for Zendesk tickets.",
    file: "server/worker/zendesk.go",
  },
  {
    name: "apple_mdm",
    purpose:
      "Side-effects after admin actions on Apple MDM: queue install/remove, refresh dep assignments, etc.",
    file: "server/worker/apple_mdm.go",
  },
  {
    name: "apple_software",
    purpose:
      "Verify VPP install state asynchronously after admin-triggered installs.",
    file: "server/worker/vpp_verification.go",
  },
  {
    name: "software_worker",
    purpose:
      "Process software install/uninstall side-effects outside the request.",
    file: "server/worker/software_worker.go",
  },
  {
    name: "macos_setup_assistant",
    purpose:
      "Rebuild and re-upload the ABM setup assistant JSON when config changes.",
    file: "server/worker/macos_setup_assistant.go",
  },
  {
    name: "db_migration",
    purpose:
      "Long-running data backfills kicked off by migration code (so startup isn't blocked).",
    file: "server/worker/db_migrations.go",
  },
  {
    name: "chart_scrub_dataset_global",
    purpose:
      "Scrub PII from exported charts (global).",
    file: "server/worker/chart_scrub.go",
  },
  {
    name: "chart_scrub_dataset_fleet",
    purpose:
      "Scrub PII from exported charts (per-fleet).",
    file: "server/worker/chart_scrub.go",
  },
  {
    name: "automation_failer",
    purpose:
      "Deliberately fail stuck automation runs so the admin UI shows an error instead of 'pending' forever.",
    file: "server/worker/automation_failer.go",
  },
  {
    name: "batch_activities",
    purpose:
      "Fan out batch operations into per-host upcoming_activities rows.",
    file: "server/worker/batch_activities.go",
  },
];

export function cadenceLabel(mins: number): string {
  if (mins < 60) return `every ${mins}m`;
  if (mins < 60 * 24) return `every ${mins / 60}h`;
  const days = mins / (60 * 24);
  return `every ${days}d`;
}
