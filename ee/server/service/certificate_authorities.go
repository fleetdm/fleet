package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) NewCertificateAuthority(ctx context.Context, p fleet.CertificateAuthorityPayload) (*fleet.CertificateAuthority, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateAuthority{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	// TODO
	// svc.scepConfigService.ValidateNDESSCEPAdminURL(ctx, newSCEPProxy)
	// err := svc.scepConfigService.ValidateSCEPURL(ctx, newSCEPProxy.URL)
	// fleet.preprocess(url, adminUrl, etc...)
	if len(svc.config.Server.PrivateKey) == 0 {
		return nil, &fleet.BadRequestError{Message: "Cannot encrypt NDES password. Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key"}
	}
}

/*

func validateCAName(name string, caType string, allCANames map[string]struct{}, invalid *fleet.InvalidArgumentError) bool {
	if name == "NDES" {
		invalid.Append("integrations."+caType+".name", "CA name cannot be NDES")
		return false
	}
	if len(name) == 0 {
		invalid.Append("integrations."+caType+".name", "CA name cannot be empty")
		return false
	}
	if len(name) > 255 {
		invalid.Append("integrations."+caType+".name", "CA name cannot be longer than 255 characters")
		return false
	}
	if !isAlphanumeric(name) {
		invalid.Append("integrations."+caType+".name",
			fmt.Sprintf("Couldn’t edit integrations.%s. Invalid characters in the \"name\" field. Only letters, "+
				"numbers and underscores allowed. %s",
				caType, name))
		return false
	}
	if _, ok := allCANames[name]; ok {
		invalid.Append("integrations."+caType+".name", fmt.Sprintf("Couldn’t edit certificate authority. "+
			"\"%s\" name is already used by another certificate authority. Please choose a different name and try again.", name))
		return false
	}
	allCANames[name] = struct{}{}
	return true
}

func validateCACN(cn string, invalid *fleet.InvalidArgumentError) bool {
	if len(strings.TrimSpace(cn)) == 0 {
		invalid.Append("integrations.digicert.certificate_common_name", "CA Common Name (CN) cannot be empty")
		return false
	}
	fleetVars := findFleetVariables(cn)
	for fleetVar := range fleetVars {
		switch fleetVar {
		case fleet.FleetVarHostEndUserEmailIDP, fleet.FleetVarHostHardwareSerial:
			// ok
		default:
			invalid.Append("integrations.digicert.certificate_common_name", "FLEET_VAR_"+fleetVar+" is not allowed in CA Common Name (CN)")
			return false
		}
	}
	return true
}

func validateSeatID(seatID string, invalid *fleet.InvalidArgumentError) bool {
	if len(strings.TrimSpace(seatID)) == 0 {
		invalid.Append("integrations.digicert.certificate_seat_id", "CA Seat ID cannot be empty")
		return false
	}
	fleetVars := findFleetVariables(seatID)
	for fleetVar := range fleetVars {
		switch fleetVar {
		case fleet.FleetVarHostEndUserEmailIDP, fleet.FleetVarHostHardwareSerial:
			// ok
		default:
			invalid.Append("integrations.digicert.certificate_seat_id", "FLEET_VAR_"+fleetVar+" is not allowed in DigiCert Seat ID")
			return false
		}
	}
	return true
}

func validateUserPrincipalNames(userPrincipalNames []string, invalid *fleet.InvalidArgumentError) bool {
	if len(userPrincipalNames) == 0 {
		return true
	}
	if len(userPrincipalNames) > 1 {
		invalid.Append("integrations.digicert.certificate_user_principal_names",
			"DigiCert CA can only have one certificate user principal name")
		return false
	}
	if len(strings.TrimSpace(userPrincipalNames[0])) == 0 {
		invalid.Append("integrations.digicert.certificate_user_principal_names",
			"DigiCert CA certificate user principal name cannot be empty if specified")
		return false
	}
	fleetVars := findFleetVariables(userPrincipalNames[0])
	for fleetVar := range fleetVars {
		switch fleetVar {
		case fleet.FleetVarHostEndUserEmailIDP, fleet.FleetVarHostHardwareSerial:
			// ok
		default:
			invalid.Append("integrations.digicert.certificate_user_principal_names",
				"FLEET_VAR_"+fleetVar+" is not allowed in CA User Principal Name")
			return false
		}
	}
	return true
}
*/
