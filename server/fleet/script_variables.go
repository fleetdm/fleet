package fleet

import (
	"fmt"
	"slices"

	"github.com/fleetdm/fleet/v4/server/variables"
)

// FleetVarsSupportedInScripts is the allow-list of built-in variables that can
// be used in script contents. They are resolved per host when fleetd fetches
// the script.
var FleetVarsSupportedInScripts = []FleetVarName{
	FleetVarHostEndUserIDPUsername,
	FleetVarHostEndUserIDPUsernameLocalPart,
	FleetVarHostEndUserIDPFullname,
	FleetVarHostEndUserIDPGroups,
	FleetVarHostEndUserIDPDepartment,
	FleetVarHostHardwareSerial,
	FleetVarHostUUID,
	FleetVarHostPlatform,
}

// FindUnsupportedScriptFleetVar returns the name of a $FLEET_VAR_* reference,
// from the names returned by variables.Find, that is not supported in
// scripts, or "" if all are supported.
func FindUnsupportedScriptFleetVar(fleetVars []string) string {
	for _, v := range fleetVars {
		if !slices.Contains(FleetVarsSupportedInScripts, FleetVarName(v)) {
			return v
		}
	}
	return ""
}

// FleetVarsInPythonMsg is returned when a Python script uses Fleet variables.
const FleetVarsInPythonMsg = "Fleet variables are not supported in Python scripts."

// ScriptFleetVarsUnsupportedByInterpreter reports whether the script's
// interpreter can't receive Fleet variables. Python has no $VAR expansion, so a
// value could only reach it by being spliced into the source.
func ScriptFleetVarsUnsupportedByInterpreter(contents string) bool {
	kind, _, err := ShebangInfo(contents)
	return err == nil && kind == ShebangPython
}

// ValidateFleetVariablesInScript returns an error if the script contents
// reference a Fleet variable that is not supported in scripts, or if variables
// are used without a premium license.
func ValidateFleetVariablesInScript(contents string, isPremium bool) error {
	fleetVars := variables.Find(contents)
	if len(fleetVars) == 0 {
		return nil
	}
	if !isPremium {
		return ErrMissingLicense
	}
	if v := FindUnsupportedScriptFleetVar(fleetVars); v != "" {
		return NewInvalidArgumentError("script",
			fmt.Sprintf("Fleet variable $FLEET_VAR_%s is not supported in scripts.", v))
	}
	if ScriptFleetVarsUnsupportedByInterpreter(contents) {
		return NewInvalidArgumentError("script", FleetVarsInPythonMsg)
	}
	return nil
}
