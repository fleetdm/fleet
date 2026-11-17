import type { ArchEdge, ArchNode } from "./graph-data";

const COL = 190;
const ROW = 200;

// Row 0: external actors (clients, OS MDM, cloud services)
// Row 1: Fleet process
// Row 2: data stores

export const SYSTEM_NODES: ArchNode[] = [
  // Row 0. Clients
  {
    id: "sys-fleetd",
    label: "fleetd on hosts",
    group: "external",
    blurb:
      "Fleet's agent: orbit + osqueryd + Fleet Desktop. Runs on every managed device and talks HTTPS to Fleet.",
    repoPaths: [{ path: "orbit/" }],
    x: 0,
    y: 0,
  },
  {
    id: "sys-browser",
    label: "Admin UI (browser)",
    group: "external",
    blurb: "React/TypeScript SPA that calls the Fleet JSON API.",
    repoPaths: [{ path: "frontend/" }],
    x: COL,
    y: 0,
  },
  {
    id: "sys-fleetctl",
    label: "fleetctl / GitOps",
    group: "external",
    blurb: "CLI client; GitOps mode reconciles YAML to Fleet state.",
    repoPaths: [{ path: "cmd/fleetctl/" }],
    x: COL * 2,
    y: 0,
  },
  {
    id: "sys-mdm-client",
    label: "OS MDM client",
    group: "external",
    blurb:
      "Apple/Windows/Android built-in MDM clients. They speak the OS vendor's MDM protocol back to Fleet.",
    repoPaths: [],
    x: COL * 3,
    y: 0,
  },

  // Row 0.5. Apple cloud (lifted slightly to one side)
  {
    id: "sys-apns",
    label: "APNs",
    group: "mdm",
    blurb: "Apple Push Notification Service. Wakes Apple MDM clients.",
    repoPaths: [],
    x: COL * 4.3,
    y: 0,
  },
  {
    id: "sys-abm",
    label: "Apple Business Manager / DEP",
    group: "mdm",
    blurb:
      "Apple cloud Fleet syncs with to pre-assign devices and deliver enrollment profiles at first boot.",
    repoPaths: [{ path: "server/mdm/nanodep/" }],
    x: COL * 4.3,
    y: 60,
  },
  {
    id: "sys-vpp",
    label: "Apple VPP / App Store API",
    group: "mdm",
    premium: true,
    blurb: "App Store Volume Purchase. Fleet assigns licenses; Apple delivers apps.",
    repoPaths: [],
    x: COL * 4.3,
    y: 120,
  },

  // Row 0.5. Microsoft cloud
  {
    id: "sys-entra",
    label: "Entra ID (Azure AD)",
    group: "mdm",
    premium: true,
    blurb:
      "Issues JWTs for Windows Autopilot sign-in. Fleet fetches JWKS to verify.",
    repoPaths: [],
    x: COL * 5.5,
    y: 0,
  },
  {
    id: "sys-autopilot",
    label: "Windows Autopilot",
    group: "mdm",
    premium: true,
    blurb:
      "Microsoft cloud that redirects Windows devices to their MDM server at first boot.",
    repoPaths: [{ path: "server/cron/microsoft_autopilot_cron.go" }],
    x: COL * 5.5,
    y: 60,
  },

  // Row 0.5. Google cloud
  {
    id: "sys-amapi",
    label: "Android Management API",
    group: "mdm",
    premium: true,
    blurb:
      "Google cloud used to manage Android Enterprise devices, policies, and managed apps.",
    repoPaths: [{ path: "server/mdm/android/" }],
    x: COL * 6.7,
    y: 0,
  },
  {
    id: "sys-pubsub-google",
    label: "Google Cloud Pub/Sub",
    group: "mdm",
    premium: true,
    blurb:
      "Transport Google uses to notify Fleet of Android device-originated events.",
    repoPaths: [],
    x: COL * 6.7,
    y: 60,
  },

  // Row 0.5. Misc external
  {
    id: "sys-scep-ca",
    label: "External SCEP CA",
    group: "external",
    premium: true,
    blurb:
      "NDES / DigiCert / Smallstep / Hydrant / custom. Fleet proxies cert requests through to the CA.",
    repoPaths: [],
    x: COL * -1.1,
    y: 0,
  },
  {
    id: "sys-idp",
    label: "SSO IdP",
    group: "external",
    premium: true,
    blurb: "Okta / Entra / Google. SAML or OIDC authority.",
    repoPaths: [{ path: "server/sso/" }],
    x: COL * -1.1,
    y: 60,
  },
  {
    id: "sys-tuf",
    label: "TUF updater (updates.fleetdm.com)",
    group: "external",
    blurb: "Signs and distributes fleetd component updates.",
    repoPaths: [],
    x: COL * -1.1,
    y: 120,
  },

  // Row 1. Fleet server
  {
    id: "sys-fleet-server",
    label: "Fleet server",
    group: "service",
    blurb:
      "The Go binary that serves all APIs, runs crons, drains workers, speaks MDM protocols, and reconciles state.",
    repoPaths: [{ path: "cmd/fleet/" }, { path: "server/" }],
    x: COL * 2.8,
    y: ROW,
  },

  // Row 2. Data stores
  {
    id: "sys-mysql",
    label: "MySQL (primary + replicas)",
    group: "store",
    blurb: "Durable source of truth for all Fleet state. MySQL 8+.",
    repoPaths: [{ path: "server/datastore/mysql/" }],
    relatedModes: [{ href: "/data/mysql", label: "Mode 3: MySQL" }],
    x: COL * 1.5,
    y: ROW * 2,
  },
  {
    id: "sys-redis",
    label: "Redis",
    group: "store",
    blurb:
      "Locks (cron coordination), pubsub (live query results), rate-limit counters, short-lived nonces.",
    repoPaths: [{ path: "server/pubsub/" }],
    relatedModes: [{ href: "/data/redis", label: "Mode 3: Redis" }],
    x: COL * 2.8,
    y: ROW * 2,
  },
  {
    id: "sys-filestore",
    label: "File store (S3 or local)",
    group: "store",
    blurb: "Blobs for installers, bootstrap packages, scripts, in-house apps.",
    repoPaths: [],
    relatedModes: [{ href: "/data/filestore", label: "Mode 3: File store" }],
    x: COL * 4.1,
    y: ROW * 2,
  },
];

