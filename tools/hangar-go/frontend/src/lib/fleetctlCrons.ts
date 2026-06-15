// Metadata for `fleetctl trigger --name <X>`. Names match the registry in
// fleet/cmd/fleet/cron.go. Schedules were sourced from that file (Oct 2025
// snapshot — verify against current code if cadence questions come up).
//
// Grouping is opinionated for the Fleet Hangar UX:
//   - "featured": triggers people actually want on-demand during dev
//   - "mdm":      device-state reconcilers; hourly-ish; less common
//   - "maintenance": 5–10m crons; rarely needed manually
//   - "fast":     <2m crons; triggering is almost never useful
//   - "migration": WithRunOnce — no-op after first run

export type CronGroup =
  | "featured"
  | "mdm"
  | "maintenance"
  | "fast"
  | "migration";

export interface CronInfo {
  name: string;
  group: CronGroup;
  interval: string;
  note?: string;
}

export const CRONS: CronInfo[] = [
  // ---------- featured ----------
  {
    name: "cleanups_then_aggregation",
    group: "featured",
    interval: "1h",
    note: "Cleans distributed queries/carves/policy membership/etc., then aggregates stats. Run after seeding fixtures to flush noise.",
  },
  {
    name: "vulnerabilities",
    group: "featured",
    interval: "config",
    note: "CVE scan. Runs its own orphan-cleanup inline — does NOT need cleanups_then_aggregation to run first.",
  },
  {
    name: "automations",
    group: "featured",
    interval: "24h",
    note: "Webhook + failing-policy automation batch. Fire after editing a policy/webhook to see it trigger now.",
  },
  {
    name: "calendar",
    group: "featured",
    interval: "—",
    note: "Calendar event sync.",
  },
  {
    name: "chart_data_collection",
    group: "featured",
    interval: "1h",
    note: "Collects chart datasets for enabled teams.",
  },
  {
    name: "maintained_apps",
    group: "featured",
    interval: "1h",
    note: "Syncs the Fleet-maintained-apps list.",
  },
  {
    name: "refresh_vpp_app_versions",
    group: "featured",
    interval: "1h",
    note: "Pulls latest VPP app versions from Apple.",
  },
  {
    name: "usage_statistics",
    group: "featured",
    interval: "1h",
    note: "Sends analytics to fleetdm.com.",
  },

  // ---------- MDM device-state ----------
  {
    name: "apple_mdm_dep_profile_assigner",
    group: "mdm",
    interval: "config",
    note: "Syncs with Apple Business Manager.",
  },
  {
    name: "apple_mdm_iphone_ipad_refetcher",
    group: "mdm",
    interval: "config",
    note: "Enqueues DeviceInformation commands on iOS/iPadOS hosts.",
  },
  {
    name: "apple_mdm_iphone_ipad_reviver",
    group: "mdm",
    interval: "1h",
    note: "APNS-pokes deleted BYOD iOS/iPadOS devices still enrolled.",
  },
  {
    name: "mdm_service_discovery",
    group: "mdm",
    interval: "1h",
    note: "Apple account-driven enrollment profile setup.",
  },
  {
    name: "mdm_android_device_reconciler",
    group: "mdm",
    interval: "1h",
    note: "Reconciles Android device existence with Google AMAPI.",
  },

  // ---------- activity / maintenance ----------
  {
    name: "host_vitals_label_membership",
    group: "maintenance",
    interval: "5m",
    note: "Refreshes membership of host-vitals labels.",
  },
  {
    name: "batch_activity_completion_checker",
    group: "maintenance",
    interval: "5m",
    note: "Marks batch activities as completed.",
  },
  {
    name: "upcoming_activities_maintenance",
    group: "maintenance",
    interval: "10m",
    note: "Unblocks hosts stuck in the upcoming-activity queue.",
  },
  {
    name: "send_managed_local_account_rotation_commands",
    group: "maintenance",
    interval: "5m",
    note: "Rotates managed local account passwords.",
  },
  {
    name: "send_recovery_lock_commands",
    group: "maintenance",
    interval: "30s",
    note: "SetRecoveryLock MDM commands for macOS.",
  },

  // ---------- fast loops (triggering is almost never useful) ----------
  {
    name: "apple_mdm_worker",
    group: "fast",
    interval: "10s",
    note: "Worker pool for Apple MDM jobs.",
  },
  {
    name: "mdm_apple_profile_manager",
    group: "fast",
    interval: "30s",
    note: "Reconciles Apple profiles/declarations.",
  },
  {
    name: "mdm_windows_profile_manager",
    group: "fast",
    interval: "30s",
    note: "Reconciles Windows profiles.",
  },
  {
    name: "mdm_android_profile_manager",
    group: "fast",
    interval: "30s",
    note: "Reconciles Android profiles.",
  },
  {
    name: "apple_mdm_apns_pusher",
    group: "fast",
    interval: "1m",
    note: "Sends APNS pushes to pending iOS/iPadOS devices.",
  },
  {
    name: "integrations",
    group: "fast",
    interval: "1m",
    note: "Worker pool for Jira/Zendesk/VPP-verification/etc.",
  },
  {
    name: "query_results_cleanup",
    group: "fast",
    interval: "1m",
    note: "Trims excess query result rows.",
  },
  {
    name: "scheduled_batch_activities",
    group: "fast",
    interval: "2m",
    note: "Worker pool for batch script execution.",
  },

  // ---------- one-shot migrations ----------
  {
    name: "uninstall_software_migration",
    group: "migration",
    interval: "once",
    note: "Fleet 4.57+ uninstall-script update. No-op after first run.",
  },
  {
    name: "upgrade_code_software_migration",
    group: "migration",
    interval: "once",
    note: "Fleet 4.72+ MSI upgrade-code update. No-op after first run.",
  },
  {
    name: "enable_android_app_reports_on_default_policy",
    group: "migration",
    interval: "once",
    note: "Fleet 4.76+ migration. No-op after first run.",
  },
  {
    name: "migrate_to_per_host_policy",
    group: "migration",
    interval: "once",
    note: "Fleet 4.77+ migration. No-op after first run.",
  },
];

export const CRON_GROUP_TITLE: Record<CronGroup, string> = {
  featured: "Featured",
  mdm: "MDM device-state",
  maintenance: "Activity & maintenance",
  fast: "Fast loops (rarely worth triggering)",
  migration: "One-shot migrations",
};

export const CRON_GROUP_SUBTITLE: Record<CronGroup, string> = {
  featured: "Common dev workflows.",
  mdm: "Hourly device-state reconcilers.",
  maintenance: "5–10 minute crons.",
  fast: "Already run every few seconds to ~2 minutes — manual trigger is usually pointless.",
  migration: "Run once on startup; subsequent triggers are no-ops.",
};
