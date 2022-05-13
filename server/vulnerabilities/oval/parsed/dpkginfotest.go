package oval_parsed

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
func (t *DpkgInfoTest) Eval(packages []HostPackage) bool {
	if len(packages) == 0 {
		return false
	}
	no, ns := t.matches(packages)

	oMatches := t.ObjectMatch.Eval(no, len(t.Objects))
	sMatches := t.StateMatch.Eval(no, ns)

	return oMatches && sMatches
}

// Returns:
//  nObjects: How many items in the set defined by the OVAL Object set exists in the system.
//  nStates: How many items in the set defined by the OVAL Object set satisfy the state requirements.
func (t *DpkgInfoTest) matches(packages []HostPackage) (int, int) {
	var nObjects int
	var nState int

	for _, p := range packages {
		for _, o := range t.Objects {
			if p.Name == o {
				nObjects++

				r := make([]bool, 0)
				for _, s := range t.States {
					r = append(r, s.Eval(p.Version, Rpmvercmp))
				}
				if t.StateOperator.Eval(r...) {
					nState++
				}
			}
		}
	}

	return nObjects, nState
}
