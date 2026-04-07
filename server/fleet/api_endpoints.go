package fleet

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// APIEndpoint represents an API endpoint that we can attach permissions to
type APIEndpoint struct {
	Method         string `json:"method" yaml:"method"`
	Path           string `json:"path" yaml:"path"`
	DisplayName    string `json:"display_name" yaml:"display_name"`
	Deprecated     bool   `json:"deprecated" yaml:"deprecated"`
	NormalizedPath string `json:"-"`
}

var validHTTPMethods = map[string]struct{}{
	http.MethodGet:    {},
	http.MethodPost:   {},
	http.MethodPut:    {},
	http.MethodPatch:  {},
	http.MethodDelete: {},
}

// NormalizePathPlaceholders replaces each variable path segment with a
// numbered placeholder. It handles both the colon-prefix style used in YAML
// (e.g. /:id) and the brace style used by gorilla/mux (e.g. /{id:[0-9]+}).
func NormalizePathPlaceholders(path string) string {
	segments := strings.Split(path, "/")
	n := 0
	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") ||
			(strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}")) {
			n++
			segments[i] = fmt.Sprintf(":placeholder_%d", n)
		}
	}
	return strings.Join(segments, "/")
}

// Normalize uppercases the method and computes NormalizedPath.
func (e *APIEndpoint) Normalize() {
	e.Method = strings.ToUpper(e.Method)
	e.NormalizedPath = NormalizePathPlaceholders(e.Path)
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

// Fingerprint returns a string that uniquely identifies an API endpoint
func (e APIEndpoint) Fingerprint() string {
	return fmt.Sprintf("%s:%s", e.Method, e.NormalizedPath)
}

func (e *APIEndpoint) UnmarshalJSON(data []byte) error {
	// Use an alias to prevent infinite recursion.
	type Alias APIEndpoint
	var alias Alias

	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	*e = APIEndpoint(alias)
	e.Normalize()
	return e.validate()
}
