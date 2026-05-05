package mysqlredis

import (
	"context"
	"errors"
	mathrand "math/rand/v2"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/go-json-experiment/json/v1"
	redigo "github.com/gomodule/redigo/redis"
)

// All host-cache keys live under this single versioned prefix so operators can
// purge with `redis-cli --scan --pattern 'fleet:hostcache:v1:*' | xargs redis-cli DEL`.
// Bumping the version on a cached-payload schema change orphans old keys; they
// TTL out within hostCacheTTL.
const hostCacheKeyPrefix = "fleet:hostcache:v1"

const (
	// hostCacheNegativeTTL caps how long a "not found" result is cached. The value trades off two forces:
	//   - DoS/retry protection: collapse bursts of identical bad-key auth attempts (retry storms, multi-verb request
	//     clusters from one agent, attacker probes) into a single DB hit within the window.
	//   - Enrollment-race safety: if a key is probed before EnrollOsquery/EnrollOrbit creates it AND the enrollment's
	//     invalidation somehow misses the negative entry, this bounds the delay on the host's first successful auth.
	//
	// Why 5s rather than neighboring values:
	//   - 1s is too short. It sits inside common HTTP retry backoffs (250ms-1s), so it adds little beyond the
	//     singleflight collapse that already dedupes concurrent in-flight misses.
	//   - 10s matches the default osquery poll interval (distributed_interval), producing noisy hit/miss behavior
	//     on clock skew and starting to feel like a real delay when debugging a failed enrollment.
	//   - 30s+ would start protecting against a persistent stale agent (deleted host, agent still polling every 10s),
	//     but magnifies the enrollment-debug window if invalidation ever misses. Bump here as a follow-up if
	//     production metrics show repeated-bad-key traffic is a real DB pressure source; the immediate goal is
	//     burst absorption, which 5s handles.
	hostCacheNegativeTTL = 5 * time.Second

	// hostCacheTTLJitterFraction spreads entry expiry across a ±(fraction/2)
	// window around the configured base TTL, so a Redis restart or TTL-driven
	// wave doesn't trigger a synchronized stampede back to the reader.
	hostCacheTTLJitterFraction = 0.2

	// hostCacheFlightTimeout caps the duration of the singleflight-shared DB call. The flight ctx is
	// detached from the originating caller's deadline (see loadHostFamily), so without this cap a wedged
	// query would pin the singleflight slot indefinitely and starve every subsequent caller for the same
	// node_key. 30s is well above the p99.9 of a single-row indexed lookup (typically 1-5ms) and below the
	// 60s window where waiting clients have any chance of caring about the result.
	hostCacheFlightTimeout = 30 * time.Second
)

// hostCacheLookup is the tri-state result of a cache read.
type hostCacheLookup int

const (
	// hostCacheLookupMiss means "no usable cache state" — caller must fall
	// through to the database. This is also returned on Redis/JSON errors:
	// the cache never fails a request.
	hostCacheLookupMiss hostCacheLookup = iota

	// hostCacheLookupHit means the returned *fleet.Host is valid and can be
	// served to the caller as-is.
	hostCacheLookupHit

	// hostCacheLookupNegative means the cache holds a prior "not found"
	// result for this node_key; caller should return NotFound without hitting
	// the database.
	hostCacheLookupNegative
)

func hostCacheKeyByNodeKey(nodeKey string) string {
	return hostCacheKeyPrefix + ":nk:" + nodeKey
}

func hostCacheKeyMiss(nodeKey string) string {
	return hostCacheKeyPrefix + ":nk_miss:" + nodeKey
}

func hostCacheIndexByID(hostID uint) string {
	return hostCacheKeyPrefix + ":id2nk:" + strconv.FormatUint(uint64(hostID), 10)
}

func hostCacheKeyByOrbitNodeKey(orbitNodeKey string) string {
	return hostCacheKeyPrefix + ":onk:" + orbitNodeKey
}

func hostCacheKeyOrbitMiss(orbitNodeKey string) string {
	return hostCacheKeyPrefix + ":onk_miss:" + orbitNodeKey
}

