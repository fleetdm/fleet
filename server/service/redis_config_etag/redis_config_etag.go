package redis_config_etag

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"
	"sync/atomic"
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
	// Do not lower this without re-deriving it:
	//
	// It must be strictly greater than the composed worst-case staleness of
	// the per-instance in-memory caches that feed the config build:
	//
	//	  packConfigCache TTL (1m, service.PackConfigCacheTTL)
	//	+ max cached_mysql TTL feeding the build (1m: defaultPacksExpiration,
	//	  defaultScheduledQueriesExpiration and
	//	  defaultTeamAgentOptionsExpiration — AgentOptionsForHost reads the
	//	  last of these; see cached_mysql.MaxConfigInputTTL)
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
	DefaultHostETagMinTTL = 50 * time.Minute
	DefaultHostETagMaxTTL = 70 * time.Minute

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
	genKeyBase        = "{cfgetag}:gen"
	fenceKeyBase      = "{cfgetag}:fence"
	legacyKeyBase     = "{cfgetag}:legacy-packs"
	scopeModesKeyBase = "{cfgetag}:scope-modes"
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
// Must be used by pointer (it carries atomic leader-election flags).
type Store struct {
	pool           fleet.RedisPool
	logger         *slog.Logger
	fenceTTL       time.Duration
	etagTTL        time.Duration
	hostETagMinTTL time.Duration
	hostETagMaxTTL time.Duration
	quarantineTTL  time.Duration
	version        string // Fleet server version baked into ETag record keys
	testPrefix     string // for tests, the key prefix to use to avoid conflicts

	// Per-gate leader-election flags: true while a database load for that
	// gate is in flight on this Fleet instance. See the non-blocking
	// leader election note on the gate methods.
	legacyLoadInFlight atomic.Bool
	scopesLoadInFlight atomic.Bool
}

var _ fleet.ConfigETagStore = (*Store)(nil)

// New returns a Store using the default TTLs and the running server's
// version for key namespacing. The logger is used only for bounded
// state-write logs and must not be nil.
func New(pool fleet.RedisPool, logger *slog.Logger) *Store {
	return &Store{
		pool:           pool,
		logger:         logger,
		fenceTTL:       DefaultFenceTTL,
		etagTTL:        DefaultETagTTL,
		hostETagMinTTL: DefaultHostETagMinTTL,
		hostETagMaxTTL: DefaultHostETagMaxTTL,
		quarantineTTL:  DefaultHostQuarantineTTL,
		version:        version.Version().Version,
	}
}

func (s *Store) genKey() string        { return s.testPrefix + genKeyBase }
func (s *Store) fenceKey() string      { return s.testPrefix + fenceKeyBase }
func (s *Store) legacyKey() string     { return s.testPrefix + legacyKeyBase }
func (s *Store) scopeModesKey() string { return s.testPrefix + scopeModesKeyBase }

func (s *Store) etagKey(scope, platform string) string {
	return fmt.Sprintf("%s%s:v%s:%s:%s", s.testPrefix, etagKeyPrefix, s.version, scope, platform)
}

func (s *Store) hostETagKey(hostID uint) string {
	return fmt.Sprintf("%s%s:v%s:%d", s.testPrefix, hostETagKeyPrefix, s.version, hostID)
}

func (s *Store) hostQuarantineKey(hostID uint) string {
	return fmt.Sprintf("%s%s:%d", s.testPrefix, hostQuarantineKeyPrefix, hostID)
}

// hostETagTTLSeconds returns a jittered TTL in [hostETagMinTTL, hostETagMaxTTL] so
// the backstop never expires a whole fleet's records at once.
func (s *Store) hostETagTTLSeconds() int {
	minSec := int(s.hostETagMinTTL.Seconds())
	maxSec := int(s.hostETagMaxTTL.Seconds())
	if maxSec <= minSec {
		return minSec
	}
	return minSec + rand.IntN(maxSec-minSec+1) //nolint:gosec // TTL jitter doesn't need cryptographic randomness
}

