package removednames

func init() { register(packEntries...) }

// packEntries are the 2017 packs features that Fleet 5 removed.
var packEntries = []Entry{
	{Kind: KindSpecKind, Scope: ScopeRequest, Old: "pack"},
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl get packs"},
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl get pack"},
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl get p"},
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl upgrade-packs"},
}
