package mock

import (
	"context"
	"sync"
	"time"
)

// NewMemKeyValueStore returns a map-backed KeyValueStore for tests that
// exercise code using the store rather than the store itself. Expiry is not
// simulated: a value stays readable until overwritten.
func NewMemKeyValueStore() *KeyValueStore {
	var mu sync.Mutex
	vals := map[string]string{}
	return &KeyValueStore{
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
}
