package oval_parsed

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// <rpmverifyfile_test> can target any file installed via RPM - but in the case of OVAL
// definitions for RHEL based systems, they are used to make assertions against the installed OS version.
type RpmVerifyFileTest struct {
	FilePath      string
	State         ObjectInfoState
	StateOperator OperatorType
	ObjectMatch   ObjectMatchType
	StateMatch    StateMatchType
}

func (t *RpmVerifyFileTest) Eval(ver fleet.OSVersion) (bool, error) {
	rEval, err := t.State.EvalOSVersion(ver)
	if err != nil {
		return false, err
	}

	// This test specifies a single (object, state) pair, meaning that the object
	// will either match the state (nState = 1) or not (nState = 0)
	var nState int
	if rEval {
		nState = 1
	}

	return t.StateMatch.Eval(1, nState), nil
}
