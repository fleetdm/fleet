package removednames

func init() { register(hostEntries...) }

// hostEntries are the host list filters and host endpoints that Fleet 5 removed.
var hostEntries = []Entry{
	{Kind: KindQueryParam, Scope: ScopeRequest, Old: "software_id", New: "software_version_id"},
	// POST /hosts/:id/query takes the same identifiers.
	removedEndpoint("POST", "/hosts/identifier/{identifier}/query"),
}
