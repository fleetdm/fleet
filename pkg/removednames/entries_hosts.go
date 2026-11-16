package removednames

func init() { register(hostEntries...) }

// hostEntries are the host list filters that Fleet 5 removed.
var hostEntries = []Entry{
	{Kind: KindQueryParam, Scope: ScopeRequest, Old: "software_id", New: "software_version_id"},
}
