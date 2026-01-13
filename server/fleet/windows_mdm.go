package fleet

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

// MDMWindowsBitLockerSummary reports the number of Windows hosts being managed by Fleet with
// BitLocker. Each host may be counted in only one of six mutually-exclusive categories:
// Verified, Verifying, ActionRequired, Enforcing, Failed, RemovingEnforcement.
//
// Note that it is expected that each of Verifying, ActionRequired, and RemovingEnforcement will be
// zero because these states are not in Fleet's current implementation of BitLocker management.
type MDMWindowsBitLockerSummary struct {
	Verified            uint `json:"verified" db:"verified"`
	Verifying           uint `json:"verifying" db:"verifying"`
	ActionRequired      uint `json:"action_required" db:"action_required"`
	Enforcing           uint `json:"enforcing" db:"enforcing"`
	Failed              uint `json:"failed" db:"failed"`
	RemovingEnforcement uint `json:"removing_enforcement" db:"removing_enforcement"`
}

// MDMWindowsConfigProfile represents a Windows MDM profile in Fleet.
type MDMWindowsConfigProfile struct {
	// ProfileUUID is the unique identifier of the configuration profile in
	// Fleet. For Windows profiles, it is the letter "w" followed by a uuid.
	ProfileUUID      string                      `db:"profile_uuid" json:"profile_uuid"`
	TeamID           *uint                       `db:"team_id" json:"team_id"`
	Name             string                      `db:"name" json:"name"`
	SyncML           []byte                      `db:"syncml" json:"-"`
	LabelsIncludeAll []ConfigurationProfileLabel `db:"-" json:"labels_include_all,omitempty"`
	LabelsIncludeAny []ConfigurationProfileLabel `db:"-" json:"labels_include_any,omitempty"`
	LabelsExcludeAny []ConfigurationProfileLabel `db:"-" json:"labels_exclude_any,omitempty"`
	CreatedAt        time.Time                   `db:"created_at" json:"created_at"`
	UploadedAt       time.Time                   `db:"uploaded_at" json:"updated_at"` // NOTE: JSON field is still `updated_at` for historical reasons, would be an API breaking change
}

type windowsProfileValidator struct {
	// Validator for SCEP profiles, to keep track of SCEP-specific profile validation
	scepValidator *windowsSCEPProfileValidator
	// Boolean indicating whether the profile is an Atomic profile, if so don't allow other top level elements.
	// Starts out as nil until we encounter the first element, to avoid Atomic coming later
	isAtomicProfile *bool

	// The current element being processed, e.g., "LocURI", "Target", etc.
	// Will also be top-level elements before we get to inner elements
	currentElement string
	// The current top-level element being processed, e.g., "Replace", "Add", etc.
	// can be empty if not within a top-level element.
	currentTopLevelElement string

	// The decoder which is used for reading the XML tokens.
	decoder *xml.Decoder

	// Whether to enable validation for custom OS updates loc URIs.
	enableCustomOSUpdates bool
}

var validTopLevelElements = map[string]struct{}{
	"Replace": {},
	"Add":     {},
	"Exec":    {},
	"Atomic":  {},
}

// ValidateUserProvided ensures that the SyncML content in the profile is valid
// for Windows.
//
// It checks that all top-level elements are <Replace> and none of the <LocURI>
// elements within <Target> are reserved URIs.
//
// It also performs basic checks for XML well-formedness as defined in the [W3C
// Recommendation section 2.8][1], as required by the [MS-MDM spec][2].
//
// Note that we only need to check for well-formedness, but validation is not required.
//
// Returns an error if these conditions are not met.
//
// [1]: http://www.w3.org/TR/2006/REC-xml-20060816
// [2]: https://winprotocoldoc.blob.core.windows.net/productionwindowsarchives/MS-MDM/%5bMS-MDM%5d.pdf
func (m *MDMWindowsConfigProfile) ValidateUserProvided(enableCustomOSUpdates bool) error {
	if len(bytes.TrimSpace(m.SyncML)) == 0 {
		return errors.New("The file should include valid XML.")
	}
	fleetNames := mdm.FleetReservedProfileNames()
	if _, ok := fleetNames[m.Name]; ok {
		return fmt.Errorf("Profile name %q is not allowed.", m.Name)
	}

	validator := newWindowsProfileValidator(m.SyncML, enableCustomOSUpdates)
	return validator.validate()
}

