package android

import (
	"database/sql"
	"time"
)

const DefaultAndroidPolicyID = 1

type SignupDetails struct {
	Url  string
	Name string
}

type Enterprise struct {
	ID           uint   `db:"id"`
	EnterpriseID string `db:"enterprise_id"`
}

func (e Enterprise) Name() string {
	return "enterprises/" + e.EnterpriseID
}

func (e Enterprise) IsValid() bool {
	return e.EnterpriseID != ""
}

func (e Enterprise) AuthzType() string {
	return "android_enterprise"
}

type EnterpriseDetails struct {
	Enterprise
	SignupName  string `db:"signup_name"`
	SignupToken string `db:"signup_token"`
	TopicID     string `db:"pubsub_topic_id"`
	UserID      uint   `db:"user_id"`
}

type EnrollmentToken struct {
	EnrollmentToken  string `json:"android_enrollment_token"`
	EnrollmentURL    string `json:"android_enrollment_url"`
	EnrollmentQRCode string `json:"android_enrollment_qr_code"`
}

type Device struct {
	ID                   uint       `db:"id"`
	HostID               uint       `db:"host_id"`
	DeviceID             string     `db:"device_id"`
	EnterpriseSpecificID *string    `db:"enterprise_specific_id"`
	LastPolicySyncTime   *time.Time `db:"last_policy_sync_time"`
	AppliedPolicyID      *string    `db:"applied_policy_id"`
	AppliedPolicyVersion *int64     `db:"applied_policy_version"`
}

type AgentManagedConfiguration struct {
	ServerURL              string                     `json:"server_url"`
	HostUUID               string                     `json:"host_uuid"`
	EnrollSecret           string                     `json:"enroll_secret"`
	CertificateTemplateIDs []AgentCertificateTemplate `json:"certificate_templates,omitempty"`
}

type AgentCertificateTemplate struct {
	ID        uint   `json:"id"`
	Status    string `json:"status"`
	Operation string `json:"operation"`
	UUID      string `json:"uuid"`
}

// MDMAndroidPolicyRequest represents a request made to the Android Management
// API (AMAPI) to patch the policy or the device (as made by
// androidsvc.ReconcileProfiles).
type MDMAndroidPolicyRequest struct {
	RequestUUID          string           `db:"request_uuid"`
	RequestName          string           `db:"request_name"`
	PolicyID             string           `db:"policy_id"`
	Payload              []byte           `db:"payload"`
	StatusCode           int              `db:"status_code"`
	ErrorDetails         sql.Null[string] `db:"error_details"`
	AppliedPolicyVersion sql.Null[int64]  `db:"applied_policy_version"`
	PolicyVersion        sql.Null[int64]  `db:"policy_version"`
}

const AppStatusAvailable = "AVAILABLE"

// MDMAndroidCommand represents a single AMAPI command Fleet issued via
// EnterprisesDevicesService.IssueCommand (Lock, Wipe, Clear passcode). One row is inserted at
// issue time and updated by the Pub/Sub COMMAND handler when the device acks or AMAPI rejects.
// CommandUUID is the Fleet-generated identifier that host_mdm_actions.{lock_ref, wipe_ref} points
// to for Android hosts; OperationName is the AMAPI-assigned operation name used to correlate
// Pub/Sub notifications back to the originating command.
type MDMAndroidCommand struct {
	CommandUUID   string           `db:"command_uuid"`
	HostUUID      string           `db:"host_uuid"`
	OperationName string           `db:"operation_name"`
	CommandType   string           `db:"command_type"`
	Status        string           `db:"status"`
	ErrorCode     sql.Null[string] `db:"error_code"`
	ErrorMessage  sql.Null[string] `db:"error_message"`
	CreatedAt     time.Time        `db:"created_at"`
	UpdatedAt     time.Time        `db:"updated_at"`
}

// MDMAndroidCommandType is the AMAPI command type for an MDMAndroidCommand row. Values are the
// strings AMAPI uses on the wire (Command.type), so they double as the value we send in
// IssueCommand and the value we read back from Pub/Sub COMMAND notifications.
type MDMAndroidCommandType string

const (
	MDMAndroidCommandTypeLock          MDMAndroidCommandType = "LOCK"
	MDMAndroidCommandTypeResetPassword MDMAndroidCommandType = "RESET_PASSWORD"
	MDMAndroidCommandTypeWipe          MDMAndroidCommandType = "WIPE"
)

// MDMAndroidCommandStatus is the lifecycle state of an MDMAndroidCommand row.
type MDMAndroidCommandStatus string

const (
	// MDMAndroidCommandStatusPending — Fleet has called IssueCommand and AMAPI accepted, but the
	// Pub/Sub COMMAND notification with the device-side result has not yet arrived.
	MDMAndroidCommandStatusPending MDMAndroidCommandStatus = "pending"
	// MDMAndroidCommandStatusAcknowledged — Pub/Sub COMMAND notification arrived and the device
	// successfully executed the command (no AMAPI error_code).
	//
	// MUST match the literal value of fleet.AndroidMDMCommandStatusAcknowledged; the constants
	// are intentionally duplicated to avoid a server/mdm/android -> server/fleet import (server/fleet
	// already imports server/mdm/android). A divergence would silently break IsLocked/IsWiped for
	// android hosts; the assertion in TestAndroidCommandStatusAcknowledgedStringMatches guards
	// against it.
	MDMAndroidCommandStatusAcknowledged MDMAndroidCommandStatus = "acknowledged"
	// MDMAndroidCommandStatusError — Pub/Sub COMMAND notification arrived with a non-empty
	// AMAPI error_code (e.g. UNSUPPORTED, API_LEVEL, INVALID_VALUE).
	MDMAndroidCommandStatusError MDMAndroidCommandStatus = "error"
)
