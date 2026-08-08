package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockService struct {
	mock.Mock
	fleet.Service
}

func (m *mockService) GetSessionByKey(ctx context.Context, sessionKey string) (*fleet.Session, error) {
	args := m.Called(ctx, sessionKey)
	if ret := args.Get(0); ret != nil {
		return ret.(*fleet.Session), nil
	}
	return nil, args.Error(1)
}

func (m *mockService) UserUnauthorized(ctx context.Context, userId uint) (*fleet.User, error) {
	args := m.Called(ctx, userId)
	if ret := args.Get(0); ret != nil {
		return ret.(*fleet.User), nil
	}
	return nil, args.Error(1)
}

var testConfig = config.FleetConfig{
	Auth: config.AuthConfig{},
}

func TestDebugHandlerAuthenticationTokenMissing(t *testing.T) {
	handler := MakeDebugHandler(&mockService{}, testConfig, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "https://fleetdm.com/debug/pprof/profile", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestDebugHandlerAuthenticationSessionInvalid(t *testing.T) {
	svc := &mockService{}
	svc.On(
		"GetSessionByKey",
		mock.Anything,
		"fake_session_key",
	).Return(nil, errors.New("invalid session"))

	handler := MakeDebugHandler(svc, testConfig, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "https://fleetdm.com/debug/pprof/profile", nil)
	req.Header.Add("Authorization", "BEARER fake_session_key")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestDebugHandlerAuthenticationFailsDueToRole(t *testing.T) {
	for test, user := range map[string]fleet.User{
		"no role":                {},
		"global observer role":   {GlobalRole: ptr.String(fleet.RoleObserver)},
		"global maintainer role": {GlobalRole: ptr.String(fleet.RoleMaintainer)},
		"non-global role":        {Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1, Name: "foo"}, Role: fleet.RoleAdmin}}},
	} {
		t.Run(test, func(t *testing.T) {
			svc := &mockService{}
			svc.On(
				"GetSessionByKey",
				mock.Anything,
				"fake_session_key",
			).Return(&fleet.Session{UserID: 42, ID: 1}, nil)
			svc.On(
				"UserUnauthorized",
				mock.Anything,
				uint(42),
			).Return(&user, nil)

			handler := MakeDebugHandler(svc, testConfig, nil, nil, nil)

			req := httptest.NewRequest(http.MethodGet, "https://fleetdm.com/debug/pprof/cmdline", nil)
			req.Header.Add("Authorization", "BEARER fake_session_key")
			res := httptest.NewRecorder()

			handler.ServeHTTP(res, req)
			assert.Equal(t, http.StatusForbidden, res.Code)
		})
	}
}

func TestDebugHandlerAuthenticationFailsForRestrictedAPIOnlyUser(t *testing.T) {
	// A global-admin API-only token scoped to an endpoint allowlist must not reach the debug
	// routes: those routes are not in the public API catalog, so they can never be allowlisted.
	svc := &mockService{}
	svc.On(
		"GetSessionByKey",
		mock.Anything,
		"fake_session_key",
	).Return(&fleet.Session{UserID: 42, ID: 1}, nil)
	svc.On(
		"UserUnauthorized",
		mock.Anything,
		uint(42),
	).Return(&fleet.User{
		GlobalRole:   ptr.String(fleet.RoleAdmin),
		APIOnly:      true,
		APIEndpoints: []fleet.APIEndpointRef{{Method: "GET", Path: "/api/v1/fleet/hosts"}},
	}, nil)

	handler := MakeDebugHandler(svc, testConfig, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "https://fleetdm.com/debug/pprof/cmdline", nil)
	req.Header.Add("Authorization", "BEARER fake_session_key")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	assert.Equal(t, http.StatusForbidden, res.Code)
}

func TestDebugHandlerAuthenticationSucceeds(t *testing.T) {
	// An unrestricted API-only admin (empty APIEndpoints) retains full access, matching the main
	// API path where APIOnlyEndpointCheck is a no-op for tokens with no endpoint restrictions.
	for test, user := range map[string]fleet.User{
		"admin session":            {GlobalRole: ptr.String(fleet.RoleAdmin)},
		"unrestricted api-only":    {GlobalRole: ptr.String(fleet.RoleAdmin), APIOnly: true},
		"api-only empty allowlist": {GlobalRole: ptr.String(fleet.RoleAdmin), APIOnly: true, APIEndpoints: []fleet.APIEndpointRef{}},
	} {
		t.Run(test, func(t *testing.T) {
			svc := &mockService{}
			svc.On(
				"GetSessionByKey",
				mock.Anything,
				"fake_session_key",
			).Return(&fleet.Session{UserID: 42, ID: 1}, nil)
			svc.On(
				"UserUnauthorized",
				mock.Anything,
				uint(42),
			).Return(&user, nil)

			handler := MakeDebugHandler(svc, testConfig, nil, nil, nil)

			req := httptest.NewRequest(http.MethodGet, "https://fleetdm.com/debug/pprof/cmdline", nil)
			req.Header.Add("Authorization", "BEARER fake_session_key")
			res := httptest.NewRecorder()

			handler.ServeHTTP(res, req)
			assert.Equal(t, http.StatusOK, res.Code)
		})
	}
}
