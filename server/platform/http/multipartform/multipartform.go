// Package multipartform provides helpers for parsing multipart/form-data
// request bodies. It lives in its own subpackage so it can depend on
// server/contexts/logging without causing an import cycle with
// server/platform/http.
package multipartform

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	platform_logging "github.com/fleetdm/fleet/v4/server/platform/logging"
)

// Parse parses the multipart form on the request, then migrates the legacy
// "team_id" field to "fleet_id" (emitting a deprecation warning).
func Parse(ctx context.Context, r *http.Request, maxMemory int64) error {
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		return err
	}
	// Check if a "team_id" field is present and valid. If so, log a deprecation warning, add a "fleet_id" field with the same value, and remove the "team_id" field to prevent confusion in handlers.
	teamIDs, teamIDPresent := r.Form["team_id"]
	if teamIDPresent && len(teamIDs) > 0 {
		teamID := teamIDs[0]
		if platform_logging.TopicEnabled(platform_logging.DeprecatedFieldTopic) {
			logging.WithExtras(ctx,
				"deprecated_param", "team_id",
				"deprecation_warning", "'team_id' is deprecated, use 'fleet_id' instead",
			)
			logging.WithLevel(ctx, slog.LevelWarn)
		}
		r.Form.Set("fleet_id", teamID)
		r.Form.Del("team_id")
		r.MultipartForm.Value["fleet_id"] = []string{teamID}
		delete(r.MultipartForm.Value, "team_id")
	}
	return nil
}
