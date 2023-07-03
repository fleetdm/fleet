package oval_parsed

import "github.com/fleetdm/fleet/v4/server/fleet"

type Result interface {
	// Eval evaluates the current OVAL definition against an OS version and a list of installed software, returns all software
	// vulnerabilities found.
	Eval(fleet.OSVersion, []fleet.Software) ([]fleet.SoftwareVulnerability, error)
}
