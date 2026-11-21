import type { ArchEdge, ArchNode } from "./graph-data";

const COL = 220;
const ROW = 150;

export const REPORTS_NODES: ArchNode[] = [
  // Row 0. Report sources
  {
    id: "rep-source-saved",
    label: "Saved reports (UI)",
    group: "external",
    blurb: "Admin creates/edits a query (now called a report) in Fleet's UI.",
    repoPaths: [{ path: "frontend/pages/queries/" }],
    x: 0,
    y: 0,
  },
  {
    id: "rep-source-live",
    label: "Live query (ad-hoc)",
    group: "external",
    blurb:
      "Admin clicks Run against a label-targeted host set. One-shot distributed campaign.",
    repoPaths: [{ path: "server/service/async/" }],
    relatedModes: [{ href: "/scenarios/live-query", label: "Scenario" }],
    x: COL,
    y: 0,
  },
  {
    id: "rep-source-schedule",
    label: "Scheduled report (GitOps/UI)",
    group: "external",
    blurb: "Report attached to a cadence (every N seconds); runs on the host automatically.",
    repoPaths: [{ path: "server/service/queries.go" }],
    x: COL * 2,
    y: 0,
  },
  {
    id: "rep-source-policy",
    label: "Policy query",
    group: "external",
    blurb:
      "Policies are secretly saved queries that evaluate pass/fail based on row count.",
    repoPaths: [{ path: "server/datastore/mysql/policies.go" }],
    x: COL * 3,
    y: 0,
  },
  {
    id: "rep-source-maintained",
    label: "Fleet-maintained queries",
    group: "external",
    blurb:
      "Query library shipped with Fleet (CVE checks, inventory helpers, etc.).",
    repoPaths: [{ path: "ee/server/queries/" }],
    x: COL * 4,
    y: 0,
  },

  // Row 1. Storage (two distinct families)
  {
    id: "rep-store-queries",
    label: "queries + scheduled_queries",
    group: "store",
    blurb:
      "Saved query definitions and their schedule (interval, platform, log destination).",
    repoPaths: [{ path: "server/datastore/mysql/queries.go" }],
    x: COL,
    y: ROW,
  },
  {
    id: "rep-store-campaigns",
    label: "distributed_query_campaigns + targets",
    group: "store",
    blurb:
      "One row per live-query run. UUID becomes the Redis pubsub channel name.",
    repoPaths: [{ path: "server/datastore/mysql/campaigns.go" }],
    x: COL * 3,
    y: ROW,
  },

  // Row 2. Distribution
  {
    id: "rep-dist-config",
    label: "osquery /config: scheduled queries baked in",
    group: "ingress",
    blurb:
      "Host's config response includes every scheduled query it should run (filtered by platform + fleet).",
    repoPaths: [{ path: "server/service/osquery.go" }],
    x: 0,
    y: ROW * 2,
  },
  {
    id: "rep-dist-distributed",
    label: "osquery /distributed/read: live queries delivered",
    group: "ingress",
    blurb:
      "Each host's next distributed/read pull returns any pending live-query SQL targeting it.",
    repoPaths: [{ path: "server/service/osquery.go" }],
    x: COL * 3,
    y: ROW * 2,
  },

  // Row 3. Host-side execution
  {
    id: "rep-exec-scheduled",
    label: "osqueryd runs scheduled queries on cadence",
    group: "external",
    blurb:
      "Scheduled queries run per their interval. Results logged on logger_tls cadence.",
    repoPaths: [],
    x: 0,
    y: ROW * 3,
  },
  {
    id: "rep-exec-live",
    label: "osqueryd runs live query once",
    group: "external",
    blurb: "One execution; results posted back immediately.",
    repoPaths: [],
    x: COL * 3,
    y: ROW * 3,
  },

  // Row 4. Ingestion
  {
    id: "rep-ingest-log",
    label: "POST /api/osquery/log",
    group: "ingress",
    blurb:
      "Scheduled-query results stream here. Also carries osquery's own status + logs.",
    repoPaths: [{ path: "server/service/osquery.go" }],
    x: 0,
    y: ROW * 4,
  },
  {
    id: "rep-ingest-write",
    label: "POST /api/osquery/distributed/write",
    group: "ingress",
    blurb: "Live-query results posted here.",
    repoPaths: [{ path: "server/service/osquery.go" }],
    x: COL * 3,
    y: ROW * 4,
  },

  // Row 5. Persistence / pubsub
  {
    id: "rep-persist-results",
    label: "query_results + scheduled_query_stats",
    group: "store",
    blurb:
      "Materialized rows + per-run performance stats for scheduled reports.",
    repoPaths: [{ path: "server/datastore/mysql/queries.go" }],
    x: 0,
    y: ROW * 5,
  },
  {
    id: "rep-persist-policy-mem",
    label: "policy_membership",
    group: "store",
    blurb:
      "Policy-query results write here (via a different ingestion branch).",
    repoPaths: [{ path: "server/datastore/mysql/policies.go" }],
    x: COL * 1.5,
    y: ROW * 5,
  },
  {
    id: "rep-pubsub",
    label: "Redis pubsub: campaign_<uuid>",
    group: "store",
    blurb:
      "Live-query result rows fan out via Redis pubsub so admins on any Fleet instance can subscribe.",
    repoPaths: [{ path: "server/pubsub/redis_query_results.go" }],
    relatedModes: [{ href: "/data/redis", label: "Redis" }],
    x: COL * 3,
    y: ROW * 5,
  },

  // Row 6. Consumers
  {
    id: "rep-consume-ws",
    label: "WebSocket → Admin UI",
    group: "ingress",
    blurb:
      "The admin's browser subscribes to the campaign's channel through a WebSocket Fleet holds open.",
    repoPaths: [{ path: "server/service/async/" }],
    x: COL * 3,
    y: ROW * 6,
  },
  {
    id: "rep-consume-log",
    label: "External log destination",
    group: "external",
    blurb:
      "Optional firehose: Fleet forwards scheduled results to a configured log endpoint (S3, Snowflake, Splunk, Kinesis).",
    repoPaths: [{ path: "server/logging/" }],
    x: 0,
    y: ROW * 6,
  },
  {
    id: "rep-consume-dashboard",
    label: "Dashboard aggregates",
    group: "store",
    blurb:
      "Scheduled-query results drive many dashboard aggregates (kernel_host_counts, aggregated_stats).",
    repoPaths: [{ path: "server/datastore/mysql/aggregated_stats.go" }],
    x: COL * 1.5,
    y: ROW * 6,
  },

  // Row 7. Secondary flows
  {
    id: "rep-secondary-carves",
    label: "File carves",
    group: "store",
    blurb:
      "Queries that invoke carve() upload file chunks to carve_metadata + carve_blocks.",
    repoPaths: [{ path: "server/datastore/mysql/carves.go" }],
    x: COL * 3,
    y: ROW * 7,
  },
];

