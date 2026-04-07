package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListHostReports(t *testing.T) {
	ds := new(mock.Store)

	now := time.Now().UTC().Truncate(time.Second)

	// global admin user
	admin := &fleet.User{
		ID:         1,
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}

	// Host with no team
	hostNoTeam := &fleet.Host{ID: 10, TeamID: nil}
	// Host with team
	teamID := uint(42)
	hostWithTeam := &fleet.Host{ID: 20, TeamID: &teamID}

	sampleReports := []*fleet.HostReport{
		{
			ReportID:     1,
			Name:         "Query Alpha",
			Description:  "desc alpha",
			LastFetched:  &now,
			FirstResult:  map[string]string{"col1": "val1"},
			NHostResults: 3,
		},
		{
			ReportID:     2,
			Name:         "Query Beta",
			Description:  "desc beta",
			LastFetched:  nil,
			FirstResult:  nil,
			NHostResults: 0,
		},
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		if id == hostNoTeam.ID {
			return hostNoTeam, nil
		}
		if id == hostWithTeam.ID {
			return hostWithTeam, nil
		}
		return nil, errors.New("host not found")
	}

	var capturedTeamID *uint
	ds.ListHostReportsFunc = func(ctx context.Context, hostID uint, tID *uint, hostPlatform string, opts fleet.ListHostReportsOptions, maxQueryReportRows int) ([]*fleet.HostReport, int, *fleet.PaginationMetadata, error) {
		capturedTeamID = tID
		return sampleReports, len(sampleReports), nil, nil
	}

	svc, ctx := newTestService(t, ds, nil, nil)

	t.Run("admin can list reports for host with no team", func(t *testing.T) {
		viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: admin})
		opts := fleet.ListHostReportsOptions{}
		reports, count, _, err := svc.ListHostReports(viewerCtx, hostNoTeam.ID, opts)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
		assert.Len(t, reports, 2)
		assert.Equal(t, "Query Alpha", reports[0].Name)
		assert.Equal(t, "Query Beta", reports[1].Name)
		assert.Equal(t, &now, reports[0].LastFetched)
		assert.Nil(t, reports[1].LastFetched)
		assert.Equal(t, 3, reports[0].NHostResults)
		assert.Equal(t, 0, reports[1].NHostResults)
		assert.Equal(t, map[string]string{"col1": "val1"}, reports[0].FirstResult)
		assert.Nil(t, reports[1].FirstResult)
		// host has no team, so nil teamID must be forwarded to the datastore.
		assert.Nil(t, capturedTeamID)
	})

	t.Run("admin can list reports for host with team", func(t *testing.T) {
		viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: admin})
		opts := fleet.ListHostReportsOptions{}
		reports, count, _, err := svc.ListHostReports(viewerCtx, hostWithTeam.ID, opts)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
		assert.Len(t, reports, 2)
		// teamID must be forwarded to the datastore so it can scope queries correctly.
		assert.Equal(t, &teamID, capturedTeamID)
	})

	t.Run("observer can list reports", func(t *testing.T) {
		observer := &fleet.User{
			ID:         2,
			GlobalRole: ptr.String(fleet.RoleObserver),
		}
		viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: observer})
		opts := fleet.ListHostReportsOptions{}
		reports, count, _, err := svc.ListHostReports(viewerCtx, hostNoTeam.ID, opts)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
		assert.Len(t, reports, 2)
	})

	t.Run("invalid order_key returns bad request", func(t *testing.T) {
		viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: admin})
		_, _, _, err := svc.ListHostReports(viewerCtx, hostNoTeam.ID, fleet.ListHostReportsOptions{
			ListOptions: fleet.ListOptions{OrderKey: "invalid_key"},
		})
		require.Error(t, err)
		var invalidArgErr *fleet.InvalidArgumentError
		require.ErrorAs(t, err, &invalidArgErr)
	})

	t.Run("unauthenticated gets error", func(t *testing.T) {
		_, _, _, err := svc.ListHostReports(ctx, hostNoTeam.ID, fleet.ListHostReportsOptions{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "forbidden")
	})

	t.Run("team observer cannot read host belonging to a different team", func(t *testing.T) {
		otherTeamID := uint(99)
		teamObserver := &fleet.User{
			ID: 3,
			Teams: []fleet.UserTeam{
				{Team: fleet.Team{ID: otherTeamID}, Role: fleet.RoleObserver},
			},
		}
		// hostWithTeam belongs to teamID=42; teamObserver only has access to teamID=99.
		viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: teamObserver})
		_, _, _, err := svc.ListHostReports(viewerCtx, hostWithTeam.ID, fleet.ListHostReportsOptions{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "forbidden")
	})
}

