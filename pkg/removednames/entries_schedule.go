package removednames

func init() { register(scheduleEntries...) }

// scheduleEntries are the schedule endpoints, deprecated since Fleet 4.35, that Fleet 5 removed. Reports are scheduled with the reports
// endpoints instead.
var scheduleEntries = []Entry{
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/global/schedule", Format: EndpointFormat{}},
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/global/schedule/{id}", Format: EndpointFormat{}},
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/schedule", Format: EndpointFormat{}},
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/schedule/{id}", Format: EndpointFormat{}},
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/fleets/{fleet_id}/schedule", Format: EndpointFormat{}},
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/fleets/{fleet_id}/schedule/{report_id}", Format: EndpointFormat{}},
}
