package removednames

func init() { register(policyEntries...) }

// policyEntries are the per-scope policy endpoints, replaced by /policies with fleet_id picking the scope, and the
// policy request fields that automation_type and software_title_id replaced.
var policyEntries = []Entry{
	pathEntry("/global/policies", "/policies"),
	pathEntry("/global/policies/delete", "/policies/delete"),
	pathEntry("/global/policies/{policy_id}", "/policies/{policy_id}"),
	pathEntry("/fleets/{fleet_id}/policies", "/policies"),
	pathEntry("/fleets/{fleet_id}/policies/count", "/policies/count"),
	pathEntry("/fleets/{fleet_id}/policies/delete", "/policies/delete"),
	pathEntry("/fleets/{fleet_id}/policies/{policy_id}", "/policies/{policy_id}"),

	// These names stay valid elsewhere (policy specs, settings integrations, Reset policy automations), so each one
	// names its replacement: an entry without one would reject the bare name in every settings and spec request.
	{Kind: KindJSONKey, Scope: ScopeRequest, Old: "calendar_events_enabled", New: "automation_type"},
	{Kind: KindJSONKey, Scope: ScopeRequest, Old: "conditional_access_enabled", New: "automation_type"},
	{Kind: KindJSONKey, Scope: ScopeRequest, Old: "patch_software_title_id", New: "software_title_id"},
	{Kind: KindJSONKey, Scope: ScopeRequest, Old: "policy_ids", New: "automation_type"},
}
