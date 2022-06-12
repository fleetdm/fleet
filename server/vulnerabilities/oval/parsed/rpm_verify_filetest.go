package oval_parsed

import "github.com/fleetdm/fleet/v4/server/fleet"

// <rpmverifyfile_test> can actually target any file installed via RPM - but in the case of the OVAL
// definitions for RHEL based systems, they are used to make assertions againts the installed OS version.
type RpmVerifyFileTest struct {
	FilePath string
	State    ObjectInfoState
}

func (t *RpmVerifyFileTest) Eval(ver fleet.OSVersion) bool {
	return false
}
