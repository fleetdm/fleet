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
	FleetFileVaultPayloadIdentifier = "com.fleetdm.fleet.mdm.filevault"

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
// that are handled and delivered by Fleet.
//
// TODO(roperzh): when I was refactoring this, I noticed that the strings are
// not constants, we should refactor that and use the constant in the templates
// we use to generate the FileVault mobileconfig.
func FleetPayloadTypes() map[string]struct{} {
	return map[string]struct{}{
		"com.apple.security.FDERecoveryKeyEscrow": {},
		"com.apple.MCX.FileVault2":                {},
		"com.apple.security.FDERecoveryRedirect":  {},
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
	if mc.isSignedProfile() {
		profileData, err := getSignedProfileData(mc)
		if err != nil {
			return nil, err
		}
		mcBytes = profileData
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
	if mc.isSignedProfile() {
		profileData, err := getSignedProfileData(mc)
		if err != nil {
			return nil, err
		}
		mcBytes = profileData
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
		return fmt.Errorf("unsupported PayloadType(s): %s", strings.Join(screenedTypes, ", "))
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
