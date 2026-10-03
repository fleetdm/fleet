// OPENFRAME(mysql-multitenancy): per-request tenant pinning for the shared multi-tenant Fleet
// topology, active only in shared mode.
package service

import (
	"context"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/fleetdm/fleet/v4/server/contexts/token"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/google/uuid"
)

// openframeTenantHeader is the trusted tenant UUID the OpenFrame gateway injects (client-supplied
// copies are stripped upstream; fleet-service is reachable only via the gateway).
const openframeTenantHeader = "X-Tenant-Id"

// openframeSessionCookieNames are where the Fleet UI keeps the session key — "__Host-token" when
// served over HTTPS, "token" otherwise (frontend/utilities/auth_token). A browser attaches cookies
// to a WebSocket upgrade but cannot attach headers to one, so on the live-query result stream the
// cookie is the only place the caller's identity can come from.
var openframeSessionCookieNames = []string{"__Host-token", "token"}

// openframeSessionBootstrapPath matches the unauthenticated session endpoints a caller must reach
// before it holds the token the superuser check recognises it by, plus the SSO settings the login
// page reads before anyone has logged in. None of them read tenant-scoped data: Fleet users and
// sessions are instance-wide, and the SSO settings come from the instance app config row.
// Anchored on purpose — /api/{v}/fleet/users/{id}/sessions is an authenticated admin endpoint and
// must keep requiring a tenant.
var openframeSessionBootstrapPath = regexp.MustCompile(`^/api/[^/]+/fleet/(login|logout|sessions|sso(/callback)?)$`)

type openframeTeamEnsurer interface {
	EnsureOpenframeTeamID(ctx context.Context, tenantUUID string) (uint, error)
}

// openframeViewerResolver resolves the caller behind a session key. It is the auth.AuthViewer seam,
// narrowed to a func so the superuser path is testable without standing up a whole fleet.Service.
type openframeViewerResolver func(ctx context.Context, sessionKey string) (*viewer.Viewer, error)

// openframeSuperuserPredicate answers whether an e-mail is on the superuser allowlist. Passed in
// rather than read through fleet.IsOpenframeSuperuser directly because that allowlist is parsed
// from the environment exactly once per process, which a test cannot steer.
type openframeSuperuserPredicate func(email string) bool

// WithOpenframeTenant pins each control-plane request to the team named by the X-Tenant-Id header.
// Outside shared mode it returns next unchanged (zero overhead); in shared mode a non-exempt
// request without a resolvable tenant is rejected (fail closed) unless it authenticates as a
// configured superuser, which then runs unpinned — every tenant fence inert — on purpose.
func WithOpenframeTenant(ds openframeTeamEnsurer, svc fleet.Service, logger *slog.Logger, next http.Handler) http.Handler {
	if !fleet.IsOpenframeSharedMode() {
		return next
	}
	resolveViewer := func(ctx context.Context, sessionKey string) (*viewer.Viewer, error) {
		return auth.AuthViewer(ctx, sessionKey, svc)
	}
	return openframeTenantHandler(ds, resolveViewer, fleet.IsOpenframeSuperuser, logger, next)
}

// openframeTenantHandler is split out so it can be tested without toggling the cached shared-mode env.
func openframeTenantHandler(ds openframeTeamEnsurer, resolveViewer openframeViewerResolver,
	isSuperuser openframeSuperuserPredicate, logger *slog.Logger, next http.Handler,
) http.Handler {
	var teamIDByTenantUUID sync.Map // tenant UUID → team id uint (a tenant's team id never changes)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Agent/device/MDM planes carry no gateway header — their tenant comes from the
		// authenticated host / enroll secret (openframePinHostTeam, the enrollment pins). The
		// session endpoints carry none either, and cannot: they are what a caller goes through to
		// obtain the token it is then recognised by.
		if openframeTenantExemptPath(r.URL.Path) || openframeSessionBootstrapPath.MatchString(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		tenantUUID := strings.TrimSpace(r.Header.Get(openframeTenantHeader))
		if tenantUUID == "" {
			if isOpenframeSuperuserRequest(ctx, r, resolveViewer, isSuperuser) {
				logger.InfoContext(ctx, "openframe shared mode: serving request unpinned for superuser",
					"path", r.URL.Path, "remote_addr", r.RemoteAddr)
				next.ServeHTTP(w, r)
				return
			}
			logger.WarnContext(ctx, "openframe shared mode: rejecting request without tenant header",
				"path", r.URL.Path, "remote_addr", r.RemoteAddr)
			encodeError(ctx, fleet.NewAuthRequiredError("missing tenant"), w)
			return
		}

		teamID, cached := teamIDByTenantUUID.Load(tenantUUID)
		if !cached {
			// Validate before resolving: EnsureOpenframeTeamID would mint a team for any string.
			if _, err := uuid.Parse(tenantUUID); err != nil {
				logger.WarnContext(ctx, "openframe shared mode: rejecting request with malformed tenant header",
					"path", r.URL.Path, "remote_addr", r.RemoteAddr)
				encodeError(ctx, fleet.NewAuthRequiredError("invalid tenant"), w)
				return
			}
			id, err := ds.EnsureOpenframeTeamID(ctx, tenantUUID)
			if err != nil {
				logger.ErrorContext(ctx, "openframe shared mode: resolving tenant team",
					"tenant_uuid", tenantUUID, "err", err)
				encodeError(ctx, err, w)
				return
			}
			teamIDByTenantUUID.Store(tenantUUID, id)
			teamID = id
		}

		next.ServeHTTP(w, r.WithContext(fleet.NewOpenframeTeamContext(ctx, teamID.(uint))))
	})
}

