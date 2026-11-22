package removednames

func init() { register(versionEntries...) }

// versionEntries is the public version endpoint, which showed the server version without authentication. It sat outside the Fleet
// API, so Old is the full path.
var versionEntries = []Entry{
	{Kind: KindPath, Scope: ScopeRequest, Old: "/version", Format: EndpointFormat{}},
}
