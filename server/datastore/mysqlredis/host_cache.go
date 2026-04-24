package mysqlredis

import (
	"context"
	"encoding/json"
	"errors"
	mathrand "math/rand/v2"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	redigo "github.com/gomodule/redigo/redis"
)

// All host-cache keys live under this single versioned prefix so operators can
// purge with `redis-cli --scan --pattern 'fleet:hostcache:v1:*' | xargs redis-cli DEL`.
// Bumping the version on a cached-payload schema change orphans old keys; they
// TTL out within hostCacheTTL.
const hostCacheKeyPrefix = "fleet:hostcache:v1"

const (
	// hostCacheNegativeTTL caps how long a "not found" result is cached. Short
	// because an enrollment can legitimately create a host with a node_key that
	// was just queried and missed.
	hostCacheNegativeTTL = 5 * time.Second

	// hostCacheTTLJitterFraction spreads entry expiry across a ±(fraction/2)
	// window around the configured base TTL, so a Redis restart or TTL-driven
	// wave doesn't trigger a synchronized stampede back to the reader.
	hostCacheTTLJitterFraction = 0.2
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

// jitteredHostCacheTTL returns the configured base TTL perturbed by
// ±(hostCacheTTLJitterFraction / 2). With the default 0.2 and a 60s base, the
// result falls in [54s, 66s] — yielding ~5 cache hits per miss at the default
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

// hostCacheGet looks up a host by node_key. It checks the positive cache first
// (the common case) and falls through to the negative cache only on positive
// miss. Never propagates Redis or JSON errors; any error is recorded and the
// caller sees a hostCacheLookupMiss.
func (d *Datastore) hostCacheGet(ctx context.Context, nodeKey string) (*fleet.Host, hostCacheLookup) {
	if !d.hostCacheEnabled || nodeKey == "" {
		return nil, hostCacheLookupMiss
	}

	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	raw, err := redigo.Bytes(conn.Do("GET", hostCacheKeyByNodeKey(nodeKey)))
	switch {
	case err == nil:
		entry := new(hostCacheEntry)
		if jerr := json.Unmarshal(raw, entry); jerr != nil {
			// Schema drift or a poisoned entry. Drop the bad key so the next
			// lookup repopulates from the database, and treat this call as a
			// miss.
			d.recordHostCacheErr(ctx, "get", jerr)
			if _, derr := conn.Do("DEL", hostCacheKeyByNodeKey(nodeKey)); derr != nil {
				d.recordHostCacheErr(ctx, "del", derr)
			}
			d.recordHostCacheLookup(ctx, "miss")
			return nil, hostCacheLookupMiss
		}
		d.recordHostCacheLookup(ctx, "hit")
		return entry.toHost(), hostCacheLookupHit

	case errors.Is(err, redigo.ErrNil):
		// positive miss; fall through to negative-cache probe

	default:
		d.recordHostCacheErr(ctx, "get", err)
		d.recordHostCacheLookup(ctx, "miss")
		return nil, hostCacheLookupMiss
	}

	_, err = redigo.Bytes(conn.Do("GET", hostCacheKeyMiss(nodeKey)))
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

// hostCachePut stores host under the positive-cache key and updates the reverse
// index so invalidation-by-ID can find the node_key later. Fire-and-forget:
// errors are recorded, not returned.
func (d *Datastore) hostCachePut(ctx context.Context, host *fleet.Host) {
	if !d.hostCacheEnabled || host == nil || host.NodeKey == nil || *host.NodeKey == "" {
		return
	}

	raw, err := json.Marshal(hostCacheEntryFromHost(host))
	if err != nil {
		d.recordHostCacheErr(ctx, "set", err)
		return
	}

	ttl := d.jitteredHostCacheTTL()
	ttlSec := int(ttl.Seconds())
	if ttlSec <= 0 {
		ttlSec = 1
	}

	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	// Write the reverse index BEFORE the payload. If both succeed, the cache
	// is consistent. If the index SET fails, no payload is written, so
	// nothing is stranded. If the payload SET fails after the index succeeds,
	// the orphaned index points at a non-existent key — the next read takes
	// the DB path, and any subsequent hostCacheDeleteByID call cleans the
	// stranded index. The problematic state (payload present, index missing,
	// so hostCacheDeleteByID silently no-ops and leaves the payload alive
	// until TTL) cannot occur with this ordering.
	if host.ID != 0 {
		if _, err := conn.Do("SET", hostCacheIndexByID(host.ID), *host.NodeKey, "EX", ttlSec); err != nil {
			d.recordHostCacheErr(ctx, "set", err)
			return
		}
	}
	if _, err := conn.Do("SET", hostCacheKeyByNodeKey(*host.NodeKey), raw, "EX", ttlSec); err != nil {
		d.recordHostCacheErr(ctx, "set", err)
	}
}

// hostCachePutNotFound stores a short-lived negative-cache entry. Used when the
// database returns NotFound for a node_key.
func (d *Datastore) hostCachePutNotFound(ctx context.Context, nodeKey string) {
	if !d.hostCacheEnabled || nodeKey == "" {
		return
	}

	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("SET", hostCacheKeyMiss(nodeKey), "1", "EX", int(hostCacheNegativeTTL.Seconds())); err != nil {
		d.recordHostCacheErr(ctx, "set", err)
	}
}

// hostCacheDeleteByNodeKey invalidates the primary, negative, and index keys
// when the caller already has the (nodeKey, hostID) pair. Pass hostID=0 if the
// caller does not know the ID; the index is skipped in that case.
// `reason` is a low-cardinality label recorded on the invalidations counter.
//
// When hostID > 0 and the reverse index points at a DIFFERENT (prior) key —
// e.g., osquery re-enrollment rotated node_key from OLD to NEW and we're
// called with NEW — the entry under the OLD key is also cleared. Without
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

// hostCacheGetByOrbitNodeKey is the orbit-side counterpart of hostCacheGet.
// Same tri-state semantics: hit returns the decoded host, negative means
// "cached NotFound", miss means caller should fall through to the DB. Redis
// and JSON errors are recorded and treated as miss.
func (d *Datastore) hostCacheGetByOrbitNodeKey(ctx context.Context, orbitNodeKey string) (*fleet.Host, hostCacheLookup) {
	if !d.hostCacheEnabled || orbitNodeKey == "" {
		return nil, hostCacheLookupMiss
	}

	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	raw, err := redigo.Bytes(conn.Do("GET", hostCacheKeyByOrbitNodeKey(orbitNodeKey)))
	switch {
	case err == nil:
		entry := new(orbitHostCacheEntry)
		if jerr := json.Unmarshal(raw, entry); jerr != nil {
			d.recordHostCacheErr(ctx, "get", jerr)
			if _, derr := conn.Do("DEL", hostCacheKeyByOrbitNodeKey(orbitNodeKey)); derr != nil {
				d.recordHostCacheErr(ctx, "del", derr)
			}
			d.recordHostCacheLookup(ctx, "miss")
			return nil, hostCacheLookupMiss
		}
		d.recordHostCacheLookup(ctx, "hit")
		return entry.toHost(), hostCacheLookupHit
	case errors.Is(err, redigo.ErrNil):
		// positive miss; fall through to negative cache
	default:
		d.recordHostCacheErr(ctx, "get", err)
		d.recordHostCacheLookup(ctx, "miss")
		return nil, hostCacheLookupMiss
	}

	_, err = redigo.Bytes(conn.Do("GET", hostCacheKeyOrbitMiss(orbitNodeKey)))
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

// hostCachePutByOrbit stores host under the orbit positive-cache key and
// updates the orbit reverse index. Fire-and-forget errors.
func (d *Datastore) hostCachePutByOrbit(ctx context.Context, host *fleet.Host) {
	if !d.hostCacheEnabled || host == nil || host.OrbitNodeKey == nil || *host.OrbitNodeKey == "" {
		return
	}

	raw, err := json.Marshal(orbitHostCacheEntryFromHost(host))
	if err != nil {
		d.recordHostCacheErr(ctx, "set", err)
		return
	}

	ttl := d.jitteredHostCacheTTL()
	ttlSec := int(ttl.Seconds())
	if ttlSec <= 0 {
		ttlSec = 1
	}

	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	// Index first, payload second — see hostCachePut for rationale.
	if host.ID != 0 {
		if _, err := conn.Do("SET", hostCacheOrbitIndexByID(host.ID), *host.OrbitNodeKey, "EX", ttlSec); err != nil {
			d.recordHostCacheErr(ctx, "set", err)
			return
		}
	}
	if _, err := conn.Do("SET", hostCacheKeyByOrbitNodeKey(*host.OrbitNodeKey), raw, "EX", ttlSec); err != nil {
		d.recordHostCacheErr(ctx, "set", err)
	}
}

// hostCachePutNotFoundByOrbitNodeKey stores a short-lived negative-cache entry
// on the orbit side. Mirrors hostCachePutNotFound.
func (d *Datastore) hostCachePutNotFoundByOrbitNodeKey(ctx context.Context, orbitNodeKey string) {
	if !d.hostCacheEnabled || orbitNodeKey == "" {
		return
	}
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()
	if _, err := conn.Do("SET", hostCacheKeyOrbitMiss(orbitNodeKey), "1", "EX", int(hostCacheNegativeTTL.Seconds())); err != nil {
		d.recordHostCacheErr(ctx, "set", err)
	}
}

// hostCacheDeleteByID invalidates both the osquery and orbit caches when only
// the host ID is known. It reads both reverse indices (id2nk, id2onk) and
// deletes whichever entries it finds. Either or both may be missing (host
// enrolled only one agent, TTL expiry, never populated) — missing entries
// mean there's nothing more to do on that side, and the invalidation is still
// counted so the metrics line up with write-path activity.
func (d *Datastore) hostCacheDeleteByID(ctx context.Context, hostID uint, reason string) {
	if !d.hostCacheEnabled || hostID == 0 {
		return
	}

	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	d.clearNodeKeyEntriesByID(ctx, conn, hostID)
	d.clearOrbitNodeKeyEntriesByID(ctx, conn, hostID)
	d.recordHostCacheInvalidation(ctx, reason)
}

// clearNodeKeyEntriesByID looks up id2nk for the given host and deletes the
// osquery-side entries. Silent-no-op when the reverse index is missing.
func (d *Datastore) clearNodeKeyEntriesByID(ctx context.Context, conn redigo.Conn, hostID uint) {
	nodeKey, err := redigo.String(conn.Do("GET", hostCacheIndexByID(hostID)))
	switch {
	case errors.Is(err, redigo.ErrNil):
		return
	case err != nil:
		d.recordHostCacheErr(ctx, "get", err)
		return
	}
	for _, k := range []string{
		hostCacheKeyByNodeKey(nodeKey),
		hostCacheKeyMiss(nodeKey),
		hostCacheIndexByID(hostID),
	} {
		if _, err := conn.Do("DEL", k); err != nil {
			d.recordHostCacheErr(ctx, "del", err)
		}
	}
}

// clearOrbitNodeKeyEntriesByID is the orbit-side analog of
// clearNodeKeyEntriesByID: resolves id2onk and DELs the orbit entries.
func (d *Datastore) clearOrbitNodeKeyEntriesByID(ctx context.Context, conn redigo.Conn, hostID uint) {
	orbitNodeKey, err := redigo.String(conn.Do("GET", hostCacheOrbitIndexByID(hostID)))
	switch {
	case errors.Is(err, redigo.ErrNil):
		return
	case err != nil:
		d.recordHostCacheErr(ctx, "get", err)
		return
	}
	for _, k := range []string{
		hostCacheKeyByOrbitNodeKey(orbitNodeKey),
		hostCacheKeyOrbitMiss(orbitNodeKey),
		hostCacheOrbitIndexByID(hostID),
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
// the host row existed — the reverse-index path can't find those entries
// (no id2 mapping is populated for a negative hit), so they must be cleared
// by key. Does NOT record an invalidation; the caller already did.
func (d *Datastore) hostCacheClearDirectEntries(ctx context.Context, nodeKey, orbitNodeKey string) {
	if !d.hostCacheEnabled || (nodeKey == "" && orbitNodeKey == "") {
		return
	}
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	if nodeKey != "" {
		for _, k := range []string{hostCacheKeyByNodeKey(nodeKey), hostCacheKeyMiss(nodeKey)} {
			if _, err := conn.Do("DEL", k); err != nil {
				d.recordHostCacheErr(ctx, "del", err)
			}
		}
	}
	if orbitNodeKey != "" {
		for _, k := range []string{hostCacheKeyByOrbitNodeKey(orbitNodeKey), hostCacheKeyOrbitMiss(orbitNodeKey)} {
			if _, err := conn.Do("DEL", k); err != nil {
				d.recordHostCacheErr(ctx, "del", err)
			}
		}
	}
}

// hostCacheInvalidateBatchSize caps the number of keys per pipelined MGET /
// DEL call. Keeps individual Redis commands bounded regardless of input size.
const hostCacheInvalidateBatchSize = 500

// invalidateHostIDs efficiently invalidates both cache families for a batch
// of host IDs. Equivalent to calling hostCacheDeleteByID in a loop but uses
// pipelined MGET + variadic DEL to collapse what would otherwise be ~8
// sequential round-trips per host into O(slots × chunks) round-trips for
// the whole batch — at 10k hosts on a loaded Redis, that's the difference
// between ~80 seconds and ~200 milliseconds. Records one invalidation
// counter bump per input ID. Errors are recorded on the errors counter and
// logged; TTL is the safety net for any keys that survived a transient
// Redis failure.
func (d *Datastore) invalidateHostIDs(ctx context.Context, ids []uint, reason string) {
	if !d.hostCacheEnabled || len(ids) == 0 {
		return
	}

	// Phase 1: batch-GET every id2nk and id2onk to discover which payloads
	// to clear. Interleaved (id2nk, id2onk, id2nk, id2onk, ...) so the
	// result[2*i] / result[2*i+1] pairing is unambiguous below.
	idxKeys := make([]string, 0, 2*len(ids))
	for _, id := range ids {
		idxKeys = append(idxKeys, hostCacheIndexByID(id))
		idxKeys = append(idxKeys, hostCacheOrbitIndexByID(id))
	}
	resolved := d.pipelinedMGET(ctx, idxKeys)

	// Phase 2: build the full DEL key list (payload + negative + reverse
	// index for whichever side was populated) and issue pipelined DELs.
	delKeys := make([]string, 0, 6*len(ids))
	for i, id := range ids {
		if nk := resolved[2*i]; nk != "" {
			delKeys = append(delKeys, hostCacheKeyByNodeKey(nk), hostCacheKeyMiss(nk))
		}
		if onk := resolved[2*i+1]; onk != "" {
			delKeys = append(delKeys, hostCacheKeyByOrbitNodeKey(onk), hostCacheKeyOrbitMiss(onk))
		}
		delKeys = append(delKeys, hostCacheIndexByID(id), hostCacheOrbitIndexByID(id))
	}
	d.pipelinedDEL(ctx, delKeys)

	// Phase 3: one invalidation counter bump per id, matching the
	// per-host-invalidation semantics of the non-batched path.
	for range ids {
		d.recordHostCacheInvalidation(ctx, reason)
	}
}

// pipelinedMGET returns the string value for each key in input order. Empty
// string for missing keys or if an error prevented retrieval. In Redis
// cluster mode, keys are grouped by slot (CROSSSLOT-safe) and one MGET is
// issued per slot group; each group is further chunked to bound individual
// command size.
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
	// BindConn MUST come before ConfigureDoer — redisc's BindConn expects the
	// raw cluster conn from the pool, not a RetryConn wrapper. See
	// server/datastore/redis/ratelimit_store.go for the canonical ordering.
	// In standalone mode BindConn is a no-op.
	if err := redis.BindConn(d.pool, conn, chunk...); err != nil {
		d.recordHostCacheErr(ctx, "get", err)
		return
	}
	conn = redis.ConfigureDoer(d.pool, conn)
	args := redigo.Args{}.AddFlat(chunk)
	values, err := redigo.Values(conn.Do("MGET", args...))
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

// pipelinedDEL issues variadic DEL across all keys, slot-grouped and chunked.
// All errors are recorded and treated as best-effort; TTL is the backstop
// for any keys we failed to DEL.
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
	// BindConn before ConfigureDoer — see mgetChunk for the rationale.
	if err := redis.BindConn(d.pool, conn, chunk...); err != nil {
		d.recordHostCacheErr(ctx, "del", err)
		return
	}
	conn = redis.ConfigureDoer(d.pool, conn)
	args := redigo.Args{}.AddFlat(chunk)
	if _, err := conn.Do("DEL", args...); err != nil {
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
	//nolint:nilaway // initialized in package init(); panic on registration failure guarantees non-nil
	hostCacheInvalidations.Add(ctx, 1, hostCacheInvalidationAttrs(reason))
}

// LoadHostByNodeKey overrides the inner Datastore's LoadHostByNodeKey to serve
// from the Redis cache when populated. On miss it falls through to the inner
// Datastore under a singleflight guard so a thundering herd of N concurrent
// misses for the same node_key collapses into a single DB call.
//
// Semantics:
//   - Cache disabled, or ctxdb.IsHostCacheBypassed(ctx): always delegate; never
//     read or write the cache. Use BypassHostCache after a write if you need
//     read-your-writes freshness within the TTL window.
//   - Positive cache hit: return cached host.
//   - Negative cache hit: return a NotFoundError without hitting the DB.
//   - Miss: singleflight-guarded DB fetch; on success populate positive cache,
//     on NotFound populate negative cache; propagate other errors without
//     populating either cache (transient failures must not poison the cache).
//
// Callers receive a host that is safe to mutate: cache-hit path returns a
// fresh struct from JSON unmarshal, singleflight path returns a shallow copy
// so concurrent callers that joined the same flight don't race on each
// other's mutations (e.g., AuthenticateHost overwrites host.SeenTime).
func (d *Datastore) LoadHostByNodeKey(ctx context.Context, nodeKey string) (*fleet.Host, error) {
	if !d.hostCacheEnabled || ctxdb.IsHostCacheBypassed(ctx) {
		return d.Datastore.LoadHostByNodeKey(ctx, nodeKey)
	}

	if host, result := d.hostCacheGet(ctx, nodeKey); result == hostCacheLookupHit {
		return host, nil
	} else if result == hostCacheLookupNegative {
		return nil, ctxerr.Wrap(ctx, common_mysql.NotFound("Host"))
	}

	// Detach the inner call's context from the initiating caller's
	// cancellation AND deadline. Without the detach, if caller A starts the
	// flight and A's request is canceled or its deadline expires (e.g.,
	// A has a tight 50ms LB timeout), the DB call is aborted and every
	// joiner — including those with much more generous deadlines — receives
	// A's error. We rely on the Redis pool's read_timeout / connect_timeout
	// and the MySQL driver's own timeouts to bound the flight duration,
	// which is what every non-cached call is already bounded by today.
	flightCtx := context.WithoutCancel(ctx)

	// Prefix the singleflight key so osquery and orbit flights can't collide
	// if a node_key ever happened to equal an orbit_node_key (astronomically
	// unlikely with 32-char random keys but cheap to defend against). Also
	// helps telemetry/debugging distinguish the two flight populations.
	//
	// DoChan lets the caller abandon the wait if its own ctx is canceled
	// without affecting the shared flight — the flight runs under flightCtx
	// (no cancellation inherited from the initiating caller) so peers that
	// joined the same flight still receive the result. Using plain Do would
	// block the canceled caller until the flight completes, burning caller
	// resources on a result it no longer needs.
	ch := d.hostCacheSF.DoChan("nk:"+nodeKey, func() (any, error) {
		h, derr := d.Datastore.LoadHostByNodeKey(flightCtx, nodeKey)
		switch {
		case derr == nil && h != nil:
			d.hostCachePut(flightCtx, h)
		case fleet.IsNotFound(derr):
			d.hostCachePutNotFound(flightCtx, nodeKey)
			// Other (transient) errors are intentionally not cached — retry on next call.
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
		// Inner returned (nil, nil); shouldn't happen but don't hand out a nil
		// pointer that downstream code will dereference.
		return nil, ctxerr.Wrap(ctx, common_mysql.NotFound("Host"))
	}
	// Shallow-copy so concurrent callers that joined the same singleflight
	// don't race on mutations like host.SeenTime = now().
	clone := *h
	return &clone, nil
}

// LoadHostByOrbitNodeKey is the orbit-side counterpart of LoadHostByNodeKey.
// Identical semantics (cache-aside, singleflight, ctx detach, shallow clone)
// but targets the orbit_node_key column and the orbitHostCacheEntry shape,
// which carries the additional fields LoadHostByOrbitNodeKey's SELECT returns
// (DEP assignment, disk encryption state, encryption key availability,
// team name).
func (d *Datastore) LoadHostByOrbitNodeKey(ctx context.Context, orbitNodeKey string) (*fleet.Host, error) {
	if !d.hostCacheEnabled || ctxdb.IsHostCacheBypassed(ctx) {
		return d.Datastore.LoadHostByOrbitNodeKey(ctx, orbitNodeKey)
	}

	if host, result := d.hostCacheGetByOrbitNodeKey(ctx, orbitNodeKey); result == hostCacheLookupHit {
		return host, nil
	} else if result == hostCacheLookupNegative {
		return nil, ctxerr.Wrap(ctx, common_mysql.NotFound("Host"))
	}

	// See LoadHostByNodeKey for the rationale on detaching without
	// reapplying the caller's deadline.
	flightCtx := context.WithoutCancel(ctx)

	// DoChan lets the caller bail on cancellation without blocking; see
	// LoadHostByNodeKey for the full rationale.
	ch := d.hostCacheSF.DoChan("onk:"+orbitNodeKey, func() (any, error) {
		h, derr := d.Datastore.LoadHostByOrbitNodeKey(flightCtx, orbitNodeKey)
		switch {
		case derr == nil && h != nil:
			d.hostCachePutByOrbit(flightCtx, h)
		case fleet.IsNotFound(derr):
			d.hostCachePutNotFoundByOrbitNodeKey(flightCtx, orbitNodeKey)
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
		return nil, ctxerr.Wrap(ctx, common_mysql.NotFound("Host"))
	}
	clone := *h
	return &clone, nil
}
