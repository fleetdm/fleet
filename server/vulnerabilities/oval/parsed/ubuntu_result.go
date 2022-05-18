package oval_parsed

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
)

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

func (r UbuntuResult) Eval(software []fleet.Software) []fleet.SoftwareVulnerability {
	// Test Id => Matching software
	tResults := make(map[int][]fleet.Software)
	for i, t := range r.PackageTests {
		tResults[i] = t.Eval(software)
	}

	vuln := make([]fleet.SoftwareVulnerability, 0)
	for _, d := range r.Definitions {
		if !d.Eval(tResults) {
			continue
		}

		for _, tId := range d.CollectTestIds() {
			for _, software := range tResults[tId] {
				for _, v := range d.Vulnerabilities {
					vuln = append(vuln, fleet.SoftwareVulnerability{
						ID:    software.ID,
						CPE:   software.GenerateCPE,
						CPEID: software.GeneratedCPEID,
						CVE:   v,
					})
				}
			}
		}
	}

	return vuln
}
