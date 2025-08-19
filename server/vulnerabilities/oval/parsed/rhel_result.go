package oval_parsed

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type RhelResult struct {
	Definitions        []Definition
	RpmInfoTests       map[int]*RpmInfoTest
	RpmVerifyFileTests map[int]*RpmVerifyFileTest
	UnameTests         map[int]*UnixUnameTest
}

// NewRhelResult is the result of parsing an OVAL file that targets a Rhel based distro.
func NewRhelResult() *RhelResult {
	return &RhelResult{
		RpmInfoTests:       make(map[int]*RpmInfoTest),
		RpmVerifyFileTests: make(map[int]*RpmVerifyFileTest),
		UnameTests:         make(map[int]*UnixUnameTest),
	}
}

func (r *RhelResult) AddUnameTest(id int, tst *UnixUnameTest) {
	r.UnameTests[id] = tst
}

func (r RhelResult) Eval(ver fleet.OSVersion, software []fleet.Software) ([]fleet.SoftwareVulnerability, error) {
	// Rpm Info Test Id => Matching software
	pkgTstResults := make(map[int][]fleet.Software)
	for i, t := range r.RpmInfoTests {
		rEval, err := t.Eval(software)
		if err != nil {
			return nil, err
		}
		pkgTstResults[i] = rEval
	}

	// Evaluate RpmVerifyFileTests, which are used to make assertions against the installed OS
	OSTstResults := make(map[int]bool)
	for i, t := range r.RpmVerifyFileTests {
		rEval, err := t.Eval(ver)
		if err != nil {
			return nil, err
		}
		OSTstResults[i] = rEval
	}

	vuln := make([]fleet.SoftwareVulnerability, 0)
	for _, d := range r.Definitions {
		if !d.Eval(OSTstResults, pkgTstResults) {
			continue
		}

		for _, tId := range d.CollectTestIds() {
			for _, software := range pkgTstResults[tId] {
				for _, v := range d.CveVulnerabilities() {

					vuln = append(vuln, fleet.SoftwareVulnerability{
						SoftwareID: software.ID,
						CVE:        v,
					})
				}
			}
		}
	}

	return vuln, nil
}

func (r RhelResult) EvalKernel(software []fleet.Software) ([]fleet.SoftwareVulnerability, error) {
	// Test Id => Matching software IDs
	uTests := make(map[int][]uint)
	fmt.Printf("r.UnameTests: %v\n", r.UnameTests)
	for _, s := range software {
		if s.Name == "kernel" || s.Name == "kernel-core" {
			// fmt.Printf("s.Name: %v\n", s.Name)
			for i, u := range r.UnameTests {
				isMatch, err := u.Eval(s.Version)
				if err != nil {
					return nil, err
				}

				if isMatch {
					// fmt.Printf("added test for s.ID: %v\n", s.ID)
					uTests[i] = append(uTests[i], s.ID)
				}
			}
		}
	}

	fmt.Printf("uTests: %v\n", uTests)

	vuln := make([]fleet.SoftwareVulnerability, 0)
	for _, d := range r.Definitions {
		swIDs := findMatchingSoftware(*d.Criteria, uTests)
		// fmt.Printf("found swIDs: %v for d: %+v\n", swIDs, d)
		for _, v := range d.CveVulnerabilities() {
			if v == "CVE-2022-2873" {
				fmt.Printf("d: %+v\n", d)
				fmt.Printf("v: %v\n", v)
				fmt.Printf("d.Criteria: %v\n", *d.Criteria)
				fmt.Printf("swIDs: %v\n", swIDs)
			}
			for _, swID := range swIDs {
				fmt.Printf("adding vuln %s to swID: %v\n", v, swID)
				vuln = append(vuln, fleet.SoftwareVulnerability{
					SoftwareID: swID,
					CVE:        v,
				})
			}
		}
	}

	return vuln, nil
}
