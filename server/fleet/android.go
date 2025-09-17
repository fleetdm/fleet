package fleet

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm"
	"google.golang.org/api/androidmanagement/v1"
)

// MDMAndroidConfigProfile represents an Android MDM profile in Fleet. This does not map
// directly to a specific policy in the Android API, rather the policy applied is the
// result of combining all applicable profiles.
type MDMAndroidConfigProfile struct {
	// ProfileUUID is the unique identifier of the configuration profile in
	// Fleet. For Android profiles, it is the letter "g" followed by a uuid.
	ProfileUUID      string                      `db:"profile_uuid" json:"profile_uuid"`
	TeamID           *uint                       `db:"team_id" json:"team_id"`
	Name             string                      `db:"name" json:"name"`
	RawJSON          []byte                      `db:"raw_json" json:"-"`
	AutoIncrement    int64                       `db:"auto_increment" json:"auto_increment"`
	LabelsIncludeAll []ConfigurationProfileLabel `db:"-" json:"labels_include_all,omitempty"`
	LabelsIncludeAny []ConfigurationProfileLabel `db:"-" json:"labels_include_any,omitempty"`
	LabelsExcludeAny []ConfigurationProfileLabel `db:"-" json:"labels_exclude_any,omitempty"`
	CreatedAt        time.Time                   `db:"created_at" json:"created_at"`
	UploadedAt       time.Time                   `db:"uploaded_at" json:"updated_at"` // Difference in DB field name vs JSON is conscious decision to match other platforms
}

// AndroidForbiddenJSONKeys are keys that may not be included in user-provided Android configuration profiles and
// associated error messages when they are included
var AndroidForbiddenJSONKeys = map[string]string{
	"statusReportingSettings":       `Android configuration profile can't include "statusReportingSettings" setting. To get host vitals, use Get host endpoint: https://fleetdm.com/docs/rest-api/rest-api#get-host`,
	"applications":                  `Android configuration profile can't include "applications" setting. Software management is coming soon.`,
	"appFunctions":                  `Android configuration profile can't include "appFunctions" setting. Software management is coming soon.`,
	"playStoreMode":                 `Android configuration profile can't include "playStoreMode" setting. Software management is coming soon.`,
	"installAppsDisabled":           `Android configuration profile can't include "installAppsDisabled" setting. Software management is coming soon.`,
	"uninstallAppsDisabled":         `Android configuration profile can't include "uninstallAppsDisabled" setting. Software management is coming soon.`,
	"blockApplicationsEnabled":      `Android configuration profile can't include "blockApplicationsEnabled" setting. Software management is coming soon.`,
	"appAutoUpdatePolicy":           `Android configuration profile can't include "appAutoUpdatePolicy" setting. Software management is coming soon.`,
	"systemUpdate":                  `Android configuration profile can't include "systemUpdate" setting. OS updates are coming soon.`,
	"kioskCustomLauncherEnabled":    `Android configuration profile can't include "kioskCustomLauncherEnabled" setting. Currently, only personal hosts are supported.`,
	"kioskCustomization":            `Android configuration profile can't include "kioskCustomization" setting. Currently, only personal hosts are supported.`,
	"persistentPreferredActivities": `Android configuration profile can't include "persistentPreferredActivities" setting. Currently, only personal hosts are supported.`,
	"setupActions":                  `Android configuration profile can't include "setupActions" setting. Currently, setup experience customization isn't supported.`,
	"encryptionPolicy":              `Android configuration profile can't include "encryptionPolicy" setting. Currently, disk encryption isn't supported.`,
}

func (m *MDMAndroidConfigProfile) ValidateUserProvided() error {
	if len(bytes.TrimSpace(m.RawJSON)) == 0 {
		return errors.New("The file should include valid JSON.")
	}
	fleetNames := mdm.FleetReservedProfileNames()
	if _, ok := fleetNames[m.Name]; ok {
		return fmt.Errorf("Profile name %q is not allowed.", m.Name)
	}
	type jsonObj map[string]interface{}
	var profileKeyMap jsonObj
	err := json.Unmarshal(m.RawJSON, &profileKeyMap)
	if err != nil {
		// TODO invalid profile err
		return err
	}
	if len(profileKeyMap) == 0 {
		return errors.New("JSON profile is empty")
	}
	for key := range profileKeyMap {
		if errMsg, ok := AndroidForbiddenJSONKeys[key]; ok {
			return errors.New(errMsg)
		}
	}

	return nil
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

type MDMAndroidProfilePayload struct {
	HostUUID                string             `db:"host_uuid"`
	Status                  *MDMDeliveryStatus `db:"status"`
	OperationType           MDMOperationType   `db:"operation_type"`
	Detail                  string             `db:"detail"`
	ProfileUUID             string             `db:"profile_uuid"`
	ProfileName             string             `db:"profile_name"`
	PolicyRequestUUID       *string            `db:"policy_request_uuid"`
	DeviceRequestUUID       *string            `db:"device_request_uuid"`
	RequestFailCount        int                `db:"request_fail_count"`
	IncludedInPolicyVersion *int               `db:"included_in_policy_version"`
}

// HostMDMAndroidProfile represents the status of an MDM profile for a Android host.
type HostMDMAndroidProfile struct {
	HostUUID      string             `db:"host_uuid" json:"host_uuid"`
	ProfileUUID   string             `db:"profile_uuid" json:"profile_uuid"`
	Name          string             `db:"name" json:"name"`
	Status        *MDMDeliveryStatus `db:"status" json:"status"`
	OperationType MDMOperationType   `db:"operation_type" json:"operation_type"`
	Detail        string             `db:"detail" json:"detail"`
}

func (p HostMDMAndroidProfile) ToHostMDMProfile() HostMDMProfile {
	return HostMDMProfile{
		HostUUID:      p.HostUUID,
		ProfileUUID:   p.ProfileUUID,
		Name:          p.Name,
		Identifier:    "",
		Status:        p.Status,
		OperationType: p.OperationType,
		Detail:        p.Detail,
		Platform:      "android",
	}
}

type AndroidPolicyRequestPayload struct {
	Policy   *androidmanagement.Policy           `json:"policy"`
	Metadata AndroidPolicyRequestPayloadMetadata `json:"metadata"`
}

type AndroidPolicyRequestPayloadMetadata struct {
	SettingsOrigin map[string]string `json:"settings_origin"` // Map of policy setting name, to profile uuid.
}
