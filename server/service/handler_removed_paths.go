package service

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
)

// removedPathTemplates are the paths that Fleet 5 removed, as they appear in pkg/removednames. A request to one gets 410 Gone with the
// message of its table entry. A path moves here from deprecatedPathAliases in the change that stops serving it.
var removedPathTemplates = []string{
	"/api/_version_/fleet/spec/teams",
	"/api/_version_/fleet/mdm/apple/profiles/preassign",
	"/api/_version_/fleet/mdm/apple/profiles/match",
	// Same shape as the live /reports/{id:[0-9]+} routes, which are registered first, so this only answers non-numeric values.
	"/api/_version_/fleet/reports/{name}",
}

func removedPaths() []endpointer.RemovedPath {
	paths := make([]endpointer.RemovedPath, 0, len(removedPathTemplates))
	for _, template := range removedPathTemplates {
		entry, ok := removednames.Find(removednames.KindPath, template, "")
		if !ok {
			panic(fmt.Sprintf("removed path %s has no entry in pkg/removednames", template))
		}
		paths = append(paths, endpointer.RemovedPath{Path: template, Message: func(method string) string {
			// The entry is for every method, and the message names the one that was used. Requests share the closure, so set it on a copy.
			withMethod := entry
			withMethod.Context = method
			return withMethod.Message()
		}})
	}
	return paths
}
