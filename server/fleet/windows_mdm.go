package fleet

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
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
	ProfileUUID string    `db:"profile_uuid" json:"profile_uuid"`
	TeamID      *uint     `db:"team_id" json:"team_id"`
	Name        string    `db:"name" json:"name"`
	SyncML      []byte    `db:"syncml" json:"-"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// ValidateUserProvided ensures that the SyncML content in the profile is valid
// for Windows.
//
// It checks that all top-level elements are <Replace> and none of the <LocURI>
// elements within <Target> are reserved URIs.
//
// Returns an error if these conditions are not met.
func (m *MDMWindowsConfigProfile) ValidateUserProvided() error {
	if len(bytes.TrimSpace(m.SyncML)) == 0 {
		return errors.New("The file should include valid XML.")
	}
	fleetNames := mdm.FleetReservedProfileNames()
	if _, ok := fleetNames[m.Name]; ok {
		return fmt.Errorf("Profile name %q is not allowed.", m.Name)
	}

	var validator struct {
		SyncBody
		NonProtocolElements []interface{} `xml:",any,omitempty"`
	}
	wrappedProfile := fmt.Sprintf("<SyncBody>%s</SyncBody>", m.SyncML)
	if err := xml.Unmarshal([]byte(wrappedProfile), &validator); err != nil {
		return fmt.Errorf("The file should include valid XML: %w", err)
	}

	// might be valid XML, but start with something other than <Replace>
	if mdm.GetRawProfilePlatform(m.SyncML) != "windows" {
		return errors.New("Only <Replace> supported as a top level element. Make sure you don't have other top level elements.")
	}

	if len(validator.Add) != 0 ||
		len(validator.Alert) != 0 ||
		len(validator.Atomic) != 0 ||
		len(validator.Delete) != 0 ||
		len(validator.Exec) != 0 ||
		len(validator.Get) != 0 ||
		len(validator.Results) != 0 ||
		len(validator.Status) != 0 ||
		len(validator.NonProtocolElements) != 0 {
		return errors.New("Only <Replace> supported as a top level element. Make sure you don't have other top level elements.")
	}

	for _, cmd := range validator.Replace {
		for _, item := range cmd.Items {
			// intentionally skipping any further validation if we
			// don't get a target per product decision.
			if item.Target == nil {
				continue
			}

			if err := validateFleetProvidedLocURI(*item.Target); err != nil {
				return err
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

type MDMWindowsBulkUpsertHostProfilePayload struct {
	ProfileUUID   string
	ProfileName   string
	HostUUID      string
	CommandUUID   string
	OperationType MDMOperationType
	Status        *MDMDeliveryStatus
	Detail        string
}
