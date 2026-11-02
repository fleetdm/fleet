package removednames

// entries is the table of removed names. Its team-related content mirrors what the code accepted as deprecated aliases before Fleet 5
// (the renameto struct tags, deprecatedPathAliases, the fleetctl flag aliases, and pkg/spec GitOps key mappings);
// TestEntriesCoverDeprecatedSources fails if any of those sources has a team-related name that is missing here.
// FLEET_JIT_USER_ROLE_TEAM_ (SAML attribute) is not listed: it is handled separately in #53925.
var entries = []Entry{
	// JSON keys in request and response bodies.
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "available_teams", New: "available_fleets"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "byod_team", New: "byod_fleet"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "byod_team_id", New: "byod_fleet_id"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "default_team", New: "default_fleet"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "host_team_id", New: "host_fleet_id"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "ios_team", New: "ios_fleet"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "ios_team_id", New: "ios_fleet_id"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "ipados_team", New: "ipados_fleet"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "ipados_team_id", New: "ipados_fleet_id"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "macos_team", New: "macos_fleet"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "macos_team_id", New: "macos_fleet_id"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "team", New: "fleet"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "team_id", New: "fleet_id"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "team_ids", New: "fleet_ids"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "team_ids_by_name", New: "fleet_ids_by_name"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "team_name", New: "fleet_name"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "teams", New: "fleets"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "apple_bm_default_team", New: "apple_business"},

	// Query parameters.
	{Kind: KindQueryParam, Scope: ScopeRequest, Old: "team_id", New: "fleet_id"},
	{Kind: KindQueryParam, Scope: ScopeRequest, Old: "team_name", New: "fleet_name"},

	// Multipart form fields (software and script uploads).
	{Kind: KindFormField, Scope: ScopeRequest, Old: "team_id", New: "fleet_id"},

	// CSV export column headers.
	{Kind: KindCSVColumn, Scope: ScopeResponse, Old: "team_id", New: "fleet_id"},
	{Kind: KindCSVColumn, Scope: ScopeResponse, Old: "team_name", New: "fleet_name"},

	// Keys inside the stored details of activities.
	{Kind: KindJSONKey, Scope: ScopeActivityDetails, Old: "team_id", New: "fleet_id"},
	{Kind: KindJSONKey, Scope: ScopeActivityDetails, Old: "team_name", New: "fleet_name"},
	{Kind: KindJSONKey, Scope: ScopeActivityDetails, Old: "teams", New: "fleets"},

	// Activity type names.
	{Kind: KindActivityType, Scope: ScopeRequest | ScopeResponse, Old: "created_team", New: "created_fleet"},
	{Kind: KindActivityType, Scope: ScopeRequest | ScopeResponse, Old: "deleted_team", New: "deleted_fleet"},
	{Kind: KindActivityType, Scope: ScopeRequest | ScopeResponse, Old: "applied_spec_team", New: "applied_spec_fleet"},
	{Kind: KindActivityType, Scope: ScopeRequest | ScopeResponse, Old: "changed_user_team_role", New: "changed_user_fleet_role"},
	{Kind: KindActivityType, Scope: ScopeRequest | ScopeResponse, Old: "deleted_user_team_role", New: "deleted_user_fleet_role"},

	// URL paths, relative to /api/_version_/fleet. Path parameters use {name}.
	pathEntry("/ab_tokens/{id}/teams", "/ab_tokens/{id}/fleets"),
	pathEntry("/abm_tokens/{id}/teams", "/ab_tokens/{id}/fleets"),
	pathEntry("/spec/teams", "/spec/fleets"),
	pathEntry("/team/{fleet_id}/policies", "/fleets/{fleet_id}/policies"),
	pathEntry("/team/{fleet_id}/policies/count", "/fleets/{fleet_id}/policies/count"),
	pathEntry("/team/{fleet_id}/policies/delete", "/fleets/{fleet_id}/policies/delete"),
	pathEntry("/team/{fleet_id}/policies/{policy_id}", "/fleets/{fleet_id}/policies/{policy_id}"),
	pathEntry("/team/{fleet_id}/schedule", "/fleets/{fleet_id}/schedule"),
	pathEntry("/team/{fleet_id}/schedule/{report_id}", "/fleets/{fleet_id}/schedule/{report_id}"),
	pathEntry("/teams", "/fleets"),
	pathEntry("/teams/{fleet_id}/policies", "/fleets/{fleet_id}/policies"),
	pathEntry("/teams/{fleet_id}/policies/count", "/fleets/{fleet_id}/policies/count"),
	pathEntry("/teams/{fleet_id}/policies/delete", "/fleets/{fleet_id}/policies/delete"),
	pathEntry("/teams/{fleet_id}/policies/{policy_id}", "/fleets/{fleet_id}/policies/{policy_id}"),
	pathEntry("/teams/{fleet_id}/schedule", "/fleets/{fleet_id}/schedule"),
	pathEntry("/teams/{fleet_id}/schedule/{report_id}", "/fleets/{fleet_id}/schedule/{report_id}"),
	pathEntry("/teams/{fleet_id}/secrets", "/fleets/{fleet_id}/secrets"),
	pathEntry("/teams/{id}", "/fleets/{id}"),
	pathEntry("/teams/{id}/agent_options", "/fleets/{id}/agent_options"),
	pathEntry("/teams/{id}/secrets", "/fleets/{id}/secrets"),
	pathEntry("/teams/{id}/users", "/fleets/{id}/users"),
	pathEntry("/vpp_tokens/{id}/teams", "/vpp_tokens/{id}/fleets"),

	// Values of order_key parameters.
	{Kind: KindSortKey, Scope: ScopeRequest, Old: "team_id", New: "fleet_id"},
	{Kind: KindSortKey, Scope: ScopeRequest, Old: "team_name", New: "fleet_name"},

	// Target type.
	{Kind: KindTargetType, Scope: ScopeRequest, Old: "team", New: "fleet"},

	// fleetctl commands, flags, and environment variables.
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl get teams", New: "fleetctl get fleets"},
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl get team", New: "fleetctl get fleets"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl report"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl user create"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl generate-gitops"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl hosts transfer"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl run-script"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl get reports"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl get labels"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl get hosts"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl get software"},
	{Kind: KindFlagValue, Scope: ScopeRequest, Old: "--team no-team", New: "--fleet unassigned", Context: "fleetctl generate-gitops"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "policies-team", New: "policies-fleet", Context: "fleetctl apply"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "delete-other-teams", New: "delete-other-fleets", Context: "fleetctl gitops"},
	{Kind: KindEnvVar, Scope: ScopeRequest, Old: "DELETE_OTHER_TEAMS", New: "DELETE_OTHER_FLEETS", Context: "fleetctl gitops"},

	// GitOps and apply YAML keys, team-related only.
	{Kind: KindYAMLKey, Scope: ScopeRequest, Old: "team_settings", New: "settings"},
	abmEntry("macos_team", "macos_fleet"),
	abmEntry("ios_team", "ios_fleet"),
	abmEntry("ipados_team", "ipados_fleet"),
	abmEntry("byod_team", "byod_fleet"),
	yamlEntry("org_settings.mdm.volume_purchasing_program[].teams",
		"org_settings.mdm.volume_purchasing_program[].fleets"),

	// Spec kinds accepted by fleetctl apply.
	{Kind: KindSpecKind, Scope: ScopeRequest, Old: "team", New: "fleet"},
}

func pathEntry(oldSuffix, newSuffix string) Entry {
	const prefix = "/api/_version_/fleet"
	return Entry{Kind: KindPath, Scope: ScopeRequest, Old: prefix + oldSuffix, New: prefix + newSuffix}
}

func abmEntry(oldLeaf, newLeaf string) Entry {
	const prefix = "org_settings.mdm.apple_business[]."
	return Entry{Kind: KindYAMLKey, Scope: ScopeRequest, Old: prefix + oldLeaf, New: prefix + newLeaf}
}

func yamlEntry(oldPath, newPath string) Entry {
	return Entry{Kind: KindYAMLKey, Scope: ScopeRequest, Old: oldPath, New: newPath}
}
