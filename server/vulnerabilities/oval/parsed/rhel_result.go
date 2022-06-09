package oval_parsed

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
