package service

import (
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/variables"
)

// Certificate Authority structs for MDM profile checking
// to ensure all variables needed for a given CA is present within a single profile.
type DigiCertVarsFound struct {
	dataCA     map[string]struct{}
	passwordCA map[string]struct{}
}

// Ok makes sure that both DATA and PASSWORD variables are present in a DigiCert profile.
func (d *DigiCertVarsFound) Ok() bool {
	if d == nil {
		return true
	}
	if len(d.dataCA) != len(d.passwordCA) {
		return false
	}
	for ca := range d.dataCA {
		if _, ok := d.passwordCA[ca]; !ok {
			return false
		}
	}
	return true
}

func (d *DigiCertVarsFound) Found() bool {
	return d != nil
}

func (d *DigiCertVarsFound) CAs() []string {
	if d == nil {
		return nil
	}
	keys := make([]string, 0, len(d.dataCA))
	for key := range d.dataCA {
		keys = append(keys, key)
	}
	return keys
}

func (d *DigiCertVarsFound) ErrorMessage() string {
	for ca := range d.passwordCA {
		if _, ok := d.dataCA[ca]; !ok {
			return fmt.Sprintf("Missing $FLEET_VAR_%s%s in the profile", fleet.FleetVarDigiCertDataPrefix, ca)
		}
	}
	for ca := range d.dataCA {
		if _, ok := d.passwordCA[ca]; !ok {
			return fmt.Sprintf("Missing $FLEET_VAR_%s%s in the profile", fleet.FleetVarDigiCertPasswordPrefix, ca)
		}
	}
	return fmt.Sprintf("CA name mismatch between $FLEET_VAR_%s<ca_name> and $FLEET_VAR_%s<ca_name> in the profile.",
		fleet.FleetVarDigiCertDataPrefix, fleet.FleetVarDigiCertPasswordPrefix)
}

func (d *DigiCertVarsFound) SetData(value string) (*DigiCertVarsFound, bool) {
	if d == nil {
		d = &DigiCertVarsFound{}
	}
	if d.dataCA == nil {
		d.dataCA = make(map[string]struct{})
	}
	_, alreadyPresent := d.dataCA[value]
	d.dataCA[value] = struct{}{}
	return d, !alreadyPresent
}

func (d *DigiCertVarsFound) SetPassword(value string) (*DigiCertVarsFound, bool) {
	if d == nil {
		d = &DigiCertVarsFound{}
	}
	if d.passwordCA == nil {
		d.passwordCA = make(map[string]struct{})
	}
	_, alreadyPresent := d.passwordCA[value]
	d.passwordCA[value] = struct{}{}
	return d, !alreadyPresent
}

type NDESVarsFound struct {
	urlFound       bool
	challengeFound bool
	renewalIdFound bool
}

// Ok makes sure that Challenge, URL, and renewal ID are present.
func (n *NDESVarsFound) Ok() bool {
	if n == nil {
		return true
	}
	return n.urlFound && n.challengeFound && n.renewalIdFound
}

func (n *NDESVarsFound) Found() bool {
	return n != nil
}

func (n *NDESVarsFound) RenewalOnly() bool {
	return n != nil && !n.urlFound && !n.challengeFound && n.renewalIdFound
}

func (n *NDESVarsFound) ErrorMessage() string {
	if n.renewalIdFound && !n.urlFound && !n.challengeFound {
		return fleet.SCEPRenewalIDWithoutURLChallengeErrMsg
	}
	return fleet.NDESSCEPVariablesMissingErrMsg
}

func (n *NDESVarsFound) SetURL() (*NDESVarsFound, bool) {
	if n == nil {
		n = &NDESVarsFound{}
	}
	alreadyPresent := n.urlFound
	n.urlFound = true
	return n, !alreadyPresent
}

func (n *NDESVarsFound) SetChallenge() (*NDESVarsFound, bool) {
	if n == nil {
		n = &NDESVarsFound{}
	}
	alreadyPresent := n.challengeFound
	n.challengeFound = true
	return n, !alreadyPresent
}

func (n *NDESVarsFound) SetRenewalID() (*NDESVarsFound, bool) {
	if n == nil {
		n = &NDESVarsFound{}
	}
	alreadyPresent := n.renewalIdFound
	n.renewalIdFound = true
	return n, !alreadyPresent
}

