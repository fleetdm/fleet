package removednames

func init() { register(teamEntries...) }

// teamEntries are the names removed by renaming teams to fleets. They mirror what the code accepted as deprecated aliases before Fleet 5
// (the renameto struct tags, deprecatedPathAliases, the fleetctl flag aliases, and pkg/spec GitOps key mappings);
// TestEntriesCoverDeprecatedSources fails if any of those sources has a team-related name that is missing here.
var teamEntries = []Entry{
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

	// CSV export column headers, which GET /hosts/report also takes in its columns parameter.
	{Kind: KindCSVColumn, Scope: ScopeRequest | ScopeResponse, Old: "team_id", New: "fleet_id"},
	{Kind: KindCSVColumn, Scope: ScopeRequest | ScopeResponse, Old: "team_name", New: "fleet_name"},

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
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl get t", New: "fleetctl get fleets"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl report"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl user create"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "t", New: "fleet", Context: "fleetctl user create"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl generate-gitops"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl hosts transfer"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl run-script"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl get reports"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl get labels"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl get hosts"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "team", New: "fleet", Context: "fleetctl get software"},
	{Kind: KindFlagValue, Scope: ScopeRequest, Old: "--team no-team", New: "--fleet unassigned", Context: "fleetctl generate-gitops"},
	{Kind: KindFlagValue, Scope: ScopeRequest, Old: "--fleet no-team", New: "--fleet unassigned", Context: "fleetctl generate-gitops"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "policies-team", New: "policies-fleet", Context: "fleetctl apply"},
	{Kind: KindFlag, Scope: ScopeRequest, Old: "delete-other-teams", New: "delete-other-fleets", Context: "fleetctl gitops"},
	{Kind: KindEnvVar, Scope: ScopeRequest, Old: "DELETE_OTHER_TEAMS", New: "DELETE_OTHER_FLEETS", Context: "fleetctl gitops"},

	// GitOps and apply YAML keys, team-related only.
	{Kind: KindYAMLKey, Scope: ScopeRequest, Old: "team_settings", New: "settings"},
	abmEntry("macos_team", "macos_fleet"),
	abmEntry("ios_team", "ios_fleet"),
	abmEntry("ipados_team", "ipados_fleet"),
	abmEntry("byod_team", "byod_fleet"),
	yamlEntry("org_settings.mdm.apple_bm_default_team", "org_settings.mdm.apple_business"),
	yamlEntry("org_settings.mdm.volume_purchasing_program[].teams",
		"org_settings.mdm.volume_purchasing_program[].fleets"),

	// The key that holds a fleet in a fleetctl apply spec of kind fleet.
	yamlEntry("spec.team", "spec.fleet"),

	// The keys in a fleetctl apply spec of kind user_roles, under each user's email.
	yamlEntry("spec.roles.<email>.teams", "spec.roles.<email>.fleets"),
	yamlEntry("spec.roles.<email>.fleets[].team", "spec.roles.<email>.fleets[].fleet"),

	// The GitOps file for hosts that belong to no fleet, and the fleet name in it. The name is compared without regard to case.
	{Kind: KindFileName, Scope: ScopeRequest, Old: "no-team.yml", New: "unassigned.yml"},
	{Kind: KindYAMLValue, Scope: ScopeRequest, Old: "name: No team", New: "name: Unassigned"},

	// SAML attributes.
	{Kind: KindSAMLAttribute, Scope: ScopeRequest, Old: "FLEET_JIT_USER_ROLE_TEAM_<ID>", New: "FLEET_JIT_USER_ROLE_FLEET_<ID>"},

	// Spec kinds accepted by fleetctl apply.
	{Kind: KindSpecKind, Scope: ScopeRequest, Old: "team", New: "fleet"},
}

func abmEntry(oldLeaf, newLeaf string) Entry {
	const prefix = "org_settings.mdm.apple_business[]."
	return Entry{Kind: KindYAMLKey, Scope: ScopeRequest, Old: prefix + oldLeaf, New: prefix + newLeaf}
}
