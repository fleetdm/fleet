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
	EnrollmentToken string `json:"android_enrollment_token"`
	EnrollmentURL   string `json:"android_enrollment_url"`
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
