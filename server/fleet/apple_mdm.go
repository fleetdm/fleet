package fleet

import (
	"encoding/json"
)

type MDMAppleEnrollmentPayload struct {
	Name      string           `json:"name"`
	DEPConfig *json.RawMessage `json:"dep_config"`
}

type MDMAppleEnrollment struct {
	// TODO(lucas): Add UpdateCreateTimestamps
	ID        uint             `json:"id" db:"id"`
	Name      string           `json:"name" db:"name"`
	DEPConfig *json.RawMessage `json:"dep_config" db:"dep_config"`
}

func (m MDMAppleEnrollment) AuthzType() string {
	return "mdm_apple_enrollment"
}

type MDMAppleCommandResult struct {
	// ID is the enrollment ID. This should be the same as the device ID.
	ID          string `json:"id" db:"id"`
	CommandUUID string `json:"command_uuid" db:"command_uuid"`
	// Status is the command status. One of Acknowledged, Error, or NotNow.
	Status string `json:"status" db:"status"`
	// Result is the original command result XML plist. If the status is Error, it will include the
	// ErrorChain key with more information.
	Result []byte `json:"result" db:"result"`
}

func (m MDMAppleCommandResult) AuthzType() string {
	return "mdm_apple_command_result"
}

type MDMAppleInstaller struct {
	// TODO(lucas): Add UpdateCreateTimestamps
	ID        uint   `json:"id" db:"id"`
	Name      string `json:"name" db:"name"`
	Size      int64  `json:"size" db:"size"`
	Manifest  string `json:"manifest" db:"manifest"`
	Installer []byte `json:"-" db:"installer"`
	URLToken  string `json:"url_token" db:"url_token"`
}

func (m MDMAppleInstaller) AuthzType() string {
	return "mdm_apple_installer"
}

type MDMAppleDevice struct {
	ID           string `json:"id" db:"id"`
	SerialNumber string `json:"serial_number" db:"serial_number"`
	Enabled      bool   `json:"enabled" db:"enabled"`
}

func (m MDMAppleDevice) AuthzType() string {
	return "mdm_apple_device"
}