////////////////////////////////////////////////////////////////////////////////
// Shared records
////////////////////////////////////////////////////////////////////////////////

// GetETagIfCurrent returns the stored SHARED ETag for (scope, platform) only when its
// recorded generation matches the current generation. Any malformed or
// generation-stale record is reported as a miss (ok=false) — the caller then
// performs a normal full config build, so there is no failure mode here that
// can serve wrong data.
func (s *Store) GetETagIfCurrent(ctx context.Context, scope, platform string) (etag string, ok bool, err error) {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	// Single MGET; both keys share the {cfgetag} slot.
	vals, err := redigo.Values(conn.Do("MGET", s.etagKey(scope, platform), s.genKey()))
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
// Atomicity is load-bearing: this check-and-set MUST be a single atomic
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
		s.fenceKey(), s.genKey(), s.etagKey(scope, platform),
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
		s.genKey(), s.fenceKey(),
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

// GetHostETagIfCurrent returns the stored PER-HOST ETag only when its recorded
// generation matches the current generation AND its recorded scope and
// platform match the authenticated host context. A team transfer or platform
// change therefore reads as a miss with no key cleanup required. Malformed or
// stale records are misses — never matches.
func (s *Store) GetHostETagIfCurrent(ctx context.Context, hostID uint, scope, platform string) (etag string, ok bool, err error) {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	vals, err := redigo.Values(conn.Do("MGET", s.hostETagKey(hostID), s.genKey()))
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
	// "team:<id>", and the ETag is a SHA-256 hex string, so none of
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
		s.fenceKey(), s.hostQuarantineKey(hostID), s.genKey(), s.hostETagKey(hostID),
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
// Atomicity is load-bearing: a pipeline is NOT atomic, so a publish
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

// gateGet reads a gate-state key with a short-lived connection. found=false
// means the key does not exist. The connection is never held across anything
// slower than the GET itself.
func (s *Store) gateGet(ctx context.Context, key string) (val string, found bool, err error) {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	val, err = redigo.String(conn.Do("GET", key))
	if err == redigo.ErrNil {
		return "", false, nil
	}
	if err != nil {
		return "", false, ctxerr.Wrap(ctx, err, "redis config etag gate get")
	}
	return val, true, nil
}

// gateSet writes a gate-state key with a short-lived connection.
func (s *Store) gateSet(ctx context.Context, key, val string) error {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("SET", key, val, "EX", int(gateStateTTL.Seconds())); err != nil {
		return ctxerr.Wrap(ctx, err, "redis config etag gate set")
	}
	return nil
}

// Non-blocking leader election (both gate loaders).
//
// A naive GET → miss → SELECT → SET turns one Redis miss into one database
// query PER CONCURRENT CONFIG REQUEST during the miss window — and the
// window widens exactly when the database is degraded, amplifying the load
// at the worst moment. A blocking singleflight would fix the duplication but
// introduce something worse: followers waiting on the leader would couple
// config-request latency to a possibly-hung database query, violating the
// fail-open guarantee.
//
// Instead: one CAS flag per gate per Fleet instance. The winner re-checks
// Redis (another container may have just loaded), runs the loader, SETs the
// state. Losers do NOT wait and do NOT query — they return
// fleet.ErrConfigETagGateLoading, which callers treat as "unknown → bypass
// for this request" (one ordinary full build, the baseline cost). Bound: at
// most one loader query per container per miss window, zero added latency,
// no new stall modes. Redis connections are scoped per operation and never
// held across the database load.

// LegacyPacksPresent reports whether the deployment has any user-created 2017
// packs, caching the answer in Redis for gateStateTTL. On any error it
// returns true — for this gate, "assume present" is the safe direction: it
// bypasses the fast path (costing only performance), whereas a wrong "false"
// could let a host match past a config change that never fires an invalidation
// event. Returns fleet.ErrConfigETagGateLoading (with present=true) when
// another request on this instance is already loading — normal contention,
// not a fault.
func (s *Store) LegacyPacksPresent(ctx context.Context, load func(ctx context.Context) (bool, error)) (bool, error) {
	val, found, err := s.gateGet(ctx, s.legacyKey())
	if err != nil {
		return true, err
	}
	if found {
		return val != "0", nil
	}

	// Cache miss: elect a loader (see the leader-election note above).
	if !s.legacyLoadInFlight.CompareAndSwap(false, true) {
		return true, fleet.ErrConfigETagGateLoading
	}
	defer s.legacyLoadInFlight.Store(false)

	// Re-check under leadership: another Fleet instance may have loaded and
	// SET while we were losing the race locally.
	if val, found, err := s.gateGet(ctx, s.legacyKey()); err != nil {
		return true, err
	} else if found {
		return val != "0", nil
	}

	metricGateLoads.Inc()
	present, err := load(ctx)
	if err != nil {
		return true, ctxerr.Wrap(ctx, err, "load legacy packs presence")
	}
	stored := "0"
	if present {
		stored = "1"
	}
	if err := s.gateSet(ctx, s.legacyKey(), stored); err != nil {
		return present, err
	}
	// Bounded state-write log: at most once per gateStateTTL/reset per
	// container (only the elected loader reaches this line).
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

	if _, err := conn.Do("DEL", s.legacyKey()); err != nil {
		return ctxerr.Wrap(ctx, err, "redis config etag legacy flag reset")
	}
	return nil
}

// LabelScopes returns the cached set of scopes containing label-scoped
// scheduled reports, loading and caching it (gateStateTTL) on a miss. On any
// error the caller must treat the answer as UNKNOWN and bypass the short
// circuit — never guess that a host is eligible for the shared key. Returns
// fleet.ErrConfigETagGateLoading when another request on this instance is
// already loading — normal contention, not a fault (see the leader-election
// note above).
func (s *Store) LabelScopes(ctx context.Context, load func(ctx context.Context) (fleet.ConfigETagLabelScopes, error)) (fleet.ConfigETagLabelScopes, error) {
	parse := func(val string) (fleet.ConfigETagLabelScopes, error) {
		var scopes fleet.ConfigETagLabelScopes
		if err := json.Unmarshal([]byte(val), &scopes); err != nil {
			// malformed cached state: treat as unknown (bypass), it will be
			// rewritten on the next successful load after reset/expiry
			return fleet.ConfigETagLabelScopes{}, ctxerr.Wrap(ctx, err, "parse cached label scopes")
		}
		return scopes, nil
	}

	val, found, err := s.gateGet(ctx, s.scopeModesKey())
	if err != nil {
		return fleet.ConfigETagLabelScopes{}, err
	}
	if found {
		return parse(val)
	}

	// Cache miss: elect a loader (see the leader-election note above).
	if !s.scopesLoadInFlight.CompareAndSwap(false, true) {
		return fleet.ConfigETagLabelScopes{}, fleet.ErrConfigETagGateLoading
	}
	defer s.scopesLoadInFlight.Store(false)

	// Re-check under leadership: another Fleet instance may have loaded and
	// SET while we were losing the race locally.
	if val, found, err := s.gateGet(ctx, s.scopeModesKey()); err != nil {
		return fleet.ConfigETagLabelScopes{}, err
	} else if found {
		return parse(val)
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
	if err := s.gateSet(ctx, s.scopeModesKey(), string(raw)); err != nil {
		// the loaded answer is still valid for this request
		return scopes, err
	}
	// Bounded state-write log: at most once per gateStateTTL/reset per
	// container (only the elected loader reaches this line).
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

	if _, err := conn.Do("DEL", s.scopeModesKey()); err != nil {
		return ctxerr.Wrap(ctx, err, "redis config etag label scopes reset")
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// helpers
////////////////////////////////////////////////////////////////////////////////

// parseRecordAndGen extracts the record value and the current generation from
// an MGET(recordKey, genKey()) reply. A missing record yields ("", gen, nil); a
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
