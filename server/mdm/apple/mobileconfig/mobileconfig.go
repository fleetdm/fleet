package mobileconfig

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/mdm"

	// we are using this package as we were having issues with pasrsing signed apple
	// mobileconfig profiles with the pcks7 package we were using before.
	cms "github.com/github/smimesign/ietf-cms"
	"howett.net/plist"
)

const (
	// FleetFileVaultPayloadIdentifier is the value for the PayloadIdentifier
	// used by Fleet to configure FileVault and FileVault Escrow.
	FleetFileVaultPayloadIdentifier        = "com.fleetdm.fleet.mdm.filevault"
	FleetFileVaultPayloadType              = "com.apple.MCX.FileVault2"
	FleetCustomSettingsPayloadType         = "com.apple.MCX"
	FleetRecoveryKeyEscrowPayloadType      = "com.apple.security.FDERecoveryKeyEscrow"
	DiskEncryptionProfileRestrictionErrMsg = "Couldn't add. The configuration profile can't include FileVault settings."

	// FleetdConfigPayloadIdentifier is the value for the PayloadIdentifier used
	// by fleetd to read configuration values from the system.
	FleetdConfigPayloadIdentifier = "com.fleetdm.fleetd.config"

	// FleetCARootConfigPayloadIdentifier TODO
	FleetCARootConfigPayloadIdentifier = "com.fleetdm.caroot"

	// FleetEnrollmentPayloadIdentifier is the value for the PayloadIdentifier used
	// by Fleet to enroll a device with the MDM server.
	FleetEnrollmentPayloadIdentifier = "com.fleetdm.fleet.mdm.apple.mdm"

	// FleetEnrollReferenceKey is the key used by Fleet of the URL query parameter representing a unique
	// identifier for an MDM enrollment. The unique value of the query parameter is appended to the
	// Fleet server URL when an MDM enrollment profile is generated for download by a device.
	//
	// TODO: We have some inconsistencies where we use enroll_reference sometimes and
	// enrollment_reference other times. It really should be the same everywhere, but
	// it seems to be working now because the values are matching where they need to match.
	// We should clean this up at some point and update hardcoded values in the codebase.
	FleetEnrollReferenceKey = "enroll_reference"
)

// FleetPayloadIdentifiers returns a map of PayloadIdentifier strings
// that are handled and delivered by Fleet.
//
// TODO(roperzh): at some point we should also include
// apple_mdm.FletPayloadIdentifier here too, but that requires moving a lot of
// files around due to import cycles.
func FleetPayloadIdentifiers() map[string]struct{} {
	return map[string]struct{}{
		FleetFileVaultPayloadIdentifier:    {},
		FleetdConfigPayloadIdentifier:      {},
		FleetCARootConfigPayloadIdentifier: {},
	}
}

// FleetPayloadTypes returns a map of PayloadType strings
// that are fully or partially handled and delivered by Fleet.
func FleetPayloadTypes() map[string]struct{} {
	return map[string]struct{}{
		FleetRecoveryKeyEscrowPayloadType:        {},
		FleetFileVaultPayloadType:                {},
		FleetCustomSettingsPayloadType:           {},
		"com.apple.security.FDERecoveryRedirect": {}, // no longer supported in macOS 10.13 and later
	}
}

// Mobileconfig is the byte slice corresponding to an XML property list (i.e. plist) representation
// of an Apple MDM configuration profile in Fleet.
//
// Configuration profiles are used to configure Apple devices. See also
// https://developer.apple.com/documentation/devicemanagement/configuring_multiple_devices_using_profiles.
type Mobileconfig []byte

type Parsed struct {
	PayloadIdentifier  string
	PayloadDisplayName string
	PayloadType        string
	PayloadScope       string
}

func (mc Mobileconfig) isSignedProfile() bool {
	return !bytes.HasPrefix(bytes.TrimSpace(mc), []byte("<?xml"))
}

