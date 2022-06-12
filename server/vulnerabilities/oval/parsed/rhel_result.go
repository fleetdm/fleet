package oval_parsed

import "github.com/fleetdm/fleet/v4/server/fleet"

type RhelResult struct {
	Definitions        []Definition
	RpmInfoTests       map[int]*RpmInfoTest
	RpmVerifyFileTests map[int]*RpmVerifyFileTest
}

// NewRhelResult is the result of parsing an OVAL file that targets a Rhel based distro.
func NewRhelResult() *RhelResult {
	return &RhelResult{
		RpmInfoTests:       make(map[int]*RpmInfoTest),
		RpmVerifyFileTests: make(map[int]*RpmVerifyFileTest),
	}
}

func (r RhelResult) Eval(ver fleet.OSVersion, software []fleet.Software) []fleet.SoftwareVulnerability {
	// Rpm Info Test Id => Matching software
	pkgTstResults := make(map[int][]fleet.Software)
	for i, t := range r.RpmInfoTests {
		pkgTstResults[i] = t.Eval(software)
	}

	// Evaluate RpmVerifyFileTests, which are used to make assertions against the installed OS
	OSTstResults := make(map[int]bool)
	for i, t := range r.RpmVerifyFileTests {
		OSTstResults[i] = t.Eval(ver)
	}

	vuln := make([]fleet.SoftwareVulnerability, 0)
	for _, d := range r.Definitions {
		if !d.Eval(OSTstResults, pkgTstResults) {
			continue
		}

		for _, tId := range d.CollectTestIds() {
			for _, software := range pkgTstResults[tId] {
				for _, v := range d.Vulnerabilities {
					vuln = append(vuln, fleet.SoftwareVulnerability{
						SoftwareID: software.ID,
						CPEID:      software.GeneratedCPEID,
						CVE:        v,
					})
				}
			}
		}
	}

	return vuln
}
