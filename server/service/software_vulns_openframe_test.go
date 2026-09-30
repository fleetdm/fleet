package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

const openframeTestTeamID uint = 7

// TestOpenframeListVulnerabilitiesEmptyPageUpdatedAt verifies the OPENFRAME(mysql-multitenancy)
// counts_updated_at of listVulnerabilitiesEndpoint: a pinned tenant with no CVEs gets the instance's
// last host-count recalculation instead of now(); unpinned and non-empty pages keep upstream behavior.
func TestOpenframeListVulnerabilitiesEmptyPageUpdatedAt(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	pinnedCtx := fleet.NewOpenframeTeamContext(ctx, openframeTestTeamID)

	recalculatedAt := time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)
	var page []fleet.VulnerabilityWithMetadata
	ds.ListVulnerabilitiesFunc = func(ctx context.Context, opt fleet.VulnListOptions) ([]fleet.VulnerabilityWithMetadata, *fleet.PaginationMetadata, error) {
		return page, &fleet.PaginationMetadata{}, nil
	}
	ds.CountVulnerabilitiesFunc = func(ctx context.Context, opt fleet.VulnListOptions) (uint, error) {
		return uint(len(page)), nil
	}
	ds.VulnerabilityHostCountsUpdatedAtFunc = func(ctx context.Context) (time.Time, error) {
		return recalculatedAt, nil
	}
	list := func(ctx context.Context) listVulnerabilitiesResponse {
		resp, err := listVulnerabilitiesEndpoint(ctx, &listVulnerabilitiesRequest{}, svc)
		require.NoError(t, err)
		out := resp.(listVulnerabilitiesResponse)
		require.NoError(t, out.Err)
		return out
	}

	require.Equal(t, recalculatedAt, list(pinnedCtx).CountsUpdatedAt)

	ds.VulnerabilityHostCountsUpdatedAtFuncInvoked = false
	require.WithinDuration(t, time.Now(), list(ctx).CountsUpdatedAt, time.Minute)
	require.False(t, ds.VulnerabilityHostCountsUpdatedAtFuncInvoked)

	rowUpdatedAt := time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC)
	page = []fleet.VulnerabilityWithMetadata{{CVE: fleet.CVE{CVE: "CVE-2026-0001"}, HostsCountUpdatedAt: rowUpdatedAt}}
	require.Equal(t, rowUpdatedAt, list(pinnedCtx).CountsUpdatedAt)
	require.False(t, ds.VulnerabilityHostCountsUpdatedAtFuncInvoked)
}

// TestOpenframeListHostsSoftwareFilterTeamFence verifies the OPENFRAME(mysql-multitenancy) fence in
// listHostsEndpoint: with a software version or title filter, a pinned tenant resolves the software
// only within its team and never falls back to the unscoped name lookups.
func TestOpenframeListHostsSoftwareFilterTeamFence(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	pinnedCtx := fleet.NewOpenframeTeamContext(ctx, openframeTestTeamID)

	const ownVersionID, foreignVersionID, foreignTitleID uint = 1, 2, 3
	ds.ListHostsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		return nil, nil
	}
	ds.TeamExistsFunc = func(ctx context.Context, teamID uint) (bool, error) {
		return true, nil
	}
	ds.SoftwareByIDFunc = func(ctx context.Context, id uint, teamID *uint, includeCVEScores bool, tmFilter *fleet.TeamFilter) (*fleet.Software, error) {
		if id == ownVersionID && teamID != nil && *teamID == openframeTestTeamID {
			return &fleet.Software{ID: id, Name: "alpha", Version: "1.0"}, nil
		}
		return nil, newNotFoundError()
	}
	ds.SoftwareLiteByIDFunc = func(ctx context.Context, id uint) (fleet.SoftwareLite, error) {
		return fleet.SoftwareLite{Name: "beta", Version: "1.0"}, nil
	}
	ds.SoftwareTitleByIDFunc = func(ctx context.Context, id uint, teamID *uint, tmFilter fleet.TeamFilter) (*fleet.SoftwareTitle, error) {
		return nil, newNotFoundError()
	}
	ds.SoftwareTitleNameForHostFilterFunc = func(ctx context.Context, id uint) (string, string, error) {
		return "beta", "", nil
	}
	list := func(ctx context.Context, opts fleet.HostListOptions) listHostsResponse {
		resp, err := listHostsEndpoint(ctx, &listHostsRequest{Opts: opts}, svc)
		require.NoError(t, err)
		out := resp.(streamHostsResponse).listHostsResponse
		require.NoError(t, out.Err)
		return out
	}

	own := list(pinnedCtx, fleet.HostListOptions{SoftwareVersionIDFilter: ptr.Uint(ownVersionID)})
	require.NotNil(t, own.Software)
	require.Equal(t, "alpha", own.Software.Name)

	foreign := list(pinnedCtx, fleet.HostListOptions{SoftwareVersionIDFilter: ptr.Uint(foreignVersionID)})
	require.Nil(t, foreign.Software)
	foreignTitle := list(pinnedCtx, fleet.HostListOptions{SoftwareTitleIDFilter: ptr.Uint(foreignTitleID)})
	require.Nil(t, foreignTitle.SoftwareTitle)
	require.False(t, ds.SoftwareLiteByIDFuncInvoked)
	require.False(t, ds.SoftwareTitleNameForHostFilterFuncInvoked)

	unpinned := list(ctx, fleet.HostListOptions{SoftwareTitleIDFilter: ptr.Uint(foreignTitleID)})
	require.NotNil(t, unpinned.SoftwareTitle)
	require.True(t, ds.SoftwareTitleNameForHostFilterFuncInvoked)
}
