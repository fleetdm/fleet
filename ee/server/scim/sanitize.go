package scim

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/elimity-com/scim"
	scimerrors "github.com/elimity-com/scim/errors"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
)

// sanitizedResourceHandler keeps datastore detail out of SCIM responses.
// errors.CheckScimError turns anything that is not already a ScimError into a 500
// whose detail is err.Error().
//
// scim.ResourceHandler has six methods; a new one must be added here too.
type sanitizedResourceHandler struct {
	scim.ResourceHandler
	logger *slog.Logger
}

func newSanitizedResourceHandler(h scim.ResourceHandler, logger *slog.Logger) scim.ResourceHandler {
	return sanitizedResourceHandler{ResourceHandler: h, logger: logger}
}

func (h sanitizedResourceHandler) sanitize(ctx context.Context, op string, err error) error {
	if err == nil {
		return nil
	}
	// A ScimError is already the library's own message to the caller.
	if _, ok := err.(scimerrors.ScimError); ok { //nolint:errorlint // CheckScimError type-asserts the same way
		return err
	}
	// A SCIM error has no uuid field, so the correlation id goes in the detail:
	// without it the caller has nothing to quote to find this log line.
	detail := platform_http.GenericErrorMessage
	attrs := []any{"op", op, "err", err}
	if logCtx, ok := logging.FromContext(ctx); ok && logCtx.RequestID != "" {
		attrs = append(attrs, "uuid", logCtx.RequestID)
		detail = fmt.Sprintf("%s (%s)", detail, logCtx.RequestID)
	}
	h.logger.ErrorContext(ctx, "scim handler error", attrs...)

	return scimerrors.ScimError{
		Status: http.StatusInternalServerError,
		Detail: detail,
	}
}

func (h sanitizedResourceHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	res, err := h.ResourceHandler.Create(r, attributes)
	return res, h.sanitize(r.Context(), "create", err)
}

func (h sanitizedResourceHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	res, err := h.ResourceHandler.Get(r, id)
	return res, h.sanitize(r.Context(), "get", err)
}

func (h sanitizedResourceHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
	page, err := h.ResourceHandler.GetAll(r, params)
	return page, h.sanitize(r.Context(), "get_all", err)
}

func (h sanitizedResourceHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	res, err := h.ResourceHandler.Replace(r, id, attributes)
	return res, h.sanitize(r.Context(), "replace", err)
}

func (h sanitizedResourceHandler) Delete(r *http.Request, id string) error {
	return h.sanitize(r.Context(), "delete", h.ResourceHandler.Delete(r, id))
}

func (h sanitizedResourceHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	res, err := h.ResourceHandler.Patch(r, id, operations)
	return res, h.sanitize(r.Context(), "patch", err)
}
