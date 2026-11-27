export type CronQuestion = {
  id: string;
  difficulty: number; // 1-10
  question: string;
  options: string[];
  correctIndex: number;
  explanation: string;
  // Which cron this question centers on, so a per-question Review link opens
  // the right entry in the cron catalog above.
  cronName?: string;
};

export const CRON_QUESTIONS: CronQuestion[] = [
  // ─── Easy (1-3): basic naming + cadence ───
  {
    id: "fma-rebuild",
    difficulty: 2,
    question: "Which cron rebuilds the Fleet-maintained apps catalog?",
    options: [
      "maintained_apps",
      "refresh_vpp_app_versions",
      "automations",
      "mdm_apple_profile_manager",
    ],
    correctIndex: 0,
    explanation:
      "`maintained_apps` runs hourly and refreshes the FMA catalog from Homebrew + winget. `windows_maintained_app_titles` is the Windows-only sibling.",
    cronName: "maintained_apps",
  },
  {
    id: "integrations-drain",
    difficulty: 2,
    question: "Which cron drains the worker queue that pushes Jira / Zendesk tickets?",
    options: [
      "automations",
      "integrations",
      "activities_streaming",
      "scheduled_batch_activities",
    ],
    correctIndex: 1,
    explanation:
      "`integrations` runs every minute and drains the worker jobs table. `automations` DECIDES to enqueue a job; `integrations` DELIVERS it.",
    cronName: "integrations",
  },
  {
    id: "fastest-cron",
    difficulty: 2,
    question: "Which cron runs most frequently in Fleet?",
    options: [
      "mdm_apple_profile_manager (30s)",
      "integrations (1m)",
      "frequent_cleanups (15m)",
      "apple_mdm_dep_profile_assigner (1m)",
    ],
    correctIndex: 0,
    explanation:
      "The three MDM profile managers (Apple, Windows, Android) all run at 30 seconds. Fastest in Fleet. `apple_mdm_worker` ticks every ~10 seconds, but it's a worker-queue drainer rather than a reconciler.",
    cronName: "mdm_apple_profile_manager",
  },
  {
    id: "vpp-catalog",
    difficulty: 3,
    question: "Which cron syncs the Apple VPP catalog from Apple's API?",
    options: [
      "refresh_vpp_app_versions",
      "maintained_apps",
      "apple_mdm_apns_sweep",
      "vpp_sync",
    ],
    correctIndex: 0,
    explanation:
      "`refresh_vpp_app_versions` runs hourly and pulls the catalog from Apple's VPP API. There is no cron literally named `vpp_sync` in Fleet.",
    cronName: "refresh_vpp_app_versions",
  },

  // ─── Intermediate (4-6): "which cron for this symptom" ───
  {
    id: "autopilot-delay",
    difficulty: 4,
    question:
      "A customer registers a device in Microsoft Autopilot but it doesn't appear in Fleet for several minutes. Which cron is the delay?",
    options: [
      "mdm_windows_profile_manager (30s)",
      "microsoft_autopilot_sync (5m)",
      "integrations (1m)",
      "mdm_android_device_reconciler (1h)",
    ],
    correctIndex: 1,
    explanation:
      "`microsoft_autopilot_sync` runs every 5 minutes and pulls Autopilot registrations from Microsoft Graph. Until it runs, Fleet doesn't know about the newly-registered device.",
    cronName: "microsoft_autopilot_sync",
  },
  {
    id: "apple-profile-stuck",
    difficulty: 4,
    question:
      "An admin uploaded an Apple config profile 10 seconds ago. It isn't on hosts yet. Which cron is the delay?",
    options: [
      "integrations",
      "mdm_apple_profile_manager (30s)",
      "apple_mdm_apns_sweep (1m)",
      "upcoming_activities_maintenance (10m)",
    ],
    correctIndex: 1,
    explanation:
      "`mdm_apple_profile_manager` ticks every 30 seconds and reconciles desired vs installed profiles per host, enqueuing install commands. 10 seconds is just within one tick window.",
    cronName: "mdm_apple_profile_manager",
  },
  {
    id: "vpp-install-stuck",
    difficulty: 5,
    question:
      "A VPP install is stuck in pending for 90 seconds. Which cron drives the license assignment half of the flow?",
    options: [
      "refresh_vpp_app_versions",
      "integrations (drains the apple_software worker)",
      "mdm_apple_profile_manager",
      "apple_mdm_apns_sweep",
    ],
    correctIndex: 1,
    explanation:
      "The `apple_software` worker job calls Apple's VPP API to assign the license. The `integrations` cron (every minute) drains the worker queue that enqueues it. APNs push happens AFTER license assignment succeeds.",
    cronName: "integrations",
  },
  {
    id: "activity-stream-delay",
    difficulty: 5,
    question:
      "A customer streams Fleet's activity log to Splunk. Activities take up to 5 minutes to appear. Which cron explains that?",
    options: [
      "activities_streaming (5m)",
      "automations (24h default)",
      "integrations (1m)",
      "cleanups_then_aggregation (1h)",
    ],
    correctIndex: 0,
    explanation:
      "`activities_streaming` ticks every 5 minutes to push new activity rows out to the configured SIEM / webhook destination. The 5-minute cadence IS the delay.",
    cronName: "activities_streaming",
  },
  {
    id: "iphone-inventory-refresh",
    difficulty: 5,
    question:
      "Which cron refreshes iPhone / iPad inventory, given that osquery doesn't run on iOS?",
    options: [
      "apple_mdm_iphone_ipad_refetcher (10m)",
      "mdm_apple_profile_manager (30s)",
      "apple_mdm_iphone_ipad_reviver (1h)",
      "apple_mdm_dep_profile_assigner (1m)",
    ],
    correctIndex: 0,
    explanation:
      "`apple_mdm_iphone_ipad_refetcher` runs every 10 minutes and uses MDM's DeviceInformation command to refresh iOS/iPadOS inventory. The reviver (hourly) only attempts to re-wake dormant devices.",
    cronName: "apple_mdm_iphone_ipad_refetcher",
  },
  {
    id: "dep-profile-delay",
    difficulty: 5,
    question:
      "An admin changed a DEP enrollment profile. Which cron eventually pushes the change to Apple Business Manager?",
    options: [
      "apple_mdm_dep_profile_assigner (1m)",
      "mdm_apple_profile_manager (30s)",
      "integrations (1m)",
      "refresh_vpp_app_versions (1h)",
    ],
    correctIndex: 0,
    explanation:
      "`apple_mdm_dep_profile_assigner` pushes DEP profile assignments back to ABM every minute by default. `mdm_apple_profile_manager` reconciles .mobileconfig delivery, not DEP assignments.",
    cronName: "apple_mdm_dep_profile_assigner",
  },

  // ─── Intermediate 2 (6-7): interactions + edge cases ───
  {
    id: "policy-automation-delay",
    difficulty: 6,
    question:
      "A policy with an auto-install configured fires on a newly-failing host. The install doesn't appear in `upcoming_activities` for ~10 minutes. Why?",
    options: [
      "The host hasn't checked in yet",
      "`automations` cron ticks on its configured interval (commonly 10 minutes); it reads newly-failing membership and enqueues installs",
      "`integrations` cron is backlogged",
      "The vulnerabilities cron hasn't run yet",
    ],
    correctIndex: 1,
    explanation:
      "`automations` default is 24h but overridden by `WebhookSettings.Interval`. typically 10 minutes in real deployments. Policy evaluation + install enqueue both wait for that tick.",
    cronName: "automations",
  },
  {
    id: "fma-auto-update",
    difficulty: 6,
    question:
      "Fleet's FMA catalog bumps Chrome to a new version. Which cron rolls out the install to hosts that have auto-update enabled?",
    options: [
      "maintained_apps (just refreshes the catalog, no installs)",
      "maintained_apps_auto_update (1h)",
      "mdm_apple_profile_manager",
      "integrations",
    ],
    correctIndex: 1,
    explanation:
      "`maintained_apps` is the catalog refresh. `maintained_apps_auto_update` (hourly) is the separate cron that actually enqueues installs on hosts running an older version.",
    cronName: "maintained_apps_auto_update",
  },
  {
    id: "apns-sweep-vs-pusher",
    difficulty: 7,
    question:
      "Which cron is the MODERN APNs push mechanism in Fleet, replacing a legacy per-minute pusher?",
    options: [
      "apple_mdm_apns_pusher",
      "apple_mdm_apns_sweep (1m)",
      "mdm_apple_profile_manager",
      "end_user_notifications",
    ],
    correctIndex: 1,
    explanation:
      "`apple_mdm_apns_sweep` is the modern default (1-minute tick, re-pushes silent enrollments). `apple_mdm_apns_pusher` is the legacy per-minute pusher, only registered if `FLEET_MDM_APPLE_LEGACY_APNS_PUSHER_INTERVAL` is set.",
    cronName: "apple_mdm_apns_sweep",
  },
  {
    id: "worker-vs-cron",
    difficulty: 7,
    question:
      "What's the difference between the `apple_mdm_worker` cron (~10s) and the `apple_software` worker job?",
    options: [
      "They're the same thing",
      "`apple_mdm_worker` is a FAST schedule that drains worker jobs; `apple_software` is one of the JOB TYPES that gets drained (specifically VPP license-assignment)",
      "`apple_mdm_worker` is for Apple-silicon only",
      "`apple_software` is deprecated",
    ],
    correctIndex: 1,
    explanation:
      "`apple_mdm_worker` is a schedule that polls the jobs table; `apple_software` is a job type (VPP license assignment) handled when the queue tick fires. Confusing because both share a prefix.",
  },
  {
    id: "activity-aging-drops",
    difficulty: 7,
    question:
      "Old activity rows disappear after N months in a customer's deployment. Which cron handles that aging?",
    options: [
      "frequent_cleanups (15m)",
      "cleanups_then_aggregation (1h)",
      "activity_cleanup",
      "query_results_cleanup (1m)",
    ],
    correctIndex: 1,
    explanation:
      "`cleanups_then_aggregation` runs hourly and includes activity-row aging based on the admin's configured retention. `frequent_cleanups` handles different housekeeping (expired tokens, locks). There is no cron named `activity_cleanup` in Fleet.",
    cronName: "cleanups_then_aggregation",
  },

  // ─── Advanced (8-9): subtle distinctions ───
  {
    id: "profile-managers-identical",
    difficulty: 8,
    question:
      "Which statement about the three MDM profile managers (Apple / Windows / Android) is TRUE?",
    options: [
      "Only the Apple one runs at 30 seconds; Windows is 2 minutes",
      "All three profile-manager crons share the 30-second default interval. The architectural difference is who DELIVERS the command afterward (Apple MDM / Windows pull / AMAPI).",
      "Windows runs at 2 minutes because SyncML commands take longer",
      "Android runs every 10 minutes because Google rate-limits AMAPI",
    ],
    correctIndex: 1,
    explanation:
      "The cron pacing is identical across the three; the delivery mechanism after the cron is what differs.",
  },
  {
    id: "cron-vs-config-tunable",
    difficulty: 8,
    question:
      "Which cron's hardcoded default is 24h but is overridden by an admin config (e.g., `WebhookSettings.Interval`)?",
    options: [
      "integrations",
      "automations",
      "vulnerabilities",
      "frequent_cleanups",
    ],
    correctIndex: 1,
    explanation:
      "`automations` hard default is 24h, but it reads `WebhookSettings.Interval` at startup and typically runs at 10m in deployments. `vulnerabilities` is also admin-configurable (via `config.Periodicity`) but has no 24h default.",
    cronName: "automations",
  },
  {
    id: "fast-crons-list",
    difficulty: 8,
    question:
      "Which combination lists ALL the Fleet crons that tick at 30 seconds by default?",
    options: [
      "Just mdm_apple_profile_manager",
      "mdm_apple_profile_manager + mdm_windows_profile_manager + mdm_android_profile_manager + send_recovery_lock_commands",
      "Any cron whose name starts with mdm_",
      "Any cron registered in server/mdm/",
    ],
    correctIndex: 1,
    explanation:
      "Four crons share the 30-second cadence: the three profile managers plus `send_recovery_lock_commands`. The APNs sweep ticks at 1m, not 30s.",
  },
  {
    id: "db-encryption-rotation",
    difficulty: 9,
    question:
      "Which cron rotates escrowed BitLocker recovery keys after an org-key change?",
    options: [
      "reencrypt_windows_disk_encryption_keys (1h)",
      "send_recovery_lock_commands (30s)",
      "mdm_windows_profile_manager (30s)",
      "frequent_cleanups (15m)",
    ],
    correctIndex: 0,
    explanation:
      "`reencrypt_windows_disk_encryption_keys` runs hourly to re-encrypt stored BitLocker keys with the current org key. `send_recovery_lock_commands` is a different cron that targets Apple Silicon Macs' Recovery Lock feature.",
    cronName: "reencrypt_windows_disk_encryption_keys",
  },
  {
    id: "legacy-apns-env",
    difficulty: 10,
    question:
      "Under what env var does Fleet register the LEGACY `apple_mdm_apns_pusher` instead of the modern `apple_mdm_apns_sweep`?",
    options: [
      "FLEET_LEGACY_APNS",
      "FLEET_MDM_APPLE_LEGACY_APNS_PUSHER_INTERVAL",
      "FLEET_APNS_FALLBACK=1",
      "FLEET_MDM_APPLE_APNS_SWEEP_DISABLED=1",
    ],
    correctIndex: 1,
    explanation:
      "`FLEET_MDM_APPLE_LEGACY_APNS_PUSHER_INTERVAL` is the rollback lever. If set to any duration, Fleet registers the legacy per-minute pusher instead of the sweep. Defined in `cmd/fleet/cron_registration.go:legacyAPNsPusherInterval`.",
    cronName: "apple_mdm_apns_pusher (legacy)",
  },
];

export function pickAdaptive(
  targetDifficulty: number,
  asked: Set<string>,
): CronQuestion | null {
  // Find questions within +/- 2 difficulty of target, unasked, preferring
  // exact-match difficulty.
  const pool = CRON_QUESTIONS.filter(
    (q) => !asked.has(q.id) && Math.abs(q.difficulty - targetDifficulty) <= 2,
  );
  if (pool.length === 0) {
    // Expand to any unasked question; if nothing, return null (quiz exhausted).
    const anyUnasked = CRON_QUESTIONS.filter((q) => !asked.has(q.id));
    if (anyUnasked.length === 0) return null;
    return anyUnasked[Math.floor(Math.random() * anyUnasked.length)];
  }
  pool.sort(
    (a, b) =>
      Math.abs(a.difficulty - targetDifficulty) -
      Math.abs(b.difficulty - targetDifficulty),
  );
  // Pick randomly within the top-tier closest-difficulty group.
  const bestDelta = Math.abs(pool[0].difficulty - targetDifficulty);
  const top = pool.filter(
    (q) => Math.abs(q.difficulty - targetDifficulty) === bestDelta,
  );
  return top[Math.floor(Math.random() * top.length)];
}
