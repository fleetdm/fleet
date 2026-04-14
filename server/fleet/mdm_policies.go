package fleet

import "time"

// MDMPolicyCheckOperator defines comparison operators for MDM policy checks.
type MDMPolicyCheckOperator string

const (
	MDMPolicyCheckEq          MDMPolicyCheckOperator = "eq"
	MDMPolicyCheckNeq         MDMPolicyCheckOperator = "neq"
	MDMPolicyCheckGt          MDMPolicyCheckOperator = "gt"
	MDMPolicyCheckLt          MDMPolicyCheckOperator = "lt"
	MDMPolicyCheckGte         MDMPolicyCheckOperator = "gte"
	MDMPolicyCheckLte         MDMPolicyCheckOperator = "lte"
	MDMPolicyCheckContains    MDMPolicyCheckOperator = "contains"
	MDMPolicyCheckNotContains MDMPolicyCheckOperator = "not_contains"
	MDMPolicyCheckVersionGte  MDMPolicyCheckOperator = "version_gte"
	MDMPolicyCheckVersionLte  MDMPolicyCheckOperator = "version_lte"
	MDMPolicyCheckExists      MDMPolicyCheckOperator = "exists"
	MDMPolicyCheckNotExists   MDMPolicyCheckOperator = "not_exists"
)

// MDMPolicyCheckSource identifies which MDM query provides the data for a check.
type MDMPolicyCheckSource string

const (
	MDMPolicySourceDeviceInformation        MDMPolicyCheckSource = "DeviceInformation"
	MDMPolicySourceSecurityInfo             MDMPolicyCheckSource = "SecurityInfo"
	MDMPolicySourceInstalledApplicationList MDMPolicyCheckSource = "InstalledApplicationList"
)

// MDMPolicyCheck is a single condition within an MDM policy definition.
type MDMPolicyCheck struct {
	Field    string                 `json:"field"`
	Operator MDMPolicyCheckOperator `json:"operator"`
	Expected string                 `json:"expected"`
	Source   MDMPolicyCheckSource   `json:"source"`
}

// MDMPolicyDefinition is the set of checks that comprise an MDM policy.
type MDMPolicyDefinition struct {
	Checks []MDMPolicyCheck `json:"checks"`
}

// DeviceStateEntry holds a single piece of device state data collected from MDM.
type DeviceStateEntry struct {
	Value      string    `json:"value"`
	Source     string    `json:"source"`
	ObservedAt time.Time `json:"observed_at"`
}

// DeviceStateStore is the interface for storing and retrieving device state data.
// Phase 1 uses in-memory implementation; Phase 2 adds MySQL-backed implementation.
type DeviceStateStore interface {
	UpdateDeviceState(hostUUID string, entries map[string]DeviceStateEntry) error
	GetDeviceState(hostUUID string) (map[string]DeviceStateEntry, error)
}

// MDMPolicyResult is the outcome of evaluating an MDM policy against device state.
type MDMPolicyResult struct {
	HostID    uint      `json:"host_id"`
	PolicyID  uint      `json:"policy_id"`
	Passes    bool      `json:"passes"`
	Err       error     `json:"-"`
	Timestamp time.Time `json:"timestamp"`
}
