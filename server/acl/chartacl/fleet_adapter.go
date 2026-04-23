// Package chartacl provides the anti-corruption layer between the chart
// bounded context and legacy Fleet code.
//
// This package is the ONLY place that imports both chart types and fleet /
// viewer-context types. It translates between them, letting the chart
// context stay free of direct server/fleet or server/contexts/viewer imports
// (which the arch_test enforces).
package chartacl

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
)

// FleetViewerAdapter resolves the current authenticated viewer into the
// minimal information the chart service needs to decide team scope. It has
// no state and no Fleet-service dependency — the viewer context is the only
// input it needs.
type FleetViewerAdapter struct{}

// NewFleetViewerAdapter returns an adapter suitable for passing to
// chart/bootstrap.New.
func NewFleetViewerAdapter() *FleetViewerAdapter {
	return &FleetViewerAdapter{}
}

// Ensure FleetViewerAdapter implements api.ViewerProvider.
var _ api.ViewerProvider = (*FleetViewerAdapter)(nil)

// ViewerScope reads the user from the viewer context and reports whether they
// have global access (isGlobal) or, otherwise, the list of team IDs they
// belong to. Returns an error when no viewer is in context — chart endpoints
// sit behind authenticated middleware, so the absence of a viewer means the
// request never went through auth and we refuse to serve data.
func (a *FleetViewerAdapter) ViewerScope(ctx context.Context) (bool, []uint, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok || vc.User == nil {
		return false, nil, errors.New("chart: no authenticated viewer in context")
	}
	u := vc.User
	if u.GlobalRole != nil && *u.GlobalRole != "" {
		return true, nil, nil
	}
	ids := make([]uint, 0, len(u.Teams))
	for _, t := range u.Teams {
		ids = append(ids, t.ID)
	}
	return false, ids, nil
}
