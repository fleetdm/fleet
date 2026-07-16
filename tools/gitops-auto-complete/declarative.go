package main

import "strings"

// The default across gitops is that omitting a key resets it to its default:
// `fleetctl gitops` sends a fully materialized config, so anything left out is
// cleared. These are the hover notes for the keys that deviate from that default.
// The three strings are the only omit/null/empty combinations that actually occur.
const (
	declKeepAlways      = "GitOps: kept when omitted, null, or empty."
	declKeepUnlessEmpty = "GitOps: kept when omitted or null; cleared when set to an empty value."
	declKeepOnOmit      = "GitOps: kept when omitted; cleared when set to null or empty."
)

// declarativeExceptions maps a gitops YAML key (dotted path from the top of a
// gitops file) to its hover note. Only the exceptions to the default are listed.
// Verified against a live server in
// cmd/fleetctl/integrationtest/gitops/declarative_test.go on the
// jk-experiment-gitops-declarative branch.
var declarativeExceptions = map[string]string{
	// Google service-account credentials, preserved so a re-apply need not resend
	// the secret. UI GitOps mode is merged onto the existing config.
	"org_settings.integrations.google_calendar.api_key_json":  declKeepAlways,
	"org_settings.integrations.google_workspace.api_key_json": declKeepAlways,
	"org_settings.gitops":                                     declKeepAlways,

	// host_expiry is the documented exception that isn't reset when omitted, but
	// an explicit empty object still resets it. Applies at team and org level.
	"settings.host_expiry_settings":     declKeepUnlessEmpty,
	"org_settings.host_expiry_settings": declKeepUnlessEmpty,

	// Label host membership: omitting keeps the current members, an explicit
	// (empty) list clears them. From docs + spec.Label.UnmarshalJSON, not the test.
	"labels.hosts": declKeepOnOmit,
}

// addDeclarativeNotes appends each declarativeExceptions note to the matching
// key's description, walking the schema by dotted path. Array `items` don't add a
// path segment, so an array field's sub-key is addressed as `array.field`. $refs
// are resolved to descend, but the note is attached at the property node itself
// (so shared $defs aren't affected). The walk only follows paths that can still
// reach an exception key, which also keeps it from looping on cyclic $refs.
func addDeclarativeNotes(root map[string]any) {
	defs, _ := root["$defs"].(map[string]any)

	resolve := func(node map[string]any) map[string]any {
		seen := map[string]bool{}
		for {
			ref, ok := node["$ref"].(string)
			if !ok {
				return node
			}
			name := strings.TrimPrefix(ref, "#/$defs/")
			if seen[name] {
				return node
			}
			seen[name] = true
			def, ok := defs[name].(map[string]any)
			if !ok {
				return node
			}
			node = def
		}
	}

	// hasDeeper reports whether some exception key lives below path, i.e. we still
	// need to descend. An exact match alone is not a reason to descend (that node
	// is a leaf for our purposes), which stops array leaves like labels.hosts from
	// being annotated a second time on their items node.
	hasDeeper := func(path string) bool {
		for key := range declarativeExceptions {
			if strings.HasPrefix(key, path+".") {
				return true
			}
		}
		return false
	}

	var walk func(node map[string]any, path string)
	walk = func(node map[string]any, path string) {
		if note, ok := declarativeExceptions[path]; ok {
			if d, ok := node["description"].(string); ok && d != "" {
				node["description"] = d + "\n\n" + note
			} else {
				node["description"] = note
			}
		}
		target := resolve(node)
		if props, ok := target["properties"].(map[string]any); ok {
			for key, child := range props {
				cm, ok := child.(map[string]any)
				if !ok {
					continue
				}
				cp := key
				if path != "" {
					cp = path + "." + key
				}
				if _, exact := declarativeExceptions[cp]; exact || hasDeeper(cp) {
					walk(cm, cp)
				}
			}
		}
		if items, ok := target["items"].(map[string]any); ok && hasDeeper(path) {
			walk(items, path)
		}
	}

	walk(root, "")
}
