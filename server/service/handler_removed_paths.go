package service

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
)

// removedPathTemplates are the paths that Fleet 5 removed, as they appear in pkg/removednames. A request to one gets 410 Gone with the
// message of its table entry, for every method, or only for the methods its entries name in Context. A path moves here from
// deprecatedPathAliases, or from its own route, in the change that stops serving it.
//
// A path variable keeps the pattern of the route it replaces, so the path gets the requests that route got and other requests still get a
// 404; the table entry writes the variable without it. A path that a later template also matches (/mdm/profiles/summary and
// /mdm/profiles/{profile_uuid}) must come first to get its own message.
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

	"/api/_version_/fleet/mdm/commands",
	"/api/_version_/fleet/mdm/apple/commands",
	"/api/_version_/fleet/mdm/commandresults",
	"/api/_version_/fleet/mdm/commands/run",
	"/api/_version_/fleet/mdm/hosts/{id:[0-9]+}/lock",
	"/api/_version_/fleet/mdm/hosts/{id:[0-9]+}/wipe",
	"/api/_version_/fleet/mdm/hosts/{id:[0-9]+}/unenroll",
	"/api/_version_/fleet/mdm/hosts/{id:[0-9]+}/profiles",
	"/api/_version_/fleet/mdm/hosts/{id:[0-9]+}/encryption_key",
	"/api/_version_/fleet/mdm/disk_encryption/summary",
	"/api/_version_/fleet/mdm/apple/filevault/summary",
	"/api/_version_/fleet/mdm/apple/settings",
	"/api/_version_/fleet/mdm/profiles",
	"/api/_version_/fleet/mdm/profiles/summary",
	"/api/_version_/fleet/mdm/profiles/{profile_uuid}",
	"/api/_version_/fleet/mdm/apple/profiles",
	"/api/_version_/fleet/mdm/apple/profiles/summary",
	"/api/_version_/fleet/mdm/apple/profiles/batch",
	"/api/_version_/fleet/mdm/apple/profiles/{profile_id:[0-9]+}",
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
	"/api/_version_/fleet/mdm/bootstrap/{fleet_id:[0-9]+}/metadata",
	"/api/_version_/fleet/mdm/apple/bootstrap/{fleet_id:[0-9]+}/metadata",
	"/api/_version_/fleet/mdm/bootstrap/{fleet_id:[0-9]+}",
	"/api/_version_/fleet/mdm/apple/bootstrap/{fleet_id:[0-9]+}",
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
}

var pathVarPattern = regexp.MustCompile(`\{(\w+):[^}]*\}`)

// removedPathName is the path as its pkg/removednames entries write it, without the patterns of its variables.
func removedPathName(template string) string {
	return pathVarPattern.ReplaceAllString(template, "{$1}")
}

func removedPaths() []endpointer.RemovedPath {
	paths := make([]endpointer.RemovedPath, 0, len(removedPathTemplates))
	for _, template := range removedPathTemplates {
		entries := removedPathEntries(template)
		if len(entries) == 0 {
			panic(fmt.Sprintf("removed path %s has no entry in pkg/removednames", removedPathName(template)))
		}
		paths = append(paths, removedPathFor(template, entries))
	}
	return paths
}

// removedPathEntries finds the entries of a path whatever their Context, which is one per removed method or a single one for every
// method: removednames.Find with an empty context skips an entry pinned to a method.
func removedPathEntries(template string) []removednames.Entry {
	name := removedPathName(template)
	var entries []removednames.Entry
	for _, entry := range removednames.Filter(removednames.KindPath, 0) {
		if entry.Old == name {
			entries = append(entries, entry)
		}
	}
	return entries
}

func removedPathFor(template string, entries []removednames.Entry) endpointer.RemovedPath {
	var methods []string
	for _, entry := range entries {
		if entry.Context == "" {
			methods = nil // removed for every method
			break
		}
		// The entry removed only this method; the path's other methods stay with whatever routes serve them.
		methods = append(methods, entry.Context)
	}
	return endpointer.RemovedPath{Path: template, Methods: methods, Overlap: removedPathOverlaps[template], Message: func(method string) string {
		entry := entries[0]
		for _, e := range entries {
			if e.Context == method {
				entry = e
			}
		}
		// The message names the method that was used. Requests share the closure, so set it on a copy.
		entry.Context = method
		return entry.Message()
	}}
}
