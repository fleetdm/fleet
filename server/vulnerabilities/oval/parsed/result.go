package oval_parsed

import "github.com/fleetdm/fleet/v4/server/fleet"

type Result interface {
	// Eval evaluates the current OVAL definition againts a list of software, returns all software
	// vulns found.
	Eval([]fleet.Software) []fleet.SoftwareVulnerability
}
