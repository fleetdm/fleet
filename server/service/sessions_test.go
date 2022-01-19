package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

func TestSessionAuth(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.ListSessionsForUserFunc = func(ctx context.Context, id uint) ([]*fleet.Session, error) {
		if id == 999 {
			return []*fleet.Session{
				{ID: 1, UserID: id, AccessedAt: time.Now()},
			}, nil
		}
		return nil, nil
	}
	ds.SessionByIDFunc = func(ctx context.Context, id uint) (*fleet.Session, error) {
		return &fleet.Session{ID: id, UserID: 999, AccessedAt: time.Now()}, nil
	}
	ds.DestroySessionFunc = func(ctx context.Context, ssn *fleet.Session) error {
		return nil
	}
	ds.MarkSessionAccessedFunc = func(ctx context.Context, ssn *fleet.Session) error {
		return nil
	}

	testCases := []struct {
		name            string
		user            *fleet.User
		shouldFailWrite bool
		shouldFailRead  bool
	}{
		{
			"global admin",
			&fleet.User{ID: 111, GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{ID: 111, GlobalRole: ptr.String(fleet.RoleMaintainer)},
			true,
			true,
		},
		{
			"global observer",
			&fleet.User{ID: 111, GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			true,
		},
		{
			"owner user",
			&fleet.User{ID: 999},
			false,
			false,
		},
		{
			"non-owner user",
			&fleet.User{ID: 888},
			true,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})

			// TODO: I think the auth in this one is wrong, see TODO comment over there.
			//_, err := svc.GetInfoAboutSessionsForUser(ctx, tt.user.ID)
			//checkAuthErr(t, tt.shouldFailWrite, err)

			_, err := svc.GetInfoAboutSession(ctx, 1)
			checkAuthErr(t, tt.shouldFailRead, err)

			err = svc.DeleteSession(ctx, 1)
			checkAuthErr(t, tt.shouldFailWrite, err)
		})
	}
}
