package removednames

func init() { register(fleetctlDeleteEntries...) }

// fleetctlDeleteEntries are the fleetctl commands removed with fleetctl delete.
var fleetctlDeleteEntries = []Entry{
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl delete"},
}
