package service

import (
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
)

// deprecatedPathAliases defines deprecated URL path aliases that map old
// (deprecated) paths to their canonical (primary) paths. Each entry causes
// the deprecated path(s) to serve the same handler as the primary path.
//
// These are organized by category:
//   - queries → reports: query CRUD, spec, report data
//   - host queries → reports
//   - live queries → reports: run, run_by_identifiers, run_by_names
//   - abm_tokens → ab_tokens
var deprecatedPathAliases = []eu.DeprecatedPathAlias{

	// ---- queries → reports ----
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/reports/{id:[0-9]+}",
		DeprecatedPaths: []string{"/api/_version_/fleet/queries/{id:[0-9]+}"},
	},
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/reports",
		DeprecatedPaths: []string{"/api/_version_/fleet/queries"},
	},
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/reports/{id:[0-9]+}/report",
		DeprecatedPaths: []string{"/api/_version_/fleet/queries/{id:[0-9]+}/report"},
	},
	{
		Method: "POST", PrimaryPath: "/api/_version_/fleet/reports",
		DeprecatedPaths: []string{"/api/_version_/fleet/queries"},
	},
	{
		Method: "PATCH", PrimaryPath: "/api/_version_/fleet/reports/{id:[0-9]+}",
		DeprecatedPaths: []string{"/api/_version_/fleet/queries/{id:[0-9]+}"},
	},
	{
		Method: "DELETE", PrimaryPath: "/api/_version_/fleet/reports/{name}",
		DeprecatedPaths: []string{"/api/_version_/fleet/queries/{name}"},
	},
	{
		Method: "DELETE", PrimaryPath: "/api/_version_/fleet/reports/id/{id:[0-9]+}",
		DeprecatedPaths: []string{"/api/_version_/fleet/queries/id/{id:[0-9]+}"},
	},
	{
		Method: "POST", PrimaryPath: "/api/_version_/fleet/reports/delete",
		DeprecatedPaths: []string{"/api/_version_/fleet/queries/delete"},
	},
	{
		Method: "POST", PrimaryPath: "/api/_version_/fleet/spec/reports",
		DeprecatedPaths: []string{"/api/_version_/fleet/spec/queries"},
	},
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/spec/reports",
		DeprecatedPaths: []string{"/api/_version_/fleet/spec/queries"},
	},
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/spec/reports/{name}",
		DeprecatedPaths: []string{"/api/_version_/fleet/spec/queries/{name}"},
	},

	// ---- host queries → reports ----
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/hosts/{id:[0-9]+}/reports/{report_id:[0-9]+}",
		DeprecatedPaths: []string{"/api/_version_/fleet/hosts/{id:[0-9]+}/queries/{report_id:[0-9]+}"},
	},

	// ---- live queries → reports ----
	{
		Method: "POST", PrimaryPath: "/api/_version_/fleet/reports/{id:[0-9]+}/run",
		DeprecatedPaths: []string{"/api/_version_/fleet/queries/{id:[0-9]+}/run"},
	},
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/reports/run",
		DeprecatedPaths: []string{"/api/_version_/fleet/queries/run"},
	},
	{
		Method: "POST", PrimaryPath: "/api/_version_/fleet/reports/run",
		DeprecatedPaths: []string{"/api/_version_/fleet/queries/run"},
	},
	{
		Method: "POST", PrimaryPath: "/api/_version_/fleet/reports/run_by_identifiers",
		DeprecatedPaths: []string{"/api/_version_/fleet/queries/run_by_identifiers"},
	},
	{
		Method: "POST", PrimaryPath: "/api/_version_/fleet/reports/run_by_names",
		DeprecatedPaths: []string{"/api/_version_/fleet/queries/run_by_names"},
	},

	// ---- abm_tokens → ab_tokens ----
	{
		Method: "PATCH", PrimaryPath: "/api/_version_/fleet/ab_tokens/{id:[0-9]+}/fleets",
		DeprecatedPaths: []string{"/api/_version_/fleet/abm_tokens/{id:[0-9]+}/fleets"},
	},

	{
		Method: "POST", PrimaryPath: "/api/_version_/fleet/ab_tokens",
		DeprecatedPaths: []string{"/api/_version_/fleet/abm_tokens"},
	},
	{
		Method: "DELETE", PrimaryPath: "/api/_version_/fleet/ab_tokens/{id:[0-9]+}",
		DeprecatedPaths: []string{"/api/_version_/fleet/abm_tokens/{id:[0-9]+}"},
	},
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/ab_tokens",
		DeprecatedPaths: []string{"/api/_version_/fleet/abm_tokens"},
	},
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/ab_tokens/count",
		DeprecatedPaths: []string{"/api/_version_/fleet/abm_tokens/count"},
	},
	{
		Method: "PATCH", PrimaryPath: "/api/_version_/fleet/ab_tokens/{id:[0-9]+}/renew",
		DeprecatedPaths: []string{"/api/_version_/fleet/abm_tokens/{id:[0-9]+}/renew"},
	},
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/mdm/apple/ab_public_key",
		DeprecatedPaths: []string{"/api/_version_/fleet/mdm/apple/abm_public_key"},
	},
}
