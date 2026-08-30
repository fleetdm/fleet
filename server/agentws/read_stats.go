package agentws

import (
	"cmp"
	"slices"
	"sync"
)

// ReadStats is the per-host count of distributed/read requests split by which
// path served them, for observability (the /debug/agentws endpoint):
//
//   - OrbitReads: requests on /api/osquery/distributed/read — orbit's client,
//     used when the WebSocket transport drives check-ins.
//   - LegacyReads: requests on the /api/v1/... alias — osqueryd's built-in tls
//     plugin (FleetFlags points it at the v1 path). With the transport active
//     these should stay flat; growth means a host's osquery is still polling.
//
// Counting is by request path, independent of whether the host currently
// holds a WebSocket connection, so legacy pollers are visible too.
type ReadStats struct {
	HostID      uint  `json:"host_id"`
	OrbitReads  int64 `json:"orbit_reads"`
	LegacyReads int64 `json:"legacy_reads"`
}

type readStatsRegistry struct {
	mu   sync.Mutex
	byID map[uint]*ReadStats
}

// RecordDistributedRead counts one distributed/read served for hostID;
// legacyPath reports whether it arrived on the /api/v1/... alias.
func (h *Hub) RecordDistributedRead(hostID uint, legacyPath bool) {
	h.reads.mu.Lock()
	defer h.reads.mu.Unlock()
	if h.reads.byID == nil {
		h.reads.byID = make(map[uint]*ReadStats)
	}
	s, ok := h.reads.byID[hostID]
	if !ok {
		s = &ReadStats{HostID: hostID}
		h.reads.byID[hostID] = s
	}
	if legacyPath {
		s.LegacyReads++
	} else {
		s.OrbitReads++
	}
}

// ReadStats returns the per-host distributed/read counts recorded since the
// server started, sorted by host ID.
func (h *Hub) ReadStats() []ReadStats {
	h.reads.mu.Lock()
	stats := make([]ReadStats, 0, len(h.reads.byID))
	for _, s := range h.reads.byID {
		stats = append(stats, *s)
	}
	h.reads.mu.Unlock()

	slices.SortFunc(stats, func(a, b ReadStats) int {
		return cmp.Compare(a.HostID, b.HostID)
	})
	return stats
}
