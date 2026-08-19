package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// traceSamplerPatchRequest is the PATCH payload. Fields are pointers so we can distinguish "unset" from a zero value. PATCH
// semantics mean only the provided fields are applied.
type traceSamplerPatchRequest struct {
	HighVolumeRatio *float64 `json:"high_volume_ratio,omitempty"`
	StandardRatio   *float64 `json:"standard_ratio,omitempty"`
	ForceFull       *bool    `json:"force_full,omitempty"`
}

// patchTraceSamplerHandler returns the PATCH /debug/trace_sampler handler. The GET path is wired separately in
// MakeDebugHandler via the existing jsonHandler helper, matching the convention used by /debug/migrations and /debug/db/*.
//
// PATCH is necessarily bespoke because no other /debug/ endpoint takes a request body. It validates ratios in [0, 1] and
// persists the change. The replica's in memory sampler picks up the change on the next poller tick (default 60s).
func patchTraceSamplerHandler(logger *slog.Logger, ds fleet.Datastore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// The /debug auth middleware always installs the viewer in context. If it is missing here, the middleware was bypassed
		// and we should refuse to record a change rather than silently log user_id=0. That value is indistinguishable from a
		// real user id of 0 and weakens the audit trail.
		v, ok := viewer.FromContext(r.Context())
		if !ok {
			handleServerError(w, r, logger, "debug trace_sampler PATCH refused: viewer missing from context", "viewer required",
				errors.New("viewer missing from context"))
			return
		}

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
			handleServerError(w, r, logger, "debug trace_sampler PATCH read-modify failed", "internal error", err)
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
			handleServerError(w, r, logger, "debug trace_sampler PATCH write failed", "internal error", err)
			return
		}

		logger.InfoContext(r.Context(), "trace sampler settings updated",
			"high_volume_ratio", current.HighVolumeRatio,
			"standard_ratio", current.StandardRatio,
			"force_full", current.ForceFull,
			"updated_by_user_id", v.UserID(),
		)

		// Return the updated row so callers can confirm what was applied. Drop UpdatedAt: the row was read before the write,
		// so current.UpdatedAt is the pre-write timestamp (stale and confusing). omitzero on the struct tag skips the field
		// when zero. Operators who want the post-write timestamp can do a follow-up GET.
		current.UpdatedAt = time.Time{}
		b, err := json.MarshalIndent(current, "", "  ")
		if err != nil {
			handleServerError(w, r, logger, "debug trace_sampler PATCH encode response failed", "encoding response", err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(b)
	}
}

// handleServerError centralizes the internal-error response used throughout the trace sampler PATCH handler: it logs logMsg
// with err, records err on the context for the error-handling middleware, and writes clientMsg to the client as a 500.
func handleServerError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, logMsg, clientMsg string, err error) {
	logger.ErrorContext(r.Context(), logMsg, "err", err)
	ctxerr.Handle(r.Context(), err)
	http.Error(w, clientMsg, http.StatusInternalServerError)
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
