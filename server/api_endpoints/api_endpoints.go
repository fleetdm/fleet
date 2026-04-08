package api_endpoints

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/ghodss/yaml"
)

//go:embed api_endpoints.yml
var apiEndpointsYAML []byte

var apiEndpoints = mustGetAPIEndpoints()

func mustGetAPIEndpoints() []fleet.APIEndpoint {
	endpoints := make([]fleet.APIEndpoint, 0)

	if err := yaml.Unmarshal(apiEndpointsYAML, &endpoints); err != nil {
		panic(fmt.Errorf("failed to parse: %w", err))
	}

	seen := make(map[string]struct{}, len(endpoints))
	for _, e := range endpoints {
		fp := e.Fingerprint()
		if _, ok := seen[fp]; ok {
			panic(fmt.Errorf("duplicate entry (%s, %s)", e.Method, e.Path))
		}
		seen[fp] = struct{}{}
	}
	return endpoints
}

func List(opts fleet.ListOptions) ([]fleet.APIEndpoint, *fleet.PaginationMetadata, int, error) {
	query := strings.ToLower(strings.TrimSpace(opts.MatchQuery))

	if query == "" {
		return paginateAPIEndpoints(apiEndpoints, opts)
	}

	normalizedQuery := fleet.NormalizePathPlaceholders(query)
	filtered := make([]fleet.APIEndpoint, 0)
	for _, e := range apiEndpoints {
		if !strings.Contains(strings.ToLower(e.DisplayName), query) &&
			!strings.Contains(strings.ToLower(e.NormalizedPath), normalizedQuery) {
			continue
		}
		filtered = append(filtered, e)
	}

	return paginateAPIEndpoints(filtered, opts)
}

func paginateAPIEndpoints(
	rows []fleet.APIEndpoint,
	opts fleet.ListOptions,
) ([]fleet.APIEndpoint, *fleet.PaginationMetadata, int, error) {
	total := len(rows)
	utotal := uint(total)

	perPage := opts.GetPerPage()
	start := opts.Page * perPage
	if start >= utotal {
		var meta fleet.PaginationMetadata
		meta.HasPreviousResults = opts.Page > 0
		return nil, &meta, total, nil
	}
	end := min(start+perPage, utotal)

	return rows[start:end], &fleet.PaginationMetadata{
		HasPreviousResults: opts.Page > 0,
		HasNextResults:     end < utotal,
	}, total, nil
}
