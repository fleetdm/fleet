package failing

import (
	"context"
	"fmt"
	"io"
	"time"
)

// commonFailingStore is an implementation of CommonStore
// that fails all operations. It is used when S3 is not configured and the
// local filesystem store could not be setup.
type commonFailingStore struct {
	Entity string
}

func (c commonFailingStore) Get(ctx context.Context, iconID string) (io.ReadCloser, int64, error) {
	return nil, 0, fmt.Errorf("%s store not properly configured", c.Entity)
}

func (c commonFailingStore) Put(ctx context.Context, iconID string, content io.ReadSeeker) error {
	return fmt.Errorf("%s store not properly configured", c.Entity)
}

func (c commonFailingStore) Exists(ctx context.Context, iconID string) (bool, error) {
	return false, fmt.Errorf("%s store not properly configured", c.Entity)
}

func (c commonFailingStore) Cleanup(ctx context.Context, usedIconIDs []string, removeCreatedBefore time.Time) (int, error) {
	return 0, nil
}

func (c commonFailingStore) Sign(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("%s store not properly configured", c.Entity)
}
