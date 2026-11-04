package removednames

func init() { register(routeEntries...) }

// routeEntries are the endpoints removed because a stdlib ServeMux cannot register them next to another route.
var routeEntries = []Entry{
	{
		Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/hosts/{host_id}/configuration_profiles/resend/{profile_uuid}",
		Format: EndpointFormat{},
	},
}
