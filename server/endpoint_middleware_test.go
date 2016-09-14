package server

import (
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/datastore"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

// TestEndpointPermissions tests that
// the endpoint.Middleware correctly grants or denies
// permissions to access or modify resources
func TestEndpointPermissions(t *testing.T) {
	ctx := context.Background()
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
		vc *viewerContext
		// what resource are we editing
		requestID uint
		// what error to expect
		wantErr interface{}
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
			vc:       &viewerContext{user: admin1},
		},
		{
			endpoint: mustBeAdmin(e),
			vc:       &viewerContext{user: user1},
			wantErr:  forbiddenError{"must be an admin"},
		},
		{
			endpoint: canModifyUser(e),
			vc:       &viewerContext{user: admin1},
		},
		{
			endpoint: canModifyUser(e),
			vc:       &viewerContext{user: user1},
			wantErr:  forbiddenError{message: "no write permissions on user"},
		},
		{
			endpoint:  canModifyUser(e),
			vc:        &viewerContext{user: user1},
			requestID: admin1.ID,
			wantErr:   forbiddenError{message: "no write permissions on user"},
		},
		{
			endpoint:  canReadUser(e),
			vc:        &viewerContext{user: user1},
			requestID: admin1.ID,
		},
		{
			endpoint:  canReadUser(e),
			vc:        &viewerContext{user: user2},
			requestID: admin1.ID,
			wantErr:   forbiddenError{message: "no read permissions on user"},
		},
	}

	for _, tt := range endpointTests {
		if tt.vc != nil {
			ctx = context.WithValue(ctx, "viewerContext", tt.vc)
		}
		if tt.requestID != 0 {
			ctx = context.WithValue(ctx, "request-id", tt.requestID)
		}
		_, eerr := tt.endpoint(ctx, req)
		assert.IsType(t, tt.wantErr, eerr)
		if ferr, ok := eerr.(forbiddenError); ok {
			assert.Equal(t, tt.wantErr.(forbiddenError).message, ferr.message)
		}
	}
}