// getSignedProfileData attempts to parse the signed mobileconfig and extract the
// profile byte data from it.
func getSignedProfileData(mc Mobileconfig) (Mobileconfig, error) {
	signedData, err := cms.ParseSignedData(mc)
	if err != nil {
		return nil, fmt.Errorf("mobileconfig is not XML nor PKCS7 parseable: %w", err)
	}
	data, err := signedData.GetData()
	if err != nil {
		return nil, fmt.Errorf("could not get profile data from the signed mobileconfig: %w", err)
	}
	return Mobileconfig(data), nil
}

// ParseConfigProfile attempts to parse the Mobileconfig byte slice as a Fleet MDMAppleConfigProfile.
//
// The byte slice must be XML or PKCS7 parseable. Fleet also requires that it contains both
// a PayloadIdentifier and a PayloadDisplayName and that it has PayloadType set to "Configuration".
//
// Adapted from https://github.com/micromdm/micromdm/blob/main/platform/profile/profile.go
func (mc Mobileconfig) ParseConfigProfile() (*Parsed, error) {
	mcBytes := mc
	// Remove Fleet variables expected in <data> section.
	mcBytes = mdm.ProfileDataVariableRegex.ReplaceAll(mcBytes, []byte(""))
	if mc.isSignedProfile() {
		profileData, err := getSignedProfileData(mc)
		if err != nil {
			return nil, err
		}
		mcBytes = profileData
		if mdm.ProfileVariableRegex.Match(mcBytes) {
			return nil, errors.New("a signed profile cannot contain Fleet variables ($FLEET_VAR_*)")
		}
	}
	var p Parsed
	if _, err := plist.Unmarshal(mcBytes, &p); err != nil {
		return nil, err
	}
	if p.PayloadType != "Configuration" {
		return nil, fmt.Errorf("invalid PayloadType: %s", p.PayloadType)
	}
	if p.PayloadIdentifier == "" {
		return nil, errors.New("empty PayloadIdentifier in profile")
	}
	if p.PayloadDisplayName == "" {
		return nil, errors.New("empty PayloadDisplayName in profile")
	}
	// PayloadScope is optional and according to
	// Apple(https://developer.apple.com/business/documentation/Configuration-Profile-Reference.pdf
	// p6) defaults to "User". We've always sent them to the Device channel but now we're saying
	// "User" means use the user channel. For backwards compatibility we are maintaining existing
	// behavior of defaulting to device channel below but we should consider whether this is correct.
	if p.PayloadScope == "" {
		p.PayloadScope = "System"
	}
	if p.PayloadScope != "System" && p.PayloadScope != "User" {
		return nil, fmt.Errorf("invalid PayloadScope: %s", p.PayloadScope)
	}

	return &p, nil
}

type payloadSummary struct {
	Type       string
	Identifier string
	Name       string
}

// payloadSummary attempts to parse the PayloadContent list of the Mobileconfig's TopLevel object.
// It returns the PayloadType for each PayloadContentItem.
//
// See also https://developer.apple.com/documentation/devicemanagement/toplevel
func (mc Mobileconfig) payloadSummary() ([]payloadSummary, error) {
	mcBytes := mc
	// Remove Fleet variables expected in <data> section.
	mcBytes = mdm.ProfileDataVariableRegex.ReplaceAll(mcBytes, []byte(""))
	if mc.isSignedProfile() {
		profileData, err := getSignedProfileData(mc)
		if err != nil {
			return nil, err
		}
		mcBytes = profileData
		if mdm.ProfileVariableRegex.Match(mcBytes) {
			return nil, errors.New("a signed profile cannot contain Fleet variables ($FLEET_VAR_*)")
		}
	}

	// unmarshal the values we need from the top-level object
	var tlo struct {
		IsEncrypted    bool
		PayloadContent []map[string]interface{}
		PayloadType    string
	}
	_, err := plist.Unmarshal(mcBytes, &tlo)
	if err != nil {
		return nil, err
	}
	// confirm that the top-level payload type matches the expected value
	if tlo.PayloadType != "Configuration" {
		return nil, &ErrInvalidPayloadType{tlo.PayloadType}
	}

	if len(tlo.PayloadContent) < 1 {
		if tlo.IsEncrypted {
			return nil, ErrEncryptedPayloadContent
		}
		return nil, ErrEmptyPayloadContent
	}

	// extract the payload types of each payload content item from the array of
	// payload dictionaries
	var result []payloadSummary
	for _, payloadDict := range tlo.PayloadContent {
		summary := payloadSummary{}

		pt, ok := payloadDict["PayloadType"]
		if ok {
			if s, ok := pt.(string); ok {
				summary.Type = s
			}
		}

		pi, ok := payloadDict["PayloadIdentifier"]
		if ok {
			if s, ok := pi.(string); ok {
				summary.Identifier = s
			}
		}

		pdn, ok := payloadDict["PayloadDisplayName"]
		if ok {
			if s, ok := pdn.(string); ok {
				summary.Name = s
			}
		}

		if summary.Type != "" || summary.Identifier != "" || summary.Name != "" {
			result = append(result, summary)
		}

	}

	return result, nil
}

