package oval_parsed

import "github.com/fleetdm/fleet/v4/server/fleet"

type Result interface {
	// Eval evaluates the current OVAL definition againts a list of software and an OS Version, returns all software
	// vulnerabilities found.
	Eval(fleet.OSVersion, []fleet.Software) []fleet.SoftwareVulnerability
}
