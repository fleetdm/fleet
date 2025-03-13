package mysql

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAsyncLastSeen(t *testing.T) {
	t.Skip("Skipping flaky test. Re-enable after fixing #24652")
	t.Parallel()

	runLoopAndWait := func(t *testing.T, als *asyncLastSeen) (ctx context.Context, stop func()) {
		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		go func() {
			als.runFlushLoop(ctx)
			// runFlushLoop should return once the context is closed
			close(done)
		}()

		return ctx, func() {
			cancel()
			select {
			case <-done:
				// ok
			case <-time.After(100 * time.Millisecond):
				t.Fatal("runFlushLoop did not return")
			}
		}
	}

	t.Run("always empty", func(t *testing.T) {
		t.Parallel()

		als := newAsyncLastSeen(time.Millisecond, 1, func(ctx context.Context, ids []string) {
			t.Fatal("unexpected call to fn")
		})
		_, stop := runLoopAndWait(t, als)

		time.Sleep(100 * time.Millisecond)
		stop()
	})

	t.Run("timed flush", func(t *testing.T) {
		t.Parallel()

		var mu sync.Mutex
		var gotIDs []string
		als := newAsyncLastSeen(10*time.Millisecond, 10, func(ctx context.Context, ids []string) {
			mu.Lock()
			defer mu.Unlock()

			// always add a "|" between calls
			if len(gotIDs) > 0 {
				gotIDs = append(gotIDs, "|")
			}
			gotIDs = append(gotIDs, ids...)
		})
		ctx, stop := runLoopAndWait(t, als)

		als.markHostSeen(ctx, "1")
		als.markHostSeen(ctx, "2")
		time.Sleep(100 * time.Millisecond) // oversleep to avoid slow timers issues on CI
		als.markHostSeen(ctx, "3")
		time.Sleep(100 * time.Millisecond) // oversleep to avoid slow timers issues on CI
		als.markHostSeen(ctx, "4")
		als.markHostSeen(ctx, "5")
		als.markHostSeen(ctx, "6")
		time.Sleep(100 * time.Millisecond) // oversleep to avoid slow timers issues on CI

		stop()

		mu.Lock()
		defer mu.Unlock()
		require.Equal(t, "12|3|456", strings.Join(gotIDs, ""))
	})

	t.Run("cap flush", func(t *testing.T) {
		t.Parallel()

		var mu sync.Mutex
		var gotIDs []string
		als := newAsyncLastSeen(100*time.Millisecond, 2, func(ctx context.Context, ids []string) {
			mu.Lock()
			defer mu.Unlock()

			// always add a "|" between calls
			if len(gotIDs) > 0 {
				gotIDs = append(gotIDs, "|")
			}
			gotIDs = append(gotIDs, ids...)
		})
		ctx, stop := runLoopAndWait(t, als)

		als.markHostSeen(ctx, "1")
		als.markHostSeen(ctx, "2")
		als.markHostSeen(ctx, "3")
		als.markHostSeen(ctx, "4")
		als.markHostSeen(ctx, "5")
		als.markHostSeen(ctx, "6")

		stop()

		mu.Lock()
		defer mu.Unlock()
		require.Equal(t, "12|34|56", strings.Join(gotIDs, ""))
	})

	t.Run("cap and timed flush", func(t *testing.T) {
		t.Parallel()

		var mu sync.Mutex
		var gotIDs []string
		als := newAsyncLastSeen(10*time.Millisecond, 3, func(ctx context.Context, ids []string) {
			mu.Lock()
			defer mu.Unlock()

			// always add a "|" between calls
			if len(gotIDs) > 0 {
				gotIDs = append(gotIDs, "|")
			}
			gotIDs = append(gotIDs, ids...)
		})
		ctx, stop := runLoopAndWait(t, als)

		als.markHostSeen(ctx, "1")
		als.markHostSeen(ctx, "2")
		als.markHostSeen(ctx, "3")
		als.markHostSeen(ctx, "4")
		time.Sleep(100 * time.Millisecond) // oversleep to avoid slow timers issues on CI
		als.markHostSeen(ctx, "5")
		time.Sleep(100 * time.Millisecond) // oversleep to avoid slow timers issues on CI
		als.markHostSeen(ctx, "6")
		time.Sleep(100 * time.Millisecond) // oversleep to avoid slow timers issues on CI

		stop()

		mu.Lock()
		defer mu.Unlock()
		require.Equal(t, "123|4|5|6", strings.Join(gotIDs, ""))
	})
}
