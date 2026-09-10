package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	otelmw "github.com/fleetdm/fleet/v4/server/service/middleware/otel"
	"github.com/gorilla/mux"
)

// gorilla/mux matches by walking the whole route table in registration order, running a compiled regex per candidate until one
// matches. Fleet registers ~550 routes and the agent endpoints register late, so every osquery and orbit request runs several
// hundred regex executions before it reaches its handler. fastPathHandler puts a stdlib ServeMux, which matches on a segment
// trie in time proportional to the path length rather than to the table size, in front of the gorilla router and gives it every
// route it can match with identical results. Anything else falls through to gorilla, which stays the source of truth for
// matching semantics.
type fastPathHandler struct {
	fast   *http.ServeMux
	router *mux.Router
}

func (h *fastPathHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Two request shapes are matched differently by the two routers, so gorilla handles both. gorilla matches on the decoded
	// path, which turns an encoded separator into a segment break, while the stdlib mux keeps it inside a single segment. And
	// an unclean path is redirected by gorilla with a 301 but by the stdlib mux with a 307.
	if r.URL.RawPath != "" || !isCanonicalPath(r.URL.Path) {
		h.router.ServeHTTP(w, r)
		return
	}
	h.fast.ServeHTTP(w, r)
}

// Router exposes the gorilla router underneath so callers that introspect the route table, such as endpoint catalog
// validation, keep working once the fast path is installed.
func (h *fastPathHandler) Router() *mux.Router { return h.router }

// fastPathExcluded lists method and path templates that have to stay on gorilla. Each is one half of a pair that gorilla tells
// apart only through a regex constraint on a path variable, which a stdlib pattern cannot express and which the stdlib mux
// rejects as an ambiguous registration:
//
//   - "/hosts/identifier/{identifier}" against "/hosts/{id:[0-9]+}/<subresource>"
//   - "/scripts/batch/summary/{id}" against "/scripts/batch/{id}/host_results"
//   - the two spellings of the host profile resend route
//
// Both halves are listed. Excluding only the half the stdlib mux rejected would leave the other half free to claim requests
// that belong to the one that left. TestFastPathExclusionsCoverEveryAmbiguousRoute keeps the list in sync with the route table.
var fastPathExcluded = map[string]struct{}{
	"GET /api/_version_/fleet/hosts/identifier/{identifier}":                                 {},
	"GET /api/_version_/fleet/hosts/{id}/certificates":                                       {},
	"GET /api/_version_/fleet/hosts/{id}/configuration_profiles":                             {},
	"GET /api/_version_/fleet/hosts/{id}/dep_assignment":                                     {},
	"GET /api/_version_/fleet/hosts/{id}/device_mapping":                                     {},
	"GET /api/_version_/fleet/hosts/{id}/device_url":                                         {},
	"GET /api/_version_/fleet/hosts/{id}/encryption_key":                                     {},
	"GET /api/_version_/fleet/hosts/{id}/health":                                             {},
	"GET /api/_version_/fleet/hosts/{id}/macadmins":                                          {},
	"GET /api/_version_/fleet/hosts/{id}/managed_account_password":                           {},
	"GET /api/_version_/fleet/hosts/{id}/mdm":                                                {},
	"GET /api/_version_/fleet/hosts/{id}/queries":                                            {},
	"GET /api/_version_/fleet/hosts/{id}/recovery_lock_password":                             {},
	"GET /api/_version_/fleet/hosts/{id}/reports":                                            {},
	"GET /api/_version_/fleet/hosts/{id}/scripts":                                            {},
	"GET /api/_version_/fleet/hosts/{id}/software":                                           {},
	"GET /api/_version_/fleet/scripts/batch/summary/{batch_execution_id}":                    {},
	"GET /api/_version_/fleet/scripts/batch/{batch_execution_id}/host_results":               {},
	"GET /api/_version_/fleet/scripts/batch/{batch_execution_id}/host-results":               {},
	"POST /api/_version_/fleet/hosts/{host_id}/configuration_profiles/resend/{profile_uuid}": {},
	"POST /api/_version_/fleet/hosts/{host_id}/configuration_profiles/{profile_uuid}/resend": {},
}

var (
	fleetVersionVar = regexp.MustCompile(`\{fleetversion:\(\?:([^)]*)\)\}`)
	constrainedVar  = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*):([^}]*)\}`)
	anyVar          = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)(?::[^}]*)?\}`)
)

