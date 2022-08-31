package oval_parsed

type StateMatchType int

const (
	All StateMatchType = iota
	AtLeastOne
	NoneSatisfy
	OnlyOne
)

// NewStateMatchType encodes an 'CheckEnumeration' into an int.
// See https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/oval-common-schema.html#CheckEnumeration
func NewStateMatchType(val string) StateMatchType {
	switch val {
	case "all":
		return All
	case "at least one":
		return AtLeastOne
	case "none satisfy", "none exist":
		return NoneSatisfy
	case "only one":
		return OnlyOne
	default:
		return All
	}
}

// Eval checks how many of the matching objects (nObjects) satisfy the matching rule by checking the
// number of objects that match the desired state (nState).
func (op StateMatchType) Eval(nObjects int, nState int) bool {
	switch op {
	case All:
		return nObjects == nState
	case AtLeastOne:
		return nState >= 1
	case NoneSatisfy:
		return nState == 0
	case OnlyOne:
		return nState == 1
	default:
		return nObjects == nState
	}
}
