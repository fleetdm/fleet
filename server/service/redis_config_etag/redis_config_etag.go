// Package redis_config_etag implements the Redis-backed store for osquery
// config ETags — the persistence half of the config "SHORT CIRCUIT" that lets
// Fleet answer a config check-in with 304 Not Modified without building the
// config (and therefore without any database reads).
//
// ██ READ THIS BEFORE CHANGING ANYTHING IN THIS PACKAGE ██████████████████████
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
//  1. cached_mysql (server/datastore/mysql/cached_mysql): AppConfig (1s TTL),
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
//     stale ETag indefinitely. Every host presenting it gets 304 forever and
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
// Loaders are trivially cheap deployment-level queries, so concurrent
// duplicate loads after expiry are harmless (no stampede protection by
// design — a lock would add a worse failure mode than the ~handful of
// duplicate 1ms queries it prevents).
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
// ETag record keys (shared and per-host) embed the Fleet server version. A
// server upgrade can change the rendered config body with ZERO datastore
// writes — no invalidation hook would ever fire. Namespacing by version makes
// every upgrade start a clean keyspace; per-record TTLs garbage-collect the
// orphaned old-version keys. (Fleet deploys scale to zero, but Redis state
// persists across the deploy — that is what this protects against.) The
// quarantine key is deliberately NOT version-namespaced: a membership change
// invalidates the host regardless of server version.
//
// # Observability
//
// Structured logs are the PRIMARY mechanism (bounded state-write logs here;
// per-request debug fields at the endpoint). Prometheus counters are ALSO
// registered — they cost nanoseconds and a few dozen time series, appear on
// the existing /metrics endpoint, and NOTHING depends on them (Fleet's
// /metrics has no known production consumers; if it is retired these are
// deleted with no design impact). ██ CARDINALITY RULE ██ no per-host or
// per-team labels on any counter: every question this design asks is an
// aggregate.
//
// ████████████████████████████████████████████████████████████████████████████
package redis_config_etag

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/version"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// DefaultFenceTTL is how long ETag publication stays suppressed after a
	// config-affecting datastore write.
	//
	// ██ DO NOT LOWER THIS WITHOUT RE-DERIVING IT ██
	//
	// It must be strictly greater than the composed worst-case staleness of
	// the per-instance in-memory caches that feed the config build:
	//
	//	  packConfigCache TTL (1m, server/service/service.go)
	//	+ max cached_mysql TTL used by the build (1m, defaultPacksExpiration
	//	  et al. in server/datastore/mysql/cached_mysql)
	//	+ slack for in-flight requests and clock slop (1m)
	//	= 3m
	//
	// If either cache TTL is ever raised, this value MUST grow with it, or
	// the stale-write poisoning described in the package docs comes back.
	DefaultFenceTTL = 3 * time.Minute

	// DefaultETagTTL is the backstop TTL on SHARED ETag records. The
	// generation/fence mechanism is the correctness mechanism and would
	// suffice alone if invalidation-hook coverage (see
	// server/datastore/etag_invalidate) were provably complete forever. The
	// TTL converts any MISSED hook — or an out-of-band DB edit, a restore
	// from backup, a future mutation added without a hook — from a
	// permanently poisoned ETag into a bounded delay, at the cost of one full
	// config build per (team, platform) per TTL. It may be lengthened as
	// confidence grows. NEVER remove it.
	DefaultETagTTL = 1 * time.Hour

	// Host record TTLs are JITTERED between these bounds (~50-70m, centered
	// on one hour) so the backstop never expires a whole fleet's records at
	// once. Long, because correctness comes from invalidation rather than
	// expiry; present at all so a lost or failed invalidation is bounded
	// rather than permanent.
	DefaultHostETagTTLMin = 50 * time.Minute
	DefaultHostETagTTLMax = 70 * time.Minute

	// DefaultHostQuarantineTTL is the per-host publish quarantine armed by
	// InvalidateHost. Derived for ITS staleness source — replica lag plus
	// in-flight request duration plus slack — NOT copied from the deployment
	// fence. At typical config-refresh intervals (60s) it suppresses ~0-1
	// publish per invalidation; suppression scales as quarantine/interval at
	// shorter intervals and remains a rounding error against the
	// un-short-circuited baseline.
	DefaultHostQuarantineTTL = 30 * time.Second

	// gateStateTTL bounds the cached gate answers (legacy packs flag, label
	// scopes). On expiry the next reader recomputes them with cheap DB
	// queries; the TTL also bounds any failure of the reset hooks.
	gateStateTTL = 5 * time.Minute
)

