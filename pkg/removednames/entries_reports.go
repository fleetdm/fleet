package removednames

func init() { register(reportEntries...) }

// reportEntries are the report endpoints that Fleet 5 moved.
var reportEntries = []Entry{
	// Only DELETE: other methods on this path shape are live routes, such as GET and PATCH /reports/{id}.
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/reports/{name}", Context: "DELETE", Format: EndpointFormat{}},
}
