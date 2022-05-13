package oval_parsed

import "github.com/fleetdm/fleet/v4/server/fleet"

type Result interface {
	// Eval evaluates the current OVAL definition againts a list of software, returns a map of
	// software ids to vulnerabilities identifiers (mostly CVEs).
	Eval([]fleet.Software) map[uint][]string
}