// newFastPathHandler builds the fast path from an already-registered gorilla router. It has to run after every route is
// registered and after addMetrics, because the handlers it promotes are the fully wrapped ones. middlewares are the
// route-agnostic wrappers gorilla applies through Use; they are applied again here because a request served by the fast path
// never enters the gorilla router.
//
// If any route cannot be promoted safely the whole fast path is dropped and the gorilla router is returned unchanged, so a
// route added later can only cost throughput, never correctness or a failed startup.
func newFastPathHandler(r *mux.Router, middlewares []mux.MiddlewareFunc, cfg config.FleetConfig,
	logger *slog.Logger,
) http.Handler {
	fast, err := buildFastPathMux(r, middlewares, cfg)
	if err != nil {
		logger.WarnContext(context.Background(),
			"stdlib fast-path router disabled, falling back to gorilla/mux for all routes", "err", err)
		return r
	}
	return &fastPathHandler{fast: fast, router: r}
}

func buildFastPathMux(r *mux.Router, middlewares []mux.MiddlewareFunc, cfg config.FleetConfig) (*http.ServeMux, error) {
	fast := http.NewServeMux()
	claimed := make(map[string]struct{})

	err := r.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		tpl, err := route.GetPathTemplate()
		if err != nil || strings.HasSuffix(tpl, "/") {
			// A route with no path, or a PathPrefix route whose subtree semantics are left to gorilla.
			return nil //nolint:nilerr
		}
		methods, err := route.GetMethods()
		if err != nil || len(methods) != 1 {
			return nil //nolint:nilerr
		}
		handler := route.GetHandler()
		if handler == nil {
			return nil
		}
		method := methods[0]
		if _, excluded := fastPathExcluded[unversionedKey(method, tpl)]; excluded {
			return nil
		}
		matchers, ok := varMatchers(tpl)
		if !ok {
			return nil
		}

		wrapped := handler
		for _, middleware := range slices.Backward(middlewares) {
			wrapped = middleware.Middleware(wrapped)
		}
		if cfg.Logging.TracingEnabled && cfg.OTELEnabled() {
			// Span names have to keep matching the gorilla route template: the trace sampler tiers routes by span name.
			wrapped = otelmw.WrapHandler(wrapped, tpl, cfg)
		}

		varNames := make([]string, 0, 4)
		for _, m := range anyVar.FindAllStringSubmatch(tpl, -1) {
			varNames = append(varNames, m[1])
		}

		for _, pattern := range stdlibPatterns(tpl) {
			full := method + " " + pattern
			if _, taken := claimed[full]; taken {
				// Two routes registered the same method and path. gorilla dispatches to whichever came first, so the
				// fast path keeps the first and ignores this one.
				continue
			}
			bridged := newFastPathRoute(tpl, method, versionSegment(tpl, pattern), varNames, matchers, wrapped, r)
			if err := handleNoConflict(fast, full, bridged); err != nil {
				return fmt.Errorf("route %s: %w", route.GetName(), err)
			}
			claimed[full] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Everything the fast path did not claim, including method mismatches and unknown paths, is served by gorilla exactly as it
	// is today.
	fast.Handle("/", r)
	return fast, nil
}

// handleNoConflict registers a pattern, turning the panic the stdlib mux raises for an ambiguous registration into an error.
// Reaching it means a new route overlaps an existing one in a way only a regex constraint can separate, and that both halves
// need an entry in fastPathExcluded.
func handleNoConflict(m *http.ServeMux, pattern string, h http.Handler) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%q is ambiguous on a stdlib ServeMux and needs an entry in fastPathExcluded: %v", pattern, r)
		}
	}()
	m.Handle(pattern, h)
	return nil
}

// newFastPathRoute adapts a stdlib match back to what Fleet's gorilla-shaped handlers expect: mux vars and the route template
// in context. It also hands back to gorilla the requests a stdlib pattern matches more loosely than the gorilla template did,
// so a request the router rejects today is still rejected by the router.
func newFastPathRoute(tpl, method, version string, varNames []string, matchers []varMatcher,
	handler, fallback http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// A stdlib GET pattern also serves HEAD, where gorilla answers 405.
		if req.Method != method {
			fallback.ServeHTTP(w, req)
			return
		}
		for _, matcher := range matchers {
			if !matcher.allow(req.PathValue(matcher.name)) {
				fallback.ServeHTTP(w, req)
				return
			}
		}
		if len(varNames) > 0 {
			vars := make(map[string]string, len(varNames))
			for _, name := range varNames {
				vars[name] = req.PathValue(name)
			}
			if version != "" {
				vars["fleetversion"] = version
			}
			req = mux.SetURLVars(req, vars)
		}
		// mux.CurrentRoute is nil here, so the route template goes into the context directly for the API-only endpoint
		// restriction check, which fails closed without it.
		req = req.WithContext(endpointer.WithRouteTemplate(req.Context(), tpl))
		handler.ServeHTTP(w, req)
	})
}

