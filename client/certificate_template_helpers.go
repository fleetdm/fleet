package client

import (
	"fmt"
	"slices"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/variables"
)

// fleetVarsSupportedInCertificateTemplates lists the Fleet variables that may
// appear in certificate template subject names.
var fleetVarsSupportedInCertificateTemplates = []fleet.FleetVarName{
	fleet.FleetVarHostUUID,
	fleet.FleetVarHostHardwareSerial,
	fleet.FleetVarHostEndUserIDPUsername,
}

func validateCertificateTemplateFleetVariables(subjectName string) error {
	fleetVars := variables.Find(subjectName)
	if len(fleetVars) == 0 {
		return nil
	}

	for _, fleetVar := range fleetVars {
		if !slices.Contains(fleetVarsSupportedInCertificateTemplates, fleet.FleetVarName(fleetVar)) {
			return fmt.Errorf("Fleet variable $FLEET_VAR_%s is not supported in certificate templates", fleetVar)
		}
	}

	return nil
}
