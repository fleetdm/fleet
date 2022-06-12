package oval_parsed

import "github.com/fleetdm/fleet/v4/server/fleet"

type RpmInfoTest struct {
	Objects       []string
	States        []ObjectInfoState
	StateOperator OperatorType
	ObjectMatch   ObjectMatchType
	StateMatch    StateMatchType
}

// Eval evaluates the given test againts a host's installed packages.
// If test evaluates to true, returns all Software involved with the test match, otherwise will
// return nil.
func (t *RpmInfoTest) Eval(packages []fleet.Software) []fleet.Software {
	if len(packages) == 0 {
		return nil
	}
	return nil
}
