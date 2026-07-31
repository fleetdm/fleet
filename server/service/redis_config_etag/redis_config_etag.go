// Package redis_config_etag implements the Redis-backed store for osquery
// config ETags — the persistence half of the config "SHORT CIRCUIT" that lets
// Fleet answer a config check-in with 304 Not Modified without building the
// config (and therefore without any database reads).
//
// ██ READ THIS BEFORE CHANGING ANYTHING IN THIS PACKAGE ██████████████████████
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
// # The write-fence design
//
// Two cooperating mechanisms with distinct jobs:
//
//   - Generation counter (read-side invalidation): every config-affecting
//     datastore write INCRs it (see server/datastore/etag_invalidate). Stored
//     ETags carry the generation current when written; the read path treats a
//     mismatch as a miss. Kills existing ETags instantly, in O(1), no matter
//     how many (team, platform) keys exist.
//
//   - Write fence (write-side quarantine): the same write arms a fence key
//     whose TTL outlives the composed in-memory cache staleness (see
//     DefaultFenceTTL). While the fence is armed, SetIfNoFence refuses to
//     persist ETags. Once it expires, every in-memory cache entry that
//     predates the mutation has necessarily expired too, so any build that
//     completes afterward is fresh and safe to persist.
//
// Correctness argument: Invalidate (INCR gen + SET fence) is one atomic EVAL;
// SetIfNoFence ("if no fence, SET record stamped with current gen") is
// another. Redis serializes them, so for a build whose inputs were fetched
// before a mutation:
//
//   - its write lands BEFORE the invalidation → the gen bump immediately
//     makes the stored record stale for reads. Safe.
//   - its write lands AFTER the invalidation → the fence exists → the write
//     is suppressed. Safe.
//
// There is no interleaving in which a stale ETag survives with a current
// generation. Fresh builds during the fence window are also suppressed —
// that is the accepted cost (the fast path is cold for ~DefaultFenceTTL after
// each config change; requests degrade to a normal full build).
//
// # Redis Cluster
//
// Every key in this package embeds the {cfgetag} hash tag so they all land in
// the same slot, which is required for the multi-key MGET and EVAL calls. The
// ops are single tiny GET/SET/INCRs, so the single-slot hot spot is
// negligible at realistic check-in rates.
//
// # Fleet version namespacing
//
// The ETag keys embed the Fleet server version. A server upgrade can change
// the rendered config body (new keys, changed defaults, encoder changes) with
// ZERO datastore writes — no invalidation hook would ever fire. Namespacing
// by version makes every upgrade start a clean keyspace; the per-key TTL
// garbage-collects the orphaned old-version keys.
//
// ████████████████████████████████████████████████████████████████████████████
package redis_config_etag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/version"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	// DefaultFenceTTL is how long ETag writes stay suppressed after a
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

	// DefaultETagTTL is the backstop TTL on stored ETag records. The
	// generation/fence mechanism is the correctness mechanism and would
	// suffice alone if invalidation-hook coverage (see
	// server/datastore/etag_invalidate) were provably complete forever. The
	// TTL converts any MISSED hook — or an out-of-band DB edit, a restore
	// from backup, a future mutation added without a hook — from a
	// permanently poisoned ETag into a bounded delay, at the cost of one full
	// config build per (team, platform) per TTL. It may be lengthened as
	// confidence grows. NEVER remove it.
	DefaultETagTTL = 1 * time.Hour

	// blockedFlagTTL caches the deployment-wide "is the short circuit
	// blocked" answer (2017 packs or label-scoped reports exist; see
	// fleet.ConfigETagStore.ShortCircuitBlocked). On expiry the next reader
	// recomputes it with cheap DB queries, so this is the full extent of the
	// DB load this package adds.
	blockedFlagTTL = 5 * time.Minute
)

// All keys share the {cfgetag} hash tag → same Redis Cluster slot, so the
// multi-key MGET/EVAL calls below are legal in cluster mode.
const (
	genKey     = "{cfgetag}:gen"
	fenceKey   = "{cfgetag}:fence"
	blockedKey = "{cfgetag}:blocked"
	// etag record keys append ":v<fleet version>:<scope>:<platform>", see
	// (*Store).etagKey.
	etagKeyPrefix = "{cfgetag}:etag"
)

// Store implements fleet.ConfigETagStore on Redis. See the package docs for
// the correctness model; see fleet.ConfigETagStore for the method contracts.
type Store struct {
	pool       fleet.RedisPool
	fenceTTL   time.Duration
	etagTTL    time.Duration
	version    string // Fleet server version baked into ETag keys
	testPrefix string // for tests, the key prefix to use to avoid conflicts
}

var _ fleet.ConfigETagStore = (*Store)(nil)

// New returns a Store using the default TTLs and the running server's
// version for key namespacing.
func New(pool fleet.RedisPool) *Store {
	return &Store{
		pool:     pool,
		fenceTTL: DefaultFenceTTL,
		etagTTL:  DefaultETagTTL,
		version:  version.Version().Version,
	}
}

