package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashHostFilterDeterministic(t *testing.T) {
	t.Run("slice order and duplicates don't change the key", func(t *testing.T) {
		a := &types.HostFilter{
			LabelIDs:       []uint{3, 1, 2},
			Platforms:      []string{"windows", "darwin"},
			IncludeHostIDs: []uint{9, 7},
			ExcludeHostIDs: []uint{5, 4},
		}
		b := &types.HostFilter{
			LabelIDs:       []uint{1, 2, 3},
			Platforms:      []string{"darwin", "windows"},
			IncludeHostIDs: []uint{7, 9},
			ExcludeHostIDs: []uint{4, 5},
		}
		assert.Equal(t, hashHostFilter(a), hashHostFilter(b))
	})

	t.Run("team distinguishes nil, zero, and non-zero", func(t *testing.T) {
		zero := uint(0)
		five := uint(5)
		keys := map[string]struct{}{
			hashHostFilter(&types.HostFilter{}):                {},
			hashHostFilter(&types.HostFilter{TeamID: &zero}):   {},
			hashHostFilter(&types.HostFilter{TeamID: &five}):   {},
		}
		assert.Len(t, keys, 3, "nil vs team=0 vs team=5 must produce distinct keys")
	})

	t.Run("label vs include collision guard", func(t *testing.T) {
		// Without a separator, labels=[1,2] + include=[] could collide with
		// labels=[] + include=[1,2] if the key was naively concatenated.
		a := &types.HostFilter{LabelIDs: []uint{1, 2}}
		b := &types.HostFilter{IncludeHostIDs: []uint{1, 2}}
		assert.NotEqual(t, hashHostFilter(a), hashHostFilter(b))
	})
}

func TestHostFilterCacheServesFromCacheUntilTTL(t *testing.T) {
	cache := newHostFilterCache(10 * time.Second)

	// Override the clock so TTL behavior is deterministic.
	var now atomic.Int64
	now.Store(time.Now().UnixNano())
	cache.clock = func() time.Time { return time.Unix(0, now.Load()) }

	var calls atomic.Int32
	fetch := func(_ context.Context) ([]byte, error) {
		calls.Add(1)
		return []byte{0x0F}, nil
	}

	filter := &types.HostFilter{LabelIDs: []uint{1}}
	for range 5 {
		b, err := cache.Get(t.Context(), filter, fetch)
		require.NoError(t, err)
		assert.Equal(t, []byte{0x0F}, b)
	}
	assert.Equal(t, int32(1), calls.Load(), "repeated gets within TTL should hit the cache")

	// Advance past TTL.
	now.Add(int64(11 * time.Second))
	_, err := cache.Get(t.Context(), filter, fetch)
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load(), "expired entry should trigger a refetch")
}

func TestHostFilterCacheDistinctFiltersMissSeparately(t *testing.T) {
	cache := newHostFilterCache(time.Minute)

	var calls atomic.Int32
	fetch := func(_ context.Context) ([]byte, error) {
		calls.Add(1)
		return []byte{0xFF}, nil
	}

	teamA := uint(1)
	teamB := uint(2)
	_, err := cache.Get(t.Context(), &types.HostFilter{TeamID: &teamA}, fetch)
	require.NoError(t, err)
	_, err = cache.Get(t.Context(), &types.HostFilter{TeamID: &teamB}, fetch)
	require.NoError(t, err)

	assert.Equal(t, int32(2), calls.Load(), "different filter keys should each trigger a fetch")
}

func TestHostFilterCacheSingleflightCoalescesConcurrentMisses(t *testing.T) {
	cache := newHostFilterCache(time.Minute)

	var calls atomic.Int32
	unblock := make(chan struct{})
	fetch := func(_ context.Context) ([]byte, error) {
		calls.Add(1)
		<-unblock // hold the fetch until all goroutines are parked on singleflight
		return []byte{0x01}, nil
	}

	filter := &types.HostFilter{LabelIDs: []uint{42}}

	var wg sync.WaitGroup
	const goroutines = 20
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			b, err := cache.Get(t.Context(), filter, fetch)
			assert.NoError(t, err)
			assert.Equal(t, []byte{0x01}, b)
		}()
	}

	// Give the goroutines a moment to all reach Get; then release the fetch.
	time.Sleep(20 * time.Millisecond)
	close(unblock)
	wg.Wait()

	assert.Equal(t, int32(1), calls.Load(), "singleflight should coalesce concurrent misses")
}

func TestHostFilterCacheDoesNotCacheErrors(t *testing.T) {
	cache := newHostFilterCache(time.Minute)

	var calls atomic.Int32
	sentinel := errors.New("boom")
	fetch := func(_ context.Context) ([]byte, error) {
		calls.Add(1)
		return nil, sentinel
	}

	filter := &types.HostFilter{}
	_, err := cache.Get(t.Context(), filter, fetch)
	require.ErrorIs(t, err, sentinel)
	_, err = cache.Get(t.Context(), filter, fetch)
	require.ErrorIs(t, err, sentinel)

	assert.Equal(t, int32(2), calls.Load(), "failed fetches must not poison the cache")
}
