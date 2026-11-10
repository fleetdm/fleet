package service

import (
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
)

// removedPathTemplates are the paths that Fleet 5 removed, as they appear in pkg/removednames. A request to one gets 410 Gone with the
// message of its table entry, for every method, or only for the entry's Context when it names one. A path moves here from
// deprecatedPathAliases, or from its own route, in the change that stops serving it. A path that a later template also matches
// (/mdm/profiles/summary and /mdm/profiles/{profile_uuid}) must come first to get its own message.
var removedPathTemplates = []string{
	"/api/_version_/fleet/spec/teams",
	"/api/_version_/fleet/teams",
	"/api/_version_/fleet/teams/{id}",
	"/api/_version_/fleet/teams/{id}/agent_options",
	"/api/_version_/fleet/teams/{id}/secrets",
	"/api/_version_/fleet/teams/{id}/users",
	"/api/_version_/fleet/mdm/apple/profiles/preassign",
	"/api/_version_/fleet/mdm/apple/profiles/match",
	"/api/_version_/fleet/hosts/{host_id}/configuration_profiles/resend/{profile_uuid}",
	"/api/_version_/fleet/abm",
	"/api/_version_/fleet/abm_tokens",
	"/api/_version_/fleet/abm_tokens/count",
	"/api/_version_/fleet/abm_tokens/{id}",
	"/api/_version_/fleet/abm_tokens/{id}/renew",
	"/api/_version_/fleet/ab_tokens/{id}/teams",
	"/api/_version_/fleet/abm_tokens/{id}/fleets",
	"/api/_version_/fleet/abm_tokens/{id}/teams",
	"/api/_version_/fleet/mdm/apple_bm",
	"/api/_version_/fleet/mdm/apple/abm_public_key",
	"/api/_version_/fleet/scripts/batch/summary/{batch_execution_id}",
	// DELETE /reports/{id:[0-9]+} is registered first, so this only answers non-numeric names.
	"/api/_version_/fleet/reports/{name}",
	"/api/_version_/fleet/queries/{name}",

	"/api/_version_/fleet/mdm/commands",
	"/api/_version_/fleet/mdm/apple/commands",
	"/api/_version_/fleet/mdm/commandresults",
	"/api/_version_/fleet/mdm/commands/run",
	"/api/_version_/fleet/mdm/apple/enqueue",
	"/api/_version_/fleet/mdm/hosts/{id}/lock",
	"/api/_version_/fleet/mdm/hosts/{id}/wipe",
	"/api/_version_/fleet/mdm/hosts/{id}/unenroll",
	"/api/_version_/fleet/mdm/hosts/{id}/profiles",
	"/api/_version_/fleet/mdm/hosts/{id}/encryption_key",
	"/api/_version_/fleet/mdm/disk_encryption/summary",
	"/api/_version_/fleet/mdm/apple/filevault/summary",
	"/api/_version_/fleet/mdm/apple/settings",
	"/api/_version_/fleet/mdm/profiles",
	"/api/_version_/fleet/mdm/profiles/summary",
	"/api/_version_/fleet/mdm/profiles/{profile_uuid}",
	"/api/_version_/fleet/mdm/apple/profiles",
	"/api/_version_/fleet/mdm/apple/profiles/summary",
	"/api/_version_/fleet/mdm/apple/profiles/batch",
	"/api/_version_/fleet/mdm/apple/profiles/{profile_id}",
	"/api/_version_/fleet/mdm/apple",
	"/api/_version_/fleet/mdm/apple/setup",
	"/api/_version_/fleet/mdm/apple/enrollment_profile",
	"/api/_version_/fleet/mdm/manual_enrollment_profile",
	"/api/_version_/fleet/mdm/apple/request_csr",
	"/api/_version_/fleet/mdm/apple/dep/key_pair",
	"/api/_version_/fleet/mdm/bootstrap",
	"/api/_version_/fleet/mdm/apple/bootstrap",
	"/api/_version_/fleet/mdm/bootstrap/summary",
	"/api/_version_/fleet/mdm/apple/bootstrap/summary",
	"/api/_version_/fleet/mdm/bootstrap/{fleet_id}/metadata",
	"/api/_version_/fleet/mdm/apple/bootstrap/{fleet_id}/metadata",
	"/api/_version_/fleet/mdm/bootstrap/{fleet_id}",
	"/api/_version_/fleet/mdm/apple/bootstrap/{fleet_id}",
	"/api/_version_/fleet/mdm/setup/eula",
	"/api/_version_/fleet/mdm/apple/setup/eula",
	"/api/_version_/fleet/mdm/setup/eula/metadata",
	"/api/_version_/fleet/mdm/apple/setup/eula/metadata",
	"/api/_version_/fleet/mdm/setup/eula/{token}",
	"/api/_version_/fleet/mdm/apple/setup/eula/{token}",
}

// removedPathOverlaps has the removed paths that a stdlib ServeMux cannot register next to a live route.
var removedPathOverlaps = map[string]endpointer.RemovedPathOverlap{
	"/api/_version_/fleet/hosts/{host_id}/configuration_profiles/resend/{profile_uuid}": {
		Method:        http.MethodPost,
		RouteTemplate: "/api/_version_/fleet/hosts/{host_id}/configuration_profiles/{resend_segment}/{profile_uuid}",
	},
	"/api/_version_/fleet/scripts/batch/summary/{batch_execution_id}": {
		Method:            http.MethodGet,
		RouteTemplate:     "/api/_version_/fleet/scripts/batch/{summary_segment}/{batch_execution_id}",
		WrongMethodRoutes: map[string]string{"/api/_version_/fleet/scripts/batch/{batch_execution_id}/cancel": http.MethodPost},
	},
}

func removedPaths() []endpointer.RemovedPath {
	paths := make([]endpointer.RemovedPath, 0, len(removedPathTemplates))
	for _, template := range removedPathTemplates {
		entry, ok := removedPathEntry(template)
		if !ok {
			panic(fmt.Sprintf("removed path %s has no entry in pkg/removednames", template))
		}
		paths = append(paths, removedPathFor(template, entry))
	}
	return paths
}

// removedPathEntry finds the entry whatever its Context: removednames.Find with an empty context skips an entry pinned to a method.
func removedPathEntry(template string) (removednames.Entry, bool) {
	for _, entry := range removednames.Filter(removednames.KindPath, 0) {
		if entry.Old == template {
			return entry, true
		}
	}
	return removednames.Entry{}, false
}

func removedPathFor(template string, entry removednames.Entry) endpointer.RemovedPath {
	var methods []string
	if entry.Context != "" {
		// The entry removed only this method; the path's other methods stay with whatever routes serve them.
		methods = []string{entry.Context}
	}
	return endpointer.RemovedPath{Path: template, Methods: methods, Overlap: removedPathOverlaps[template], Message: func(method string) string {
		// The message names the method that was used. Requests share the closure, so set it on a copy.
		withMethod := entry
		withMethod.Context = method
		return withMethod.Message()
	}}
}
