package service

import (
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
)

// removedPathTemplates are the paths that Fleet 5 removed, as they appear in pkg/removednames. A request to one gets 410 Gone with the
// message of its table entry. A path moves here from deprecatedPathAliases in the change that stops serving it.
var removedPathTemplates = []string{
	"/api/_version_/fleet/spec/teams",
	"/api/_version_/fleet/mdm/apple/profiles/preassign",
	"/api/_version_/fleet/mdm/apple/profiles/match",
	"/api/_version_/fleet/hosts/{host_id}/configuration_profiles/resend/{profile_uuid}",
	"/api/_version_/fleet/scripts/batch/summary/{batch_execution_id}",
}

// removedPathOverlaps has the removed paths that a stdlib ServeMux cannot register next to a live route.
var removedPathOverlaps = map[string]endpointer.RemovedPathOverlap{
	"/api/_version_/fleet/hosts/{host_id}/configuration_profiles/resend/{profile_uuid}": {
		Methods:       []string{http.MethodPost},
		RouteTemplate: "/api/_version_/fleet/hosts/{host_id}/configuration_profiles/{resend_segment}/{profile_uuid}",
	},
	"/api/_version_/fleet/scripts/batch/summary/{batch_execution_id}": {
		Methods:       []string{http.MethodGet, http.MethodHead, http.MethodPost},
		RouteTemplate: "/api/_version_/fleet/scripts/batch/{summary_segment}/{batch_execution_id}",
		LivePaths: []string{
			"/api/_version_/fleet/scripts/batch/{batch_execution_id}/cancel",
			"/api/_version_/fleet/scripts/batch/{batch_execution_id}/host_results",
			"/api/_version_/fleet/scripts/batch/{batch_execution_id}/host-results",
		},
	},
}

func removedPaths() []endpointer.RemovedPath {
	paths := make([]endpointer.RemovedPath, 0, len(removedPathTemplates))
	for _, template := range removedPathTemplates {
		entry, ok := removednames.Find(removednames.KindPath, template, "")
		if !ok {
			panic(fmt.Sprintf("removed path %s has no entry in pkg/removednames", template))
		}
		paths = append(paths, endpointer.RemovedPath{Path: template, Overlap: removedPathOverlaps[template], Message: func(method string) string {
			// The entry is for every method, and the message names the one that was used. Requests share the closure, so set it on a copy.
			withMethod := entry
			withMethod.Context = method
			return withMethod.Message()
		}})
	}
	return paths
}
