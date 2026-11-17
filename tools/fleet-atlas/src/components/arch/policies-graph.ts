import type { ArchEdge, ArchNode } from "./graph-data";

const COL = 220;
const ROW = 150;

export const POLICIES_NODES: ArchNode[] = [
  // Row 0. Policy sources
  {
    id: "pol-source-ui",
    label: "Admin UI (SQL editor)",
    group: "external",
    blurb: "Admin writes/edits a policy SQL in the UI; POSTs to the API.",
    repoPaths: [{ path: "frontend/pages/policies/" }],
    x: 0,
    y: 0,
  },
  {
    id: "pol-source-gitops",
    label: "GitOps YAML",
    group: "external",
    blurb: "Policies declared declaratively; fleetctl gitops reconciles.",
    repoPaths: [{ path: "cmd/fleetctl/" }],
    x: COL,
    y: 0,
  },
  {
    id: "pol-source-maintained",
    label: "Fleet-maintained policies",
    group: "external",
    blurb:
      "Policy library shipped with Fleet (e.g. CIS benchmarks, common vuln checks).",
    repoPaths: [{ path: "ee/server/policies/" }],
    x: COL * 2,
    y: 0,
  },

  // Row 1. Policy storage
  {
    id: "pol-store-policies",
    label: "policies (MySQL)",
    group: "store",
    blurb:
      "Each row = one policy: query SQL, name, platform, fleet scope, automation config.",
    repoPaths: [{ path: "server/datastore/mysql/policies.go" }],
    x: COL * 0.5,
    y: ROW,
  },
  {
    id: "pol-store-labels",
    label: "policy_labels",
    group: "store",
    blurb: "Narrows policy evaluation to labeled host subsets.",
    repoPaths: [{ path: "server/datastore/mysql/policies.go" }],
    x: COL * 1.5,
    y: ROW,
  },

  // Row 2. Distribution (via osquery scheduled queries)
  {
    id: "pol-dist-schedule",
    label: "Policies piggyback on osquery scheduled queries",
    group: "service",
    blurb:
      "A policy's SQL is included in the config a host pulls. No separate protocol.",
    repoPaths: [{ path: "server/service/osquery.go" }],
    x: COL,
    y: ROW * 2,
  },
  {
    id: "pol-dist-pull",
    label: "Host config pull (every ~10s)",
    group: "background",
    blurb:
      "fleetd calls POST /api/osquery/config. Policy SQLs are included and refreshed on cadence.",
    repoPaths: [{ path: "server/service/osquery.go" }],
    x: COL,
    y: ROW * 3,
  },

  // Row 3. Host-side evaluation
  {
    id: "pol-eval-host",
    label: "osqueryd runs the policy SQL",
    group: "external",
    blurb:
      "Policy convention: if the query returns rows, host is PASSING; empty result = FAILING.",
    repoPaths: [],
    x: COL,
    y: ROW * 4,
  },

  // Row 4. Result ingestion
  {
    id: "pol-ingest-log",
    label: "POST /api/osquery/log",
    group: "ingress",
    blurb:
      "fleetd posts scheduled-query (policy) results back to Fleet with the host's row count.",
    repoPaths: [{ path: "server/service/osquery.go" }],
    x: COL,
    y: ROW * 5,
  },
  {
    id: "pol-ingest-service",
    label: "svc.RecordPolicyResults",
    group: "service",
    blurb:
      "Translates row-count-per-policy into pass/fail; writes policy_membership.",
    repoPaths: [{ path: "server/service/policies.go" }],
    x: COL,
    y: ROW * 6,
  },

  // Row 5. Persistence
  {
    id: "pol-store-membership",
    label: "policy_membership",
    group: "store",
    blurb: "Host ↔ policy ↔ passing/failing. Hot write path.",
    repoPaths: [{ path: "server/datastore/mysql/policies.go" }],
    x: 0,
    y: ROW * 7,
  },
  {
    id: "pol-store-stats",
    label: "policy_stats + host_issues",
    group: "store",
    blurb:
      "Pre-aggregated pass/fail counts (powers UI) and per-host failing counts (powers dashboard tiles).",
    repoPaths: [{ path: "server/datastore/mysql/policies.go" }],
    x: COL * 2,
    y: ROW * 7,
  },

  // Row 6. Automations cron
  {
    id: "pol-automations-cron",
    label: "automations cron (10m)",
    group: "background",
    premium: true,
    blurb:
      "Reads newly-failing policy_membership rows for policies that have automations configured.",
    repoPaths: [{ path: "cmd/fleet/cron.go" }],
    x: COL,
    y: ROW * 8,
  },

  // Row 7. Dedup
  {
    id: "pol-dedup",
    label: "host_policy_automation_run_counts",
    group: "store",
    premium: true,
    blurb:
      "Counter that prevents re-firing the same automation for the same host-policy pair while remediation is in flight.",
    repoPaths: [{ path: "server/datastore/mysql/policies.go" }],
    x: COL,
    y: ROW * 9,
  },

  // Row 8. Automation actions (branches)
  {
    id: "pol-action-install",
    label: "Software install",
    group: "service",
    premium: true,
    blurb:
      "Calls svc.InstallSoftwareTitle for each failing host where auto-install is configured.",
    repoPaths: [{ path: "server/service/software_installers.go" }],
    relatedModes: [
      { href: "/scenarios/policy-driven-auto-install", label: "Scenario" },
    ],
    x: COL * -0.5,
    y: ROW * 10,
  },
  {
    id: "pol-action-ticket",
    label: "Jira / Zendesk worker",
    group: "background",
    premium: true,
    blurb: "Enqueues a ticket-create job per failing host.",
    repoPaths: [
      { path: "server/worker/jira.go" },
      { path: "server/worker/zendesk.go" },
    ],
    x: COL * 0.5,
    y: ROW * 10,
  },
  {
    id: "pol-action-webhook",
    label: "Generic webhook",
    group: "background",
    premium: true,
    blurb: "Fires a configured webhook URL via worker.",
    repoPaths: [{ path: "server/service/webhooks.go" }],
    x: COL * 1.5,
    y: ROW * 10,
  },
  {
    id: "pol-action-calendar",
    label: "Calendar event",
    group: "background",
    premium: true,
    blurb:
      "calendar cron creates/updates a Google or MS calendar event on the end user's calendar.",
    repoPaths: [{ path: "server/cron/calendar_cron.go" }],
    x: COL * 2.5,
    y: ROW * 10,
  },
  {
    id: "pol-action-script",
    label: "Script run",
    group: "service",
    premium: true,
    blurb:
      "Admin can configure a remediation script to run on failing hosts.",
    repoPaths: [{ path: "server/service/scripts.go" }],
    x: COL * 3.5,
    y: ROW * 10,
  },

  // Row 9. External delivery / effects
  {
    id: "pol-delivery-ext",
    label: "External effect (ticket / event / install result)",
    group: "external",
    premium: true,
    blurb:
      "Jira issue, Zendesk ticket, calendar invite, posted webhook, remediation install, script output.",
    repoPaths: [],
    x: COL,
    y: ROW * 11,
  },
];

