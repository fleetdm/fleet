package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	shared_mdm "github.com/fleetdm/fleet/v4/pkg/mdm"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeFrontend(t *testing.T) {
	if !hasBuildTag("full") {
		t.Skip("This test requires running with -tags full")
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	h := ServeFrontend("", false, logger, false)
	ts := httptest.NewServer(h)
	t.Cleanup(func() {
		ts.Close()
	})

	// Simulate a misconfigured osquery sending log requests to the root endpoint.
	requestBody := []byte(`
	{"data":[{"snapshot":[{"build_distro":"10.14","build_platform":"darwin","config_hash":"d8d220440ebea888f8704c4a0a5c1ced4ab601b5",
	"config_valid":"1","extensions":"active","instance_id":"522e6020-37de-460b-bb01-b76c77298f75","pid":"57456","platform_mask":"21",
	"start_time":"1707768989","uuid":"408F3B27-434F-4776-8538-DA394A3D545F","version":"5.11.0","watcher":"57455"}],"action":"snapshot",
	"name":"packFOOBARGlobalFOOBARQuery_50","hostIdentifier":"589966AE-074A-503B-B17B-54B05684A120","calendarTime":"Mon Feb 12 20:16:40 2024 UTC",
	"unixTime":1707769000,"epoch":0,"counter":0,"numerics":false,"decorations":{"host_uuid":"589966AE-074A-503B-B17B-54B05684A120",
	"hostname":"foobar.local"}},{"snapshot":[{"build_distro":"10.14","build_platform":"darwin",
	"config_hash":"d8d220440ebea888f8704c4a0a5c1ced4ab601b5","config_valid":"1","extensions":"active",
	"instance_id":"522e6020-37de-460b-bb01-b76c77298f75","pid":"57456","platform_mask":"21","start_time":"1707768989",
	"uuid":"408F3B27-434F-4776-8538-DA394A3D545F","version":"5.11.0","watcher":"57455"}],"action":"snapshot",
	"name":"packFOOBARGlobalFOOBARQuery_28","hostIdentifier": "589966AE-074A-503B-B17B-54B05684A120","calendarTime":"Mon Feb 12 20:16:41 2024 UTC",
	"unixTime":1707769001,"epoch":0,"counter":0,"numerics":false,"decorations":{"host_uuid":"408F3B27-434F-4776-8538-DA394A3D545F",
	"hostname":"foobar.local"}}],"log_type":"result","node_key":"J9pA1CmjydHGi0bqS1XkkR9pOJQJzoPA"}`)
	response, err := http.DefaultClient.Post(ts.URL, "", bytes.NewReader(requestBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
}

func TestAssetCacheControl(t *testing.T) {
	for _, tc := range []struct {
		path   string
		status int
		want   string
	}{
		// Content-hashed build output (JS/CSS, fonts, images) is safe to cache
		// forever — webpack emits everything as [name]@[hash][ext].
		{"/bundle-3ccf015bc0fac64b4ce8.js", http.StatusOK, "public, max-age=31536000, immutable"},
		{"/bundle-1e51316ac7963e1112c1.css", http.StatusOK, "public, max-age=31536000, immutable"},
		{"/Inter-Bold@1a2b3c4d5e6f7890.woff2", http.StatusOK, "public, max-age=31536000, immutable"},
		{"/404-dark@1a2b3c4d5e6f7890.svg", http.StatusOK, "public, max-age=31536000, immutable"},
		{"/jira-preview-400x419@2x@1a2b3c4d5e6f7890.png", http.StatusOK, "public, max-age=31536000, immutable"},
		// A 304 keeps the immutable header (the cached copy is still valid).
		{"/bundle-3ccf015bc0fac64b4ce8.js", http.StatusNotModified, "public, max-age=31536000, immutable"},
		// A missing/errored hashed asset (deploy race) must NOT be cached for a
		// year, or a transient failure would pin a broken asset at the CDN/browser.
		{"/bundle-deadbeefdeadbeef.js", http.StatusNotFound, "no-cache"},
		{"/Inter-Bold@deadbeef12345678.woff2", http.StatusInternalServerError, "no-cache"},
		// Unhashed (dev builds, favicon, static scripts) must keep revalidating.
		{"/bundle.js", http.StatusOK, "no-cache"},
		{"/bundle.css", http.StatusOK, "no-cache"},
		{"/favicon.ico", http.StatusOK, "no-cache"},
		// status 0 = handler writes a body without calling WriteHeader, exercising
		// the implicit-200 path in cacheControlResponseWriter.Write.
		{"/bundle-3ccf015bc0fac64b4ce8.js", 0, "public, max-age=31536000, immutable"},
	} {
		t.Run(fmt.Sprintf("%s_%d", tc.path, tc.status), func(t *testing.T) {
			handler := assetCacheControl(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.status == 0 {
					_, _ = w.Write([]byte("body"))
					return
				}
				w.WriteHeader(tc.status)
			}))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			require.Equal(t, tc.want, rec.Header().Get("Cache-Control"))
		})
	}
}

func TestServeEndUserEnrollOTA(t *testing.T) {
	if !hasBuildTag("full") {
		t.Skip("This test requires running with -tags full")
	}

	ds := new(mock.DataStore)
	ds.HasUsersFunc = func(ctx context.Context) (bool, error) {
		return true, nil
	}
	appCfg := &fleet.AppConfig{
		MDM: fleet.MDM{
			EnabledAndConfigured:        false,
			AndroidEnabledAndConfigured: false,
		},
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return appCfg, nil
	}
	ds.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
		return nil, &notFoundError{}
	}
	ds.TeamIDsWithSetupExperienceIdPEnabledFunc = func(ctx context.Context) ([]uint, error) {
		return nil, nil
	}

	svc, _ := newTestService(t, ds, nil, nil)

	for _, enabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("MDM enabled: %t", enabled), func(t *testing.T) {
			appCfg.MDM.EnabledAndConfigured = enabled
			appCfg.MDM.AndroidEnabledAndConfigured = enabled

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			h := ServeEndUserEnrollOTA(svc, "", ds, logger, false)
			ts := httptest.NewServer(h)
			t.Cleanup(func() {
				ts.Close()
			})

			// assert html is returned
			response, err := http.DefaultClient.Get(ts.URL + "?enroll_secret=foo")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, response.StatusCode)
			require.Equal(t, response.Header.Get("Content-Type"), "text/html; charset=utf-8")
			assert.True(t, ds.AppConfigFuncInvoked)

			// assert it contains the content we expect
			defer response.Body.Close()
			bodyBytes, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			bodyString := string(bodyBytes)
			require.Contains(t, bodyString, "api/v1/fleet/enrollment_profiles/ota?enroll_secret=foo")
			require.Contains(t, bodyString, "/api/v1/fleet/android_enterprise/enrollment_token")
			require.Contains(t, bodyString, fmt.Sprintf(`const ANDROID_MDM_ENABLED = "%t" === "true";`, enabled))
			require.Contains(t, bodyString, fmt.Sprintf(`const MAC_MDM_ENABLED = "%t" == "true";`, enabled))
		})
	}
}

