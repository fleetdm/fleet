// Fleet's six background queues matched to their storage + drain profile.
// Teaches the "same enqueue-wait-drain shape, different storage and different
// drain mechanism" pattern. The whole lesson of the queues page.

export type QueueStoragePair = {
  key: string;
  queue: string;
  profile: string;
};

export const QUEUE_STORAGE_PAIRS: QueueStoragePair[] = [
  {
    key: "worker",
    queue: "Worker queue",
    profile:
      "MySQL jobs table (durable). Drained by the integrations cron (every 1m). On failure, retry_count bumps and run_after advances with backoff.",
  },
  {
    key: "vpp-install",
    queue: "VPP install queue",
    profile:
      "MySQL upcoming_activities + vpp_app_upcoming_activities (durable). Drained by an apple_software worker job via the integrations cron, which then calls Apple's VPP API.",
  },
  {
    key: "custom-install",
    queue: "Custom installer queue",
    profile:
      "MySQL upcoming_activities + software_install_upcoming_activities (durable). Drained by the host's own osquery /config poll (~10s). No cron, no push; pure pull.",
  },
  {
    key: "mdm-apple-command",
    queue: "Apple MDM command queue",
    profile:
      "MySQL nano_commands + nano_enrollment_queue (durable, nanomdm's schema). Drained when apple_mdm_apns_pusher (1m) sends an APNs push and the device then polls Fleet's MDM endpoint.",
  },
  {
    key: "live-query-pubsub",
    queue: "Live query result pubsub",
    profile:
      "Redis pubsub channel keyed by campaign uuid (ephemeral. No persistence). Drained by any Fleet instance holding an admin's WebSocket subscribed to that channel.",
  },
  {
    key: "scheduled-batch",
    queue: "Scheduled batch activity queue",
    profile:
      "MySQL batch_activities row with a future run_at (durable). Drained by the scheduled_batch_activities cron (1m) when run_at has passed, which fans out to per-host children.",
  },
];