export const POLICIES_EDGES: ArchEdge[] = [
  // Sources → storage
  {
    from: "pol-source-ui",
    to: "pol-store-policies",
    tooltip:
      "Admin writes a policy's SQL in the UI editor; POST creates or updates a row in the policies table. The UI is a thin shell around the API. Any policy created here is identical to one authored via GitOps.",
  },
  {
    from: "pol-source-gitops",
    to: "pol-store-policies",
    tooltip:
      "fleetctl gitops writes YAML-declared policies into the same policies table. Reconciles the full desired set: policies in the YAML but not in Fleet get created; policies in Fleet but not in the YAML get deleted.",
  },
  {
    from: "pol-source-maintained",
    to: "pol-store-policies",
    tooltip:
      "Fleet ships a built-in library of common policies (CIS benchmarks, known vuln checks). Admins import them into their deployment, which copies the policy row into their policies table. After import, they're editable like any admin-authored policy.",
  },
  {
    from: "pol-store-policies",
    to: "pol-store-labels",
    dashed: true,
    tooltip:
      "A policy can be scoped to only hosts carrying certain labels. policy_labels is the join table: (policy_id, label_id, include_any|include_all|exclude). The evaluator reads this to narrow which hosts get the policy in their config.",
  },

  // Storage → distribution
  {
    from: "pol-store-policies",
    to: "pol-dist-schedule",
    tooltip:
      "At config-build time, each policy's SQL is bundled into the osquery scheduled-queries section of the host's config. No separate policy protocol. Policies ride along with regular scheduled queries, with a flag marking them as policies.",
  },
  {
    from: "pol-dist-schedule",
    to: "pol-dist-pull",
    tooltip:
      "The scheduled-queries bundle is served inline when fleetd calls POST /api/osquery/config (every ~10s). Policy edits reach hosts on the next config poll. Not instantly, but within a few seconds.",
  },
  {
    from: "pol-dist-pull",
    to: "pol-eval-host",
    tooltip:
      "fleetd hands the policy SQL to osqueryd on the device, which runs it locally against the device's own tables. Policy convention: if the query returns at least one row, the host is PASSING; if it returns zero rows, the host is FAILING.",
  },

  // Evaluation → ingestion
  {
    from: "pol-eval-host",
    to: "pol-ingest-log",
    tooltip:
      "osqueryd posts the policy result (host row count per policy) back to Fleet via POST /api/osquery/log, bundled with any other scheduled-query results it has to send.",
  },
  {
    from: "pol-ingest-log",
    to: "pol-ingest-service",
    tooltip:
      "The osquery log endpoint hands incoming policy results to svc.RecordPolicyResults, which is where the row-count-per-policy becomes a pass/fail verdict Fleet can act on.",
  },

  // Persistence
  {
    from: "pol-ingest-service",
    to: "pol-store-membership",
    tooltip:
      "svc.RecordPolicyResults writes one row per (host_id, policy_id) into policy_membership with the passing/failing verdict. This is the hot write path during policy ingest. Every host runs every policy on cadence.",
  },
  {
    from: "pol-ingest-service",
    to: "pol-store-stats",
    dashed: true,
    tooltip:
      "After membership rows update, Fleet also recomputes pre-aggregated counts (policy_stats powers per-policy pass/fail counts in the UI; host_issues powers per-host failing count tiles). Live GROUP BY would be too slow at scale.",
  },

  // Membership → automations
  {
    from: "pol-store-membership",
    to: "pol-automations-cron",
    label: "failing rows",
    tooltip:
      "The automations cron (10m) queries policy_membership for rows where the host flipped from passing to failing since the cron's last run. Only policies with automations configured are checked. The rest are ignored.",
  },
  {
    from: "pol-automations-cron",
    to: "pol-dedup",
    label: "check counter",
    tooltip:
      "Before firing an automation for a (host, policy) pair, the cron checks host_policy_automation_run_counts. If an automation already fired for this pair and remediation is still pending, skip to avoid duplicating tickets/installs/calendar events.",
  },

  // Automations → actions
  {
    from: "pol-dedup",
    to: "pol-action-install",
    tooltip:
      "For policies with an install action configured, the automation calls svc.InstallSoftwareTitle on the failing host. Loops back into the software-install flow. Same code path as the Install button.",
  },
  {
    from: "pol-dedup",
    to: "pol-action-ticket",
    tooltip:
      "For policies with a Jira/Zendesk integration configured, the automation enqueues a ticket-create worker job carrying the failing host and policy context. integrations cron (1m) later drains it.",
  },
  {
    from: "pol-dedup",
    to: "pol-action-webhook",
    tooltip:
      "For policies configured with a generic webhook URL, the automation enqueues a webhook-fire worker job. The webhook POST goes out carrying the policy + host context as JSON.",
  },
  {
    from: "pol-dedup",
    to: "pol-action-calendar",
    tooltip:
      "For policies configured with a calendar integration, the calendar cron creates or updates an event on the failing user's Google or Microsoft calendar. E.g. a block titled \"Fix your Mac's compliance\" during work hours.",
  },
  {
    from: "pol-dedup",
    to: "pol-action-script",
    tooltip:
      "For policies configured with a remediation script, the automation queues a script run against the failing host. fleetd pulls the script on its next poll and executes it locally.",
  },

  // Actions → external delivery
  {
    from: "pol-action-install",
    to: "pol-delivery-ext",
    tooltip:
      "The downstream install flow eventually places software on the host, which (if remediation succeeds) causes the policy to flip back to passing on the next evaluation cycle.",
  },
  {
    from: "pol-action-ticket",
    to: "pol-delivery-ext",
    tooltip:
      "The worker calls Jira/Zendesk's API with Fleet's creds; a ticket appears in the ops team's queue referencing the host and policy.",
  },
  {
    from: "pol-action-webhook",
    to: "pol-delivery-ext",
    tooltip:
      "The worker POSTs to the admin's configured webhook URL; downstream consumers (PagerDuty, Slack, custom tooling) react based on the payload.",
  },
  {
    from: "pol-action-calendar",
    to: "pol-delivery-ext",
    tooltip:
      "The calendar event lands on the user's work calendar; the user sees a nudging block of time, remediation happens, policy re-passes.",
  },
  {
    from: "pol-action-script",
    to: "pol-delivery-ext",
    tooltip:
      "Script output and exit code are reported back to Fleet. If the script succeeded in fixing the state the policy checks, the next evaluation flips the host to passing.",
  },
];
