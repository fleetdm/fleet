package removednames

func init() { register(routeEntries...) }

// routeEntries are the endpoints removed because a stdlib ServeMux cannot register them next to another route.
var routeEntries = []Entry{
	{
		Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/hosts/{host_id}/configuration_profiles/resend/{profile_uuid}",
		New: pathPrefix + "/hosts/{host_id}/configuration_profiles/{profile_uuid}/resend",
	},
	{
		Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/scripts/batch/summary/{batch_execution_id}",
		New: pathPrefix + "/scripts/batch/{batch_execution_id}", Context: "GET",
	},
}
