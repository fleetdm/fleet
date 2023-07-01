package oval_parsed

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type RpmInfoTest struct {
	Objects       []string
	States        []ObjectInfoState
	StateOperator OperatorType
	ObjectMatch   ObjectMatchType
	StateMatch    StateMatchType
}

// Eval evaluates the given test against a host's installed packages.
// If test evaluates to true, returns all Software involved with the test match, otherwise will
// return nil.
func (t *RpmInfoTest) Eval(packages []fleet.Software) ([]fleet.Software, error) {
	if len(packages) == 0 {
		return nil, nil
	}

	no, ns, m, err := t.matches(packages)
	if err != nil {
		return nil, err
	}

	oMatches := t.ObjectMatch.Eval(no, len(t.Objects))
	sMatches := t.StateMatch.Eval(no, ns)

	if oMatches && sMatches {
		return m, nil
	}
	return nil, nil
}

// Returns:
//
//	nObjects: How many items in the set defined by the OVAL Object set exists in the system.
//	nStates: How many items in the set defined by the OVAL Object set satisfy the state requirements.
//	Slice with software matching both the object and state criteria.
func (t *RpmInfoTest) matches(software []fleet.Software) (int, int, []fleet.Software, error) {
	var nObjects int
	var nState int
	var matches []fleet.Software

	for _, p := range software {
		for _, o := range t.Objects {
			if p.Name == o {
				nObjects++

				r := make([]bool, 0)
				for _, s := range t.States {
					evalR, err := s.EvalSoftware(p)
					if err != nil {
						return 0, 0, nil, err
					}
					r = append(r, evalR)
				}
				if t.StateOperator.Eval(r...) {
					matches = append(matches, p)
					nState++
				}
			}
		}
	}

	return nObjects, nState, matches, nil
}
