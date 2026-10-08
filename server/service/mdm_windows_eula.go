package service

import (
	"context"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/markdown"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"golang.org/x/sync/singleflight"
)

////////////////////////////////////////////////////////////////////////////////
// POST /setup_experience/windows_eula
////////////////////////////////////////////////////////////////////////////////

type createMDMWindowsEULARequest struct {
	EULA   *multipart.FileHeader
	DryRun bool `query:"dry_run,optional"` // if true, apply validation but do not save changes
}

func (createMDMWindowsEULARequest) DecodeRequest(ctx context.Context, r *http.Request) (any, error) {
	eula, dryRun, err := decodeEULAUpload(r, "windows_eula", fleet.MaxWindowsEULARequestSize)
	if err != nil {
		return nil, err
	}
	return &createMDMWindowsEULARequest{
		EULA:   eula,
		DryRun: dryRun,
	}, nil
}

type createMDMWindowsEULAResponse struct {
	Err error `json:"error,omitempty"`
}

func (r createMDMWindowsEULAResponse) Error() error { return r.Err }

func createMDMWindowsEULAEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*createMDMWindowsEULARequest)
	ff, err := req.EULA.Open()
	if err != nil {
		return createMDMWindowsEULAResponse{Err: err}, nil
	}
	defer ff.Close()

	if err := svc.MDMCreateWindowsEULA(ctx, req.EULA.Filename, ff, req.DryRun); err != nil {
		return createMDMWindowsEULAResponse{Err: err}, nil
	}

	return createMDMWindowsEULAResponse{}, nil
}

func (svc *Service) MDMCreateWindowsEULA(ctx context.Context, name string, file io.ReadSeeker, dryRun bool) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// GET /setup_experience/windows_eula/:token
////////////////////////////////////////////////////////////////////////////////

type getMDMWindowsEULARequest struct {
	Token string `url:"token"`
}

type getMDMWindowsEULAResponse struct {
	Err error `json:"error,omitempty"`

	// fields used in hijackRender to build the response
	eula *fleet.MDMEULA
}

func (r getMDMWindowsEULAResponse) Error() error { return r.Err }

func (r getMDMWindowsEULAResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	// Markdown is downloaded, never rendered as a page in the admin's browser.
	writeEULAFile(ctx, w, r.eula, "text/markdown; charset=utf-8", true)
}

func getMDMWindowsEULAEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getMDMWindowsEULARequest)

	eula, err := svc.MDMGetWindowsEULABytes(ctx, req.Token)
	if err != nil {
		return getMDMWindowsEULAResponse{Err: err}, nil
	}

	return getMDMWindowsEULAResponse{eula: eula}, nil
}

func (svc *Service) MDMGetWindowsEULABytes(ctx context.Context, token string) (*fleet.MDMEULA, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// GET /setup_experience/windows_eula/metadata
////////////////////////////////////////////////////////////////////////////////

type getMDMWindowsEULAMetadataRequest struct{}

type getMDMWindowsEULAMetadataResponse struct {
	*fleet.MDMEULA
	Err error `json:"error,omitempty"`
}

func (r getMDMWindowsEULAMetadataResponse) Error() error { return r.Err }

func getMDMWindowsEULAMetadataEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	eula, err := svc.MDMGetWindowsEULAMetadata(ctx)
	if err != nil && !fleet.IsNotFound(err) {
		return getMDMWindowsEULAMetadataResponse{Err: err}, nil
	}

	if eula == nil {
		// Returned as an error rather than in the response so the server sees
		// the 404 without logging it as an error.
		return nil, newNotFoundError()
	}

	return getMDMWindowsEULAMetadataResponse{MDMEULA: eula}, nil
}

func (svc *Service) MDMGetWindowsEULAMetadata(ctx context.Context) (*fleet.MDMEULA, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// DELETE /setup_experience/windows_eula/:token
////////////////////////////////////////////////////////////////////////////////

type deleteMDMWindowsEULARequest struct {
	Token  string `url:"token"`
	DryRun bool   `query:"dry_run,optional"` // if true, apply validation but do not delete
}

type deleteMDMWindowsEULAResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteMDMWindowsEULAResponse) Error() error { return r.Err }

func deleteMDMWindowsEULAEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*deleteMDMWindowsEULARequest)
	if err := svc.MDMDeleteWindowsEULA(ctx, req.Token, req.DryRun); err != nil {
		return deleteMDMWindowsEULAResponse{Err: err}, nil
	}
	return deleteMDMWindowsEULAResponse{}, nil
}

func (svc *Service) MDMDeleteWindowsEULA(ctx context.Context, token string, dryRun bool) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

