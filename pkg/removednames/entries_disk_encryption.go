package removednames

func init() { register(diskEncryptionEntries...) }

// diskEncryptionEntries are the macOS-named disk encryption activity types, which a migration rewrote as edited_disk_encryption_settings.
var diskEncryptionEntries = []Entry{
	{
		Kind: KindActivityType, Scope: ScopeRequest | ScopeResponse,
		Old: "enabled_macos_disk_encryption", New: "edited_disk_encryption_settings",
	},
	{
		Kind: KindActivityType, Scope: ScopeRequest | ScopeResponse,
		Old: "disabled_macos_disk_encryption", New: "edited_disk_encryption_settings",
	},
}
