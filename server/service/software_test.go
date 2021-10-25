package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_ListSoftware(t *testing.T) {
	ds := new(mock.Store)

	var calledWithTeamID *uint
	var calledWithOpt fleet.SoftwareListOptions
	ds.ListSoftwareFunc = func(ctx context.Context, opt fleet.SoftwareListOptions) ([]fleet.Software, error) {
		calledWithTeamID = opt.TeamID
		calledWithOpt = opt
		return []fleet.Software{}, nil
	}

	user := &fleet.User{ID: 3, Email: "foo@bar.com", GlobalRole: ptr.String(fleet.RoleObserver)}

	svc := newTestService(ds, nil, nil)
	ctx := context.Background()
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})

	_, err := svc.ListSoftware(ctx, fleet.SoftwareListOptions{TeamID: ptr.Uint(42), ListOptions: fleet.ListOptions{PerPage: 77, Page: 4}})
	require.NoError(t, err)

	assert.True(t, ds.ListSoftwareFuncInvoked)
	assert.Equal(t, ptr.Uint(42), calledWithTeamID)
	assert.Equal(t, fleet.ListOptions{PerPage: 77, Page: 4}, calledWithOpt.ListOptions)
}
