package apiendpoints

import (
	_ "embed"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/ghodss/yaml"
	"github.com/gorilla/mux"
)

//go:embed api_endpoints.yml
var apiEndpointsYAML []byte

var apiEndpoints []fleet.APIEndpoint

// GetAPIEndpoints returns a copy of the embedded API endpoints slice.
func GetAPIEndpoints() []fleet.APIEndpoint {
	result := make([]fleet.APIEndpoint, len(apiEndpoints))
	copy(result, apiEndpoints)
	return result
}

func Init(h http.Handler) error {
	r, ok := h.(*mux.Router)
	if !ok {
		return fmt.Errorf("expected *mux.Router, got %T", h)
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
		for _, m := range meths {
			val := fleet.NewAPIEndpointFromTpl(m, tpl)
			registered[val.Fingerprint()] = struct{}{}
		}
		return nil
	})

	loadedApiEndpoints, err := loadAPIEndpoints()
	if err != nil {
		return err
	}

	var missing []string
	for _, e := range loadedApiEndpoints {
		if _, ok := registered[e.Fingerprint()]; !ok {
			missing = append(missing, e.Method+" "+e.Path)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("the following API endpoints are unknown: %v", missing)
	}

	apiEndpoints = loadedApiEndpoints

	return nil
}

func loadAPIEndpoints() ([]fleet.APIEndpoint, error) {
	endpoints := make([]fleet.APIEndpoint, 0)

	if err := yaml.Unmarshal(apiEndpointsYAML, &endpoints); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(endpoints))
	for _, e := range endpoints {
		fp := e.Fingerprint()
		if _, ok := seen[fp]; ok {
			panic(fmt.Errorf("duplicate entry (%s, %s)", e.Method, e.Path))
		}
		seen[fp] = struct{}{}
	}
	return endpoints, nil
}
