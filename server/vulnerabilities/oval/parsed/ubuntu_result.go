package oval_parsed

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type UbuntuResult struct {
	Definitions  []Definition
	PackageTests map[int]*DpkgInfoTest
	UnameTests   map[int]*UnixUnameTest
}

// NewUbuntuResult is the result of parsing an OVAL file that targets an Ubuntu distro.
// Used to evaluate whether an Ubuntu host is vulnerable based on one or more package tests.
func NewUbuntuResult() *UbuntuResult {
	return &UbuntuResult{
		PackageTests: make(map[int]*DpkgInfoTest),
		UnameTests:   make(map[int]*UnixUnameTest),
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

func (r *UbuntuResult) AddUnameTest(id int, tst *UnixUnameTest) {
	r.UnameTests[id] = tst
}

func (r UbuntuResult) Eval(ver fleet.OSVersion, software []fleet.Software) ([]fleet.SoftwareVulnerability, error) {
	// Test Id => Matching software
	pkgTstResults := make(map[int][]fleet.Software)
	for i, t := range r.PackageTests {
		r, err := t.Eval(software)
		if err != nil {
			return nil, err
		}
		pkgTstResults[i] = r
	}

	// We don't parse/analyze any tests against the installed OS Ver on Ubuntu hosts
	var OSTstResults map[int]bool

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

var kernelImageRegex = regexp.MustCompile(`^linux-image-(\d+\.\d+\.\d+-\d+)-\w+`)

func (r UbuntuResult) EvalKernel(software []fleet.Software) ([]fleet.SoftwareVulnerability, error) {
	// Test Id => Matching software IDs
	uTests := make(map[int][]uint)
	for _, s := range software {
		if kernelImageRegex.MatchString(s.Name) {
			v, ok := strings.CutPrefix(s.Name, "linux-image-")
			if !ok {
				return nil, fmt.Errorf("linux kernel package %s does not match expected format:", s.Name)
			}

			for i, u := range r.UnameTests {
				isMatch, err := u.Eval(v)
				if err != nil {
					return nil, err
				}

				if isMatch {
					uTests[i] = append(uTests[i], s.ID)
				}
			}
		}
	}

	vuln := make([]fleet.SoftwareVulnerability, 0)
	for _, d := range r.Definitions {
		swIDs := findMatchingSoftware(*d.Criteria, uTests)
		for _, v := range d.CveVulnerabilities() {
			for _, swID := range swIDs {
				vuln = append(vuln, fleet.SoftwareVulnerability{
					SoftwareID: swID,
					CVE:        v,
				})
			}
		}
	}

	return vuln, nil
}
