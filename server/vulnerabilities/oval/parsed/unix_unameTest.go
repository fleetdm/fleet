package oval_parsed

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/utils"
)

type UnixUnameTest struct {
	States []ObjectStateString
}

func (t UnixUnameTest) Eval(version string) (bool, error) {
	var match bool
	var err error
	for _, s := range t.States {
		op, val := s.unpack()
		switch op {
		case LessThan:
			if utils.Rpmvercmp(version, val) != -1 {
				return false, nil
			}
		case PatternMatch:
			match, err = s.Eval(version)
			if !match {
				return false, nil
			}
		default:
			return false, fmt.Errorf("operation %q not supported for uname test", op)
		}
	}
	if err != nil {
		return false, err
	}

	return true, nil
}
