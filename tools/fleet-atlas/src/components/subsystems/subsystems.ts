export type SubsystemCategory =
  | "Customer-facing surfaces"
  | "Agent & data collection"
  | "Identity & access"
  | "Device management"
  | "Software installs"
  | "Vulnerability management"
  | "Observability & integrations"
  | "Development process";

export const SUBSYSTEM_CATEGORIES: SubsystemCategory[] = [
  "Customer-facing surfaces",
  "Agent & data collection",
  "Identity & access",
  "Device management",
  "Software installs",
  "Vulnerability management",
  "Observability & integrations",
  "Development process",
];

export type Subsystem = {
  slug: string;
  label: string;
  category: SubsystemCategory;
  one_liner: string;
  body: string; // markdown
  repoPaths: string[];
  featured?: boolean; // umbrella pages flagged as featured in the category
};

export const SUBSYSTEMS: Subsystem[] = [
  // ─── Customer-facing surfaces ───
  {
    slug: "admin-ui-anatomy",
    label: "Admin UI anatomy",
    category: "Customer-facing surfaces",
    featured: true,
    one_liner:
      "The admin UI tour. What a Fleet operator sees when they log in, screen by screen.",
    repoPaths: ["frontend/pages/"],
    body: `
### What this subsystem is

The React admin UI is where every Fleet customer spends most of their time. It's a sibling of the API. Anything an admin does here goes through the same endpoints customers can call via fleetctl.

If you're in sales, CS, or support and you've never clicked around a running Fleet, this page is the map of what the admin sees.

### The top nav

Fleet's admin UI has these primary tabs:

- **Dashboard**: aggregate rollups. Host counts by OS, policy pass rate, vuln counts. Powered by scheduled-query results.
- **Hosts**: the host inventory. The list every admin lives in. Click a row for the per-host drill-in.
- **Controls** (sometimes "Policies & profiles"): config profiles, OS updates, scripts, setup experience.
- **Software**: catalog of titles added to the fleet (custom installers, FMAs, VPP, in-house). Admins click Install here.
- **Reports**: saved and ad-hoc osquery. Each report wraps one SQL \`query\`.
- **Policies**: pass/fail checks + automations.
- **Settings**: integrations, SSO, SCIM, fleets, agent options, API tokens.

Non-engineers trip on the terminology mismatch: the URL path says \`/queries\` but the UI label says "Reports". Code still has \`fleet.Query\`. Explained on the terminology page in Backend conventions.

### The Hosts page

The single most-used page. Features worth knowing:

- **Filters** by platform, team/fleet, label, policy, vuln, software, MDM status. Each filter maps to a SQL WHERE on \`hosts\`.
- **Columns** admins frequently customize. Showing OS version, disk encryption state, last-seen, teams, etc.
- **Bulk actions**: transfer to fleet, run script, lock/wipe. These enqueue upcoming_activities per selected host.
- **Per-host drill-in**: 10+ tabs including Details, Software, Reports, Policies, Scripts, Users (device-local user accounts), Activity history.

If a customer says "I don't see my host". They're almost always on this page.

### The Software page

The second-busiest customer surface. Admin picks a title, Fleet routes to the right install path. Tabs:

- **Available**: titles that have been added to the fleet (what admins can click Install for).
- **Software library / Add software**: the pickers for Custom installer upload, FMA (macOS / Windows), VPP (macOS + mobile), In-house apps, Managed Google Play, Scripts.

Per-title drill-in shows version, platform, scripts, pre-install query, and the list of hosts that have (or don't have) it installed.

### Settings → Teams / Fleets

Where multi-tenancy happens. Each fleet has its own agent options, enroll secrets, admin memberships, software library, policies. Free deployments only have one global "Unassigned" group.

### What customers never see

- MySQL. SQL knowledge is not required for admins.
- Redis.
- The command queue (\`nano_commands\`). Admins see activity items, not raw MDM commands.
- Fleet's own backend logs.

### Edge cases

- **GitOps mode locks the UI read-only** for a given fleet. Admins can view everything, but every Edit button is disabled. Trips confused customers constantly.
- **Primo mode hides the multi-fleet UI** and treats every host as "Unassigned". Customers on this mode don't see teams concepts at all.
`,
  },
  {
    slug: "fleet-desktop",
    label: "Fleet Desktop",
    category: "Customer-facing surfaces",
    one_liner:
      "The end-user menubar app that sits on every managed Mac. States, what the user sees, what admins can control.",
    repoPaths: ["orbit/cmd/desktop/", "frontend/pages/MyDevicePage/"],
    body: `
### What this subsystem is

Fleet Desktop is the small menubar / system-tray app that ships inside fleetd. The end user clicks the Fleet logo in their menu bar and sees a short menu. It's the only Fleet surface most *end users* ever interact with.

Not to be confused with the admin UI (that's for operators, over the web).

### What's in the menu

A typical click reveals:

- **"Open My Device"**: launches the user's browser to the per-device My Device page.
- **"Report a bug"** or similar hooks (configurable by the admin).
- **Status indicators**: whether the host is healthy, failing policies, missing updates.

### How it starts

- fleetd (the agent) ships with the desktop binary bundled.
- At first login, Orbit launches the desktop binary into the user's session.
- Desktop reads a per-host device token from fleetd's local state, which authenticates its calls to the \`/device/*\` endpoints.

### Fleet Desktop states

- **Healthy**: green dot, all policies passing.
- **Policy failing**: yellow or red indicator; the specific policies may show in the menu.
- **Software update required**: nudge the user to update macOS / Windows / their browser.
- **Not enrolled**: the host hasn't yet gotten a node key or device token.

### What admins can configure

- Transparency URL (where the "What does Fleet collect?" link goes).
- End-user SSO (if premium).
- Whether end-users can self-install software (the self-service catalog).

### Edge cases

- **Fleet Desktop silently fails to launch** when the user isn't logged in. It's a GUI, not a daemon.
- **Device token rotation** happens on re-enroll; the desktop restarts and picks up the new token.
- On **shared Macs**, each user gets their own Fleet Desktop instance.
`,
  },
  {
    slug: "my-device-page",
    label: "My Device page",
    category: "Customer-facing surfaces",
    one_liner:
      "The end-user web page that shows a user what Fleet knows about THEIR specific device.",
    repoPaths: [
      "frontend/pages/MyDevicePage/",
      "server/service/devices.go",
    ],
    body: `
### What this subsystem is

The web page an end user lands on when they click "Open My Device" in Fleet Desktop. It's a self-service surface. The user can see their device's health, failing policies, available software, and (if self-service is enabled) click Install on things.

### How auth works

Different from the admin UI's auth:

- The URL is \`/device/{token}/...\`. The token is a per-host, URL-safe secret Fleet generated at enrollment and stored on the device.
- Fleet Desktop knows the token and bakes it into the URL when it opens the browser.
- No admin login required. The token *is* the auth.
- Device tokens can be rotated; a stale token returns 401.

### What the user sees

- **Device info**: hostname, OS, serial, status.
- **Policies**: pass/fail per policy. For failing policies, the admin-configured resolution text explains what the user should do.
- **Software**: a list of installed software + a self-service catalog of admin-pre-approved titles the user can click Install on.
- **Activity**: install history, policy changes, enrollment events.

### Self-service installs

If an admin added a title with \`self_service=true\`, it shows up here. User clicks Install:

1. UI calls \`POST /device/{token}/software/install/{title_id}\`.
2. Service writes a host_software_installs row (or vpp / in-house sibling).
3. Delivery follows the normal install path (osquery poll for custom, APNs for VPP, etc.).
4. User sees status update when fleetd / MDM reports the result back.

### Edge cases

- **The token lives on the device, not the user**. If the device is shared, every user who opens Fleet Desktop lands on the same My Device page.
- **BYOD / user-enrollment** devices show less info because the admin's visibility is scoped to the work container.
- **Primo mode** customers get a simplified My Device page.
- **If the user is on end-user SSO**, the My Device page may require SSO sign-in on top of the device token (premium).
`,
  },

  // ─── Agent & data collection ───
  {
    slug: "osquery-fundamentals",
    label: "osquery fundamentals",
    category: "Agent & data collection",
    featured: true,
    one_liner:
      "What osquery actually is, how it works, and the mental model you need before anything else in Fleet makes sense.",
    repoPaths: ["orbit/", "server/service/osquery.go"],
    body: `
### What this subsystem is

osquery is an open-source agent (originally from Facebook, now Linux Foundation) that lets you query a device's state using SQL. It's the data-collection foundation Fleet is built on. Everything Fleet knows about a device. Processes, installed software, battery health, FileVault status, startup items, open ports. Comes from osquery queries.

If you understand osquery, Fleet makes sense. If you don't, Fleet looks like magic.

### The mental model

A device is a database. The tables are OS state:

- \`processes\`: what's running right now
- \`system_info\`: hostname, uuid, hardware
- \`os_version\`: what OS and version
- \`users\`: local user accounts
- \`listening_ports\`: open sockets
- \`apps\` (macOS) or \`programs\` (Windows). Installed software
- hundreds more

\`SELECT name, pid FROM processes WHERE name = 'chrome';\` runs on the device and returns rows exactly like a database query would. osquery is a translator between SQL and OS-specific state.

### Tables are platform-specific

Not every table exists on every platform:

- \`startup_items\`: macOS only.
- \`appcompat_shims\`. Windows only.
- \`rpm_packages\`. Linux only.
- \`system_info\`, \`processes\`, \`users\`: cross-platform.

Writing a query that targets a Mac-only table will return empty rows on a Windows host, not an error. This is a common source of "my query returns nothing!" tickets.

### Scheduled vs live queries

Two completely different mechanisms:

- **Scheduled** (Fleet calls these "reports"): a query that runs on the device on a cadence (every N seconds). Results post back on a logger cadence. Powers the dashboard, policies, and software inventory. The device never gets ad-hoc instructions. It's reading from its baked-in config.
- **Live** (Fleet calls these "live queries"): the admin fires an ad-hoc query right now and watches results stream in. Travels over a different endpoint (\`/distributed/read\`). Doesn't persist in \`query_results\`: results flow through Redis pubsub to a WebSocket.

### Snapshot vs differential

Scheduled queries have two modes:

- **Snapshot**: every tick, return the current full set of rows.
- **Differential**: return only rows that have *changed* since the previous tick (added, removed). Smaller payloads, time-sensitive data.

Fleet mostly uses snapshot for inventory, differential for change detection.

### Packs

A "pack" is a bundle of scheduled queries. osquery itself has a pack format. Fleet largely replaces packs with its own saved-report + policy abstractions, but packs still exist and admins can import them.

### The Fleet ↔ osquery handshake

- osquery enrolls once at install (\`POST /api/osquery/enroll\`) and gets a per-host **node key**.
- Every ~10s osquery calls \`POST /api/osquery/config\` ("what's my schedule?") and \`POST /api/osquery/distributed/read\` ("anything to run right now?").
- Results post to \`POST /api/osquery/log\` (scheduled) or \`POST /api/osquery/distributed/write\` (live).

That's the whole protocol. Covered in detail on the osquery TLS page.

### Debugging "my query returns nothing"

Common causes, in order of likelihood:

1. **Query targets a Mac-only table on Windows hosts** (or vice versa). Check the osquery table reference.
2. **Query requires elevated permissions**: the \`users\` table shows OS users; the \`shadow\` table (if you had one) would need root. Some tables need privileges osquery doesn't have on that host.
3. **Query uses the wrong column name**: osquery schema changes between versions.
4. **Device isn't running osquery**: fleetd is stopped or hasn't checked in. Verify host.seen_time is recent.
5. **Discard-data flag is on**: results don't persist. Live view only.

### Why Fleet picked osquery

- Already open-source, no new agent to ship.
- SQL is a lingua franca; security engineers know it.
- Cross-platform (Mac, Windows, Linux, newer tooling for ChromeOS/iOS/iPadOS/Android via different approaches).
- A declarative data-query model fits Fleet's "describe desired state, let the server reconcile" philosophy.

### Where to go next

- osquery TLS: the wire-protocol details.
- Live queries: ad-hoc campaign mechanics.
- The query / report / policy terminology map lives on the Backend conventions page.

### Technical help

**SQL as an API** is the key mental shift. In webdev, you query YOUR OWN database. SQL is the shape of your schema, which you designed. In osquery, SQL is the shape of OPERATING-SYSTEM STATE, which Apple/Microsoft/Linux designed. You're asking macOS the same way you'd ask Postgres: "what processes are running?", "what packages are installed?" The device is the database; the schema is the OS.

The closest webdev analogy is a GraphQL schema you don't own. You get a documented set of tables and columns from the vendor (osquery schema docs), the implementation runs locally on the device, and queries return rows. You don't SELECT INTO or UPDATE. Osquery is read-only. The "mutations" happen through MDM (Apple, Microsoft, Google), not osquery.

Why it's hard to internalize at first: web-side, SQL implies "data I keep." Here, SQL means "state the OS keeps on your behalf, that you can introspect but not change via SQL." Once that flips, every Fleet mystery about how data gets collected becomes "oh, it's just a scheduled SELECT."
`,
  },

  {
    slug: "osquery-tls",
    label: "osquery TLS",
    category: "Agent & data collection",
    one_liner: "How Fleet talks to the agent on every managed host.",
    repoPaths: ["server/service/osquery.go", "orbit/"],
    body: `
### What this subsystem is

The HTTP(S) protocol Fleet speaks to the osquery agent (fleetd/Orbit) running on every managed host. It's a small set of JSON endpoints under \`/api/osquery/\`.

### Core endpoints

- \`POST /api/osquery/enroll\` → device sends a shared enroll secret, Fleet returns a per-host **node key** (bearer token).
- \`POST /api/osquery/config\` → device asks "what's my config right now?"; Fleet returns scheduled queries, options, labels.
- \`POST /api/osquery/distributed/read\` → device asks "is there anything I need to run right now?"; Fleet returns zero or more live-query SQLs.
- \`POST /api/osquery/distributed/write\` → device returns results from live-query runs.
- \`POST /api/osquery/log\` → device returns results from scheduled queries, plus osquery's own logs.
- \`POST /api/osquery/carve/*\` → file carving (osquery uploading a file in chunks).

### Lifecycle of a healthy host

1. fleetd starts, loads its config, calls \`enroll\` once at install time.
2. Every ~10s, calls \`config\` + \`distributed/read\`. If live-query work is waiting, runs it, posts to \`distributed/write\`.
3. On the schedule's own cadence (e.g. every 30 min), posts scheduled-query results to \`log\`.
4. On \`distributed/read\`, Fleet also sends refetch signals (e.g. "re-run detail queries to update your inventory").

### Edge cases

- **Node key rotates** on re-enrollment. Old key is forgotten.
- **Clock skew** on the host breaks TLS cert validation and query time-based filters.
- **Config caching**: fleetd caches the config file; a change in Fleet shows up at the next \`config\` call.
- **Carves** are chunked; the host uploads one block at a time to \`carve_blocks\`.

### FE analogue

The server never pushes to the agent. There is no WebSocket. All comms are agent-initiated polling. Think of it like a mobile app that polls your API every ten seconds for notifications, except the server is handing it SQL to run.
`,
  },
  {
    slug: "live-queries",
    label: "Live queries",
    category: "Agent & data collection",
    one_liner:
      "Ad-hoc one-shot queries that stream results from hosts to the UI.",
    repoPaths: ["server/service/async/", "server/pubsub/", "frontend/pages/queries/"],
    body: `
### What this subsystem is

Ad-hoc queries an admin runs right now, against a set of hosts, with results streaming back live. Not to be confused with scheduled queries ("reports"), which run on a cadence and persist results.

### Lifecycle

1. Admin clicks "Run" in the UI. API call creates a row in \`distributed_query_campaigns\` and populates \`distributed_query_campaign_targets\` with host IDs.
2. The UI opens a **WebSocket** to Fleet (campaign ID in the URL) and subscribes.
3. Fleet's WebSocket handler subscribes to a Redis pubsub channel keyed by the campaign ID.
4. Each affected host, on its next \`distributed/read\` poll, picks up the campaign's SQL.
5. Host runs the SQL, posts rows to \`distributed/write\`.
6. Fleet publishes those rows to the Redis channel.
7. Every Fleet instance subscribed to that channel receives the publish and forwards to their WebSocket subscribers.
8. UI renders rows as they arrive. Campaign closes when all hosts respond or a timeout fires.

### Why Redis

Live-query results are fan-out: one producer (the host), many potential consumers (any admin UI watching). Fleet is horizontally scaled: the instance that gets a host's result may not be the one holding the admin's WebSocket. Redis pubsub bridges them.

### Edge cases

- **Dormant hosts** don't respond. The campaign times out and shows "no response" per host.
- **Giant result sets** stream through Redis; Fleet caps the per-host row count.
- **Network blips** on the WebSocket cause the UI to reconnect; already-received rows are kept client-side.
`,
  },
  {
    slug: "auth-sessions",
    label: "Auth & sessions",
    category: "Identity & access",
    one_liner: "How Fleet decides who you are on each request.",
    repoPaths: ["server/service/sessions.go", "server/authz/"],
    body: `
### What this subsystem is

Fleet admin API auth. Not SSO (that's its own page). Not MDM auth. Just: how a logged-in human and their API tokens become a request context.

### How a login works (local user)

1. POST /api/v1/fleet/login with email + password.
2. Fleet looks up the user by email, verifies the bcrypt password hash.
3. Fleet generates a random session token, writes it into the \`sessions\` table with the user ID and an expiry.
4. Returns the token in the response.
5. Client stores it, sends \`Authorization: Bearer <token>\` on every subsequent call.

### How a request is authenticated

1. Middleware extracts the token from the Authorization header.
2. Looks up \`sessions\` by token.
3. Loads the user and attaches a \`viewer\` struct to the request context: who, their roles, their team memberships, premium license status.
4. Later, \`authz.Authorize(ctx, ...)\` calls read the viewer from the context.

### API tokens

An admin can create a long-lived API token in the UI. It's just another row in \`sessions\` with a longer expiry and metadata marking it as an API token.

### Edge cases

- Rotating a user's password invalidates their sessions.
- Admin-side logout deletes the session row.
- Role changes take effect on next request; the viewer is rebuilt per-request.
`,
  },
  {
    slug: "sso-scim",
    label: "SSO & SCIM",
    category: "Identity & access",
    one_liner: "Delegated auth and automated user provisioning from an IdP.",
    repoPaths: ["server/sso/", "server/datastore/mysql/scim.go"],
    body: `
### What this subsystem is

Two related things:

- **SSO**: delegating authentication to an IdP (Okta, Entra, Google) via SAML or OIDC. Fleet is the Service Provider.
- **SCIM**: letting the IdP push user/group lifecycle changes to Fleet without an admin manually creating accounts.

Fleet has TWO SSO flows:

1. **Admin login SSO**: a Fleet admin signs in via IdP.
2. **End-user authentication (EUA)**: during device enrollment, the end user signs in with their work identity so Fleet knows which human owns the device.

### SAML login (admin)

1. User clicks "Log in with SSO."
2. Fleet responds with a 302 to the IdP with a signed SAML AuthnRequest.
3. IdP authenticates the user (whatever MFA).
4. IdP POSTs a signed SAML Response to Fleet's Assertion Consumer Service (ACS) endpoint.
5. Fleet verifies the XML signature against the configured IdP cert.
6. Extracts email + attribute claims; finds the user (or auto-provisions one if JIT is enabled).
7. Mints a session the same way local-password login does.

### SCIM

The IdP pushes user/group changes to Fleet's SCIM API. Fleet stores them in \`scim_users\`, \`scim_groups\`, and join tables, then uses that data when evaluating authz for SSO logins.

### Edge cases

- IdP cert rotation: Fleet admin must update the configured cert.
- Clock skew between IdP and Fleet breaks SAML signature validation (strict time windows).
- Group-based roles depend on the IdP sending group claims; some configs forget.
`,
  },
  {
    slug: "apple-mdm",
    label: "Apple MDM",
    category: "Device management",
    one_liner:
      "Managing Macs, iPhones, and iPads through Apple's MDM protocol.",
    repoPaths: ["server/mdm/apple/", "server/mdm/nanomdm/", "server/mdm/nanodep/"],
    body: `
### What this subsystem is

The entire Apple MDM stack: enrollment, config profile delivery, commands (lock, wipe, install software), DEP (Automated Device Enrollment via ABM), DDM (newer declarative model), SCEP + ACME cert enrollment, APNs push.

### Embedded components

- **nanomdm**: lightweight MDM server. Handles the command queue, device checkin, cert auth.
- **nanodep**: Apple Business Manager / DEP integration. Syncs device assignments.
- Fleet's own code wraps both and adds the "Fleet-shaped" API.

### How a Mac enrolls via DEP

1. IT buys the Mac through Apple Business Manager and assigns it to Fleet's MDM server in ABM.
2. End user unboxes the Mac; on first boot it contacts Apple.
3. Apple tells the Mac: "your MDM is at fleet.example.com."
4. Mac hits Fleet's enrollment endpoint (\`/mdm/apple/mdm\` and related).
5. Fleet responds with an enrollment profile (XML plist).
6. Mac installs the profile → MDM channel is now open.
7. Fleet writes an \`host_dep_assignments\` row and a \`nano_enrollments\` row.
8. Setup assistant runs; bootstrap package installs; initial config profiles push.

### Day-two operations

- Admin uploads a \`.mobileconfig\` → \`mdm_apple_configuration_profiles\` row.
- The cron \`manage_apple_profiles\` diffs desired vs installed per host and queues install/remove commands.
- APNs push wakes the device.
- Device polls Fleet, gets the command, installs the profile, acks.

### DDM (Declarative Device Management)

Apple's newer declarative model. Instead of imperative commands, Fleet declares "the host should be in state X" and the device figures out the diff locally. Lives in \`mdm_apple_declarations\`.

### Edge cases

- APNs cert expiration (annual) is a classic ops outage.
- Mismatch between Fleet's view of profiles and Apple's actual device state is the #1 bug class ("why is this profile still 'pending'?").
- iPhones/iPads don't run osquery; iOS inventory comes via MDM queries, refreshed by the \`apple_mdm_iphone_ipad_refetcher\` cron.
`,
  },
  {
    slug: "microsoft-mdm",
    label: "Microsoft MDM",
    category: "Device management",
    one_liner:
      "Managing Windows devices via SyncML + Autopilot zero-touch enrollment.",
    repoPaths: ["server/mdm/microsoft/"],
    body: `
### What this subsystem is

Microsoft's enterprise MDM protocol is **SyncML** over HTTPS. More verbose and XML-heavy than Apple. Plus Autopilot for zero-touch provisioning (equivalent to Apple's DEP).

### How a Windows device enrolls

1. User signs into Windows with an Entra (Azure AD) account scoped to Fleet.
2. Windows discovers the MDM endpoint via Entra.
3. Device posts SyncML to Fleet's enrollment endpoint.
4. Fleet responds with a SyncML config.
5. Device acks each configuration setting via SyncML OMA-DM commands.

### Config profiles

A Windows "profile" is a SyncML document that references **CSPs** (Configuration Service Providers). Each setting maps to a CSP path, like \`./Device/Vendor/MSFT/Policy/Config/Experience/AllowCortana\`. The CSP on the device knows how to apply the setting.

### Autopilot

Admin registers the device's hardware hash in Microsoft's Autopilot service. On first boot, the device looks up its registration and auto-enrolls into Fleet. Fleet syncs the Autopilot device list via the \`microsoft_autopilot_sync\` cron.

### Conditional access

Fleet can act as a Microsoft **compliance partner**: when a user's device is non-compliant per Fleet's policies, Entra blocks the user from accessing other apps. Config in \`microsoft_compliance_partner_*\` tables.

### Edge cases

- SyncML is strict: a malformed element fails the whole sync.
- Entra group changes don't retroactively re-trigger enrollment.
- Windows' MDM diagnostic logs (on the device) are usually the only way to debug a stuck enrollment.
`,
  },
  {
    slug: "android-mdm",
    label: "Android MDM",
    category: "Device management",
    one_liner:
      "Managing Android via Google's AMAPI. Fleet talks to Google, Google talks to devices.",
    repoPaths: ["server/mdm/android/"],
    body: `
### What this subsystem is

Android Management API (AMAPI). A different model than Apple/Microsoft: Fleet does NOT talk directly to Android devices. Fleet talks to Google. Google holds the device management channel.

### How a device enrolls

1. Admin creates an \`android_enterprises\` entry (one-time per Fleet customer, tied to a Google Workspace org).
2. Admin generates an enrollment token (or Fleet pushes zero-touch via \`android_zero_touch_tokens\`).
3. End user scans the enrollment QR / token on an Android device.
4. The device's Device Policy Controller contacts Google, which creates the device in Google's system and reports it to Fleet.
5. Fleet writes an \`android_devices\` row.

### Day-two operations

- Admin edits an Android profile → \`mdm_android_configuration_profiles\` updates.
- The \`mdm_android_profile_manager\` cron diffs and makes API calls to Google.
- Google pushes the policy to the device.
- Google reports the result; Fleet reconciles in \`host_mdm_android_profiles\`.

### Edge cases

- Fleet's view of device state can lag Google's by minutes.
- BYOD (personal) vs managed-device semantics differ: see Fleet's \`suppressAndroidBYODWipeStatus\` logic.
- Google API quotas are a real constraint for large fleets.
`,
  },
  {
    slug: "vuln-processing",
    label: "Vulnerability processing",
    category: "Vulnerability management",
    one_liner:
      "Downloading CVE feeds, matching installed software, writing results.",
    repoPaths: ["server/vulnerabilities/", "cmd/cpe/", "cmd/cve/"],
    body: `
### What this subsystem is

An hourly cron pipeline that answers "which CVEs apply to the software installed on managed hosts?"

### Pipeline

1. Download the latest CVE feeds:
   - NVD (National Vulnerability Database) for canonical US-gov entries.
   - OSV (Open Source Vulnerabilities) for cross-ecosystem data.
   - OVAL for Linux distros.
   - MSRC for Microsoft.
   - A separate Office feed for Microsoft Office.
2. Parse feeds, extract CPEs (software identifiers) and CVE metadata.
3. Pull installed software from \`host_software\` and \`software\` tables.
4. Match: for each installed CPE, find CVEs that affect it.
5. Write matches into \`software_cve\` (join software ↔ CVE).
6. Update rollups: \`vulnerability_host_counts\`, \`software_host_counts\`.

### Why this is tricky

- **False positives**: osquery reports software name "Chromium Content Shell" and the NVD feed has "chromium" as its CPE. If the match is too fuzzy, you get noise. Too strict, you miss real vulns.
- **False negatives**: vendor-rebranded software (think "Zoom Rooms" vs "Zoom") can slip the net.
- **Feed freshness**: CVEs can lag the real vuln by days or weeks.

### Edge cases

- Fleet has a known tension between the NVD CPE format and real-world software metadata. Many vuln bugs are CPE-matching bugs.
- MSRC data is XLSX files. Yes really.

### Technical help

**XLSX** = Microsoft Excel's spreadsheet format (since Excel 2007). Under the hood it's a zip archive containing XML files that describe cells, formulas, formatting, and relationships. \`unzip chrome.xlsx\` and you'll see a tree of \`.xml\` files. Not binary like the older \`.xls\`.

MSRC genuinely ships CVE and patch metadata as Excel workbooks. NVD gives you JSON, OSV gives you JSON, OVAL gives you XML. All machine-friendly. Microsoft hands you a spreadsheet. Fleet's vuln pipeline reads it with a Go library that unzips the archive and parses the sheet XMLs to extract CVE IDs, affected products, and KB-article references.

Why Microsoft does this is partly historic (MSRC predates the industry standardizing on JSON advisories), partly audience (enterprise patch managers consume this in Excel directly), partly inertia. The practical effect for Fleet: the Microsoft branch of the vuln pipeline reads differently than the other feeds. The only place in the whole pipeline where Fleet interprets a spreadsheet to decide if a Windows host is vulnerable.
`,
  },
  {
    slug: "activity-feed",
    label: "Activity feed",
    category: "Observability & integrations",
    one_liner: "The audit log of everything admins do.",
    repoPaths: ["server/service/activities.go", "server/fleet/activities.go"],
    body: `
### What this subsystem is

Every admin action in Fleet writes an **activity** row. "Admin X added policy Y." "Admin Z ran a live query on 42 hosts." This is both the audit log and the "Recent activity" feed in the UI.

### How it works

1. Service method completes its primary action.
2. Service calls \`svc.NewActivity(ctx, user, activity)\`.
3. \`NewActivity\` writes a row into \`activity_past\` (and \`activity_host_past\` for host-scoped activities).
4. The activity row carries a type enum, a details JSON blob, and the actor's user ID.
5. UI reads them with filters.

### Activity types

Enumerated in \`server/fleet/activities.go\`. Each type is a Go struct implementing \`ActivityDetails\`. The struct's fields become the JSON blob.

### Streaming

If the admin has configured activity streaming to a SIEM / webhook endpoint, the \`activities_streaming\` cron picks up new rows and ships them.

### Edge cases

- Forgetting to call \`NewActivity\` after a mutating service call is the #1 audit-log bug.
- Deleted users: activities keep the actor_id, which refers to \`users_deleted\` after a hard delete, so the UI must handle "deleted user" labels.
`,
  },
  {
    slug: "webhooks-integrations",
    label: "Webhooks & integrations",
    category: "Observability & integrations",
    one_liner:
      "Jira, Zendesk, generic webhooks, SIEM streaming, calendar automations.",
    repoPaths: ["server/worker/", "server/service/webhooks.go"],
    body: `
### What this subsystem is

Fleet fires events out to third parties when things happen:

- Failing policies → create a Jira or Zendesk ticket.
- New vulnerabilities → ticket or webhook.
- Policy failure or software install → optionally create a calendar event on the end user's calendar.
- Activity feed → ship rows to a configured webhook or SIEM.

### How it works

Almost everything goes through the worker queue, not inline in the request:

1. Service decides a webhook / ticket should fire.
2. Service calls \`worker.QueueJob(ctx, ds, "jira", args)\` (or zendesk, or webhook).
3. The \`integrations\` cron (every minute) drains the queue.
4. Worker makes the HTTP call to Jira / Zendesk / whatever.
5. On failure, retry with backoff.

### Why async

External APIs are flaky. If creating a Jira ticket is slow, the admin's "save policy" button shouldn't block. Durability means a Fleet crash during the Jira call doesn't lose the ticket intent.

### Edge cases

- Credentials rot: Jira / Zendesk auth tokens expire. Jobs retry forever until an admin fixes them.
- Deduplication: Fleet avoids re-creating the same ticket for the same policy failure using per-fleet dedup keys.
`,
  },

  // ─── Software installs ───

  {
    slug: "software-installs-overview",
    label: "Software installs: Overview",
    category: "Software installs",
    featured: true,
    one_liner:
      "The five install types side by side: how they differ, who owns the bytes, how delivery works.",
    repoPaths: [
      "server/service/software_installers.go",
      "server/datastore/mysql/software_installers.go",
      "ee/server/maintainedapps/",
    ],
    body: `
### What this subsystem is

Fleet's Software library covers eight upload types: seven push a binary artifact to the host, and one (**Scripts**) ships admin-authored code that fleetd runs locally. All eight share the same admin workflow (upload once, scope to a fleet, trigger via admin button, self-service, or policy automation) and the same shared infrastructure (software titles, upcoming_activities queue, status tables). They diverge on *who owns the bytes* and *how the device gets them*. Read this before any of the per-type pages.

### The mental model: title vs installer

- A **software title** is the abstract thing: "Google Chrome". One row in \`software_titles\`.
- A **software installer** (or VPP app, in-house app, FMA) is the concrete artifact: a specific install type on a specific platform.

A single title *can* roll up multiple artifacts, but only across platforms, not across types. For example, one "Chrome" title can be backed by \`Chrome.pkg\` for macOS and \`Chrome.msi\` for Windows (both custom installers). What you can't do within a fleet is layer two install types onto the same (title, platform). A fleet's macOS Chrome entry is either a custom installer OR an FMA OR a VPP app, not two of them at once. If an admin wants to switch Chrome from FMA to a custom installer, that's a replace, not an add.

Admins act on titles; delivery acts on the concrete artifact underneath.

### The types at a glance

Ordered to match the Fleet UI's Software library. Click any underlined term for its glossary definition.

| Type | Catalog | Owns bytes | Delivery mechanism | Status table |
|---|---|---|---|---|
| **[FMA](glossary:fma) (macOS)** | Nightly Homebrew cask sync | Fleet | [osqueryd](glossary:osqueryd) config poll + [fleetd](glossary:fleetd) scripts | host_software_installs |
| **[FMA](glossary:fma) (Windows)** | Nightly winget package sync | Fleet | [osqueryd](glossary:osqueryd) config poll + [fleetd](glossary:fleetd) scripts | host_software_installs |
| **[VPP](glossary:vpp) (macOS)** | Apple VPP API | Apple | [APNs](glossary:apns) → Apple MDM InstallApplication → Apple CDN | host_vpp_software_installs |
| **[VPP](glossary:vpp) (iOS/iPadOS)** | Apple VPP API | Apple | [APNs](glossary:apns) → Apple MDM InstallApplication → Apple CDN | host_vpp_software_installs |
| **Managed Google Play** | Google Play catalog | Google | [AMAPI](glossary:amapi) policy update → Google → Play Store | host_vpp_software_installs (platform='android') |
| **Custom installer** | Admin-uploaded | Fleet | [osqueryd](glossary:osqueryd) config poll + [fleetd](glossary:fleetd) scripts | host_software_installs |
| **In-house apps** | Admin-uploaded IPA | Fleet | [APNs](glossary:apns) → InstallEnterpriseApplication → Fleet CDN | host_in_house_software_installs |
| **Scripts** | Admin-authored (bash / zsh / powershell) | Fleet | [osqueryd](glossary:osqueryd) config poll + [fleetd](glossary:fleetd) execution | host_script_results |

### Shared infrastructure

- \`upcoming_activities\` + per-type sibling tables track pending intent.
- \`svc.InstallSoftwareTitle\` is the single service entry point. It branches on title type and routes to the type-specific delivery path.
- \`host_software\` (populated by osquery detail queries) is the inventory source of truth across all types.

### Why not one unified type?

Each type exists because of a different constraint:

- **FMA**: common apps (Chrome, Slack, 1Password…) without admin curation
- **VPP**. Apple requires volume licensing for public App Store apps on managed devices
- **Managed Google Play**. Android Enterprise's sanctioned distribution path
- **Custom installers**: admins' own packages (internal tools, licensed software)
- **In-house apps**: iOS/iPadOS can't side-load; distribution needs Apple's Enterprise program
- **Scripts**: admin-authored code (bash / zsh / powershell) that fleetd runs on the host. Not a binary install, but behaves as a software library entry: upload, scope to a fleet, trigger via admin button or self-service

### Tier

All eight paths are premium-only. Software **inventory** (viewing what's installed on a host) is free; writing anything (installs, uninstalls, script execution) is premium.
`,
  },

  {
    slug: "software-install-custom",
    label: "Software: Custom installers",
    category: "Software installs",
    one_liner:
      "Admin uploads a .pkg / .msi / .deb / .exe. Fleet stores the bytes; fleetd runs the three-script lifecycle.",
    repoPaths: [
      "server/service/software_installers.go",
      "server/datastore/mysql/software_installers.go",
      "orbit/pkg/installer/",
    ],
    body: `
### What this subsystem is

The admin uploads a software package via the Fleet UI, Fleet stores the bytes in the file store, and fleetd downloads + runs it on target hosts.

### The three-script pattern

A custom installer bundles **three scripts** the admin can edit:

1. **install_script**: how to install (e.g., \`sudo installer -pkg "$INSTALLER_PATH" -target /\` on macOS)
2. **uninstall_script**: how to uninstall (symmetric to install)
3. **post_install_script**: runs after install succeeds

Plus a **pre_install_query**. SQL that must return at least one row for the install to proceed. Used for "only install if Chrome < v120 is on the host."

If the pre_install_query fails (returns zero rows), the install is skipped unless \`override_pre_install_query\` is set.

### Lifecycle

1. Admin uploads \`chrome.pkg\` via the UI. Fleet stores bytes in file store, metadata in \`software_installers\`.
2. Admin clicks Install on a host. Service writes a \`host_software_installs\` row with status \`pending\` and an \`upcoming_activities\` entry.
3. Host's fleetd calls \`POST /api/osquery/config\` on its next poll (~10s). Fleet responds with the pending install work.
4. fleetd runs pre_install_query locally. If it fails, status flips to \`failed\`.
5. fleetd downloads the installer bytes from Fleet.
6. fleetd runs install_script, then post_install_script.
7. fleetd posts to \`POST /api/fleet/orbit/software_install/result\` with exit codes + stdout/stderr tails.
8. Fleet updates host_software_installs to \`installed\` or \`failed\`.
9. Next osquery detail-queries run picks up the new software in host_software inventory.

### Edge cases

- **Script timeouts**: scripts have a timeout (~1h per script). Long installers can hang the flow.
- **Transient failures**: fleetd retries on network errors. Script failures don't retry. Admin re-triggers.
- **Admin re-upload**: new installer bytes don't auto-rebuild existing installs. Hosts need re-install.
- **Platform mismatch**: Fleet filters by platform; a .pkg won't be offered to Windows hosts.
- **Output truncation**: Fleet only stores the tail of stdout/stderr.

### Why fleetd runs it, not MDM

Custom installers go through osquery/fleetd (not MDM) because Fleet owns the bytes and the install logic. MDM is a thin command channel; fleetd is a full agent that can run arbitrary scripts.
`,
  },

  {
    slug: "software-install-fma",
    label: "Software: Fleet-maintained apps (FMA)",
    category: "Software installs",
    one_liner:
      "Nightly-maintained catalog of common apps (Chrome, Slack, 1Password…) sourced from Homebrew (Mac) and winget (Windows).",
    repoPaths: [
      "ee/server/maintainedapps/",
      "server/datastore/mysql/maintained_apps.go",
    ],
    body: `
### What this subsystem is

Fleet maintains a nightly-refreshed catalog of common business apps. Admins pick from the catalog instead of authoring their own installer scripts. The delivery path reuses the custom-installer plumbing. Only the *source* differs.

### Catalog sources

- **macOS**: Homebrew casks. The \`maintained_apps\` cron queries Homebrew's JSON API nightly.
- **Windows**: winget packages. Same cron, different source.

The sync produces entries in \`fleet_maintained_apps\` with name, version, platform, install/uninstall/post-install scripts, pre-install query, and installer URL.

### Lifecycle

1. **Nightly catalog refresh** (maintained_apps cron, 1h): fetch sources, upsert catalog rows, note version bumps.
2. **Admin adds FMA to a fleet**: picks "Chrome" from the catalog. Fleet copies the catalog entry's scripts + metadata into \`software_installers\` scoped to that fleet.
3. **Admin clicks Install**: identical path to custom installer from here.
4. **Catalog version bump**: if the fleet has auto-update enabled, the FMA auto-update cron (hourly) enqueues installs on hosts running an older version.

### Why FMA uses custom-installer delivery

FMAs ARE custom installers under the hood. Fleet just populates the three scripts + pre-install query for you. Once added, delivery is identical. Savings is admin time: no scripting required.

### Mac vs Windows differences

Same pattern, different sources:

- **Mac Homebrew casks** rely on brew's install logic in the generated scripts.
- **Windows winget** uses winget's CLI (\`winget install\`).
- **Version detection**: Mac uses brew's version pin; Windows uses winget's version string.

### Edge cases

- **Catalog outage**: if Homebrew/winget APIs are down, nightly sync fails. Existing FMAs still work.
- **Package rename upstream**: cask renames can drop apps until Fleet's mapping updates.
- **Platform mismatch**: Mac FMAs are filtered out of Windows fleets automatically.
- **Catalog vs installed delta**: osquery inventory may lag the catalog refresh.
`,
  },

  {
    slug: "software-install-vpp-mac",
    label: "Software: VPP (macOS)",
    category: "Software installs",
    one_liner:
      "Mac App Store apps delivered via Apple's Volume Purchase Program. Apple ships the bytes; Fleet assigns the license.",
    repoPaths: [
      "server/service/vpp.go",
      "server/worker/vpp_verification.go",
      "server/mdm/apple/",
    ],
    body: `
### What this subsystem is

Apple's Volume Purchase Program (VPP) lets a company buy App Store apps in bulk and assign them to managed devices. Fleet integrates with Apple's VPP API to assign licenses, then uses Apple MDM to tell the device to pull the app.

Fleet never owns the app bytes. Apple ships them from the App Store CDN.

### Lifecycle

1. **One-time setup**: admin uploads a VPP token (from Apple Business Manager). Stored in \`vpp_tokens\`, scoped per fleet.
2. **Catalog sync** (\`refresh_vpp_app_versions\` cron, hourly): Fleet calls Apple's VPP API to pull the catalog. Rows land in \`vpp_apps\`.
3. **Admin assigns app to a fleet**: creates a \`software_titles\` row + \`vpp_apps_teams\` association.
4. **Admin clicks Install**: Fleet writes \`host_vpp_software_installs\` row + \`vpp_app_upcoming_activities\` entry.
5. **apple_software worker** (drained by integrations cron, every minute): calls Apple VPP API to **assign the license** to the device's serial.
6. **APNs push**: once license is assigned, Fleet pushes APNs to wake the device.
7. **Device pulls InstallApplication**: device makes MDM check-in to Fleet; Fleet responds with InstallApplication command referencing the App Store ID.
8. **Device downloads from Apple**: OS MDM client pulls bytes from Apple's CDN. Fleet is uninvolved.
9. **MDM ACK**: device responds when install completes.
10. **Status**: Fleet updates host_vpp_software_installs.

### Why so many steps

Apple's VPP flow is explicitly two-phase: license assignment first (sync API call), delivery second (push + MDM pull). Fleet's worker job handles phase 1; APNs + Apple MDM handles phase 2.

### Edge cases

- **License exhaustion**: if the token has fewer licenses than target hosts, assignments fail. Status reflects \`license_unavailable\`.
- **Token expiry**: VPP tokens last ~1 year. Fleet surfaces warnings; expired tokens block new assignments.
- **Device not VPP-ready**: devices must be enrolled with a managed Apple ID OR via DEP with VPP enabled.
- **App not in device's region**: Apple scopes App Store availability by country.
- **Public vs custom B2B apps**: public App Store apps use normal VPP. Custom B2B apps (private to one org) also go through VPP but may have additional provisioning.

### Mac specifics

On Mac, VPP installs are generally seamless after license assignment. The Mac App Store handles the download. End users don't need a personal Apple ID. The VPP license is tied to the device.
`,
  },

  {
    slug: "software-install-vpp-mobile",
    label: "Software: VPP (iOS / iPadOS)",
    category: "Software installs",
    one_liner:
      "Same VPP mechanism as macOS, but enrollment type matters a lot. User-enrollment limits what you can push.",
    repoPaths: [
      "server/service/vpp.go",
      "server/worker/vpp_verification.go",
      "server/mdm/apple/",
    ],
    body: `
### What this subsystem is

VPP on iOS and iPadOS uses the same Apple-side infrastructure as macOS. The important difference is **enrollment type**: iOS has three modes with different capabilities.

### Enrollment types matter

- **Device enrollment** (DEP / zero-touch, corporate-owned): Fleet can push InstallApplication without user action. Silent installs. VPP license tied to device serial.
- **User enrollment** (BYOD on personal device): limited to the work container. Install prompts need user tap on first install. VPP license tied to Managed Apple ID.
- **Supervised** (corporate-owned via Apple Configurator / ABM): most powerful; same as device enrollment plus extra MDM commands.

### Lifecycle (device enrollment path)

Same as macOS VPP:

1. VPP token setup → catalog sync → app assignment to fleet.
2. Admin clicks Install → host_vpp_software_installs row, vpp_app_upcoming_activities entry.
3. apple_software worker assigns the license via VPP API.
4. APNs push wakes device.
5. Device MDM check-in → Fleet sends InstallApplication with App Store ID.
6. Device pulls app bytes from Apple CDN.
7. MDM ACK → Fleet updates status.

### Lifecycle (user enrollment path)

Mostly the same, with these differences:

- Device shows a notification: "Fleet is installing [App]. Open?". User has to accept the first time.
- Subsequent installs from the same MDM are silent.
- App ends up in the work container with a briefcase badge.
- Removing enrollment removes all VPP-assigned apps.

### Edge cases

- **Age restrictions**: iOS App Store enforces age restrictions on the device's Apple ID.
- **Family Sharing**: VPP apps can't be family-shared.
- **OTA updates**: once VPP-installed, apps update via normal App Store updates. MDM can force updates by re-pushing.
- **App not iPhone-compatible**: iPad-only apps won't push to iPhones.
- **Lost device**: VPP license survives device loss; can be re-assigned.

### Managed App Config

Beyond install, Fleet can push **Managed App Config**: a dictionary of configuration keys the app reads at runtime. Shipped via Apple MDM's Settings command, not VPP. Covered in the Apple MDM subsystem.

### iOS vs iPadOS

Nearly identical. iPadOS adds a few multi-user features (Shared iPad) that affect VPP license allocation but don't change the per-install flow.
`,
  },

  {
    slug: "software-install-inhouse",
    label: "Software: In-house apps",
    category: "Software installs",
    one_liner:
      "Private IPAs (iOS) or installers that Fleet owns the bytes for. Delivered via Apple MDM InstallEnterpriseApplication.",
    repoPaths: [
      "server/service/software_installers.go",
      "server/datastore/mysql/in_house_apps.go",
      "server/mdm/apple/",
    ],
    body: `
### What this subsystem is

Private apps the admin uploads directly to Fleet. Typically internal line-of-business apps on iOS/iPadOS not on the App Store. Fleet stores the IPA, serves it from the file store, and uses Apple MDM to tell the device to install from Fleet's URL.

### Why it's different from VPP

- **VPP**: Apple owns the bytes; Fleet assigns a license; device pulls from the App Store.
- **In-house**: Fleet owns the bytes; Fleet serves the IPA directly to the device.

In-house apps require Apple's **Enterprise Developer Program** account and must be signed with an enterprise distribution certificate.

### Lifecycle

1. **Admin uploads IPA + manifest**: Fleet validates signing, stores bytes in file store, metadata in \`in_house_apps\`.
2. **Admin assigns to a fleet**: software_titles row + association.
3. **Admin clicks Install**: host_in_house_software_installs row + in_house_app_upcoming_activities entry.
4. **APNs push** wakes the device.
5. **Device MDM check-in**: Fleet sends \`InstallEnterpriseApplication\` with a Fleet-served manifest URL + expected IPA hash.
6. **Device downloads from Fleet**: OS MDM client fetches manifest and IPA bytes from Fleet's endpoint.
7. **Device installs**: OS validates signature against the enterprise cert and installs.
8. **MDM ACK**: device reports status.

### Signing requirements

- **Enterprise distribution cert**: issued by Apple to Enterprise Developer Program members. Devices trust it only after a one-time trust prompt in Settings.
- **Provisioning profile**: embedded in the IPA, specifies which devices can install.
- **Expiry**: both cert and profile expire. Expired apps stop launching until re-signed and re-pushed.

### Edge cases

- **Cert revoked or expired**: Apple can revoke enterprise certs on abuse detection. All installed apps signed with that cert stop working.
- **Device not supervised**: user-enrolled devices may require manual trust acceptance in Settings.
- **Hash mismatch**: if IPA bytes change but the manifest hash doesn't update, install fails silently.
- **Bandwidth**: Fleet serves the IPA directly, consuming Fleet's bandwidth.
- **No incremental update**: full IPA re-download on every version bump.

### Why not VPP for private apps?

VPP's "custom B2B" path exists for private App Store apps, but requires Apple's involvement for every version. In-house apps bypass Apple entirely for distribution. Faster iteration for internal tools.
`,
  },

  {
    slug: "software-install-managed-google-play",
    label: "Software: Managed Google Play (Android)",
    category: "Software installs",
    one_liner:
      "Android's equivalent of VPP. Google owns the bytes; Fleet declares policy via AMAPI; Google Play on the device does the install.",
    repoPaths: ["server/mdm/android/", "server/service/android_amapi.go"],
    body: `
### What this subsystem is

Android doesn't have VPP. The equivalent is **Managed Google Play**. Google's business-tier catalog that admins can push apps from via Android Enterprise.

Fleet doesn't talk to the device directly. Fleet talks to **Google** (via AMAPI); Google tells the device; Google Play on the device handles the install.

### Lifecycle

1. **One-time setup**: admin links Fleet to their Managed Google Play enterprise. Fleet gets credentials to call AMAPI.
2. **Catalog browsing**: admin picks apps from the Managed Google Play catalog. Fleet queries AMAPI for app metadata.
3. **Admin assigns app to a fleet**: Fleet writes the app to a policy document sent to all hosts in that fleet.
4. **Policy update via AMAPI**: Fleet calls Google's \`enterprises.policies.patch\` API to push the updated policy.
5. **Google notifies the device**: Google's push service delivers the policy update to the Google Device Policy app.
6. **Google Play installs**: Google Play on the device sees the policy, pulls the app from the Play Store, installs in the work container.
7. **AMAPI callback**: Google calls Fleet's \`/api/v1/fleet/android_enterprise/pubsub\` endpoint via Pub/Sub when install completes.
8. **Fleet updates status**: host_mdm_android table reflects the new install.

### Why Fleet doesn't talk to the device

Android Enterprise is deliberately Google-mediated. The device trusts Google; Google trusts Fleet. Google holds the policy-enforcement authority. Reconciliation is on Google's side.

### Enrollment types

- **Fully managed** (corporate-owned, factory reset): entire device under policy. Installs land device-wide.
- **Work profile** (BYOD on personal Android): Fleet-pushed apps go into the work container. Personal apps are untouched. Removing enrollment removes the work container.
- **COPE** (corporate-owned, personally enabled): hybrid.

### Edge cases

- **App unavailable in device's country**: Google Play filters by region. Policy drops unavailable apps silently.
- **Device out of sync**: if the device hasn't checked in recently, policy lag. Push wakes it; dead device won't install.
- **Private apps**: Managed Google Play supports private app distribution via Google's EMM channel. Fleet surfaces as "Private apps"; same policy-push path.
- **Policy conflicts**: a device gets policy from one EMM. Fleet + another EMM = only one wins.
- **Google infra outage**: Fleet can't deliver installs when AMAPI or Pub/Sub is down.

### Tier

Entirely premium. All of Fleet's AMAPI integration is premium-only.
`,
  },

  // ─── Development process ───
  {
    slug: "testing-strategy",
    label: "Testing strategy",
    category: "Development process",
    featured: true,
    one_liner:
      "Every automated test type Fleet runs, what each one covers, how to run it locally, and which CI suite it belongs to.",
    repoPaths: [
      "Makefile",
      ".github/workflows/test-go.yaml",
      ".github/workflows/test-go-suite.yaml",
      ".github/workflows/test-js.yml",
      "docs/Contributing/getting-started/testing-and-local-development.md",
      "docs/Contributing/guides/ui/fleet-ui-testing.md",
      "handbook/engineering/Load-testing.md",
    ],
    body: `
### What this subsystem is

Fleet runs several distinct test types across Go backend, TypeScript frontend, and specialized agents. CI runs all of them on every PR via a matrix of workflows. This page is the catalog: what each one covers, how to run it locally, and which CI suite it belongs to.

### Where this all lives in the repo

- **Makefile**: the orchestration layer. \`test\`, \`test-go\`, \`test-js\`, \`lint\`, \`run-go-tests\` are the user-facing targets.
- **.github/workflows/test-go.yaml**: top-level Go test workflow. Fans out to the suite-level workflow.
- **.github/workflows/test-go-suite.yaml**: the reusable per-suite workflow. Branches on \`inputs.suite\` to configure docker-compose services and \`CI_TEST_PKG\`.
- **.github/workflows/test-js.yml**: the frontend Jest workflow.
- **~35 other test-\\*.yml workflows**: specialized change-type-specific tests (FMAs, fleetd packaging, Android, etc.).

### Go test suites (the big one)

Fleet slices Go tests into **10 suites** so CI parallelizes cleanly and docker-compose only spins up services each suite actually needs. The matrix lives in \`test-go.yaml\`:

| Suite | What's in it | External deps | Notes |
|---|---|---|---|
| **fast** | No external deps, no DB, no Redis | None | Runs on every PR, fastest signal. **Do not add dependencies to this suite.** |
| **scripts** | \`cmd/scripts/...\` and tooling | None | Also no-deps, separate from fast. |
| **service** | \`server/service/*_test.go\` with \`-skip=^TestIntegrations\` | MySQL + Redis | Unit tests of the service layer only; integration tests are in the integration suites. |
| **integration-core** | \`server/service\` integration tests, core only | MySQL + Redis + S3 + SAML IdP | Full HTTP round-trips through the admin + osquery APIs. |
| **integration-enterprise** | Premium-only integration tests | MySQL + Redis + S3 + SAML IdP | The \`ee/\` wrapper's behavior. |
| **integration-mdm** | MDM-specific integration tests | MySQL + Redis + nanomdm fixtures | Apple + Windows + Android MDM flows. |
| **mysql** | \`server/datastore/mysql/*_test.go\` | MySQL | Database layer only. |
| **fleetctl** | \`cmd/fleetctl/...\` | varies per test | CLI behavior. |
| **vuln** | \`server/vulnerabilities/...\` | varies | Vuln pipeline. |
| **main** | Everything else | MySQL + Redis + S3 + localstack + Azurite | Catch-all; also runs under the most dockerized dependencies. |

The CI matrix also runs each MySQL-touching suite against **three MySQL versions** (8.0.42, 8.4.8, 9.5.0) on scheduled cron, and against the single pinned version on PRs.

### Running Go tests locally

Everything routes through the Makefile:

- **All unit tests, no external deps**:
  \`\`\`
  go test ./server/fleet/...
  \`\`\`
- **A specific package with MySQL + Redis set up**:
  \`\`\`
  make run-go-tests PKG_TO_TEST="server/datastore/mysql"
  \`\`\`
  Sets \`MYSQL_TEST=1 REDIS_TEST=1 S3_STORAGE_TEST=1 SAML_IDP_TEST=1 NETWORK_TEST=1\` and runs with \`-v\`.
- **Named tests within a package**:
  \`\`\`
  make run-go-tests PKG_TO_TEST="server/service" TESTS_TO_RUN="TestInstallSoftwareTitle"
  \`\`\`
- **With a specific \`go test\` flag**:
  \`\`\`
  make run-go-tests PKG_TO_TEST="server/service" GO_TEST_EXTRA_FLAGS="-race"
  \`\`\`
- **The whole CI test-go target**:
  \`\`\`
  make test-go
  \`\`\`
- **Everything (lint + Go + JS)**:
  \`\`\`
  make test
  \`\`\`
  Same thing CI runs.

### Standing up the test dependencies

Fleet uses docker-compose to bring up test services. From the repo root:

\`\`\`
docker compose up -d mysql_test mysql_replica_test redis s3 saml_idp mailhog
\`\`\`

Each service corresponds to a \`*_TEST\` env var the Go tests read.

### Schema tests

\`make test-schema\` regenerates \`server/datastore/mysql/schema.sql\` from the current migrations. Run it when you touch migrations so \`schema.sql\` stays in sync. CI has a separate \`test-db-changes.yml\` workflow that enforces this.

### Linting

Treated as a test type by CI:

- **lint-go**: \`make lint-go\` runs \`golangci-lint\` with Fleet's config (\`.golangci.yml\`), plus the custom \`check-no-testing-in-prod\` + \`check-nilaway-func-size\` guards.
- **lint-go-incremental**: same, but scoped to files changed since branching from main. Fast; use this during active development.
- **lint-js**: \`make lint-js\` runs ESLint + TypeScript against the admin UI.

### Frontend tests (Jest + Testing Library)

The admin UI uses **Jest** + **@testing-library/react**:

- Config at \`frontend/test/jest.config.js\`.
- Run with \`yarn test\` from \`frontend/\` or \`make test-js\` from repo root.
- Covers four layers per Fleet's strategy doc: shared utilities, UI building blocks (components + hooks), app widgets (feature-level), user journeys (page-level flows).
- Mocking: MSW for API, mocks for things like \`react-router\`.

Fleet's full UI testing strategy. Philosophy, what to test, what not to test, cheat sheet. Lives in \`docs/Contributing/guides/ui/fleet-ui-testing.md\`.

### End-to-end tests

Covered in detail in \`docs/Contributing/getting-started/testing-and-local-development.md\` under "End-to-end tests":

- Driven from the admin UI stack.
- Interactive mode for iteration; command-line mode for CI.
- Prep steps include seeding simulated hosts via the \`cmd/osquery-perf\` tool.

### Specialized workflows

Beyond the main Go + JS matrix, Fleet has ~25 other test-\\* workflows for specific change types:

- **test-fma-darwin.yml / test-fma-windows.yml**: validate each FMA (Fleet-maintained app) still installs on fresh VMs.
- **test-fleetd-chrome.yml**: fleetd Chrome extension.
- **test-fleet-agent-downloader-changes.yml**: agent-downloader behavior.
- **test-android.yml**: Android MDM pipeline.
- **test-db-changes.yml**: enforces schema.sql ↔ migrations consistency.
- **test-mock-changes.yml**: mocks-under-\`server/mock\` consistency.
- **test-yml-specs.yml**: YAML spec files validated.
- **test-website.yml**: fleetdm.com (separate Sails.js stack).
- **test-puppet.yml**: fleetd Puppet module.
- **test-vulnerability-dashboard-changes.yml**, **test-configuration-profile-generator.yml**, **test-tools.yml**, etc.

Each is triggered by its own path filter on PRs, so you don't pay for them unless you touched the matching code.

### Load tests

A separate category: \`.github/workflows/loadtest-\\*.yml\` + Fleet's "Scaling Fleet" story.

- **loadtest-osquery-perf.yml**: spins up thousands of simulated hosts via \`cmd/osquery-perf\`.
- **loadtest-infra.yml**, **loadtest-shared.yml**: shared infrastructure.
- Full write-up in \`handbook/engineering/Load-testing.md\`.
- Typical use: validate a new feature doesn't regress host-check-in throughput, osquery result-ingest latency, or MDM reconciler-cron run time at scale.

### Where the official docs live

Three handbook / docs pages worth bookmarking:

- **[fleetdm.com/docs/contributing/testing-and-local-development](https://fleetdm.com/docs/contributing/testing-and-local-development)**: the ground truth for how to run every test type locally. Rendered from \`docs/Contributing/getting-started/testing-and-local-development.md\`.
- **[fleet UI testing guide](https://fleetdm.com/docs/contributing/fleet-ui-testing)**. Fleet's frontend testing strategy, philosophy, what to test, cheat sheet. From \`docs/Contributing/guides/ui/fleet-ui-testing.md\`.
- **[fleetdm.com/handbook/engineering/Load-testing](https://fleetdm.com/handbook/engineering/Load-testing)**: load-test setup, tools, scenarios, metrics Fleet watches.

### Edge cases

- **Race detector**: scheduled cron runs enable \`-race\` (\`RACE_ENABLED: true\`); PR runs don't. Catches races on the nightly build.
- **Flaky integration tests**: typically caused by Redis/MySQL docker-compose timing. If a suite flakes consistently in CI, the fix is usually in the suite's \`CI_TEST_PKG\` setup, not the test.
- **Suite ownership**: \`service\`, \`integration-core\`, \`integration-enterprise\`, \`integration-mdm\` all touch some of the same files. If a bug reproduces in one but not another, check whether it's in a \`TestIntegrations\\*\` function (integration suites) or a regular \`Test\\*\` (the \`service\` suite skips those).
- **\`fast\` suite drift**: easy to accidentally add a dependency and silently break the "fast is dep-free" promise. The CI workflow has a comment forbidding it, but it's not enforced automatically.
- **Change-type-specific workflows have path filters**: if you edit \`frontend/\` the Go matrix still runs but \`test-fma-darwin.yml\` doesn't. Combined with required-checks on PRs, this means some workflows are only triggered by specific file paths.
`,
  },
];

export function getSubsystem(slug: string): Subsystem | undefined {
  return SUBSYSTEMS.find((s) => s.slug === slug);
}
