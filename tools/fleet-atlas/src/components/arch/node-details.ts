// Depth-y drill-in content for each architecture node.
// Written for a senior FE engineer who did a backend bootcamp in 2021
// and hasn't touched backend since. Rebuilds mechanisms, not vocabulary.

export const NODE_DETAILS: Record<string, string> = {
  osquery: `
### What it actually is

osquery is an open-source agent that lets you query a device's state (processes, users, installed packages, Bluetooth status, basically anything) using SQL. It runs on every host Fleet manages. Fleet ships its own wrapper around osquery called **fleetd** (sometimes still called "Orbit") that handles updates, scripts, and the MDM channel.

### Why Fleet needs it

A device management platform has to see what's happening on devices. Fleet's answer is: don't invent a new protocol, just run SQL. The device opens an HTTPS connection to Fleet, Fleet hands back SQL, the device runs it locally, and sends back rows. Fleet never pulls data from the device without the device asking first.

### How it works

1. fleetd starts on the host and calls \`POST /api/osquery/enroll\` with a shared secret. Fleet returns a per-host **node key** (think: session token).
2. Every ~10 seconds fleetd calls \`POST /api/osquery/config\` and \`POST /api/osquery/distributed/read\` to pick up work.
3. If Fleet has queued a query, fleetd runs it locally and posts the rows back to \`distributed/write\`.
4. Scheduled queries (reports) run on fleetd's own clock and submit results on their own cadence.

### Your bootcamp parallel

You built REST APIs where the server was the one with the database. Here the agent is the database (it holds the only truth about that device) and Fleet is the client asking for rows. The server-client roles are flipped from what you're used to.

### What to notice

- Hosts talk to Fleet over regular HTTPS. There is no osquery "protocol." It's JSON over HTTP with specific endpoint names.
- The "node key" is just a bearer token stored in the hosts table. If a device re-enrolls, it gets a new node key.
- Open Mode 2 and trace one of these endpoints to see the full flow.
`,

  "mdm-push": `
### What it actually is

Three different push notification systems:
- **APNs** (Apple Push Notification service) for Apple devices
- **FCM** (Firebase Cloud Messaging) for Android
- **WNS** (Windows Notification Service) for Windows MDM

Fleet never pushes data in the push itself. The push is just a "hey, wake up and check in with your MDM server."

### Why Fleet needs it

osquery runs on a 10-second-ish timer so Fleet can wait. MDM is a different story: when an admin says "install this profile right now," you don't want to wait up to an hour for the device to notice. Apple and Microsoft require you to send a push to wake the device.

### How it works

1. Admin takes an action in Fleet (install profile, run command, lock device).
2. Fleet writes an MDM command to MySQL.
3. Fleet sends a push via APNs/FCM/WNS to the device identifier it knows about.
4. The device wakes up and makes a request to Fleet's MDM endpoint to find out what changed.
5. Fleet returns the queued command, device executes it, device acks.

### Your bootcamp parallel

Think about how web push notifications work: your server sends a message to Google's push server, Google's push server delivers it to the user's browser, the browser wakes up a service worker. Same shape here, but the "service worker" is the OS's MDM client.

### What to notice

- Fleet needs an **APNs certificate** per customer (Apple requires it). That cert expiration is a classic ops gotcha.
- The push is best-effort. Fleet also schedules periodic MDM checkins as a safety net.
- Pushes don't carry the command. The device always pulls, never receives directly.
`,

  ui: `
### What it actually is

The Fleet web app. React 18, TypeScript, React Query v3 (\`react-query\`'s \`useQuery\`/\`useMutation\`), SCSS. Lives in \`frontend/\`. It's a plain SPA that calls Fleet's JSON API with a bearer token.

### Why it exists as a separate layer

The backend doesn't render HTML. The UI is just another API client, same as fleetctl. This matters because every page in the admin UI maps to API endpoints you can call directly, and anything Fleet admins can do, GitOps YAML can also do.

### How it works

1. User logs in; Fleet returns a session token stored in localStorage.
2. Every API call sends \`Authorization: Bearer <token>\`.
3. TanStack Query handles caching, invalidation, retries.
4. The API responds in JSON. There's no server-side rendering.

### What to notice

- This layer isn't backend. You can skip past it when learning Fleet BE.
- When something breaks, look at the Network tab. The URL is the fastest clue to what server-side code is involved.
`,

  fleetctl: `
### What it actually is

A single Go binary users run from a terminal. It's a CLI client for Fleet's REST API. GitOps is a mode where fleetctl reads a directory of YAML files and reconciles Fleet's state to match them.

### Why Fleet has it

Two reasons. One, admins want scripting (CI jobs, bulk changes). Two, Fleet is a GitOps-first product: you should be able to commit all config to a repo and have fleetctl apply it.

### How it works

- fleetctl shares its API client with the server (\`server/service/client.go\`).
- You authenticate once, token is saved to \`~/.fleet/config\`.
- Each subcommand is a thin wrapper around one or more API calls.
- GitOps mode diffs the YAML against Fleet's current state, computes a plan, and applies it.

### Your bootcamp parallel

Like running \`heroku config:set\` or \`gh pr create\`. GitOps mode is more like Terraform or kubectl apply: declarative state in version control, reconciler compares desired vs actual.

### What to notice

- When something works in the UI but not in GitOps, the bug is usually that an endpoint has special-cased logic for a single operation that the batch GitOps path doesn't hit.
- Look at \`cmd/fleetctl/\` to see which commands exist.
`,

  sso: `
### What it actually is

Fleet is a SAML / OIDC **service provider**. Your IdP (Okta, Entra, Google) is the authority on who the user is. Fleet trusts the IdP's signed assertion and creates a session.

### How a login works (SAML)

1. User clicks "Log in with SSO."
2. Fleet redirects browser to the IdP with a SAML AuthnRequest.
3. User authenticates with the IdP (whatever MFA, etc.).
4. IdP posts a signed SAML assertion to Fleet's ACS endpoint.
5. Fleet verifies the signature, extracts the email/group claims, looks up or creates the user, mints a session token.

### Your bootcamp parallel

You probably learned OAuth2 "log in with GitHub" in bootcamp. SAML is the same shape (redirect, callback, trust) but the token is a signed XML blob instead of a code+token pair.

### What to notice

- SSO lives in \`server/sso/\`. The handler is wired into the main HTTP router.
- There are TWO SSO flows in Fleet: admin login and end-user device enrollment ("End User Authentication"). Different code paths.
- SCIM (automatic user provisioning from the IdP) is also here conceptually, though implemented separately.
`,

  "http-router": `
### What it actually is

The function \`MakeHandler\` in \`server/service/handler.go\` is one giant registry. Every URL path Fleet responds to is declared here with its decoder, endpoint, and encoder. The router is **gorilla/mux**.

### Why Fleet builds it this way

Fleet uses the **go-kit** transport pattern. Instead of one function per endpoint that reads the request, does the work, and writes the response, go-kit splits that into three pieces:

- **Decoder**: HTTP request → typed Go struct
- **Endpoint**: typed Go struct → service call → typed response
- **Encoder**: typed response → JSON HTTP response

This lets Fleet put auth, logging, metrics, and tracing in middleware that wraps endpoints uniformly.

### Your bootcamp parallel

In Express you wrote \`app.post('/users', async (req, res) => { ... })\` where everything happened in one function. Fleet separates them: the equivalent in Fleet is roughly \`decodeCreateUserRequest\`, \`makeCreateUserEndpoint\`, and \`encodeJSONResponse\`. Verbose, but it means auth middleware doesn't have to touch any endpoint-specific code.

### How it works end to end

1. Request hits Fleet's \`http.Server\` (set up in \`cmd/fleet/serve.go\`).
2. gorilla/mux matches the URL to a handler.
3. Middleware chain runs: logging, auth check, rate limit, metrics.
4. Decoder parses the request body/params into a Go struct.
5. Endpoint calls into \`fleet.Service\` with that struct.
6. Encoder writes the service's return value as JSON.

### What to notice

- When you want to add a new endpoint, you edit \`handler.go\` AND the service AND the datastore. Three places minimum.
- Middleware order matters. Auth before rate limit, logging around everything.
- Open Mode 2 to see this layer in motion.
`,

  "device-router": `
### What it actually is

A separate set of HTTP routes under \`/device/...\` for the end-user My Device page. These are visible to employees (not admins) so they have their own auth: a **device token** (a URL-safe secret unique per host).

### Why Fleet separates them

Admin endpoints require a logged-in Fleet user with permissions. Device endpoints are hit by a browser on a user's laptop with just the device token in the URL. Different auth, different code paths, different authz policies.

### What to notice

- The device token lives in a URL like \`/device/<token>/foo\`. Anyone with the token can see that host's info.
- These endpoints are deliberately minimal. They cannot do anything an admin would do.
- Device router lives alongside the main HTTP router but gets its own middleware stack.
`,

  "mdm-router": `
### What it actually is

Protocol-specific endpoints under \`/mdm/...\` that implement what Apple, Microsoft, and Google require.

### What lives here

- **SCEP** (certificate enrollment): the device requests an identity cert
- **DEP** / **ABM** (Apple Business Manager sync): devices get pre-assigned to Fleet
- **MDM checkin** / **command**: the device picks up queued commands and acks them
- **SyncML**: Microsoft's XML protocol
- **AMAPI callbacks**: Google's Android Management API

### Why Fleet separates them

These endpoints don't speak JSON. They speak binary XML, plist, XML+base64, or protobuf depending on who's talking. The code is more "parser" than "handler." Rather than cram them into the main API router, they live in their own files that know the protocol.

### What to notice

- When a device "goes offline" or "doesn't enroll," 90% of the time it's an endpoint in here returning something malformed. These are the hardest paths to debug because the client is the OS.
- Logging is critical here. Fleet keeps per-request MDM logs in a dedicated table.
`,

  "live-query-ws": `
### What it actually is

A WebSocket endpoint that streams live-query results from Fleet to the admin UI as hosts respond.

### Why Fleet needs it

A live query goes to 10,000 hosts. They respond over seconds to minutes. The UI wants to show rows as they arrive, not after all hosts have reported. HTTP polling would be clumsy; WebSockets are a natural fit.

### How it works

1. Admin starts a live query. Fleet queues the query on each matching host (via the osquery distributed read endpoint).
2. Each host returns rows to Fleet.
3. Fleet publishes those rows on a **Redis pubsub channel** keyed by the query ID.
4. The admin UI subscribed via WebSocket receives each publish and renders a new row.

### Your bootcamp parallel

Imagine a chat server. Fleet is the pubsub broker, hosts are the message authors, the admin UI is the subscriber. Redis is doing the fan-out.

### What to notice

- Redis is load-bearing here. If Redis is down, live queries don't stream.
- One Fleet instance may receive a host's answer; another instance may be holding the admin's WebSocket. Redis bridges them.
`,

  "service-core": `
### What it actually is

A big Go interface, \`fleet.Service\`, declared in \`server/fleet/service.go\`, with hundreds of methods. The implementation lives in \`server/service/\`. This is where **business logic** lives: "to create a report, you must do these checks, write these rows, log this activity."

### Why Fleet separates service from router

- The router cares about HTTP: parsing, routing, auth middleware.
- The service cares about rules: can this user do this? What side effects fire? What goes in the activity log?
- The datastore cares about SQL.

Three layers, each ignorant of the other two's concerns. The service calls \`fleet.Datastore\` methods without ever writing SQL.

### Your bootcamp parallel

If you built a Rails or Express app with a \`controllers/\`, \`services/\`, \`models/\` split, you did this. Fleet is just more disciplined about it. The service never sees an \`http.Request\`.

### How a method typically goes

1. Service method is called with a typed struct from the endpoint.
2. First line: \`svc.authz.Authorize(ctx, ...)\` to permission-check.
3. Business logic: validation, lookups, transformations.
4. One or more datastore calls.
5. Activity log write (\`svc.NewActivity(...)\`).
6. Return the response struct.

### What to notice

- The service is where the "why" lives. Grep service.go for the method name to find the rules behind an endpoint.
- Every datastore call is one hop to MySQL. Watch for N+1 patterns in loops.
- Activity logs are side effects: skipping them means the audit trail is wrong.
`,

  "service-ee": `
### What it actually is

Enterprise-licensed functionality. The pattern: a Go struct in \`ee/server/service/\` **embeds** the core \`fleet.Service\` implementation, then overrides specific methods to add license checks or premium behavior before (or instead of) delegating.

### Why Fleet does it this way

Go has no inheritance. It has **embedding**: a struct can contain another struct's methods directly. So \`eeService{ Service }\` has every method the core service has. The ee file then redeclares methods it wants to intercept.

### Your bootcamp parallel

Decorator or higher-order component. The ee layer wraps and gates. In React terms:

\`\`\`tsx
function withLicense(Component) {
  return (props) => isPremium
    ? <Component {...props} premium />
    : <FreeFallback />;
}
\`\`\`

Same idea. The core method is the wrapped Component, the ee method is the wrapper.

### What to notice

- When looking for the "real" implementation of a service method, check \`ee/server/service/\` first for premium features.
- The core service file might have a stub returning \`ErrMissingLicense\`; the ee file has the actual code.
- Build tags are **not** used here. The ee code is always compiled in; it just won't run without a valid license.
`,

  authz: `
### What it actually is

Fleet uses **Open Policy Agent (OPA)** with policies written in a language called **Rego**. On every protected service call, the service passes the current user and the target object (host, policy, report, etc.) to OPA, which returns allow or deny.

### Why Fleet uses OPA

Role-based access control gets gnarly when there are team-level roles, global roles, premium gates, and object-type-specific rules. OPA lets Fleet express all of that as declarative policy files instead of if-statements scattered across the service layer.

### Your bootcamp parallel

You wrote \`if (user.role !== 'admin') return 403\` in middleware. Fleet does the same thing, but the "if" is in a Rego file, not inline in the handler. The upside is one audit of the policy covers the whole app.

### What to notice

- All Rego lives in \`server/authz/\`.
- Every service method should start with an \`authz.Authorize(ctx, target, action)\` call.
- Forgetting that call is a security bug. Fleet has architecture tests that look for it.
`,

  "datastore-iface": `
### What it actually is

A single giant Go interface declared in \`server/fleet/datastore.go\`. Every method is one data operation Fleet needs: \`ListHosts\`, \`NewReport\`, \`GetSoftwareInstaller\`, etc.

### Why Fleet does this

Three reasons:

1. **Testing**: the service layer takes a \`fleet.Datastore\` as a dependency. In tests, Fleet uses a generated mock from \`server/mock/\` that lets you make any method return anything.
2. **Separation**: the service layer can't accidentally write SQL. If it needs data, it has to add a method to the interface and implement it in \`server/datastore/mysql/\`.
3. **Future-proofing**: in theory, you could swap MySQL for Postgres by writing a second implementation of this interface.

### Your bootcamp parallel

Repository pattern. The service depends on an abstract repository, not a concrete database. Dependency injection in constructor.

### What to notice

- The interface is HUGE. Hundreds of methods. Don't try to read it top to bottom; search for the method you care about.
- Adding a new method here means also adding it to the MySQL impl AND regenerating the mock, or tests crash.
`,

  "datastore-mysql": `
### What it actually is

The implementation of \`fleet.Datastore\` against MySQL. Lives in \`server/datastore/mysql/\`. One file per topic area (hosts.go, policies.go, software.go, etc.) with a struct \`Datastore\` whose methods satisfy the interface.

### How a datastore method typically looks

1. Takes a context and typed arguments.
2. Builds a SQL query (often with sqlx and named parameters).
3. Decides primary vs replica (reads from replica, writes to primary).
4. Executes, scans rows into typed structs.
5. Wraps errors with \`ctxerr\`.
6. Returns.

### Schema migrations

Live in \`server/datastore/mysql/migrations/\`. Use **goose** as the migration runner. New migrations are named with a timestamp prefix. Fleet runs them on startup if the schema is behind.

### Your bootcamp parallel

Any ORM-free DAO layer. If your bootcamp used Knex or sqlx or raw \`pg\`, same shape: hand-written SQL, named parameters, scan rows into structs. Fleet doesn't use an ORM.

### What to notice

- Everything in here is "SQL as code." No GORM, no AREL. Read the SQL to understand the data.
- MySQL has read replicas. Fleet uses \`DBReader\` for queries that are OK to be slightly stale and \`DBWriter\` for the primary.
- Replica lag is a real thing: after a write, a read may not see it for a split second. That's a bug class to watch for.
`,

  mysql: `
### What it actually is

Fleet's source of truth. MySQL 8+. In production, usually AWS RDS Aurora with a primary and read replicas.

### What's in it

Everything durable:
- Hosts, their last check-in time, their labels
- Policies, policy results, policy runs
- Software inventory, software vulnerabilities
- MDM commands, enrollment records
- Users, sessions, API tokens
- The activity feed (every admin action)
- The async worker's job queue
- Scheduled queries (reports) and their stats
- Scripts, software installers metadata

### What's NOT in it

- Live query results (those go through Redis)
- Blob data (installers themselves go to S3 or local disk)
- Short-lived locks (Redis)

### Your bootcamp parallel

If your bootcamp used Postgres, substitute Postgres and almost everything applies. MySQL vs Postgres differences come up in: full-text search, JSON column syntax, generated columns, replication semantics.

### What to notice

- The schema is enormous. \`server/datastore/mysql/schema.sql\` is auto-generated from migrations.
- Indexes matter a lot. Fleet is read-heavy on hosts and software, and bad indexes show up as slow dashboards.
- Replica lag is why some "I just saved it" bugs happen: the read came from a replica that hadn't caught up.
`,

  redis: `
### What it actually is

An in-memory key-value store with pubsub. Fleet uses it for four things that MySQL would do badly:

1. **Distributed locks** for cron jobs ("only one Fleet instance should run vuln processing right now")
2. **Live query pubsub** (fan-out of host results to the admin UI's WebSocket)
3. **Rate limiting** (counters that reset on a sliding window)
4. **Small shared caches** (short-lived)

### Why Fleet needs it on top of MySQL

MySQL is durable but slow for high-rate ephemeral ops. Redis is fast and in-memory, but loses everything on restart. Fleet uses each for what it's good at.

### Your bootcamp parallel

If you learned Redis in bootcamp at all, it was probably "a cache" and "session storage." Fleet uses it mostly for coordination and messaging. Think of it less as a cache and more as "the little coordination server."

### What to notice

- Keys are namespaced by Fleet. Look at \`server/pubsub/\` and \`server/service/async/\` for patterns.
- Redis is a hard dependency. Fleet won't run without it.
- When a cron job appears to "run twice," the lock in Redis is the first thing to check.
`,

  filestore: `
### What it actually is

Blob storage: either an S3-compatible bucket in production or a local filesystem path in development. Holds files too large or weird to put in MySQL.

### What's in it

- Software installer packages (.msi, .pkg, .deb)
- Bootstrap packages (an admin's custom package deployed via MDM at enrollment)
- Config profiles (Apple .mobileconfig XML)
- Scripts (admin-authored, run on hosts)
- MDM manifests

### How Fleet uses it

MySQL stores metadata about the file (name, size, hash, upload time, who uploaded). The actual bytes live in the blob store, retrieved by hash or UUID.

### What to notice

- A broken installer is often a mismatch between the hash in MySQL and the bytes in the store.
- Dev mode uses a local directory, which is fine for a laptop but doesn't mimic S3 permissions or multipart upload quirks.
`,

  cron: `
### What it actually is

Named recurring jobs. Fleet runs many of them: vulnerability processing, host cleanup, policy runs, calendar sync, Google Workspace sync, MDM profile reconciliation, and more. Each job has a name, a schedule, and a function.

### Why Fleet does it this way

Horizontally scaled Fleet runs with many server instances behind a load balancer. If every instance ran every cron, you'd get duplicates: hosts cleaned up twice, Jira tickets created twice. Fleet solves this with **distributed locks in Redis**.

### How it works

1. Every Fleet instance tries to acquire a Redis lock named after the job.
2. Exactly one succeeds and runs the job.
3. On the next tick, every instance tries again. (The lock expires between runs.)
4. If an instance crashes mid-run, the lock expires and another instance picks up the next tick.

### Your bootcamp parallel

In bootcamp cron was usually "node-cron on your single server." Fleet is the multi-server version of the same thing. You've probably seen the pattern as "leader election."

### What to notice

- Cron code is in \`server/cron/\` and \`server/service/schedule/\`.
- Each job has its own interval. Some run every minute, some every hour.
- Flaky crons usually have one of: the lock expiring mid-run, Redis flakiness, or long-running work (the job takes longer than its interval).
`,

  worker: `
### What it actually is

A **durable job queue** stored in MySQL. The service enqueues a job ("send this Jira ticket," "install this software on this host"). A worker goroutine picks it up, runs it, marks it done or retries.

### Why not just do the work inline

Three reasons:

1. **Latency**: the admin's API request shouldn't wait for a Jira API round-trip.
2. **Durability**: if Fleet crashes mid-work, the job is still in MySQL and resumes.
3. **Retry**: external APIs fail. A queue lets you retry with backoff.

### Why MySQL, not Redis or a real queue

Fleet already requires MySQL. Making the queue durable in MySQL avoids a new dependency (RabbitMQ, Kafka, SQS). The tradeoff is you don't get the throughput of a real queue, but Fleet doesn't need it.

### How it works

1. Service calls \`worker.QueueJob(...)\`. A row is inserted into the \`jobs\` table.
2. Worker goroutines poll the \`jobs\` table on an interval.
3. A worker claims a row by updating its state to \`running\`.
4. Worker runs the job function. Success → delete the row. Failure → increment retry count, requeue with a delay.

### Your bootcamp parallel

Any job queue you touched (Sidekiq, Bull, Celery). Same mental model: enqueue, worker drains, retries on failure. Fleet's implementation is simpler because MySQL is the queue.

### What to notice

- Workers live in \`server/worker/\`.
- Each job type has its own file (jira.go, zendesk.go, software_worker.go, etc.).
- If a feature "sometimes works, sometimes doesn't," check whether it enqueues a worker job and whether that job is draining.
`,

  "mdm-apple": `
### What it actually is

The Apple MDM subsystem. Combines several embedded libraries:

- **nanomdm**: the lightweight MDM server (handles checkin, command queue, acks)
- **nanodep**: Apple Business Manager DEP integration (device assignment)
- **SCEP**: certificate enrollment
- **ACME**: a newer cert enrollment protocol
- **DDM** (Declarative Device Management): Apple's newer declarative config model
- **APNs**: push notifications

### How a Mac enrolls through DEP

1. The Mac is purchased through Apple Business Manager, assigned to Fleet's MDM server in ABM.
2. On first boot, the Mac contacts Apple's servers, which say "your MDM is at fleet.example.com."
3. The Mac hits Fleet's MDM enrollment URL. Fleet returns an enrollment profile.
4. The Mac installs the profile, which creates an MDM channel back to Fleet.
5. Fleet now has a two-way management channel for that device.

### What to notice

- Code lives in \`server/mdm/apple/\` plus the embedded \`server/mdm/nanomdm/\` and friends.
- Apple's requirements are strict. The commands and responses are plist XML, specified by Apple.
- Many Apple MDM bugs are protocol-level: a missing field in a plist, a wrong UUID format.
`,

  "mdm-microsoft": `
### What it actually is

Windows MDM over Microsoft's **SyncML** protocol (XML-over-HTTPS). Plus **Autopilot** integration for zero-touch provisioning.

### How a Windows device enrolls

1. User signs into Windows with an Azure AD / Entra account tied to Fleet.
2. Windows discovers the MDM endpoint via Entra.
3. Device posts SyncML to Fleet's MDM endpoint.
4. Fleet responds with configuration via SyncML commands.
5. Device acks each command.

### What to notice

- Code lives in \`server/mdm/microsoft/\`.
- SyncML is older and more verbose than Apple's protocol. Lots of XML.
- The "CSP" (Configuration Service Provider) is the thing on the Windows side that interprets the commands. Each setting maps to a CSP path.
`,

  "mdm-android": `
### What it actually is

Android device management via Google's **Android Management API (AMAPI)**. Fleet talks to Google's servers, which talk to devices. It's a slightly different model than Apple/Microsoft where Fleet talks to devices directly.

### What to notice

- Code lives in \`server/mdm/android/\`.
- Fleet is an EMM (Enterprise Mobility Management) partner with Google.
- Most device actions are HTTP calls to Google's API, with Google proxying to the device.
`,

  vuln: `
### What it actually is

A pipeline that answers: "which installed software on my devices has known CVEs?"

### The feeds Fleet consumes

- **NVD** (National Vulnerability Database): the canonical US gov feed
- **OSV** (Open Source Vulnerabilities): Google's cross-ecosystem feed
- **OVAL**: Linux distro-specific feeds
- **MSRC**: Microsoft's security feed
- **Office**: Microsoft Office vulnerability data

### How a vuln ends up on a host

1. Cron job downloads the latest feeds (hourly by default, configurable via \`vulnerabilities.periodicity\`).
2. Pipeline parses CPEs (software identifiers) and CVEs (vulnerability identifiers).
3. Pipeline matches installed software (collected from osquery) against known vulnerable CPEs.
4. Matches are written to the \`software_cve\` table (joined to hosts via \`host_software\`).
5. Fleet's UI reads from there.

### What to notice

- Code lives in \`server/vulnerabilities/\`.
- Supporting binaries live in \`cmd/cpe/\`, \`cmd/cve/\`, \`cmd/osv-processor/\`.
- The dreaded "false positive" and "false negative" bugs almost always come from CPE matching (the identifier format differs between feeds and reality).
`,
};