// All keys share the {cfgetag} hash tag → same Redis Cluster slot, so the
// multi-key MGET/EVAL calls below are legal in cluster mode.
const (
	genKey        = "{cfgetag}:gen"
	fenceKey      = "{cfgetag}:fence"
	legacyKey     = "{cfgetag}:legacy-packs"
	scopeModesKey = "{cfgetag}:scope-modes"
	// shared etag record keys append ":v<fleet version>:<scope>:<platform>",
	// see (*Store).etagKey.
	etagKeyPrefix = "{cfgetag}:etag"
	// per-host etag record keys append ":v<fleet version>:<hostID>", see
	// (*Store).hostETagKey.
	hostETagKeyPrefix = "{cfgetag}:host-etag"
	// per-host publish quarantine keys append ":<hostID>" (NOT
	// version-namespaced), see (*Store).hostQuarantineKey.
	hostQuarantineKeyPrefix = "{cfgetag}:host-quarantine"
)

// Prometheus counters — secondary, dependency-free observability (see the
// package docs). Registered once at package load on the default registry and
// exposed via Fleet's existing /metrics endpoint.
var (
	metricRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "fleet",
		Subsystem: "config_etag",
		Name:      "requests_total",
		Help:      "Config ETag store read attempts by cache mode and result.",
	}, []string{"mode", "result"})
	metricPublish = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "fleet",
		Subsystem: "config_etag",
		Name:      "publish_total",
		Help:      "Config ETag publication attempts by result.",
	}, []string{"result"})
	metricInvalidations = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "fleet",
		Subsystem: "config_etag",
		Name:      "invalidations_total",
		Help:      "Config ETag invalidations by kind and result.",
	}, []string{"kind", "result"})
	metricGateLoads = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "fleet",
		Subsystem: "config_etag",
		Name:      "gate_loads_total",
		Help:      "Executions of the gate-state loaders (legacy packs, label scopes).",
	})
)

// Store implements fleet.ConfigETagStore on Redis. See the package docs for
// the correctness model; see fleet.ConfigETagStore for the method contracts.
type Store struct {
	pool          fleet.RedisPool
	logger        *slog.Logger
	fenceTTL      time.Duration
	etagTTL       time.Duration
	hostETagMin   time.Duration
	hostETagMax   time.Duration
	quarantineTTL time.Duration
	version       string // Fleet server version baked into ETag record keys
	testPrefix    string // for tests, the key prefix to use to avoid conflicts
}

var _ fleet.ConfigETagStore = (*Store)(nil)

// New returns a Store using the default TTLs and the running server's
// version for key namespacing. The logger is used only for bounded
// state-write logs and must not be nil.
func New(pool fleet.RedisPool, logger *slog.Logger) *Store {
	return &Store{
		pool:          pool,
		logger:        logger,
		fenceTTL:      DefaultFenceTTL,
		etagTTL:       DefaultETagTTL,
		hostETagMin:   DefaultHostETagTTLMin,
		hostETagMax:   DefaultHostETagTTLMax,
		quarantineTTL: DefaultHostQuarantineTTL,
		version:       version.Version().Version,
	}
}

func (s *Store) etagKey(scope, platform string) string {
	return fmt.Sprintf("%s%s:v%s:%s:%s", s.testPrefix, etagKeyPrefix, s.version, scope, platform)
}

func (s *Store) hostETagKey(hostID uint) string {
	return fmt.Sprintf("%s%s:v%s:%d", s.testPrefix, hostETagKeyPrefix, s.version, hostID)
}

func (s *Store) hostQuarantineKey(hostID uint) string {
	return fmt.Sprintf("%s%s:%d", s.testPrefix, hostQuarantineKeyPrefix, hostID)
}

