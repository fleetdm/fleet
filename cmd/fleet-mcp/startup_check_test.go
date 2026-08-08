package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestVerifyEndpointAccess_AllReachable(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 404 (nonexistent probe IDs) still proves the route is reachable.
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	fc := newTestClient(srv.URL)
	blocked, unverified := verifyEndpointAccess(t.Context(), fc)
	if len(blocked) != 0 {
		t.Errorf("expected no blocked endpoints, got %v", blocked)
	}
	if unverified != 0 {
		t.Errorf("expected 0 unverified endpoints, got %d", unverified)
	}
}

func TestVerifyEndpointAccess_CountsUnverifiable(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	srv.Close() // unreachable Fleet: every probe fails at the transport level

	fc := newTestClient(srv.URL)
	blocked, unverified := verifyEndpointAccess(t.Context(), fc)
	if len(blocked) != 0 {
		t.Errorf("expected no blocked endpoints, got %v", blocked)
	}
	if unverified != len(requiredEndpoints) {
		t.Errorf("expected %d unverified endpoints, got %d", len(requiredEndpoints), unverified)
	}
}

func TestVerifyEndpointAccess_FlagsBlockedEndpoints(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/fleet/software/titles"),
			r.Method == "POST" && r.URL.Path == "/api/v1/fleet/reports/run":
			w.WriteHeader(http.StatusForbidden)
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()

	fc := newTestClient(srv.URL)
	blocked, unverified := verifyEndpointAccess(t.Context(), fc)
	if unverified != 0 {
		t.Errorf("expected 0 unverified endpoints, got %d", unverified)
	}
	want := []string{
		"POST /api/v1/fleet/reports/run",
		"GET /api/v1/fleet/software/titles",
		"GET /api/v1/fleet/software/titles/:id",
	}
	slices.Sort(blocked)
	slices.Sort(want)
	if !slices.Equal(blocked, want) {
		t.Errorf("blocked endpoints mismatch:\n got %v\nwant %v", blocked, want)
	}
}

func TestRequiredEndpoints_ProbesAreSideEffectFree(t *testing.T) {
	t.Parallel()
	// Every POST probe must carry a body that fails Fleet-side validation
	// before anything is created or executed: no query SQL, no real
	// host/query IDs in the path.
	realIDRe := regexp.MustCompile(`^[1-9][0-9]*$`)
	for _, e := range requiredEndpoints {
		if e.method == "GET" {
			continue
		}
		if e.body == nil {
			t.Errorf("POST probe %s must send a JSON body", e.route)
		}
		path, _, _ := strings.Cut(e.probe, "?")
		for _, seg := range strings.Split(path, "/") {
			if realIDRe.MatchString(seg) {
				t.Errorf("POST probe %s must not reference a potentially real ID (segment %q)", e.route, seg)
			}
		}
	}
}

// normalizeAPIPath reduces a Fleet API path to a comparable shape: the query
// string is dropped and any path segment that is a fmt verb (%d, %s), a
// :param placeholder, or a bare number is replaced with "*".
func normalizeAPIPath(path string) string {
	numRe := regexp.MustCompile(`^[0-9]+$`)
	path, _, _ = strings.Cut(path, "?")
	segs := strings.Split(path, "/")
	for i, s := range segs {
		if strings.Contains(s, "%") || strings.HasPrefix(s, ":") || numRe.MatchString(s) {
			segs[i] = "*"
		}
	}
	return strings.Join(segs, "/")
}

// TestRequiredEndpointsCoverSourcePaths fails when a Fleet API path used
// anywhere in this package is missing from requiredEndpoints — the drift
// this startup check exists to prevent. When it fails, add the new route to
// requiredEndpoints in startup_check.go AND to the "Required Fleet API
// endpoints" section of README.md.
func TestRequiredEndpointsCoverSourcePaths(t *testing.T) {
	t.Parallel()

	// Paths not subject to (or deliberately kept out of) the allowlist:
	// results/websocket is a raw handler that endpoint restrictions don't
	// apply to; POST /reports is only called by the developer-only -seed
	// mode, never during MCP serving.
	exemptPaths := map[string]struct{}{
		"/api/v1/fleet/results/websocket": {},
	}
	exemptMethodPaths := map[string]struct{}{
		"POST /api/v1/fleet/reports": {},
	}

	paramRe := regexp.MustCompile(`:[A-Za-z_]+`)
	covered := make(map[string]struct{}, len(requiredEndpoints))
	coveredMethod := make(map[string]struct{}, len(requiredEndpoints))
	for _, e := range requiredEndpoints {
		covered[normalizeAPIPath(e.route)] = struct{}{}
		coveredMethod[e.method+" "+normalizeAPIPath(e.route)] = struct{}{}
		// The probe must exercise the exact route it claims to verify.
		probePath, _, _ := strings.Cut(e.probe, "?")
		routePattern := regexp.MustCompile("^" + paramRe.ReplaceAllString(regexp.QuoteMeta(e.route), `[^/]+`) + "$")
		if !routePattern.MatchString(probePath) {
			t.Errorf("probe %q does not match route %s", e.probe, e.route)
		}
	}

	litRe := regexp.MustCompile(`"(/api/v1/fleet/[^"]*)"`)
	// Method-aware variant for call sites where the method and path literal
	// share a line, e.g. makeFleetRequest(ctx, "POST", "/api/v1/fleet/...",
	// or with the path wrapped in fmt.Sprintf.
	callRe := regexp.MustCompile(`makeFleetRequest\(ctx, "(GET|POST|PUT|PATCH|DELETE)",\s*(?:fmt\.Sprintf\()?"(/api/v1/fleet/[^"]*)"`)
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		// startup_check.go defines the list itself; its probe literals are
		// not additional API usage.
		if strings.HasSuffix(f, "_test.go") || f == "startup_check.go" {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range litRe.FindAllStringSubmatch(string(src), -1) {
			lit := m[1]
			norm := normalizeAPIPath(lit)
			if _, ok := exemptPaths[norm]; ok {
				continue
			}
			if _, ok := covered[norm]; !ok {
				t.Errorf("%s uses Fleet API path %q which is not listed in requiredEndpoints (startup_check.go); add it there and to the README endpoint list", f, lit)
			}
		}
		for _, m := range callRe.FindAllStringSubmatch(string(src), -1) {
			method, lit := m[1], m[2]
			norm := normalizeAPIPath(lit)
			if _, ok := exemptPaths[norm]; ok {
				continue
			}
			if _, ok := exemptMethodPaths[method+" "+norm]; ok {
				continue
			}
			if _, ok := coveredMethod[method+" "+norm]; !ok {
				t.Errorf("%s calls %s %s which is not listed (with that method) in requiredEndpoints (startup_check.go); add it there and to the README endpoint list", f, method, lit)
			}
		}
	}
}
