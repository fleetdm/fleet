package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomHostVitalsAuth(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.CreateCustomHostVitalFunc = func(ctx context.Context, name string) (fleet.CustomHostVital, error) {
		return fleet.CustomHostVital{ID: 1, Name: name}, nil
	}
	ds.UpdateCustomHostVitalFunc = func(ctx context.Context, id uint, name string) (fleet.CustomHostVital, error) {
		return fleet.CustomHostVital{ID: id, Name: name}, nil
	}
	ds.DeleteCustomHostVitalFunc = func(ctx context.Context, id uint) (string, error) {
		return "Asset tag", nil
	}
	ds.ListCustomHostVitalsFunc = func(ctx context.Context, opt fleet.ListOptions) ([]fleet.CustomHostVital, *fleet.PaginationMetadata, int, error) {
		return nil, &fleet.PaginationMetadata{}, 0, nil
	}
	ds.GetCustomHostVitalsFunc = func(ctx context.Context, ids []uint) ([]fleet.CustomHostVital, error) {
		return []fleet.CustomHostVital{{ID: 1, Name: "Asset tag"}}, nil
	}
	ds.SetHostCustomHostVitalValueFunc = func(ctx context.Context, hostID, vitalID uint, value string) error {
		return nil
	}
	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return &fleet.Host{ID: id}, nil
	}

	globalRoles := []struct {
		name    string
		user    *fleet.User
		readOK  bool
		writeOK bool
	}{
		{"global admin", &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)}, true, true},
		{"global maintainer", &fleet.User{ID: 2, GlobalRole: new(fleet.RoleMaintainer)}, true, true},
		{"global gitops", &fleet.User{ID: 3, GlobalRole: new(fleet.RoleGitOps)}, true, true},
		{"global observer", &fleet.User{ID: 4, GlobalRole: new(fleet.RoleObserver)}, true, false},
		{"global observer+", &fleet.User{ID: 5, GlobalRole: new(fleet.RoleObserverPlus)}, true, false},
		{"team admin", &fleet.User{ID: 6, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}}, true, false},
		{"team maintainer", &fleet.User{ID: 7, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}}, true, false},
		{"team gitops", &fleet.User{ID: 8, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}}, true, false},
		{"team observer", &fleet.User{ID: 9, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}}, true, false},
	}

	for _, tt := range globalRoles {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, _, _, err := svc.ListCustomHostVitals(ctx, fleet.ListOptions{})
			checkAuthErr(t, !tt.readOK, err)

			_, err = svc.CreateCustomHostVital(ctx, "Asset tag")
			checkAuthErr(t, !tt.writeOK, err)

			_, err = svc.UpdateCustomHostVital(ctx, 1, "Asset tag")
			checkAuthErr(t, !tt.writeOK, err)

			err = svc.DeleteCustomHostVital(ctx, 1)
			checkAuthErr(t, !tt.writeOK, err)
		})
	}
}

func TestListCustomHostVitalsPassesSearchQuery(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)}})

	var gotOpts fleet.ListOptions
	ds.ListCustomHostVitalsFunc = func(ctx context.Context, opt fleet.ListOptions) ([]fleet.CustomHostVital, *fleet.PaginationMetadata, int, error) {
		gotOpts = opt
		return nil, &fleet.PaginationMetadata{}, 0, nil
	}

	_, _, _, err := svc.ListCustomHostVitals(ctx, fleet.ListOptions{MatchQuery: "asset"})
	require.NoError(t, err)
	require.True(t, ds.ListCustomHostVitalsFuncInvoked)
	// MatchQuery is forwarded to the datastore (search by name or variable name).
	assert.Equal(t, "asset", gotOpts.MatchQuery)
}

func TestSetHostCustomHostVitalValueAuth(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	hostTeamID := uint(1)
	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return &fleet.Host{ID: id, TeamID: &hostTeamID}, nil
	}
	ds.GetCustomHostVitalsFunc = func(ctx context.Context, ids []uint) ([]fleet.CustomHostVital, error) {
		return []fleet.CustomHostVital{{ID: 1, Name: "Asset tag"}}, nil
	}
	ds.SetHostCustomHostVitalValueFunc = func(ctx context.Context, hostID, vitalID uint, value string) error {
		return nil
	}

	// Per-host value is a host-scoped write (authz type host_custom_vital): global
	// admin/maintainer and admins/maintainers of the host's team can set it;
	// observers, gitops (blocked at the host-list gate), and users of another team
	// cannot.
	testCases := []struct {
		name       string
		user       *fleet.User
		shouldFail bool
	}{
		{"global admin", &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)}, false},
		{"global maintainer", &fleet.User{ID: 2, GlobalRole: new(fleet.RoleMaintainer)}, false},
		{"global gitops", &fleet.User{ID: 3, GlobalRole: new(fleet.RoleGitOps)}, true},
		{"global observer", &fleet.User{ID: 4, GlobalRole: new(fleet.RoleObserver)}, true},
		{"team admin (host team)", &fleet.User{ID: 5, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}}, false},
		{"team maintainer (host team)", &fleet.User{ID: 6, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}}, false},
		{"team observer (host team)", &fleet.User{ID: 7, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}}, true},
		{"team maintainer (other team)", &fleet.User{ID: 8, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}}, true},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})
			err := svc.SetHostCustomHostVitalValue(ctx, 42, 1, "engineering")
			checkAuthErr(t, tt.shouldFail, err)
		})
	}
}

func TestCustomHostVitalNameValidation(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)}})

	ds.CreateCustomHostVitalFunc = func(ctx context.Context, name string) (fleet.CustomHostVital, error) {
		return fleet.CustomHostVital{ID: 1, Name: name}, nil
	}

	invalidNames := []struct {
		name  string
		value string
	}{
		{"empty", ""},
		{"leading space", " Asset tag"},
		{"trailing space", "Asset tag "},
		{"leading tab", "\tAsset tag"},
		{"trailing newline", "Asset tag\n"},
	}
	for _, tt := range invalidNames {
		t.Run("reject "+tt.name, func(t *testing.T) {
			ds.CreateCustomHostVitalFuncInvoked = false
			_, err := svc.CreateCustomHostVital(ctx, tt.value)
			require.Error(t, err)
			assert.False(t, ds.CreateCustomHostVitalFuncInvoked)
		})
	}

	validNames := []struct {
		name  string
		value string
	}{
		{"internal spaces", "Asset tag"},
		{"lowercase", "asset tag"},
		{"mixed case with digits", "Rack 12B Location"},
	}
	for _, tt := range validNames {
		t.Run("accept "+tt.name, func(t *testing.T) {
			vital, err := svc.CreateCustomHostVital(ctx, tt.value)
			require.NoError(t, err)
			require.NotNil(t, vital)
		})
	}
}
