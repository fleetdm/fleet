package ratelimit

import (
	"context"
	"errors"
	"testing"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/stretchr/testify/assert"
	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"
)

// Intent is to test the middleware functionality. We rely on the tests within
// Throttled to verify that the rate limiting algorithm works properly.

func TestLimit(t *testing.T) {
	t.Parallel()

	store, _ := memstore.New(0)
	limiter := NewMiddleware(store)
	endpoint := func(context.Context, interface{}) (interface{}, error) { return struct{}{}, nil }
	wrapped := limiter.Limit(
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
		"test_limit", throttled.RateQuota{MaxRate: throttled.PerHour(1), MaxBurst: 0},
	)(endpoint)

	// Does NOT hit any rate limits because the endpoint doesn't fail
	_, err := wrapped(context.Background(), struct{}{})
	assert.NoError(t, err)
	_, err = wrapped(context.Background(), struct{}{})
	assert.NoError(t, err)

	expectedError := errors.New("error")
	failingEndpoint := func(context.Context, interface{}) (interface{}, error) { return nil, expectedError }
	wrappedFailer := limiter.Limit(
		"test_limit", throttled.RateQuota{MaxRate: throttled.PerHour(1), MaxBurst: 0},
	)(failingEndpoint)

	_, err = wrappedFailer(context.Background(), struct{}{})
	assert.ErrorIs(t, err, expectedError)

	// Hits rate limit now that it fails
	_, err = wrappedFailer(context.Background(), struct{}{})
	assert.Error(t, err)
	var rle Error
	assert.True(t, errors.As(err, &rle))
}
