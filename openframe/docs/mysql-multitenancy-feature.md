# Fleet MySQL multi-tenancy (OpenFrame)

Single-document reference for the OpenFrame shared-database multi-tenancy feature in this fork.
It lets **one Fleet server + one shared MySQL serve many tenants**, where **tenant = Fleet team
(`team_id`)**, with row-level isolation enforced at the datastore. Everything is behind a master
flag; **with the flag off, Fleet behaves exactly as the fork did before this feature** (schema
change aside — see [Backward compatibility](#backward-compatibility)).

## The master flag

`FLEET_OPENFRAME_MULTI_TENANCY_ENABLED` (maps the platform property
`openframe.fleet.multi-tenancy.enabled`). Parsed once, cached. Three states:

| State | Meaning |
|---|---|
| **off** (default) | Pre-feature fork behavior. Every fence is inert; a stray `FLEET_OPENFRAME_TEAM_ID` does **not** pin the process. |
| **on + process pin** (`FLEET_OPENFRAME_TENANT_UUID`, or `FLEET_OPENFRAME_TEAM_ID`) | **Pinned mode** — one Fleet per tenant on the shared DB; the whole process is scoped to one team. |
| **on + no pin** | **Shared mode** — one Fleet per cluster; **every request is pinned individually** (X-Tenant-Id header / host team / enroll-secret team), fail-closed. |

`ValidateOpenframeMultitenancy()` (called in `cmd/fleet/serve.go`) refuses to boot only on a
`FLEET_OPENFRAME_TEAM_ID` that is set-but-unparsable (a typo must not silently change the mode).
Key API in `server/fleet/openframe.go`: `IsOpenframeMultitenancy()`, `IsOpenframeSharedMode()`,
`OpenframeTeamID(ctx)`, `NewOpenframeTeamContext(ctx, teamID)`, `SetOpenframeTeamID(teamID)`.

## Identity: the team pin and the UUID→team bridge

`OpenframeTeamID(ctx)` is the single source of the current tenant team, in precedence order:
1. **ctx value** — per-request pin (shared-mode middleware, agent pins, tests);
2. **process pin** — team resolved from `FLEET_OPENFRAME_TENANT_UUID` at startup;
3. **env fallback** — `FLEET_OPENFRAME_TEAM_ID` (inert unless the flag is on).
Returns `ok=false` when none yields a non-zero team; callers must then assume no tenant scope
(fences no-op).

**`EnsureOpenframeTeamID(ctx, tenantUUID)`** (`server/datastore/mysql/openframe.go`) bridges the
platform's UUID tenant identity to Fleet's integer `team_id`: it resolves-or-creates the team keyed
by `teams.openframe_tenant_uuid` (unique; race-safe via insert-lose-reselect). A **newly created
team is seeded, in the same transaction, with one random team-scoped enroll secret** (same default
as EE team creation) — without it a fresh tenant could never enroll an agent (the pinned
`GET /spec/enroll_secret` would be empty). The resolve path never touches an existing team's
secrets. In pinned mode `serve.go` calls this at startup and `SetOpenframeTeamID`s the result.

## Schema migrations (`server/datastore/mysql/migrations/openframe/`)

Applied by `fleet prepare db`. Idempotent (`columnExists`/`indexExists` guards).

| Migration | Change | Why |
|---|---|---|
| `20260629000001_AddTeamsOpenframeTenantUUID` | `teams.openframe_tenant_uuid CHAR(36)` + unique key | the UUID→team bridge |
| `20260626000001_ScopeHostIdentityUniqueToTeam` | `hosts` virtual col `openframe_team_key = IFNULL(team_id,0)` + `UNIQUE(osquery_host_id, openframe_team_key)`, drop global `UNIQUE(osquery_host_id)` | host identity unique **per team** — the same device can exist in two tenants |
| `20260620000001_ScopeLabelUniqueNameToTeam` | same generated-column pattern for `labels.name` | label names unique per team; built-ins stay global |
| `20260722000001_AddTeamIdToCdcTables` | nullable `team_id` on `activity_past`, `activity_host_past`, `query_results`, `policy_membership` (no index, no FK) | the Debezium CDC tables must be self-describing on a shared DB — see "CDC team stamping" below |

The `IFNULL(team_id,0)` sentinel collapses all NULL-team rows onto key `0` (team ids start at 1), so
**flag-off / pre-backfill the uniqueness is bit-for-bit the old global uniqueness**, and the
`labels` ODKU upsert (`ApplyLabelSpecs`) keeps working. `fleet.Label` carries an ignored
`OpenframeTeamKey` field so `SELECT l.*` scans don't break.

## Datastore fences (the tenant boundary)

The pattern everywhere: `if teamID, ok := fleet.OpenframeTeamID(ctx); ok { … AND team_id = ? … }`
— a **no-op when unpinned**. On the shared DB this is the real isolation (the Fleet team was only a
role-authz grouping upstream; here it is a hard boundary regardless of token role). Coverage:

- **hosts** — `ListHosts`/`CountHosts` (in `applyHostFilters`), by-id `Host`/`HostLite`, bulk delete
  (`filterHostIDsByTeam`), enrollment matcher (`matchHostDuringEnrollment`), `AddHostsToTeam`, and
  the minor getters `HostLiteByIdentifier`/`HostLiteByID`, `ListHostsLiteByIDs`, `HostIDsByIdentifier`.
  Deliberately **unfenced**: `HostByUUID` (pre-auth iDevice identity lookup — no pin yet).
- **policies** — list/count/by-id (`Policy`, `PolicyLite`, `PoliciesByID`), create→pinned, save
  (verify-on-primary), delete (filter foreign ids); service-layer `DeleteGlobalPolicies` and
  `modifyPolicy` (the global modify endpoint) treat an own-pinned-team policy as global — creation
  re-homes "global" policies to the pinned team, so the tenant UI's global endpoints must accept
  them back.
- **queries** — list/by-id/name, create→pinned, save-verify, delete, `ApplyQueries` re-home.
- **enroll_secrets** (`app_configs.go`) — `GetEnrollSecrets`/`ApplyEnrollSecrets` force `teamID =
  pinned`; `VerifyEnrollSecret` only accepts a secret whose `team_id = pinned` (agent boundary).
- **live-query targets** (`targets.go`) — `HostIDsInTargets`/`CountHostsInTargets` scoped.
- **live-query campaigns** (`campaigns.go`) — `DistributedQueryCampaign`/
  `DistributedQueryCampaignTargetIDs` fenced via the campaign's query team (`EXISTS` against
  `queries.team_id`; pinned creation re-homes ad-hoc query rows, so every campaign's query carries
  its tenant's team). Upstream's only guard is `campaign.UserID`, which is useless on the shared
  Fleet where every tenant operates as the same Admin user.
- **host-assignments** (`policy_hosts`/`query_hosts`) — parent verified in team + foreign host ids
  dropped (pre-existing `OPENFRAME(host-assignments)` feature, extended here).
- **teams** — `TeamLite`/`ListTeams` read fence.
- **app_config** cache (`cached_mysql.go`) — cache key includes the pinned team so one tenant's
  config is never served to another; unpinned keeps the constant key.
- **app_config** reads (`app_configs.go`) — tenant rows at `id = team id`; `id = 1` is the
  *instance* row, read by every unpinned path (crons, workers, boot). Unpinned multitenant reads
  select `WHERE id = 1` instead of upstream's bare `LIMIT 1`, which returns an arbitrary row. 
  A missing row falls back to `OpenframeDefaultAppConfig()` for any multitenant read (nothing else 
  seeds `id = 1`; a zero-value config would disable software inventory and historical data instance-wide).
  Non-multitenant keeps the upstream statement and fallback byte-identical. Migration
  `20260831000001` seeds/repairs the row and reserves team id 1.
- **software titles + vulnerabilities** (`software_titles.go`, `vulnerabilities.go`) —
  `ListSoftwareTitles`, `SoftwareTitleByID`, `ListVulnerabilities`, `CountVulnerabilities`,
  `Vulnerability`, `SoftwareByCVE`, `OSVersionsByCVE` replace the caller's team scope with the
  pinned team (`openframeTeamScope`), so titles, versions, host counts and CVEs come from the
  per-team rows of `software_titles_host_counts` / `software_host_counts` /
  `vulnerability_host_counts` only. Applied in the datastore, not via the `team_id` param, because
  the service gates `team_id` on titles behind a Premium license. The OpenFrame api reads these
  server-side for Software Management, so the gateway allowlist does not protect them.
  `GET /vulnerabilities`: a pinned empty page reports `counts_updated_at` as the instance's last
  host-count recalculation (`VulnerabilityHostCountsUpdatedAt`, zero if never) instead of `now()`.
  `SoftwareByID` stays unfenced on purpose — the Office vulnerability analyzers call it.
- **host list software filters** (`server/service/hosts.go`) — with `software_version_id` /
  `software_id` / `software_title_id`, a pinned request resolves the `software` / `software_title`
  block within its team and skips the unscoped name lookups (`SoftwareLiteByID`,
  `SoftwareTitleNameForHostFilter`), so another tenant's software is never described.

## Per-request pinning

- **Control-plane / UI** — `service.WithOpenframeTenant` (`server/service/openframe_middleware.go`),
  wired in `serve.go` after `apiendpoints.Validate`. Shared mode only (returns `next` unchanged
  otherwise). Pins each `/api/**` request from the gateway-injected trusted `X-Tenant-Id`; a
  non-exempt request without it is **401 (fail closed)**. Exempt (tenant comes from host/secret):
  paths containing `/osquery/`, `/fleet/orbit/`, `/fleet/device/`, `/mdm/`, `/fleet/ota_enrollment`;
  and the session bootstrap `/api/{v}/fleet/{login,logout,sessions,sso,sso/callback}` (anchored —
  the admin `/users/{id}/sessions` still needs a tenant), which carry no header and cannot: they
  are how a caller obtains the token it is afterwards recognised by. Without them a shared-mode
  Fleet cannot be logged into at all.
- **Agent plane** — `openframePinHostTeam` pins from the authenticated `host.team_id` in
  `authenticatedHost`/`authenticatedOrbitHost`/`authenticatedDevice` (`endpoint_middleware.go`) and
  the osquery header pre-auth paths (`osquery_header_auth.go`); fail-closed on a team-less host.
- **Enrollment** — after `VerifyEnrollSecret`, `osquery.go`/`orbit.go` pin from `secret.TeamID`
  (reject if the secret has no team). The host row is then created carrying that `team_id`.
- **Live-query results websocket** (`endpoint_campaigns.go`) — the sockjs handler rebuilds its
  context from `context.Background()`, discarding the middleware-pinned upgrade-request context,
  so it **re-pins from `session.Request().Context()`** (read once — polling transports mutate the
  session request). Fail closed: in shared mode an unpinned session is rejected unless its viewer
  is a superuser. Without the re-pin the whole campaign stream ran unfenced and the `live_query`
  activity was stamped `team_id NULL`.

## Superuser (shared mode only)

`FLEET_OPENFRAME_SUPERUSER_EMAILS` (comma-separated, case-insensitive; chart value
`fleet.openframe.multiTenancy.superuserEmails`) names Fleet users that may run a shared-mode request
**unpinned** — every tenant fence inert — which is what reaching all tenants' devices from one live
query requires. Empty is the default and means no such caller exists, so shared mode stays fail
closed; naming one is an explicit per-deployment decision. Use a dedicated account, never
`admin@openframe.local`: every tenant operates as that user and the gateway holds its token.

Consulted on the two gates that would otherwise fail closed — the tenant middleware and the campaign
stream — and only for a caller already holding a **valid Fleet session**. It widens what an
authenticated operator may reach, never who may authenticate: no session ⇒ 401 as before, and a
gateway request that lost its header still 401s, since it carries no superuser session.

The session key is read from `Authorization: Bearer`, falling back to the UI's session cookie
(`__Host-token` over HTTPS, `token` otherwise). The fallback is not a convenience: a browser can
attach **neither** `X-Tenant-Id` nor `Authorization` to a WebSocket upgrade, so on the live-query
result stream — the only transport Fleet's SockJS endpoint serves, as `xhr_streaming`/`xhr_send` are
registered GET-only and answer 405 — the cookie is the only identity the request can carry.

Reached only when the gateway header is absent, so normal tenant traffic never pays for the session
lookup. Unpinned reaches the read-views listed under *Known deferred* across all tenants.

Agents send **no tenant header** — tenant identity flows in via the enroll secret and thereafter via
the host record (node key → host → team). This is by design and stronger than a header.

## Migration serialization on a shared DB

`fleet prepare db` wraps the migration sequence in a MySQL named lock
`GET_LOCK('openframe_fleet_migrations', 900s)` (`AcquireOpenframeMigrationLock` in
`server/datastore/mysql/openframe.go`, called from `cmd/fleet/prepare.go`) — **flag-gated**. With N
clusters' migration Jobs pointed at one DB, the first wins and migrates; the rest block, then find
the schema already applied and no-op. Session-scoped (auto-released if the Job dies). Flag-off runs
are untouched.

## CDC team stamping (Debezium on the shared DB)

The OpenFrame Debezium pipeline captures `activity_past`, `activity_host_past`, `query_results`
and `policy_membership`. On a shared DB one connector serves every tenant, and its SMTs are
stateless — a CDC record must therefore carry its own tenant discriminator. The fork stamps the
`team_id` column (added by `20260722000001`) at write time:

| Table | Stamp source | Where |
|---|---|---|
| `query_results` | request team pin (`fleet.OpenframeTeamID(ctx)` — the agent plane is always pinned in shared mode) | `OverwriteQueryResultRows` |
| `policy_membership` (sync) | request team pin | `RecordPolicyQueryExecutions` |
| `policy_membership` (async collector) | the row's own host, `(SELECT team_id FROM hosts WHERE id = ?)` — the cron context is never pinned | `AsyncBatchInsertPolicyMembership` |
| `activity_past` | request team pin — the only tenant signal for host-less (user/team-level) activities | `server/activity/internal/mysql/new_activity.go` |
| `activity_host_past` | the row's own host via subselect (host activities can come from unpinned crons; the host is authoritative anyway) | same |

Unpinned writes (flag off, or background crons writing host-less activities) keep the original
statement **byte-identical** and leave `team_id` NULL; the downstream consumer drops NULL-team
events (fail closed). The `teams` table (with the `openframe_tenant_uuid` bridge) is also added
to the connector's capture list platform-side, so consumers can resolve `team_id` → tenant UUID
without touching Fleet's API. Platform side of this pipeline: shared connector registration in
`openframe-saas-shared`, and shared-plane consumption in that repo's `openframe-saas-stream`
(the MeshCentral pattern — per-event tenant resolution, gated by
`openframe.fleet.multi-tenancy.enabled`).

## Helm / config wiring (`charts/fleet/`)

`values.yaml` adds `fleet.openframe.multiTenancy` (`enabled: false` default; `tenantUuid` /
`existingConfigMap`+`tenantUuidKey` / `teamId`). `deployment.yaml` injects
`FLEET_OPENFRAME_MULTI_TENANCY_ENABLED` (+ optional `FLEET_OPENFRAME_TENANT_UUID` /
`FLEET_OPENFRAME_TEAM_ID`); `job-migration.yaml` gets the flag too (so the `GET_LOCK` guard engages).

## Backward compatibility

Flag off ⇒ pre-feature fork behavior everywhere **except the schema**: the three migrations run
unconditionally at `prepare db`, but are semantics-preserving — all rows are `team_id = NULL` →
generated key `0` → the original global uniqueness, byte-for-byte. Verified locally (fresh
branch-native DB): `prepare db` applies all three, the server boots healthy flag-off,
flag-on-pinned (team auto-created + secret seeded, `team_id=1`), and flag-on-shared. The upstream
`TestLabels`/`TestHosts` suites pass on the migrated schema with no OpenFrame env.

## Tests

MySQL-backed (`MYSQL_TEST=1`), all in `*_openframe_test.go`: enrollment isolation, host-identity
per-team, host by-id/list/identifier fences, policy/query CRUD + by-id + GitOps, enroll-secret
fence, host-assignment fence, live-query target fence, teams read fence, app-config isolation,
`EnsureOpenframeTeamID` (incl. secret seeding), delete-global-policies pin, migration pipeline,
software-title / vulnerability / CVE-detail fences.
Flag-parsing / mode-precedence unit tests in `server/fleet/openframe_test.go`. Middleware tests in
`server/service/openframe_middleware_test.go`. Harness: `make openframe-verify` (add `MYSQL_TEST=1`
+ Docker for the deep tier).

## Known deferred (latent, non-applicable to OpenFrame today)

- **Users/sessions, software versions (`/software/versions`), os_versions and activities
  read-views** — unfenced; only matters if the Fleet UI is exposed beyond the gateway allowlist and
  the OpenFrame api does not read them. Not today — except for a configured superuser, which is
  unpinned by definition and therefore sees these across every tenant. Fence them before naming a
  superuser on an environment holding customer data.
- **Custom-label creation while unpinned** → `team_id NULL` → distributed globally. OpenFrame seeds
  only built-ins and targets via host-assignments; pinned label writes are team-scoped.
- **Legacy 2017 "user packs" scheduled-query-stats join** — resolves by global pack name; packs are
  unused. New-format stats are team-scoped.
- **`IsEnrollSecretAvailable`** — intentionally unfenced (cross-team uniqueness check).

## Deployment / cutover

Roll out in order: (1) deploy the fenced code (inert until pinned); (2) run migrations
(`prepare db`); (3) **backfill each tenant's existing rows' `team_id` before flipping its flag** —
un-backfilled rows fail closed (hide data, never leak); (4) enable the flag per tenant. Greenfield
(empty shared DB, agents re-enroll) skips the backfill at the cost of host history.
