package ratelimit

import (
	"context"
	"errors"
	"net/http"
	"testing"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/publicip"
	kitlog "github.com/go-kit/log"
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

func TestLimitOnlyWhenError(t *testing.T) {
	t.Parallel()

	store, _ := memstore.New(1)
	limiter := NewErrorMiddleware(store)
	endpoint := func(context.Context, interface{}) (interface{}, error) { return struct{}{}, nil }
	wrapped := limiter.Limit(
		"test_limit", throttled.RateQuota{MaxRate: throttled.PerHour(1), MaxBurst: 0}, kitlog.NewNopLogger(),
	)(endpoint)

	// Does NOT hit any rate limits because the endpoint doesn't fail
	ctx := publicip.NewContext(context.Background(), "0.0.0.0")
	_, err := wrapped(ctx, struct{}{})
	assert.NoError(t, err)
	_, err = wrapped(ctx, struct{}{})
	assert.NoError(t, err)

	expectedError := errors.New("error")
	failingEndpoint := func(context.Context, interface{}) (interface{}, error) { return nil, expectedError }
	wrappedFailer := limiter.Limit(
		"test_limit", throttled.RateQuota{MaxRate: throttled.PerHour(1), MaxBurst: 0}, kitlog.NewNopLogger(),
	)(failingEndpoint)

	// First request that fails should be allowed.
	_, err = wrappedFailer(ctx, struct{}{})
	assert.ErrorIs(t, err, expectedError)

	// Second request that fails should not be allowed.
	_, err = wrappedFailer(ctx, struct{}{})
	assert.Error(t, err)
	var rle Error
	require.True(t, errors.As(err, &rle))
	// github.com/throttled/throttled has a bug where "peeking" with RateLimit(key, 0)
	// always returns a RetryAfter=-1. So I'll just leave this here but in the future
	// we could return the correct Retry-After. Also, we are not making use of "Retry-After"
	// on the agent side yet.
	require.EqualValues(t, rle.Result().RetryAfter, -1)
	require.Equal(t, "limit exceeded", rle.Error())
}

func TestNoRateLimitWithoutPublicIP(t *testing.T) {
	t.Parallel()

	store, _ := memstore.New(1)
	limiter := NewErrorMiddleware(store)

	expectedError := errors.New("error")
	failingEndpoint := func(context.Context, interface{}) (interface{}, error) {
		return nil, expectedError
	}
	wrappedFailer := limiter.Limit(
		"test_limit", throttled.RateQuota{MaxRate: throttled.PerHour(1), MaxBurst: 0}, kitlog.NewNopLogger(),
	)(failingEndpoint)

	ctx := context.Background()

	// Requests should not be rate limited because there's no "Public IP" identifier in the request.
	_, err := wrappedFailer(ctx, struct{}{})
	assert.ErrorIs(t, err, expectedError)
	_, err = wrappedFailer(ctx, struct{}{})
	assert.ErrorIs(t, err, expectedError)
}