func newWindowsProfileValidator(syncML []byte, enableCustomOSUpdates bool) *windowsProfileValidator {
	dec := xml.NewDecoder(bytes.NewReader(syncML))
	// use strict mode to check for a variety of common mistakes like
	// unclosed tags, etc.
	dec.Strict = true

	return &windowsProfileValidator{
		scepValidator:         newWindowsSCEPProfileValidator(),
		decoder:               dec,
		enableCustomOSUpdates: enableCustomOSUpdates,
	}
}

func (v *windowsProfileValidator) validate() error {
	for {
		tok, err := v.decoder.Token()
		if err != nil {
			if err != io.EOF {
				return fmt.Errorf("The file should include valid XML: %w", err)
			}
			break
		}

		if err := v.processToken(tok); err != nil {
			return err
		}
	}

	return v.scepValidator.finalizeValidation()
}

func (v *windowsProfileValidator) processToken(tok xml.Token) error {
	switch t := tok.(type) {
	// no processing instructions allowed (<?target inst?>)
	// see #16316 for details
	case xml.ProcInst:
		return errors.New("The file should include valid XML: processing instructions are not allowed.")
	case xml.Comment:
		// TODO: Do we really care about comments? Why not allow them everywhere?
	case xml.StartElement:
		return v.handleStartElement(t)
	case xml.EndElement:
		v.handleEndElement(t)
	case xml.CharData:
		return v.handleCharData(t)

	}
	return nil
}

func (v *windowsProfileValidator) handleStartElement(el xml.StartElement) error {
	elementName := el.Name.Local

	if v.isAtTopLevel() {

		if _, valid := validTopLevelElements[elementName]; !valid {
			// We agreed with Design that it's okay to not include <Atomic> in the msg here.
			return errors.New("Windows configuration profiles can only have <Replace> or <Add> top level elements.")
		}

		// We have an atomic profile and we see another top level element, we don't care what it is.
		if v.isAtomicProfile != nil && *v.isAtomicProfile {
			return errors.New("<Atomic> element must wrap all the elements in a Windows configuration profile.")
		}

		if elementName == "Atomic" && v.isAtomicProfile == nil {
			// We are at top level, and we see Atomic, mark the entire profile.
			v.isAtomicProfile = ptr.Bool(true)
		} else if elementName == "Atomic" && v.isAtomicProfile != nil && !*v.isAtomicProfile {
			// We are at top level, we have already seen other top level elements, and now we see Atomic
			return errors.New("Windows configuration profiles can only have <Replace> or <Add> top level elements.")
		}

		v.currentTopLevelElement = elementName
		if v.isAtomicProfile == nil {
			// We are at top level, and we see a non-Atomic element first, mark the profile as non-Atomic.
			v.isAtomicProfile = ptr.Bool(false)
		}
	} else {
		if v.currentElement == "Atomic" && !v.isValidNestedAtomicElement(elementName) {
			return errors.New("Windows configuration profiles can only include <Replace> or <Add> within the <Atomic> element.")
		}
	}

	v.currentElement = elementName
	return nil
}

func (v *windowsProfileValidator) handleEndElement(el xml.EndElement) {
	elementName := el.Name.Local

	if elementName == v.currentTopLevelElement {
		// We are closing a top-level element.
		v.currentTopLevelElement = ""
	}

	v.currentElement = ""
}

