package android

import (
	"time"
)

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
	AndroidPolicyID      *uint      `db:"android_policy_id"`
	LastPolicySyncTime   *time.Time `db:"last_policy_sync_time"`
}
