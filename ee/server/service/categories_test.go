package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSelfServiceSoftwareCategoriesForHost(t *testing.T) {
	ctx := t.Context()

	t.Run("wraps datastore errors", func(t *testing.T) {
		ds := new(mock.Store)
		ds.ListSoftwareCategoriesFunc = func(ctx context.Context, teamID uint) ([]fleet.SoftwareCategory, error) {
			return nil, errors.New("boom")
		}
		svc := newTestService(t, ds)

		_, err := svc.ListSelfServiceSoftwareCategoriesForHost(ctx, &fleet.Host{ID: 1, TeamID: new(uint(7))})
		require.ErrorContains(t, err, "list self-service software categories for host")
		assert.True(t, ds.ListSoftwareCategoriesFuncInvoked)
	})

	t.Run("returns the host team's categories", func(t *testing.T) {
		ds := new(mock.Store)
		var gotTeamID uint
		ds.ListSoftwareCategoriesFunc = func(ctx context.Context, teamID uint) ([]fleet.SoftwareCategory, error) {
			gotTeamID = teamID
			return []fleet.SoftwareCategory{{ID: 1, Name: "🌎 Browsers", TeamID: teamID}}, nil
		}
		svc := newTestService(t, ds)

		got, err := svc.ListSelfServiceSoftwareCategoriesForHost(ctx, &fleet.Host{ID: 1, TeamID: new(uint(7))})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "🌎 Browsers", got[0].Name)
		assert.Equal(t, uint(7), gotTeamID)
	})

	t.Run("no-team host queries team 0", func(t *testing.T) {
		ds := new(mock.Store)
		var gotTeamID uint
		ds.ListSoftwareCategoriesFunc = func(ctx context.Context, teamID uint) ([]fleet.SoftwareCategory, error) {
			gotTeamID = teamID
			return nil, nil
		}
		svc := newTestService(t, ds)

		_, err := svc.ListSelfServiceSoftwareCategoriesForHost(ctx, &fleet.Host{ID: 1, TeamID: nil})
		require.NoError(t, err)
		assert.True(t, ds.ListSoftwareCategoriesFuncInvoked)
		assert.Equal(t, uint(0), gotTeamID)
	})
}