func (v *windowsProfileValidator) handleCharData(el xml.CharData) error {
	// We only care about LocURI elements.
	if !v.isInLocURI() {
		return nil
	}

	locURI := string(el)

	if v.isInExec() {
		if err := v.scepValidator.validateExecLocURI(locURI); err != nil {
			return err
		}
	} else {
		if err := v.scepValidator.validateLocURI(locURI); err != nil {
			return err
		}
	}

	return validateFleetProvidedLocURI(locURI, v.enableCustomOSUpdates)
}

func (v *windowsProfileValidator) isAtTopLevel() bool {
	return v.currentTopLevelElement == ""
}

func (v *windowsProfileValidator) isValidNestedAtomicElement(elementName string) bool {
	_, valid := validTopLevelElements[elementName]
	return valid && elementName != "Atomic"
}

func (v *windowsProfileValidator) isInLocURI() bool {
	return v.currentElement == "LocURI"
}

func (v *windowsProfileValidator) isInExec() bool {
	return v.currentTopLevelElement == "Exec"
}

var fleetProvidedLocURIValidationMap = map[string][]string{
	syncml.FleetBitLockerTargetLocURI: nil,
	syncml.FleetOSUpdateTargetLocURI:  {"Windows updates", "mdm.windows_updates"},
}

func validateFleetProvidedLocURI(locURI string, enableCustomOSUpdates bool) error {
	sanitizedLocURI := strings.TrimSpace(locURI)
	for fleetLocURI, errHints := range fleetProvidedLocURIValidationMap {
		if strings.Contains(sanitizedLocURI, fleetLocURI) {
			if fleetLocURI == syncml.FleetOSUpdateTargetLocURI && enableCustomOSUpdates {
				continue
			}
			if fleetLocURI == syncml.FleetBitLockerTargetLocURI {
				return errors.New(syncml.DiskEncryptionProfileRestrictionErrMsg)
			}
			if len(errHints) == 2 {
				return fmt.Errorf("Custom configuration profiles can't include %s settings. To control these settings, use the %s option.",
					errHints[0], errHints[1])
			}
			return fmt.Errorf("Custom configuration profiles can't include these settings. %q", errHints)
		}
	}

	return nil
}

// The following list of SCEP LocURIs is based on the documentation at
// https://learn.microsoft.com/en-us/windows/client-management/mdm/clientcertificateinstall-csp#devicescep
// Where going through all items, only for those with (Add or Replace) under "Supported operations" are included,
// and then based on it being marked Optional, or Required in the description.

// A list containg all valid SCEP Profile LocURIs, a combination of optional and required to validate for non-SCEP LocURIs.
var validDeviceSCEPProfileLocURIs = slices.Concat([]string{
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/AADKeyIdentifierList", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/ContainerName", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/CustomTextToShowInPrompt", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/KeyProtection", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/RetryCount", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/RetryDelay", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/SubjectAlternativeNames", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/TemplateName", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/ValidPeriod", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/ValidPeriodUnits", FleetVarSCEPWindowsCertificateID.WithPrefix()),
}, requiredDeviceSCEPProfileLocURIs)

var validUserSCEPProfileLocURIs = slices.Concat([]string{
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/AADKeyIdentifierList", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/ContainerName", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/CustomTextToShowInPrompt", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/KeyProtection", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/RetryCount", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/RetryDelay", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/SubjectAlternativeNames", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/TemplateName", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/ValidPeriod", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/ValidPeriodUnits", FleetVarSCEPWindowsCertificateID.WithPrefix()),
}, requiredUserSCEPProfileLocURIs)

var requiredDeviceSCEPProfileLocURIs = []string{
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/CAThumbprint", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/Challenge", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/EKUMapping", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/HashAlgorithm", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/KeyLength", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/KeyUsage", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/ServerURL", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/SubjectName", FleetVarSCEPWindowsCertificateID.WithPrefix()),
}

