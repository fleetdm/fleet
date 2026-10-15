package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/installersize"
	"github.com/stretchr/testify/assert"
)

type deadlineRecorder struct {
	*httptest.ResponseRecorder
	readDeadlineSet bool
}

func (d *deadlineRecorder) SetReadDeadline(time.Time) error {
	d.readDeadlineSet = true
	return nil
}

func TestAPITimeoutOverrideHandler(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	const customMax int64 = 4242

	cfg := config.FleetConfig{}
	cfg.Server.MaxInstallerSizeBytes = customMax

	for _, tc := range []struct {
		name              string
		method            string
		path              string
		wantInstallerSize int64
	}{
		{
			name:              "software package upload threads configured max size",
			method:            http.MethodPost,
			path:              "/api/latest/fleet/software/package",
			wantInstallerSize: customMax,
		},
		{
			name:              "bootstrap package upload threads configured max size",
			method:            http.MethodPost,
			path:              "/api/latest/fleet/mdm/bootstrap",
			wantInstallerSize: customMax,
		},
		{
			name:              "app store apps batch keeps the default read deadline",
			method:            http.MethodPost,
			path:              "/api/latest/fleet/software/app_store_apps/batch",
			wantInstallerSize: installersize.MaxSoftwareInstallerSize,
		},
		{
			name:              "non-upload request leaves the default max size",
			method:            http.MethodGet,
			path:              "/api/latest/fleet/hosts",
			wantInstallerSize: installersize.MaxSoftwareInstallerSize,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var (
				called bool
				seen   int64
			)
			downstream := http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
				called = true
				seen = installersize.FromContext(req.Context())
			})

			rec := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
			apiTimeoutOverrideHandler(downstream, cfg, logger).ServeHTTP(
				rec,
				httptest.NewRequest(tc.method, tc.path, nil),
			)

			assert.True(t, called, "the wrapped API handler must always be invoked")
			assert.Equal(t, tc.wantInstallerSize, seen)
			assert.False(t, rec.readDeadlineSet)
		})
	}
}

func TestAPITimeoutOverrideHandlerExtendsWriteDeadline(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	slow := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewUnstartedServer(apiTimeoutOverrideHandler(slow, config.FleetConfig{}, logger))
	srv.EnableHTTP2 = true
	srv.Config.WriteTimeout = 100 * time.Millisecond
	srv.StartTLS()
	t.Cleanup(srv.Close)

	for _, tc := range []struct {
		path   string
		wantOK bool
	}{
		{"/api/latest/fleet/software/app_store_apps/batch", true},
		{"/api/latest/fleet/software/batch", false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			resp, err := srv.Client().Post(srv.URL+tc.path, "application/json", nil)
			if !tc.wantOK {
				assert.ErrorContains(t, err, "INTERNAL_ERROR", "the default write timeout must still apply")
				return
			}
			if assert.NoError(t, err) {
				defer resp.Body.Close()
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Equal(t, 2, resp.ProtoMajor)
			}
		})
	}
}