type windowsTOSTemplateData struct {
	RedirectURL string
	ClientData  string
	// Content is the admin's rendered agreement. Empty means the template shows Fleet's default terms.
	Content template.HTML
}

// customWindowsTOSContent returns the admin-uploaded agreement rendered as HTML, or empty when there is none or it
// cannot be rendered. This page sits on the enrollment critical path, where any error surfaces to the end user as a
// lost network connection, so a bad document falls back to the default terms and is logged rather than failing the
// request.
func (svc *Service) customWindowsTOSContent(ctx context.Context) template.HTML {
	// The agreement is a Premium feature; after a downgrade the admin can no longer manage it, so devices must not keep
	// seeing it either.
	if !license.IsPremium(ctx) {
		windowsTOSCache.clear()
		return ""
	}

	// The page is unauthenticated, so only the small metadata row is read per request; the document is loaded and
	// rendered once per upload.
	meta, err := svc.ds.MDMGetEULAMetadata(ctx, fleet.MDMEULAPlatformWindows)
	switch {
	case fleet.IsNotFound(err):
		windowsTOSCache.clear()
		return ""
	case err != nil:
		// Keep showing the agreement devices already get rather than switch them to the default terms while the
		// database is unreachable.
		content := windowsTOSCache.last()
		svc.logger.ErrorContext(ctx, "loading Windows end user agreement", "err", err, "serving_cached", content != "")
		ctxerr.Handle(ctx, err)
		return content
	}
	if cached, ok := windowsTOSCache.get(meta.Token); ok {
		return cached
	}

	// Devices enrolling right after an upload miss the cache together; one request loads and renders the document for
	// all of them. It must not carry that request's cancellation to the others, but it is capped so a stuck query
	// can't hold the flight that every later request joins.
	result := windowsTOSRender.DoChan(meta.Token, func() (any, error) {
		flightCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), windowsTOSRenderTimeout)
		defer cancel()
		return svc.renderWindowsTOS(flightCtx, meta.Token), nil
	})
	select {
	case res := <-result:
		return res.Val.(template.HTML)
	case <-ctx.Done():
		return windowsTOSCache.last()
	}
}

// renderWindowsTOS loads and renders the agreement for token and caches the result. A document that fails to render is
// cached as empty, so it is not reloaded on every request.
func (svc *Service) renderWindowsTOS(ctx context.Context, token string) template.HTML {
	gen := windowsTOSCache.generation()
	// A render that finished just before this flight started has cached it.
	if cached, ok := windowsTOSCache.get(token); ok {
		return cached
	}

	eula, err := svc.ds.MDMGetEULABytes(ctx, fleet.MDMEULAPlatformWindows, token)
	switch {
	case fleet.IsNotFound(err):
		// Deleted or replaced since the metadata was read; the next request reads the new state.
		return ""
	case err != nil:
		content := windowsTOSCache.last()
		svc.logger.ErrorContext(ctx, "loading Windows end user agreement", "err", err, "serving_cached", content != "")
		ctxerr.Handle(ctx, err)
		return content
	}

	rendered, err := markdown.RenderTerms(eula.Bytes)
	if err != nil {
		svc.logger.ErrorContext(ctx, "rendering Windows end user agreement, serving default terms", "err", err)
		ctxerr.Handle(ctx, err)
		rendered = ""
	}
	content := template.HTML(rendered) //nolint:gosec // output of a sanitizer, see pkg/markdown
	windowsTOSCache.setIfUnchanged(gen, token, content)
	return content
}

// windowsTOSCache holds the rendered agreement for the current upload. Tokens are unique per upload, so a token match
// means the content is current.
var (
	windowsTOSCache  renderedTOSCache
	windowsTOSRender singleflight.Group
)

const windowsTOSRenderTimeout = 30 * time.Second

type renderedTOSCache struct {
	mu      sync.Mutex
	gen     uint64 // changes on every write, so a render can tell the cache moved on while it ran
	token   string
	content template.HTML
}

func (c *renderedTOSCache) generation() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.gen
}

func (c *renderedTOSCache) get(token string) (template.HTML, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token == "" || c.token != token {
		return "", false
	}
	return c.content, true
}

// setIfUnchanged stores content only if nothing was written since gen was read: a render that started before a newer
// upload or a delete must not replace that upload or bring the deleted agreement back. Losing the race costs one extra
// render on the next request.
func (c *renderedTOSCache) setIfUnchanged(gen uint64, token string, content template.HTML) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.gen != gen {
		return
	}
	c.gen++
	c.token, c.content = token, content
}

// last returns the most recent content whatever its token, for when the current token can't be read.
func (c *renderedTOSCache) last() template.HTML {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.content
}

func (c *renderedTOSCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gen++
	c.token, c.content = "", ""
}
