package oval_parsed

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/utils"
)

type ObjectStateSimpleValue string

var complexTypes = []DataType{
	Version,
	Binary,
	FilesetRevision,
	IosVersion,
	Ipv4Address,
	Ipv6Address,
}

// NewObjectStateSimpleValue produces a string with 'datatype', 'op' and 'val' encoded as
// dtype|op|val. See
// https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/oval-definitions-schema.html#EntityStateAnySimpleType
func NewObjectStateSimpleValue(dtype string, op string, val string) ObjectStateSimpleValue {
	return ObjectStateSimpleValue(fmt.Sprintf("%s|%s|%s", dtype, op, val))
}

func (sta ObjectStateSimpleValue) unpack() (DataType, OperationType, string) {
	parts := strings.Split(string(sta), "|")
	return NewDataType(parts[0]), NewOperationType(parts[1]), parts[2]
}

func (sta ObjectStateSimpleValue) Eval(other string) (bool, error) {
	dType, op, val := sta.unpack()

	for _, cType := range complexTypes {
		if dType == cType {
			return false, fmt.Errorf("type %q not supported", dType)
		}
	}

	switch dType {
	case Boolean:
		val1, err := strconv.ParseBool(val)
		if err != nil {
			return false, err
		}
		val2, err := strconv.ParseBool(other)
		if err != nil {
			return false, err
		}

		switch op {
		case Equals:
			return val1 == val2, nil
		case NotEqual:
			return val1 != val2, nil
		default:
			return false, fmt.Errorf("Operation %q not supported for type boolean", op)
		}
	case EvrString:
		evr := NewObjectStateEvrString(op.String(), val)
		return evr.Eval(other, utils.Rpmvercmp, true)
	case Float:
		val1, err := strconv.ParseFloat(val, 32)
		if err != nil {
			return false, err
		}
		val2, err := strconv.ParseFloat(other, 32)
		if err != nil {
			return false, err
		}
		switch op {
		case Equals:
			return val1 == val2, nil
		case NotEqual:
			return val1 != val2, nil
		case GreaterThan:
			return val1 > val2, nil
		case GreaterThanOrEqual:
			return val1 >= val2, nil
		case LessThan:
			return val1 < val2, nil
		case LessThanOrEqual:
			return val1 <= val2, nil
		default:
			return false, fmt.Errorf("Operation %q not supported for type float", op)
		}
	case Int:
		val1, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return false, err
		}
		val2, err := strconv.ParseInt(other, 10, 32)
		if err != nil {
			return false, err
		}
		switch op {
		case Equals:
			return val1 == val2, nil
		case NotEqual:
			return val1 != val2, nil
		case GreaterThan:
			return val1 > val2, nil
		case GreaterThanOrEqual:
			return val1 >= val2, nil
		case LessThan:
			return val1 < val2, nil
		case LessThanOrEqual:
			return val1 <= val2, nil
		default:
			return false, fmt.Errorf("Operation %q not supported for type int", op)
		}
	case String:
		val1 := NewObjectStateString(op.String(), val)
		return val1.Eval(other)
	}
	return false, nil
}
