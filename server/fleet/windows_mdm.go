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

	dec := xml.NewDecoder(bytes.NewReader(m.SyncML))
	// use strict mode to check for a variety of common mistakes like
	// unclosed tags, etc.
	dec.Strict = true

	// keep track of certain elements to perform Fleet-validations.
	//
	// NOTE: since we're only checking for well-formedness
	// we don't need to validate the required nesting
	// structure (Target>Item>LocURI) so we don't need to track all the tags.
	var inValidNode bool
	var inExec bool
	var inLocURI bool
	var inComment bool

	windowSCEPProfileValidator := newWindowsSCEPProfileValidator()

	for {
		tok, err := dec.Token()
		if err != nil {
			if err != io.EOF {
				return fmt.Errorf("The file should include valid XML: %w", err)
			}
			// EOF means no more tokens to process
			break
		}

		switch t := tok.(type) {
		// no processing instructions allowed (<?target inst?>)
		// see #16316 for details
		case xml.ProcInst:
			return errors.New("The file should include valid XML: processing instructions are not allowed.")

		case xml.Comment:
			inComment = true
			continue

		case xml.StartElement:
			// Top-level comments should be followed by <Replace> or <Add> elements
			if inComment {
				if !inValidNode && t.Name.Local != "Replace" && t.Name.Local != "Add" && t.Name.Local != "Exec" {
					return errors.New("Windows configuration profiles can only have <Replace>, <Add> or <Exec> top level elements after comments")
				}
				inValidNode = true
				inComment = false
			}

			switch t.Name.Local {
			case "Replace", "Add":
				inValidNode = true
			case "Exec":
				inValidNode = true
				inExec = true
			case "LocURI":
				if !inValidNode {
					return errors.New("Windows configuration profiles can only have <Replace>, <Add> or <Exec> top level elements.")
				}
				inLocURI = true

			default:
				if !inValidNode {
					return errors.New("Windows configuration profiles can only have <Replace>, <Add> or <Exec> top level elements.")
				}
			}

		case xml.EndElement:
			switch t.Name.Local {
			case "Replace", "Add":
				inValidNode = false
			case "Exec":
				inValidNode = false
				inExec = false
			case "LocURI":
				inLocURI = false
			}

		case xml.CharData:
			if inLocURI {
				if inExec {
					if err := windowSCEPProfileValidator.validateExecLocURI(string(t)); err != nil {
						return err
					}
					continue
				}

				if err := windowSCEPProfileValidator.validateLocURI(string(t)); err != nil {
					return err
				}

				if err := validateFleetProvidedLocURI(string(t), enableCustomOSUpdates); err != nil {
					return err
				}
			}
		}
	}

	if err := windowSCEPProfileValidator.finalizeValidation(); err != nil {
		return err
	}

	return nil
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
var validSCEPProfileLocURIs = slices.Concat([]string{
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
}, requiredSCEPProfileLocURIs)

var requiredSCEPProfileLocURIs = []string{
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

var validExecSCEPProfileLocURIs = []string{
	fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s/Install/Enroll", FleetVarSCEPWindowsCertificateID.WithPrefix()),
}

type windowsSCEPProfileValidator struct {
	totalLocURIs     int
	totalExecLocURIs int
	foundLocURIs     map[string]bool
	foundExecLocURIs map[string]bool
}

func newWindowsSCEPProfileValidator() *windowsSCEPProfileValidator {
	return &windowsSCEPProfileValidator{
		foundLocURIs:     make(map[string]bool),
		foundExecLocURIs: make(map[string]bool),
	}
}

func (v *windowsSCEPProfileValidator) isSCEPProfile() bool {
	return len(v.foundLocURIs) > 0 || (v.totalExecLocURIs > 0 && len(v.foundExecLocURIs) > 0)
}

func (v *windowsSCEPProfileValidator) validateLocURI(locURI string) error {
	sanitizedLocURI := strings.TrimSpace(locURI)

	// If we see a LocURI with SCEP prefix, but no Fleet Var we fail early.
	if v.isSCEPLocURIWithoutFleetVar(sanitizedLocURI) {
		return fmt.Errorf("You must use %q after \"./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/\".", FleetVarSCEPWindowsCertificateID.WithPrefix())
	}

	if slices.Contains(validSCEPProfileLocURIs, sanitizedLocURI) {
		v.foundLocURIs[sanitizedLocURI] = true
	}

	v.totalLocURIs++
	return nil
}

func (v *windowsSCEPProfileValidator) validateExecLocURI(locURI string) error {
	sanitizedLocURI := strings.TrimSpace(locURI)

	// If we see a LocURI with SCEP prefix, but no Fleet Var we fail early.
	if v.isSCEPLocURIWithoutFleetVar(sanitizedLocURI) {
		return fmt.Errorf("You must use %q after \"./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/\".", FleetVarSCEPWindowsCertificateID.WithPrefix())
	}

	if slices.Contains(validExecSCEPProfileLocURIs, sanitizedLocURI) {
		v.foundExecLocURIs[sanitizedLocURI] = true
	}

	v.totalExecLocURIs++
	return nil
}

// isSCEPLocURIWithoutFleetVar checks that the provided locURI starts with the SCEP prefix
// and that it includes the required Fleet Var for SCEP Windows Certificate ID.
// Skips any locURI that does not start with the SCEP prefix.
func (v windowsSCEPProfileValidator) isSCEPLocURIWithoutFleetVar(locURI string) bool {
	if strings.HasPrefix(locURI, "./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/") &&
		!strings.HasPrefix(locURI, fmt.Sprintf("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/%s", FleetVarSCEPWindowsCertificateID.WithPrefix())) {
		return true
	}
	return false
}

func (v *windowsSCEPProfileValidator) finalizeValidation() error {
	if !v.isSCEPProfile() {
		// Cheeky validation here, to only allow Exec elements in SCEP profiles.
		if v.totalExecLocURIs > 0 {
			return errors.New("Only SCEP profiles can include <Exec> elements.")
		}
		return nil // Not a SCEP profile, nothing to validate here.
	}

	// Verify that we do not have any non-scep loc URIs present
	if v.totalLocURIs != len(v.foundLocURIs) {
		return errors.New("Only options that have <LocURI> starting with \"./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/\" can be added to SCEP profile.")
	}

	// Check that at least one Exec LocURI is present and it matches the only one we have in the array.
	if len(v.foundExecLocURIs) != 1 && !v.foundExecLocURIs[validExecSCEPProfileLocURIs[0]] {
		return errors.New("Couldn't add. \"./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Enroll\" must be included within <Exec>. Please add and try again.")
	}

	// Check that all required LocURIs are present
	for _, requiredLocURI := range requiredSCEPProfileLocURIs {
		if !v.foundLocURIs[requiredLocURI] {
			return fmt.Errorf("%q is missing. Please add and try again", requiredLocURI)
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
