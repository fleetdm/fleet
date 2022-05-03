package oval_parsed

type UbuntuResult struct {
	Definitions  []Definition          `json:"d"`
	PackageTests map[int]*DpkgInfoTest `json:"p"`
}

func NewResult() *UbuntuResult {
	return &UbuntuResult{
		PackageTests: make(map[int]*DpkgInfoTest),
	}
}

func (r *UbuntuResult) AddDefinition(def Definition) {
	r.Definitions = append(r.Definitions, def)
}

func (r *UbuntuResult) AddPackageTest(id int, tst *DpkgInfoTest) {
	r.PackageTests[id] = tst
}
