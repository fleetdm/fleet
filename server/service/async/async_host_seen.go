package async

import (
	"context"
	"sync"
	"time"
)

func (t *Task) RecordHostLastSeen(ctx context.Context, hostID uint) error {
	if t.AsyncEnabled {
		// TODO: store in redis
	}

	t.seenHostSet.addHostID(hostID)
	return nil
}

func (t *Task) FlushHostsLastSeen(ctx context.Context, now time.Time) error {
	if t.AsyncEnabled {
		// no-op, flushing the hosts' last seen is done via the cron that runs the
		// Task's collectors.
		return nil
	}

	hostIDs := t.seenHostSet.getAndClearHostIDs()
	return t.Datastore.MarkHostsSeen(ctx, hostIDs, now)
}

// seenHostSet implements synchronized storage for the set of seen hosts.
type seenHostSet struct {
	mutex   sync.Mutex
	hostIDs map[uint]bool
}

// addHostID adds the host identified by ID to the set
func (m *seenHostSet) addHostID(id uint) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.hostIDs == nil {
		m.hostIDs = make(map[uint]bool)
	}
	m.hostIDs[id] = true
}

// getAndClearHostIDs gets the list of unique host IDs from the set and empties
// the set.
func (m *seenHostSet) getAndClearHostIDs() []uint {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var ids []uint
	for id := range m.hostIDs {
		ids = append(ids, id)
	}
	m.hostIDs = make(map[uint]bool)
	return ids
}
