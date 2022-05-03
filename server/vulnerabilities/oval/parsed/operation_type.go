package oval_parsed

// Encodes an 'OperationEnumeration' see:
// https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/oval-common-schema.html#OperationEnumeration

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
