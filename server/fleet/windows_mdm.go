package fleet

import (
	"errors"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/fleetdm/fleet/v4/server/mdm"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
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
	if mdm.GetRawProfilePlatform(m.SyncML) != "windows" {
		return errors.New("Only <Replace> supported as a top level element. Make sure you don’t have other top level elements.")
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(m.SyncML); err != nil {
		return err
	}

	for _, element := range doc.ChildElements() {
		if element.Tag != CmdReplace {
			return errors.New("Only <Replace> supported as a top level element. Make sure you don’t have other top level elements.")
		}

		for _, target := range element.FindElements("Target") {
			locURI := target.FindElement("LocURI")
			if locURI != nil {
				locURIText := locURI.Text()

				if strings.HasPrefix(locURIText, microsoft_mdm.FleetBitLockerTargetLocURI) {
					return errors.New("Custom configuration profiles can't include BitLocker settings. To control these settings, use the mdm.enable_disk_encryption option.")
				}

				if strings.HasPrefix(locURIText, microsoft_mdm.FleetOSUpdateTargetLocURI) {
					return errors.New("Custom configuration profiles can’t include Windows updates settings. To control these settings, use the mdm.windows_updates option.")
				}

			}
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