func hostCacheOrbitIndexByID(hostID uint) string {
	return hostCacheKeyPrefix + ":id2onk:" + strconv.FormatUint(uint64(hostID), 10)
}

// cacheFamily holds the family-specific keying primitives shared between the osquery (LoadHostByNodeKey) and
// orbit (LoadHostByOrbitNodeKey) cache paths. The cache read/write/invalidation methods take a cacheFamily value
// to avoid duplicating the same logic for both families. Adding a third agent in the future means adding one more
// cacheFamily value at this site; the read/write code does not change.
type cacheFamily struct {
	sfPrefix   string                    // singleflight key prefix to disambiguate flights ("nk:" or "onk:")
	primaryKey func(string) string       // positive-cache key constructor
	missKey    func(string) string       // negative-cache key constructor
	indexKey   func(uint) string         // reverse-index key constructor
	nodeKeyOf  func(*fleet.Host) *string // accessor for the family's host key field on *fleet.Host
}

var (
	osqueryCacheFamily = cacheFamily{
		sfPrefix:   "nk:",
		primaryKey: hostCacheKeyByNodeKey,
		missKey:    hostCacheKeyMiss,
		indexKey:   hostCacheIndexByID,
		nodeKeyOf:  func(h *fleet.Host) *string { return h.NodeKey },
	}
	orbitCacheFamily = cacheFamily{
		sfPrefix:   "onk:",
		primaryKey: hostCacheKeyByOrbitNodeKey,
		missKey:    hostCacheKeyOrbitMiss,
		indexKey:   hostCacheOrbitIndexByID,
		nodeKeyOf:  func(h *fleet.Host) *string { return h.OrbitNodeKey },
	}
)

// jitteredHostCacheTTL returns the configured base TTL perturbed by
// ±(hostCacheTTLJitterFraction / 2). With the default 0.2 and a 60s base, the
// result falls in [54s, 66s], yielding ~5 cache hits per miss at the default
// 10s osquery check-in interval (~83% hit rate).
func (d *Datastore) jitteredHostCacheTTL() time.Duration {
	if d.hostCacheTTL <= 0 {
		return 0
	}
	half := float64(d.hostCacheTTL) * hostCacheTTLJitterFraction / 2
	// TTL jitter is a coarse scheduling concern, not a security boundary;
	// crypto/rand would be overkill and its failure modes (EAGAIN on low
	// entropy) are worse than using math/rand/v2 here.
	delta := (mathrand.Float64()*2 - 1) * half //nolint:gosec // non-security randomness
	return d.hostCacheTTL + time.Duration(delta)
}

// hostCacheGetFamily looks up a host by key in the given cache family. It checks the positive cache first (the
// common case) and falls through to the negative cache only on positive miss. Never propagates Redis or JSON
// errors; any error is recorded and the caller sees a hostCacheLookupMiss.
//
// On unmarshal failure the bad key is DELed so the next lookup repopulates from the database; this is the
// schema-drift defense.
func (d *Datastore) hostCacheGetFamily(ctx context.Context, fam cacheFamily, key string) (*fleet.Host, hostCacheLookup) {
	if !d.hostCacheEnabled || key == "" {
		return nil, hostCacheLookupMiss
	}

	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	raw, err := redigo.Bytes(conn.Do("GET", fam.primaryKey(key)))
	switch {
	case err == nil:
		envelope := new(hostCacheEnvelope)
		if jerr := json.Unmarshal(raw, envelope); jerr != nil {
			d.recordHostCacheErr(ctx, "get", jerr)
			if _, derr := conn.Do("DEL", fam.primaryKey(key)); derr != nil {
				d.recordHostCacheErr(ctx, "del", derr)
			}
			d.recordHostCacheLookup(ctx, "miss")
			return nil, hostCacheLookupMiss
		}
		d.recordHostCacheLookup(ctx, "hit")
		return envelope.toHost(), hostCacheLookupHit

	case errors.Is(err, redigo.ErrNil):
		// positive miss; fall through to negative-cache probe

	default:
		d.recordHostCacheErr(ctx, "get", err)
		d.recordHostCacheLookup(ctx, "miss")
		return nil, hostCacheLookupMiss
	}

	_, err = redigo.Bytes(conn.Do("GET", fam.missKey(key)))
	switch {
	case err == nil:
		d.recordHostCacheLookup(ctx, "negative_hit")
		return nil, hostCacheLookupNegative
	case errors.Is(err, redigo.ErrNil):
		d.recordHostCacheLookup(ctx, "miss")
		return nil, hostCacheLookupMiss
	default:
		d.recordHostCacheErr(ctx, "get", err)
		d.recordHostCacheLookup(ctx, "miss")
		return nil, hostCacheLookupMiss
	}
}

