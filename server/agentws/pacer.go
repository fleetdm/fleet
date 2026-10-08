package agentws

import (
	"context"
	"sync"
	"time"
)

// pacerTick is how often the pacer releases a share of its queue.
const pacerTick = 100 * time.Millisecond

type pacedKey struct {
	hostID  uint
	msgType string
}

// pacer spreads the notifications of a fleet-wide or global change evenly
// over a window, so that every connected agent doesn't fetch at once. Targets
// queued while a drain is in progress are merged into it, at most once per
// host and message type (keeping the latest reason).
type pacer struct {
	hub    *Hub
	spread time.Duration
	wake   chan struct{}

	mu       sync.Mutex
	queue    []pacedKey
	reasons  map[pacedKey]string
	deadline time.Time
}

func newPacer(hub *Hub, spread time.Duration) *pacer {
	return &pacer{
		hub:     hub,
		spread:  spread,
		wake:    make(chan struct{}, 1),
		reasons: make(map[pacedKey]string),
	}
}

// add queues a notification for hostIDs and returns how many were not
// already queued. Each add that queues new targets restarts the spread
// window for the whole queue, so the release rate stays bounded by
// len(queue)/spread.
func (p *pacer) add(msgType, reason string, hostIDs []uint) int {
	p.mu.Lock()
	added := 0
	for _, id := range hostIDs {
		key := pacedKey{hostID: id, msgType: msgType}
		if _, ok := p.reasons[key]; !ok {
			p.queue = append(p.queue, key)
			added++
		}
		p.reasons[key] = reason
	}
	if added > 0 {
		p.deadline = time.Now().Add(p.spread)
	}
	p.mu.Unlock()

	select {
	case p.wake <- struct{}{}:
	default:
	}
	return added
}

// queueLen returns the number of queued targets.
func (p *pacer) queueLen() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.queue)
}

// next pops the share of the queue due at now: everything once the deadline
// is reached, otherwise the queue's fraction that one tick represents of the
// remaining window (at least one target).
func (p *pacer) next(now time.Time) map[pacedKey]string {
	p.mu.Lock()
	defer p.mu.Unlock()

	n := len(p.queue)
	if left := p.deadline.Sub(now); left > pacerTick && n > 0 {
		n = max(1, int((int64(n)*int64(pacerTick)+int64(left)-1)/int64(left)))
	}
	batch := make(map[pacedKey]string, n)
	for _, key := range p.queue[:n] {
		batch[key] = p.reasons[key]
		delete(p.reasons, key)
	}
	p.queue = p.queue[n:]
	if len(p.queue) == 0 {
		p.queue = nil
	}
	return batch
}

// run releases queued notifications until ctx is done.
func (p *pacer) run(ctx context.Context) {
	ticker := time.NewTicker(pacerTick)
	defer ticker.Stop()
	for {
		select {
		case <-p.wake:
		case <-ctx.Done():
			return
		}
		// Drop the tick buffered while idle, so the second release waits a tick.
		ticker.Reset(pacerTick)
		for p.queueLen() > 0 {
			p.release(p.next(time.Now()))
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (p *pacer) release(batch map[pacedKey]string) {
	type group struct{ msgType, reason string }
	groups := make(map[group][]uint)
	for key, reason := range batch {
		g := group{msgType: key.msgType, reason: reason}
		groups[g] = append(groups[g], key.hostID)
	}
	for g, ids := range groups {
		p.hub.Notify(g.msgType, g.reason, ids)
	}
}
