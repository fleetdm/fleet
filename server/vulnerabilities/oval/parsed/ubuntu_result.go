package oval_parsed

import "github.com/fleetdm/fleet/v4/server/fleet"

type UbuntuResult struct {
	Definitions  []Definition
	PackageTests map[int]*DpkgInfoTest
}

// NewUbuntuResult is the result of parsing an OVAL file that targets an Ubuntu distro.
// Used to evaluate whether an Ubuntu host is vulnerable based on one or more package tests.
func NewUbuntuResult() *UbuntuResult {
	return &UbuntuResult{
		PackageTests: make(map[int]*DpkgInfoTest),
	}
}

// AddDefinition add a definition to the given result.
func (r *UbuntuResult) AddDefinition(def Definition) {
	r.Definitions = append(r.Definitions, def)
}

// AddPackageTest adds a package test to the given result.
func (r *UbuntuResult) AddPackageTest(id int, tst *DpkgInfoTest) {
	r.PackageTests[id] = tst
}

func (r UbuntuResult) Eval(software []fleet.Software) (map[int][]string, error) {
	testResults := make(map[int]bool)
	testPacks := make(map[int][]fleet.Software)

	for i, t := range r.PackageTests {
		r, mPacks := t.Eval(software)
		testResults[i] = r
		testPacks[i] = mPacks
	}

	return nil, nil
}
