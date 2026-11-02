// Package removednames is the single source of truth for the "team" names that Fleet 5 removed, and for the error message that
// tells users what to use instead. The server, fleetctl, and GitOps reject removed names with that message once they use this package.
package removednames

import (
	"fmt"
	"slices"
)

// GuideURL is the Fleet 5 migration guide that every removal message links to.
const GuideURL = "https://fleetdm.com/learn-more-about/fleet-5-migration"

// Kind is the kind of thing a removed name was.
type Kind string

const (
	KindPath         Kind = "path"          // URL path template, e.g. /api/_version_/fleet/teams
	KindQueryParam   Kind = "query_param"   // URL query parameter
	KindJSONKey      Kind = "json_key"      // key in a JSON request or response body
	KindFormField    Kind = "form_field"    // multipart form field
	KindCSVColumn    Kind = "csv_column"    // column header of a CSV export
	KindSortKey      Kind = "sort_key"      // value of an order_key parameter
	KindTargetType   Kind = "target_type"   // value of a target type
	KindFlag         Kind = "flag"          // fleetctl flag, without the leading dashes
	KindFlagValue    Kind = "flag_value"    // fleetctl flag and value, verbatim
	KindEnvVar       Kind = "env_var"       // environment variable read by fleetctl
	KindCommand      Kind = "command"       // fleetctl command, verbatim
	KindYAMLKey      Kind = "yaml_key"      // GitOps or apply YAML key path, dot notation, [] for array elements
	KindSpecKind     Kind = "spec_kind"     // kind of a fleetctl apply spec
	KindActivityType Kind = "activity_type" // activity type name
)

// Scope says where a removed name appears, so each consumer loads only the entries it needs.
type Scope uint8

const (
	// ScopeRequest is a name a client sends (paths, parameters, request bodies, flags, YAML).
	ScopeRequest Scope = 1 << iota
	// ScopeResponse is a name the server returns (response bodies, CSV columns).
	ScopeResponse
	// ScopeActivityDetails is a key inside the stored details of activities.
	ScopeActivityDetails
)

// Entry is one removed name and its replacement.
type Entry struct {
	Kind  Kind
	Scope Scope
	Old   string
	New   string
	// Context narrows the entry to one fleetctl command (for example "fleetctl get hosts"). Empty means it applies everywhere.
	Context string
}

// Message returns the error message for the removed name, for example
// "team_id was removed in Fleet 5. Use fleet_id instead. See https://fleetdm.com/learn-more-about/fleet-5-migration".
func (e Entry) Message() string {
	return Message(e.display(e.Old), e.display(e.New))
}

func (e Entry) display(name string) string {
	if e.Kind == KindFlag {
		return "--" + name
	}
	return name
}

// Message formats the removal message for an arbitrary old and new name.
func Message(oldName, newName string) string {
	return fmt.Sprintf("%s was removed in Fleet 5. Use %s instead. See %s", oldName, newName, GuideURL)
}

// All returns a copy of every entry.
func All() []Entry {
	return slices.Clone(entries)
}

// Filter returns the entries of the given kind that apply in at least one of the given scopes. A zero scope matches every scope.
func Filter(kind Kind, scope Scope) []Entry {
	var out []Entry
	for _, e := range entries {
		if e.Kind == kind && (scope == 0 || e.Scope&scope != 0) {
			out = append(out, e)
		}
	}
	return out
}

// Find returns the entry for a removed name. An entry with an empty Context matches every context; otherwise the context must match.
func Find(kind Kind, oldName, context string) (Entry, bool) {
	for _, e := range entries {
		if e.Kind == kind && e.Old == oldName && (e.Context == "" || e.Context == context) {
			return e, true
		}
	}
	return Entry{}, false
}
