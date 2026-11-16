package removednames

func init() {
	register(windowsMDMKeyEntries...)
	// GitOps validates the global MDM settings under controls and org_settings.mdm.
	for _, parent := range []string{"controls", "org_settings.mdm"} {
		register(windowsMDMYAMLKeyEntries(parent)...)
	}
}

// windowsMDMKeyEntries are the Windows MDM settings renamed to start with the platform and include "mdm".
var windowsMDMKeyEntries = []Entry{
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "windows_enabled_and_configured", New: "windows_mdm_enabled"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "windows_migration_enabled", New: "windows_mdm_migration_enabled"},
}

func windowsMDMYAMLKeyEntries(parent string) []Entry {
	entry := func(oldKey, newKey string) Entry {
		return Entry{Kind: KindYAMLKey, Scope: ScopeRequest, Old: parent + "." + oldKey, New: parent + "." + newKey}
	}
	return []Entry{
		entry("windows_enabled_and_configured", "windows_mdm_enabled"),
		entry("windows_migration_enabled", "windows_mdm_migration_enabled"),
	}
}