export const SYSTEM_EDGES: ArchEdge[] = [
  // top → Fleet server
  {
    from: "sys-fleetd",
    to: "sys-fleet-server",
    tooltip:
      "fleetd on every managed device talks to Fleet over HTTPS: osquery TLS for queries/results and the Orbit API for software installs, scripts, and self-service. Device never accepts inbound connections. Fleet never pushes, fleetd always pulls.",
  },
  {
    from: "sys-browser",
    to: "sys-fleet-server",
    tooltip:
      "The admin SPA calls /api/v1/fleet/* with a session token in the Authorization header. All UI reads and writes go through this JSON API; the frontend holds no state the server doesn't also know.",
  },
  {
    from: "sys-fleetctl",
    to: "sys-fleet-server",
    tooltip:
      "fleetctl is the admin CLI. It shares the same Go API client as the server itself (server/service/client.go) and hits the same /api/v1/fleet/* endpoints the browser does. Just scripted instead of interactive.",
  },
  {
    from: "sys-mdm-client",
    to: "sys-fleet-server",
    tooltip:
      "The OS's built-in MDM client (iOS/macOS Apple MDM, Windows SyncML, Android Device Policy) speaks its vendor's protocol back to Fleet's MDM endpoints to pull commands and report state.",
    techHelp:
      "Every modern OS ships with a built-in MDM client. A system-level component Apple/Microsoft/Google wrote and manage. It's not a Fleet install; Fleet just talks the vendor's protocol to it. That's why Fleet can manage iPhones without an app on the device: the MDM client is already there.",
  },
  {
    from: "sys-idp",
    to: "sys-fleet-server",
    dashed: true,
    tooltip:
      "When an admin signs in via SSO, their IdP (Okta, Entra, Google) POSTs a signed SAML assertion (or OIDC JWT) to Fleet's ACS endpoint. Fleet verifies the signature with the IdP's public cert, reads the claims, and mints a Fleet session.",
    techHelp:
      "ACS stands for Assertion Consumer Service. The URL on Fleet's side that accepts SAML assertions. It's the SAML-world equivalent of an OAuth redirect/callback URL. The signing cert exchange happens once at SSO setup time; after that, every login is just: IdP signs, Fleet verifies, session created.",
  },
  {
    from: "sys-scep-ca",
    to: "sys-fleet-server",
    dashed: true,
    tooltip:
      "When a device needs a certificate (Wi-Fi auth, 802.1X, VPN), Fleet acts as a SCEP proxy: the device sends its SCEP request to Fleet, Fleet forwards to the configured external CA (NDES, DigiCert, Smallstep, Hydrant, custom), Fleet relays the signed cert back.",
    techHelp:
      "SCEP (Simple Certificate Enrollment Protocol) is how MDM-managed devices get certificates issued at scale without a human in the loop. The device generates a key pair locally, posts a certificate signing request to the CA, and the CA returns a signed cert. Fleet sits in the middle so the admin doesn't have to expose the CA directly to every device.",
  },
  {
    from: "sys-tuf",
    to: "sys-fleetd",
    dashed: true,
    tooltip:
      "fleetd periodically fetches signed metadata from updates.fleetdm.com (the TUF repo) to check if its own components (orbit, osqueryd, fleet-desktop) have new versions. If so, it downloads and installs them.",
    techHelp:
      "TUF (The Update Framework) is a signing + distribution standard designed for secure software updates. It uses role-separated signing keys so a single compromised key can't push a malicious update. Fleet uses TUF so fleetd can self-update across hundreds of thousands of devices without each one trusting a single update server.",
  },

  // Fleet server → Apple cloud
  {
    from: "sys-fleet-server",
    to: "sys-apns",
    tooltip:
      "Fleet pushes to APNs whenever an Apple device has pending MDM commands waiting. APNs then wakes the device, which calls back to Fleet's MDM endpoint to pull the commands.",
  },
  {
    from: "sys-fleet-server",
    to: "sys-abm",
    tooltip:
      "Fleet syncs with Apple Business Manager on a cron: pulls the list of device-serial assignments to Fleet's MDM, and pushes updates to the enrollment profile ABM should serve new devices at first boot.",
  },
  {
    from: "sys-fleet-server",
    to: "sys-vpp",
    tooltip:
      "Fleet talks to Apple's VPP API to sync the admin's purchased app catalog and to assign/revoke individual licenses at install/uninstall time. Fleet doesn't deliver the app. Apple pushes it from the App Store CDN after the license assignment.",
  },
  {
    from: "sys-apns",
    to: "sys-mdm-client",
    dashed: true,
    tooltip:
      "Apple delivers the push to the device. Push payload is empty. It's a doorbell, not a message. The device then calls into Fleet to find out what's waiting.",
  },

  // Fleet server ↔ Microsoft cloud
  {
    from: "sys-mdm-client",
    to: "sys-autopilot",
    dashed: true,
    tooltip:
      "At first boot after an Entra sign-in, the Windows OOBE queries Autopilot to find out which MDM server to enroll with. Autopilot answers \"Fleet\" (if the device is in Fleet's Autopilot assignment), the device then connects to Fleet's SyncML endpoint.",
  },
  {
    from: "sys-fleet-server",
    to: "sys-entra",
    dashed: true,
    tooltip:
      "Fleet fetches Entra's JWKS (public key set) on a cadence to verify the JWTs Entra-authenticated Autopilot devices present during enrollment. The JWT proves the user signed in to Entra; Fleet uses that to tie the device to the right fleet.",
    techHelp:
      "JWKS (JSON Web Key Set) is just a well-known URL that returns the current public keys an identity provider is using to sign JWTs. Fleet fetches it periodically rather than hardcoding keys so Entra can rotate its signing keys without breaking Fleet. Every OIDC/JWT-using service does this.",
  },

  // Fleet server ↔ Google cloud
  {
    from: "sys-fleet-server",
    to: "sys-amapi",
    tooltip:
      "Fleet calls Google's Android Management API to create enterprises, push policies, assign managed Play Store apps, and send device commands (lock, wipe, reboot). Fleet never calls the device directly. AMAPI is the entire control plane.",
    techHelp:
      "AMAPI (Android Management API) is Google's HTTPS control plane for managing enterprise Android. Unlike Apple MDM (direct device-to-Fleet) or Windows MDM (device POSTs SyncML to Fleet), Android MDM is one layer removed: Fleet calls Google, Google calls the device, device state changes come back via Pub/Sub notifications. This is Google's design, not a Fleet choice.",
  },
  {
    from: "sys-pubsub-google",
    to: "sys-fleet-server",
    tooltip:
      "Google posts enrollment events, status reports, and command results to a Pub/Sub topic Fleet subscribes to. This is how Fleet learns anything about Android devices. The devices never talk to Fleet directly.",
    techHelp:
      "Google Cloud Pub/Sub is a managed asynchronous message queue. Fleet creates a subscription at setup time; Google writes device events (ENROLLMENT, STATUS_REPORT, COMMAND) to the topic; Fleet polls (or receives via push) the subscription. This is the one-way inbound channel for every Android MDM observation.",
  },

  // Fleet server → data stores
  {
    from: "sys-fleet-server",
    to: "sys-mysql",
    label: "read/write",
    tooltip:
      "MySQL is Fleet's durable source of truth: hosts, users, policies, software, MDM profiles, scheduled queries, activities, everything. Fleet uses DBReader for reads (replica) and DBWriter for writes (primary) through sqlx with named parameters.",
    techHelp:
      "The DBReader/DBWriter split lets Fleet route reads to read replicas and keep the primary free for writes. Not every read is safe to send to a replica (replication lag means recent writes may not be visible yet), so Fleet also has ctxdb.RequirePrimary(ctx, true) for read-after-write cases. See .claude/rules/fleet-database.md for the rules.",
  },
  {
    from: "sys-fleet-server",
    to: "sys-redis",
    label: "read/write",
    tooltip:
      "Redis handles the state Fleet needs faster or more ephemeral than MySQL: cron locks (so only one Fleet instance runs a given cron per tick), pubsub channels for live query results, rate-limit counters, short-lived nonces.",
  },
  {
    from: "sys-fleet-server",
    to: "sys-filestore",
    label: "blobs",
    tooltip:
      "Non-structured blobs. Software installers, bootstrap packages, config profile uploads, in-house IPAs, carved files. Live in S3 in production (or local disk in dev). MySQL stores metadata and references; the file store holds the bytes.",
  },
];
