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

// pathReferenceDefinitions are the $defs where GitOps accepts a file reference
// instead of inline content. path/paths go only here, not on every object.
var pathReferenceDefinitions = []string{
	"GitOpsOrgSettings", "GitOpsFleetSettings", "AgentOptions", "ControlsWithTypes", "GitOpsSoftware",
	"GitOpsPolicySpec", "Query", "LabelSpec",
	"SoftwarePackageSpec", "TeamSpecAppStoreApp", "MaintainedAppSpec",
}

// An installer reference tells Fleet where a software item comes from: a url, hash,
// App Store id, FMA slug, or path. Each software $def needs one, and Fleet enforces
// it in validation code rather than struct tags, so they're listed here.
type installerReference struct {
	definition string
	keys       []string
	message    string
}

var installerReferences = []installerReference{
	{"SoftwarePackageSpec", []string{"url", "hash_sha256", "path"}, "A package must set one of: url, hash_sha256, or path."},
	{"TeamSpecAppStoreApp", []string{"app_store_id", "path"}, "An app_store_apps entry must set app_store_id (or path)."},
	{"MaintainedAppSpec", []string{"slug", "path"}, "A fleet_maintained_apps entry must set slug (or path)."},
}

// strictInstallerReferenceKeys are the installer-reference keys kept as strict
// strings, so a wrong-typed value like `url: 12345` is caught. `path` is excluded
// so it stays nullable.
var strictInstallerReferenceKeys = map[string][]string{
	"SoftwarePackageSpec": {"url", "hash_sha256"},
	"TeamSpecAppStoreApp": {"app_store_id"},
	"MaintainedAppSpec":   {"slug"},
}