func TestServeEndUserEnrollOTAClearsCookieForFullyManaged(t *testing.T) {
	if !hasBuildTag("full") {
		t.Skip("This test requires running with -tags full")
	}

	ds := new(mock.DataStore)
	ds.HasUsersFunc = func(ctx context.Context) (bool, error) {
		return true, nil
	}
	teamID := uint(1)
	ds.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
		return &fleet.EnrollSecret{Secret: secret, TeamID: &teamID}, nil
	}
	ds.TeamLiteFunc = func(ctx context.Context, id uint) (*fleet.TeamLite, error) {
		return &fleet.TeamLite{
			ID: id,
			Config: fleet.TeamConfigLite{
				MDM: fleet.TeamMDM{
					MacOSSetup: fleet.MacOSSetup{
						EnableEndUserAuthentication: true,
					},
				},
			},
		}, nil
	}
	appCfg := &fleet.AppConfig{
		MDM: fleet.MDM{
			EnabledAndConfigured:        true,
			AndroidEnabledAndConfigured: true,
		},
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return appCfg, nil
	}

	svc, _ := newTestService(t, ds, nil, nil)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	h := ServeEndUserEnrollOTA(svc, "", ds, logger, false)
	ts := httptest.NewServer(h)
	t.Cleanup(func() {
		ts.Close()
	})

	idpUUID := "test-idp-uuid-1234"

	// Simulate a request with a valid BYOD cookie + matching enrollment_reference
	// for a fully-managed Android enrollment.
	req, err := http.NewRequest("GET", ts.URL+"?enroll_secret=foo&fully_managed=true&enrollment_reference="+idpUUID, nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  shared_mdm.BYODIdpCookieName,
		Value: idpUUID,
	})

	client := fleethttp.NewClient(fleethttp.WithFollowRedir(false))
	response, err := client.Do(req)
	require.NoError(t, err)
	defer response.Body.Close()

	require.Equal(t, http.StatusOK, response.StatusCode)

	// Assert that Set-Cookie header is present and clears the BYOD cookie.
	setCookieHeaders := response.Header.Values("Set-Cookie")
	var foundClear bool
	for _, sc := range setCookieHeaders {
		if bytes.Contains([]byte(sc), []byte(shared_mdm.BYODIdpCookieName)) &&
			bytes.Contains([]byte(sc), []byte("Max-Age=0")) {
			foundClear = true
			break
		}
	}
	require.True(t, foundClear, "expected Set-Cookie header to clear %s, got: %v", shared_mdm.BYODIdpCookieName, setCookieHeaders)

	// Assert that the rendered HTML contains the IdP UUID for JS to use.
	bodyBytes, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	bodyString := string(bodyBytes)
	require.Contains(t, bodyString, fmt.Sprintf(`const IDP_UUID = "%s";`, idpUUID))

	// BYOD (non-fully-managed) requests should NOT clear the cookie
	req2, err := http.NewRequest("GET", ts.URL+"?enroll_secret=foo&enrollment_reference="+idpUUID, nil)
	require.NoError(t, err)
	req2.AddCookie(&http.Cookie{
		Name:  shared_mdm.BYODIdpCookieName,
		Value: idpUUID,
	})

	response2, err := client.Do(req2)
	require.NoError(t, err)
	defer response2.Body.Close()

	require.Equal(t, http.StatusOK, response2.StatusCode)

	setCookieHeaders2 := response2.Header.Values("Set-Cookie")
	for _, sc := range setCookieHeaders2 {
		require.False(t,
			bytes.Contains([]byte(sc), []byte(shared_mdm.BYODIdpCookieName)) &&
				bytes.Contains([]byte(sc), []byte("Max-Age=0")),
			"BYOD request should not clear %s, got: %v", shared_mdm.BYODIdpCookieName, setCookieHeaders2,
		)
	}

	// Assert that BYOD rendered HTML has an empty IdP UUID (not passed through template).
	bodyBytes2, err := io.ReadAll(response2.Body)
	require.NoError(t, err)
	require.Contains(t, string(bodyBytes2), `const IDP_UUID = "";`)
}
