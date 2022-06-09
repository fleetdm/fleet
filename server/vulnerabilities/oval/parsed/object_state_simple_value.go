package oval_parsed

import (
	"fmt"
	"strings"
)

type ObjectStateSimpleValue string

// NewObjectStateSimpleValue produces a string with 'datatype', 'op' and 'val' encoded as dtype|op|val
func NewObjectStateSimpleValue(dtype string, op string, evr string) ObjectStateSimpleValue {
	return ObjectStateSimpleValue(fmt.Sprintf("%s|%s|%s", dtype, op, evr))
}

func (sta ObjectStateSimpleValue) unpack() (OperationType, string) {
	parts := strings.Split(string(sta), "|")
	return NewOperationType(parts[0]), parts[1]
}

func (sta ObjectStateSimpleValue) Eval(val string) bool {
	return false
}