var requiredUserSCEPProfileLocURIs = []string{
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/CAThumbprint", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/Challenge", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/EKUMapping", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/HashAlgorithm", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/KeyLength", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/KeyUsage", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/ServerURL", FleetVarSCEPWindowsCertificateID.WithPrefix()),
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/SubjectName", FleetVarSCEPWindowsCertificateID.WithPrefix()),
}

var validDeviceExecSCEPProfileLocURIs = []string{
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/Enroll", FleetVarSCEPWindowsCertificateID.WithPrefix()),
}

var validUserExecSCEPProfileLocURIs = []string{
	fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/Enroll", FleetVarSCEPWindowsCertificateID.WithPrefix()),
}

type windowsSCEPProfileValidator struct {
	totalLocURIs                int
	totalExecLocURIs            int
	foundLocURIs                map[string]bool
	foundExecLocURIs            map[string]bool
	requiredSCEPProfileLocURIs  *[]string
	validSCEPProfileLocURIs     *[]string
	validExecSCEPProfileLocURIs *[]string
}

func newWindowsSCEPProfileValidator() *windowsSCEPProfileValidator {
	return &windowsSCEPProfileValidator{
		foundLocURIs:     make(map[string]bool),
		foundExecLocURIs: make(map[string]bool),
	}
}

func (v windowsSCEPProfileValidator) normalizeSCEPLocURI(locURI string) string {
	trimmed := strings.TrimSpace(locURI)
	// Accept braces version of the Fleet Var, and normalize it to the non-braces for validation.
	return strings.ReplaceAll(trimmed, FleetVarSCEPWindowsCertificateID.WithBraces(), FleetVarSCEPWindowsCertificateID.WithPrefix())
}

func (v *windowsSCEPProfileValidator) isSCEPProfile() bool {
	return len(v.foundLocURIs) > 0 || (v.totalExecLocURIs > 0 && len(v.foundExecLocURIs) > 0)
}

func (v *windowsSCEPProfileValidator) validateLocURI(locURI string) error {
	normalizedLocURI := v.normalizeSCEPLocURI(locURI)

	if err := v.setLocURIArrays(normalizedLocURI); err != nil {
		return err
	}

	// If we see a LocURI with SCEP prefix, but no Fleet Var we fail early.
	if v.isSCEPLocURIWithoutFleetVar(normalizedLocURI) {
		return fmt.Errorf("You must use %q after \"ClientCertificateInstall/SCEP/\".", FleetVarSCEPWindowsCertificateID.WithPrefix())
	}

	if slices.Contains(*v.validSCEPProfileLocURIs, normalizedLocURI) {
		v.foundLocURIs[normalizedLocURI] = true
	}

	v.totalLocURIs++
	return nil
}

func (v *windowsSCEPProfileValidator) validateExecLocURI(locURI string) error {
	normalizedLocURI := v.normalizeSCEPLocURI(locURI)

	if err := v.setLocURIArrays(normalizedLocURI); err != nil {
		return err
	}

	// If we see a LocURI with SCEP prefix, but no Fleet Var we fail early.
	if v.isSCEPLocURIWithoutFleetVar(normalizedLocURI) {
		return fmt.Errorf("You must use %q after \"ClientCertificateInstall/SCEP/\".", FleetVarSCEPWindowsCertificateID.WithPrefix())
	}

	if slices.Contains(*v.validExecSCEPProfileLocURIs, normalizedLocURI) {
		v.foundExecLocURIs[normalizedLocURI] = true
	}

	v.totalExecLocURIs++
	return nil
}

