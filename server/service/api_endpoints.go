package service

import (
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

//go:embed api_endpoints.yml
var apiEndpointsYAML []byte

var apiEndpoints = mustParseAPIEndpoints()

// APIEndpoint represents an API endpoint that we can attach permissions to.
type APIEndpoint struct {
	Method     string `yaml:"method"`
	Path       string `yaml:"path"`
	Name       string `yaml:"name"`
	Deprecated bool   `yaml:"deprecated"`
}

var validHTTPMethods = map[string]struct{}{
	http.MethodGet:    {},
	http.MethodPost:   {},
	http.MethodPut:    {},
	http.MethodPatch:  {},
	http.MethodDelete: {},
}

// validate checks that all required fields are present and well-formed.
// It returns an error describing the first violation found.
func (e APIEndpoint) validate() error {
	if e.Name == "" {
		return errors.New("name is required")
	}
	if _, ok := validHTTPMethods[strings.ToUpper(e.Method)]; !ok {
		return fmt.Errorf("invalid HTTP method %q", e.Method)
	}
	if !strings.HasPrefix(e.Path, "/") {
		return fmt.Errorf("path %q must start with '/'", e.Path)
	}
	return nil
}

// mustParseAPIEndpoints parses and validates api_endpoints.yml.
func mustParseAPIEndpoints() []APIEndpoint {
	var routes []APIEndpoint
	if err := yaml.Unmarshal(apiEndpointsYAML, &routes); err != nil {
		panic(fmt.Sprintf("api_endpoints.yml: failed to parse: %v", err))
	}
	for i, r := range routes {
		if err := r.validate(); err != nil {
			panic(fmt.Sprintf("api_endpoints.yml: entry %d: %v", i, err))
		}
		// Normalise method to upper-case so callers don't have to.
		routes[i].Method = strings.ToUpper(r.Method)
	}
	return routes
}

// GetAPIEndpoints returns all routes defined in api_endpoints.yml.
func GetAPIEndpoints() []APIEndpoint {
	return apiEndpoints
}

// versionSegmentRe matches the gorilla/mux version segment that attachFleetAPIRoutes
// inserts in place of /_version_/ (e.g. /{fleetversion:(?:v1|2022-04|latest)}/).
var versionSegmentRe = regexp.MustCompile(`/\{fleetversion:[^}]+\}/`)

// ValidateAPIEndpoints checks that every route declared in api_endpoints.yml is
// registered in h. It returns (true, nil) on success, or (false, <missing routes>)
// when one or more routes are absent.
//
// It panics if h is not a *mux.Router
func ValidateAPIEndpoints(h http.Handler) (bool, []string) {
	r, ok := h.(*mux.Router)
	if !ok {
		panic(fmt.Sprintf("ValidateAPIEndpoints: expected *mux.Router, got %T", h))
	}

	registered := make(map[string]struct{})
	_ = r.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		tpl, err := route.GetPathTemplate()
		if err != nil {
			return nil
		}
		meths, err := route.GetMethods()
		if err != nil || len(meths) == 0 {
			return nil
		}
		normalized := versionSegmentRe.ReplaceAllString(tpl, "/_version_/")
		for _, m := range meths {
			registered[m+":"+normalized] = struct{}{}
		}
		return nil
	})

	var missing []string
	for _, route := range GetAPIEndpoints() {
		key := route.Method + ":" + route.Path
		if _, ok := registered[key]; !ok {
			missing = append(missing, route.Method+" "+route.Path)
		}
	}

	return len(missing) == 0, missing
}