// TestListHostReportsDatastorePassthrough verifies the options are forwarded
// to the datastore correctly.
func TestListHostReportsDatastorePassthrough(t *testing.T) {
	ds := new(mock.Store)

	teamID := uint(5)
	// Platform is "ubuntu" so PlatformFromHost maps it to "linux" — a non-trivial
	// conversion that would be missed if the service passed host.Platform raw.
	host := &fleet.Host{ID: 7, TeamID: &teamID, Platform: "ubuntu"}

	admin := &fleet.User{
		ID:         1,
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return host, nil
	}

	capturedHostID := uint(0)
	capturedTeamID := (*uint)(nil)
	capturedPlatform := ""
	capturedOpts := fleet.ListHostReportsOptions{}

	ds.ListHostReportsFunc = func(ctx context.Context, hostID uint, tID *uint, hostPlatform string, opts fleet.ListHostReportsOptions, maxQueryReportRows int) ([]*fleet.HostReport, int, *fleet.PaginationMetadata, error) {
		capturedHostID = hostID
		capturedTeamID = tID
		capturedPlatform = hostPlatform
		capturedOpts = opts
		return nil, 0, nil, nil
	}

	svc, ctx := newTestService(t, ds, nil, nil)
	viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: admin})

	opts := fleet.ListHostReportsOptions{
		IncludeReportsDontStoreResults: true,
		ListOptions: fleet.ListOptions{
			Page:           1,
			PerPage:        10,
			OrderKey:       "name",
			OrderDirection: fleet.OrderAscending,
			MatchQuery:     "Alpha",
		},
	}

	_, _, _, err := svc.ListHostReports(viewerCtx, host.ID, opts)
	require.NoError(t, err)

	assert.Equal(t, host.ID, capturedHostID)
	assert.Equal(t, &teamID, capturedTeamID)
	assert.Equal(t, fleet.PlatformFromHost(host.Platform), capturedPlatform)
	assert.True(t, capturedOpts.IncludeReportsDontStoreResults)
	assert.Equal(t, uint(1), capturedOpts.ListOptions.Page)
	assert.Equal(t, uint(10), capturedOpts.ListOptions.PerPage)
	assert.Equal(t, "name", capturedOpts.ListOptions.OrderKey)
	assert.Equal(t, "Alpha", capturedOpts.ListOptions.MatchQuery)
}

// TestHostReportJSONRoundTrip verifies that HostReport serializes and
// deserializes correctly, including the FirstResult and LastFetched fields.
func TestHostReportJSONRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	// This test exercises the HostReport struct's FirstResult field to ensure
	// the data mapping from query_results.data JSON is correct.
	report := &fleet.HostReport{
		ReportID:     1,
		Name:         "USB Devices",
		Description:  "List USB devices",
		LastFetched:  &now,
		FirstResult:  map[string]string{"model": "USB Keyboard", "vendor": "Apple Inc."},
		NHostResults: 0,
	}

	// Verify JSON serialization round-trip
	b, err := json.Marshal(report)
	require.NoError(t, err)

	var decoded fleet.HostReport
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, report.ReportID, decoded.ReportID)
	assert.Equal(t, report.Name, decoded.Name)
	assert.Equal(t, report.Description, decoded.Description)
	assert.Equal(t, report.FirstResult, decoded.FirstResult)
	assert.Equal(t, report.NHostResults, decoded.NHostResults)
	assert.NotNil(t, decoded.LastFetched)
}