func (v *windowsSCEPProfileValidator) setLocURIArrays(locURI string) error {
	switch {
	case IsWindowsSCEPLocURI(locURI) && v.validExecSCEPProfileLocURIs == nil:
		if strings.HasPrefix(locURI, "./User") {
			v.requiredSCEPProfileLocURIs = &requiredUserSCEPProfileLocURIs
			v.validSCEPProfileLocURIs = &validUserSCEPProfileLocURIs
			v.validExecSCEPProfileLocURIs = &validUserExecSCEPProfileLocURIs
		} else {
			v.requiredSCEPProfileLocURIs = &requiredDeviceSCEPProfileLocURIs
			v.validSCEPProfileLocURIs = &validDeviceSCEPProfileLocURIs
			v.validExecSCEPProfileLocURIs = &validDeviceExecSCEPProfileLocURIs
		}
	case !IsWindowsSCEPLocURI(locURI) && v.validExecSCEPProfileLocURIs == nil:
		// Not a SCEP profile, set empty arrays to avoid nil pointer dereference later.
		emptyArray := []string{}
		v.requiredSCEPProfileLocURIs = &emptyArray
		v.validSCEPProfileLocURIs = &emptyArray
		v.validExecSCEPProfileLocURIs = &emptyArray
	case IsWindowsSCEPLocURI(locURI) && v.validExecSCEPProfileLocURIs != nil:
		// Check against mixing Device and User SCEP LocURIs.
		firstValidLocURI := (*v.validSCEPProfileLocURIs)[0]
		if strings.HasPrefix(firstValidLocURI, "./Device") && strings.HasPrefix(locURI, "./User") ||
			strings.HasPrefix(firstValidLocURI, "./User") && strings.HasPrefix(locURI, "./Device") {
			return errors.New("All <LocURI> elements in the SCEP profile must start either with \"./Device\" or \"./User\".")
		}
	}

	return nil
}

