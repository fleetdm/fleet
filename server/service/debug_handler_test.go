package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
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

func TestDebugHandlerAuthenticationSuccess(t *testing.T) {
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
	).Return(&fleet.User{}, nil)

	handler := MakeDebugHandler(svc, testConfig, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "https://fleetdm.com/debug/pprof/cmdline", nil)
	req.Header.Add("Authorization", "BEARER fake_session_key")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
}
