package oval_parsed

import (
	"fmt"
	"regexp"
	"strings"
)

type ObjectStateString string

// NewObjectStateString produces a string with 'op' and 'value' encoded as op|value
func NewObjectStateString(op string, val string) ObjectStateString {
	return ObjectStateString(fmt.Sprintf("%s|%s", op, val))
}

func (sta ObjectStateString) unpack() (OperationType, string) {
	parts := strings.SplitN(string(sta), "|", 2)
	return NewOperationType(parts[0]), parts[1]
}

// Eval evaluates the provided value against the encoded value in sta according to the encoded
// operation.
func (sta ObjectStateString) Eval(other string) (bool, error) {
	op, val := sta.unpack()

	switch op {
	case Equals:
		return val == other, nil
	case NotEqual:
		return val != other, nil
	case CaseInsensitiveEquals:
		return strings.EqualFold(val, other), nil
	case CaseInsensitiveNotEqual:
		return !strings.EqualFold(val, other), nil
	case PatternMatch:
		r, err := regexp.Compile(val)
		if err != nil {
			return false, err
		}
		return r.MatchString(other), nil
	}

	return false, fmt.Errorf("can not compute op %q", op)
}
