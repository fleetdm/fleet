package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Audit note: the schema deliberately has no updated_by_user_id column. The
// /debug auth middleware logs the authenticated user, the PATCH access log
// records the request, and the slog InfoContext below records the user ID
// from viewer context — those three together cover the audit trail without
// a stale-on-user-deletion column on the table.

// traceSamplerPatchRequest is the PATCH payload. Fields are pointers so we
// can distinguish "unset" from a zero value — PATCH semantics mean only the
// provided fields are applied.
type traceSamplerPatchRequest struct {
	HighVolumeRatio *float64 `json:"high_volume_ratio,omitempty"`
	StandardRatio   *float64 `json:"standard_ratio,omitempty"`
	ForceFull       *bool    `json:"force_full,omitempty"`
}

// traceSamplerHandler serves GET and PATCH for /debug/trace_sampler. Wired in
// MakeDebugHandler behind debugAuthenticationMiddleware (admin-only). GET
// returns the current settings; PATCH validates ratios in [0,1] and persists
// the change. The replica's in-memory sampler picks up the change on the
// next poller tick (default 60s).
func traceSamplerHandler(logger *slog.Logger, ds fleet.Datastore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleTraceSamplerGet(w, r, logger, ds)
		case http.MethodPatch:
			handleTraceSamplerPatch(w, r, logger, ds)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleTraceSamplerGet(w http.ResponseWriter, r *http.Request, logger *slog.Logger, ds fleet.Datastore) {
	settings, err := ds.GetTraceSamplerSettings(r.Context())
	if err != nil {
		logger.ErrorContext(r.Context(), "debug trace_sampler GET failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, settings)
}

func handleTraceSamplerPatch(w http.ResponseWriter, r *http.Request, logger *slog.Logger, ds fleet.Datastore) {
	var req traceSamplerPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON body: %v", err), http.StatusBadRequest)
		return
	}

	if err := validateTraceSamplerPatch(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	current, err := ds.GetTraceSamplerSettings(r.Context())
	if err != nil {
		logger.ErrorContext(r.Context(), "debug trace_sampler PATCH read-modify failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if req.HighVolumeRatio != nil {
		current.HighVolumeRatio = *req.HighVolumeRatio
	}
	if req.StandardRatio != nil {
		current.StandardRatio = *req.StandardRatio
	}
	if req.ForceFull != nil {
		current.ForceFull = *req.ForceFull
	}

	if err := ds.SetTraceSamplerSettings(r.Context(), current); err != nil {
		logger.ErrorContext(r.Context(), "debug trace_sampler PATCH write failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	updatedBy := uint(0)
	if v, ok := viewer.FromContext(r.Context()); ok {
		updatedBy = v.UserID()
	}
	logger.InfoContext(r.Context(), "trace sampler settings updated",
		"high_volume_ratio", current.HighVolumeRatio,
		"standard_ratio", current.StandardRatio,
		"force_full", current.ForceFull,
		"updated_by_user_id", updatedBy,
	)

	// Return the updated row so callers can confirm what was applied.
	writeJSON(w, current)
}

func validateTraceSamplerPatch(req traceSamplerPatchRequest) error {
	if req.HighVolumeRatio == nil && req.StandardRatio == nil && req.ForceFull == nil {
		return errors.New("request body must include at least one of high_volume_ratio, standard_ratio, force_full")
	}
	if req.HighVolumeRatio != nil && (*req.HighVolumeRatio < 0 || *req.HighVolumeRatio > 1) {
		return fmt.Errorf("high_volume_ratio must be in [0, 1], got %v", *req.HighVolumeRatio)
	}
	if req.StandardRatio != nil && (*req.StandardRatio < 0 || *req.StandardRatio > 1) {
		return fmt.Errorf("standard_ratio must be in [0, 1], got %v", *req.StandardRatio)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		http.Error(w, "encoding response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(b)
}
