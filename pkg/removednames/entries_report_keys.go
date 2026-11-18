package removednames

func init() { register(reportKeyEntries...) }

// reportKeyEntries are the deprecated names that responses carried next to their replacements.
var reportKeyEntries = []Entry{
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "query_id", New: "report_id"},
	{Kind: KindJSONKey, Scope: ScopeRequest, Old: "query_ids", New: "report_ids"},
	{Kind: KindJSONKey, Scope: ScopeResponse, Old: "queries", New: "reports"},
	{Kind: KindJSONKey, Scope: ScopeResponse, Old: "inherited_query_count", New: "inherited_report_count"},
	{Kind: KindJSONKey, Scope: ScopeResponse, Old: "live_query_results", New: "live_report_results"},
	{Kind: KindJSONKey, Scope: ScopeResponse, Old: "mia_count", New: "missing_30_days_count"},
	// Live report result rows repeated the host object's hostname under this key.
	{Kind: KindJSONKey, Scope: ScopeResponse, Old: "host_hostname"},

	// Keys inside the stored details of activities.
	{Kind: KindJSONKey, Scope: ScopeActivityDetails, Old: "query_id", New: "report_id"},
	{Kind: KindJSONKey, Scope: ScopeActivityDetails, Old: "query_ids", New: "report_ids"},
	{Kind: KindJSONKey, Scope: ScopeActivityDetails, Old: "query_name", New: "report_name"},
	{Kind: KindJSONKey, Scope: ScopeActivityDetails, Old: "query_sql", New: "query"},
}
