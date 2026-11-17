package removednames

func init() { register(scriptsDisabledEntries...) }

// scriptsDisabledEntries are the setting that blocked running scripts for the whole instance, removed without a replacement.
var scriptsDisabledEntries = []Entry{
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "scripts_disabled"},
	{Kind: KindYAMLKey, Scope: ScopeRequest, Old: "org_settings.server_settings.scripts_disabled"},
}
