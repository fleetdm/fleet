export type Layer =
  | "Router"
  | "Decoder"
  | "Endpoint"
  | "Service"
  | "Authz"
  | "Datastore"
  | "MySQL"
  | "Response";

export type TracerStep = {
  layer: Layer;
  title: string;
  narrative: string;
  snippet?: { path: string; start: number; end: number; caption?: string };
  technicalHelp?: { technical: string; analogies: string[] };
};

export type TracedEndpoint = {
  slug: string;
  method: "GET" | "POST" | "PATCH" | "DELETE";
  path: string;
  title: string;
  summary: string;
  overview: string;
  steps: TracerStep[];
  takeaways: string;
  stub?: boolean;
};

export const TRACERS: TracedEndpoint[] = [
  {
    slug: "list-hosts",
    method: "GET",
    path: "/api/v1/fleet/hosts",
    title: "List hosts",
    summary:
      "The canonical authed read. Filters, team scoping, replica read, iterator pattern.",
    overview: `
This is the endpoint behind the Hosts page in the admin UI. It's a great first trace because:

- It hits every architectural layer: router → decoder → endpoint → service → authz → datastore → MySQL → response.
- It demonstrates how Fleet separates HTTP concerns from business logic from SQL.
- It shows one real Go idiom you'll see everywhere in Fleet: **iterators** (introduced in Go 1.23).
- It uses the **team filter** pattern you'll see in nearly every multi-fleet read.

Follow it slowly. By the end you'll be able to add a new filter field end to end without asking anyone where to put what.
`,
    steps: [
      {
        layer: "Router",
        title: "The URL is registered in handler.go",
        narrative: `
Every HTTP route Fleet responds to is registered inside \`MakeHandler\` in \`server/service/handler.go\`. One giant function wires every URL path to its handler.

The \`ue\` here is a \`userEndpointer\`: a wrapper that applies the "logged-in user required" middleware before calling the endpoint function.

The three arguments are the URL path, the endpoint function (the business adapter), and a **zero-value instance of the request type** so the framework knows how to build and decode it.
`,
        snippet: {
          path: "server/service/handler.go",
          start: 515,
          end: 515,
          caption: "ue.GET registers the route with user-auth middleware applied",
        },
        technicalHelp: {
          technical: "ue.GET(path, endpointFunc, requestZeroValue) tells the framework: for this URL, decode into this struct, then call this function.",
          analogies: ["Like app.get(path, authMiddleware, handler) in Express, except the request-shape is declared up-front so decoding is automatic."],
        },
      },
      {
        layer: "Decoder",
        title: "listHostsRequest is auto-decoded from query params",
        narrative: `
Because \`listHostsRequest{}\` was passed as the zero-value, Fleet's endpoint infrastructure uses reflection + struct tags to decode query parameters into typed fields. You never write a decoder for this endpoint; the struct's \`url\` tags do the work.

The service never sees an \`*http.Request\`. By the time your business code runs, you have a typed Go struct.
`,
        snippet: {
          path: "server/service/hosts.go",
          start: 215,
          end: 250,
          caption:
            "listHostsRequest struct: query params → typed fields via struct tags",
        },
        technicalHelp: {
          technical: "Struct tags like `url:\"page\"` tell the decoder which query-string key fills which field.",
          analogies: ["Like Zod schemas you parse `req.query` into, except Go does it at the framework layer, not inside your handler."],
        },
      },
      {
        layer: "Endpoint",
        title: "listHostsEndpoint is the adapter between HTTP and Service",
        narrative: `
The endpoint function is thin on purpose. Its one job: take the decoded request, call the service, wrap the result in a response struct.

It also does a tiny bit of hydration (e.g. resolving a software ID into the actual software object before passing it to the service). This kind of logic should be rare. It's "the shape the UI wants to send" vs "the shape the service wants to receive."

Notice this function has no SQL and no auth checks. Both are elsewhere.
`,
        snippet: {
          path: "server/service/hosts.go",
          start: 269,
          end: 302,
          caption:
            "The endpoint resolves a few request-shape conveniences then delegates to svc",
        },
        technicalHelp: {
          technical: "The endpoint function is the Go-kit layer between transport (HTTP) and business logic (Service).",
          analogies: ["Picture a thin controller: no DB calls, no business rules, just marshal request → service → response."],
        },
      },
      {
        layer: "Service",
        title: "svc.ListHosts does the work (actually, it delegates to StreamHosts)",
        narrative: `
ListHosts in Fleet is a thin wrapper around a newer iterator-based method, StreamHosts. The service collects all hosts from the iterator into a slice.

Why the iterator? Fleet can be asked to return a lot of hosts. The iterator pattern (Go 1.23+) lets callers process rows as they come instead of waiting for all of them. For this endpoint, we still materialize them into a slice, but the pattern is in place for streaming callers.
`,
        snippet: {
          path: "server/service/hosts.go",
          start: 418,
          end: 432,
          caption: "ListHosts → StreamHosts; just collects the iterator",
        },
        technicalHelp: {
          technical: "iter.Seq2[T, error] = a sequence of (value, error) pairs you range over.",
          analogies: ["Async iterators / for-await loops. for host, err := range iter == for await (const [host, err] of iter)."],
        },
      },
      {
        layer: "Authz",
        title: "The first real line of StreamHosts: authz.Authorize",
        narrative: `
Here's the Fleet pattern: every protected service method opens with an authz check. The caller must be allowed to perform \`fleet.ActionList\` on a \`fleet.Host{}\` object.

Under the hood, this ships the request context (which includes the logged-in user, their roles, and their team memberships) and the target+action to an **OPA / Rego** policy evaluation. The policy returns allow or deny.

Miss this call and you have a security bug. Fleet has architecture tests that lint for its presence.
`,
        snippet: {
          path: "server/service/hosts.go",
          start: 460,
          end: 478,
          caption:
            "Authz at the top; team filter built from the viewer; premium filter sanitization",
        },
        technicalHelp: {
          technical: "svc.authz.Authorize(ctx, target, action) hands off to OPA; policy files decide allow/deny.",
          analogies: ["Instead of if (user.role !== 'admin') return 403 scattered through handlers, Fleet declares access rules in Rego and consults them uniformly."],
        },
      },
      {
        layer: "Datastore",
        title: "svc.ds.ListHosts is the Datastore interface call",
        narrative: `
The service holds \`ds fleet.Datastore\` as a dependency. It calls ListHosts with the team filter it just built from the viewer and the request options.

The service doesn't know whether the datastore is MySQL, a mock, or anything else. In tests, this is swapped for a mock from \`server/mock/\`. In prod, it's the MySQL implementation.
`,
        snippet: {
          path: "server/service/hosts.go",
          start: 470,
          end: 473,
          caption: "The datastore call: no SQL in sight",
        },
        technicalHelp: {
          technical: "ds fleet.Datastore is injected; service depends on the interface, not the concrete MySQL type.",
          analogies: ["Dependency injection via constructor. In tests, pass a stub; in prod, pass the real one."],
        },
      },
      {
        layer: "MySQL",
        title: "ds.ListHosts assembles SQL and reads from a replica",
        narrative: `
Here's where SQL lives. The function builds a SELECT with a long column list, joins to related tables (host_disks for disk info, etc.), applies filters constructed from the options struct, and executes against the **reader DB** (replica).

Two backend conventions to notice:

1. **DBReader vs DBWriter**: reads can be served from replicas; writes must go to the primary. Replica lag means a read right after a write may not see the write.
2. **applyHostFilters** is a shared helper that mutates the SQL string and params to inject WHERE clauses for team scoping, OS filter, labels, etc. Fleet avoids duplicating filter logic across endpoints.
`,
        snippet: {
          path: "server/datastore/mysql/hosts.go",
          start: 1184,
          end: 1200,
          caption:
            "ds.ListHosts: hand-written SQL assembled from the options struct",
        },
        technicalHelp: {
          technical: "Fleet uses sqlx with named/positional params; no ORM. SELECTs read replicas, INSERT/UPDATE/DELETE hit primary.",
          analogies: ["Like writing raw SQL with Knex or the pg driver, but with Go's typed struct scanning via sqlx tags."],
        },
      },
      {
        layer: "Response",
        title: "listHostsResponse carries the hosts (or error) back out",
        narrative: `
The service returns \`[]*fleet.Host, error\`. The endpoint wraps it as \`listHostsResponse\`. The framework's encoder calls \`json.Marshal\` on it and writes it to the HTTP response.

If error was non-nil, the response's \`Err\` field is populated and the encoder translates it to an HTTP status code (often 500, 400, or 403 depending on error type).
`,
        technicalHelp: {
          technical: "Response struct's JSON tags define the wire format. Err field is special-cased by the encoder.",
          analogies: ["res.json(result) with uniform error handling. The response type is the API contract."],
        },
      },
    ],
    takeaways: `
### What to carry forward

- **Three places, every time.** Adding a new endpoint touches at minimum: the router (handler.go), the service (business logic), the datastore (SQL). More if there's a new table.
- **Authz is service-layer, not router-layer.** Rego policies decide; the service calls Authorize.
- **No SQL above the datastore line.** If you're writing \`SELECT\` in a service method, you're in the wrong layer.
- **Reads can be stale.** Replica lag will bite you if you read right after a write expecting the write to be visible.
- **Iterators are the new pattern.** Expect to see more StreamXxx methods appear over time; they decouple "produce rows" from "collect rows."
`,
  },
  {
    slug: "get-host",
    method: "GET",
    path: "/api/v1/fleet/hosts/{id}",
    title: "Get a single host",
    summary:
      "The simplest authed read. Shows two-layer authz, cached rollups, and the hostDetailResponse composition.",
    overview: `
A single-row read is the opposite of ListHosts: no filters, no team-wide scan, just one host. But it still has to answer the thorniest Fleet question: *are you allowed to see this host?*

This trace is worth doing after list-hosts because it shows:
- A **two-layer authz pattern**: Fleet first checks that the viewer can list hosts at all, then (after loading team_id) checks that they can see this specific host.
- A **cached-rollup refresh**: Fleet recalculates per-host issue counts (failing policies + critical vulns) at most once a minute, inline during this request.
- A **response composition helper** (\`hostDetailResponseForHost\`) that enriches the raw host with device-mapping, policy status, software, labels, and more depending on query flags.
`,
    steps: [
      {
        layer: "Router",
        title: "The URL is registered in handler.go",
        narrative: `
Standard \`ue.GET\` registration: user-auth middleware required, dispatches to \`getHostEndpoint\` with a decoded \`getHostRequest\`.

Notice the route constraint \`{id:[0-9]+}\`: gorilla/mux rejects non-numeric IDs before any Fleet code runs.
`,
        snippet: {
          path: "server/service/handler.go",
          start: 520,
          end: 520,
          caption: "Numeric-only route constraint via regex",
        },
      },
      {
        layer: "Decoder",
        title: "getHostRequest pulls id + ExcludeSoftware",
        narrative: `
The path param (\`id\`) and query param (\`exclude_software\`) land in a typed struct. No body.
`,
        snippet: {
          path: "server/service/hosts.go",
          start: 903,
          end: 913,
          caption: "Minimal request shape: an ID and a toggle",
        },
      },
      {
        layer: "Endpoint",
        title: "getHostEndpoint builds options and calls the service",
        narrative: `
The endpoint sets \`IncludePolicies: true\` (unconditionally. Comment calls this out) and passes \`ExcludeSoftware\` through. The service does the auth check and the fetch; the endpoint then calls \`hostDetailResponseForHost\` to turn \`*fleet.HostDetail\` into the UI shape.
`,
        snippet: {
          path: "server/service/hosts.go",
          start: 915,
          end: 932,
          caption: "Endpoint composes the response via a helper",
        },
      },
      {
        layer: "Authz",
        title: "First check: 'can you list hosts at all?'",
        narrative: `
\`svc.GetHost\` opens with a *global* list-hosts authz check. If the viewer can't list hosts (an observer with no team access, say), stop here.

Device-token and device-URL authentications skip this because they already proved they own the device.
`,
        snippet: {
          path: "server/service/hosts.go",
          start: 938,
          end: 949,
          caption: "The 'can you see any hosts?' gate",
        },
        technicalHelp: {
          technical: "svc.authz.Authorize checks a generic action on a zero-value Host. It's a coarse pre-check.",
          analogies: ["Like checking isAuthenticated before loading the record and *then* checking record-level permissions."],
        },
      },
      {
        layer: "Service",
        title: "svc.ds.GetHostIssuesLastUpdated. Maybe refresh rollups",
        narrative: `
Fleet keeps per-host rollups in \`host_issues\` (failing policy count + critical vuln count). If the row is older than a minute, the service refreshes it inline before returning. This is why GetHost can be slower than ListHosts: a stale rollup triggers a recompute.
`,
        snippet: {
          path: "server/service/hosts.go",
          start: 951,
          end: 956,
          caption: "Inline rollup freshness check",
        },
        technicalHelp: {
          technical: "Lazy cache invalidation: read, check age, recompute if stale.",
          analogies: ["Like TanStack Query's staleTime: 60_000. Serve cached, revalidate in the background."],
        },
      },
      {
        layer: "Datastore",
        title: "ds.Host loads the row with its related tables",
        narrative: `
The datastore method joins host_disks, host_operating_system, host_additional, and more depending on the options. One SQL round-trip, then struct scan via sqlx tags.

Separate from this, the service makes several more datastore calls to load MDM state, policies, software, labels. Each with its own authz consideration.
`,
        technicalHelp: {
          technical: "Multiple datastore calls per endpoint are normal. Fleet does not try to one-shot every endpoint as a monster JOIN.",
          analogies: ["Think of it as a service orchestrating multiple API calls, not one GraphQL query."],
        },
      },
      {
        layer: "Authz",
        title: "Second check: 'can you see THIS host specifically?'",
        narrative: `
Now that \`team_id\` is loaded, Fleet re-runs authz against the loaded host. A team maintainer who can list hosts globally may still fail here if the host belongs to a team they don't manage.

Two-stage authz prevents leaking existence of hosts.
`,
        technicalHelp: {
          technical: "Pre-authz on the class (Host), then post-authz on the instance with team_id attached.",
          analogies: ["Load-then-check vs check-then-load. Fleet does both, in that order, to avoid existence leaks."],
        },
      },
      {
        layer: "Response",
        title: "hostDetailResponseForHost composes the final JSON",
        narrative: `
A helper walks the \`*fleet.HostDetail\` and hoists device-mapping, labels, software, policies, and MDM state into the response shape the UI expects. Query flags like \`exclude_software\` toggle which blocks are populated.
`,
        technicalHelp: {
          technical: "Composition in the response layer, not the service. Keeps the service surface stable while the API shape evolves.",
          analogies: ["Like React Query's select: whittle the server shape to what the UI wants."],
        },
      },
    ],
    takeaways: `
### What to carry forward

- **Authz is layered.** Pre-check at the class level before loading, post-check at the instance level after loading. Both matter.
- **Lazy rollups.** host_issues is recomputed inline on read-after-staleness. This is a cheap pattern Fleet uses in several places.
- **Response composition helpers** keep the service returning a stable domain type while the API shape evolves separately.
- **The route regex is a free validator.** Numeric constraints in the route save a decoder error.
`,
  },
  {
    slug: "create-user",
    method: "POST",
    path: "/api/v1/fleet/users/admin",
    title: "Create a user (admin)",
    summary:
      "The canonical authed write. Validation, premium gating, password hashing, invite token, session key.",
    overview: `
This is the write-path cousin of list-hosts, with four things worth noticing:

1. The admin-created endpoint (\`/users/admin\`) is **separate** from the invite-accept endpoint (\`/users\`). Different auth model.
2. The service does **premium gating** inline (Fleet Free can't create API-endpoint-scoped users).
3. The service rejects any attempt to pass \`api_endpoints\` through this endpoint even on Premium. That field is managed elsewhere. The endpoint checks for this before authz.
4. The response carries a **session token** (\`sessionKey\`) so the newly created user can log in without a password reset flow.
`,
    steps: [
      {
        layer: "Router",
        title: "Admin-create route is distinct from invite-accept",
        narrative: `
Fleet has two user-creation routes:
- \`POST /api/v1/fleet/users/admin\`: admin-authed create (this trace)
- \`POST /api/v1/fleet/users\`: accept-invite flow with no login required

They share \`createUserRequest\` but go through different endpoint functions and different authz.
`,
        snippet: {
          path: "server/service/handler.go",
          start: 369,
          end: 369,
          caption: "The admin-create route under user-auth middleware",
        },
      },
      {
        layer: "Endpoint",
        title: "createUserEndpoint rejects API-endpoint payloads up front",
        narrative: `
Before calling the service, the endpoint checks that \`req.APIEndpoints\` is nil. If a client tries to set that field through the normal create endpoint, Fleet returns \`fleet.NewInvalidArgumentError\` and marks auth as pre-checked so the error can bypass the authz middleware's "did you check authz?" assertion.
`,
        snippet: {
          path: "server/service/users.go",
          start: 42,
          end: 65,
          caption: "Pre-authz guardrail for a managed field",
        },
        technicalHelp: {
          technical: "setAuthCheckedOnPreAuthErr tells middleware 'yes, I know what I'm doing. Ship this error even though authz didn't run.'",
          analogies: ["Like short-circuiting form validation before touching any backend state."],
        },
      },
      {
        layer: "Service",
        title: "svc.CreateUser: authz, premium gating, team validation",
        narrative: `
The service method starts with an authz check parameterized on the teams the user is being assigned to. Then it validates the admin-create payload shape, then enforces Fleet Free restrictions (no premium roles, no API-only users with scoped endpoints or team memberships), then loads a teams summary to verify every assigned team actually exists.

All of this happens before any INSERT touches MySQL.
`,
        snippet: {
          path: "server/service/users.go",
          start: 129,
          end: 170,
          caption:
            "Authz → payload validation → license gating → team existence check",
        },
        technicalHelp: {
          technical: "Fleet puts premium gating in the service layer so both main and ee binaries share the same restriction code paths.",
          analogies: ["Equivalent of 'validate on both server and client'. but here the layering means premium checks aren't duplicated between core and ee."],
        },
      },
      {
        layer: "Service",
        title: "Password hash + session key generation",
        narrative: `
Below the snippet range, the service does the write-path work:
- Hash the password with bcrypt (if one was supplied).
- Generate a secure random \`sessionKey\` so the created user can log in immediately.
- Compose a \`*fleet.User\` with the hashed password + team memberships.
`,
        technicalHelp: {
          technical: "Password hashing is service-layer, not handler-layer. The datastore stores the hash.",
          analogies: ["You'd never hash on the client; same idea here. Hashing is business logic, not transport."],
        },
      },
      {
        layer: "Datastore",
        title: "svc.ds.NewUser INSERTs into users",
        narrative: `
The datastore method INSERTs into \`users\` and (if teams were assigned) into \`user_teams\`. Both go through the writer connection; replica lag means a read-back on the response might not see the row for a split second.

Fleet is strict about using \`DBWriter\` for writes and following them with any read-your-writes on the writer connection as well.
`,
        technicalHelp: {
          technical: "DBWriter for the INSERT; subsequent reads in the same request also go to DBWriter if you need to read-back-your-write.",
          analogies: ["Like optimistic-UI with a cache update: you write and immediately read the same key. But on the writer side."],
        },
      },
      {
        layer: "Service",
        title: "Activity log side effect",
        narrative: `
After the INSERT succeeds, the service calls \`svc.NewActivity(ctx, actor, activity_api.ActivityTypeCreatedUser)\` to record who created whom. This is the audit log; missing calls here are the #1 "where's the audit trail?" bug class.
`,
        technicalHelp: {
          technical: "Side-effect after primary action, deliberately NOT in a transaction with it. A failed activity write doesn't fail the user create.",
          analogies: ["Like firing analytics after a successful mutation, not before."],
        },
      },
      {
        layer: "Response",
        title: "createUserResponse carries the user + session token",
        narrative: `
The response includes the created \`*fleet.User\` and the generated \`sessionKey\`. The frontend uses the latter to seed a session for the new user without a password flow.

Response-layer encoding translates any error into an HTTP status code via the standard \`Err\` field.
`,
        technicalHelp: {
          technical: "A single response struct carries both the entity and the auth token; typical of 'just-created-this' POSTs.",
          analogies: ["Like a signup endpoint that returns the new user AND their session token in one shot."],
        },
      },
    ],
    takeaways: `
### What to carry forward

- **Two routes, one request type.** Admin-create and invite-accept share \`createUserRequest\` but travel different authz paths. Fleet does this a lot: different trust models, different endpoints, shared decoding.
- **Guardrails before authz.** When a field is categorically forbidden on an endpoint, reject it before authz. Pair it with \`setAuthCheckedOnPreAuthErr\` so middleware doesn't complain.
- **Premium gating lives in the core service.** Keeps license-tier behavior near business logic, not in build tags.
- **Activity writes are side effects, not transactions.** The audit log being missing is a bug; the user create failing because the audit log is down would be worse.
`,
  },
  {
    slug: "osquery-enroll",
    method: "POST",
    path: "/api/osquery/enroll",
    title: "Enroll a host (osquery)",
    summary:
      "Different auth model: enroll secret in, node key out. Writes to the hosts table. Unauthenticated at the router level.",
    overview: `
This endpoint is where every managed host in a Fleet instance first appears. It's unlike any other trace in this mode:

1. **No user auth.** The host presents a shared enroll secret (per-fleet) or a one-time secret (optional). Fleet validates it, not a session.
2. **No authz.** The service explicitly calls \`svc.authz.SkipAuthorization(ctx)\` because the authz machinery assumes a logged-in user context that doesn't exist here.
3. **Mutable identity.** The response is a new \`node_key\` (bearer token). The host persists this and uses it on every subsequent \`/api/osquery/*\` request.
4. **Hardware fingerprinting.** The service extracts UUID and serial from the \`system_info\` and \`os_version\` tables the agent includes in its enroll request, and uses them to deduplicate re-enrollments.
`,
    steps: [
      {
        layer: "Router",
        title: "Registered under 'noAuth' prefix (ne.POST)",
        narrative: `
Fleet's handler builder has several endpoint prefixes. \`ue\` is user-authed; \`ne\` is no-auth. The osquery enroll endpoint uses the no-auth prefix because there's no session to check. The host presents an enroll secret instead.
`,
        snippet: {
          path: "server/service/handler.go",
          start: 1181,
          end: 1181,
          caption: "No user-auth middleware for this route",
        },
      },
      {
        layer: "Decoder",
        title: "EnrollOsqueryAgentRequest decodes the enroll body",
        narrative: `
Request body includes:
- \`enroll_secret\`: the pre-shared secret the host was given
- \`host_identifier\`: how the host identifies itself (uuid, hostname, instance id, …)
- \`host_details\`: a map keyed by osquery table name (\`system_info\`, \`os_version\`) carrying the one-shot inventory the agent ships with its first-ever request
`,
        technicalHelp: {
          technical: "The decoded struct is defined in Fleet's contract package so the osquery agent and server share the shape.",
          analogies: ["Treat 'contract' packages like shared TypeScript types published to both ends of a boundary."],
        },
      },
      {
        layer: "Endpoint",
        title: "enrollAgentEndpoint just forwards to the service",
        narrative: `
Minimal endpoint logic. Hands the three fields to \`svc.EnrollOsquery\` and wraps the returned node key in the response struct.
`,
        snippet: {
          path: "server/service/osquery.go",
          start: 143,
          end: 152,
          caption: "Thin adapter between HTTP and service",
        },
      },
      {
        layer: "Authz",
        title: "svc.authz.SkipAuthorization. Explicit skip",
        narrative: `
The service immediately calls \`SkipAuthorization\`. Fleet has an architecture test that asserts every service method either calls \`Authorize\` or \`SkipAuthorization\`. Skipping is deliberate and auditable; nothing silent.

The comment calls out that authz is "currently for user endpoints only". Osquery TLS endpoints trust the enroll secret, not an authz policy.
`,
        snippet: {
          path: "server/service/osquery.go",
          start: 152,
          end: 160,
          caption: "An explicit skip beats a forgotten check",
        },
        technicalHelp: {
          technical: "SkipAuthorization is the escape hatch for request paths without a user. Architecture tests enforce one of {Authorize, SkipAuthorization} per method.",
          analogies: ["Like having a lint rule that requires either an auth check or an explicit \"// unauthenticated\" marker."],
        },
      },
      {
        layer: "Service",
        title: "Extract hardware identifiers from the agent-shipped tables",
        narrative: `
The agent's first \`enroll\` request includes a snapshot of some osquery tables. The service pulls \`uuid\` and \`hardware_serial\` from \`system_info\` and \`platform\` from \`os_version\`. These identifiers are the key input to Fleet's dedup logic. If the hardware matches an existing host, the service re-enrolls that host instead of creating a new one.
`,
        snippet: {
          path: "server/service/osquery.go",
          start: 162,
          end: 180,
          caption: "The enroll payload includes a one-shot inventory",
        },
        technicalHelp: {
          technical: "osquery tables arrive as nested string maps. Fleet pulls out exactly the fields it needs for identity.",
          analogies: ["Like a signup request body carrying device fingerprint fields; Fleet uses them to deduplicate re-registrations."],
        },
      },
      {
        layer: "Service",
        title: "Validate the enroll secret (one-time or shared)",
        narrative: `
Fleet checks the enroll secret in one of two paths:
- \`svc.lookupOneTimeEnrollSecret\` if one-time enroll secrets are enabled in config
- otherwise \`svc.ds.VerifyEnrollSecret\` against the \`enroll_secrets\` table

Either path produces a team_id (which fleet the enrolling host joins) and emits a rejection via \`svc.recordEnrollmentRejected\` if the secret doesn't resolve.
`,
        snippet: {
          path: "server/service/osquery.go",
          start: 182,
          end: 200,
          caption: "One-time secrets first; shared secret as fallback",
        },
      },
      {
        layer: "Datastore",
        title: "ds.EnrollHost. INSERT into hosts (or UPDATE existing)",
        narrative: `
Below the snippet range, the service hands off to a datastore method that either inserts a new row into \`hosts\` or re-adopts an existing row (same UUID + serial from a previous enroll). It mints a fresh node_key for both paths. For one-time secrets, it also writes to \`host_one_time_enroll_secrets\` to burn the token.
`,
        technicalHelp: {
          technical: "Create-or-adopt is one of Fleet's most repeated patterns: dedupe on hardware identity so re-enrollments stay 'the same host'.",
          analogies: ["Like upsert semantics on a user record keyed by email: new email = new row, existing email = adopt."],
        },
      },
      {
        layer: "Response",
        title: "node_key returned; host now has a bearer token",
        narrative: `
The service returns the newly-minted node_key. The endpoint wraps it in \`EnrollOsqueryAgentResponse\`. The agent stores it; every subsequent \`/api/osquery/*\` request includes the node_key as its auth proof.
`,
        technicalHelp: {
          technical: "Enroll secret → node key: it's an exchange of a shared secret for a per-host bearer token, like OAuth2's authorization code → access token.",
          analogies: ["A session token issued at signup. The host 'logs in' once and reuses the token forever (until re-enroll)."],
        },
      },
    ],
    takeaways: `
### What to carry forward

- **No user context, no authz.** Fleet's osquery TLS endpoints trust the enroll secret, not a Rego policy. \`SkipAuthorization\` is deliberate.
- **Hardware identity is king.** UUID + serial are what makes a re-enroll "the same host." Changing a motherboard = new host.
- **One-time secrets burn.** If enabled, the first use of the secret deletes it; a stolen token can't re-enroll a different device.
- **Node keys rotate per re-enroll.** Any time Fleet re-adopts a host, it mints a fresh token. Old tokens stop working immediately.
`,
  },
  {
    slug: "install-software-title",
    method: "POST",
    path: "/api/v1/fleet/hosts/{host_id}/software/{software_title_id}/install",
    title: "Install software on a host (Premium)",
    summary:
      "License-gated write with the Enterprise-wrapper pattern laid bare. Core returns ErrMissingLicense; ee overrides.",
    overview: `
This trace is specifically chosen to show the **Enterprise wrapper pattern** in action. The core service method is a license-error stub; the real logic lives in \`ee/server/service/software_installers.go\`. On a Premium instance, Go's struct embedding routes the call to the ee override.

Along the way you'll see:
- The admin UI's "Install" button touching four queue tables indirectly (upcoming_activities + per-type).
- A branch per software title type (custom installer / FMA / VPP / in-house) inside InstallSoftwareTitle.
- The deliberate choice to NOT download bytes server-side: Fleet writes intent and the host pulls on its next osquery poll.

Read this with \`/scenarios/install-from-host-library\` open for the step-level view.
`,
    steps: [
      {
        layer: "Router",
        title: "Compound numeric route under user-auth middleware",
        narrative: `
Two numeric path params. gorilla/mux validates both before any Fleet code runs. User-auth required.
`,
        snippet: {
          path: "server/service/handler.go",
          start: 454,
          end: 455,
          caption: "Two IDs in one URL; both numeric-constrained",
        },
      },
      {
        layer: "Decoder",
        title: "installSoftwareRequest just picks up the path params",
        narrative: `
No body on this endpoint. The two IDs in the URL are the entire request.
`,
        snippet: {
          path: "server/service/software_installers.go",
          start: 782,
          end: 793,
          caption: "A path-only request; no POST body",
        },
      },
      {
        layer: "Endpoint",
        title: "installSoftwareTitleEndpoint is a one-liner",
        narrative: `
Delegates to \`svc.InstallSoftwareTitle(ctx, hostID, titleID)\` and wraps the result. Does nothing else.
`,
        snippet: {
          path: "server/service/software_installers.go",
          start: 795,
          end: 804,
          caption: "Thin adapter",
        },
      },
      {
        layer: "Service",
        title: "Core InstallSoftwareTitle is a license-error stub",
        narrative: `
This is the Enterprise-wrapper pattern laid bare. The core service's method is three lines: skip authz (the ee override will do it), return \`ErrMissingLicense\`. On a Fleet Free instance, that's the only implementation reachable, and the admin gets a 402.
`,
        snippet: {
          path: "server/service/software_installers.go",
          start: 806,
          end: 814,
          caption: "Fleet Free: the method is a stub",
        },
        technicalHelp: {
          technical: "Core stubs with ErrMissingLicense; ee struct embeds core and overrides the method. Dispatch picks the ee version via Go's method set resolution.",
          analogies: ["Like a plan-gated feature flag, except enforced by code structure instead of a boolean check scattered through the codebase."],
        },
      },
      {
        layer: "Service",
        title: "On Premium, ee's InstallSoftwareTitle takes over",
        narrative: `
When Fleet is built with the Enterprise wrapper (every production binary), \`ee/server/service/software_installers.go:1778\` provides the real method. It:

1. Runs authz against the host and the title.
2. Loads the title and branches on type:
   - **Custom installer** → svc.insertSoftwareInstallation
   - **VPP app** → svc.insertVPPAppInstall (plus apple_software worker enqueue)
   - **FMA** → resolves to a custom installer under the hood
   - **In-house app** → svc.insertInHouseAppInstall

Each branch writes to \`upcoming_activities\` and a per-type companion row (\`software_install_upcoming_activities\`, \`vpp_app_upcoming_activities\`, …) with status=pending.
`,
        technicalHelp: {
          technical: "Branching happens inside the ee service, not the endpoint. The admin button doesn't know the title's type; the service figures it out.",
          analogies: ["Like a 'install' button whose click handler dispatches to one of several side-effects based on resource type. But the dispatch is server-side."],
        },
      },
      {
        layer: "Datastore",
        title: "Multi-table write: upcoming_activities + per-type",
        narrative: `
The install intent is persisted across multiple tables in one transaction. On success the admin sees a pending row immediately. The device-side delivery is driven separately by osquery polling (for custom + FMA + in-house) or by the apple_software worker (for VPP).
`,
        technicalHelp: {
          technical: "A single mutation touches multiple tables inside one DB transaction. If the transaction fails, nothing is partially-visible.",
          analogies: ["Like an atomic optimistic UI update across multiple cache entries. All or nothing."],
        },
      },
      {
        layer: "Service",
        title: "Activity + no bytes transfer",
        narrative: `
After the writes land, the service logs an activity ('user U queued install of X on host H'). It does **not** attempt to push the bytes to the device synchronously.

Why? Because Fleet never pushes to hosts. The device will see the pending install on its next osquery config poll (~10 s), download the bytes itself, run the install scripts, and post the result back. This decoupling is the whole reason upcoming_activities exists.
`,
        technicalHelp: {
          technical: "Write intent, let the client pull. The service is a scheduler, not an executor.",
          analogies: ["Like a React Query mutation that updates server state and relies on polling to pick up the completion."],
        },
      },
      {
        layer: "Response",
        title: "202-ish: success is 'queued', not 'installed'",
        narrative: `
The endpoint returns an empty success response. The admin UI immediately shows the pending row in Host → Software and polls the \`software/install/{install_uuid}/results\` endpoint for the actual outcome.
`,
      },
    ],
    takeaways: `
### What to carry forward

- **The ee pattern is in every premium feature.** Core service returns ErrMissingLicense; ee overrides. One codebase, one binary, runtime gating.
- **The service dispatches by title type.** The admin button is dumb; the service is where custom/VPP/FMA/in-house diverge.
- **Fleet never pushes to hosts.** This endpoint writes intent. The host pulls on its next poll. Every scenario in \`/scenarios\` repeats this pattern.
- **Activity logs are the audit trail for mutations.** Forgetting \`svc.NewActivity\` after a mutation is a review-caught bug class.
`,
  },
];

export function getTracer(slug: string): TracedEndpoint | undefined {
  return TRACERS.find((t) => t.slug === slug);
}
