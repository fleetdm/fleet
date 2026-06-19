package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/installersize"
)

// apiTimeoutOverrideHandler wraps the main API handler with per-route request
// read/write deadline overrides for endpoints that legitimately run long:
// synchronous script runs, large software-installer and bootstrap-package
// uploads, the Android enterprise signup SSE stream, and large MDM profile
// batch operations. For package-upload routes it also caps the request body and
// threads the configured max installer size through the request context.
//
// Deadline overrides are best-effort: if the ResponseWriter does not support
// SetReadDeadline/SetWriteDeadline the error is logged and the request proceeds.
func apiTimeoutOverrideHandler(apiHandler http.Handler, cfg config.FleetConfig, logger *slog.Logger) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/fleet/scripts/run/sync") {
			// when running a script synchronously, we wait a while for a script
			// execution result, so the write timeout (to write the response)
			// must be extended.
			rc := http.NewResponseController(rw)
			// add an additional 30 seconds to prevent race conditions where the
			// request is terminated early.
			if err := rc.SetWriteDeadline(time.Now().Add(scripts.MaxServerWaitTime + (30 * time.Second))); err != nil {
				logger.ErrorContext(req.Context(),
					"http middleware failed to override endpoint write timeout for script sync run",
					"response_writer_type", fmt.Sprintf("%T", rw),
					"response_writer", fmt.Sprintf("%+v", rw),
					"err", err,
				)
			}
		}

		if (req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/fleet/software/package")) ||
			(req.Method == http.MethodPatch && strings.HasSuffix(req.URL.Path, "/package") && strings.Contains(req.URL.Path,
				"/fleet/software/titles/")) ||
			(req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/bootstrap")) ||
			(req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/fleet_maintained_apps")) ||
			(req.Method == http.MethodGet && strings.Contains(req.URL.Path, "/package/token")) ||
			(req.Method == http.MethodPost && strings.Contains(req.URL.Path, "orbit/software_install/package")) {
			var zeroTime time.Time
			rc := http.NewResponseController(rw)
			// For large software installers and bootstrap packages, the server time needs time to read the full
			// request body so we use the zero value to remove the deadline and override the
			// default read timeout.
			// TODO: Is this really how we want to handle this? Or would an arbitrarily long
			// timeout be better?
			if err := rc.SetReadDeadline(zeroTime); err != nil {
				logger.ErrorContext(req.Context(),
					"http middleware failed to override endpoint read timeout for software package upload",
					"response_writer_type", fmt.Sprintf("%T", rw),
					"response_writer", fmt.Sprintf("%+v", rw),
					"err", err,
				)
			}
			// For large software installers, the server time needs time to store the
			// installer to S3 (or the configured storage location) and write the response
			// body so we use the zero value to remove the deadline and override the
			// default write timeout.
			// TODO: Is this really how we want to handle this? Or would an arbitrarily long
			// timeout be better?
			if err := rc.SetWriteDeadline(zeroTime); err != nil {
				logger.ErrorContext(req.Context(),
					"http middleware failed to override endpoint write timeout for software package upload",
					"response_writer_type", fmt.Sprintf("%T", rw),
					"response_writer", fmt.Sprintf("%+v", rw),
					"err", err,
				)
			}

			// We need to add the context value here because we need the installer max size when doing request
			// parsing, which happens somewhere where we're only passed the request (and not the service object)
			req.Body = http.MaxBytesReader(rw, req.Body, cfg.Server.MaxInstallerSizeBytes)
			req = req.WithContext(installersize.NewContext(req.Context(), cfg.Server.MaxInstallerSizeBytes))
		}

		if req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/fleet/android_enterprise/signup_sse") {
			// When enabling Android MDM, frontend UI will wait for the admin to finish the setup in Google.
			rc := http.NewResponseController(rw)
			if err := rc.SetWriteDeadline(time.Now().Add(30 * time.Minute)); err != nil {
				logger.ErrorContext(req.Context(),
					"http middleware failed to override endpoint write timeout for android enterpriset setup",
					"response_writer_type", fmt.Sprintf("%T", rw),
					"response_writer", fmt.Sprintf("%+v", rw),
					"err", err,
				)
			}
		}

		if req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/fleet/mdm/profiles/batch") ||
			(req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/fleet/configuration_profiles/batch")) {
			// For customers using large profiles and/or large numbers of profiles, the
			// server needs time to completely read the request body and also to process
			// all the side effects of a potentially large number of profiles being changed
			// across a large number of hosts, so set the timeouts a bit higher than default
			rc := http.NewResponseController(rw)
			if err := rc.SetWriteDeadline(time.Now().Add(5 * time.Minute)); err != nil {
				logger.ErrorContext(req.Context(),
					"http middleware failed to override endpoint write timeout for MDM profiles batch endpoint",
					"response_writer_type", fmt.Sprintf("%T", rw),
					"response_writer", fmt.Sprintf("%+v", rw),
					"err", err,
				)
			}
			if err := rc.SetReadDeadline(time.Now().Add(5 * time.Minute)); err != nil {
				logger.ErrorContext(req.Context(),
					"http middleware failed to override endpoint read timeout for MDM profiles batch endpoint",
					"response_writer_type", fmt.Sprintf("%T", rw),
					"response_writer", fmt.Sprintf("%+v", rw),
					"err", err,
				)
			}
		}

		apiHandler.ServeHTTP(rw, req)
	}
}
