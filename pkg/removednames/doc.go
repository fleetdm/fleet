// Package removednames is the single source of truth for the names that Fleet 5 removed (paths, parameters, JSON keys, flags, YAML keys,
// and so on, whether renamed or removed outright), and for the error message that tells users what to do instead. The server, fleetctl,
// and GitOps reject removed names with that message once they use this package.
//
// Every message ends with a link to the Fleet 5 migration guide (GuideURL), so a user who hits a removed name lands on the page that
// explains it. Add the name here and the guide entry in the same change.
//
// # Removing a name
//
// Each change that removes names adds its own file, entries_<feature>.go, and registers its entries from an init function. Separate
// files keep changes that run in parallel from editing the same lines.
//
//	func init() { register(widgetEntries...) }
//
//	var widgetEntries = []Entry{
//		// Renamed: the message says "Use frobnicator_id instead".
//		{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "widget_id", New: "frobnicator_id"},
//		// Removed outright: "widget_mode was removed in Fleet 5." with no replacement.
//		{Kind: KindQueryParam, Scope: ScopeRequest, Old: "widget_mode"},
//		// Removed endpoint: `The "POST /widgets/sync" endpoint is not available anymore. What to use instead: <guide>`.
//		{
//			Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/widgets/sync", Context: "POST",
//			Format: EndpointFormat{},
//		},
//	}
//
// Fields of an [Entry]:
//
//   - Kind says what the name is. It decides how a consumer matches the name and how the message displays it (flags get their leading
//     dashes).
//   - Scope says where the name appears: ScopeRequest for what a client sends, ScopeResponse for what the server returns, and
//     ScopeActivityDetails for keys inside the stored details of activities. Combine them with |. Consumers load only the scopes they
//     need, so a request check does not reject a name that only appears in a response.
//   - Old is the removed name. A KindPath entry is a full path template that starts with /api/_version_/fleet and writes path
//     parameters as {name}. A KindYAMLKey entry is a dotted path where [] marks an array element.
//   - New is the replacement. Leave it empty when there is none.
//   - Context narrows the entry. For a flag, env var, or command it is the fleetctl command (for example "fleetctl get hosts"), so the
//     same flag name can be removed from one command and stay on another. For a path it is an HTTP method, so GET can be removed while
//     POST stays. Empty means it applies everywhere.
//   - Format picks the message. Nil means the default for the kind, which follows the Fleet 5 error spec: [EndpointFormat] for a path
//     ("The "POST /software" endpoint is not available anymore..."), [ParameterFormat] for a query parameter, form field, or JSON key,
//     [KeyFormat] for a YAML key, and [ReplacedFormat] ("x was removed in Fleet 5. Use y instead.") for the kinds the spec does not cover
//     yet. Setting EndpointFormat.ByDefault says "by default" for an endpoint that a setting can turn back on, and it is up to the
//     caller to reject the request only while that setting is off.
//
// To support a message that none of the formats fit, add a type that implements [Format] to format.go and a case for it in
// TestEntriesWellFormed.
//
// # Rejecting a name
//
// Look the name up and return the entry's message, with the HTTP status that suits the caller (the Fleet 5 error spec uses 410 for a
// removed endpoint and 422 for a removed parameter):
//
//	if entry, ok := removednames.Find(removednames.KindQueryParam, name, ""); ok {
//		return errors.New(entry.Message())
//	}
//
// Find returns the first entry of the kind whose Old equals the name and whose Context is empty or equals the context passed in. Pass
// the fleetctl command for flags, env vars, and commands, the HTTP method for paths, and "" otherwise.
//
// Filter returns every entry of a kind that applies in a scope, for a consumer that builds its own lookup once (a router that registers
// a handler for each removed path, or a struct validator that walks request bodies). A zero scope matches every scope. All returns a
// copy of the whole table.
//
// The package only holds names and messages. It does not match requests or choose the status code. A removed path's message shows the
// path without its /api/_version_/fleet prefix, and the method of the entry's Context. A path that is removed for every method has no
// Context, so the handler sets it to the method of the request before it asks for the message.
//
// # Keeping the table honest
//
// The tests in this package check every registered entry: it has a kind and a scope, its old and new names differ, its message links to
// the guide, its format is a known one that suits its kind, and no entry is registered twice. They do not check that the name was
// really removed or that the guide mentions it. TestEntriesCoverDeprecatedSources goes further for teams: it scans the code that
// accepts the old team names as deprecated aliases and fails if one is missing here. Each of its subtests is deleted by the change that
// removes the source it scans.
//
// Entries stay after the old name is gone. They are what produces the message when someone still sends it.
package removednames
