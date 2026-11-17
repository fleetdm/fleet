export type NodeGroup =
  | "external"
  | "ingress"
  | "service"
  | "datastore"
  | "store"
  | "background"
  | "mdm"
  | "vuln";

export type ArchNode = {
  id: string;
  label: string;
  group: NodeGroup;
  blurb: string;
  repoPaths: { path: string; note?: string }[];
  analogue?: { go: string; fe: string };
  relatedModes?: { href: string; label: string }[];
  // Marks nodes whose underlying feature is premium-only.
  // The node often exists in Free codepaths too, but the capability it
  // represents (VPP delivery, policy automation action, DDM, SSO, ...) is
  // gated behind license.IsPremium().
  premium?: boolean;
  x: number;
  y: number;
};

export type ArchEdge = {
  from: string;
  to: string;
  label?: string;
  dashed?: boolean;
  tooltip?: string;
  // Shown as a secondary "Technical help" block in the hover tooltip when
  // Technical help mode is on. Use for edges whose main tooltip assumes
  // familiarity with a protocol or acronym a FE/novice reader might not have.
  techHelp?: string;
};

export const GROUP_META: Record<
  NodeGroup,
  { label: string; card: string; chip: string }
> = {
  external: {
    label: "External actors",
    card: "bg-zinc-100 border-zinc-300 dark:bg-zinc-800 dark:border-zinc-600",
    chip: "bg-zinc-200 border-zinc-400 dark:bg-zinc-700 dark:border-zinc-500",
  },
  ingress: {
    label: "Ingress",
    card: "bg-sky-50 border-sky-300 dark:bg-sky-950 dark:border-sky-700",
    chip: "bg-sky-100 border-sky-400 dark:bg-sky-900 dark:border-sky-600",
  },
  service: {
    label: "Service layer",
    card: "bg-indigo-50 border-indigo-300 dark:bg-indigo-950 dark:border-indigo-700",
    chip: "bg-indigo-100 border-indigo-400 dark:bg-indigo-900 dark:border-indigo-600",
  },
  datastore: {
    label: "Datastore layer",
    card: "bg-violet-50 border-violet-300 dark:bg-violet-950 dark:border-violet-700",
    chip: "bg-violet-100 border-violet-400 dark:bg-violet-900 dark:border-violet-600",
  },
  store: {
    label: "Data stores",
    card: "bg-emerald-50 border-emerald-300 dark:bg-emerald-950 dark:border-emerald-700",
    chip: "bg-emerald-100 border-emerald-400 dark:bg-emerald-900 dark:border-emerald-600",
  },
  background: {
    label: "Background work",
    card: "bg-amber-50 border-amber-300 dark:bg-amber-950 dark:border-amber-700",
    chip: "bg-amber-100 border-amber-400 dark:bg-amber-900 dark:border-amber-600",
  },
  mdm: {
    label: "MDM subsystems",
    card: "bg-rose-50 border-rose-300 dark:bg-rose-950 dark:border-rose-700",
    chip: "bg-rose-100 border-rose-400 dark:bg-rose-900 dark:border-rose-600",
  },
  vuln: {
    label: "Vulnerability pipeline",
    card: "bg-fuchsia-50 border-fuchsia-300 dark:bg-fuchsia-950 dark:border-fuchsia-700",
    chip: "bg-fuchsia-100 border-fuchsia-400 dark:bg-fuchsia-900 dark:border-fuchsia-600",
  },
};

const COL = 260;
const ROW = 110;

