package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/gorilla/mux"
)

const orgLogoMaxFileSize = fleet.OrgLogoMaxFileSize

// PUT /api/v1/fleet/logo

type putOrgLogoRequest struct {
	Mode fleet.OrgLogoMode
	Body []byte
}

type putOrgLogoResponse struct {
	Err error `json:"error,omitempty"`
}

func (r putOrgLogoResponse) Error() error { return r.Err }

func (putOrgLogoRequest) DecodeRequest(_ context.Context, r *http.Request) (any, error) {
	mode, err := parseLogoModeQuery(r.URL.Query().Get("mode"))
	if err != nil {
		return nil, err
	}
	if err := r.ParseMultipartForm(platform_http.MaxMultipartFormSize); err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}
	files := r.MultipartForm.File["logo"]
	if len(files) == 0 {
		return nil, &fleet.BadRequestError{Message: "missing 'logo' file in multipart form"}
	}
	f, err := files[0].Open()
	if err != nil {
		return nil, &fleet.BadRequestError{Message: "failed to open uploaded logo", InternalErr: err}
	}
	defer f.Close()
	body, err := io.ReadAll(io.LimitReader(f, orgLogoMaxFileSize+1))
	if err != nil {
		return nil, &fleet.BadRequestError{Message: "failed to read uploaded logo", InternalErr: err}
	}
	return putOrgLogoRequest{Mode: mode, Body: body}, nil
}

func putOrgLogoEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(putOrgLogoRequest)
	if err := svc.UploadOrgLogo(ctx, req.Mode, bytes.NewReader(req.Body)); err != nil {
		return putOrgLogoResponse{Err: err}, nil
	}
	return putOrgLogoResponse{}, nil
}

// DELETE /api/v1/fleet/logo

type deleteOrgLogoRequest struct {
	Mode fleet.OrgLogoMode
}

type deleteOrgLogoResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteOrgLogoResponse) Error() error { return r.Err }

func (deleteOrgLogoRequest) DecodeRequest(_ context.Context, r *http.Request) (any, error) {
	mode, err := parseLogoModeQuery(r.URL.Query().Get("mode"))
	if err != nil {
		return nil, err
	}
	return deleteOrgLogoRequest{Mode: mode}, nil
}

func deleteOrgLogoEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(deleteOrgLogoRequest)
	if err := svc.DeleteOrgLogo(ctx, req.Mode); err != nil {
		return deleteOrgLogoResponse{Err: err}, nil
	}
	return deleteOrgLogoResponse{}, nil
}

// GET /api/latest/fleet/logo

type getOrgLogoRequest struct {
	Mode fleet.OrgLogoMode
}

type getOrgLogoResponse struct {
	Err  error
	Body []byte
}

func (r getOrgLogoResponse) Error() error { return r.Err }

func (r getOrgLogoResponse) HijackRender(_ context.Context, w http.ResponseWriter) {
	if r.Err != nil {
		return
	}
	contentType := fleet.ContentTypeForOrgLogo(r.Body)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(r.Body)))
	w.Header().Set("Cache-Control", "no-store")
	// nosniff: stops the browser from MIME-sniffing the body as HTML
	// (XSS vector) if upstream Content-Type ever drifts.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if contentType == "image/svg+xml" {
		// CSP keeps the direct-URL view inert (where SVG loads as a
		// document, not <img>). 'unsafe-inline' allows the inline
		// <style> blocks most SVGs include.
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	}
	_, _ = w.Write(r.Body)
}

func (getOrgLogoRequest) DecodeRequest(_ context.Context, r *http.Request) (any, error) {
	raw := mux.Vars(r)["mode"]
	if raw == "" {
		raw = r.URL.Query().Get("mode")
	}
	if raw == "" {
		return nil, &fleet.BadRequestError{Message: "mode query parameter is required (light or dark)"}
	}
	m := fleet.OrgLogoMode(raw)
	// GET serves a single stored file. OrgLogoModeAll is rejected even
	// though it's a recognized request mode for PUT/DELETE.
	if !m.IsStorable() {
		return nil, &fleet.BadRequestError{
			Message: fmt.Sprintf("invalid mode %q: must be 'light' or 'dark'", raw),
		}
	}
	return getOrgLogoRequest{Mode: m}, nil
}

func getOrgLogoEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(getOrgLogoRequest)
	body, _, err := svc.GetOrgLogo(ctx, req.Mode)
	if err != nil {
		return getOrgLogoResponse{Err: err}, nil
	}
	return getOrgLogoResponse{Body: body}, nil
}

