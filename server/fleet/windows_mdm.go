package fleet

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
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
func (m *MDMWindowsConfigProfile) ValidateUserProvided() error {
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
	var inLocURI bool

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

		case xml.StartElement:
			switch t.Name.Local {
			case "Replace", "Add":
				inValidNode = true
			case "LocURI":
				if !inValidNode {
					return errors.New("Windows configuration profiles can only have <Replace> or <Add> top level elements.")
				}
				inLocURI = true

			default:
				if !inValidNode {
					return errors.New("Windows configuration profiles can only have <Replace> or <Add> top level elements.")
				}
			}

		case xml.EndElement:
			switch t.Name.Local {
			case "Replace", "Add":
				inValidNode = false
			case "LocURI":
				inLocURI = false
			}

		case xml.CharData:
			if inLocURI {
				if err := validateFleetProvidedLocURI(string(t)); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

var fleetProvidedLocURIValidationMap = map[string][2]string{
	syncml.FleetBitLockerTargetLocURI: {"BitLocker", "mdm.enable_disk_encryption"},
	syncml.FleetOSUpdateTargetLocURI:  {"Windows updates", "mdm.windows_updates"},
}

func validateFleetProvidedLocURI(locURI string) error {
	sanitizedLocURI := strings.TrimSpace(locURI)
	for fleetLocURI, errHints := range fleetProvidedLocURIValidationMap {
		if strings.Contains(sanitizedLocURI, fleetLocURI) {
			return fmt.Errorf("Custom configuration profiles can't include %s settings. To control these settings, use the %s option.", errHints[0], errHints[1])
		}
	}

	return nil
}

type MDMWindowsProfilePayload struct {
	ProfileUUID   string             `db:"profile_uuid"`
	ProfileName   string             `db:"profile_name"`
	HostUUID      string             `db:"host_uuid"`
	Status        *MDMDeliveryStatus `db:"status" json:"status"`
	OperationType MDMOperationType   `db:"operation_type"`
	Detail        string             `db:"detail"`
	CommandUUID   string             `db:"command_uuid"`
	Retries       int                `db:"retries"`
}

func (p MDMWindowsProfilePayload) Equal(other MDMWindowsProfilePayload) bool {
	statusEqual := p.Status == nil && other.Status == nil || p.Status != nil && other.Status != nil && *p.Status == *other.Status
	return statusEqual &&
		p.ProfileUUID == other.ProfileUUID &&
		p.HostUUID == other.HostUUID &&
		p.ProfileName == other.ProfileName &&
		p.OperationType == other.OperationType &&
		p.Detail == other.Detail &&
		p.CommandUUID == other.CommandUUID &&
		p.Retries == other.Retries
}

type MDMWindowsBulkUpsertHostProfilePayload struct {
	ProfileUUID   string
	ProfileName   string
	HostUUID      string
	CommandUUID   string
	OperationType MDMOperationType
	Status        *MDMDeliveryStatus
	Detail        string
}
