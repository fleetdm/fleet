package fleet

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// APIEndpoint represents an API endpoint that we can attach permissions to
type APIEndpoint struct {
	Method         string `json:"method" yaml:"method"`
	Path           string `json:"path" yaml:"path"`
	NormalizedPath string `json:"-"`
	DisplayName    string `json:"display_name" yaml:"display_name"`
	Deprecated     bool   `json:"deprecated" yaml:"deprecated"`
}

// AuthzType implements authz.AuthzTyper.
func (e *APIEndpoint) AuthzType() string {
	return "api_endpoint"
}

var validHTTPMethods = map[string]struct{}{
	http.MethodGet:    {},
	http.MethodPost:   {},
	http.MethodPut:    {},
	http.MethodPatch:  {},
	http.MethodDelete: {},
}

// versionSegmentRe matches the gorilla/mux version segment that attachFleetAPIRoutes
// inserts in place of /_version_/ (e.g. /{fleetversion:(?:v1|2022-04|latest)}/).
var versionSegmentRe = regexp.MustCompile(`/\{fleetversion:[^}]+\}/`)

// NewAPIEndpointFromTpl creates a new APIEndpoint from the provided params.
// tpl is meant to be a route template as usually defined in the mux router,
// or a path using the /_version_/ placeholder convention.
func NewAPIEndpointFromTpl(method string, tpl string) APIEndpoint {
	path := versionSegmentRe.ReplaceAllString(tpl, "/v1/")
	path = strings.ReplaceAll(path, "/_version_/", "/v1/")
	val := APIEndpoint{
		Method: method,
		Path:   path,
	}
	val.normalize()
	return val
}

// normalize method and path properties
func (e *APIEndpoint) normalize() {
	e.Method = strings.ToUpper(e.Method)

	segments := strings.Split(e.Path, "/")
	n := 0
	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") ||
			(strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}")) {
			n++
			segments[i] = fmt.Sprintf(":placeholder_%d", n)
		}
	}
	e.NormalizedPath = strings.ToLower(strings.Join(segments, "/"))
}

// Fingerprint return a string that uniquely identifies
// the APIEndpoint
func (e APIEndpoint) Fingerprint() string {
	return "|" + e.Method + "|" + e.NormalizedPath + "|"
}

func (e APIEndpoint) validate() error {
	if strings.TrimSpace(e.DisplayName) == "" {
		return errors.New("display_name is required")
	}
	if _, ok := validHTTPMethods[e.Method]; !ok {
		return fmt.Errorf("invalid HTTP method %q", e.Method)
	}
	if strings.TrimSpace(e.Path) == "" {
		return errors.New("path is required")
	}
	return nil
}

func (e *APIEndpoint) UnmarshalJSON(data []byte) error {
	// Use an alias to prevent infinite recursion.
	type Alias APIEndpoint
	var alias Alias

	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	*e = APIEndpoint(alias)
	e.normalize()
	return e.validate()
}