// hostCacheGetByNodeKey is the osquery-family wrapper of hostCacheGetFamily.
func (d *Datastore) hostCacheGetByNodeKey(ctx context.Context, nodeKey string) (*fleet.Host, hostCacheLookup) {
	return d.hostCacheGetFamily(ctx, osqueryCacheFamily, nodeKey)
}

// hostCachePutFamily stores host under the given family's primaryKey and updates that family's indexKey
// (reverse index) so invalidation-by-ID can find the primaryKey later. Fire-and-forget: errors are recorded,
// not returned.
//
// Write order: primaryKey BEFORE indexKey. This ordering is correctness-critical for safe interaction with
// a concurrent invalidate-by-ID for the same host. invalidate-by-ID's first command is GET indexKey; on
// ErrNil it returns silently without issuing any DELs. With primaryKey written first, an invalidator that
// arrives between the two SETs sees no indexKey yet, exits without touching the in-flight primaryKey, and
// the populate completes both writes consistently.
//
// Known limitation: if SET indexKey fails after SET primaryKey succeeded (Redis transient failure
// mid-pipeline with RetryConn also failing), the orphan primaryKey survives until TTL because
// invalidate-by-ID can't find it via the missing indexKey. This is a very rare fail, and TTL bounds the staleness.
//
// Both SETs use EXAT (absolute Unix-second expiration) with a single computed expiresAt. EXAT requires Redis 6.2+
func (d *Datastore) hostCachePutFamily(ctx context.Context, fam cacheFamily, host *fleet.Host) {
	if !d.hostCacheEnabled || host == nil {
		return
	}
	keyPtr := fam.nodeKeyOf(host)
	if keyPtr == nil || *keyPtr == "" {
		return
	}
	key := *keyPtr

	raw, err := json.Marshal(envelopeFromHost(host))
	if err != nil {
		d.recordHostCacheErr(ctx, "set", err)
		return
	}

	ttl := d.jitteredHostCacheTTL()
	// Redis only accepts integer TTLs. Floor at 1 second so a degenerate near-zero jittered value still produces a
	// future expiry rather than EXAT'ing into the past (which would immediately expire the keys).
	ttlSec := int(ttl.Seconds())
	if ttlSec <= 0 {
		ttlSec = 1
	}
	expiresAt := time.Now().Unix() + int64(ttlSec)

	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("SET", fam.primaryKey(key), raw, "EXAT", expiresAt); err != nil {
		d.recordHostCacheErr(ctx, "set", err)
		return
	}
	if host.ID != 0 {
		if _, err := conn.Do("SET", fam.indexKey(host.ID), key, "EXAT", expiresAt); err != nil {
			d.recordHostCacheErr(ctx, "set", err)
		}
	}
}

// hostCachePutByNodeKey is the osquery-family wrapper of hostCachePutFamily.
func (d *Datastore) hostCachePutByNodeKey(ctx context.Context, host *fleet.Host) {
	d.hostCachePutFamily(ctx, osqueryCacheFamily, host)
}

// hostCachePutNotFoundFamily stores a short-lived negative-cache entry under the given family. Used when the
// database returns NotFound; bounded by hostCacheNegativeTTL.
//
// Why per-key SET, not SADD: native Redis sets have set-level TTL only, so we can't expire individual members at
// the per-entry 5s granularity we want. Per-key SETs also spread across cluster slots and pair cleanly with the
// positive/index DELs in the variadic-DEL batched paths.
func (d *Datastore) hostCachePutNotFoundFamily(ctx context.Context, fam cacheFamily, key string) {
	if !d.hostCacheEnabled || key == "" {
		return
	}
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()
	if _, err := conn.Do("SET", fam.missKey(key), "1", "EX", int(hostCacheNegativeTTL.Seconds())); err != nil {
		d.recordHostCacheErr(ctx, "set", err)
	}
}