type CustomSCEPVarsFound struct {
	urlCA          map[string]struct{}
	challengeCA    map[string]struct{}
	renewalIdFound bool
	// Whether or not presence of renewal ID should be validated.
	// Currently used for microsoft MDM as it does not support renewal, once it does, this can be reverted.
	supportsRenewal bool
	found           bool // Workaround until renewal support, to see if any vars was found in the first place
}

// Ok makes sure that Challenge is present only if URL is also present in SCEP profile.
// This allows the Admin to override the SCEP challenge in the profile.
func (cs *CustomSCEPVarsFound) Ok() bool {
	if cs == nil {
		return true
	}
	if len(cs.challengeCA) != len(cs.urlCA) {
		return false
	}
	if len(cs.challengeCA) == 0 {
		return false
	}
	for ca := range cs.challengeCA {
		if _, ok := cs.urlCA[ca]; !ok {
			return false
		}
	}

	if !cs.supportsRenewal {
		return true
	}

	return cs.renewalIdFound
}

func (cs *CustomSCEPVarsFound) Found() bool {
	return cs.found
}

func (cs *CustomSCEPVarsFound) RenewalOnly() bool {
	return cs != nil && len(cs.urlCA) == 0 && len(cs.challengeCA) == 0 && cs.renewalIdFound
}

func (cs *CustomSCEPVarsFound) CAs() []string {
	if cs == nil {
		return nil
	}
	keys := make([]string, 0, len(cs.urlCA))
	for key := range cs.urlCA {
		keys = append(keys, key)
	}
	return keys
}

func (cs *CustomSCEPVarsFound) ErrorMessage() string {
	if cs.renewalIdFound && len(cs.challengeCA) == 0 && len(cs.urlCA) == 0 {
		return fleet.SCEPRenewalIDWithoutURLChallengeErrMsg
	}
	if !cs.supportsRenewal && (len(cs.challengeCA) == 0 || len(cs.urlCA) == 0) {
		return fmt.Sprintf("SCEP profile for custom SCEP certificate authority requires: $FLEET_VAR_%s<CA_NAME> and $FLEET_VAR_%s<CA_NAME> variables.", fleet.FleetVarCustomSCEPChallengePrefix, fleet.FleetVarCustomSCEPProxyURLPrefix)
	} else if (!cs.renewalIdFound && cs.supportsRenewal) && (len(cs.challengeCA) == 0 || len(cs.urlCA) == 0) {
		return fmt.Sprintf("SCEP profile for custom SCEP certificate authority requires: $FLEET_VAR_%s<CA_NAME>, $FLEET_VAR_%s<CA_NAME>, and $FLEET_VAR_%s variables.", fleet.FleetVarCustomSCEPChallengePrefix, fleet.FleetVarCustomSCEPProxyURLPrefix, fleet.FleetVarSCEPRenewalID)
	}

	for ca := range cs.challengeCA {
		if _, ok := cs.urlCA[ca]; !ok {
			return fmt.Sprintf("Missing $FLEET_VAR_%s%s in the profile", fleet.FleetVarCustomSCEPProxyURLPrefix, ca)
		}
	}
	for ca := range cs.urlCA {
		if _, ok := cs.challengeCA[ca]; !ok {
			return fmt.Sprintf("Missing $FLEET_VAR_%s%s in the profile", fleet.FleetVarCustomSCEPChallengePrefix, ca)
		}
	}

	return fmt.Sprintf("CA name mismatch between $FLEET_VAR_%s<ca_name> and $FLEET_VAR_%s<ca_name> in the profile.",
		fleet.FleetVarCustomSCEPProxyURLPrefix, fleet.FleetVarCustomSCEPChallengePrefix)
}

func (cs *CustomSCEPVarsFound) SetURL(value string) (*CustomSCEPVarsFound, bool) {
	if cs == nil {
		cs = &CustomSCEPVarsFound{
			found: true,
		}
	}
	if cs.urlCA == nil {
		cs.urlCA = make(map[string]struct{})
	}
	_, alreadyPresent := cs.urlCA[value]
	cs.urlCA[value] = struct{}{}
	return cs, !alreadyPresent
}

