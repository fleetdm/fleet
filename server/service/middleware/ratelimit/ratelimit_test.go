package ratelimit

import (
	"context"
	"testing"

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
		throttled.RateQuota{MaxRate: throttled.PerHour(1), MaxBurst: 0},
	)(endpoint)

	_, err := wrapped(context.Background(), struct{}{})
	assert.NoError(t, err)

	// Hits rate limit
	_, err = wrapped(context.Background(), struct{}{})
	assert.Error(t, err)
	assert.Implements(t, (*Error)(nil), err)
}