// hostCachePutNotFoundByNodeKey is the osquery-family wrapper of hostCachePutNotFoundFamily.
func (d *Datastore) hostCachePutNotFoundByNodeKey(ctx context.Context, nodeKey string) {
	d.hostCachePutNotFoundFamily(ctx, osqueryCacheFamily, nodeKey)
}

// hostCacheDeleteByNodeKey invalidates the primary, negative, and index keys
// when the caller already has the (nodeKey, hostID) pair. Pass hostID=0 if the
// caller does not know the ID; the index is skipped in that case.
// `reason` is a low-cardinality label recorded on the invalidations counter.
//
// When hostID > 0 and the reverse index points at a DIFFERENT (prior) key,
// e.g., osquery re-enrollment rotated node_key from OLD to NEW and we're
// called with NEW, the entry under the OLD key is also cleared. Without
// this step the rotated key would keep authenticating for up to TTL.
func (d *Datastore) hostCacheDeleteByNodeKey(ctx context.Context, nodeKey string, hostID uint, reason string) {
	if !d.hostCacheEnabled || nodeKey == "" {
		return
	}

	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	if hostID > 0 {
		if priorKey, err := redigo.String(conn.Do("GET", hostCacheIndexByID(hostID))); err == nil && priorKey != "" && priorKey != nodeKey {
			for _, k := range []string{hostCacheKeyByNodeKey(priorKey), hostCacheKeyMiss(priorKey)} {
				if _, err := conn.Do("DEL", k); err != nil {
					d.recordHostCacheErr(ctx, "del", err)
				}
			}
		} else if err != nil && !errors.Is(err, redigo.ErrNil) {
			d.recordHostCacheErr(ctx, "get", err)
		}
	}

	for _, k := range []string{
		hostCacheKeyByNodeKey(nodeKey),
		hostCacheKeyMiss(nodeKey),
	} {
		if _, err := conn.Do("DEL", k); err != nil {
			d.recordHostCacheErr(ctx, "del", err)
		}
	}
	if hostID > 0 {
		if _, err := conn.Do("DEL", hostCacheIndexByID(hostID)); err != nil {
			d.recordHostCacheErr(ctx, "del", err)
		}
	}
	d.recordHostCacheInvalidation(ctx, reason)
}

// hostCacheGetByOrbitNodeKey is the orbit-family wrapper of hostCacheGetFamily.
func (d *Datastore) hostCacheGetByOrbitNodeKey(ctx context.Context, orbitNodeKey string) (*fleet.Host, hostCacheLookup) {
	return d.hostCacheGetFamily(ctx, orbitCacheFamily, orbitNodeKey)
}

// hostCachePutByOrbitNodeKey is the orbit-family wrapper of hostCachePutFamily.
func (d *Datastore) hostCachePutByOrbitNodeKey(ctx context.Context, host *fleet.Host) {
	d.hostCachePutFamily(ctx, orbitCacheFamily, host)
}

// hostCacheDeleteByID invalidates both the osquery and orbit caches when only the host ID is known. It reads
// both reverse indices (id2nk, id2onk) and deletes whichever entries it finds. Either or both may be missing
// (host enrolled only one agent, TTL expiry, never populated). Missing entries mean there's nothing more to do
// on that side, and the invalidation is still counted so the metrics line up with write-path activity.
func (d *Datastore) hostCacheDeleteByID(ctx context.Context, hostID uint, reason string) {
	if !d.hostCacheEnabled || hostID == 0 {
		return
	}

	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	for _, fam := range []cacheFamily{osqueryCacheFamily, orbitCacheFamily} {
		d.clearEntriesByID(ctx, conn, fam, hostID)
	}
	d.recordHostCacheInvalidation(ctx, reason)
}

