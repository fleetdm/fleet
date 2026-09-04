package mdm

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/fleet"
	mockredis "github.com/fleetdm/fleet/v4/server/mock/redis"
	"github.com/stretchr/testify/require"
)

func TestBYODIdPSession(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	vals := map[string]string{}
	kv := &mockredis.KeyValueStore{
		SetFunc: func(_ context.Context, key, value string, _ time.Duration) error {
			mu.Lock()
			defer mu.Unlock()
			vals[key] = value
			return nil
		},
		GetFunc: func(_ context.Context, key string) (*string, error) {
			mu.Lock()
			defer mu.Unlock()
			v, ok := vals[key]
			if !ok {
				return nil, nil
			}
			return &v, nil
		},
	}
	clk := clock.NewMockClock()
	ctx := t.Context()

	sessionID, err := CreateBYODIdPSession(ctx, kv, clk, "idp-account-uuid")
	require.NoError(t, err)
	require.NotEmpty(t, sessionID)
	require.NotEqual(t, "idp-account-uuid", sessionID)

	got, err := ValidateBYODIdPSession(ctx, kv, clk, sessionID)
	require.NoError(t, err)
	require.Equal(t, "idp-account-uuid", got)

	// The reference the client would have sent is not a session id.
	_, err = ValidateBYODIdPSession(ctx, kv, clk, "idp-account-uuid")
	require.ErrorAs(t, err, new(*fleet.AuthRequiredError))

	_, err = ValidateBYODIdPSession(ctx, kv, clk, "")
	require.ErrorAs(t, err, new(*fleet.AuthRequiredError))

	clk.AddTime(BYODIdPSessionTTL - time.Second)
	got, err = ValidateBYODIdPSession(ctx, kv, clk, sessionID)
	require.NoError(t, err)
	require.Equal(t, "idp-account-uuid", got)

	// The expiry instant itself is out; the window is closed, not half-open.
	clk.AddTime(time.Second)
	_, err = ValidateBYODIdPSession(ctx, kv, clk, sessionID)
	require.ErrorAs(t, err, new(*fleet.AuthRequiredError))

	// Consuming ends the session immediately, before the store drops the key.
	sessionID, err = CreateBYODIdPSession(ctx, kv, clk, "idp-account-uuid")
	require.NoError(t, err)
	require.NoError(t, ConsumeBYODIdPSession(ctx, kv, clk, sessionID))
	_, err = ValidateBYODIdPSession(ctx, kv, clk, sessionID)
	require.ErrorAs(t, err, new(*fleet.AuthRequiredError))

	// A missing or failing store is not the user being signed out.
	_, err = ValidateBYODIdPSession(ctx, nil, clk, sessionID)
	require.Error(t, err)
	require.NotErrorAs(t, err, new(*fleet.AuthRequiredError))
	broken := &mockredis.KeyValueStore{GetFunc: func(context.Context, string) (*string, error) { return nil, errors.New("redis down") }}
	_, err = ValidateBYODIdPSession(ctx, broken, clk, sessionID)
	require.ErrorContains(t, err, "redis down")
	require.NotErrorAs(t, err, new(*fleet.AuthRequiredError))
}
