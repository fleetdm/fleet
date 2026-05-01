package service

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart/internal/types"
	"golang.org/x/sync/singleflight"
)

// hostFilterCacheTTL is how long a host-ID bitmap is served from cache before
// being recomputed. Kept well under the chart collection cadence (10m) so data
// and mask staleness stay roughly aligned.
const hostFilterCacheTTL = 60 * time.Second

// hostBitmapFetcher is the signature used by cache callers to compute a fresh
// bitmap on a miss. Returning an error bypasses caching for that call.
type hostBitmapFetcher func(ctx context.Context) ([]byte, error)

// hostFilterCache maps a canonicalized HostFilter to the bitmap of host IDs
// that match it. Entries are considered valid for ttl; concurrent misses for
// the same key are collapsed via singleflight. Expired entries are swept from
// the map opportunistically on each write, so stale keys (e.g. one-off
// IncludeHostIDs filters) don't accumulate indefinitely.
type hostFilterCache struct {
	ttl     time.Duration
	clock   func() time.Time
	sf      singleflight.Group
	mu      sync.RWMutex
	entries map[string]hostFilterCacheEntry
}

type hostFilterCacheEntry struct {
	bitmap    []byte
	expiresAt time.Time
}

func newHostFilterCache(ttl time.Duration) *hostFilterCache {
	return &hostFilterCache{
		ttl:     ttl,
		clock:   time.Now,
		entries: make(map[string]hostFilterCacheEntry),
	}
}

// Get returns the cached bitmap for the filter or computes a fresh one via
// fetch on miss/expiry. Concurrent misses for the same filter share one fetch.
func (c *hostFilterCache) Get(ctx context.Context, filter *types.HostFilter, fetch hostBitmapFetcher) ([]byte, error) {
	key := hashHostFilter(filter)

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if ok && c.clock().Before(entry.expiresAt) {
		return entry.bitmap, nil
	}

	val, err, _ := c.sf.Do(key, func() (any, error) {
		// Re-check after acquiring the singleflight slot: another goroutine
		// may have populated the cache while we were waiting.
		c.mu.RLock()
		entry, ok := c.entries[key]
		c.mu.RUnlock()
		if ok && c.clock().Before(entry.expiresAt) {
			return entry.bitmap, nil
		}

		bitmap, err := fetch(ctx)
		if err != nil {
			return nil, err
		}
		now := c.clock()
		c.mu.Lock()
		// Sweep expired entries so keys we never see again don't leak. Cheap
		// because this only runs on misses (already the slow path).
		for k, e := range c.entries {
			if !now.Before(e.expiresAt) {
				delete(c.entries, k)
			}
		}
		c.entries[key] = hostFilterCacheEntry{
			bitmap:    bitmap,
			expiresAt: now.Add(c.ttl),
		}
		c.mu.Unlock()
		return bitmap, nil
	})
	if err != nil {
		return nil, err
	}
	return val.([]byte), nil
}

// hashHostFilter produces a deterministic string key for a HostFilter. Slice
// fields are sorted and copied so caller mutations can't affect keying; a
// shared separator that can't appear in the encoded values keeps distinct
// filters from collapsing to the same key.
//
// TeamIDs specifically distinguishes nil from empty-non-nil — the two have
// different semantics (no filter vs match nothing) and must never share a
// cache entry.
func hashHostFilter(f *types.HostFilter) string {
	if f == nil {
		return "nil"
	}
	teams := slices.Clone(f.TeamIDs)
	slices.Sort(teams)
	labels := slices.Clone(f.LabelIDs)
	slices.Sort(labels)
	platforms := slices.Clone(f.Platforms)
	slices.Sort(platforms)
	include := slices.Clone(f.IncludeHostIDs)
	slices.Sort(include)
	exclude := slices.Clone(f.ExcludeHostIDs)
	slices.Sort(exclude)

	var b strings.Builder
	if f.TeamIDs == nil {
		b.WriteString("teams=nil")
	} else {
		fmt.Fprintf(&b, "teams=%v", teams)
	}
	fmt.Fprintf(&b, "|labels=%v|platforms=%s|include=%v|exclude=%v",
		labels, strings.Join(platforms, ","), include, exclude)
	return b.String()
}