// clearEntriesByID resolves the given family's reverse index for hostID and DELs the resulting positive,
// negative, and reverse-index keys. Silent no-op when the reverse index is missing (host enrolled only one
// agent, TTL expiry, never populated).
func (d *Datastore) clearEntriesByID(ctx context.Context, conn redigo.Conn, fam cacheFamily, hostID uint) {
	key, err := redigo.String(conn.Do("GET", fam.indexKey(hostID)))
	switch {
	case errors.Is(err, redigo.ErrNil):
		return
	case err != nil:
		d.recordHostCacheErr(ctx, "get", err)
		return
	}
	for _, k := range []string{
		fam.primaryKey(key),
		fam.missKey(key),
		fam.indexKey(hostID),
	} {
		if _, err := conn.Do("DEL", k); err != nil {
			d.recordHostCacheErr(ctx, "del", err)
		}
	}
}

// hostCacheClearDirectEntries DELs the positive and negative cache entries
// for whichever of (nodeKey, orbitNodeKey) the caller supplies, without
// touching the reverse indices. Used alongside hostCacheDeleteByID to cover
// the case where a stale NotFound was negatively-cached under a key before
// the host row existed. The reverse-index path can't find those entries
// (no id2 mapping is populated for a negative hit), so they must be cleared
// by key. Does NOT record an invalidation; the caller already did.
func (d *Datastore) hostCacheClearDirectEntries(ctx context.Context, nodeKey, orbitNodeKey string) {
	if !d.hostCacheEnabled || (nodeKey == "" && orbitNodeKey == "") {
		return
	}
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	pairs := []struct {
		fam cacheFamily
		key string
	}{
		{osqueryCacheFamily, nodeKey},
		{orbitCacheFamily, orbitNodeKey},
	}
	for _, p := range pairs {
		if p.key == "" {
			continue
		}
		for _, k := range []string{p.fam.primaryKey(p.key), p.fam.missKey(p.key)} {
			if _, err := conn.Do("DEL", k); err != nil {
				d.recordHostCacheErr(ctx, "del", err)
			}
		}
	}
}

// hostCacheInvalidateBatchSize caps the number of keys per pipelined MGET / DEL call. Keeps individual Redis
// commands bounded regardless of input size.
//
// Why 500: Redis is single-threaded, so each command blocks all other clients for its duration. The chosen value
// targets ~3ms of server time per chunk, the rough community threshold for "noticeable but not problematic." Each
// MGET response carries ~1.7KB of JSON envelope per key, so 500 keys = ~850KB per chunk, comparable in wire-bytes
// to redisSetMembersBatchSize=10000 in hosts.go (whose elements are ~10-byte host IDs). Common practice for
// pipelined batches is 100-1000; 500 is a defensible middle. Lower if metrics show invalidation causing Redis CPU
// spikes during bulk team moves; raise if round-trip latency dominates.
const hostCacheInvalidateBatchSize = 500

// invalidateHostIDs efficiently invalidates both cache families for a batch
// of host IDs. Equivalent to calling hostCacheDeleteByID in a loop but uses
// pipelined MGET + variadic DEL to collapse what would otherwise be ~8
// sequential round-trips per host into O(slots × chunks) round-trips for
// the whole batch. At 10k hosts on a loaded Redis, that's the difference
// between ~80 seconds and ~200 milliseconds. Records one invalidation
// counter bump per input ID. Errors are recorded on the errors counter and
// logged; TTL is the safety net for any keys that survived a transient
// Redis failure.
func (d *Datastore) invalidateHostIDs(ctx context.Context, ids []uint, reason string) {
	if !d.hostCacheEnabled || len(ids) == 0 {
		return
	}

	families := []cacheFamily{osqueryCacheFamily, orbitCacheFamily}
	nFam := len(families)

	// Phase 1: batch-GET the reverse index for every (id, family) pair, in family-major order
	// [osq_id1, orb_id1, osq_id2, orb_id2, ...] so resolved[i*nFam+f] maps unambiguously to (id i, family f).
	idxKeys := make([]string, 0, nFam*len(ids))
	for _, id := range ids {
		for _, fam := range families {
			idxKeys = append(idxKeys, fam.indexKey(id))
		}
	}
	resolved := d.pipelinedMGET(ctx, idxKeys)

	// Phase 2: build the full DEL key list (payload + negative + reverse index for whichever family was
	// populated) and issue pipelined DELs.
	delKeys := make([]string, 0, 3*nFam*len(ids))
	for i, id := range ids {
		for f, fam := range families {
			if key := resolved[i*nFam+f]; key != "" {
				delKeys = append(delKeys, fam.primaryKey(key), fam.missKey(key))
			}
			delKeys = append(delKeys, fam.indexKey(id))
		}
	}
	d.pipelinedDEL(ctx, delKeys)

	// Phase 3: bump the invalidations counter by len(ids) in a single Add to match the per-host-invalidation
	// semantics of the non-batched path. (OTEL counters aggregate identically whether you Add(N) once or
	// Add(1) N times; one call is cheaper.)
	d.recordHostCacheInvalidations(ctx, reason, len(ids))
}