export const NODES: ArchNode[] = [
  // Row 0: external actors
  {
    id: "osquery",
    label: "osquery agents (fleetd)",
    group: "external",
    blurb:
      "Agents installed on managed hosts. They poll Fleet for config and queries, submit results, and distribute live-query answers.",
    repoPaths: [
      { path: "orbit/", note: "Fleet's osquery runtime" },
      { path: "server/service/osquery.go", note: "server-side TLS endpoints" },
    ],
    analogue: {
      go: "Agents send HTTP(S) requests to Fleet TLS endpoints.",
      fe: "Think of this like a mobile app polling your API, except the API is also sending it work to do.",
    },
    relatedModes: [{ href: "/subsystems/osquery-tls", label: "Mode 5: osquery TLS" }],
    x: 0,
    y: 0,
  },
  {
    id: "mdm-push",
    label: "MDM push (APNs / FCM / WNS)",
    group: "external",
    blurb:
      "Fleet pushes device management commands via Apple APNs, Google FCM, and Microsoft WNS. These wake up a device, which then pulls commands from Fleet.",
    repoPaths: [
      { path: "server/mdm/apple/" },
      { path: "server/mdm/microsoft/" },
      { path: "server/mdm/android/" },
    ],
    x: COL,
    y: 0,
  },
  {
    id: "ui",
    label: "Admin UI (browser)",
    group: "external",
    blurb:
      "React/TypeScript Fleet UI. Hits the `/api/v1/fleet/...` JSON API with a bearer token.",
    repoPaths: [
      { path: "frontend/" },
      { path: "frontend/services/", note: "API client" },
    ],
    analogue: {
      go: "The UI is just a client. Nothing you learn in Atlas is about the UI.",
      fe: "You already know this layer: it's TanStack Query against a REST API.",
    },
    x: COL * 2,
    y: 0,
  },
  {
    id: "fleetctl",
    label: "fleetctl / GitOps",
    group: "external",
    blurb:
      "CLI for admin operations and declarative config. GitOps applies YAML through the same API as the UI.",
    repoPaths: [
      { path: "cmd/fleetctl/" },
      { path: "server/service/client.go", note: "shared Go API client" },
    ],
    x: COL * 3,
    y: 0,
  },
  {
    id: "sso",
    label: "SSO / IdP",
    group: "external",
    blurb: "SAML / OIDC identity providers (Okta, Entra, Google, etc.) for admin and end-user auth.",
    repoPaths: [{ path: "server/sso/" }],
    x: COL * 4,
    y: 0,
  },

  // Row 1: ingress
  {
    id: "http-router",
    label: "HTTP router",
    group: "ingress",
    blurb:
      "All HTTP routes. One function wires every endpoint to its service method via the kit/transport pattern.",
    repoPaths: [
      { path: "server/service/handler.go", note: "MakeHandler" },
      { path: "server/service/endpoint_utils.go" },
    ],
    analogue: {
      go: "Each endpoint is a (request → service call → response) triple registered on a mux.",
      fe: "Like an Express router but each route explicitly declares its request/response types.",
    },
    relatedModes: [{ href: "/tracer", label: "Mode 2: trace a request" }],
    x: COL * 0.5,
    y: ROW * 1.5,
  },
  {
    id: "device-router",
    label: "Device endpoints",
    group: "ingress",
    blurb: "`/device/...` routes for the end-user My Device page. Device-token auth.",
    repoPaths: [{ path: "server/service/devices.go" }],
    x: COL * 1.5,
    y: ROW * 1.5,
  },
  {
    id: "mdm-router",
    label: "MDM endpoints",
    group: "ingress",
    blurb: "`/mdm/...` protocol endpoints (SCEP, DEP, checkin, command, SyncML).",
    repoPaths: [
      { path: "server/mdm/apple/" },
      { path: "server/mdm/microsoft/" },
    ],
    x: COL * 2.5,
    y: ROW * 1.5,
  },
  {
    id: "live-query-ws",
    label: "Live query WebSocket",
    group: "ingress",
    blurb: "Streams live-query results to the UI.",
    repoPaths: [{ path: "server/service/async/" }, { path: "server/pubsub/" }],
    x: COL * 3.5,
    y: ROW * 1.5,
  },

  // Row 2: services
  {
    id: "service-core",
    label: "Core service",
    group: "service",
    blurb:
      "Fleet.Service interface + its implementation. Business logic, auth checks, orchestration between datastore + subsystems.",
    repoPaths: [
      { path: "server/fleet/service.go", note: "interface" },
      { path: "server/service/", note: "impl" },
    ],
    analogue: {
      go: "The service type satisfies the fleet.Service interface. Endpoints call it, it calls the datastore.",
      fe: "Like your service/ folder on the FE: pure business logic between API client and UI.",
    },
    relatedModes: [{ href: "/tracer", label: "Mode 2: trace a request" }],
    x: COL * 1,
    y: ROW * 3.2,
  },
  {
    id: "service-ee",
    label: "Enterprise wrapper",
    group: "service",
    blurb:
      "`ee/server/service` embeds the core service and intercepts methods with license-tier checks before delegating.",
    repoPaths: [{ path: "ee/server/service/" }],
    analogue: {
      go: "Embedded struct = inheritance by composition. The ee type overrides specific methods and calls super.",
      fe: "Like a higher-order component that wraps and conditionally gates its child.",
    },
    x: COL * 2.3,
    y: ROW * 3.2,
  },
  {
    id: "authz",
    label: "Authorization (OPA)",
    group: "service",
    blurb: "Rego policies evaluated per request to allow/deny.",
    repoPaths: [{ path: "server/authz/" }],
    x: COL * 3.5,
    y: ROW * 3.2,
  },

  // Row 3: datastore
  {
    id: "datastore-iface",
    label: "fleet.Datastore interface",
    group: "datastore",
    blurb:
      "The giant interface the service layer talks to. Keeps business logic unaware of SQL.",
    repoPaths: [{ path: "server/fleet/datastore.go" }],
    analogue: {
      go: "An interface so services are decoupled from storage. Mocked in tests, implemented by MySQL in prod.",
      fe: "Like injecting a repository as a prop or React context so you can swap it in tests.",
    },
    x: COL * 1.5,
    y: ROW * 4.8,
  },
  {
    id: "datastore-mysql",
    label: "MySQL implementation",
    group: "datastore",
    blurb:
      "Every Datastore method implemented against MySQL. Also owns schema migrations.",
    repoPaths: [
      { path: "server/datastore/mysql/" },
      { path: "server/datastore/mysql/migrations/", note: "goose migrations" },
    ],
    relatedModes: [{ href: "/data/mysql", label: "Mode 3: MySQL deep-dive" }],
    x: COL * 2.8,
    y: ROW * 4.8,
  },

  // Row 4: stores
  {
    id: "mysql",
    label: "MySQL (primary + replicas)",
    group: "store",
    blurb:
      "Fleet's source of truth. All durable state: hosts, policies, software, MDM commands, activity feed, more.",
    repoPaths: [
      { path: "server/datastore/mysql/schema.sql", note: "generated schema" },
    ],
    relatedModes: [{ href: "/data/mysql", label: "Mode 3: MySQL" }],
    x: COL * 1.2,
    y: ROW * 6.4,
  },
  {
    id: "redis",
    label: "Redis",
    group: "store",
    blurb:
      "Locks (so only one Fleet instance runs each cron), live-query result bus (pubsub), rate limits, caches.",
    repoPaths: [
      { path: "server/pubsub/" },
      { path: "server/service/async/" },
    ],
    relatedModes: [{ href: "/data/redis", label: "Mode 3: Redis" }],
    x: COL * 2.5,
    y: ROW * 6.4,
  },
  {
    id: "filestore",
    label: "File store (S3 / local)",
    group: "store",
    blurb:
      "Blob storage: software installers, bootstrap packages, config profiles, scripts.",
    repoPaths: [{ path: "server/service/software_installers.go" }],
    x: COL * 3.8,
    y: ROW * 6.4,
  },

  // Side column (right): background
  {
    id: "cron",
    label: "Cron scheduler",
    group: "background",
    blurb:
      "Named recurring jobs. Each job acquires a Redis lock before running, so only one Fleet instance runs it at a time.",
    repoPaths: [
      { path: "server/service/schedule/schedule.go" },
      { path: "server/cron/" },
    ],
    analogue: {
      go: "Goroutine ticker + distributed lock in Redis. On start, every instance tries to grab the lock; one wins.",
      fe: "Imagine every browser tab setInterval'ing the same job and racing for a shared lock.",
    },
    relatedModes: [{ href: "/background", label: "Mode 4: Cron" }],
    x: COL * 5,
    y: ROW * 3.2,
  },
  {
    id: "worker",
    label: "Async worker queue",
    group: "background",
    blurb:
      "Durable job queue in MySQL. The service enqueues (e.g. 'install this software, send this Jira ticket') and workers drain.",
    repoPaths: [{ path: "server/worker/" }],
    relatedModes: [{ href: "/background", label: "Mode 4: Workers" }],
    x: COL * 5,
    y: ROW * 4.8,
  },

  // MDM column
  {
    id: "mdm-apple",
    label: "Apple MDM",
    group: "mdm",
    blurb:
      "nanomdm + nanodep, SCEP, DEP, Apple Business Manager, DDM, APNs push.",
    repoPaths: [{ path: "server/mdm/apple/" }, { path: "server/mdm/nanomdm/" }],
    relatedModes: [{ href: "/subsystems/apple-mdm", label: "Mode 5: Apple MDM" }],
    x: COL * 5,
    y: ROW * 6.4,
  },
  {
    id: "mdm-microsoft",
    label: "Microsoft MDM",
    group: "mdm",
    blurb: "Windows MDM via SyncML, Autopilot enrollment.",
    repoPaths: [{ path: "server/mdm/microsoft/" }],
    x: COL * 5,
    y: ROW * 7.3,
  },
  {
    id: "mdm-android",
    label: "Android MDM",
    group: "mdm",
    blurb: "Android Management API (AMAPI) integration.",
    repoPaths: [{ path: "server/mdm/android/" }],
    x: COL * 5,
    y: ROW * 8.2,
  },

  // Vuln pipeline
  {
    id: "vuln",
    label: "Vulnerability pipeline",
    group: "vuln",
    blurb:
      "Downloads NVD / OSV / OVAL / MSRC / Office feeds, matches installed software, writes vulnerabilities to MySQL.",
    repoPaths: [
      { path: "server/vulnerabilities/" },
      { path: "cmd/cpe/" },
      { path: "cmd/cve/" },
    ],
    x: 0,
    y: ROW * 4.8,
  },
];