// parseLogoModeQuery interprets the `mode` query string for the PUT and
// DELETE endpoints. Empty defaults to OrgLogoModeAll. All three values
// (light / dark / all) are valid here; the GET endpoint has its own
// stricter parser since "all" doesn't make sense when serving bytes.
func parseLogoModeQuery(raw string) (fleet.OrgLogoMode, error) {
	if raw == "" {
		return fleet.OrgLogoModeAll, nil
	}
	m := fleet.OrgLogoMode(raw)
	if !m.IsValid() {
		return "", &fleet.BadRequestError{
			Message: fmt.Sprintf("invalid mode %q: must be 'light', 'dark', or 'all'", raw),
		}
	}
	return m, nil
}

// Service implementation

func (svc *Service) UploadOrgLogo(ctx context.Context, mode fleet.OrgLogoMode, content io.ReadSeeker) error {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionWrite); err != nil {
		return err
	}
	if !mode.IsValid() {
		return &fleet.BadRequestError{Message: fmt.Sprintf("invalid mode %q", mode)}
	}
	if svc.orgLogoStore == nil {
		return ctxerr.New(ctx, "org logo store not configured")
	}

	// Buffer once so each Put gets its own reader without re-Seeking the
	// underlying source — and so a mid-loop failure can roll back.
	body, err := io.ReadAll(io.LimitReader(content, orgLogoMaxFileSize+1))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "buffering logo content")
	}
	if err := fleet.ValidateOrgLogoBytes(body); err != nil {
		return err
	}

	modes := mode.Modes()
	var stored []fleet.OrgLogoMode
	for _, m := range modes {
		if err := svc.orgLogoStore.Put(ctx, m, bytes.NewReader(body)); err != nil {
			// Best-effort rollback so the store doesn't end up with a
			// half-written set (e.g. light stored, dark failed). We
			// don't fail the request on rollback errors — the original
			// Put error is what the caller cares about.
			for _, sm := range stored {
				_ = svc.orgLogoStore.Delete(ctx, sm)
			}
			return ctxerr.Wrapf(ctx, err, "storing org logo (%s)", m)
		}
		stored = append(stored, m)
	}
	if err := svc.updateOrgLogoURLs(ctx, modes, true); err != nil {
		return err
	}

	if vc, ok := viewer.FromContext(ctx); ok {
		if err := svc.NewActivity(ctx, vc.User, fleet.ActivityTypeChangedOrgLogo{
			Mode: string(mode),
		}); err != nil {
			return ctxerr.Wrap(ctx, err, "create changed_org_logo activity")
		}
	}
	return nil
}

func (svc *Service) DeleteOrgLogo(ctx context.Context, mode fleet.OrgLogoMode) error {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionWrite); err != nil {
		return err
	}
	if !mode.IsValid() {
		return &fleet.BadRequestError{Message: fmt.Sprintf("invalid mode %q", mode)}
	}
	if svc.orgLogoStore == nil {
		return ctxerr.New(ctx, "org logo store not configured")
	}

	// Read AppConfig up front so we can distinguish Fleet-hosted URLs (which
	// also have a blob in the store) from external URLs (which don't). For
	// external URLs there's nothing to delete from the store — we just clear
	// the URL field. Bypass the cache so we see any concurrent write.
	ctx = ctxdb.BypassCachedMysql(ctx, true)
	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "loading app config")
	}

	// toDelete: modes whose blob lives in the store and must be removed.
	// toClear: modes whose URL field is non-empty and must be cleared.
	// The two sets can diverge:
	//   - External URL → toClear only (no blob backs an external URL).
	//   - Empty URL + lingering blob → toDelete only. Happens in the
	//     GitOps flow, which PATCHes the URL to "" before calling
	//     DeleteOrgLogo to drop the blob (see doGitOpsOrgLogos).
	//   - Fleet-hosted URL + blob → both.
	modes := mode.Modes()
	var toDelete, toClear []fleet.OrgLogoMode
	for _, m := range modes {
		currentURL := orgLogoURLForMode(&ac.OrgInfo, m)
		if currentURL != "" {
			toClear = append(toClear, m)
		}
		// External URLs never have a backing blob — skip the Exists check.
		// For empty or Fleet-hosted URLs the blob may exist.
		if currentURL != "" && !fleet.IsFleetHostedLogoURL(currentURL) {
			continue
		}
		// The S3 store's Delete is silent on missing keys, so check Exists
		// first — otherwise we'd persist a misleading "logo deleted" activity
		// when there was nothing to delete.
		exists, err := svc.orgLogoStore.Exists(ctx, m)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "checking org logo exists (%s)", m)
		}
		if exists {
			toDelete = append(toDelete, m)
		}
	}
	if len(toClear) == 0 && len(toDelete) == 0 {
		return &fleet.BadRequestError{Message: "no org logo to delete for the given mode"}
	}

	// Try every requested mode even if one fails so a partial in-store
	// state isn't left where one blob is gone but another lingers under a
	// URL we already cleared. URLs are only cleared if all deletes
	// succeeded — on a partial failure the caller retries and Delete is
	// idempotent.
	var errs []error
	for _, m := range toDelete {
		if err := svc.orgLogoStore.Delete(ctx, m); err != nil {
			errs = append(errs, ctxerr.Wrapf(ctx, err, "deleting org logo (%s)", m))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	// Skip the SaveAppConfig round-trip when there are no URLs to clear
	// (e.g. the GitOps flow already PATCHed them to "" before calling us).
	if len(toClear) > 0 {
		if err := svc.updateOrgLogoURLs(ctx, toClear, false); err != nil {
			return err
		}
	}

	if vc, ok := viewer.FromContext(ctx); ok {
		if err := svc.NewActivity(ctx, vc.User, fleet.ActivityTypeDeletedOrgLogo{
			Mode: string(mode),
		}); err != nil {
			return ctxerr.Wrap(ctx, err, "create deleted_org_logo activity")
		}
	}
	return nil
}

func (svc *Service) GetOrgLogo(ctx context.Context, mode fleet.OrgLogoMode) ([]byte, int64, error) {
	svc.authz.SkipAuthorization(ctx)
	if svc.orgLogoStore == nil {
		return nil, 0, ctxerr.New(ctx, "org logo store not configured")
	}
	if !mode.IsStorable() {
		return nil, 0, &fleet.BadRequestError{Message: fmt.Sprintf("invalid mode %q: must be 'light' or 'dark'", mode)}
	}
	// Discard the metadata size: we cap reads at orgLogoMaxFileSize and
	// derive the response size from the actual bytes read so an oversized
	// or mid-flight-changed object can't desync Content-Length from body.
	r, _, err := svc.orgLogoStore.Get(ctx, mode)
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "fetching org logo")
	}
	defer r.Close()
	// Read one byte over the limit so we can detect (and reject) anything
	// larger than the upload validator should ever have allowed in.
	body, err := io.ReadAll(io.LimitReader(r, orgLogoMaxFileSize+1))
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "reading org logo bytes")
	}
	if int64(len(body)) > orgLogoMaxFileSize {
		return nil, 0, ctxerr.New(ctx, "stored org logo exceeds max size")
	}
	// Re-validate on read so a blob planted directly in the object store
	// (bypassing the upload API) is still rejected.
	if err := fleet.ValidateOrgLogoBytes(body); err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "stored org logo failed validation")
	}
	return body, int64(len(body)), nil
}

