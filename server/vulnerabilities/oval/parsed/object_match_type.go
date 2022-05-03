package oval_parsed

// Encodes a 'ExistenceEnumeration' value (used in tests for object assertions) into an int, see:
// https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/oval-common-schema.html#ExistenceEnumeration
type ObjectMatchType int

const (
	AllExist ObjectMatchType = iota
	AnyExist
	AtLeastOneExists
	NoneExist
	OnlyOneExists
)

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
