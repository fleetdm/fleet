package service

import (
	"context"
	"fmt"
	"slices"

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
			if host.UUID == "" {
				return "", ctxerr.Errorf(ctx, "host does not have a UUID for variable %s", fleetVar)
			}
			result = fleet.FleetVarHostUUIDRegexp.ReplaceAllString(result, host.UUID)
		case string(fleet.FleetVarHostHardwareSerial):
			if host.HardwareSerial == "" {
				return "", ctxerr.Errorf(ctx, "host %s does not have a hardware serial for variable %s", host.UUID, fleetVar)
			}
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
		default:
			return "", ctxerr.Errorf(ctx, "unsupported Fleet variable %s in certificate template", fleetVar)
		}
	}

	return result, nil
}
