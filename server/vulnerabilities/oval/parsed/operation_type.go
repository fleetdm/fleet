package oval_parsed

type OperationType int

const (
	Equals OperationType = iota
	NotEqual
	CaseInsensitiveEquals
	CaseInsensitiveNotEqual
	GreaterThan
	LessThan
	GreaterThanOrEqual
	LessThanOrEqual
	BitwiseAnd
	BitwiseOr
	PatternMatch
	SubsetOf
	SupersetOf
)

// NewOperationType encodes an 'OperationEnumeration' into an int.
// See https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/oval-common-schema.html#OperationEnumeration
func NewOperationType(val string) OperationType {
	switch val {
	case "equals":
		return Equals
	case "not equal":
		return NotEqual
	case "case insensitive equals":
		return CaseInsensitiveEquals
	case "case insensitive not equal":
		return CaseInsensitiveNotEqual
	case "greater than":
		return GreaterThan
	case "less than":
		return LessThan
	case "greater than or equal":
		return GreaterThanOrEqual
	case "less than or equal":
		return LessThanOrEqual
	case "bitwise and":
		return BitwiseAnd
	case "bitwise or":
		return BitwiseOr
	case "pattern match":
		return PatternMatch
	case "subset of":
		return SubsetOf
	case "superset of":
		return SupersetOf
	default:
		return Equals
	}
}

func (op OperationType) String() string {
	switch op {
	case Equals:
		return "equals"
	case NotEqual:
		return "not equal"
	case CaseInsensitiveEquals:
		return "case insensitive equals"
	case CaseInsensitiveNotEqual:
		return "case insensitive not equal"
	case GreaterThan:
		return "greater than"
	case LessThan:
		return "less than"
	case GreaterThanOrEqual:
		return "greater than or equal"
	case LessThanOrEqual:
		return "less than or equal"
	case BitwiseAnd:
		return "bitwise and"
	case BitwiseOr:
		return "bitwise or"
	case PatternMatch:
		return "pattern match"
	case SubsetOf:
		return "subset of"
	case SupersetOf:
		return "superset of"
	default:
		return "equals"
	}
}
