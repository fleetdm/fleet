package mobileconfig

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"go.mozilla.org/pkcs7"
	"howett.net/plist"
)

const (
	FleetFileVaultPayloadIdentifier = "com.fleetdm.fleet.mdm.filevault"

	// FleetdConfigPayloadIdentifier is the value for the PayloadIdentifier used
	// by fleetd to read configuration values from the system.
	FleetdConfigPayloadIdentifier = "com.fleetdm.fleetd.config"
)

// FleetPayloadIdentifiers returns a map of profile identifiers
// that are handled and delivered by Fleet.
func FleetPayloadIdentifiers() map[string]struct{} {
	return map[string]struct{}{
		//FleetPayloadIdentifier:          {},
		FleetFileVaultPayloadIdentifier: {},
		FleetdConfigPayloadIdentifier:   {},
	}
}

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

// ParseConfigProfile attempts to parse the Mobileconfig byte slice as a Fleet MDMAppleConfigProfile.
//
// The byte slice must be XML or PKCS7 parseable. Fleet also requires that it contains both
// a PayloadIdentifier and a PayloadDisplayName and that it has PayloadType set to "Configuration".
//
// Adapted from https://github.com/micromdm/micromdm/blob/main/platform/profile/profile.go
func (mc Mobileconfig) ParseConfigProfile() (*Parsed, error) {
	mcBytes := mc
	if !bytes.HasPrefix(mcBytes, []byte("<?xml")) {
		p7, err := pkcs7.Parse(mcBytes)
		if err != nil {
			return nil, fmt.Errorf("mobileconfig is not XML nor PKCS7 parseable: %w", err)
		}
		err = p7.Verify()
		if err != nil {
			return nil, err
		}
		mcBytes = Mobileconfig(p7.Content)
	}
	p := &Parsed{}
	_, err := plist.Unmarshal(mcBytes, &p)
	if err != nil {
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

	return p, nil
}

type PayloadSummary struct {
	Type       string
	Identifier string
}

// GetPayloadSummary attempts to parse the PayloadContent list of the Mobileconfig's TopLevel object.
// It returns the PayloadType for each PayloadContentItem.
//
// See also https://developer.apple.com/documentation/devicemanagement/toplevel
func (mc Mobileconfig) GetPayloadSummary() ([]PayloadSummary, error) {
	mcBytes := mc
	if !bytes.HasPrefix(mcBytes, []byte("<?xml")) {
		p7, err := pkcs7.Parse(mcBytes)
		if err != nil {
			return nil, fmt.Errorf("mobileconfig is not XML nor PKCS7 parseable: %w", err)
		}
		err = p7.Verify()
		if err != nil {
			return nil, err
		}
		mcBytes = Mobileconfig(p7.Content)
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
	var result []PayloadSummary
	for _, payloadDict := range tlo.PayloadContent {
		summary := PayloadSummary{}

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

		if summary.Type != "" || summary.Identifier != "" {
			result = append(result, summary)
		}

	}

	return result, nil
}

func (mc *Mobileconfig) ScreenPayloads() error {
	pct, err := mc.GetPayloadSummary()
	if err != nil {
		// don't error if there's nothing for us to screen.
		if !errors.Is(err, ErrEmptyPayloadContent) || !errors.Is(err, ErrEncryptedPayloadContent) {
			return err
		}
	}

	fleetIdentifiers := FleetPayloadIdentifiers()
	fleetTypes := FleetPayloadTypes()
	screenedTypes := []string{}
	screenedIdentifiers := []string{}
	for _, t := range pct {
		if _, ok := fleetTypes[t.Type]; ok {
			screenedTypes = append(screenedTypes, t.Type)
		}
		if _, ok := fleetIdentifiers[t.Identifier]; ok {
			screenedIdentifiers = append(screenedIdentifiers, t.Identifier)
		}
	}

	if len(screenedTypes) > 0 {
		return fmt.Errorf("unsupported PayloadType(s): %s", strings.Join(screenedTypes, ", "))
	}

	if len(screenedIdentifiers) > 0 {
		return fmt.Errorf("unsupported PayloadIdentifier(s): %s", strings.Join(screenedIdentifiers, ", "))
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
