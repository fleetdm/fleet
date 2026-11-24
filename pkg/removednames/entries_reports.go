package removednames

func init() { register(reportEntries...) }

// reportEntries are the /queries endpoints, renamed to /reports, and the deprecated report endpoints that Fleet 5 removed.
var reportEntries = []Entry{
	// Also covers DELETE /queries/{name}, which has the same shape.
	pathEntry("/queries/{id}", "/reports/{id}"),
	pathEntry("/queries", "/reports"),
	// Pinned to the one method each served, because on every method these would be ambiguous with each other or with other host paths.
	pathEntryFor("GET", "/queries/{id}/report", "/reports/{id}/report"),
	pathEntryFor("DELETE", "/queries/id/{id}", "/reports/{id}"),
	pathEntryFor("DELETE", "/reports/id/{id}", "/reports/{id}"),
	pathEntryFor("DELETE", "/reports/{name}", "/reports/{id}"),
	pathEntryFor("POST", "/queries/{id}/run", "/reports/{id}/run"),
	pathEntryFor("GET", "/hosts/{id}/queries", "/hosts/{id}/reports"),
	pathEntryFor("GET", "/hosts/{id}/queries/{report_id}", "/hosts/{id}/reports/{report_id}"),
	pathEntry("/queries/delete", "/reports/delete"),
	pathEntry("/queries/run", "/reports/run"),
	pathEntry("/queries/run_by_identifiers", "/reports/run_by_identifiers"),
	pathEntry("/queries/run_by_names", "/reports/run_by_identifiers"),
	pathEntry("/reports/run_by_names", "/reports/run_by_identifiers"),
	pathEntry("/spec/queries", "/spec/reports"),
	pathEntry("/spec/queries/{name}", "/spec/reports/{name}"),
}

func pathEntryFor(method, oldSuffix, newSuffix string) Entry {
	e := pathEntry(oldSuffix, newSuffix)
	e.Context = method
	return e
}