// openframeTenantExemptPath matches the agent/device/MDM-enrollment planes, which derive their
// tenant from the authenticated principal (host node key / enroll secret / device cert) rather
// than the gateway X-Tenant-Id header.
//
// The MDM marker is "/api/mdm/" — the device enrollment/management endpoints (Apple
// /api/mdm/apple/{enroll,installer,account_driven_enroll}, Microsoft /api/mdm/microsoft…). It is
// deliberately NOT the broad "/mdm/": the user-authenticated admin MDM APIs live under
// /api/{v}/fleet/mdm/… (e.g. .../mdm/apple/commands, .../mdm/apple/enrollment_profile) and MUST
// still require X-Tenant-Id in shared mode. The raw device protocol (/mdm/apple/scep, /mdm/apple/mdm,
// SCEP proxy) is served on the root mux and never reaches this wrapper.
//
// Residual: a few device-facing DEP endpoints live under /api/{v}/fleet/mdm/ too (GET
// .../mdm/bootstrap download, GET .../mdm/setup/eula/{token}) and share their path with admin
// variants that differ only by HTTP method — so path-only matching cannot exempt them without also
// exempting the admin route. They are NOT exempted here (they'd need X-Tenant-Id). This is
// acceptable because OpenFrame does not use Apple/Windows MDM; if it ever does, make this exemption
// method-aware. See openframe/docs/agent-ingestion-isolation.md.
func openframeTenantExemptPath(path string) bool {
	for _, marker := range []string{"/osquery/", "/fleet/orbit/", "/fleet/device/", "/api/mdm/", "/fleet/ota_enrollment"} {
		if strings.Contains(path, marker) {
			return true
		}
	}
	return false
}

// isOpenframeSuperuserRequest reports whether an unpinned request may run unpinned anyway: it must
// carry a session key that resolves to a live Fleet session whose user is on the superuser
// allowlist. This widens what an already-authenticated operator may reach — never who may
// authenticate — and with an empty allowlist (the default) it is always false, so shared mode stays
// fail closed. Reached only when the gateway header is absent, so normal tenant traffic never pays
// for the session lookup.
func isOpenframeSuperuserRequest(ctx context.Context, r *http.Request, resolveViewer openframeViewerResolver,
	isSuperuser openframeSuperuserPredicate,
) bool {
	sessionKey := openframeSessionKey(r)
	if sessionKey == "" {
		return false
	}
	vc, err := resolveViewer(ctx, sessionKey)
	if err != nil || !vc.CanPerformActions() {
		return false
	}
	email := vc.Email()
	return isSuperuser(email)
}

// openframeSessionKey reads the caller's Fleet session key, preferring the Authorization header and
// falling back to the UI's session cookie — the WebSocket upgrade that carries the live-query
// result stream can carry no header, so without the cookie a browser could never be recognised.
func openframeSessionKey(r *http.Request) string {
	bearer := token.FromHTTPRequest(r)
	if bearer != "" {
		return string(bearer)
	}
	for _, name := range openframeSessionCookieNames {
		cookie, err := r.Cookie(name)
		if err == nil && cookie.Value != "" {
			return cookie.Value
		}
	}
	return ""
}

// openframePinHostTeam scopes ctx to the authenticated host's team in shared mode; a host with no
// team fails auth (fail closed) rather than running unscoped. No-op outside shared mode.
func openframePinHostTeam(ctx context.Context, host *fleet.Host) (context.Context, error) {
	if !fleet.IsOpenframeSharedMode() {
		return ctx, nil
	}
	return openframePinHostTeamShared(ctx, host)
}

func openframePinHostTeamShared(ctx context.Context, host *fleet.Host) (context.Context, error) {
	if host == nil || host.TeamID == nil || *host.TeamID == 0 {
		return ctx, fleet.NewAuthFailedError("openframe shared mode: authenticated host has no team")
	}
	return fleet.NewOpenframeTeamContext(ctx, *host.TeamID), nil
}