func (cs *CustomSCEPVarsFound) SetChallenge(value string) (*CustomSCEPVarsFound, bool) {
	if cs == nil {
		cs = &CustomSCEPVarsFound{
			found: true,
		}
	}
	if cs.challengeCA == nil {
		cs.challengeCA = make(map[string]struct{})
	}
	_, alreadyPresent := cs.challengeCA[value]
	cs.challengeCA[value] = struct{}{}
	return cs, !alreadyPresent
}

func (cs *CustomSCEPVarsFound) SetRenewalID() (*CustomSCEPVarsFound, bool) {
	if cs == nil {
		cs = &CustomSCEPVarsFound{
			found: true,
		}
	}
	alreadyPresent := cs.renewalIdFound
	cs.renewalIdFound = true
	return cs, !alreadyPresent
}

type SmallstepVarsFound struct {
	urlCA          map[string]struct{}
	challengeCA    map[string]struct{}
	renewalIdFound bool
}

// Ok makes sure that Challenge is present only if URL is also present in SCEP profile.
// This allows the Admin to override the SCEP challenge in the profile.
func (cs *SmallstepVarsFound) Ok() bool {
	if cs == nil {
		return true
	}
	// There must be a 1:1 mapping between URL and Challenge CAs
	if len(cs.challengeCA) != len(cs.urlCA) {
		return false
	}
	if len(cs.challengeCA) == 0 {
		return false
	}
	for ca := range cs.challengeCA {
		if _, ok := cs.urlCA[ca]; !ok {
			// Unable to find matching URL CA for Challenge CA
			return false
		}
	}
	return cs.renewalIdFound
}

func (cs *SmallstepVarsFound) Found() bool {
	return cs != nil
}

func (cs *SmallstepVarsFound) RenewalOnly() bool {
	return cs != nil && len(cs.urlCA) == 0 && len(cs.challengeCA) == 0 && cs.renewalIdFound
}

func (cs *SmallstepVarsFound) CAs() []string {
	if cs == nil {
		return nil
	}
	keys := make([]string, 0, len(cs.urlCA))
	for key := range cs.urlCA {
		keys = append(keys, key)
	}
	return keys
}

func (cs *SmallstepVarsFound) ErrorMessage() string {
	if cs.renewalIdFound && len(cs.challengeCA) == 0 && len(cs.urlCA) == 0 {
		return fleet.SCEPRenewalIDWithoutURLChallengeErrMsg
	}
	if !cs.renewalIdFound || len(cs.challengeCA) == 0 || len(cs.urlCA) == 0 {
		return fmt.Sprintf("SCEP profile for Smallstep certificate authority requires: $FLEET_VAR_%s<CA_NAME>, $FLEET_VAR_%s<CA_NAME>, and $FLEET_VAR_%s variables.", fleet.FleetVarSmallstepSCEPChallengePrefix, fleet.FleetVarSmallstepSCEPProxyURLPrefix, fleet.FleetVarSCEPRenewalID)
	}
	for ca := range cs.challengeCA {
		if _, ok := cs.urlCA[ca]; !ok {
			return fmt.Sprintf("Missing $FLEET_VAR_%s%s in the profile", fleet.FleetVarSmallstepSCEPProxyURLPrefix, ca)
		}
	}
	for ca := range cs.urlCA {
		if _, ok := cs.challengeCA[ca]; !ok {
			return fmt.Sprintf("Missing $FLEET_VAR_%s%s in the profile", fleet.FleetVarSmallstepSCEPChallengePrefix, ca)
		}
	}
	return fmt.Sprintf("CA name mismatch between $FLEET_VAR_%s<ca_name> and $FLEET_VAR_%s<ca_name> in the profile.",
		fleet.FleetVarSmallstepSCEPProxyURLPrefix, fleet.FleetVarSmallstepSCEPChallengePrefix)
}

func (cs *SmallstepVarsFound) SetURL(value string) (*SmallstepVarsFound, bool) {
	if cs == nil {
		cs = &SmallstepVarsFound{}
	}
	if cs.urlCA == nil {
		cs.urlCA = make(map[string]struct{})
	}
	_, alreadyPresent := cs.urlCA[value]
	cs.urlCA[value] = struct{}{}
	return cs, !alreadyPresent
}

