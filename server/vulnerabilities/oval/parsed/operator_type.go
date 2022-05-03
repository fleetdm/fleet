package oval_parsed

// Encodes an 'OperatorEnumeration' see:
// https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/oval-common-schema.html#OperatorEnumeration
type OperatorType int

const (
	And OperatorType = iota
	One
	Or
	Xor
	NotAnd
	NotOne
	NotOr
	NotXor
)

func NewOperatorType(val string) OperatorType {
	switch val {
	case "AND", "and":
		return And
	case "ONE", "one":
		return One
	case "OR", "or":
		return Or
	case "XOR", "xor":
		return Xor
	default:
		return And
	}
}

func (op OperatorType) Negate(neg string) OperatorType {
	if neg == "true" {
		switch op {
		case And:
			return NotAnd
		case One:
			return NotOne
		case Or:
			return NotOr
		case Xor:
			return NotXor
		default:
			return NotAnd
		}
	}
	return op
}

func (op OperatorType) Eval(vals ...bool) bool {
	if len(vals) == 0 {
		return false
	}

	if op == One || op == NotOne {
		var nVals int
		for _, val := range vals {
			if val {
				nVals++
			}
		}
		if op == One {
			return nVals == 1
		}
		return nVals != 1
	}

	r := vals[0]
	for _, val := range vals[1:] {
		switch op {
		case And:
			r = r && val
		case NotAnd:
			r = !r || !val
		case Or:
			r = r || val
		case NotOr:
			r = !r && !val
		case Xor:
			r = (r || val) && !(r && val)
		case NotXor:
			r = !(r || val) || (r && val)
		default:
			r = r && val
		}
	}

	return r
}