func (mc *Mobileconfig) ScreenPayloads() error {
	pct, err := mc.payloadSummary()
	if err != nil {
		// don't error if there's nothing for us to screen.
		if !errors.Is(err, ErrEmptyPayloadContent) && !errors.Is(err, ErrEncryptedPayloadContent) {
			return err
		}
	}

	fleetNames := mdm.FleetReservedProfileNames()
	fleetIdentifiers := FleetPayloadIdentifiers()
	fleetTypes := FleetPayloadTypes()
	screenedTypes := []string{}
	screenedIdentifiers := []string{}
	screenedNames := []string{}
	for _, t := range pct {
		if _, ok := fleetTypes[t.Type]; ok {
			screenedTypes = append(screenedTypes, t.Type)
		}
		if _, ok := fleetIdentifiers[t.Identifier]; ok {
			screenedIdentifiers = append(screenedIdentifiers, t.Identifier)
		}
		if _, ok := fleetNames[t.Name]; ok {
			screenedNames = append(screenedNames, t.Name)
		}
	}

	if len(screenedTypes) > 0 {
		var unsupportedTypes []string
		for _, t := range screenedTypes {
			switch t {
			case FleetFileVaultPayloadType, FleetRecoveryKeyEscrowPayloadType:
				return errors.New(DiskEncryptionProfileRestrictionErrMsg)
			case FleetCustomSettingsPayloadType:
				contains, err := ContainsFDEFileVaultOptionsPayload(*mc)
				if err != nil {
					return fmt.Errorf("checking for FDEVileVaultOptions payload: %w", err)
				}
				if contains {
					return errors.New(DiskEncryptionProfileRestrictionErrMsg)
				}
			default:
				unsupportedTypes = append(unsupportedTypes, t)
			}
		}
		if len(unsupportedTypes) > 0 {
			return fmt.Errorf("unsupported PayloadType(s): %s", strings.Join(screenedTypes, ", "))
		}
	}

	if len(screenedIdentifiers) > 0 {
		return fmt.Errorf("unsupported PayloadIdentifier(s): %s", strings.Join(screenedIdentifiers, ", "))
	}

	if len(screenedNames) > 0 {
		return fmt.Errorf("unsupported PayloadDisplayName(s): %s", strings.Join(screenedNames, ", "))
	}

	return nil
}

type ErrInvalidPayloadType struct {
	payloadType string
}

func (e ErrInvalidPayloadType) Error() string {
	return fmt.Sprintf("invalid PayloadType: %s", e.payloadType)
}

var (
	ErrEmptyPayloadContent     = errors.New("empty PayloadContent")
	ErrEncryptedPayloadContent = errors.New("encrypted PayloadContent")
)

// XMLEscapeString returns the escaped XML equivalent of the plain text data s.
func XMLEscapeString(s string) (string, error) {
	// avoid allocation if we can.
	if !strings.ContainsAny(s, "'\"&<>\t\n\r") {
		return s, nil
	}
	var b strings.Builder
	if err := xml.EscapeText(&b, []byte(s)); err != nil {
		return "", err
	}

	return b.String(), nil
}
