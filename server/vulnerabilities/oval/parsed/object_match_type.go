package oval_parsed

type ObjectMatchType int

const (
	AllExist ObjectMatchType = iota
	AnyExist
	AtLeastOneExists
	NoneExist
	OnlyOneExists
)

// NewObjectMatchType encodes a 'ExistenceEnumeration' value (used in tests for object assertions) into an int.
// See https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/oval-common-schema.html#ExistenceEnumeration.
func NewObjectMatchType(val string) ObjectMatchType {
	switch val {
	case "all_exist":
		return AllExist
	case "any_exist":
		return AnyExist
	case "at_least_one_exists":
		return AtLeastOneExists
	case "none_exist":
		return NoneExist
	case "only_one_exists":
		return OnlyOneExists
	default:
		return AtLeastOneExists
	}
}

// Eval evalutes the given object match type according to the rules outlined in the OVAL spec.
// See https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/oval-common-schema.html#ExistenceEnumeration.
func (op ObjectMatchType) Eval(matches int, total int) bool {
	switch op {
	case AllExist:
		return matches == total
	case AnyExist:
		return matches >= 0
	case AtLeastOneExists:
		return matches > 0
	case NoneExist:
		return matches == 0
	case OnlyOneExists:
		return matches == 1
	default:
		return matches > 0
	}
}
