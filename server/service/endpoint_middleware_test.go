package service

import (
	"context"
	"errors"
	"testing"

	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO update this test for new patterns

// TestEndpointPermissions tests that
// the endpoint.Middleware correctly grants or denies
// permissions to access or modify resources
// func TestEndpointPermissions(t *testing.T) {
// 	req := struct{}{}
// 	ds, err := inmem.New(config.TestConfig())
// 	assert.Nil(t, err)

// 	createTestUsers(t, ds)

// 	admin1, err := ds.User("admin1")
// 	assert.Nil(t, err)
// 	admin1Session, err := ds.NewSession(&fleet.Session{
// 		UserID: admin1.ID,
// 		Key:    "admin1",
// 	})
// 	assert.Nil(t, err)

// 	user1, err := ds.User("user1")
// 	assert.Nil(t, err)
// 	user1Session, err := ds.NewSession(&fleet.Session{
// 		UserID: user1.ID,
// 		Key:    "user1",
// 	})
// 	assert.Nil(t, err)

// 	user2, err := ds.User("user2")
// 	assert.Nil(t, err)
// 	user2Session, err := ds.NewSession(&fleet.Session{
// 		UserID: user2.ID,
// 		Key:    "user2",
// 	})
// 	assert.Nil(t, err)
// 	user2.Enabled = false

// 	e := endpoint.Nop // a test endpoint
// 	var endpointTests = []struct {
// 		endpoint endpoint.Endpoint
// 		// who is making the request
// 		vc *viewer.Viewer
// 		// what resource are we editing
// 		requestID uint
// 		// what error to expect
// 		wantErr interface{}
// 		// custom request struct
// 		request interface{}
// 	}{
// 		{
// 			endpoint: canReadUser(e),
// 			wantErr:  fleet.ErrNoContext,
// 		},
// 		{
// 			endpoint: canModifyUser(e),
// 			wantErr:  fleet.ErrNoContext,
// 		},
// 		{
// 			endpoint: canModifyUser(e),
// 			vc:       &viewer.Viewer{User: admin1, Session: admin1Session},
// 		},
// 		{
// 			endpoint: canModifyUser(e),
// 			vc:       &viewer.Viewer{User: user1, Session: user1Session},
// 			wantErr:  permissionError{message: "no write permissions on user"},
// 		},
// 		{
// 			endpoint:  canModifyUser(e),
// 			vc:        &viewer.Viewer{User: user1, Session: user1Session},
// 			requestID: admin1.ID,
// 			wantErr:   permissionError{message: "no write permissions on user"},
// 		},
// 		{
// 			endpoint:  canReadUser(e),
// 			vc:        &viewer.Viewer{User: user1, Session: user1Session},
// 			requestID: admin1.ID,
// 		},
// 		{
// 			endpoint:  canReadUser(e),
// 			vc:        &viewer.Viewer{User: user2, Session: user2Session},
// 			requestID: admin1.ID,
// 			wantErr:   permissionError{message: "no read permissions on user"},
// 		},
// 	}

// 	for _, tt := range endpointTests {
// 		tt := tt
// 		t.Run("", func(st *testing.T) {
// 			st.Parallel()
// 			ctx := context.Background()
// 			if tt.vc != nil {
// 				ctx = viewer.NewContext(ctx, *tt.vc)
// 			}
// 			if tt.requestID != 0 {
// 				ctx = context.WithValue(ctx, "request-id", tt.requestID)
// 			}
// 			var request interface{}
// 			if tt.request != nil {
// 				request = tt.request
// 			} else {
// 				request = req
// 			}
// 			_, eerr := tt.endpoint(ctx, request)
// 			assert.IsType(st, tt.wantErr, eerr)
// 			if ferr, ok := eerr.(permissionError); ok {
// 				assert.Equal(st, tt.wantErr.(permissionError).message, ferr.Error())
// 			}
// 		})
// 	}
// }

type testNodeKeyRequest struct {
	NodeKey string
}

func (r *testNodeKeyRequest) hostNodeKey() string { return r.NodeKey }

func TestAuthenticatedHost(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	expectedHost := fleet.Host{Hostname: "foo!"}
	goodNodeKey := "foo bar baz bing bang boom"

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.LoadHostByNodeKeyFunc = func(ctx context.Context, nodeKey string) (*fleet.Host, error) {
		switch nodeKey {
		case goodNodeKey:
			return &expectedHost, nil
		default:
			return nil, errors.New("no host found")

		}
	}

	endpoint := authenticatedHost(
		svc,
		kitlog.NewNopLogger(),
		func(ctx context.Context, request interface{}) (interface{}, error) {
			host, ok := hostctx.FromContext(ctx)
			assert.True(t, ok)
			assert.Equal(t, &expectedHost, host)
			return nil, nil
		},
	)

	authenticatedHostTests := []struct {
		nodeKey   string
		shouldErr bool
	}{
		{
			nodeKey:   "invalid",
			shouldErr: true,
		},
		{
			nodeKey:   "",
			shouldErr: true,
		},
		{
			nodeKey:   goodNodeKey,
			shouldErr: false,
		},
	}

	for _, tt := range authenticatedHostTests {
		t.Run("", func(t *testing.T) {
			r := &testNodeKeyRequest{NodeKey: tt.nodeKey}
			_, err := endpoint(ctx, r)
			if tt.shouldErr {
				assert.IsType(t, &osqueryError{}, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestAuthenticatedUserMW(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	authenticatedUserTests := []struct {
		user      *fleet.User
		shouldErr bool
	}{
		{
			user: &fleet.User{
				ID:                       32,
				Name:                     "name",
				Email:                    "em@il.com",
				AdminForcedPasswordReset: true,
				SSOEnabled:               true,
			},
			shouldErr: false,
		},
		{
			user: &fleet.User{
				ID:                       32,
				Name:                     "name",
				Email:                    "em@il.com",
				AdminForcedPasswordReset: true,
				SSOEnabled:               false,
			},
			shouldErr: true,
		},
	}

	for _, tt := range authenticatedUserTests {
		t.Run("", func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			nextCalled := false
			endpoint := authenticatedUser(svc, func(ctx context.Context, request interface{}) (response interface{}, err error) {
				nextCalled = true
				return nil, nil
			})
			_, err := endpoint(ctx, nil)
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, nextCalled)
			}
		})
	}
}
