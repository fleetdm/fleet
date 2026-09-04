package agentws

import (
	"context"
	"maps"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSeenRecorder struct {
	mu   sync.Mutex
	seen map[uint]int
}

func (f *fakeSeenRecorder) RecordHostLastSeen(ctx context.Context, hostID uint) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.seen == nil {
		f.seen = make(map[uint]int)
	}
	f.seen[hostID]++
	return nil
}

func (f *fakeSeenRecorder) counts() map[uint]int {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[uint]int, len(f.seen))
	maps.Copy(out, f.seen)
	return out
}

func TestRecordSeenLoop(t *testing.T) {
	hub := NewHub(discardLogger(), time.Minute, 30*time.Second)
	srv := newTestServer(t, hub)

	dial(t, srv, "key-1")
	dial(t, srv, "key-2")
	waitForConnCount(t, hub, 2)

	recorder := &fakeSeenRecorder{}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		RecordSeenLoop(ctx, hub, recorder, 10*time.Millisecond, discardLogger())
		close(done)
	}()

	// Both connected hosts get recorded, repeatedly.
	require.Eventually(t, func() bool {
		c := recorder.counts()
		return c[1] >= 2 && c[2] >= 2
	}, 5*time.Second, 5*time.Millisecond)

	// Unconnected hosts are never recorded.
	assert.NotContains(t, recorder.counts(), uint(3))

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RecordSeenLoop did not stop on context cancellation")
	}
}
