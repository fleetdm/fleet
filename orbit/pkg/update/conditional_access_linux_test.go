//go:build linux

package update

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRunner() (*ConditionalAccessRunner, *int, *sync.Mutex) {
	r := NewConditionalAccessRunner("dir", "https://fleet.example.com", "secret", "uuid", "", false, zerolog.Nop())
	count := 0
	mu := &sync.Mutex{}
	r.enrollFn = func(ctx context.Context, metadataDir, scepURL, challenge, uuid, rootCA string, insecure bool, logger zerolog.Logger) error {
		mu.Lock()
		count++
		mu.Unlock()
		return nil
	}
	return r, &count, mu
}

func TestConditionalAccessRunner_NotificationFalse(t *testing.T) {
	r, count, mu := newTestRunner()

	cfg := &fleet.OrbitConfig{}
	cfg.Notifications.RunConditionalAccessEnrollment = false
	require.NoError(t, r.Run(cfg))

	// Give any goroutine a moment to run (there should be none).
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 0, *count, "enrollFn must not be called when notification is false")
}

func TestConditionalAccessRunner_NotificationTrue(t *testing.T) {
	r, count, mu := newTestRunner()

	// Add a brief sleep so we can detect the goroutine finishing.
	done := make(chan struct{})
	r.enrollFn = func(ctx context.Context, metadataDir, scepURL, challenge, uuid, rootCA string, insecure bool, logger zerolog.Logger) error {
		mu.Lock()
		*count++
		mu.Unlock()
		close(done)
		return nil
	}

	cfg := &fleet.OrbitConfig{}
	cfg.Notifications.RunConditionalAccessEnrollment = true
	require.NoError(t, r.Run(cfg))

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("enrollFn was not called within 2s")
	}

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, *count)
}

func TestConditionalAccessRunner_IdempotentWhileRunning(t *testing.T) {
	r, count, mu := newTestRunner()

	started := make(chan struct{})
	unblock := make(chan struct{})
	r.enrollFn = func(ctx context.Context, metadataDir, scepURL, challenge, uuid, rootCA string, insecure bool, logger zerolog.Logger) error {
		mu.Lock()
		*count++
		mu.Unlock()
		close(started)
		<-unblock
		return nil
	}

	cfg := &fleet.OrbitConfig{}
	cfg.Notifications.RunConditionalAccessEnrollment = true

	// First call starts the goroutine.
	require.NoError(t, r.Run(cfg))
	<-started // wait until goroutine is inside enrollFn

	// Second call while goroutine is running — should be no-op.
	require.NoError(t, r.Run(cfg))

	close(unblock) // let goroutine finish

	// Brief pause to let goroutine exit.
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, *count, "enrollFn must only be called once despite two Run() calls")
}
