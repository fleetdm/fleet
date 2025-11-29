package service

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/variables"
)

// Fleet variables supported in certificate template subject names.
var fleetVarsSupportedInCertificateTemplates = []fleet.FleetVarName{
	fleet.FleetVarHostUUID,
	fleet.FleetVarHostHardwareSerial,
	fleet.FleetVarHostEndUserIDPUsername,
}

func validateCertificateTemplateFleetVariables(subjectName string) error {
	// reject any variable that doesn't use the FLEET_VAR_ prefix.
	re := regexp.MustCompile(`\$(?:\{)?([A-Za-z_][A-Za-z0-9_]*)\}?`)
	matches := re.FindAllStringSubmatch(subjectName, -1)
	for _, m := range matches {
		name := m[1]
		if !strings.HasPrefix(name, "FLEET_VAR_") {
			return fmt.Errorf("variable $%s is not allowed; certificate template variables must be FLEET_VAR_*", name)
		}
	}

	// check that all FLEET_VAR_* variables are supported
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

// replaceCertificateVariables replaces FLEET_VAR_* variables in the subject name with actual host values
func (svc *Service) replaceCertificateVariables(ctx context.Context, subjectName string, host *fleet.Host) (string, error) {
	fleetVars := variables.Find(subjectName)
	if len(fleetVars) == 0 {
		return subjectName, nil
	}

	result := subjectName
	for _, fleetVar := range fleetVars {
		switch fleetVar {
		case string(fleet.FleetVarHostUUID):
			result = fleet.FleetVarHostUUIDRegexp.ReplaceAllString(result, host.UUID)
		case string(fleet.FleetVarHostHardwareSerial):
			result = fleet.FleetVarHostHardwareSerialRegexp.ReplaceAllString(result, host.HardwareSerial)
		case string(fleet.FleetVarHostEndUserIDPUsername):
			users, err := fleet.GetEndUsers(ctx, svc.ds, host.ID)
			if err != nil {
				return "", ctxerr.Wrapf(ctx, err, "getting host end users for variable %s", fleetVar)
			}
			if len(users) == 0 || users[0].IdpUserName == "" {
				return "", ctxerr.Errorf(ctx, "host %s does not have an IDP username for variable %s", host.UUID, fleetVar)
			}
			result = fleet.FleetVarHostEndUserIDPUsernameRegexp.ReplaceAllString(result, users[0].IdpUserName)
		}
	}

	return result, nil
}
