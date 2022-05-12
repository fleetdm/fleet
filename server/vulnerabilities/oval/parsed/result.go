package oval_parsed

import "github.com/fleetdm/fleet/v4/server/fleet"

type Result interface {
	Eval([]fleet.Software) (map[int][]string, error)
}
