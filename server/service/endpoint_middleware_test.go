package service

import (
	"context"
	"testing"
	"time"

	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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

// TestGetNodeKey tests the reflection logic for pulling the node key from
// various (fake) request types
func TestGetNodeKey(t *testing.T) {
	type Foo struct {
		Foo     string
		NodeKey string
	}

	type Bar struct {
		Bar     string
		NodeKey string
	}

	type Nope struct {
		Nope string
	}

	type Almost struct {
		NodeKey int
	}

	var getNodeKeyTests = []struct {
		i         interface{}
		expectKey string
		shouldErr bool
	}{
		{
			i:         Foo{Foo: "foo", NodeKey: "fookey"},
			expectKey: "fookey",
			shouldErr: false,
		},
		{
			i:         Bar{Bar: "bar", NodeKey: "barkey"},
			expectKey: "barkey",
			shouldErr: false,
		},
		{
			i:         Nope{Nope: "nope"},
			expectKey: "",
			shouldErr: true,
		},
		{
			i:         Almost{NodeKey: 10},
			expectKey: "",
			shouldErr: true,
		},
	}

	for _, tt := range getNodeKeyTests {
		t.Run("", func(t *testing.T) {
			key, err := getNodeKey(tt.i)
			assert.Equal(t, tt.expectKey, key)
			if tt.shouldErr {
				assert.IsType(t, osqueryError{}, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestAuthenticatedHost(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	expectedHost := fleet.Host{Hostname: "foo!"}
	goodNodeKey := "foo bar baz bing bang boom"

	ds.AuthenticateHostFunc = func(secret string) (*fleet.Host, error) {
		switch secret {
		case goodNodeKey:
			return &expectedHost, nil
		default:
			return nil, errors.New("no host found")

		}
	}
	ds.MarkHostSeenFunc = func(host *fleet.Host, t time.Time) error {
		return nil
	}

	endpoint := authenticatedHost(
		svc,
		func(ctx context.Context, request interface{}) (interface{}, error) {
			host, ok := hostctx.FromContext(ctx)
			assert.True(t, ok)
			assert.Equal(t, expectedHost, host)
			return nil, nil
		},
	)

	var authenticatedHostTests = []struct {
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
			var r = struct{ NodeKey string }{NodeKey: tt.nodeKey}
			_, err := endpoint(context.Background(), r)
			if tt.shouldErr {
				assert.IsType(t, osqueryError{}, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

}
