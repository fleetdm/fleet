package mysql

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"time"
)

type asyncLastSeen struct {
	flushInterval time.Duration
	flushCap      int
	set           *seenSet[string]
	fn            func(ctx context.Context, ids []string)
}

func newAsyncLastSeen(flushInterval time.Duration, flushCap int, fn func(ctx context.Context, ids []string)) *asyncLastSeen {
	return &asyncLastSeen{
		flushInterval: flushInterval,
		flushCap:      flushCap,
		set:           &seenSet[string]{},
		fn:            fn,
	}
}

func (a *asyncLastSeen) markHostSeen(ctx context.Context, id string) {
	ids, flush := a.set.add(id, a.flushCap)
	if flush && len(ids) > 0 {
		a.fn(ctx, ids)
	}
}

func (a *asyncLastSeen) runFlushLoop(ctx context.Context) {
	tickCh := time.Tick(a.flushInterval)
	for {
		select {
		case <-tickCh:
			ids := a.set.getAndClear()
			if len(ids) > 0 {
				a.fn(ctx, ids)
			}

		case <-ctx.Done():
			return
		}
	}
}

// TODO: this could replace the seenHostSet in server/service/async package,
// but I did not want to introduce a dependency between nanomdm and our
// internal code at this point.

// seenSet implements synchronized storage for the set of seen identifiers.
type seenSet[T cmp.Ordered] struct {
	mutex   sync.Mutex
	seenIDs map[T]struct{}
}

// add adds the identifier to the set and returns the list of unique IDs -
// clearing the set - if the cap is reached, returning true as the second
// value. Otherwise it returns nil and false. Essentially, if the number of
// unique IDs >= cap, it acts as if getAndClear was called after adding the new
// id. A cap <= 0 is ignored.
func (s *seenSet[T]) add(id T, cap int) ([]T, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.seenIDs == nil {
		s.seenIDs = make(map[T]struct{})
	}
	s.seenIDs[id] = struct{}{}

	if cap > 0 && len(s.seenIDs) >= cap {
		return s.getAndClearLocked(), true
	}
	return nil, false
}

// getAndClear gets the list of unique IDs from the set and empties it.
func (s *seenSet[T]) getAndClear() []T {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.getAndClearLocked()
}

// getAndClearLocked is identical to getAndClear but must only be called when
// the s.mutex lock is held.
func (s *seenSet[T]) getAndClearLocked() []T {
	var ids []T
	for id := range s.seenIDs {
		ids = append(ids, id)
	}
	// clear the set
	s.seenIDs = make(map[T]struct{})

	// sort to help prevent deadlocks when processing the batch SQL statement
	slices.Sort(ids)
	return ids
}
