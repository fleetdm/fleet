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
