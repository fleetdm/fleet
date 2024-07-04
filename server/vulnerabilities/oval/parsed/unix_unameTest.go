package oval_parsed

import (
	"fmt"
	"regexp"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/utils"
)

type UnixUnameTest struct {
	States []ObjectStateString
}

// Eval evaluates a kernel version against a UnameTest.  Returns true
// if the kernel version matches the test.  Currently only used for Ubuntu.
func (t UnixUnameTest) Eval(version string) (bool, error) {
	for _, s := range t.States {
		op, val := s.unpack()
		switch op {
		case LessThan:
			if utils.Rpmvercmp(version, val) != -1 {
				return false, nil
			}
		case PatternMatch:
			match, err := regexp.Compile(val)
			if err != nil {
				return false, err
			}
			if !match.MatchString(version) {
				return false, nil
			}
		default:
			return false, fmt.Errorf("operation %q not supported for uname test", op)
		}
	}

	return true, nil
}
