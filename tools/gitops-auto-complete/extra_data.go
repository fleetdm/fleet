package main

// Data tables that Fleet's Go structs don't express but GitOps YAML relies on:
// declarative apply notes, path-reference support, and installer-reference keys.

// gitops sends a fully materialized config, so omitting a key normally resets it.
// These hover notes cover the keys that instead keep their value.
const (
	declarativeKeepAlways      = "GitOps: kept unchanged when omitted, null, or empty."
	declarativeKeepUnlessEmpty = "GitOps: kept unchanged when omitted or null; cleared when set to an empty value."
	declarativeKeepOnOmit      = "GitOps: kept unchanged when omitted; cleared when set to null or empty."
)

// declarativeExceptions maps a gitops key (dotted path) to its hover note. Only the
// exceptions to the reset-on-omit default are listed, verified against a live apply.
var declarativeExceptions = map[string]string{
	// Google service-account credentials, preserved so a re-apply need not resend
	// the secret. UI GitOps mode is merged onto the existing config.
	"org_settings.integrations.google_calendar.api_key_json":  declarativeKeepAlways,
	"org_settings.integrations.google_workspace.api_key_json": declarativeKeepAlways,
	"org_settings.gitops": declarativeKeepAlways,

	// host_expiry is the documented exception that isn't reset when omitted, but
	// an explicit empty object still resets it. Applies at team and org level.
	"settings.host_expiry_settings":     declarativeKeepUnlessEmpty,
	"org_settings.host_expiry_settings": declarativeKeepUnlessEmpty,

	// Label host membership: omitting keeps the current members, an explicit empty
	// list clears them. From the docs and label parsing.
	"labels.hosts": declarativeKeepOnOmit,
}

// pathReferenceDefinitions are the $defs that accept a `path` (one external file) in
// place of inline content. pathsReferenceDefinitions additionally accept `paths` (a
// single glob string). Defs whose Go type embeds fleet.BaseItem (reports, scripts,
// configuration_profiles) already get both from reflection, so they aren't listed.
var pathReferenceDefinitions = []string{
	"GitOpsOrgSettings", "GitOpsFleetSettings", "AgentOptions", "ControlsWithTypes",
	"SoftwarePackageSpec",
}

var pathsReferenceDefinitions = []string{
	"GitOpsPolicySpec", "LabelSpec",
}

// requiredKeyRule gates a $def: an item is valid if it has all the keys of any one of
// its validKeyCombinations. So {{"a"},{"b"}} means "a or b" and {{"a","b"}} means
// "a and b". Fleet enforces these at gitops apply time in validation code rather than
// struct tags. Where an item can also be a file reference, path/paths are listed as
// their own combinations.
type requiredKeyRule struct {
	definition           string
	message              string
	validKeyCombinations [][]string
}

var requiredKeys = []requiredKeyRule{
	{
		definition: "SoftwarePackageSpec",
		message:    "A package must set one of: url, hash_sha256, or path.",
		validKeyCombinations: [][]string{
			{"url"},
			{"hash_sha256"},
			{"path"},
		},
	},
	{
		definition:           "TeamSpecAppStoreApp",
		message:              "An app_store_apps entry must set app_store_id.",
		validKeyCombinations: [][]string{{"app_store_id"}},
	},
	{
		definition:           "MaintainedAppSpec",
		message:              "A fleet_maintained_apps entry must set slug.",
		validKeyCombinations: [][]string{{"slug"}},
	},
	{
		definition: "LabelSpec",
		message:    "A label must set name (or reference a file with path/paths).",
		validKeyCombinations: [][]string{
			{"name"},
			{"path"},
			{"paths"},
		},
	},
	{
		definition: "GitOpsPolicySpec",
		message:    "A policy must set name (or reference a file with path/paths).",
		validKeyCombinations: [][]string{
			{"name"},
			{"path"},
			{"paths"},
		},
	},
	{
		definition: "Query",
		message:    "A report must set name and query (or reference a file with path/paths).",
		validKeyCombinations: [][]string{
			{"name", "query"},
			{"path"},
			{"paths"},
		},
	},
	{
		definition:           "YaraRule",
		message:              "A yara_rules entry must set path.",
		validKeyCombinations: [][]string{{"path"}},
	},
}

// strictStringKeys are the keys kept as strict strings, so a wrong-typed value like
// `url: 12345` is caught. `path`/`paths` are excluded so they stay nullable.
var strictStringKeys = map[string][]string{
	"SoftwarePackageSpec": {"url", "hash_sha256"},
	"TeamSpecAppStoreApp": {"app_store_id"},
	"MaintainedAppSpec":   {"slug"},
	"LabelSpec":           {"name"},
	"GitOpsPolicySpec":    {"name"},
	"Query":               {"name", "query"},
}
