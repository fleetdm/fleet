package mdm

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	mockredis "github.com/fleetdm/fleet/v4/server/mock/redis"
	"github.com/stretchr/testify/require"
)

func TestEndUserAuthPrompt(t *testing.T) {
	t.Parallel()

	kv := mockredis.NewMemKeyValueStore()
	clk := clock.NewMockClock()
	ctx := t.Context()

	// Nothing recorded yet.
	pending, err := HasEndUserAuthPrompt(ctx, kv, "host-uuid-1", clk.Now())
	require.NoError(t, err)
	require.False(t, pending)

	require.NoError(t, RecordEndUserAuthPrompt(ctx, kv, "host-uuid-1", clk.Now()))

	raw, err := kv.Get(ctx, "eua_pending:host-uuid-1")
	require.NoError(t, err)
	require.NotNil(t, raw, "the record must be keyed on the exact host uuid")

	pending, err = HasEndUserAuthPrompt(ctx, kv, "host-uuid-1", clk.Now())
	require.NoError(t, err)
	require.True(t, pending)

	// A record for one device says nothing about another.
	pending, err = HasEndUserAuthPrompt(ctx, kv, "host-uuid-2", clk.Now())
	require.NoError(t, err)
	require.False(t, pending)

	// fleetd backing off well past the prompt does not drop the record: the
	// lifetime is event-based, not a short clock.
	clk.AddTime(2 * time.Hour)
	pending, err = HasEndUserAuthPrompt(ctx, kv, "host-uuid-1", clk.Now())
	require.NoError(t, err)
	require.True(t, pending)

	require.NoError(t, ClearEndUserAuthPrompt(ctx, kv, "host-uuid-1"))
	pending, err = HasEndUserAuthPrompt(ctx, kv, "host-uuid-1", clk.Now())
	require.NoError(t, err)
	require.False(t, pending)
}

func TestEndUserAuthPromptExpires(t *testing.T) {
	t.Parallel()

	kv := mockredis.NewMemKeyValueStore()
	clk := clock.NewMockClock()
	ctx := t.Context()

	require.NoError(t, RecordEndUserAuthPrompt(ctx, kv, "host-uuid-1", clk.Now()))

	clk.AddTime(euaPendingTTL)
	pending, err := HasEndUserAuthPrompt(ctx, kv, "host-uuid-1", clk.Now())
	require.NoError(t, err)
	require.False(t, pending, "a record the store has not evicted yet is still treated as gone")
}

func TestEndUserAuthPromptUnparseableValue(t *testing.T) {
	t.Parallel()

	kv := mockredis.NewMemKeyValueStore()
	clk := clock.NewMockClock()
	ctx := t.Context()

	// A value in another shape can only come from an older build or tampering.
	// It must read as "not pending" rather than an error, so the unauthenticated
	// endpoint keeps returning the same 401 for every refusal.
	require.NoError(t, kv.Set(ctx, "eua_pending:host-uuid-1", `{"expires_at":"2099-01-01T00:00:00Z"}`, time.Hour))
	pending, err := HasEndUserAuthPrompt(ctx, kv, "host-uuid-1", clk.Now())
	require.NoError(t, err)
	require.False(t, pending)

	// Recording over it heals the record.
	require.NoError(t, RecordEndUserAuthPrompt(ctx, kv, "host-uuid-1", clk.Now()))
	pending, err = HasEndUserAuthPrompt(ctx, kv, "host-uuid-1", clk.Now())
	require.NoError(t, err)
	require.True(t, pending)
}

func TestEndUserAuthPromptHostUUIDBounds(t *testing.T) {
	t.Parallel()

	kv := mockredis.NewMemKeyValueStore()
	clk := clock.NewMockClock()
	ctx := t.Context()

	tooLong := strings.Repeat("a", maxEUAPendingHostUUIDLen+1)
	for _, hostUUID := range []string{"", tooLong} {
		require.Error(t, RecordEndUserAuthPrompt(ctx, kv, hostUUID, clk.Now()))
		require.Error(t, ClearEndUserAuthPrompt(ctx, kv, hostUUID))

		pending, err := HasEndUserAuthPrompt(ctx, kv, hostUUID, clk.Now())
		require.NoError(t, err)
		require.False(t, pending)
	}
}

func TestEndUserAuthPromptStoreErrors(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	clk := clock.NewMockClock()

	// A nil store is a misconfiguration, not an absent record: every call must
	// report it so the caller fails closed rather than skipping the check.
	pending, err := HasEndUserAuthPrompt(ctx, nil, "host-uuid-1", clk.Now())
	require.Error(t, err)
	require.False(t, pending)
	require.Error(t, RecordEndUserAuthPrompt(ctx, nil, "host-uuid-1", clk.Now()))
	require.Error(t, ClearEndUserAuthPrompt(ctx, nil, "host-uuid-1"))

	boom := errors.New("redis is down")
	kv := &mockredis.KeyValueStore{
		GetFunc: func(_ context.Context, _ string) (*string, error) { return nil, boom },
		SetFunc: func(_ context.Context, _, _ string, _ time.Duration) error { return boom },
	}
	pending, err = HasEndUserAuthPrompt(ctx, kv, "host-uuid-1", clk.Now())
	require.ErrorIs(t, err, boom)
	require.False(t, pending)
	require.ErrorIs(t, RecordEndUserAuthPrompt(ctx, kv, "host-uuid-1", clk.Now()), boom)
}
