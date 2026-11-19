package removednames

func init() { register(reportEntries...) }

// reportEntries are the delete report by name endpoints that Fleet 5 removed.
var reportEntries = []Entry{
	// Only DELETE: other methods on this path shape are live routes, such as the deprecated GET and PATCH /queries/{id}.
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/queries/{name}", Context: "DELETE", Format: EndpointFormat{}},
}
