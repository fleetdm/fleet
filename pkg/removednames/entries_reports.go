package removednames

func init() { register(reportEntries...) }

// reportEntries are the delete report by name endpoints that Fleet 5 removed.
var reportEntries = []Entry{
	// Only DELETE: other methods on these path shapes are live routes, such as GET and PATCH /reports/{id}.
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/reports/{name}", Context: "DELETE", Format: EndpointFormat{}},
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/queries/{name}", Context: "DELETE", Format: EndpointFormat{}},
}
