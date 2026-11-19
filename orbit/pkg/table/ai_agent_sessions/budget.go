package ai_agent_sessions

import (
	"sync"
	"time"
)

// maxResultBytes is roughly how much ai_agent_session_messages returns per
// query, counting every column as it goes over the wire. It keeps a result
// well under what Fleet ingests from a host in one report (5 MiB for live
// queries in body-auth mode, and the server's per-message buffers).
const maxResultBytes = 1 << 20

// budgetIdle is how long after a call the budget stays shared. osquery
// generates the table once per value of an IN list (or OR, or join), a few
// milliseconds apart, and those calls together are one query's result.
const budgetIdle = 500 * time.Millisecond

type budget struct {
	mu     sync.Mutex
	used   int
	last   time.Time
	active int
}

var messageBudget budget

// begin starts a call, resetting the budget if the last call ended long
// enough ago to belong to another query.
func (b *budget) begin(t time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.active == 0 && t.Sub(b.last) > budgetIdle {
		b.used = 0
	}
	b.active++
}

// take reserves n bytes, reporting false when they don't fit. A call's first
// row is admitted even once the budget is spent, so the later values of an IN
// list can't be mistaken for empty sessions; that overrun is itself bounded.
func (b *budget) take(n int, first bool) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.used+n > maxResultBytes && (!first || b.used >= 2*maxResultBytes) {
		b.used = max(b.used, maxResultBytes)
		return false
	}
	b.used += n
	return true
}

func (b *budget) end(t time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.last = t
	b.active--
}

func (b *budget) reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.used, b.last, b.active = 0, time.Time{}, 0
}
