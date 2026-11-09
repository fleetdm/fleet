package removednames

func init() { register(softwareEntries...) }

// softwareEntries are the software version endpoints that Fleet 4 deprecated in favor of /software/versions.
var softwareEntries = []Entry{
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/software", New: pathPrefix + "/software/versions"},
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/software/count", New: pathPrefix + "/software/versions"},
	{
		Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/software/{id}", New: pathPrefix + "/software/versions/{id}",
		Context: "GET",
	},
}
