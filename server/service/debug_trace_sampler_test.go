package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	mockds "github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/tracing"
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
		Return(&fleet.User{ID: 42, GlobalRole: new(fleet.RoleAdmin)}, nil)

	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reqBody)
	req.Header.Add("Authorization", "BEARER fake_session_key")
	return svc, req
}

func TestTraceSamplerHandler_GET(t *testing.T) {
	svc, req := adminAuthedRequest(t, http.MethodGet, "https://fleetdm.com/debug/trace_sampler", "")

	ds := new(mockds.Store)
	ds.GetTraceSamplerSettingsFunc = func(_ context.Context) (*tracing.Settings, error) {
		return &tracing.Settings{
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

	var got tracing.Settings
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &got))
	require.InDelta(t, 0.001, got.HighVolumeRatio, 1e-9)
	require.InDelta(t, 0.02, got.StandardRatio, 1e-9)
	require.False(t, got.ForceFull)
}

func TestTraceSamplerHandler_PATCH_PersistsChangesAndReturnsRow(t *testing.T) {
	svc, req := adminAuthedRequest(t, http.MethodPatch,
		"https://fleetdm.com/debug/trace_sampler",
		`{"force_full": true}`)

	ds := new(mockds.Store)
	ds.GetTraceSamplerSettingsFunc = func(_ context.Context) (*tracing.Settings, error) {
		return &tracing.Settings{
			HighVolumeRatio: 0.001,
			StandardRatio:   0.02,
			ForceFull:       false,
		}, nil
	}
	var saved *tracing.Settings
	ds.SetTraceSamplerSettingsFunc = func(_ context.Context, s *tracing.Settings) error {
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

	// Verify the response body matches what was saved. If we forgot to write the response, the test would still see 200 from
	// httptest's default but the body would be empty.
	var returned tracing.Settings
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &returned))
	require.True(t, returned.ForceFull)
	require.InDelta(t, saved.HighVolumeRatio, returned.HighVolumeRatio, 1e-9)
	require.InDelta(t, saved.StandardRatio, returned.StandardRatio, 1e-9)

	// PATCH response must NOT include updated_at. The handler reads the row before the write, so the pre-write timestamp
	// would be stale. Operators do a follow-up GET to see the post-write value.
	require.NotContains(t, res.Body.String(), "updated_at",
		"PATCH response must drop updated_at to avoid returning a stale timestamp")
}

func TestTraceSamplerHandler_PATCH_PartialUpdatePreservesOtherFields(t *testing.T) {
	// Locks in the docstring claim that "PATCH semantics mean only the provided fields are applied." Sending only
	// high_volume_ratio must leave standard_ratio and force_full at their prior values.
	svc, req := adminAuthedRequest(t, http.MethodPatch,
		"https://fleetdm.com/debug/trace_sampler",
		`{"high_volume_ratio": 0.5}`)

	ds := new(mockds.Store)
	ds.GetTraceSamplerSettingsFunc = func(_ context.Context) (*tracing.Settings, error) {
		return &tracing.Settings{
			HighVolumeRatio: 0.001,
			StandardRatio:   0.07,
			ForceFull:       true,
		}, nil
	}
	var saved *tracing.Settings
	ds.SetTraceSamplerSettingsFunc = func(_ context.Context, s *tracing.Settings) error {
		saved = s
		return nil
	}

	handler := MakeDebugHandler(svc, testConfig, discardLogger(), nil, ds)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code, "body=%s", res.Body.String())
	require.NotNil(t, saved)
	require.InDelta(t, 0.5, saved.HighVolumeRatio, 1e-9, "high_volume_ratio should be applied")
	require.InDelta(t, 0.07, saved.StandardRatio, 1e-9, "standard_ratio should be preserved from the prior row")
	require.True(t, saved.ForceFull, "force_full should be preserved from the prior row")
}

func TestTraceSamplerHandler_PATCH_ReadFailureReturns500(t *testing.T) {
	svc, req := adminAuthedRequest(t, http.MethodPatch,
		"https://fleetdm.com/debug/trace_sampler",
		`{"force_full": true}`)

	ds := new(mockds.Store)
	ds.GetTraceSamplerSettingsFunc = func(_ context.Context) (*tracing.Settings, error) {
		return nil, errors.New("db unavailable")
	}

	handler := MakeDebugHandler(svc, testConfig, discardLogger(), nil, ds)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	require.Equal(t, http.StatusInternalServerError, res.Code)
	require.False(t, ds.SetTraceSamplerSettingsFuncInvoked, "should not attempt to write when read fails")
}

func TestTraceSamplerHandler_PATCH_WriteFailureReturns500(t *testing.T) {
	svc, req := adminAuthedRequest(t, http.MethodPatch,
		"https://fleetdm.com/debug/trace_sampler",
		`{"force_full": true}`)

	ds := new(mockds.Store)
	ds.GetTraceSamplerSettingsFunc = func(_ context.Context) (*tracing.Settings, error) {
		return &tracing.Settings{HighVolumeRatio: 0.001, StandardRatio: 0.02}, nil
	}
	ds.SetTraceSamplerSettingsFunc = func(_ context.Context, _ *tracing.Settings) error {
		return errors.New("constraint violation")
	}

	handler := MakeDebugHandler(svc, testConfig, discardLogger(), nil, ds)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	require.Equal(t, http.StatusInternalServerError, res.Code)
	require.True(t, ds.SetTraceSamplerSettingsFuncInvoked)
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