// varMatcher validates a path variable the way gorilla's compiled regex would, without running a regex. A value that fails is
// handed to gorilla, which produces the same 404 it produces today rather than letting the request reach a decoder.
type varMatcher struct {
	name  string
	allow func(string) bool
}

const (
	digitPattern        = "[0-9]+"
	hexAndDashPattern   = "[a-f0-9-]+"
	alphanumDashPattern = "[a-zA-Z0-9-]+"

	hexAndDashChars   = "0123456789abcdef-"
	alphanumDashChars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-"
)

// varMatchers returns one matcher per regex-constrained variable in the template. It reports false for a constraint this code
// cannot check without a regex, in which case the route stays on gorilla.
func varMatchers(tpl string) ([]varMatcher, bool) {
	var matchers []varMatcher
	for _, m := range constrainedVar.FindAllStringSubmatch(tpl, -1) {
		name, expr := m[1], m[2]
		if name == "fleetversion" {
			continue // expanded into literal segments by stdlibPatterns
		}
		switch expr {
		case digitPattern:
			matchers = append(matchers, varMatcher{name: name, allow: digitsOnly})
		case alphanumDashPattern:
			matchers = append(matchers, varMatcher{name: name, allow: charsetMatcher(alphanumDashChars)})
		case hexAndDashPattern:
			matchers = append(matchers, varMatcher{name: name, allow: charsetMatcher(hexAndDashChars)})
		default:
			return nil, false
		}
	}
	return matchers, true
}

func digitsOnly(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func charsetMatcher(set string) func(string) bool {
	var allowed [256]bool
	for i := 0; i < len(set); i++ {
		allowed[set[i]] = true
	}
	return func(s string) bool {
		if s == "" {
			return false
		}
		for i := 0; i < len(s); i++ {
			if !allowed[s[i]] {
				return false
			}
		}
		return true
	}
}

// stdlibPatterns converts a gorilla template into stdlib ServeMux patterns. The fleetversion alternation becomes one pattern
// per literal version so an unknown version is still a 404 at the router. Other constraints are dropped from the pattern and
// enforced by varMatchers instead.
//
// Versions are deduplicated. A bounded context that lists "latest" in its own version set gets it appended a second time by
// the endpointer, producing a template like {fleetversion:(?:v1|latest|latest)}; the repeat is invisible to a regex but would
// otherwise register the same stdlib pattern twice.
func stdlibPatterns(tpl string) []string {
	base := constrainedVar.ReplaceAllString(tpl, "{$1}")
	m := fleetVersionVar.FindStringSubmatch(tpl)
	if m == nil {
		return []string{base}
	}
	versions := strings.Split(m[1], "|")
	slices.Sort(versions)
	patterns := make([]string, 0, len(versions))
	for _, version := range slices.Compact(versions) {
		patterns = append(patterns, strings.Replace(base, "{fleetversion}", version, 1))
	}
	return patterns
}

// versionSegment recovers the literal version a pattern was expanded with, so the fast path can still expose it as a mux var.
func versionSegment(tpl, pattern string) string {
	if !strings.Contains(tpl, "{fleetversion:") {
		return ""
	}
	if segments := strings.Split(pattern, "/"); len(segments) > 2 {
		return segments[2]
	}
	return ""
}

// unversionedKey renders a route in the method and path form used by fastPathExcluded.
func unversionedKey(method, tpl string) string {
	p := fleetVersionVar.ReplaceAllString(tpl, "_version_")
	return method + " " + constrainedVar.ReplaceAllString(p, "{$1}")
}

// isCanonicalPath reports whether p is already in the form both routers would clean it to. It mirrors the cleanPath helper
// gorilla and net/http each carry, and does not allocate for a path that is already canonical.
func isCanonicalPath(p string) bool {
	if p == "" || p[0] != '/' {
		return false
	}
	cleaned := path.Clean(p)
	if p[len(p)-1] == '/' && cleaned != "/" {
		cleaned += "/"
	}
	return cleaned == p
}
