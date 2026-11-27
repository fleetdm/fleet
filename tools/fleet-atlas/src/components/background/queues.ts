export type QueueNode = {
  kind: "entry" | "storage" | "drain" | "exit";
  label: string;
  detail?: string;
  file?: string;
};

export type QueueFlow = {
  slug: string;
  label: string;
  one_liner: string;
  storageType: "MySQL" | "Redis" | "Memory";
  overview: string;
  nodes: QueueNode[];
};

export const QUEUES: QueueFlow[] = [
  {
    slug: "worker",
    label: "Worker queue",
    one_liner:
      "Durable job queue in MySQL for anything that calls an external API.",
    storageType: "MySQL",
    overview:
      "The service enqueues a row; the integrations cron picks it up and runs the job function; on failure, the row is re-queued with backoff.",
    nodes: [
      {
        kind: "entry",
        label: "worker.QueueJob(...)",
        detail:
          "Called from service methods after a mutation that needs an async side-effect (ticket create, VPP verify, etc.).",
        file: "server/worker/worker.go",
      },
      {
        kind: "storage",
        label: "jobs (MySQL)",
        detail: "One row per queued job. Has state, retry count, run-after time.",
      },
      {
        kind: "drain",
        label: "integrations cron (every 1m)",
        detail:
          "Pulls N claimed rows, runs each job function (jira, zendesk, apple_mdm, apple_software, software_worker, …).",
      },
      {
        kind: "exit",
        label: "External API call",
        detail:
          "Jira, Zendesk, Apple VPP, nanomdm. On success the row is deleted; on failure, retry count bumps and run_after advances.",
      },
    ],
  },
  {
    slug: "vpp-install",
    label: "VPP install queue",
    one_liner:
      "A VPP install goes through both the upcoming_activities queue AND the worker queue.",
    storageType: "MySQL",
    overview:
      "VPP is interesting because two queues chain: the admin's click lands in upcoming_activities, and the worker queue carries the actual Apple VPP API call. Apple then takes over.",
    nodes: [
      {
        kind: "entry",
        label: "Admin clicks Install (VPP title)",
        detail: "POST /api/v1/fleet/hosts/{id}/software/{title_id}/install",
      },
      {
        kind: "storage",
        label: "upcoming_activities + vpp_app_upcoming_activities + host_vpp_software_installs",
        detail: "pending intent, visible in UI immediately",
      },
      {
        kind: "drain",
        label: "Service enqueues apple_software worker job",
        detail:
          "The service immediately queues a VPP verification job in the worker table.",
        file: "server/worker/vpp_verification.go",
      },
      {
        kind: "storage",
        label: "jobs (MySQL) with name='apple_software'",
      },
      {
        kind: "drain",
        label: "integrations cron drains (every 1m)",
      },
      {
        kind: "exit",
        label: "Apple VPP API assignment",
        detail:
          "Worker calls Apple's VPP API: 'assign this license to this device.' Apple pushes InstallApplication via APNs. Device installs; MDM ACKs; Fleet flips host_vpp_software_installs to installed.",
      },
    ],
  },
  {
    slug: "custom-install",
    label: "Custom installer queue",
    one_liner:
      "The 'Install' button for a .pkg/.msi queues work the host picks up on its next osquery poll.",
    storageType: "MySQL",
    overview:
      "The user-visible queue is upcoming_activities. The host polls Fleet on its 10s cycle; when the config response says 'you have pending installs,' fleetd downloads the bytes and runs the install.",
    nodes: [
      {
        kind: "entry",
        label: "Admin or self-service click",
        detail:
          "POST /api/v1/fleet/hosts/{id}/software/{title_id}/install OR POST /device/{token}/software/install/{title_id}",
      },
      {
        kind: "storage",
        label: "upcoming_activities + software_install_upcoming_activities + host_software_installs",
        detail: "status=pending",
      },
      {
        kind: "drain",
        label: "Host osquery poll (every ~10s)",
        detail:
          "fleetd calls POST /api/osquery/config. Fleet's response includes pending install hints.",
      },
      {
        kind: "exit",
        label: "fleetd downloads bytes, runs scripts, POSTs result",
        detail:
          "Bytes streamed from the file store. Scripts run locally. Result posted to /api/fleet/orbit/software_install_result; Fleet flips the install row to installed/failed.",
      },
    ],
  },
  {
    slug: "mdm-apple-command",
    label: "Apple MDM command queue",
    one_liner:
      "nanomdm's command queue. Fleet enqueues commands; device pulls on APNs push.",
    storageType: "MySQL",
    overview:
      "The command plane for Apple MDM. Different from the osquery and software install queues. Lives in nano_commands + nano_enrollment_queue (the embedded nanomdm's schema).",
    nodes: [
      {
        kind: "entry",
        label: "Admin action or reconciler cron",
        detail:
          "Admin clicks Lock/Wipe/InstallProfile; or mdm_apple_profile_manager cron diffs desired vs. installed and enqueues.",
      },
      {
        kind: "storage",
        label: "nano_commands + nano_enrollment_queue (MySQL)",
        detail: "Fleet-local persistence of the MDM command plane",
      },
      {
        kind: "drain",
        label: "apple_mdm_apns_pusher cron (every 1m)",
        detail:
          "Sends an APNs push to each device with pending commands: 'wake up and check in.'",
      },
      {
        kind: "exit",
        label: "Device polls MDM endpoint, Fleet returns command",
        detail:
          "Device sends an Idle status report to Fleet's MDM endpoint; Fleet responds with the queued plist command; device executes; device ACKs; Fleet writes to nano_command_results.",
      },
    ],
  },
  {
    slug: "live-query-pubsub",
    label: "Live query result pubsub",
    one_liner:
      "Not a persistent queue: a Redis pubsub channel per campaign, fan-out across Fleet instances.",
    storageType: "Redis",
    overview:
      "Hosts respond to live queries over seconds to minutes. Results need to reach the admin's browser live. Redis pubsub is the fan-out plane; MySQL is only involved for the campaign record.",
    nodes: [
      {
        kind: "entry",
        label: "Host posts rows to /api/osquery/distributed/write",
        detail: "The Fleet instance that receives the host's write publishes to Redis.",
      },
      {
        kind: "storage",
        label: "Redis pubsub channel: campaign_<uuid>",
        detail: "Ephemeral; no durable queue. Dropped rows are lost.",
      },
      {
        kind: "drain",
        label: "Subscribed Fleet instances receive publish",
        detail:
          "Any Fleet instance holding an admin's WebSocket for this campaign gets the publish.",
      },
      {
        kind: "exit",
        label: "WebSocket sends row to admin UI",
        detail: "UI appends row to the live result table.",
      },
    ],
  },
  {
    slug: "scheduled-batch",
    label: "Scheduled batch activity queue",
    one_liner:
      "Admins can schedule a batch install/run-script for a future time. Rows wait in MySQL until then.",
    storageType: "MySQL",
    overview:
      "Batch activities split into parent (batch_activities) + child (upcoming_activities per host). Scheduled ones sit waiting; the scheduled_batch_activities cron releases them at the configured time.",
    nodes: [
      {
        kind: "entry",
        label: "Admin schedules a batch install",
        detail: "e.g. 'install Chrome 120 on 500 hosts next Monday at 10am.'",
      },
      {
        kind: "storage",
        label: "batch_activities with status=scheduled",
        detail: "run_at is in the future.",
      },
      {
        kind: "drain",
        label: "scheduled_batch_activities cron (every 1m)",
        detail:
          "Checks for batches whose run_at has passed.",
      },
      {
        kind: "storage",
        label: "batch_activity_host_results + per-host upcoming_activities",
        detail: "Fan-out to one child row per targeted host.",
      },
      {
        kind: "drain",
        label: "Per-host install mechanics (same as Custom installer queue)",
      },
      {
        kind: "exit",
        label: "batch_activity_completion_checker closes the batch",
        detail: "Marks parent complete when all per-host children have finished.",
      },
    ],
  },
];
