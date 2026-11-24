package removednames

func init() { register(osVersionEntries...) }

// osVersionEntries are the List operating systems parameters that Fleet 5 renamed.
var osVersionEntries = []Entry{
	// Written with its parameter rather than as the bare value: every KindSortKey value is rejected on all endpoints, and
	// other lists still sort by "version". List operating systems checks it itself.
	{Kind: KindSortKey, Scope: ScopeRequest, Old: "order_key=version", New: "order_key=os_version"},
}