// pipelinedMGET returns the string value for each key in input order. Empty string for missing keys or if an
// error prevented retrieval.
//
// Works in both modes via redis.SplitKeysBySlot: in cluster mode keys are grouped by slot (CROSSSLOT-safe)
// and one MGET per slot group is issued; in standalone mode SplitKeysBySlot returns a single group containing
// all keys and BindConn is a no-op. Either way the group is further chunked at hostCacheInvalidateBatchSize.
func (d *Datastore) pipelinedMGET(ctx context.Context, keys []string) []string {
	result := make([]string, len(keys))
	if len(keys) == 0 {
		return result
	}

	// Slot grouping rearranges keys; this map restores input order.
	indexOf := make(map[string]int, len(keys))
	for i, k := range keys {
		indexOf[k] = i
	}

	for _, group := range redis.SplitKeysBySlot(d.pool, keys...) {
		for len(group) > 0 {
			n := min(len(group), hostCacheInvalidateBatchSize)
			chunk := group[:n]
			group = group[n:]
			d.mgetChunk(ctx, chunk, indexOf, result)
		}
	}
	return result
}

func (d *Datastore) mgetChunk(ctx context.Context, chunk []string, indexOf map[string]int, out []string) {
	conn := d.pool.Get()
	defer conn.Close()
	// BindConn MUST come before ConfigureDoer. redisc's BindConn expects the raw cluster conn from the pool,
	// not a RetryConn wrapper. See server/datastore/redis/ratelimit_store.go for the canonical ordering.
	// In standalone mode BindConn is a no-op.
	if err := redis.BindConn(d.pool, conn, chunk...); err != nil {
		d.recordHostCacheErr(ctx, "get", err)
		return
	}
	// ConfigureDoer returns a RetryConn wrapper around conn. We use the wrapper for the Do() call but keep the
	// `defer conn.Close()` against the raw conn so the lifecycle is explicit and tools don't flag the assignment
	// as a potential leak. The wrapper holds no extra resources beyond what the raw conn does.
	doer := redis.ConfigureDoer(d.pool, conn)
	args := redigo.Args{}.AddFlat(chunk)
	values, err := redigo.Values(doer.Do("MGET", args...))
	if err != nil {
		d.recordHostCacheErr(ctx, "get", err)
		return
	}
	for i, v := range values {
		if i >= len(chunk) {
			break
		}
		if v == nil {
			continue // missing key — leave out[] zero value
		}
		if b, ok := v.([]byte); ok {
			out[indexOf[chunk[i]]] = string(b)
		}
	}
}

// pipelinedDEL issues variadic DEL across all keys. Slot-grouped and chunked the same way as pipelinedMGET
// (see that function for the standalone-vs-cluster behavior). All errors are recorded and treated as
// best-effort; TTL is the backstop for any keys we failed to DEL.
func (d *Datastore) pipelinedDEL(ctx context.Context, keys []string) {
	if len(keys) == 0 {
		return
	}
	for _, group := range redis.SplitKeysBySlot(d.pool, keys...) {
		for len(group) > 0 {
			n := min(len(group), hostCacheInvalidateBatchSize)
			chunk := group[:n]
			group = group[n:]
			d.delChunk(ctx, chunk)
		}
	}
}

