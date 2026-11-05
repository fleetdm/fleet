package removednames

func init() { register(abmEntries...) }

// abmEntries are the endpoints removed because, they mention the deprecated name ABM over AB.
var abmEntries = []Entry{
	pathEntry("/abm_tokens", "/ab_tokens"),
	pathEntry("/abm_tokens/count", "/ab_tokens/count"),
	pathEntry("/abm_tokens/{id}", "/ab_tokens/{id}"),
	pathEntry("/abm_tokens/{id}/renew", "/ab_tokens/{id}/renew"),
	pathEntry("/mdm/apple/abm_public_key", "/mdm/apple/ab_public_key"),
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/abm"},
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/mdm/apple_bm"},

	// fleetctl
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl get mdm-apple-bm"},
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl get mdm_apple_bm"},
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl generate mdm-apple-bm"},
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl generate mdm_apple_bm"},
}
