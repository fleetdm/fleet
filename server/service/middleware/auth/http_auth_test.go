package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	svcmock "github.com/fleetdm/fleet/v4/server/mock/service"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth/externalauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	sessionKey = "c2Vzc2lvbi1rZXk+Lw=="
	jwtToken   = "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJib3QifQ.c2ln"
)

type spyExternalAuth struct {
	calls  int
	viewer *viewer.Viewer
	ip     string
}

func (s *spyExternalAuth) AuthViewer(_ context.Context, _, clientIP string) (*viewer.Viewer, error) {
	s.calls++
	s.ip = clientIP
	if s.viewer == nil {
		return nil, errors.New("rejected")
	}
	return s.viewer, nil
}

func sessionService() *svcmock.Service {
	svc := &svcmock.Service{}
	svc.GetSessionByKeyFunc = func(_ context.Context, key string) (*fleet.Session, error) {
		if key != sessionKey {
			return nil, errors.New("no session")
		}
		return &fleet.Session{ID: 1, UserID: 3, Key: key}, nil
	}
	svc.UserUnauthorizedFunc = func(_ context.Context, id uint) (*fleet.User, error) {
		return &fleet.User{ID: id}, nil
	}
	return svc
}

func requestWithBearer(token string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/latest/fleet/hosts", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	r.RemoteAddr = "192.0.2.10:5555"
	return r
}

// authenticate runs the request through SetRequestsContexts and then AuthenticatedUser,
// as Fleet's handler chain does, and returns the authn method recorded for the request.
func authenticate(t *testing.T, svc fleet.Service, token string, opts ...RequestContextOption) (authz.AuthenticationMethod, *viewer.Viewer, error) {
	method, v, _, err := authenticateWithMarker(t, svc, token, opts...)
	return method, v, err
}

// authenticateWithMarker also reports whether the endpoint saw the external-auth marker.
func authenticateWithMarker(t *testing.T, svc fleet.Service, token string, opts ...RequestContextOption) (authz.AuthenticationMethod, *viewer.Viewer, bool, error) {
	ctx := SetRequestsContexts(svc, opts...)(t.Context(), requestWithBearer(token))
	ac := &authz.AuthorizationContext{}
	ctx = authz.NewContext(ctx, ac)
	var got *viewer.Viewer
	var marked bool
	_, err := AuthenticatedUser(svc, func(ctx context.Context, _ any) (any, error) {
		v, _ := viewer.FromContext(ctx)
		got = &v
		marked = externalauth.FromContext(ctx)
		return nil, nil
	})(ctx, nil)
	return ac.AuthnMethod(), got, marked, err
}

func TestSetRequestsContextsExternalAuth(t *testing.T) {
	clientIP := func(r *http.Request) string { return "ip-from-" + r.RemoteAddr }

	t.Run("session key without authenticator", func(t *testing.T) {
		svc := sessionService()
		method, v, err := authenticate(t, svc, sessionKey)
		require.NoError(t, err)
		assert.Equal(t, authz.AuthnUserToken, method)
		assert.Equal(t, uint(1), v.SessionID())
	})

	t.Run("session key never reaches the authenticator", func(t *testing.T) {
		svc := sessionService()
		spy := &spyExternalAuth{}
		method, v, marked, err := authenticateWithMarker(t, svc, sessionKey, WithExternalAuth(spy, clientIP))
		require.NoError(t, err)
		assert.Zero(t, spy.calls)
		// AuthnUserToken is the zero value, so the marker is what proves nothing leaked.
		assert.False(t, marked, "session requests must not carry the external-auth marker")
		assert.Equal(t, authz.AuthnUserToken, method)
		assert.Equal(t, uint(1), v.SessionID())
	})

	t.Run("JWT authenticates without a session lookup", func(t *testing.T) {
		svc := sessionService()
		spy := &spyExternalAuth{viewer: &viewer.Viewer{User: &fleet.User{ID: 7, APIOnly: true}}}
		method, v, marked, err := authenticateWithMarker(t, svc, jwtToken, WithExternalAuth(spy, clientIP))
		require.NoError(t, err)
		assert.True(t, marked)
		assert.Equal(t, 1, spy.calls)
		assert.Equal(t, "ip-from-192.0.2.10:5555", spy.ip)
		assert.Equal(t, authz.AuthnExternalJWT, method)
		assert.Equal(t, uint(7), v.UserID())
		assert.Nil(t, v.Session)
		assert.False(t, svc.GetSessionByKeyFuncInvoked)
	})

	t.Run("rejected JWT is a 401 without a session lookup", func(t *testing.T) {
		svc := sessionService()
		spy := &spyExternalAuth{}
		_, _, err := authenticate(t, svc, jwtToken, WithExternalAuth(spy, clientIP))
		var authErr *fleet.AuthRequiredError
		require.ErrorAs(t, err, &authErr)
		assert.Equal(t, 1, spy.calls)
		assert.False(t, svc.GetSessionByKeyFuncInvoked)
	})

	t.Run("JWT with no authenticator configured is a 401 without a session lookup", func(t *testing.T) {
		svc := sessionService()
		_, _, err := authenticate(t, svc, jwtToken)
		var authErr *fleet.AuthRequiredError
		require.ErrorAs(t, err, &authErr)
		assert.False(t, svc.GetSessionByKeyFuncInvoked)
	})

	t.Run("external auth context marker", func(t *testing.T) {
		assert.False(t, externalauth.FromContext(t.Context()))
		assert.True(t, externalauth.FromContext(externalauth.NewContext(t.Context())))
	})
}
