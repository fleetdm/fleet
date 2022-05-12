package oval_parsed

import "github.com/fleetdm/fleet/v4/server/fleet"

// DpkgInfoTest encapsulates a Dpkg info test.
// see https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/linux-definitions-schema.html#dpkginfo_test
type DpkgInfoTest struct {
	Objects       []string
	States        []ObjectStateEvrString
	StateOperator OperatorType
	ObjectMatch   ObjectMatchType
	StateMatch    StateMatchType
}

// Eval evaluates the given dpkg info test againts a host's installed packages.
// If test evaluates to true, returns all package IDs involved with the test match, otherwise will
// return nil.
func (t *DpkgInfoTest) Eval(packages []fleet.Software) []uint {
	if len(packages) == 0 {
		return nil
	}
	no, ns, mPacks := t.matches(packages)

	oMatches := t.ObjectMatch.Eval(no, len(t.Objects))
	sMatches := t.StateMatch.Eval(no, ns)

	if oMatches && sMatches {
		return mPacks
	}
	return nil
}

// Returns:
//  nObjects: How many items in the set defined by the OVAL Object set exists in the system.
//  nStates: How many items in the set defined by the OVAL Object set satisfy the state requirements.
//  Slice with software matching both the object and state criteria.
func (t *DpkgInfoTest) matches(packages []fleet.Software) (int, int, []uint) {
	var nObjects int
	var nState int
	var mPackages []uint

	for _, p := range packages {
		for _, o := range t.Objects {
			if p.Name == o {
				nObjects++

				r := make([]bool, 0)
				for _, s := range t.States {
					r = append(r, s.Eval(p.Version, Rpmvercmp))
				}
				if t.StateOperator.Eval(r...) {
					mPackages = append(mPackages, p.ID)
					nState++
				}
			}
		}
	}

	return nObjects, nState, mPackages
}