func (cs *SmallstepVarsFound) SetChallenge(value string) (*SmallstepVarsFound, bool) {
	if cs == nil {
		cs = &SmallstepVarsFound{}
	}
	if cs.challengeCA == nil {
		cs.challengeCA = make(map[string]struct{})
	}
	_, alreadyPresent := cs.challengeCA[value]
	cs.challengeCA[value] = struct{}{}
	return cs, !alreadyPresent
}

func (cs *SmallstepVarsFound) SetRenewalID() (*SmallstepVarsFound, bool) {
	if cs == nil {
		cs = &SmallstepVarsFound{}
	}
	alreadyPresent := cs.renewalIdFound
	cs.renewalIdFound = true
	return cs, !alreadyPresent
}

// validateProfileCertificateAuthorityVariables checks that all Fleet variables
// used in the given profile contents correspond to existing Certificate Authorities,
// and that is mapped to a set of CA vars, that can later be used for validation.
//
// TODO: Make this function also handle validation across platforms, but due to time I left it in the respective apple and windows mdm flows.
func validateProfileCertificateAuthorityVariables(profileContents string, lic *fleet.LicenseInfo, platform string, groupedCAs *fleet.GroupedCertificateAuthorities,
	additionalDigiCertValidation func(contents string, digicertVars *DigiCertVarsFound) error,
	additionalCustomSCEPValidation func(contents string, customSCEPVars *CustomSCEPVarsFound) error,
	additionalNDESValidation func(contents string, ndesVars *NDESVarsFound) error,
	additionalSmallstepValidation func(contents string, smallstepVars *SmallstepVarsFound) error,
) error {
	fleetVars := variables.FindKeepDuplicates(profileContents)
	if len(fleetVars) == 0 {
		return nil
	}

	fmt.Println(fleetVars)

	// Check for premium license if the profile contains Fleet variables
	if lic == nil || !lic.IsPremium() {
		return fleet.ErrMissingLicense
	}

	customSCEPVars := &CustomSCEPVarsFound{
		supportsRenewal: platform == fleet.MDMPlatformApple,
	}
	var (
		digiCertVars  *DigiCertVarsFound
		ndesVars      *NDESVarsFound
		smallstepVars *SmallstepVarsFound
	)
	for _, k := range fleetVars {
		caFound := false
		ok := true
		switch {
		case strings.HasPrefix(k, string(fleet.FleetVarDigiCertDataPrefix)):
			caName := strings.TrimPrefix(k, string(fleet.FleetVarDigiCertDataPrefix))
			for _, ca := range groupedCAs.DigiCert {
				if ca.Name == caName {
					caFound = true
					digiCertVars, ok = digiCertVars.SetData(caName)
					break
				}
			}
			if !caFound {
				ok = false
			}
		case strings.HasPrefix(k, string(fleet.FleetVarDigiCertPasswordPrefix)):
			caName := strings.TrimPrefix(k, string(fleet.FleetVarDigiCertPasswordPrefix))
			for _, ca := range groupedCAs.DigiCert {
				if ca.Name == caName {
					caFound = true
					digiCertVars, ok = digiCertVars.SetPassword(caName)
					break
				}
			}
			if !caFound {
				ok = false
			}
		case strings.HasPrefix(k, string(fleet.FleetVarCustomSCEPProxyURLPrefix)):
			caName := strings.TrimPrefix(k, string(fleet.FleetVarCustomSCEPProxyURLPrefix))
			for _, ca := range groupedCAs.CustomScepProxy {
				if ca.Name == caName {
					caFound = true
					customSCEPVars, ok = customSCEPVars.SetURL(caName)
					break
				}
			}
			if !caFound {
				ok = false
			}
		case strings.HasPrefix(k, string(fleet.FleetVarCustomSCEPChallengePrefix)):
			caName := strings.TrimPrefix(k, string(fleet.FleetVarCustomSCEPChallengePrefix))
			for _, ca := range groupedCAs.CustomScepProxy {
				if ca.Name == caName {
					caFound = true
					customSCEPVars, ok = customSCEPVars.SetChallenge(caName)
					break
				}
			}
			if !caFound {
				ok = false
			}
		case strings.HasPrefix(k, string(fleet.FleetVarSmallstepSCEPProxyURLPrefix)):
			caName := strings.TrimPrefix(k, string(fleet.FleetVarSmallstepSCEPProxyURLPrefix))
			for _, ca := range groupedCAs.Smallstep {
				if ca.Name == caName {
					caFound = true
					smallstepVars, ok = smallstepVars.SetURL(caName)
					break
				}
			}
			if !caFound {
				ok = false
			}
		case strings.HasPrefix(k, string(fleet.FleetVarSmallstepSCEPChallengePrefix)):
			caName := strings.TrimPrefix(k, string(fleet.FleetVarSmallstepSCEPChallengePrefix))
			for _, ca := range groupedCAs.Smallstep {
				if ca.Name == caName {
					caFound = true
					smallstepVars, ok = smallstepVars.SetChallenge(caName)
					break
				}
			}
			if !caFound {
				ok = false
			}
		case k == string(fleet.FleetVarNDESSCEPProxyURL):
			caFound = true
			ndesVars, ok = ndesVars.SetURL()
		case k == string(fleet.FleetVarNDESSCEPChallenge):
			caFound = true
			ndesVars, ok = ndesVars.SetChallenge()
		case k == string(fleet.FleetVarSCEPRenewalID):
			caFound = true
			// This is kind of a goofy way of doing things but essentially, since custom SCEP, NDES, and Smallstep
			// share the renewal ID Fleet variable, we need to set the

			customSCEPVars, ok = customSCEPVars.SetRenewalID()
			if ok {
				ndesVars, ok = ndesVars.SetRenewalID()
				if ok {
					smallstepVars, ok = smallstepVars.SetRenewalID()
				}
			}
		}

		if !ok {
			if !caFound {
				return &fleet.BadRequestError{Message: fmt.Sprintf("Fleet variable $FLEET_VAR_%s does not exist.", k)}
			}

			if k == string(fleet.FleetVarSCEPRenewalID) {
				// Special message for renewal ID
				return &fleet.BadRequestError{Message: "Variable $FLEET_VAR_SCEP_RENEWAL_ID must be in the SCEP certificate's organizational unit (OU)."}
			}

			return &fleet.BadRequestError{Message: fmt.Sprintf("Fleet variable $FLEET_VAR_%s is already present in configuration profile.", k)}
		}
	}

	if digiCertVars.Found() {
		if !digiCertVars.Ok() {
			return &fleet.BadRequestError{Message: digiCertVars.ErrorMessage()}
		}
		if additionalDigiCertValidation != nil {
			err := additionalDigiCertValidation(profileContents, digiCertVars)
			if err != nil {
				return err
			}
		}
	}

	// Since custom SCEP, NDES, and Smallstep share the renewal ID Fleet variable, we need to figure out which one to validate.
	if customSCEPVars.Found() || ndesVars.Found() || smallstepVars.Found() {
		if ndesVars.RenewalOnly() {
			ndesVars = nil
		}
		if customSCEPVars.RenewalOnly() {
			customSCEPVars = nil
		}
		if smallstepVars.RenewalOnly() {
			smallstepVars = nil
		}
		// If only the renewal ID variable appeared without any of its associated variables, return an error. It is shared
		// by the 3 CA types but is only allowed when CA vars are in use
		if ndesVars == nil && smallstepVars == nil && customSCEPVars == nil {
			return &fleet.BadRequestError{Message: fleet.SCEPRenewalIDWithoutURLChallengeErrMsg}
		}
	}

	if customSCEPVars.Found() {
		if !customSCEPVars.Ok() {
			return &fleet.BadRequestError{Message: customSCEPVars.ErrorMessage()}
		}
		if additionalCustomSCEPValidation != nil {
			err := additionalCustomSCEPValidation(profileContents, customSCEPVars)
			if err != nil {
				return err
			}
		}
	}
	if ndesVars.Found() {
		if !ndesVars.Ok() {
			return &fleet.BadRequestError{Message: ndesVars.ErrorMessage()}
		}
		if additionalNDESValidation != nil {
			err := additionalNDESValidation(profileContents, ndesVars)
			if err != nil {
				return err
			}
		}
	}
	if smallstepVars.Found() {
		if !smallstepVars.Ok() {
			return &fleet.BadRequestError{Message: smallstepVars.ErrorMessage()}
		}
		if additionalSmallstepValidation != nil {
			err := additionalSmallstepValidation(profileContents, smallstepVars)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