func (d *Datastore) delChunk(ctx context.Context, chunk []string) {
	conn := d.pool.Get()
	defer conn.Close()
	// BindConn before ConfigureDoer, see mgetChunk for the rationale.
	if err := redis.BindConn(d.pool, conn, chunk...); err != nil {
		d.recordHostCacheErr(ctx, "del", err)
		return
	}
	doer := redis.ConfigureDoer(d.pool, conn)
	args := redigo.Args{}.AddFlat(chunk)
	if _, err := doer.Do("DEL", args...); err != nil {
		d.recordHostCacheErr(ctx, "del", err)
	}
}

// recordHostCacheErr attaches the error to context-based logging (surfaced on
// the surrounding HTTP response log line via middleware) and increments the
// errors counter. Cache errors are always best-effort; the DB path still
// serves the request.
func (d *Datastore) recordHostCacheErr(ctx context.Context, op string, err error) {
	logging.WithErr(ctx, err)
	//nolint:nilaway // initialized in package init(); panic on registration failure guarantees non-nil
	hostCacheErrors.Add(ctx, 1, hostCacheErrorAttrs(op))
}

func (d *Datastore) recordHostCacheLookup(ctx context.Context, result string) {
	//nolint:nilaway // initialized in package init(); panic on registration failure guarantees non-nil
	hostCacheLookups.Add(ctx, 1, hostCacheLookupAttrs(result))
}

func (d *Datastore) recordHostCacheInvalidation(ctx context.Context, reason string) {
	d.recordHostCacheInvalidations(ctx, reason, 1)
}

// recordHostCacheInvalidations bumps the invalidations counter by n. The bulk-batch path uses this to add
// len(ids) in one Add() call instead of looping; observation at the metrics backend is identical.
func (d *Datastore) recordHostCacheInvalidations(ctx context.Context, reason string, n int) {
	if n <= 0 {
		return
	}
	//nolint:nilaway // initialized in package init(); panic on registration failure guarantees non-nil
	hostCacheInvalidations.Add(ctx, int64(n), hostCacheInvalidationAttrs(reason))
}

