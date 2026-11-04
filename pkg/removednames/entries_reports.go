package removednames

func init() { register(reportEntries...) }

// reportEntries are the report endpoints that Fleet 5 moved.
var reportEntries = []Entry{
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/reports/{name}", Format: EndpointFormat{}},
}
