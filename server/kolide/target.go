package kolide

import (
	"golang.org/x/net/context"
)

type TargetSearchResults struct {
	Hosts  []Host
	Labels []Label
}

type TargetService interface {
	// SearchTargets will accept a search query, a slice of IDs of hosts to omit,
	// and a slice of IDs of labels to omit, and it will return a set of targets
	// (hosts and label) which match the supplied search query.
	SearchTargets(ctx context.Context, query string, selectedHostIDs []uint, selectedLabelIDs []uint) (*TargetSearchResults, error)

	CountHostsInTargets(ctx context.Context, hosts []uint, labels []uint) (uint, error)
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