export const REPORTS_EDGES: ArchEdge[] = [
  // Sources → storage
  {
    from: "rep-source-saved",
    to: "rep-store-queries",
    tooltip:
      "Admin saves a query (what the product now calls a report) through the UI; POST writes a row to the queries table. No schedule attached. Just the SQL and metadata.",
  },
  {
    from: "rep-source-schedule",
    to: "rep-store-queries",
    tooltip:
      "A report with an attached cadence lives in queries + scheduled_queries. scheduled_queries carries the interval, platform filter, log destination; queries carries the SQL. Running on a cadence requires rows in both.",
  },
  {
    from: "rep-source-policy",
    to: "rep-store-queries",
    dashed: true,
    tooltip:
      "A policy is a saved query with pass/fail semantics attached. The SQL is written into the queries table the same way as any report; what makes it a policy is a separate row in the policies table referencing it + the row-count-based pass/fail convention.",
  },
  {
    from: "rep-source-maintained",
    to: "rep-store-queries",
    tooltip:
      "Fleet ships a library of built-in reports (CVE detection helpers, inventory enrichment, OS version detail). When an admin imports one, Fleet copies the query definition into their queries table. Same as any other report after import.",
  },
  {
    from: "rep-source-live",
    to: "rep-store-campaigns",
    tooltip:
      "When an admin clicks Run on a live query, Fleet writes a one-shot row to distributed_query_campaigns with the SQL + target host set. The campaign's UUID doubles as the Redis pubsub channel name. Results will fan out through that channel until the campaign ends.",
  },

  // Storage → distribution
  {
    from: "rep-store-queries",
    to: "rep-dist-config",
    tooltip:
      "At config-build time, Fleet reads scheduled_queries + queries and bundles the scheduled reports a host should run into its osquery config response. Platform filters and label scopes narrow which reports go to which hosts.",
  },
  {
    from: "rep-store-campaigns",
    to: "rep-dist-distributed",
    tooltip:
      "Each host targeted by an in-flight campaign gets the campaign's SQL attached to its next POST /api/osquery/distributed/read response. Live queries don't ride on the config. They travel the distributed endpoint, which is osquery's ad-hoc channel.",
    techHelp:
      "osquery exposes two distinct channels for server-to-device SQL:\n\n- **/config**: scheduled queries that run on a cadence. osquery persists them in its local schedule and runs them on their own intervals.\n- **/distributed/read**: one-shot queries the server wants to run RIGHT NOW. Single-run SQL that osquery doesn't persist.\n\nBoth are polled on roughly the same ~10s cadence. Live queries use the distributed channel because they shouldn't become part of the host's standing schedule.",
  },

  // Distribution → execution
  {
    from: "rep-dist-config",
    to: "rep-exec-scheduled",
    tooltip:
      "osqueryd on the device runs each scheduled report at its own interval (e.g. a 1-hour report runs 24 times a day). The device never calls back out just to run them. Execution is purely local, driven by the osquery scheduler.",
  },
  {
    from: "rep-dist-distributed",
    to: "rep-exec-live",
    tooltip:
      "osqueryd runs the live-query SQL once, in the context of its next distributed/read cycle. One shot. No cadence, no persistence of the SQL beyond this run.",
  },

  // Execution → ingestion
  {
    from: "rep-exec-scheduled",
    to: "rep-ingest-log",
    tooltip:
      "Scheduled-report results stream back to POST /api/osquery/log on osquery's logger_tls cadence (not necessarily on each query run. Osquery batches). This endpoint also carries osquery's own status + info logs.",
  },
  {
    from: "rep-exec-live",
    to: "rep-ingest-write",
    tooltip:
      "Live-query results POST immediately to /api/osquery/distributed/write as soon as the one-shot query finishes. No batching. The admin is waiting on the other end.",
  },

  // Ingestion → persistence
  {
    from: "rep-ingest-log",
    to: "rep-persist-results",
    tooltip:
      "Non-policy scheduled-report rows are written to query_results (if the report is configured to persist results) and performance stats land in scheduled_query_stats. Reports configured as 'live only' skip persistence entirely.",
  },
  {
    from: "rep-ingest-log",
    to: "rep-persist-policy-mem",
    dashed: true,
    tooltip:
      "Policy-query results take a different branch: instead of writing to query_results, the log endpoint converts row-count-per-policy into a pass/fail verdict and writes to policy_membership. Same ingestion endpoint, different persistence depending on whether the query is a policy.",
  },
  {
    from: "rep-ingest-write",
    to: "rep-pubsub",
    label: "publish",
    tooltip:
      "The distributed/write endpoint receives live-query rows and immediately publishes them to the campaign's Redis pubsub channel. No persistence. If nobody's subscribed when the publish fires, those rows are gone.",
    techHelp:
      "Live queries use Redis pubsub (not MySQL persistence) because the goal is real-time streaming to the admin's browser, not durable storage. The admin's WebSocket subscribes to campaign_<uuid>; hosts post rows to that channel as they finish; the WebSocket pushes each row to the browser as it arrives. If the admin closes the tab, the campaign still runs on the hosts but the rows evaporate. Exactly the trade-off live queries are designed for.",
  },

  // Persistence → consumers
  {
    from: "rep-persist-results",
    to: "rep-consume-log",
    tooltip:
      "If a log destination is configured (S3, Snowflake, Splunk, Kinesis), scheduled-report rows are forwarded to that destination from query_results as they're persisted. This is Fleet's firehose output for downstream security analytics.",
  },
  {
    from: "rep-persist-results",
    to: "rep-consume-dashboard",
    tooltip:
      "Many dashboard tiles are powered by pre-aggregated rollups computed from scheduled-report results (kernel version counts, OS distribution, hardware inventory). Live GROUP BY across every host would be too slow. The aggregates are the fast path.",
  },
  {
    from: "rep-pubsub",
    to: "rep-consume-ws",
    label: "subscribe",
    tooltip:
      "The admin's browser opens a WebSocket to Fleet; Fleet subscribes that connection to the campaign's Redis pubsub channel. Every row a host posts fans out: Redis → every subscribed Fleet instance → every admin WebSocket for that campaign.",
  },

  // Secondary. Carves (parallel path from live-query execution)
  {
    from: "rep-exec-live",
    to: "rep-secondary-carves",
    dashed: true,
    tooltip:
      "If a live query uses osquery's carve() function (to pull back actual file contents, not just metadata), the resulting file chunks upload to Fleet separately and are stored in carve_metadata + carve_blocks instead of the regular results path.",
    techHelp:
      "carve() is an osquery function that reads a file off the host and uploads it to Fleet in chunks. Normal queries return rows; carved files return bytes. Used for incident response (pull a suspicious binary off a host) or deep forensics. Fleet reassembles the chunks and makes the file available for download through the admin UI.",
  },
];