// hostETagTTLSeconds returns a jittered TTL in [hostETagMin, hostETagMax] so
// the backstop never expires a whole fleet's records at once.
func (s *Store) hostETagTTLSeconds() int {
	minSec := int(s.hostETagMin.Seconds())
	maxSec := int(s.hostETagMax.Seconds())
	if maxSec <= minSec {
		return minSec
	}
	return minSec + rand.IntN(maxSec-minSec+1) //nolint:gosec // TTL jitter doesn't need cryptographic randomness
}

////////////////////////////////////////////////////////////////////////////////
// Shared records
////////////////////////////////////////////////////////////////////////////////

// GetValid returns the stored SHARED ETag for (scope, platform) only when its
// recorded generation matches the current generation. Any malformed or
// generation-stale record is reported as a miss (ok=false) — the caller then
// performs a normal full config build, so there is no failure mode here that
// can serve wrong data.
func (s *Store) GetValid(ctx context.Context, scope, platform string) (etag string, ok bool, err error) {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	// Single MGET; both keys share the {cfgetag} slot.
	vals, err := redigo.Values(conn.Do("MGET", s.etagKey(scope, platform), s.testPrefix+genKey))
	if err != nil {
		metricRequests.WithLabelValues("shared", "error").Inc()
		return "", false, ctxerr.Wrap(ctx, err, "redis config etag get")
	}
	record, currentGen, err := parseRecordAndGen(ctx, vals)
	if err != nil {
		metricRequests.WithLabelValues("shared", "error").Inc()
		return "", false, err
	}
	if record == "" {
		metricRequests.WithLabelValues("shared", "miss").Inc()
		return "", false, nil
	}

	// The record is "<generation>|<etag>", written by SetIfNoFence.
	recordGen, recordETag, found := strings.Cut(record, "|")
	if !found || recordETag == "" {
		// malformed record: treat as a miss, never as a match
		metricRequests.WithLabelValues("shared", "miss").Inc()
		return "", false, nil
	}
	if recordGen != currentGen {
		// A config-affecting write happened after this record was stored.
		metricRequests.WithLabelValues("shared", "stale_gen").Inc()
		return "", false, nil
	}
	metricRequests.WithLabelValues("shared", "hit").Inc()
	return recordETag, true, nil
}

// setIfNoFenceScript persists a SHARED ETag record stamped with the current
// generation, unless the write fence is armed.
//
// ██ ATOMICITY IS LOAD-BEARING ██ This check-and-set MUST be a single atomic
// script. If the fence check and the SET were separate round trips, an
// invalidation could land between them and a stale ETag would be persisted
// with a current generation — exactly the poisoning this package exists to
// prevent. See the package docs for the full interleaving argument.
//
// KEYS[1] = fence key, KEYS[2] = gen key, KEYS[3] = etag record key
// ARGV[1] = etag, ARGV[2] = record TTL in seconds
const setIfNoFenceScript = `
if redis.call('EXISTS', KEYS[1]) == 1 then
  return 0
end
local gen = redis.call('GET', KEYS[2])
if not gen then
  gen = '0'
end
redis.call('SET', KEYS[3], gen .. '|' .. ARGV[1], 'EX', ARGV[2])
return 1
`

// SetIfNoFence persists the SHARED ETag for (scope, platform) with the
// current generation and the backstop TTL — unless the write fence is armed
// (a config-affecting write happened within the last fenceTTL), in which case
// nothing is stored and stored=false is returned. Callers must treat a
// suppressed write as normal operation, not an error.
func (s *Store) SetIfNoFence(ctx context.Context, scope, platform, etag string) (stored bool, err error) {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	res, err := redigo.Int64(conn.Do("EVAL", setIfNoFenceScript, 3,
		s.testPrefix+fenceKey, s.testPrefix+genKey, s.etagKey(scope, platform),
		etag, int(s.etagTTL.Seconds())))
	if err != nil {
		metricPublish.WithLabelValues("error").Inc()
		return false, ctxerr.Wrap(ctx, err, "redis config etag set")
	}
	if res != 1 {
		metricPublish.WithLabelValues("fence_suppressed").Inc()
		return false, nil
	}
	metricPublish.WithLabelValues("stored").Inc()
	return true, nil
}

