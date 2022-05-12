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

func (r UbuntuResult) Eval(software []fleet.Software) (map[uint][]string, error) {
	// Test Id => Software IDs
	tResults := make(map[int][]uint)
	for i, t := range r.PackageTests {
		tResults[i] = t.Eval(software)
	}

	// Software ID => Vulnerabilities
	vuln := make(map[uint][]string)
	for _, d := range r.Definitions {
		if d.Eval(tResults) {
			for _, tId := range d.CollectTestIds() {
				for _, sId := range tResults[tId] {
					vuln[sId] = append(vuln[sId], d.Vulnerabilities...)
				}
			}
		}
	}

	return vuln, nil
}
