package async

import (
	"context"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	hostSeenRecordedHostIDsKey   = "{host_seen:host_ids}"            // the SET of current (pending) host ids
	hostSeenProcessingHostIDsKey = "{host_seen:host_ids}:processing" // the SET of host ids in the process of being collected
	hostSeenKeysMinTTL           = 7 * 24 * time.Hour                // 1 week
)

// RecordHostLastSeen records that the specified host ID was seen.
func (t *Task) RecordHostLastSeen(ctx context.Context, hostID uint) error {
	if !t.AsyncEnabled {
		t.seenHostSet.addHostID(hostID)
		return nil
	}

	// set an expiration on the SET key, ensuring that if async processing is
	// disabled, the set (eventually) does not use any redis space. Ensure that
	// TTL is reasonably big to avoid deleting information that hasn't been
	// collected yet - 1 week or 10 * the collector interval, whichever is
	// biggest.
	ttl := hostSeenKeysMinTTL
	if maxTTL := 10 * t.CollectorInterval; maxTTL > ttl {
		ttl = maxTTL
	}

	// keys and arguments passed to the script are:
	// KEYS[1]: recorded set (hostSeenRecordedHostIDsKey)
	// ARGV[1]: host id
	// ARGV[2]: ttl for the key
	script := redigo.NewScript(1, `
    redis.call('SADD', KEYS[1], ARGV[1])
    return redis.call('EXPIRE', KEYS[1], ARGV[2])
  `)

	conn := t.Pool.Get()
	defer conn.Close()
	if err := redis.BindConn(t.Pool, conn, hostSeenRecordedHostIDsKey); err != nil {
		return ctxerr.Wrap(ctx, err, "bind redis connection")
	}

	if _, err := script.Do(conn, hostSeenRecordedHostIDsKey, int(ttl.Seconds())); err != nil {
		return ctxerr.Wrap(ctx, err, "run redis script")
	}
	return nil
}

// FlushHostsLastSeen updates the last seen timestamp for the hosts that have
// been recorded since the last time FlushHostsLastSeen was called. It is a
// no-op if asychronous host processing is enabled, because then it is the
// task collector that will process the writes to mysql.
func (t *Task) FlushHostsLastSeen(ctx context.Context, now time.Time) error {
	if !t.AsyncEnabled {
		hostIDs := t.seenHostSet.getAndClearHostIDs()
		return t.Datastore.MarkHostsSeen(ctx, hostIDs, now)
	}

	// no-op, flushing the hosts' last seen is done via the cron that runs the
	// Task's collectors.
	return nil
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
