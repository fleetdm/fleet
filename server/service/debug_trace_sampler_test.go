package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	mockds "github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// adminAuthedRequest builds a request and primes the mockService so the debug auth middleware lets it through as a global
// admin.
func adminAuthedRequest(t *testing.T, method, target string, body string) (*mockService, *http.Request) {
	t.Helper()
	svc := &mockService{}
	svc.On("GetSessionByKey", mock.Anything, "fake_session_key").
		Return(&fleet.Session{UserID: 42, ID: 1}, nil)
	svc.On("UserUnauthorized", mock.Anything, uint(42)).
		Return(&fleet.User{ID: 42, GlobalRole: ptr.String(fleet.RoleAdmin)}, nil)

	var reqBody *bytes.Reader
	if body != "" {
		reqBody = bytes.NewReader([]byte(body))
	}
	var req *http.Request
	if reqBody != nil {
		req = httptest.NewRequest(method, target, reqBody)
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	req.Header.Add("Authorization", "BEARER fake_session_key")
	return svc, req
}

func TestTraceSamplerHandler_GET(t *testing.T) {
	svc, req := adminAuthedRequest(t, http.MethodGet, "https://fleetdm.com/debug/trace_sampler", "")

	ds := new(mockds.Store)
	ds.GetTraceSamplerSettingsFunc = func(ctx context.Context) (*fleet.TraceSamplerSettings, error) {
		return &fleet.TraceSamplerSettings{
			HighVolumeRatio: 0.001,
			StandardRatio:   0.02,
			ForceFull:       false,
		}, nil
	}

	handler := MakeDebugHandler(svc, testConfig, discardLogger(), nil, ds)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)
	require.True(t, ds.GetTraceSamplerSettingsFuncInvoked)

	var got fleet.TraceSamplerSettings
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &got))
	require.InDelta(t, 0.001, got.HighVolumeRatio, 1e-9)
	require.InDelta(t, 0.02, got.StandardRatio, 1e-9)
	require.False(t, got.ForceFull)
}

func TestTraceSamplerHandler_PATCH_PersistsChanges(t *testing.T) {
	svc, req := adminAuthedRequest(t, http.MethodPatch,
		"https://fleetdm.com/debug/trace_sampler",
		`{"force_full": true}`)

	ds := new(mockds.Store)
	ds.GetTraceSamplerSettingsFunc = func(ctx context.Context) (*fleet.TraceSamplerSettings, error) {
		return &fleet.TraceSamplerSettings{
			HighVolumeRatio: 0.001,
			StandardRatio:   0.02,
			ForceFull:       false,
		}, nil
	}
	var saved *fleet.TraceSamplerSettings
	ds.SetTraceSamplerSettingsFunc = func(ctx context.Context, s *fleet.TraceSamplerSettings) error {
		saved = s
		return nil
	}

	handler := MakeDebugHandler(svc, testConfig, discardLogger(), nil, ds)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code, "PATCH should return 200, body=%s", res.Body.String())
	require.True(t, ds.SetTraceSamplerSettingsFuncInvoked)
	require.NotNil(t, saved)
	require.True(t, saved.ForceFull, "force_full should now be true")
	require.InDelta(t, 0.001, saved.HighVolumeRatio, 1e-9, "other fields should be preserved")
}

func TestTraceSamplerHandler_PATCH_RejectsBadJSON(t *testing.T) {
	svc, req := adminAuthedRequest(t, http.MethodPatch,
		"https://fleetdm.com/debug/trace_sampler",
		`{"force_full":`) // malformed

	ds := new(mockds.Store)
	handler := MakeDebugHandler(svc, testConfig, discardLogger(), nil, ds)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	require.Equal(t, http.StatusBadRequest, res.Code)
	require.False(t, ds.SetTraceSamplerSettingsFuncInvoked)
}

func TestTraceSamplerHandler_PATCH_RejectsOutOfRangeRatio(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"high above 1", `{"high_volume_ratio": 1.5}`},
		{"high below 0", `{"high_volume_ratio": -0.1}`},
		{"standard above 1", `{"standard_ratio": 2.0}`},
		{"standard below 0", `{"standard_ratio": -1.0}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, req := adminAuthedRequest(t, http.MethodPatch,
				"https://fleetdm.com/debug/trace_sampler", c.body)
			ds := new(mockds.Store)
			handler := MakeDebugHandler(svc, testConfig, discardLogger(), nil, ds)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)

			require.Equal(t, http.StatusBadRequest, res.Code)
			require.Contains(t, res.Body.String(), "must be in [0, 1]")
			require.False(t, ds.SetTraceSamplerSettingsFuncInvoked)
		})
	}
}

func TestTraceSamplerHandler_PATCH_RequiresAtLeastOneField(t *testing.T) {
	svc, req := adminAuthedRequest(t, http.MethodPatch,
		"https://fleetdm.com/debug/trace_sampler", `{}`)
	ds := new(mockds.Store)
	handler := MakeDebugHandler(svc, testConfig, discardLogger(), nil, ds)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	require.Equal(t, http.StatusBadRequest, res.Code)
	require.Contains(t, res.Body.String(), "at least one")
}

func TestTraceSamplerHandler_RejectsNonAdmin(t *testing.T) {
	svc := &mockService{}
	svc.On("GetSessionByKey", mock.Anything, "fake_session_key").
		Return(&fleet.Session{UserID: 42, ID: 1}, nil)
	svc.On("UserUnauthorized", mock.Anything, uint(42)).
		Return(&fleet.User{ID: 42, GlobalRole: ptr.String(fleet.RoleObserver)}, nil)

	req := httptest.NewRequest(http.MethodGet, "https://fleetdm.com/debug/trace_sampler", nil)
	req.Header.Add("Authorization", "BEARER fake_session_key")

	ds := new(mockds.Store)
	handler := MakeDebugHandler(svc, testConfig, discardLogger(), nil, ds)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	require.Equal(t, http.StatusForbidden, res.Code)
	require.False(t, ds.GetTraceSamplerSettingsFuncInvoked)
}
