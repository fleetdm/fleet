package oval_parsed

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
