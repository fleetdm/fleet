// Package redis_config_etag implements the Redis-backed store for osquery
// config ETags — the persistence half of the config "SHORT CIRCUIT" that lets
// Fleet answer a config check-in as "unchanged" without building the
// config (and therefore without any database reads).
//
// # Record families and cache modes
//
// Two record families exist, selected per request by cache mode (see
// fleet.ConfigETagStore and Service.GetClientConfigWithETag):
//
//   - SHARED records, one per (scope, platform): used when the host's
//     effective report scope (global ∪ team) has NO label-scoped reports, so
//     the rendered config is identical for every host in the scope.
//   - PER-HOST records, one per host: used when label-scoped reports make the
//     rendered config host-specific. A per-host record is never read by
//     another host, so cross-host isolation is structural — no cohort
//     signatures, membership generations, or CAS validation are needed.
//
// Deployments with user-created 2017 packs bypass the short circuit entirely
// (pack targeting drifts with label membership and pack content is
// host-specific in unbounded ways).
//
// # Why a plain "delete the ETag on change" scheme is NOT correct
//
// Fleet builds the osquery config through two stacked PER-INSTANCE in-memory
// caches:
//
//  1. cached_mysql (server/datastore/cached_mysql): AppConfig (1s TTL),
//     ListPacksForHost (1m), ListScheduledQueriesInPack (1m),
//     TeamAgentOptions (1m), ...
//  2. svc.packConfigCache (server/service/service.go): the marshaled pack
//     JSON keyed by (teamID, queryReportsDisabled), 1m TTL, filled FROM
//     cached_mysql reads — so worst-case staleness is additive (~2 minutes).
//
// The poison sequence a naive implementation allows:
//
//  1. Instance A fills its in-memory caches at T0.
//  2. At T1 an admin changes a schedule → the event deletes the Redis ETag.
//  3. At T1+ε a host hits instance A. Redis has no ETag, so A builds the
//     config FROM ITS STILL-STALE IN-MEMORY CACHES, computes the stale ETag,
//     and writes it to Redis.
//  4. The in-memory caches expire a minute later, but Redis now holds the
//     stale ETag indefinitely. Every host presenting it is told "unchanged" forever and
//     never receives the new config.
//
// The staleness originates at cache-FILL time, before the request arrives, so
// compare-and-set against a version read at request time cannot fix it — a
// stale build happily stamps itself with the new version.
//
// # The write-fence design (deployment-wide config/report mutations)
//
// Two cooperating mechanisms with distinct jobs:
//
//   - Generation counter (read-side invalidation): every config-affecting
//     datastore write INCRs it (see server/datastore/etag_invalidate). Stored
//     ETags — shared AND per-host — carry the generation current when
//     written; the read path treats a mismatch as a miss. Kills existing
//     ETags instantly, in O(1), no matter how many records exist.
//
//   - Write fence (write-side quarantine): the same write arms a fence key
//     whose TTL outlives the composed in-memory cache staleness (see
//     DefaultFenceTTL). While the fence is armed, publication refuses to
//     persist ETags. Once it expires, every in-memory cache entry that
//     predates the mutation has necessarily expired too, so any build that
//     completes afterward is fresh and safe to persist.
//
// Correctness argument: Invalidate (INCR gen + SET fence) is one atomic EVAL;
// publication ("if no fence, SET record stamped with current gen") is
// another. Redis serializes them, so for a build whose inputs were fetched
// before a mutation: its write lands BEFORE the invalidation → the gen bump
// makes the record stale for reads; or AFTER → the fence suppresses it. There
// is no interleaving in which a stale ETag survives with a current
// generation.
//
// # The per-host publish quarantine (label-membership invalidation)
//
// Per-host records are invalidated CONSERVATIVELY: every successful
// persistence of a host's label results deletes its record (no value diff).
// Ordering the DEL after the DB commit is necessary but not sufficient — a
// build that STARTED before the membership commit, or read a lagging MySQL
// replica after it, can publish a pre-change ETag AFTER the DEL. The
// deployment fence cannot gate this (membership persistence is continuous at
// fleet scale; arming a global fence would keep it armed forever), so each
// InvalidateHost also arms a PER-HOST publish quarantine
// (DefaultHostQuarantineTTL) that SetHostIfNoFence honors. The DEL and the
// quarantine SET are ONE Lua script: a pipeline is not atomic, and a publish
// interleaved between them would persist the stale record before the
// quarantine arms.
//
// The quarantine TTL is DERIVED for its staleness source — replica lag plus
// in-flight request duration plus slack — and is deliberately NOT the
// deployment fence's 180s (which is derived from in-memory cache TTLs that do
// not apply to membership reads). Residual risk beyond the quarantine
// (replication incidents with lag > 30s) is bounded by the next label cycle's
// conservative invalidation and, failing that, the record's jittered backstop
// TTL. This bounded-staleness contract is explicit and accepted.
//
// # Gate state (cache-mode selection)
//
// Two small cached answers drive mode selection, each with a bounded TTL and
// reset hooks (see server/datastore/etag_invalidate):
//
//   - legacy-packs flag: deployment-wide hard bypass; reset on pack CRUD.
//   - label scopes: which scopes have label-scoped reports (shared vs
//     per-host mode); reset on query CRUD and label deletion.
//
// The loaders are cheap deployment-level queries, but a Redis miss must not
// become one database query per concurrent config request — and the miss window
// widens exactly when the database is degraded. Each gate therefore uses
// NON-BLOCKING leader election (one CAS flag per gate per Fleet instance): the
// winner runs the loader, and losers return fleet.ErrConfigETagGateLoading
// immediately rather than waiting or querying. Callers treat that as "state
// unknown" and bypass for the request. A blocking singleflight is deliberately
// avoided: it would couple config-request latency to a possibly-hung query.
// TestGateLoaderLeaderElection pins one loader per miss window.
//
// # Key layout
//
// Seven keys, all sharing the {cfgetag} hash tag (see Redis Cluster below).
// Record values are "<gen>|<etag>" for shared records and
// "<gen>|<scope>|<platform>|<etag>" for per-host ones, so a read revalidates
// the generation — and, per host, the scope and platform — before trusting the
// stored validator.
//
//	{cfgetag}:etag:v<ver>:<scope>:<platform>   shared record   TTL 1h
//	{cfgetag}:host-etag:v<ver>:<hostID>        per-host record TTL 50-70m jittered
//	{cfgetag}:host-quarantine:<hostID>         per-host publish quarantine, TTL 30s
//	{cfgetag}:gen                              generation counter, no TTL
//	{cfgetag}:fence                            deployment write fence, TTL 3m
//	{cfgetag}:legacy-packs                     cached gate answer, TTL 5m
//	{cfgetag}:scope-modes                      cached gate answer, TTL 5m
//
// Only the two record families carry the Fleet version, and the split is not
// cosmetic — it follows from whether the key's VALUE is version-dependent or
// its MEANING is version-independent:
//
//   - Records hold a hash of the rendered config body. If a release changes
//     that body at all, the same logical config hashes differently, so sharing
//     one key across versions would make two instances overwrite each other
//     for the length of every rolling deploy. Versioning isolates them.
//   - The coordination keys must act ACROSS versions, so versioning them would
//     be a bug. If gen were versioned, a config write handled by a v4.91
//     instance would bump gen-v4.91 and leave every v4.92 record still
//     validating — precisely the stale-read poisoning this design exists to
//     prevent. The same argument applies to the fence (a write on any version
//     must suppress publication on all of them) and to the quarantine (a label
//     invalidation must quarantine publishes regardless of which version
//     handled it).
//   - The two gate keys cache database facts, which are version-independent.
//     Sharing them costs one DB load per cluster rather than per version. The
//     one hazard, remote enough to accept: a release that changed what counts
//     as a label-scoped scheduled report would read an older version's cached
//     answer as its own.
//
// # Redis Cluster
//
// Every key in this package embeds the {cfgetag} hash tag so they all land in
// the same slot, which is required for the multi-key MGET and EVAL calls. The
// ops are single tiny GET/SET/INCR/DELs, so the single-slot hot spot is
// negligible at realistic check-in rates.
//
// # Fleet version namespacing
//
// The motivating case for versioning record keys (see Key layout above) is that
// a server upgrade can change the rendered config body with ZERO datastore
// writes, so no invalidation hook would ever fire. Redis state outlives the
// deploy even though Fleet instances scale to zero, so without namespacing the
// new version would read the old version's validators as current. Orphaned
// old-version keys are garbage-collected by their TTLs.
//
// # Observability
//
// Structured logs are the PRIMARY mechanism (bounded state-write logs here;
// per-request debug fields at the endpoint). Prometheus counters are ALSO
// registered — they cost nanoseconds and a few dozen time series, appear on
// the existing /metrics endpoint, and NOTHING depends on them (Fleet's
// /metrics has no known production consumers; if it is retired these are
// deleted with no design impact).
//
// Cardinality rule: no per-host or per-team labels on any counter. Every
// question this design asks is an aggregate.
package redis_config_etag
