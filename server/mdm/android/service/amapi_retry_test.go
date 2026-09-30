package service

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	ds_mock "github.com/fleetdm/fleet/v4/server/mock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTooManyRequestsProxy points the AMAPI proxy client at a server that always answers 429, and returns
// the number of requests it received.
func newTooManyRequestsProxy(t *testing.T) *atomic.Int64 {
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":429,"message":"rate limit exceeded"}}`))
	}))
	t.Cleanup(srv.Close)
	dev_mode.SetOverride("FLEET_DEV_ANDROID_GOOGLE_CLIENT", "", t)
	dev_mode.SetOverride("FLEET_DEV_ANDROID_PROXY_ENDPOINT", srv.URL+"/", t)
	return &calls
}

func TestNewAMAPIClientRetriesTooManyRequests(t *testing.T) {
	calls := newTooManyRequestsProxy(t)
	client := NewAMAPIClient(t.Context(), slog.New(slog.DiscardHandler), "license")
	require.NotNil(t, client)

	t.Run("retries until the context is done", func(t *testing.T) {
		calls.Store(0)
		ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
		defer cancel()

		_, err := client.EnterprisesDevicesGet(ctx, "enterprises/e/devices/d")
		require.ErrorIs(t, err, context.DeadlineExceeded, "the client should be waiting to retry when the context expires")
		assert.True(t, androidmgmt.IsTooManyRequestsError(err))
		assert.EqualValues(t, 1, calls.Load())
	})

	t.Run("fails fast without retry", func(t *testing.T) {
		calls.Store(0)
		_, err := client.EnterprisesDevicesGet(androidmgmt.WithoutRetry(t.Context()), "enterprises/e/devices/d")
		require.NotErrorIs(t, err, context.DeadlineExceeded)
		assert.True(t, androidmgmt.IsTooManyRequestsError(err))
		assert.EqualValues(t, 1, calls.Load())
	})
}

func TestReconcileAndroidDevicesDoesNotRetryTooManyRequests(t *testing.T) {
	calls := newTooManyRequestsProxy(t)

	ds := new(ds_mock.Store)
	ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
		appCfg := &fleet.AppConfig{}
		appCfg.MDM.AndroidEnabledAndConfigured = true
		return appCfg, nil
	}
	ds.GetEnterpriseFunc = func(context.Context) (*android.Enterprise, error) {
		return &android.Enterprise{EnterpriseID: "e"}, nil
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(context.Context, []fleet.MDMAssetName, sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{}, nil
	}
	ds.ListAndroidEnrolledDevicesForReconcileFunc = func(context.Context) ([]*android.Device, error) {
		return []*android.Device{{HostID: 1, DeviceID: "d"}}, nil
	}

	// With retry enabled, the call would still be waiting to retry when this context expires.
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	err := ReconcileAndroidDevices(ctx, ds, slog.New(slog.DiscardHandler), "license", noopNewActivity)
	require.NotErrorIs(t, err, context.DeadlineExceeded)
	assert.True(t, androidmgmt.IsTooManyRequestsError(err))
	assert.EqualValues(t, 1, calls.Load())
}

type enrollmentTokenCtxService struct {
	android.Service
	gotCtx context.Context
}

func (s *enrollmentTokenCtxService) CreateEnrollmentToken(ctx context.Context, _, _ string, _ bool) (*android.EnrollmentToken, error) {
	s.gotCtx = ctx
	return &android.EnrollmentToken{}, nil
}

func TestEnrollmentTokenEndpointDisablesRetry(t *testing.T) {
	svc := &enrollmentTokenCtxService{}
	enrollmentTokenEndpoint(t.Context(), &enrollmentTokenRequest{EnrollSecret: "secret"}, svc)
	require.NotNil(t, svc.gotCtx)
	assert.True(t, androidmgmt.RetryDisabled(svc.gotCtx))
}
