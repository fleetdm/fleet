package fleet

import (
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
)

// WindowsEnforcementProfile represents a Windows enforcement policy stored in
// the database. Each profile corresponds to one policy YAML file per team.
// The profile UUID is prefixed with "e" (enforcement) to distinguish it from
// "w" (Windows MDM) and "a" (Apple MDM) profiles.
type WindowsEnforcementProfile struct {
	ProfileUUID string    `db:"profile_uuid" json:"profile_uuid"`
	TeamID      *uint     `db:"team_id" json:"team_id"`
	Name        string    `db:"name" json:"name"`
	RawPolicy   []byte    `db:"raw_policy" json:"-"`
	Checksum    []byte    `db:"checksum" json:"-"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// HostWindowsEnforcement tracks the enforcement status of a profile on a
// specific host, following the same status model as MDM profile delivery.
type HostWindowsEnforcement struct {
	HostUUID      string             `db:"host_uuid"`
	ProfileUUID   string             `db:"profile_uuid"`
	Name          string             `db:"name"`
	Status        *MDMDeliveryStatus `db:"status" json:"status"`
	OperationType MDMOperationType   `db:"operation_type"`
	Detail        string             `db:"detail"`
	Retries       uint               `db:"retries"`
}

// WindowsEnforcementSettings holds the enforcement configuration for a team or
// the global (no-team) scope, following the same pattern as WindowsSettings.
type WindowsEnforcementSettings struct {
	CustomSettings optjson.Slice[MDMProfileSpec] `json:"custom_settings"`
}
