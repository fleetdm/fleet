package ratelimit

import (
	"context"
	"errors"
	"net/http"
	"testing"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"
)

// Intent is to test the middleware functionality. We rely on the tests within
// Throttled to verify that the rate limiting algorithm works properly.

func TestLimit(t *testing.T) {
	t.Parallel()

	store, _ := memstore.New(0)
	limiter := NewMiddleware(store)
	var endpointCallCount uint

	endpoint := func(context.Context, interface{}) (interface{}, error) {
		endpointCallCount++
		return struct{}{}, nil
	}
	wrapped := limiter.Limit(
		"test_limit",
		throttled.RateQuota{MaxRate: throttled.PerHour(1), MaxBurst: 0},
	)(endpoint)

	wrapped2 := limiter.Limit(
		"test_limit2",
		throttled.RateQuota{MaxRate: throttled.PerHour(1), MaxBurst: 0},
	)(endpoint)

	sameWrapped := limiter.Limit(
		"test_limit",
		throttled.RateQuota{MaxRate: throttled.PerHour(1), MaxBurst: 0},
	)(endpoint)

	authzCtx := &authz_ctx.AuthorizationContext{}
	ctx := authz_ctx.NewContext(context.Background(), authzCtx)

	_, err := wrapped(ctx, struct{}{})
	assert.NoError(t, err)

	// Hits rate limit
	_, err = wrapped(ctx, struct{}{})
	assert.Error(t, err)
	var rle Error
	assert.True(t, errors.As(err, &rle))
	assert.True(t, authzCtx.Checked())
	require.Contains(t, rle.Error(), "limit exceeded, retry after: ")
	rle_, ok := rle.(*rateLimitError)
	require.True(t, ok)
	require.NotZero(t, rle_.RetryAfter())
	require.Equal(t, http.StatusTooManyRequests, rle_.StatusCode())

	// ensure that the same endpoint wrapped with a different limiter doesn't hit the error
	_, err = wrapped2(ctx, struct{}{})
	assert.NoError(t, err)

	// Same underlying key, so hits same limit
	_, err = sameWrapped(ctx, struct{}{})
	assert.Error(t, err)

	assert.True(t, errors.As(err, &rle))
	assert.True(t, authzCtx.Checked())

	assert.Equal(t, uint(2), endpointCallCount) // when rate limit is exceeded, shouldn't call endpoint
}

func TestNewErrorMiddlewarePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	NewErrorMiddleware(nil)
}
