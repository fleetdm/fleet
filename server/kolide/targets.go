package kolide

import (
	"golang.org/x/net/context"
)

type TargetSearchResults struct {
	Hosts  []Host
	Labels []Label
}

// TargetMetrics contains information about the state
// of hosts that are tracked by the app
type TargetMetrics struct {
	TotalHosts uint
	// OnlineHosts have updated within the last 30 minutes
	OnlineHosts uint
	// OfflineHosts are hosts that haven't updated in 30 minutes
	OfflineHosts uint
	// MissingInActionHosts are hosts that haven't had an update for more
	// than thirty days
	MissingInActionHosts uint
}

type TargetService interface {
	// SearchTargets will accept a search query, a slice of IDs of hosts to omit,
	// and a slice of IDs of labels to omit, and it will return a set of targets
	// (hosts and label) which match the supplied search query.
	SearchTargets(ctx context.Context, query string, selectedHostIDs []uint, selectedLabelIDs []uint) (*TargetSearchResults, error)

	// CountHostsInTargets returns the count of hosts in the selected
	// targets. The first return uint is the total number of hosts in the
	// targets. The second return uint is the total online hosts. The third
	// returned uint is the total number of hosts that have been offline for more
	// than 30 days. (Missing in action)
	CountHostsInTargets(ctx context.Context, hostIDs []uint, labelIDs []uint) (*TargetMetrics, error)
}

type TargetType int

const (
	TargetLabel TargetType = iota
	TargetHost
)

type Target struct {
	Type     TargetType
	TargetID uint
}
