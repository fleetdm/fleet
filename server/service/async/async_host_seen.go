package async

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (t *Task) RecordHostLastSeen(ctx context.Context, hostID uint) error {
	t.seenHostSet.addHostID(hostID)
	return nil
}

// FlushHostsLastSeen updates the last seen timestamp for the hosts that have
// been recorded since the last time FlushHostsLastSeen was called.
func (t *Task) FlushHostsLastSeen(ctx context.Context, now time.Time) error {
	if t.otelEnabled {
		tracer := otel.Tracer("async")
		var span trace.Span
		ctx, span = tracer.Start(ctx, "async.flush_hosts_last_seen",
			trace.WithAttributes(
				attribute.String("async.mode", "synchronous"),
				attribute.String("async.task", "host_last_seen"),
			),
		)
		defer span.End()
	}

	hostIDs := t.seenHostSet.getAndClearHostIDs()
	return t.datastore.MarkHostsSeen(ctx, hostIDs, now)
}

type seenHostSet struct {
	mutex   sync.Mutex
	hostIDs map[uint]bool
}

func (m *seenHostSet) addHostID(id uint) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.hostIDs == nil {
		m.hostIDs = make(map[uint]bool)
	}
	m.hostIDs[id] = true
}

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
