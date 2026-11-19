package removednames

func init() { register(puppetEntries...) }

// puppetEntries are the endpoints that only Fleet's Puppet module called, removed with the module.
var puppetEntries = []Entry{
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/mdm/apple/profiles/preassign", Format: EndpointFormat{}},
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/mdm/apple/profiles/match", Format: EndpointFormat{}},
}
