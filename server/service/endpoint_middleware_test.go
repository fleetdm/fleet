package service

import (
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/server/contexts/viewer"
	"github.com/kolide/kolide-ose/server/datastore"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

// TestEndpointPermissions tests that
// the endpoint.Middleware correctly grants or denies
// permissions to access or modify resources
func TestEndpointPermissions(t *testing.T) {
	req := struct{}{}
	ds, _ := datastore.New("inmem", "")
	createTestUsers(t, ds)
	admin1, _ := ds.User("admin1")
	user1, _ := ds.User("user1")
	user2, _ := ds.User("user2")
	user2.Enabled = false

	e := endpoint.Nop // a test endpoint
	var endpointTests = []struct {
		endpoint endpoint.Endpoint
		// who is making the request
		vc *viewer.Viewer
		// what resource are we editing
		requestID uint
		// what error to expect
		wantErr interface{}
		// custom request struct
		request interface{}
	}{
		{
			endpoint: mustBeAdmin(e),
			wantErr:  errNoContext,
		},
		{
			endpoint: canReadUser(e),
			wantErr:  errNoContext,
		},
		{
			endpoint: canModifyUser(e),
			wantErr:  errNoContext,
		},
		{
			endpoint: mustBeAdmin(e),
			vc:       &viewer.Viewer{User: admin1},
		},
		{
			endpoint: mustBeAdmin(e),
			vc:       &viewer.Viewer{User: user1},
			wantErr:  permissionError{message: "must be an admin"},
		},
		{
			endpoint: canModifyUser(e),
			vc:       &viewer.Viewer{User: admin1},
		},
		{
			endpoint: canModifyUser(e),
			vc:       &viewer.Viewer{User: user1},
			wantErr:  permissionError{message: "no write permissions on user"},
		},
		{
			endpoint:  canModifyUser(e),
			vc:        &viewer.Viewer{User: user1},
			requestID: admin1.ID,
			wantErr:   permissionError{message: "no write permissions on user"},
		},
		{
			endpoint:  canReadUser(e),
			vc:        &viewer.Viewer{User: user1},
			requestID: admin1.ID,
		},
		{
			endpoint:  canReadUser(e),
			vc:        &viewer.Viewer{User: user2},
			requestID: admin1.ID,
			wantErr:   permissionError{message: "no read permissions on user"},
		},
		{
			endpoint: validateModifyUserRequest(e),
			request:  modifyUserRequest{},
			wantErr:  errNoContext,
		},
		{
			endpoint: validateModifyUserRequest(e),
			request:  modifyUserRequest{payload: kolide.UserPayload{Enabled: boolPtr(true)}},
			vc:       &viewer.Viewer{User: user1},
			wantErr:  permissionError{message: "unauthorized: must be an admin"},
		},
	}

	for _, tt := range endpointTests {
		tt := tt
		t.Run("", func(st *testing.T) {
			st.Parallel()
			ctx := context.Background()
			if tt.vc != nil {
				ctx = viewer.NewContext(ctx, *tt.vc)
			}
			if tt.requestID != 0 {
				ctx = context.WithValue(ctx, "request-id", tt.requestID)
			}
			var request interface{}
			if tt.request != nil {
				request = tt.request
			} else {
				request = req
			}
			_, eerr := tt.endpoint(ctx, request)
			assert.IsType(st, tt.wantErr, eerr)
			if ferr, ok := eerr.(permissionError); ok {
				assert.Equal(st, tt.wantErr.(permissionError).message, ferr.Error())
			}
		})
	}
}

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
	ds, err := datastore.New("inmem", "")
	require.Nil(t, err)
	svc, err := newTestService(ds)
	require.Nil(t, err)

	endpoint := authenticatedHost(
		svc,
		func(ctx context.Context, request interface{}) (interface{}, error) {
			return nil, nil
		},
	)

	ctx := context.Background()
	goodNodeKey, err := svc.EnrollAgent(ctx, "", "host123")
	assert.Nil(t, err)
	require.NotEmpty(t, goodNodeKey)

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
			_, err = endpoint(context.Background(), r)
			if tt.shouldErr {
				assert.IsType(t, osqueryError{}, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

}
