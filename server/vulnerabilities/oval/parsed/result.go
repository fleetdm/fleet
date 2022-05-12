package oval_parsed

import "github.com/fleetdm/fleet/v4/server/fleet"

type Result interface {
	// Eval evaluates the OVAL definition parsed referenced by Result, returning a map of software
	// Id to vulnerabities.
	Eval([]fleet.Software) (map[uint][]string, error)
}