// invalidateScript bumps the deployment generation (read-side invalidation of
// every stored ETag, shared and per-host) and arms the write fence,
// atomically. Doing both in one script is what makes the interleaving
// argument in the package docs hold.
//
// KEYS[1] = gen key, KEYS[2] = fence key
// ARGV[1] = fence TTL in seconds
const invalidateScript = `
redis.call('INCR', KEYS[1])
redis.call('SET', KEYS[2], '1', 'EX', ARGV[1])
return 1
`

// Invalidate must be called after every successful config-affecting datastore
// write (see server/datastore/etag_invalidate for the hook points). It is
// deliberately coarse — one deployment-wide generation — because
// over-invalidating only costs the optimization for the fence window, never
// correctness.
func (s *Store) Invalidate(ctx context.Context) error {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("EVAL", invalidateScript, 2,
		s.testPrefix+genKey, s.testPrefix+fenceKey,
		int(s.fenceTTL.Seconds())); err != nil {
		metricInvalidations.WithLabelValues("deployment", "error").Inc()
		return ctxerr.Wrap(ctx, err, "redis config etag invalidate")
	}
	metricInvalidations.WithLabelValues("deployment", "ok").Inc()
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Per-host records
////////////////////////////////////////////////////////////////////////////////

// GetValidHost returns the stored PER-HOST ETag only when its recorded
// generation matches the current generation AND its recorded scope and
// platform match the authenticated host context. A team transfer or platform
// change therefore reads as a miss with no key cleanup required. Malformed or
// stale records are misses — never matches.
func (s *Store) GetValidHost(ctx context.Context, hostID uint, scope, platform string) (etag string, ok bool, err error) {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	vals, err := redigo.Values(conn.Do("MGET", s.hostETagKey(hostID), s.testPrefix+genKey))
	if err != nil {
		metricRequests.WithLabelValues("host", "error").Inc()
		return "", false, ctxerr.Wrap(ctx, err, "redis config etag host get")
	}
	record, currentGen, err := parseRecordAndGen(ctx, vals)
	if err != nil {
		metricRequests.WithLabelValues("host", "error").Inc()
		return "", false, err
	}
	if record == "" {
		metricRequests.WithLabelValues("host", "miss").Inc()
		return "", false, nil
	}

	// The record is "<generation>|<scope>|<platform>|<etag>", written by
	// SetHostIfNoFence. The generation is numeric, the scope is "global" or
	// "team:<id>", and the ETag is a quoted SHA-256 hex string, so none of
	// the server-controlled fields can contain the separator. Platform is
	// HOST-REPORTED data: if it ever contained '|', the SplitN below would
	// misalign fields — which is SAFE (the scope/platform/gen comparisons
	// then reliably mismatch, producing a miss and a full build, never a
	// false hit), but a value with '|' written by SetHostIfNoFence could
	// never validate, so such a host would simply never short-circuit.
	parts := strings.SplitN(record, "|", 4)
	if len(parts) != 4 || parts[3] == "" {
		metricRequests.WithLabelValues("host", "miss").Inc()
		return "", false, nil
	}
	if parts[0] != currentGen {
		metricRequests.WithLabelValues("host", "stale_gen").Inc()
		return "", false, nil
	}
	if parts[1] != scope || parts[2] != platform {
		// team transfer or platform change since the record was written
		metricRequests.WithLabelValues("host", "miss").Inc()
		return "", false, nil
	}
	metricRequests.WithLabelValues("host", "hit").Inc()
	return parts[3], true, nil
}

// setHostIfNoFenceScript persists a PER-HOST ETag record stamped with the
// current generation, unless the deployment write fence OR the host's publish
// quarantine is armed. Atomicity is load-bearing for the same interleaving
// reasons as the shared script; the quarantine check additionally stops a
// build that read pre-invalidation membership from republishing after the
// host's DEL.
//
// KEYS[1] = fence key, KEYS[2] = host quarantine key, KEYS[3] = gen key,
// KEYS[4] = host etag record key
// ARGV[1] = scope, ARGV[2] = platform, ARGV[3] = etag,
// ARGV[4] = record TTL in seconds
const setHostIfNoFenceScript = `
if redis.call('EXISTS', KEYS[1]) == 1 then
  return 0
end
if redis.call('EXISTS', KEYS[2]) == 1 then
  return -1
end
local gen = redis.call('GET', KEYS[3])
if not gen then
  gen = '0'
end
redis.call('SET', KEYS[4], gen .. '|' .. ARGV[1] .. '|' .. ARGV[2] .. '|' .. ARGV[3], 'EX', ARGV[4])
return 1
`

// SetHostIfNoFence persists the PER-HOST ETag with the current generation and
// a jittered backstop TTL — unless the deployment write fence or the host's
// publish quarantine is armed, in which case nothing is stored and
// stored=false is returned. Suppression is normal operation, not an error;
// the response the caller already built is unaffected.
func (s *Store) SetHostIfNoFence(ctx context.Context, hostID uint, scope, platform, etag string) (stored bool, err error) {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	res, err := redigo.Int64(conn.Do("EVAL", setHostIfNoFenceScript, 4,
		s.testPrefix+fenceKey, s.hostQuarantineKey(hostID), s.testPrefix+genKey, s.hostETagKey(hostID),
		scope, platform, etag, s.hostETagTTLSeconds()))
	if err != nil {
		metricPublish.WithLabelValues("error").Inc()
		return false, ctxerr.Wrap(ctx, err, "redis config etag host set")
	}
	switch res {
	case 1:
		metricPublish.WithLabelValues("stored").Inc()
		return true, nil
	case -1:
		metricPublish.WithLabelValues("quarantine_suppressed").Inc()
		return false, nil
	default:
		metricPublish.WithLabelValues("fence_suppressed").Inc()
		return false, nil
	}
}

// invalidateHostScript deletes the host's record and arms its publish
// quarantine, atomically.
//
// ██ ATOMICITY IS LOAD-BEARING ██ A pipeline is NOT atomic: a publish
// interleaved between the DEL and the quarantine SET would persist a stale
// record before the quarantine arms, defeating the guard.
//
// KEYS[1] = host etag record key, KEYS[2] = host quarantine key
// ARGV[1] = quarantine TTL in seconds
const invalidateHostScript = `
redis.call('DEL', KEYS[1])
redis.call('SET', KEYS[2], '1', 'EX', ARGV[1])
return 1
`

// InvalidateHost must be called after every successful persistence of the
// host's label results (conservatively — no membership value diff required)
// and from manual membership paths where the affected host IDs are known. It
// deletes the host's record and arms its publish quarantine so a straddling
// build cannot republish pre-change membership.
func (s *Store) InvalidateHost(ctx context.Context, hostID uint) error {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("EVAL", invalidateHostScript, 2,
		s.hostETagKey(hostID), s.hostQuarantineKey(hostID),
		int(s.quarantineTTL.Seconds())); err != nil {
		metricInvalidations.WithLabelValues("host", "error").Inc()
		return ctxerr.Wrap(ctx, err, "redis config etag host invalidate")
	}
	metricInvalidations.WithLabelValues("host", "ok").Inc()
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Gate state (cache-mode selection)
////////////////////////////////////////////////////////////////////////////////

// LegacyPacksPresent reports whether the deployment has any user-created 2017
// packs, caching the answer in Redis for gateStateTTL. On any error it
// returns true — for this gate, "assume present" is the safe direction: it
// bypasses the fast path (costing only performance), whereas a wrong "false"
// could let a host 304 past a config change that never fires an invalidation
// event.
func (s *Store) LegacyPacksPresent(ctx context.Context, load func(ctx context.Context) (bool, error)) (bool, error) {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	val, err := redigo.String(conn.Do("GET", s.testPrefix+legacyKey))
	if err == nil {
		return val != "0", nil
	}
	if err != redigo.ErrNil {
		return true, ctxerr.Wrap(ctx, err, "redis config etag legacy flag get")
	}

	// Cache miss: recompute. Gate loaders are the only DB load this package
	// causes, at most once per gateStateTTL (or reset) per cluster.
	metricGateLoads.Inc()
	present, err := load(ctx)
	if err != nil {
		return true, ctxerr.Wrap(ctx, err, "load legacy packs presence")
	}
	stored := "0"
	if present {
		stored = "1"
	}
	if _, err := conn.Do("SET", s.testPrefix+legacyKey, stored, "EX", int(gateStateTTL.Seconds())); err != nil {
		return present, ctxerr.Wrap(ctx, err, "redis config etag legacy flag set")
	}
	// Bounded state-write log: at most once per gateStateTTL/reset.
	s.logger.InfoContext(ctx, "config etag gate state written",
		"kind", "legacy_packs", "present", present)
	return present, nil
}

// ResetLegacyPacksFlag drops the cached legacy-packs answer so the next
// LegacyPacksPresent recomputes it. Pack CRUD hooks call this because the
// answer may have just changed in either direction.
func (s *Store) ResetLegacyPacksFlag(ctx context.Context) error {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("DEL", s.testPrefix+legacyKey); err != nil {
		return ctxerr.Wrap(ctx, err, "redis config etag legacy flag reset")
	}
	return nil
}

// LabelScopes returns the cached set of scopes containing label-scoped
// scheduled reports, loading and caching it (gateStateTTL) on a miss. On any
// error the caller must treat the answer as UNKNOWN and bypass the short
// circuit — never guess that a host is eligible for the shared key.
func (s *Store) LabelScopes(ctx context.Context, load func(ctx context.Context) (fleet.ConfigETagLabelScopes, error)) (fleet.ConfigETagLabelScopes, error) {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	val, err := redigo.String(conn.Do("GET", s.testPrefix+scopeModesKey))
	if err == nil {
		var scopes fleet.ConfigETagLabelScopes
		if uerr := json.Unmarshal([]byte(val), &scopes); uerr != nil {
			// malformed cached state: treat as unknown (bypass), it will be
			// rewritten on the next successful load after reset/expiry
			return fleet.ConfigETagLabelScopes{}, ctxerr.Wrap(ctx, uerr, "parse cached label scopes")
		}
		return scopes, nil
	}
	if err != redigo.ErrNil {
		return fleet.ConfigETagLabelScopes{}, ctxerr.Wrap(ctx, err, "redis config etag label scopes get")
	}

	metricGateLoads.Inc()
	scopes, err := load(ctx)
	if err != nil {
		return fleet.ConfigETagLabelScopes{}, ctxerr.Wrap(ctx, err, "load label scoped report scopes")
	}
	raw, err := json.Marshal(scopes)
	if err != nil {
		return fleet.ConfigETagLabelScopes{}, ctxerr.Wrap(ctx, err, "marshal label scopes")
	}
	if _, err := conn.Do("SET", s.testPrefix+scopeModesKey, string(raw), "EX", int(gateStateTTL.Seconds())); err != nil {
		// the loaded answer is still valid for this request
		return scopes, ctxerr.Wrap(ctx, err, "redis config etag label scopes set")
	}
	// Bounded state-write log: at most once per gateStateTTL/reset.
	s.logger.InfoContext(ctx, "config etag gate state written",
		"kind", "label_scopes", "global", scopes.Global, "team_count", len(scopes.TeamIDs))
	return scopes, nil
}

// ResetLabelScopes drops the cached label-scope answer so the next
// LabelScopes recomputes it. Query CRUD and label-deletion hooks call this
// because the answer may have just changed in either direction. Prompt reset
// matters more than the TTL: a stale "shared" answer combined with a newly
// label-scoped report could let team-shared keys be trusted for
// host-specific configs once the fence expires.
func (s *Store) ResetLabelScopes(ctx context.Context) error {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("DEL", s.testPrefix+scopeModesKey); err != nil {
		return ctxerr.Wrap(ctx, err, "redis config etag label scopes reset")
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// helpers
////////////////////////////////////////////////////////////////////////////////

// parseRecordAndGen extracts the record value and the current generation from
// an MGET(recordKey, genKey) reply. A missing record yields ("", gen, nil); a
// missing generation key means no invalidation ever ran and reads as "0".
func parseRecordAndGen(ctx context.Context, vals []any) (record, currentGen string, err error) {
	if len(vals) != 2 || vals[0] == nil {
		return "", "", nil
	}
	record, err = redigo.String(vals[0], nil)
	if err != nil {
		return "", "", ctxerr.Wrap(ctx, err, "redis config etag record type")
	}
	currentGen = "0"
	if vals[1] != nil {
		if currentGen, err = redigo.String(vals[1], nil); err != nil {
			return "", "", ctxerr.Wrap(ctx, err, "redis config etag gen type")
		}
	}
	return record, currentGen, nil
}
