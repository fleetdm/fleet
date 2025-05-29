package android

import (
	"time"

	"github.com/fleetdm/fleet/v4/server/ptr"
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

type Host struct {
	*Device
	ID              uint
	TeamID          *uint
	OSVersion       string
	Build           string
	Memory          int64
	CPUType         string
	HardwareSerial  string
	HardwareModel   string
	HardwareVendor  string
	NodeKey         *string
	DetailUpdatedAt time.Time
	LabelUpdatedAt  time.Time
}

func (h *Host) Platform() string {
	return "android"
}

func (h *Host) DisplayName() string {
	return h.HardwareModel
}

func (h *Host) SetNodeKey(enterpriseSpecificID string) {
	if h.Device == nil {
		return
	}
	h.Device.EnterpriseSpecificID = ptr.String(enterpriseSpecificID)
	// We use node_key as a unique identifier for the host table row.
	// Since this key is used by other hosts, we use a prefix to avoid conflicts.
	hostNodeKey := "android/" + enterpriseSpecificID
	h.NodeKey = &hostNodeKey
}

func (h *Host) IsValid() bool {
	return !(h == nil || h.Device == nil ||
		h.NodeKey == nil || h.Device.EnterpriseSpecificID == nil ||
		*h.NodeKey != "android/"+*h.Device.EnterpriseSpecificID)
}
