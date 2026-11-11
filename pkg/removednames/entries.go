package removednames

// entries is the table of every removed name. Each change that removes names adds its own entries_<feature>.go file and registers the
// entries from an init function, so changes that run in parallel do not edit the same lines.
var entries []Entry

func register(newEntries ...Entry) {
	entries = append(entries, newEntries...)
}

// pathEntry is a renamed URL path. Both arguments are relative to /api/_version_/fleet.
func pathEntry(oldSuffix, newSuffix string) Entry {
	return Entry{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + oldSuffix, New: pathPrefix + newSuffix}
}

func yamlEntry(oldPath, newPath string) Entry {
	return Entry{Kind: KindYAMLKey, Scope: ScopeRequest, Old: oldPath, New: newPath}
}

func serverConfigEntry(oldKey, newKey string) Entry {
	return Entry{Kind: KindConfigKey, Scope: ScopeRequest, Old: oldKey, New: newKey}
}
