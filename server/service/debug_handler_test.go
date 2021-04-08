package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/server/config"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockService struct {
	mock.Mock
	kolide.Service
}

func (m *mockService) GetSessionByKey(ctx context.Context, sessionKey string) (*kolide.Session, error) {
	args := m.Called(ctx, sessionKey)
	if ret := args.Get(0); ret != nil {
		return ret.(*kolide.Session), nil
	}
	return nil, args.Error(1)
}

func (m *mockService) User(ctx context.Context, userId uint) (*kolide.User, error) {
	args := m.Called(ctx, userId)
	if ret := args.Get(0); ret != nil {
		return ret.(*kolide.User), nil
	}
	return nil, args.Error(1)
}

var testConfig = config.KolideConfig{
	Auth: config.AuthConfig{
		JwtKey: "insecure",
	},
}

func TestDebugHandlerAuthenticationTokenMissing(t *testing.T) {
	handler := MakeDebugHandler(&mockService{}, testConfig, nil)

	req := httptest.NewRequest(http.MethodGet, "https://fleetdm.com/debug/pprof/profile", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestDebugHandlerAuthenticationTokenInvalid(t *testing.T) {
	handler := MakeDebugHandler(&mockService{}, testConfig, nil)

	req := httptest.NewRequest(http.MethodGet, "https://fleetdm.com/debug/pprof/profile", nil)
	req.Header.Add("Authorization", "BEARER foobar")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestDebugHandlerAuthenticationSessionInvalid(t *testing.T) {
	svc := &mockService{}
	svc.On(
		"GetSessionByKey",
		mock.Anything,
		"session",
	).Return(nil, errors.New("invalid session"))

	handler := MakeDebugHandler(svc, testConfig, nil)

	req := httptest.NewRequest(http.MethodGet, "https://fleetdm.com/debug/pprof/profile", nil)
	req.Header.Add("Authorization", "BEARER eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzZXNzaW9uX2tleSI6InNlc3Npb24iLCJpYXQiOjE1MTYyMzkwMjJ9.YZIL9fKxfVg7fCms4CTKCPT2w8x8N3e2pciV_h0OvTk")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestDebugHandlerAuthenticationDisabled(t *testing.T) {
	svc := &mockService{}
	svc.On(
		"GetSessionByKey",
		mock.Anything,
		"session",
	).Return(&kolide.Session{UserID: 42, ID: 1}, nil)
	svc.On(
		"User",
		mock.Anything,
		uint(42),
	).Return(&kolide.User{Enabled: false}, nil)
	handler := MakeDebugHandler(svc, testConfig, nil)

	req := httptest.NewRequest(http.MethodGet, "https://fleetdm.com/debug/pprof/profile", nil)
	req.Header.Add("Authorization", "BEARER eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzZXNzaW9uX2tleSI6InNlc3Npb24iLCJpYXQiOjE1MTYyMzkwMjJ9.YZIL9fKxfVg7fCms4CTKCPT2w8x8N3e2pciV_h0OvTk")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	assert.Equal(t, http.StatusForbidden, res.Code)
}

func TestDebugHandlerAuthenticationSuccess(t *testing.T) {
	svc := &mockService{}
	svc.On(
		"GetSessionByKey",
		mock.Anything,
		"session",
	).Return(&kolide.Session{UserID: 42, ID: 1}, nil)
	svc.On(
		"User",
		mock.Anything,
		uint(42),
	).Return(&kolide.User{Enabled: true}, nil)

	handler := MakeDebugHandler(svc, testConfig, nil)

	req := httptest.NewRequest(http.MethodGet, "https://fleetdm.com/debug/pprof/cmdline", nil)
	req.Header.Add("Authorization", "BEARER eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzZXNzaW9uX2tleSI6InNlc3Npb24iLCJpYXQiOjE1MTYyMzkwMjJ9.YZIL9fKxfVg7fCms4CTKCPT2w8x8N3e2pciV_h0OvTk")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
}
