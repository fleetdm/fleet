package removednames

func init() {
	register(reportSettingsEntries...)
	register(reportSettingsYAMLKeyEntries("org_settings.server_settings")...)
}

// reportSettingsEntries are the server settings renamed from "query" to "report".
var reportSettingsEntries = []Entry{
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "live_query_disabled", New: "live_reporting_disabled"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "query_reports_disabled", New: "discard_reports_data"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "query_report_cap", New: "report_cap"},
}

func reportSettingsYAMLKeyEntries(parent string) []Entry {
	entry := func(oldKey, newKey string) Entry {
		return Entry{Kind: KindYAMLKey, Scope: ScopeRequest, Old: parent + "." + oldKey, New: parent + "." + newKey}
	}
	return []Entry{
		entry("live_query_disabled", "live_reporting_disabled"),
		entry("query_reports_disabled", "discard_reports_data"),
		entry("query_report_cap", "report_cap"),
	}
}
