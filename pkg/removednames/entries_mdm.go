package removednames

func init() { register(mdmEntries...) }

// mdmEntries are the deprecated /mdm endpoints. Each is removed for one HTTP method, because some of these paths stay for other
// methods: GET /mdm/bootstrap serves hosts that were sent its URL before the upgrade, and GET /mdm/apple/request_csr is the current APNs
// setup flow.
var mdmEntries = []Entry{
	removedEndpoint("GET", "/mdm/commands"),
	removedEndpoint("GET", "/mdm/apple/commands"),
	removedEndpoint("GET", "/mdm/commandresults"),
	removedEndpoint("POST", "/mdm/commands/run"),
	removedEndpoint("POST", "/mdm/apple/enqueue"),

	removedEndpoint("POST", "/mdm/hosts/{id}/lock"),
	removedEndpoint("POST", "/mdm/hosts/{id}/wipe"),
	removedEndpoint("PATCH", "/mdm/hosts/{id}/unenroll"),
	removedEndpoint("GET", "/mdm/hosts/{id}/profiles"),
	removedEndpoint("GET", "/mdm/hosts/{id}/encryption_key"),

	removedEndpoint("GET", "/mdm/disk_encryption/summary"),
	removedEndpoint("GET", "/mdm/apple/filevault/summary"),
	removedEndpoint("PATCH", "/mdm/apple/settings"),

	removedEndpoint("GET", "/mdm/profiles"),
	removedEndpoint("POST", "/mdm/profiles"),
	removedEndpoint("GET", "/mdm/profiles/summary"),
	removedEndpoint("GET", "/mdm/profiles/{profile_uuid}"),
	removedEndpoint("DELETE", "/mdm/profiles/{profile_uuid}"),
	removedEndpoint("GET", "/mdm/apple/profiles"),
	removedEndpoint("POST", "/mdm/apple/profiles"),
	removedEndpoint("GET", "/mdm/apple/profiles/summary"),
	removedEndpoint("POST", "/mdm/apple/profiles/batch"),
	removedEndpoint("GET", "/mdm/apple/profiles/{profile_id}"),
	removedEndpoint("DELETE", "/mdm/apple/profiles/{profile_id}"),

	removedEndpoint("GET", "/mdm/apple"),
	removedEndpoint("PATCH", "/mdm/apple/setup"),
	removedEndpoint("GET", "/mdm/apple/enrollment_profile"),
	removedEndpoint("POST", "/mdm/apple/enrollment_profile"),
	removedEndpoint("DELETE", "/mdm/apple/enrollment_profile"),
	removedEndpoint("GET", "/mdm/manual_enrollment_profile"),
	removedEndpoint("POST", "/mdm/apple/request_csr"),
	removedEndpoint("POST", "/mdm/apple/dep/key_pair"),

	removedEndpoint("POST", "/mdm/bootstrap"),
	removedEndpoint("POST", "/mdm/apple/bootstrap"),
	removedEndpoint("GET", "/mdm/apple/bootstrap"),
	removedEndpoint("GET", "/mdm/bootstrap/summary"),
	removedEndpoint("GET", "/mdm/apple/bootstrap/summary"),
	removedEndpoint("GET", "/mdm/bootstrap/{fleet_id}/metadata"),
	removedEndpoint("GET", "/mdm/apple/bootstrap/{fleet_id}/metadata"),
	removedEndpoint("DELETE", "/mdm/bootstrap/{fleet_id}"),
	removedEndpoint("DELETE", "/mdm/apple/bootstrap/{fleet_id}"),

	removedEndpoint("POST", "/mdm/setup/eula"),
	removedEndpoint("POST", "/mdm/apple/setup/eula"),
	removedEndpoint("GET", "/mdm/setup/eula/metadata"),
	removedEndpoint("GET", "/mdm/apple/setup/eula/metadata"),
	removedEndpoint("GET", "/mdm/setup/eula/{token}"),
	removedEndpoint("GET", "/mdm/apple/setup/eula/{token}"),
	removedEndpoint("DELETE", "/mdm/setup/eula/{token}"),
	removedEndpoint("DELETE", "/mdm/apple/setup/eula/{token}"),
}

// removedEndpoint is an endpoint removed for one HTTP method. The path is relative to /api/_version_/fleet.
func removedEndpoint(method, path string) Entry {
	return Entry{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + path, Context: method, Format: EndpointFormat{}}
}