// isSCEPLocURIWithoutFleetVar checks that the provided locURI starts with the SCEP prefix
// and that it includes the required Fleet Var for SCEP Windows Certificate ID.
// Skips any locURI that does not start with the SCEP prefix.
func (v windowsSCEPProfileValidator) isSCEPLocURIWithoutFleetVar(locURI string) bool {
	if (strings.HasPrefix(locURI, "./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/") &&
		!strings.HasPrefix(locURI, fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s", FleetVarSCEPWindowsCertificateID.WithPrefix())) &&
		!strings.HasPrefix(locURI, fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s", FleetVarSCEPWindowsCertificateID.WithBraces()))) ||
		(strings.HasPrefix(locURI, "./User/Vendor/MSFT/ClientCertificateInstall/SCEP/") &&
			!strings.HasPrefix(locURI, fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s", FleetVarSCEPWindowsCertificateID.WithPrefix())) &&
			!strings.HasPrefix(locURI, fmt.Sprintf("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/%s", FleetVarSCEPWindowsCertificateID.WithBraces()))) {
		return true
	}
	return false
}

func IsWindowsSCEPLocURI(locURI string) bool {
	return strings.HasPrefix(locURI, "./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/") ||
		strings.HasPrefix(locURI, "./User/Vendor/MSFT/ClientCertificateInstall/SCEP/")
}

func (v *windowsSCEPProfileValidator) finalizeValidation() error {
	if !v.isSCEPProfile() {
		// Cheeky validation here, to only allow Exec elements in SCEP profiles.
		if v.totalExecLocURIs > 0 {
			return errors.New("Only SCEP profiles can include <Exec> elements.")
		}
		return nil // Not a SCEP profile, nothing to validate here.
	}

	// If we reach here with empty arrays something has gone wrong.
	if len(*v.validExecSCEPProfileLocURIs) == 0 || len(*v.validSCEPProfileLocURIs) == 0 || len(*v.requiredSCEPProfileLocURIs) == 0 {
		return errors.New("Internal error validating SCEP profile LocURIs.")
	}

	// Check that at least one Exec LocURI is present and it matches the only one we have in the array.
	validExecLocURIs := *v.validExecSCEPProfileLocURIs
	if len(v.foundExecLocURIs) != 1 && !v.foundExecLocURIs[validExecLocURIs[0]] {
		return errors.New("\"ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Enroll\" must be included within <Exec>. Please add and try again.")
	}

	if v.totalExecLocURIs != 1 {
		return errors.New("SCEP profiles must include exactly one <Exec> element.")
	}

	// Verify that we do not have any non-scep loc URIs present
	if v.totalLocURIs != len(v.foundLocURIs) {
		return errors.New("Only options that have <LocURI> starting with \"ClientCertificateInstall/SCEP/\" can be added to SCEP profile.")
	}

	// Check that all required LocURIs are present
	for _, requiredLocURI := range *v.requiredSCEPProfileLocURIs {
		if !v.foundLocURIs[requiredLocURI] {
			trimmedPrefix := strings.TrimPrefix(requiredLocURI, "./Device/Vendor/MSFT/")
			trimmedPrefix = strings.TrimPrefix(trimmedPrefix, "./User/Vendor/MSFT/")
			return fmt.Errorf("%q is missing. Please add and try again.", trimmedPrefix)
		}
	}

	return nil
}

type MDMWindowsProfilePayload struct {
	ProfileUUID      string             `db:"profile_uuid"`
	ProfileName      string             `db:"profile_name"`
	HostUUID         string             `db:"host_uuid"`
	Status           *MDMDeliveryStatus `db:"status" json:"status"`
	OperationType    MDMOperationType   `db:"operation_type"`
	Detail           string             `db:"detail"`
	CommandUUID      string             `db:"command_uuid"`
	Retries          int                `db:"retries"`
	Checksum         []byte             `db:"checksum"`
	SecretsUpdatedAt *time.Time         `db:"secrets_updated_at"`
}

func (p MDMWindowsProfilePayload) Equal(other MDMWindowsProfilePayload) bool {
	statusEqual := p.Status == nil && other.Status == nil || p.Status != nil && other.Status != nil && *p.Status == *other.Status
	secretsEqual := p.SecretsUpdatedAt == nil && other.SecretsUpdatedAt == nil || p.SecretsUpdatedAt != nil && other.SecretsUpdatedAt != nil && p.SecretsUpdatedAt.Equal(*other.SecretsUpdatedAt)
	return statusEqual && secretsEqual &&
		p.ProfileUUID == other.ProfileUUID &&
		p.HostUUID == other.HostUUID &&
		p.ProfileName == other.ProfileName &&
		p.OperationType == other.OperationType &&
		p.Detail == other.Detail &&
		p.CommandUUID == other.CommandUUID &&
		p.Retries == other.Retries &&
		bytes.Equal(p.Checksum, other.Checksum)
}

type MDMWindowsBulkUpsertHostProfilePayload struct {
	ProfileUUID   string
	ProfileName   string
	HostUUID      string
	CommandUUID   string
	OperationType MDMOperationType
	Status        *MDMDeliveryStatus
	Detail        string
	Checksum      []byte
}

type MDMWindowsProfileContents struct {
	SyncML   []byte `db:"syncml"`
	Checksum []byte `db:"checksum"`
}

// MDMWindowsWipeType specifies what type of remote wipe we want
// to perform.
type MDMWindowsWipeType int

const (
	MDMWindowsWipeTypeDoWipe MDMWindowsWipeType = iota
	MDMWindowsWipeTypeDoWipeProtected
)

var wipeTypeVariants = map[MDMWindowsWipeType]string{
	MDMWindowsWipeTypeDoWipe:          "doWipe",
	MDMWindowsWipeTypeDoWipeProtected: "doWipeProtected",
}

func (wt *MDMWindowsWipeType) String() string {
	if wt == nil {
		return "<nil>"
	}
	return wipeTypeVariants[*wt]
}

func (wt *MDMWindowsWipeType) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	for k, v := range wipeTypeVariants {
		if v == s {
			*wt = k
			return nil
		}
	}
	return fmt.Errorf("invalid WipeType: %s", s)
}

type MDMWindowsWipeMetadata struct {
	WipeType MDMWindowsWipeType `json:"wipe_type"`
}
