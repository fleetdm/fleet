package service

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/gorilla/mux"
	_ "golang.org/x/image/webp"
)

const orgLogoMaxFileSize = fleet.OrgLogoMaxFileSize

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

// looksLikeSVG only decides which validator to run; validateSVG is what
// actually rejects unsafe content.
func looksLikeSVG(b []byte) bool {
	// Editors may prepend a UTF-8 BOM (byte-order mark) or whitespace.
	if bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}) {
		b = b[3:]
	}
	b = bytes.TrimLeft(b, " \t\r\n")
	// 512B keeps routing cheap; the root <svg> is always near the top.
	const window = 512
	if len(b) > window {
		b = b[:window]
	}
	return bytes.Contains(bytes.ToLower(b), []byte("<svg"))
}

// contentTypeForBytes returns the HTTP Content-Type for the accepted
// formats (PNG, JPEG, WebP, SVG) and "" for anything else. Used by the GET
// HijackRender to set the correct response header.
func contentTypeForBytes(b []byte) string {
	switch {
	case bytes.HasPrefix(b, pngMagic):
		return "image/png"
	case bytes.HasPrefix(b, jpegMagic):
		return "image/jpeg"
	case hasWebPMagic(b):
		return "image/webp"
	case looksLikeSVG(b):
		return "image/svg+xml"
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

func validateOrgLogoBytes(b []byte) error {
	if int64(len(b)) > orgLogoMaxFileSize {
		return &fleet.BadRequestError{Message: "logo must be 100KB or less"}
	}
	if looksLikeSVG(b) {
		return validateSVG(b)
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return &fleet.BadRequestError{
			Message:     "logo must be a valid PNG, JPEG, WebP, or SVG image",
			InternalErr: err,
		}
	}
	switch format {
	case "png", "jpeg", "webp":
		return nil
	}
	return &fleet.BadRequestError{Message: "logo must be a PNG, JPEG, WebP, or SVG file"}
}

// Elements that can run scripts or load foreign content. <img>-rendered
// SVGs are script-sandboxed, but pasting the URL loads it as a document
// — so reject structurally instead of trusting the renderer.
var disallowedSVGElements = map[string]struct{}{
	"script":        {},
	"foreignobject": {},
	"iframe":        {},
	"object":        {},
	"embed":         {},
}

// validateSVG rejects unsafe SVG content. Leaving decoder.Entity nil and
// rejecting DOCTYPE neutralizes XXE (external entities reading local
// files) and billion-laughs (DoS via recursive entity expansion).
func validateSVG(b []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(b))
	decoder.Strict = true

	sawRoot := false
	for {
		tok, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return &fleet.BadRequestError{
				Message:     "logo is not valid SVG",
				InternalErr: err,
			}
		}
		switch t := tok.(type) {
		case xml.StartElement:
			name := strings.ToLower(t.Name.Local)
			if !sawRoot {
				if name != "svg" {
					return &fleet.BadRequestError{Message: "logo SVG must have <svg> as the root element"}
				}
				sawRoot = true
			}
			if _, bad := disallowedSVGElements[name]; bad {
				return &fleet.BadRequestError{Message: fmt.Sprintf("SVG element <%s> is not allowed", name)}
			}
			for _, attr := range t.Attr {
				attrName := strings.ToLower(attr.Name.Local)
				// on* (onclick, onload, …) is SVG's main XSS vector.
				if strings.HasPrefix(attrName, "on") {
					return &fleet.BadRequestError{Message: "SVG event-handler attributes are not allowed"}
				}
				// Name.Local matches both href and xlink:href.
				if attrName == "href" || attrName == "src" {
					val := strings.ToLower(strings.TrimSpace(attr.Value))
					if strings.HasPrefix(val, "javascript:") || strings.HasPrefix(val, "data:") {
						return &fleet.BadRequestError{Message: "SVG javascript: and data: URLs are not allowed"}
					}
				}
			}
		case xml.Directive:
			// DOCTYPE / ENTITY (XXE, billion-laughs vectors).
			return &fleet.BadRequestError{Message: "SVG DTD/DOCTYPE declarations are not allowed"}
		}
	}
	if !sawRoot {
		return &fleet.BadRequestError{Message: "logo SVG missing root <svg> element"}
	}
	return nil
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

	// Filter to modes that actually have something to delete.
	// (The S3 store's Delete is silent on missing keys, so without this check
	// we'd persist a "logo deleted" activity.)
	modes := mode.Modes()
	toDelete := modes[:0:0]
	for _, m := range modes {
		exists, err := svc.orgLogoStore.Exists(ctx, m)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "checking org logo exists (%s)", m)
		}
		if exists {
			toDelete = append(toDelete, m)
		}
	}
	if len(toDelete) == 0 {
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
	if err := svc.updateOrgLogoURLs(ctx, toDelete, false); err != nil {
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
	// Re-validate on read so a blob planted directly in the object store
	// (bypassing the upload API) is still rejected.
	if err := validateOrgLogoBytes(body); err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "stored org logo failed validation")
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

// orgLogoServingURL builds the URL persisted in AppConfig after an upload. The `v` param is a cache-buster (ignored server-side; only `mode` is read).
func orgLogoServingURL(mode fleet.OrgLogoMode) string {
	return fmt.Sprintf("/api/latest/fleet/logo?mode=%s&v=%d", mode, time.Now().UnixNano())
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
