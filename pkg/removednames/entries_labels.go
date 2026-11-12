package removednames

func init() { register(labelEntries...) }

// labelEntries are removed because fleets can have labels with the same name, so a name doesn't identify one label.
var labelEntries = []Entry{
	{
		Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/labels/{name}", Context: "DELETE",
		New: pathPrefix + "/labels/id/{id}", Format: EndpointFormat{},
	},
}
