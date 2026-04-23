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
	"fmt"

	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
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

// FleetDataCollectionReader is the narrow subset of fleet.Datastore that
// FleetDataCollectionAdapter needs. Using an interface keeps the adapter
// testable without constructing a full datastore in tests.
type FleetDataCollectionReader interface {
	AppConfig(ctx context.Context) (*fleet.AppConfig, error)
	ListTeams(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error)
	TeamFeatures(ctx context.Context, teamID uint) (*fleet.Features, error)
}

// FleetDataCollectionAdapter translates the chart context's DataCollectionState
// question into reads against the Fleet datastore's cached AppConfig and
// TeamFeatures. Both upstream methods call ApplyDefaults before unmarshal, so
// fleets that predate data_collection still read as on-by-default without any
// backfill migration.
type FleetDataCollectionAdapter struct {
	ds FleetDataCollectionReader
}

// NewFleetDataCollectionAdapter returns an adapter suitable for passing to
// chart/bootstrap.New.
func NewFleetDataCollectionAdapter(ds FleetDataCollectionReader) *FleetDataCollectionAdapter {
	return &FleetDataCollectionAdapter{ds: ds}
}

// DataCollectionState returns (globalEnabled, enabledFleetIDs, err). When
// globalEnabled is false, enabledFleetIDs is nil — callers short-circuit
// the entire dataset in that case.
func (a *FleetDataCollectionAdapter) DataCollectionState(ctx context.Context, dataset string) (bool, []uint, error) {
	cfg, err := a.ds.AppConfig(ctx)
	if err != nil {
		return false, nil, err
	}
	globalOn, err := datasetFlag(cfg.Features.DataCollection, dataset)
	if err != nil || !globalOn {
		return false, nil, err
	}

	// Cron runs as a system context with no viewer. TeamFilter requires a
	// user; synthesize an admin-role one so whereFilterTeams returns all
	// fleets.
	teams, err := a.ds.ListTeams(ctx, fleet.TeamFilter{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}}, fleet.ListOptions{})
	if err != nil {
		return false, nil, err
	}

	enabled := make([]uint, 0, len(teams))
	for _, t := range teams {
		feat, err := a.ds.TeamFeatures(ctx, t.ID)
		if err != nil {
			return false, nil, err
		}
		on, err := datasetFlag(feat.DataCollection, dataset)
		if err != nil {
			return false, nil, err
		}
		if on {
			enabled = append(enabled, t.ID)
		}
	}
	return true, enabled, nil
}

// datasetFlag maps a dataset name to the corresponding bool field on
// DataCollectionSettings via a typed switch. Adding a dataset means adding a
// case here plus a field on fleet.DataCollectionSettings.
func datasetFlag(dc fleet.DataCollectionSettings, dataset string) (bool, error) {
	switch dataset {
	case "uptime":
		return dc.Uptime, nil
	case "cve":
		return dc.CVE, nil
	default:
		return false, fmt.Errorf("unknown dataset %q", dataset)
	}
}