// loadHostFamily is the family-agnostic implementation of LoadHostByNodeKey and LoadHostByOrbitNodeKey.
// dbLoad is the inner Datastore method to invoke on cache miss.
//
// Semantics:
//   - Cache disabled: always delegate; never read or write the cache.
//   - Positive cache hit: return cached host.
//   - Negative cache hit: return a NotFoundError without hitting the DB.
//   - Miss: singleflight-guarded DB fetch; on success populate positive cache, on NotFound populate negative
//     cache; propagate other errors without populating either cache (transient failures must not poison).
//
// Concurrent-request handling (the reason for the singleflight + WithoutCancel + WithTimeout dance):
//
// When N concurrent HTTP requests authenticate the same host (e.g., one host's osquery agent has multiple
// in-flight calls to /api/v1/osquery/config and /distributed/read), all N callers cache-miss in the same
// instant. Without singleflight all N would issue identical DB queries; with singleflight only the first
// caller runs the closure and the other N-1 attach to that same execution and receive its result.
//
// The detail that needs care: the singleflight closure runs in the FIRST caller's goroutine and lexically
// captures whatever ctx is in scope. If we used the caller's ctx directly inside the closure and the first
// caller's request was canceled (e.g., a tight LB timeout fires at 50ms), the DB call would observe that
// cancellation and abort, returning an error that singleflight delivers to every attached caller, including
// ones that had seconds of remaining budget. context.WithoutCancel(ctx) detaches the inner DB call from any
// one caller's lifecycle: the call survives whichever caller happened to start it, so peers that joined the
// shared execution still receive the result. WithoutCancel preserves ctx VALUES (logger, request id, otel
// span) but drops cancellation; that's why we don't just use context.Background().
//
// We then wrap the detached ctx in context.WithTimeout(_, hostCacheFlightTimeout) so a wedged DB query
// cannot pin the singleflight slot for this node_key indefinitely. Fleet's MysqlConfig does not set
// readTimeout/writeTimeout on the DSN, so without an explicit cap the safety net for a stuck query would be
// operator-side only (MySQL server max_execution_time, TCP keepalive, or manual KILL). 30s is the cap.
//
// The singleflight key is prefixed with fam.sfPrefix so osquery and orbit shared executions cannot collide
// if a node_key ever happened to equal an orbit_node_key (astronomically unlikely with 32-char random keys
// but cheap to defend against), and so telemetry/debugging can tell the two populations apart.
//
// DoChan + select on ctx.Done() lets each individual caller abandon its own wait when its own ctx is
// canceled, without affecting the shared execution that joiners are still waiting on.
//
// Callers receive a host that is safe to mutate: cache-hit path returns a fresh struct from JSON unmarshal,
// shared-execution path returns a shallow copy so concurrent callers that joined the same execution don't
// race on each other's mutations (e.g., AuthenticateHost overwrites host.SeenTime).
func (d *Datastore) loadHostFamily(
	ctx context.Context,
	fam cacheFamily,
	key string,
	dbLoad func(context.Context, string) (*fleet.Host, error),
) (*fleet.Host, error) {
	if !d.hostCacheEnabled {
		return dbLoad(ctx, key)
	}

	if host, result := d.hostCacheGetFamily(ctx, fam, key); result == hostCacheLookupHit {
		return host, nil
	} else if result == hostCacheLookupNegative {
		return nil, ctxerr.Wrap(ctx, common_mysql.NotFound("Host"))
	}

	ch := d.hostCacheSF.DoChan(fam.sfPrefix+key, func() (any, error) {
		// flightCtx and cancel are scoped to the closure (i.e., to the actual DB call), NOT to the
		// surrounding loadHostFamily call. If they were declared outside, `defer cancel()` would fire when
		// the originating caller bails (e.g., on its own ctx cancellation), which would in turn cancel
		// flightCtx and abort the shared DB call, poisoning every other caller waiting on the same flight.
		// Scoping them here ties cancel() to the lifetime of the actual work.
		flightCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), hostCacheFlightTimeout)
		defer cancel()

		h, derr := dbLoad(flightCtx, key)
		switch {
		case derr == nil && h != nil:
			d.hostCachePutFamily(flightCtx, fam, h)
		case fleet.IsNotFound(derr):
			d.hostCachePutNotFoundFamily(flightCtx, fam, key)
			// Other (transient) errors are intentionally not cached, retry on next call.
		}
		return h, derr
	})
	var v any
	var err error
	select {
	case r := <-ch:
		v, err = r.Val, r.Err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	h, _ := v.(*fleet.Host)
	if h == nil {
		// Inner returned (nil, nil); shouldn't happen but don't hand out a nil pointer that downstream code
		// will dereference.
		return nil, ctxerr.Wrap(ctx, common_mysql.NotFound("Host"))
	}
	// Shallow-copy so concurrent callers that joined the same singleflight don't race on mutations like
	// host.SeenTime = now(). Cannot use Go 1.26 new(expr) form here because the leading `*` in `*h` is
	// ambiguous with pointer-type syntax in the parser.
	clone := *h
	return &clone, nil
}

// LoadHostByNodeKey overrides the inner Datastore's LoadHostByNodeKey to serve from the Redis cache when
// populated. The osquery-family wrapper of loadHostFamily; see loadHostFamily for full semantics.
func (d *Datastore) LoadHostByNodeKey(ctx context.Context, nodeKey string) (*fleet.Host, error) {
	return d.loadHostFamily(ctx, osqueryCacheFamily, nodeKey, d.Datastore.LoadHostByNodeKey)
}

// LoadHostByOrbitNodeKey is the orbit-family wrapper of loadHostFamily. The additional fields
// LoadHostByOrbitNodeKey's SELECT returns (DEP assignment, disk encryption state, encryption key availability,
// team name) ride along automatically via the embedded fleet.Host in the cache envelope.
func (d *Datastore) LoadHostByOrbitNodeKey(ctx context.Context, orbitNodeKey string) (*fleet.Host, error) {
	return d.loadHostFamily(ctx, orbitCacheFamily, orbitNodeKey, d.Datastore.LoadHostByOrbitNodeKey)
}
