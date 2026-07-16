package main

// This file holds the data tables that Fleet's Go structs don't express but real
// GitOps YAML relies on: declarative apply notes, file-reference (`path:`) support,
// and required "source" keys. The passes that read them live with the other schema
// post-processing.

// --- declarative notes: how an omitted/null/empty value is applied ---

// The default across gitops is that omitting a key resets it to its default:
// `fleetctl gitops` sends a fully materialized config, so anything left out is
// cleared. These are the hover notes for the keys that deviate from that default.
// The three strings are the only omit/null/empty combinations that actually occur.
const (
	declarativeKeepAlways      = "GitOps: kept unchanged when omitted, null, or empty."
	declarativeKeepUnlessEmpty = "GitOps: kept unchanged when omitted or null; cleared when set to an empty value."
	declarativeKeepOnOmit      = "GitOps: kept unchanged when omitted; cleared when set to null or empty."
)

// declarativeExceptions maps a gitops YAML key (dotted path from the top of a
// gitops file) to its hover note. Only the exceptions to the default are listed.
// Verified against a live server in
// cmd/fleetctl/integrationtest/gitops/declarative_test.go on the
// jk-experiment-gitops-declarative branch.
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
	// list clears them. From the docs and label parsing, not the integration test.
	"labels.hosts": declarativeKeepOnOmit,
}

// --- file references: GitOps accepts `path:`/`paths:` in place of inline content ---

// pathReferenceDefinitions are the $defs where GitOps supports a file reference in
// place of inline content: the top-level section values and the list-item types. We
// add path/paths only there instead of on every object, so completion isn't
// cluttered with path everywhere.
var pathReferenceDefinitions = []string{
	"GitOpsOrgSettings", "GitOpsFleetSettings", "AgentOptions", "Controls", "GitOpsSoftware",
	"GitOpsPolicySpec", "Query", "LabelSpec",
	"SoftwarePackageSpec", "TeamSpecAppStoreApp", "MaintainedAppSpec",
}

// --- required source keys: each software item must point at a source ---

// requiredSource lists, for one software $def, the keys any one of which counts as
// "the item has a source". `path` is always allowed (the item is defined in another
// file). Fleet enforces this in validation code rather than struct tags, so it's
// authored explicitly here.
type requiredSource struct {
	definition string
	keys       []string
	message    string
}

var requiredSources = []requiredSource{
	{"SoftwarePackageSpec", []string{"url", "hash_sha256", "path"}, "A package must set one of: url, hash_sha256, or path."},
	{"TeamSpecAppStoreApp", []string{"app_store_id", "path"}, "An app_store_apps entry must set app_store_id (or path)."},
	{"MaintainedAppSpec", []string{"slug", "path"}, "A fleet_maintained_apps entry must set slug (or path)."},
}

// --- strict-typed source keys ---

// strictStringKeys are the source/identifier keys kept as strict `string` (not null,
// not untyped) so a wrong-typed value like `url: 12345` is caught. They pair with
// requiredSources and are always present-and-a-string when used. Applied after
// relaxNulls, which would otherwise drop their type.
var strictStringKeys = map[string][]string{
	"SoftwarePackageSpec": {"url", "hash_sha256"},
	"TeamSpecAppStoreApp": {"app_store_id"},
	"MaintainedAppSpec":   {"slug"},
}