// orgLogoServingURL builds the URL persisted in AppConfig after an upload. The `v` param is a cache-buster (ignored server-side; only `mode` is read).
func orgLogoServingURL(mode fleet.OrgLogoMode) string {
	return fmt.Sprintf("/api/latest/fleet/logo?mode=%s&v=%d", mode, time.Now().UnixNano())
}

// orgLogoURLForMode returns the currently-configured URL for the given mode,
// preferring the new field but falling back to the deprecated one when only
// the legacy field is populated (e.g. a value written directly to the DB
// before NormalizeLogoFields could mirror it).
func orgLogoURLForMode(o *fleet.OrgInfo, mode fleet.OrgLogoMode) string {
	switch mode {
	case fleet.OrgLogoModeLight:
		if o.OrgLogoURLLightMode != "" {
			return o.OrgLogoURLLightMode
		}
		return o.OrgLogoURLLightBackground
	case fleet.OrgLogoModeDark:
		if o.OrgLogoURLDarkMode != "" {
			return o.OrgLogoURLDarkMode
		}
		return o.OrgLogoURL
	}
	return ""
}

// updateOrgLogoURLs sets (uploaded=true) or clears (uploaded=false) the
// AppConfig URL fields for the given modes. The dual-write between the
// new and deprecated fields is handled by NormalizeLogoFields.
func (svc *Service) updateOrgLogoURLs(ctx context.Context, modes []fleet.OrgLogoMode, uploaded bool) error {
	// Bypass the AppConfig cache so this read-modify-write picks up any
	// concurrent writes to other fields (e.g. a settings PATCH that
	// landed between the upload start and now). Otherwise we'd save a
	// stale snapshot and clobber that other write.
	ctx = ctxdb.BypassCachedMysql(ctx, true)
	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "loading app config")
	}
	for _, m := range modes {
		var url string
		if uploaded {
			url = orgLogoServingURL(m)
		}
		switch m {
		case fleet.OrgLogoModeLight:
			ac.OrgInfo.OrgLogoURLLightMode = url
			ac.OrgInfo.OrgLogoURLLightBackground = url
		case fleet.OrgLogoModeDark:
			ac.OrgInfo.OrgLogoURLDarkMode = url
			ac.OrgInfo.OrgLogoURL = url
		}
	}
	if err := svc.ds.SaveAppConfig(ctx, ac); err != nil {
		return ctxerr.Wrap(ctx, err, "saving app config after logo update")
	}
	return nil
}
