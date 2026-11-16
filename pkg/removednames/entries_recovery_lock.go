package removednames

func init() { register(recoveryLockEntries...) }

var recoveryLockEntries = []Entry{
	{
		Kind: KindActivityType, Scope: ScopeRequest | ScopeResponse,
		Old: "viewed_host_recovery_lock_password", New: "read_host_recovery_lock_password",
	},
}
