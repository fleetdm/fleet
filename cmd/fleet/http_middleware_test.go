package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/installersize"
	"github.com/stretchr/testify/assert"
)

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

			apiTimeoutOverrideHandler(downstream, cfg, logger).ServeHTTP(
				httptest.NewRecorder(),
				httptest.NewRequest(tc.method, tc.path, nil),
			)

			assert.True(t, called, "the wrapped API handler must always be invoked")
			assert.Equal(t, tc.wantInstallerSize, seen)
		})
	}
}
