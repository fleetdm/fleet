package removednames

func init() {
	register(platformKeyEntries...)
	// GitOps validates the same MDM settings under controls, org_settings.mdm, and a fleet's settings.mdm.
	for _, parent := range []string{"controls", "org_settings.mdm", "settings.mdm"} {
		register(platformYAMLKeyEntries(parent)...)
	}
}

// platformKeyEntries are the settings renamed so the platform comes first, and the host summary count removed with
// them.
var platformKeyEntries = []Entry{
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "enable_recovery_lock_password", New: "macos_recovery_lock_password_enabled"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "require_all_software_macos", New: "macos_require_all_software"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "require_all_software_windows", New: "windows_require_all_software"},
	// Response only, so no request ever gets this message.
	{Kind: KindJSONKey, Scope: ScopeResponse, Old: "all_linux_count"},
}

func platformYAMLKeyEntries(parent string) []Entry {
	entry := func(oldKey, newKey string) Entry {
		return Entry{Kind: KindYAMLKey, Scope: ScopeRequest, Old: parent + "." + oldKey, New: parent + "." + newKey}
	}
	return []Entry{
		entry("enable_recovery_lock_password", "macos_recovery_lock_password_enabled"),
		entry("setup_experience.require_all_software_macos", "setup_experience.macos_require_all_software"),
		entry("setup_experience.require_all_software_windows", "setup_experience.windows_require_all_software"),
	}
}