func (s *Store) etagKey(scope, platform string) string {
	return fmt.Sprintf("%s%s:v%s:%s:%s", s.testPrefix, etagKeyPrefix, s.version, scope, platform)
}

// GetValid returns the stored ETag for (scope, platform) only when its
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
		return "", false, ctxerr.Wrap(ctx, err, "redis config etag get")
	}
	if len(vals) != 2 || vals[0] == nil {
		// no record stored for this (scope, platform)
		return "", false, nil
	}
	record, err := redigo.String(vals[0], nil)
	if err != nil {
		return "", false, ctxerr.Wrap(ctx, err, "redis config etag record type")
	}

	// The record is "<generation>|<etag>", written by SetIfNoFence. A missing
	// gen key means no invalidation ever ran; records are then stamped "0".
	currentGen := "0"
	if vals[1] != nil {
		if currentGen, err = redigo.String(vals[1], nil); err != nil {
			return "", false, ctxerr.Wrap(ctx, err, "redis config etag gen type")
		}
	}
	recordGen, recordETag, found := strings.Cut(record, "|")
	if !found || recordETag == "" {
		// malformed record: treat as a miss, never as a match
		return "", false, nil
	}
	if recordGen != currentGen {
		// A config-affecting write happened after this record was stored.
		// The record is stale — this is the read half of the invalidation.
		return "", false, nil
	}
	return recordETag, true, nil
}

// setIfNoFenceScript persists an ETag record stamped with the current
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

// SetIfNoFence persists the ETag for (scope, platform) with the current
// generation and the backstop TTL — unless the write fence is armed (a
// config-affecting write happened within the last fenceTTL), in which case
// nothing is stored and stored=false is returned. Callers must treat a
// suppressed write as normal operation, not an error.
func (s *Store) SetIfNoFence(ctx context.Context, scope, platform, etag string) (stored bool, err error) {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	res, err := redigo.Int64(conn.Do("EVAL", setIfNoFenceScript, 3,
		s.testPrefix+fenceKey, s.testPrefix+genKey, s.etagKey(scope, platform),
		etag, int(s.etagTTL.Seconds())))
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "redis config etag set")
	}
	return res == 1, nil
}

// invalidateScript bumps the generation (read-side invalidation of every
// stored ETag) and arms the write fence, atomically. Doing both in one script
// is what makes the interleaving argument in the package docs hold.
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
// deliberately coarse — one global generation — because over-invalidating
// only costs the optimization for the fence window, never correctness.
func (s *Store) Invalidate(ctx context.Context) error {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("EVAL", invalidateScript, 2,
		s.testPrefix+genKey, s.testPrefix+fenceKey,
		int(s.fenceTTL.Seconds())); err != nil {
		return ctxerr.Wrap(ctx, err, "redis config etag invalidate")
	}
	return nil
}

// ShortCircuitBlocked reports whether the deployment has any short circuit
// blocker (user-created 2017 packs or label-scoped reports — see the
// fleet.ConfigETagStore docs), caching the answer in Redis for
// blockedFlagTTL. On any error it returns true — for this gate, "assume
// blocked" is the safe direction: it bypasses the fast path (costing only
// performance), whereas a wrong "false" could let a host 304 past a config
// change that never fires an invalidation event.
func (s *Store) ShortCircuitBlocked(ctx context.Context, load func(ctx context.Context) (bool, error)) (bool, error) {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	val, err := redigo.String(conn.Do("GET", s.testPrefix+blockedKey))
	if err == nil {
		return val != "0", nil
	}
	if err != redigo.ErrNil {
		return true, ctxerr.Wrap(ctx, err, "redis config etag blocked flag get")
	}

	// Cache miss: recompute. This is the only DB load this package causes,
	// at most once per blockedFlagTTL per cluster.
	blocked, err := load(ctx)
	if err != nil {
		return true, ctxerr.Wrap(ctx, err, "load config etag short circuit blockers")
	}
	stored := "0"
	if blocked {
		stored = "1"
	}
	if _, err := conn.Do("SET", s.testPrefix+blockedKey, stored, "EX", int(blockedFlagTTL.Seconds())); err != nil {
		return blocked, ctxerr.Wrap(ctx, err, "redis config etag blocked flag set")
	}
	return blocked, nil
}

// ResetShortCircuitBlockedFlag drops the cached blocked answer so the next
// ShortCircuitBlocked recomputes it. Pack and query CRUD hooks call this
// because the answer may have just changed in either direction — and calling
// it promptly matters: the flag lives longer than the write fence, so a
// stale "not blocked" answer combined with a newly host-specific config
// could otherwise poison team-shared keys once the fence expires.
func (s *Store) ResetShortCircuitBlockedFlag(ctx context.Context) error {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("DEL", s.testPrefix+blockedKey); err != nil {
		return ctxerr.Wrap(ctx, err, "redis config etag blocked flag reset")
	}
	return nil
}
