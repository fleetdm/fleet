package oval_parsed

import "github.com/fleetdm/fleet/v4/server/fleet"

type Result interface {
	// Eval evaluates the current OVAL definition against an OS version and a list of installed software, returns all software
	// vulnerabilities found.
	Eval(fleet.OSVersion, []fleet.Software) ([]fleet.SoftwareVulnerability, error)

	// EvalKernel evaluates the current OVAL definition against a list of installed kernel-image software,
	// returns all kernel-image vulnerabilities found.  Currently only used for Ubuntu.
	EvalKernel([]fleet.Software) ([]fleet.SoftwareVulnerability, error)
}
