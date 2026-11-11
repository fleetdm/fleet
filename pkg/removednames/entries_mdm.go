package removednames

func init() { register(mdmEntries...) }

// mdmEntries are the deprecated /mdm endpoints.
var mdmEntries = []Entry{
	removedPath("/mdm/commands"),
	removedPath("/mdm/apple/commands"),
	removedPath("/mdm/commandresults"),
	removedPath("/mdm/commands/run"),
	removedPath("/mdm/apple/enqueue"),

	removedPath("/mdm/hosts/{id}/lock"),
	removedPath("/mdm/hosts/{id}/wipe"),
	removedPath("/mdm/hosts/{id}/unenroll"),
	removedPath("/mdm/hosts/{id}/profiles"),
	removedPath("/mdm/hosts/{id}/encryption_key"),

	removedPath("/mdm/disk_encryption/summary"),
	removedPath("/mdm/apple/filevault/summary"),
	removedPath("/mdm/apple/settings"),

	removedPath("/mdm/profiles"),
	removedPath("/mdm/profiles/summary"),
	removedPath("/mdm/profiles/{profile_uuid}"),
	removedPath("/mdm/apple/profiles"),
	removedPath("/mdm/apple/profiles/summary"),
	removedPath("/mdm/apple/profiles/batch"),
	removedPath("/mdm/apple/profiles/{profile_id}"),

	removedPath("/mdm/apple"),
	removedPath("/mdm/apple/setup"),
	removedPath("/mdm/apple/enrollment_profile"),
	removedPath("/mdm/manual_enrollment_profile"),
	removedPath("/mdm/apple/dep/key_pair"),

	removedPath("/mdm/apple/bootstrap"),
	removedPath("/mdm/bootstrap/summary"),
	removedPath("/mdm/apple/bootstrap/summary"),
	removedPath("/mdm/bootstrap/{fleet_id}/metadata"),
	removedPath("/mdm/apple/bootstrap/{fleet_id}/metadata"),
	removedPath("/mdm/bootstrap/{fleet_id}"),
	removedPath("/mdm/apple/bootstrap/{fleet_id}"),

	removedPath("/mdm/setup/eula"),
	removedPath("/mdm/apple/setup/eula"),
	removedPath("/mdm/setup/eula/metadata"),
	removedPath("/mdm/apple/setup/eula/metadata"),
	removedPath("/mdm/setup/eula/{token}"),
	removedPath("/mdm/apple/setup/eula/{token}"),

	// GET stays on these paths: GET /mdm/bootstrap serves hosts that were sent its URL before the upgrade, and
	// GET /mdm/apple/request_csr is the current APNs setup flow.
	removedEndpoint("POST", "/mdm/bootstrap"),
	removedEndpoint("POST", "/mdm/apple/request_csr"),
}

// removedPath is a path removed for every method. The path is relative to /api/_version_/fleet.
func removedPath(path string) Entry {
	return Entry{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + path, Format: EndpointFormat{}}
}

// removedEndpoint is a path removed for one method only, because other methods of it are still served. The path is relative to
// /api/_version_/fleet.
func removedEndpoint(method, path string) Entry {
	entry := removedPath(path)
	entry.Context = method
	return entry
}
