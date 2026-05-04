package service

import (
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
)

// deprecatedPathAliases defines deprecated URL path aliases that map old
// (deprecated) paths to their canonical (primary) paths. Each entry causes
// the deprecated path(s) to serve the same handler as the primary path.
//
// These are organized by category:
//   - teams → fleets: team CRUD, secrets, agent_options, users, spec
//   - team/teams → fleets: policies, schedule (both singular and plural deprecated)
//   - queries → reports: query CRUD, spec, report data
//   - host queries → reports
//   - live queries → reports: run, run_by_identifiers, run_by_names
//   - ABM/VPP token teams → fleets
var deprecatedPathAliases = []eu.DeprecatedPathAlias{
	// ---- teams → fleets ----
	{
		Method: "POST", PrimaryPath: "/api/_version_/fleet/spec/fleets",
		DeprecatedPaths: []string{"/api/_version_/fleet/spec/teams"},
	},
	{
		Method: "PATCH", PrimaryPath: "/api/_version_/fleet/fleets/{fleet_id:[0-9]+}/secrets",
		DeprecatedPaths: []string{"/api/_version_/fleet/teams/{fleet_id:[0-9]+}/secrets"},
	},
	{
		Method: "POST", PrimaryPath: "/api/_version_/fleet/fleets",
		DeprecatedPaths: []string{"/api/_version_/fleet/teams"},
	},
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/fleets",
		DeprecatedPaths: []string{"/api/_version_/fleet/teams"},
	},
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/fleets/{id:[0-9]+}",
		DeprecatedPaths: []string{"/api/_version_/fleet/teams/{id:[0-9]+}"},
	},
	{
		Method: "PATCH", PrimaryPath: "/api/_version_/fleet/fleets/{id:[0-9]+}",
		DeprecatedPaths: []string{"/api/_version_/fleet/teams/{id:[0-9]+}"},
	},
	{
		Method: "DELETE", PrimaryPath: "/api/_version_/fleet/fleets/{id:[0-9]+}",
		DeprecatedPaths: []string{"/api/_version_/fleet/teams/{id:[0-9]+}"},
	},
	{
		Method: "POST", PrimaryPath: "/api/_version_/fleet/fleets/{id:[0-9]+}/agent_options",
		DeprecatedPaths: []string{"/api/_version_/fleet/teams/{id:[0-9]+}/agent_options"},
	},
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/fleets/{id:[0-9]+}/users",
		DeprecatedPaths: []string{"/api/_version_/fleet/teams/{id:[0-9]+}/users"},
	},
	{
		Method: "PATCH", PrimaryPath: "/api/_version_/fleet/fleets/{id:[0-9]+}/users",
		DeprecatedPaths: []string{"/api/_version_/fleet/teams/{id:[0-9]+}/users"},
	},
	{
		Method: "DELETE", PrimaryPath: "/api/_version_/fleet/fleets/{id:[0-9]+}/users",
		DeprecatedPaths: []string{"/api/_version_/fleet/teams/{id:[0-9]+}/users"},
	},
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/fleets/{id:[0-9]+}/secrets",
		DeprecatedPaths: []string{"/api/_version_/fleet/teams/{id:[0-9]+}/secrets"},
	},

	// ---- team/teams → fleets (policies) ----
	{
		Method: "POST", PrimaryPath: "/api/_version_/fleet/fleets/{fleet_id}/policies",
		DeprecatedPaths: []string{
			"/api/_version_/fleet/team/{fleet_id}/policies",
			"/api/_version_/fleet/teams/{fleet_id}/policies",
		},
	},
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/fleets/{fleet_id}/policies",
		DeprecatedPaths: []string{
			"/api/_version_/fleet/team/{fleet_id}/policies",
			"/api/_version_/fleet/teams/{fleet_id}/policies",
		},
	},
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/fleets/{fleet_id}/policies/count",
		DeprecatedPaths: []string{
			"/api/_version_/fleet/team/{fleet_id}/policies/count",
			"/api/_version_/fleet/teams/{fleet_id}/policies/count",
		},
	},
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/fleets/{fleet_id}/policies/{policy_id}",
		DeprecatedPaths: []string{
			"/api/_version_/fleet/team/{fleet_id}/policies/{policy_id}",
			"/api/_version_/fleet/teams/{fleet_id}/policies/{policy_id}",
		},
	},
	{
		Method: "POST", PrimaryPath: "/api/_version_/fleet/fleets/{fleet_id}/policies/delete",
		DeprecatedPaths: []string{
			"/api/_version_/fleet/team/{fleet_id}/policies/delete",
			"/api/_version_/fleet/teams/{fleet_id}/policies/delete",
		},
	},
	{
		Method: "PATCH", PrimaryPath: "/api/_version_/fleet/fleets/{fleet_id}/policies/{policy_id}",
		DeprecatedPaths: []string{"/api/_version_/fleet/teams/{fleet_id}/policies/{policy_id}"},
	},

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

	// ---- team/teams → fleets (schedule) ----
	{
		Method: "GET", PrimaryPath: "/api/_version_/fleet/fleets/{fleet_id}/schedule",
		DeprecatedPaths: []string{
			"/api/_version_/fleet/team/{fleet_id}/schedule",
			"/api/_version_/fleet/teams/{fleet_id}/schedule",
		},
	},
	{
		Method: "POST", PrimaryPath: "/api/_version_/fleet/fleets/{fleet_id}/schedule",
		DeprecatedPaths: []string{
			"/api/_version_/fleet/team/{fleet_id}/schedule",
			"/api/_version_/fleet/teams/{fleet_id}/schedule",
		},
	},
	{
		Method: "PATCH", PrimaryPath: "/api/_version_/fleet/fleets/{fleet_id}/schedule/{report_id}",
		DeprecatedPaths: []string{
			"/api/_version_/fleet/team/{fleet_id}/schedule/{report_id}",
			"/api/_version_/fleet/teams/{fleet_id}/schedule/{report_id}",
		},
	},
	{
		Method: "DELETE", PrimaryPath: "/api/_version_/fleet/fleets/{fleet_id}/schedule/{report_id}",
		DeprecatedPaths: []string{
			"/api/_version_/fleet/team/{fleet_id}/schedule/{report_id}",
			"/api/_version_/fleet/teams/{fleet_id}/schedule/{report_id}",
		},
	},

	// ---- ABM/VPP token teams → fleets ----
	{
		Method: "PATCH", PrimaryPath: "/api/_version_/fleet/abm_tokens/{id:[0-9]+}/fleets",
		DeprecatedPaths: []string{"/api/_version_/fleet/abm_tokens/{id:[0-9]+}/teams"},
	},
	{
		Method: "PATCH", PrimaryPath: "/api/_version_/fleet/vpp_tokens/{id}/fleets",
		DeprecatedPaths: []string{"/api/_version_/fleet/vpp_tokens/{id}/teams"},
	},
}
