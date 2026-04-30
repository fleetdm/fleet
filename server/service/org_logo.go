package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/gorilla/mux"
)

const orgLogoMaxFileSize = 100 * 1024

// Magic-byte signatures used to identify accepted image formats. We compare
// against raw upload bytes rather than trusting the multipart Content-Type
// header.
var (
	pngMagic  = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	jpegMagic = []byte{0xFF, 0xD8, 0xFF}
)

// hasWebPMagic reports whether b begins with a WebP RIFF container header
// ("RIFF" at bytes 0-3, "WEBP" at bytes 8-11). WebP isn't a simple prefix
// check because the 4 bytes between the two markers carry the file size.
func hasWebPMagic(b []byte) bool {
	return len(b) >= 12 && bytes.Equal(b[0:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP"))
}

// contentTypeForBytes returns the HTTP Content-Type for the accepted
// formats (PNG, JPEG, WebP) and "" for anything else. Used by the GET
// HijackRender to set the correct response header.
func contentTypeForBytes(b []byte) string {
	switch {
	case bytes.HasPrefix(b, pngMagic):
		return "image/png"
	case bytes.HasPrefix(b, jpegMagic):
		return "image/jpeg"
	case hasWebPMagic(b):
		return "image/webp"
	}
	return ""
}

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
	if err := validateOrgLogoBytes(body); err != nil {
		return nil, err
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
	contentType := contentTypeForBytes(r.Body)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(r.Body)))
	w.Header().Set("Cache-Control", "no-store")
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

func validateOrgLogoBytes(b []byte) error {
	if int64(len(b)) > orgLogoMaxFileSize {
		return &fleet.BadRequestError{Message: "logo must be 100KB or less"}
	}
	if contentTypeForBytes(b) != "" {
		return nil
	}
	return &fleet.BadRequestError{Message: "logo must be a PNG, JPEG, or WebP file"}
}

// Service implementation

func (svc *Service) UploadOrgLogo(ctx context.Context, mode fleet.OrgLogoMode, content io.ReadSeeker) error {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionWrite); err != nil {
		return err
	}
	if err := requireGlobalAdmin(ctx); err != nil {
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
	if int64(len(body)) > orgLogoMaxFileSize {
		return &fleet.BadRequestError{Message: "logo must be 100KB or less"}
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
	if err := requireGlobalAdmin(ctx); err != nil {
		return err
	}
	if !mode.IsValid() {
		return &fleet.BadRequestError{Message: fmt.Sprintf("invalid mode %q", mode)}
	}
	if svc.orgLogoStore == nil {
		return ctxerr.New(ctx, "org logo store not configured")
	}

	// Try every requested mode even if one fails so a partial in-store
	// state isn't left where one blob is gone but another lingers under a
	// URL we already cleared. URLs are only cleared if all deletes
	// succeeded — on a partial failure the caller retries and Delete is
	// idempotent.
	modes := mode.Modes()
	var errs []error
	for _, m := range modes {
		if err := svc.orgLogoStore.Delete(ctx, m); err != nil {
			errs = append(errs, ctxerr.Wrapf(ctx, err, "deleting org logo (%s)", m))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	if err := svc.updateOrgLogoURLs(ctx, modes, false); err != nil {
		return err
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
	if err := validateOrgLogoBytes(body); err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "validating stored org logo")
	}
	return body, int64(len(body)), nil
}

func requireGlobalAdmin(ctx context.Context) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok || vc.User == nil || vc.User.GlobalRole == nil || *vc.User.GlobalRole != fleet.RoleAdmin {
		return authz.ForbiddenWithInternal("org logo write requires global admin", nil, nil, nil)
	}
	return nil
}

func orgLogoServingURL(mode fleet.OrgLogoMode) string {
	return fmt.Sprintf("/api/latest/fleet/logo?mode=%s", mode)
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
