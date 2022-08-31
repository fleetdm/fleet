package oval_parsed

import (
	"fmt"
	"strings"
)

type ObjectStateEvrString string

// NewObjectStateEvrString produces a string with 'op' and 'evr' encoded as op|evr
// This is just one possible children of <dpkginfo_state>, that said
// all deb package tests are written against evr strings
// see: https://oval.mitre.org/language/version5.10.1/ovaldefinition/documentation/oval-definitions-schema.html#EntityStateEVRStringType
func NewObjectStateEvrString(op string, evr string) ObjectStateEvrString {
	return ObjectStateEvrString(fmt.Sprintf("%s|%s", op, evr))
}

func (sta ObjectStateEvrString) unpack() (OperationType, string) {
	parts := strings.Split(string(sta), "|")
	return NewOperationType(parts[0]), parts[1]
}

// Eval evaluates the evr object state against another evr string using 'cmp'
// for performing the comparison.
func (sta ObjectStateEvrString) Eval(ver string, cmp func(string, string) int, ignoreEpoch bool) (bool, error) {
	op, evr := sta.unpack()

	// TODO: see https://github.com/fleetdm/fleet/issues/6236 -
	// ATM we are not storing the epoch, so we will need to removed it when working with RHEL based
	// distros
	if ignoreEpoch {
		parts := strings.Split(evr, ":")
		if len(parts) > 1 {
			evr = parts[1]
		}
	}

	r := cmp(ver, evr)
	switch op {
	case LessThan:
		return r == -1, nil
	case Equals:
		return r == 0, nil
	case NotEqual:
		return r != 0, nil
	case GreaterThan:
		return r == 1, nil
	case GreaterThanOrEqual:
		return r == 1 || r == 0, nil
	case LessThanOrEqual:
		return r == -1 || r == 0, nil
	}

	return false, fmt.Errorf("can not compute op %q", op)
}
