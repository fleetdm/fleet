package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/certserial"
	"github.com/fleetdm/fleet/v4/server/contexts/devicesso"
	"github.com/fleetdm/fleet/v4/server/fleet"
	svcmock "github.com/fleetdm/fleet/v4/server/mock/service"
	"github.com/go-kit/kit/endpoint"
	"github.com/stretchr/testify/require"
)

func newDeviceSSOGateTestService(host *fleet.Host) *svcmock.Service {
	svc := new(svcmock.Service)
	svc.AuthenticateDeviceFunc = func(ctx context.Context, token string) (*fleet.Host, bool, error) {
		return host, false, nil
	}
	svc.AuthenticateDeviceByCertificateFunc = func(ctx context.Context, serial uint64, uuid string) (*fleet.Host, bool, error) {
		return host, false, nil
	}
	svc.AuthenticateIDeviceByURLFunc = func(ctx context.Context, uuid string) (*fleet.Host, bool, error) {
		return host, false, nil
	}
	return svc
}

func gatedDeviceChain(svc fleet.Service, endpointFn endpoint.Endpoint) endpoint.Endpoint {
	return authenticatedDevice(svc, slog.New(slog.DiscardHandler), requireDeviceSSOSession(svc)(endpointFn))
}

func TestAuthenticatedDeviceSSOGate(t *testing.T) {
	host := &fleet.Host{ID: 1, UUID: "host-uuid-1", Platform: "darwin"}
	ssoRequired := fleet.NewDeviceSSORequiredError("no device sso session")

	t.Run("gated endpoint passes the resolved host and the cookie session", func(t *testing.T) {
		svc := newDeviceSSOGateTestService(host)
		var gotSessionID string
		var gotHost *fleet.Host
		svc.RequireDeviceSSOSessionFunc = func(ctx context.Context, host *fleet.Host, sessionID string) error {
			gotHost, gotSessionID = host, sessionID
			return nil
		}

		mw := gatedDeviceChain(svc, func(ctx context.Context, request any) (any, error) {
			return "success", nil
		})

		ctx := devicesso.NewContext(t.Context(), "session-id-from-cookie")
		resp, err := mw(ctx, mockDeviceAuthRequest{Token: "device-token"})
		require.NoError(t, err)
		require.Equal(t, "success", resp)
		require.Equal(t, "session-id-from-cookie", gotSessionID)
		// the host the gate binds the session against is the one authentication
		// resolved, not something it looks up for itself
		require.Equal(t, host, gotHost)
	})

	// iOS/iPadOS reach the device API by client certificate or by device UUID in
	// the URL rather than by rotating token; the gate runs after host resolution
	// so it has to cover all three.
	t.Run("gate covers certificate authentication", func(t *testing.T) {
		svc := newDeviceSSOGateTestService(host)
		svc.RequireDeviceSSOSessionFunc = func(ctx context.Context, host *fleet.Host, sessionID string) error {
			return ssoRequired
		}

		mw := gatedDeviceChain(svc, func(ctx context.Context, request any) (any, error) {
			return "success", nil
		})

		ctx := certserial.NewContext(t.Context(), 1234)
		_, err := mw(ctx, mockDeviceAuthRequest{Token: host.UUID})
		require.ErrorIs(t, err, ssoRequired)
		require.True(t, svc.AuthenticateDeviceByCertificateFuncInvoked)
	})

	t.Run("gate covers device URL authentication", func(t *testing.T) {
		svc := newDeviceSSOGateTestService(host)
		svc.AuthenticateDeviceFunc = func(ctx context.Context, token string) (*fleet.Host, bool, error) {
			return nil, false, newNotFoundError()
		}
		svc.RequireDeviceSSOSessionFunc = func(ctx context.Context, host *fleet.Host, sessionID string) error {
			return ssoRequired
		}

		mw := gatedDeviceChain(svc, func(ctx context.Context, request any) (any, error) {
			return "success", nil
		})

		_, err := mw(t.Context(), mockDeviceAuthRequest{Token: host.UUID})
		require.ErrorIs(t, err, ssoRequired)
		require.True(t, svc.AuthenticateIDeviceByURLFuncInvoked)
	})

	t.Run("an invalid token is rejected before the gate", func(t *testing.T) {
		svc := new(svcmock.Service)
		svc.AuthenticateDeviceFunc = func(ctx context.Context, token string) (*fleet.Host, bool, error) {
			return nil, false, newNotFoundError()
		}
		svc.AuthenticateIDeviceByURLFunc = func(ctx context.Context, uuid string) (*fleet.Host, bool, error) {
			return nil, false, newNotFoundError()
		}
		svc.RequireDeviceSSOSessionFunc = func(ctx context.Context, host *fleet.Host, sessionID string) error {
			return nil
		}

		mw := gatedDeviceChain(svc, func(ctx context.Context, request any) (any, error) {
			return "success", nil
		})

		_, err := mw(t.Context(), mockDeviceAuthRequest{Token: "bad-token"})
		require.Error(t, err)
		var ssoErr *fleet.DeviceSSORequiredError
		require.NotErrorAs(t, err, &ssoErr, "a bad token must not tell the browser to sign in")
		require.False(t, svc.RequireDeviceSSOSessionFuncInvoked)
	})
}

func TestDeviceSSORequiredErrorResponse(t *testing.T) {
	rr := httptest.NewRecorder()
	encodeError(t.Context(), fleet.NewDeviceSSORequiredError("no device sso session"), rr)

	require.Equal(t, http.StatusUnauthorized, rr.Code)

	var body struct {
		Message     string              `json:"message"`
		Errors      []map[string]string `json:"errors"`
		SSORequired bool                `json:"sso_required"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.True(t, body.SSORequired)
	require.Equal(t, "Single sign-on required", body.Message)
	require.Len(t, body.Errors, 1)

	// The internal reason stays out of the response.
	require.NotContains(t, rr.Body.String(), "no device sso session")
}

func TestNonSSOErrorResponseHasNoMarker(t *testing.T) {
	rr := httptest.NewRecorder()
	encodeError(t.Context(), fleet.NewAuthRequiredError("invalid device token"), rr)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
	require.NotContains(t, rr.Body.String(), "sso_required")
}