export const EDGES: ArchEdge[] = [
  {
    from: "osquery",
    to: "http-router",
    tooltip:
      "fleetd posts to /api/osquery/* over HTTPS: enroll, pull config, read distributed queries, submit results.",
  },
  {
    from: "ui",
    to: "http-router",
    tooltip:
      "Browser calls /api/v1/fleet/* with a bearer session token in the Authorization header.",
  },
  {
    from: "fleetctl",
    to: "http-router",
    tooltip:
      "fleetctl shares the Go API client with the server (server/service/client.go) and calls the same /api/v1/fleet/* endpoints as the UI.",
  },
  {
    from: "sso",
    to: "http-router",
    dashed: true,
    label: "SAML/OIDC",
    tooltip:
      "IdP posts a signed SAML assertion to Fleet's ACS endpoint. Fleet verifies, extracts claims, mints a session.",
    techHelp:
      "SAML is an XML-based auth protocol from the pre-OIDC era. The IdP (Okta, Entra, Google) signs a small XML document asserting \"this user is authenticated and here are their attributes\" and POSTs it to Fleet's ACS (Assertion Consumer Service) URL. Fleet validates the signature with the IdP's public cert, reads the claims (email, groups, name), and creates a session. OIDC is the newer JSON/JWT-based cousin; Fleet speaks both. The ACS URL is basically an OAuth callback for the SAML world.",
  },
  {
    from: "mdm-push",
    to: "mdm-router",
    dashed: true,
    label: "wake",
    tooltip:
      "Fleet sends a push via APNs/FCM/WNS. The push has no payload; it just wakes the device so it will call back into the MDM endpoints.",
    techHelp:
      "APNs (Apple Push Notification service), FCM (Firebase Cloud Messaging, Google), and WNS (Windows Notification Service, Microsoft) are the three OS-vendor push gateways. Think of them as the device-world equivalent of a WebSocket server you don't own: you register a device, get back a token, and when you want to reach that device you POST a notification to the gateway using the token. For MDM, Fleet uses pushes as a doorbell, not a message. The device sees the push and knows to call back into Fleet's HTTP endpoints to pull whatever commands are waiting. Keeps commands auditable and the push payload minimal.",
  },
  {
    from: "osquery",
    to: "device-router",
    tooltip:
      "End-user My Device page hits /device/<token>/* with a per-host device token (not an admin session).",
  },

  {
    from: "http-router",
    to: "service-core",
    tooltip:
      "Router decodes the HTTP request into a typed struct and calls the matching fleet.Service method.",
  },
  {
    from: "device-router",
    to: "service-core",
    tooltip:
      "Device endpoints hit the same service layer, but with narrower authz policies.",
  },
  {
    from: "mdm-router",
    to: "mdm-apple",
    tooltip: "Apple MDM protocol endpoints route to the Apple MDM subsystem.",
  },
  {
    from: "mdm-router",
    to: "mdm-microsoft",
    tooltip: "Windows SyncML endpoints route to the Microsoft MDM subsystem.",
    techHelp:
      "SyncML is an XML-based protocol OMA originally designed in the early 2000s for syncing phone contacts, which Microsoft repurposed as the wire format for Windows MDM. Every command the server wants to run (install a profile, run a wmic query, lock a device) is encoded as an XML `<SyncBody>` with `<Add>`, `<Replace>`, `<Delete>`, `<Exec>` operations. The Windows device POSTs its own SyncML envelope back with status codes for each operation. It feels verbose compared to Apple's plist commands or Android's JSON, but it IS the Windows MDM wire format. Not a Fleet choice.",
  },
  {
    from: "mdm-router",
    to: "mdm-android",
    tooltip: "AMAPI callbacks route to the Android MDM subsystem.",
    techHelp:
      "AMAPI is Google's Android Management API. The HTTPS control plane for managing enterprise Android devices. Unlike Apple MDM (where Fleet talks directly to devices through APNs) or Windows MDM (where devices POST SyncML to Fleet), Android MDM is one layer removed: Fleet talks to Google's API, Google talks to the device. When something changes on the device, Google POSTs an ENROLLMENT or STATUS_REPORT message to a Pub/Sub topic Fleet subscribes to. So the \"MDM channel\" for Android is really Google + Pub/Sub, not a direct device connection.",
  },
  {
    from: "live-query-ws",
    to: "redis",
    label: "pubsub",
    tooltip:
      "Live query WebSocket subscribes to a Redis pubsub channel keyed by the campaign id; results fan out as hosts respond.",
    techHelp:
      "Redis pubsub is the simplest possible fan-out: publishers call `PUBLISH channel message`, subscribers call `SUBSCRIBE channel` and receive each publish. No persistence, no acknowledgments, no consumer groups. If you're not subscribed when a message publishes, you miss it. Fleet uses it for live queries because the pattern matches exactly: one admin starts a live query (one channel), N hosts respond with rows (N publishes), one browser WebSocket subscribes to that channel until the campaign ends. Fleet also uses Redis for cron locks and rate counters, but those use regular keys, not pubsub.",
  },

  {
    from: "service-core",
    to: "service-ee",
    dashed: true,
    label: "ee wraps core",
    tooltip:
      "The ee service embeds the core service struct and overrides specific methods to add license checks before delegating.",
  },
  {
    from: "service-core",
    to: "authz",
    tooltip:
      "Every protected service method calls authz.Authorize(ctx, target, action) against OPA/Rego policy before doing work.",
    techHelp:
      "OPA (Open Policy Agent) is a general-purpose policy engine. You hand it a decision input (who's acting, what they're acting on, what action) and it runs your rules against it to return allow/deny. Rego is OPA's DSL. Datalog-flavored, declarative, think \"SQL for permission rules.\" Fleet keeps its auth rules in `server/authz/policy.rego`, compiled into the binary. The upside vs. hand-rolled role checks: all authz logic lives in one file, is testable in isolation, and can be audited without reading Go code paths.",
  },
  {
    from: "service-core",
    to: "datastore-iface",
    tooltip:
      "Service methods call fleet.Datastore methods. Business logic never sees SQL directly.",
  },
  {
    from: "service-ee",
    to: "datastore-iface",
    tooltip:
      "ee service calls the same fleet.Datastore interface. One datastore for the whole app.",
  },

  {
    from: "datastore-iface",
    to: "datastore-mysql",
    tooltip:
      "The MySQL impl in server/datastore/mysql/ satisfies the fleet.Datastore interface.",
  },
  {
    from: "datastore-mysql",
    to: "mysql",
    tooltip:
      "Datastore methods use DBReader for reads (replica) and DBWriter for writes (primary). sqlx + named parameters.",
  },
  {
    from: "service-core",
    to: "redis",
    label: "locks / pubsub",
    tooltip:
      "Service writes to Redis directly for cron locks, live-query result publishes, and rate limit counters.",
  },
  {
    from: "service-core",
    to: "filestore",
    tooltip:
      "Software installers, bootstrap packages, config profiles, and scripts live in S3 (prod) or local disk (dev). MySQL stores metadata; the blob lives here.",
  },

  {
    from: "cron",
    to: "service-core",
    tooltip:
      "Cron jobs call service methods. The service does the actual work; cron just schedules it.",
  },
  {
    from: "cron",
    to: "redis",
    label: "lock",
    tooltip:
      "Before running, each cron tries to acquire a named Redis lock. Only one Fleet instance per interval wins.",
  },
  {
    from: "worker",
    to: "mysql",
    label: "job table",
    tooltip:
      "Workers poll the jobs table in MySQL. Claim a row by updating state to running; delete on success, requeue with backoff on failure.",
  },
  {
    from: "service-core",
    to: "worker",
    label: "enqueue",
    tooltip:
      "Service enqueues a job (insert into the jobs table). A worker picks it up and does the external I/O (Jira, Zendesk, software install).",
  },

  {
    from: "mdm-apple",
    to: "service-core",
    tooltip:
      "Apple MDM subsystem calls the service to persist enrollments, record commands, update host MDM state.",
  },
  {
    from: "mdm-microsoft",
    to: "service-core",
    tooltip:
      "Microsoft MDM subsystem calls the service to persist enrollments and commands.",
  },
  {
    from: "mdm-android",
    to: "service-core",
    tooltip:
      "Android MDM subsystem calls the service to persist enrollments and policy state.",
  },

  {
    from: "vuln",
    to: "datastore-mysql",
    tooltip:
      "Vuln pipeline writes matched vulnerabilities to host_software_vulnerabilities and related tables.",
  },
];

import { SYSTEM_NODES } from "./system-graph";
import { SOFTWARE_NODES } from "./software-graph";
import { MDM_NODES } from "./mdm-graph";
import { POLICIES_NODES } from "./policies-graph";
import { REPORTS_NODES } from "./reports-graph";
import { GITOPS_NODES } from "./gitops-graph";

export function getNode(id: string): ArchNode | undefined {
  return (
    NODES.find((n) => n.id === id) ??
    SYSTEM_NODES.find((n) => n.id === id) ??
    SOFTWARE_NODES.find((n) => n.id === id) ??
    MDM_NODES.find((n) => n.id === id) ??
    POLICIES_NODES.find((n) => n.id === id) ??
    REPORTS_NODES.find((n) => n.id === id) ??
    GITOPS_NODES.find((n) => n.id === id)
  );
}

export function nodeLabel(id: string): string {
  return getNode(id)?.label ?? id;
}
