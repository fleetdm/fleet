package removednames

func init() { register(abmEntries...) }

// abmEntries are the endpoints removed because, they mention the deprecated name ABM over AB.
var abmEntries = []Entry{
	pathEntry("/abm_tokens", "/ab_tokens"),
	pathEntry("/abm_tokens/{id}", "/ab_tokens/{id}"),
	pathEntry("/abm_tokens/{id}/renew", "/ab_tokens/{id}/renew"),
	pathEntry("/abm_tokens/{id}/fleets", "/ab_tokens/{id}/fleets"),
	pathEntry("/mdm/apple/abm_public_key", "/mdm/apple/ab_public_key"),
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/abm"},
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/mdm/apple_bm"},

	// fleetctl
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl get mdm-apple-bm"},
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl get mdm_apple_bm"},
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl generate mdm-apple-bm"},
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl generate mdm_apple_bm"},

	// server config
	serverConfigEntry("mdm.apple_bm_server_token", "mdm.apple_business_server_token"),
	serverConfigEntry("mdm.apple_bm_server_token_bytes", "mdm.apple_business_server_token_bytes"),
	serverConfigEntry("mdm.apple_bm_cert", "mdm.apple_business_cert"),
	serverConfigEntry("mdm.apple_bm_cert_bytes", "mdm.apple_business_cert_bytes"),
	serverConfigEntry("mdm.apple_bm_key", "mdm.apple_business_key"),
	serverConfigEntry("mdm.apple_bm_key_bytes", "mdm.apple_business_key_bytes"),
}
