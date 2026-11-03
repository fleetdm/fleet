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
	writeDeadline   time.Time
}

func (d *deadlineRecorder) SetReadDeadline(time.Time) error {
	d.readDeadlineSet = true
	return nil
}

func (d *deadlineRecorder) SetWriteDeadline(t time.Time) error {
	d.writeDeadline = t
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
			apiTimeoutOverrideHandler(downstream, cfg, logger, func() time.Duration { return 40 * time.Second }).ServeHTTP(
				rec,
				httptest.NewRequest(tc.method, tc.path, nil),
			)

			assert.True(t, called, "the wrapped API handler must always be invoked")
			assert.Equal(t, tc.wantInstallerSize, seen)
			assert.False(t, rec.readDeadlineSet)
		})
	}
}

func TestAPITimeoutOverrideHandlerEmbeddedMCP(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	for _, tc := range []struct {
		name         string
		period       string
		writeTimeout time.Duration
		want         time.Duration
	}{
		{"never shortens the server's write timeout", "25s", 100 * time.Second, 100 * time.Second},
		{"extends it to cover the live query period", "90s", 40 * time.Second, 120 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FLEET_LIVE_QUERY_REST_PERIOD", tc.period)
			var ctxDeadline time.Time
			downstream := http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
				ctxDeadline, _ = req.Context().Deadline()
			})

			rec := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
			start := time.Now()
			apiTimeoutOverrideHandler(downstream, config.FleetConfig{}, logger, func() time.Duration { return tc.writeTimeout }).ServeHTTP(
				rec,
				httptest.NewRequest(http.MethodPost, "/api/latest/fleet/mcp", nil),
			)

			assert.WithinDuration(t, start.Add(tc.want), rec.writeDeadline, time.Second)
			assert.Equal(t, rec.writeDeadline, ctxDeadline, "the tools' work must stop when the response can no longer be written")
		})
	}

	t.Run("reads the server's write timeout per request", func(t *testing.T) {
		t.Setenv("FLEET_LIVE_QUERY_REST_PERIOD", "25s")
		writeTimeout := 100 * time.Second
		handler := apiTimeoutOverrideHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), config.FleetConfig{}, logger,
			func() time.Duration { return writeTimeout })
		for _, wt := range []time.Duration{100 * time.Second, 200 * time.Second} {
			writeTimeout = wt
			rec := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
			start := time.Now()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/latest/fleet/mcp", nil))
			assert.WithinDuration(t, start.Add(wt), rec.writeDeadline, time.Second)
		}
	})

	t.Run("leaves other methods alone", func(t *testing.T) {
		t.Setenv("FLEET_LIVE_QUERY_REST_PERIOD", "90s")
		var hasDeadline bool
		downstream := http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
			_, hasDeadline = req.Context().Deadline()
		})
		rec := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
		apiTimeoutOverrideHandler(downstream, config.FleetConfig{}, logger, func() time.Duration { return 40 * time.Second }).ServeHTTP(
			rec,
			httptest.NewRequest(http.MethodGet, "/api/latest/fleet/mcp", nil),
		)
		assert.True(t, rec.writeDeadline.IsZero())
		assert.False(t, hasDeadline)
	})
}
