package service

import (
	"context"
	"errors"
	"html/template"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

func TestCustomWindowsTOSContent(t *testing.T) {
	// windowsTOSCache is package state, so this test resets it and is not parallel. Each step builds on the state the
	// one before left behind, so they run as one sequence.
	windowsTOSCache.clear()
	t.Cleanup(windowsTOSCache.clear)

	ds := new(mock.Store)
	svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler)}
	ctx := license.NewContext(t.Context(), &fleet.LicenseInfo{Tier: fleet.TierPremium})

	var metaErr error
	uploadID, doc := "upload-1", "# Terms\n"
	ds.MDMGetEULAMetadataFunc = func(ctx context.Context, platform fleet.MDMEULAPlatform) (*fleet.MDMEULA, error) {
		if metaErr != nil {
			return nil, metaErr
		}
		return &fleet.MDMEULA{Token: uploadID, Platform: platform}, nil
	}
	var loads atomic.Int32
	var loadErr error
	release := make(chan struct{})
	ds.MDMGetEULABytesFunc = func(ctx context.Context, platform fleet.MDMEULAPlatform, token string) (*fleet.MDMEULA, error) {
		loads.Add(1)
		if _, ok := ctx.Deadline(); !ok {
			return nil, errors.New("the shared load must be bounded")
		}
		<-release
		if loadErr != nil {
			return nil, loadErr
		}
		if token != uploadID {
			return nil, newNotFoundError()
		}
		return &fleet.MDMEULA{Platform: platform, Bytes: []byte(doc)}, nil
	}

	// Concurrent misses share one load and render. Without a shared render, the waiting requests would each start a
	// load.
	var wg sync.WaitGroup
	results := make(chan template.HTML, 10)
	for range 10 {
		wg.Go(func() { results <- svc.customWindowsTOSContent(ctx) })
	}
	require.Eventually(t, func() bool { return loads.Load() == 1 }, time.Second, 10*time.Millisecond)
	require.Never(t, func() bool { return loads.Load() > 1 }, 200*time.Millisecond, 10*time.Millisecond)
	close(release)
	wg.Wait()
	close(results)
	for got := range results {
		require.Contains(t, string(got), "<h1>Terms</h1>")
	}
	require.EqualValues(t, 1, loads.Load())

	// Cached content needs no load.
	require.Contains(t, string(svc.customWindowsTOSContent(ctx)), "<h1>Terms</h1>")
	require.EqualValues(t, 1, loads.Load())

	// A database error keeps the cached agreement.
	metaErr = errors.New("connection refused")
	require.Contains(t, string(svc.customWindowsTOSContent(ctx)), "<h1>Terms</h1>")
	metaErr = nil

	// A document that fails to render is cached as empty, so it isn't loaded again.
	uploadID, doc = "upload-2", strings.Repeat("a", 9000) // over the line limit
	require.Empty(t, svc.customWindowsTOSContent(ctx))
	require.Empty(t, svc.customWindowsTOSContent(ctx))
	require.EqualValues(t, 2, loads.Load())

	// A deleted agreement clears the cache, so a later database error has nothing to fall back on.
	uploadID, doc = "upload-3", "# New terms\n"
	require.Contains(t, string(svc.customWindowsTOSContent(ctx)), "New terms")
	metaErr = newNotFoundError()
	require.Empty(t, svc.customWindowsTOSContent(ctx))
	metaErr = errors.New("connection refused")
	require.Empty(t, svc.customWindowsTOSContent(ctx))
	metaErr = nil

	// Fleet Free shows the default terms, and the downgrade clears the cache.
	require.NotEmpty(t, svc.customWindowsTOSContent(ctx))
	freeCtx := license.NewContext(t.Context(), &fleet.LicenseInfo{Tier: fleet.TierFree})
	require.Empty(t, svc.customWindowsTOSContent(freeCtx))
	metaErr = errors.New("connection refused")
	require.Empty(t, svc.customWindowsTOSContent(ctx))
	metaErr = nil

	// A database error while loading keeps the cached agreement.
	uploadID, doc = "upload-4", "# Current terms\n"
	require.Contains(t, string(svc.customWindowsTOSContent(ctx)), "Current terms")
	uploadID, loadErr = "upload-5", errors.New("connection refused")
	require.Contains(t, string(svc.customWindowsTOSContent(ctx)), "Current terms")
	loadErr = nil

	// An agreement deleted between the metadata read and the load shows the default terms.
	uploadID, loadErr = "upload-6", newNotFoundError()
	require.Empty(t, svc.customWindowsTOSContent(ctx))
	loadErr = nil

	// A request that stops waiting for a render gets the cached agreement.
	uploadID, doc = "upload-7", "# Earlier terms\n"
	require.Contains(t, string(svc.customWindowsTOSContent(ctx)), "Earlier terms")
	uploadID, doc, release = "upload-8", "# Later terms\n", make(chan struct{})
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	require.Contains(t, string(svc.customWindowsTOSContent(canceled)), "Earlier terms")
	// Let the shared render finish before the test ends, since the cache is package state.
	close(release)
	require.Eventually(t, func() bool {
		_, ok := windowsTOSCache.get("upload-8")
		return ok
	}, time.Second, 10*time.Millisecond)
}

func TestRenderedTOSCacheKeepsNewerContent(t *testing.T) {
	t.Parallel()
	var c renderedTOSCache

	// A render that started first but finishes last doesn't replace the newer agreement.
	older := c.generation()
	c.setIfUnchanged(c.generation(), "new", "new terms")
	c.setIfUnchanged(older, "old", "old terms")
	got, ok := c.get("new")
	require.True(t, ok)
	require.Equal(t, template.HTML("new terms"), got)
	require.Equal(t, template.HTML("new terms"), c.last())

	// Nor does it bring back an agreement deleted while it ran.
	started := c.generation()
	c.clear()
	c.setIfUnchanged(started, "new", "new terms")
	require.Empty(t, c.last())
}
