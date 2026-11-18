package removednames

func init() {
	register(appleSettingsEntries...)
	// GitOps validates the same MDM settings under controls, org_settings.mdm, and a fleet's settings.mdm.
	for _, parent := range []string{"controls", "org_settings.mdm", "settings.mdm"} {
		register(yamlEntry(parent+".macos_settings", parent+".apple_settings"))
	}
	// Profile lists are only read under controls.
	for _, platform := range []string{"apple", "windows", "android"} {
		register(yamlEntry(
			"controls."+platform+"_settings.custom_settings",
			"controls."+platform+"_settings.configuration_profiles",
		))
	}
}

var appleSettingsEntries = []Entry{
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "macos_settings", New: "apple_settings"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "custom_settings", New: "configuration_profiles"},
	{Kind: KindQueryParam, Scope: ScopeRequest, Old: "macos_settings", New: "apple_settings"},
}
