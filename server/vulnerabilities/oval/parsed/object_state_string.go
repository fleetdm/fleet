package oval_parsed

import (
	"fmt"
	"strings"
)

type ObjectStateString string

// NewObjectStateString produces a string with 'op' and 'value' encoded as op|value
func NewObjectStateString(op string, val string) ObjectStateString {
	return ObjectStateString(fmt.Sprintf("%s|%s", op, val))
}

func (sta ObjectStateString) unpack() (OperationType, string) {
	parts := strings.Split(string(sta), "|")
	return NewOperationType(parts[0]), parts[1]
}

// Eval evaluates the provided value againts the encoded value in sta according to the encoded op in
// sta.
func (sta ObjectStateString) Eval(val string) bool {
	return false
}
