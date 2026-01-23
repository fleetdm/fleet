package fleet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
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
	"kioskCustomLauncherEnabled":    `Android configuration profile can't include "kioskCustomLauncherEnabled" setting. Currently, only personal hosts are supported.`,
	"kioskCustomization":            `Android configuration profile can't include "kioskCustomization" setting. Currently, only personal hosts are supported.`,
	"persistentPreferredActivities": `Android configuration profile can't include "persistentPreferredActivities" setting. Currently, only personal hosts are supported.`,
	"setupActions":                  `Android configuration profile can't include "setupActions" setting. Currently, setup experience customization isn't supported.`,
	"encryptionPolicy":              `Android configuration profile can't include "encryptionPolicy" setting. Currently, disk encryption isn't supported.`,
}

// AndroidPremiumOnlyJSONKeys are keys that may not be included in user-provided Android
// configuration profiles for non-Premium licenses and associated error messages when they are included
var AndroidPremiumOnlyJSONKeys = map[string]string{
	"systemUpdate": `Android OS updates ("systemUpdate") is Fleet Premium only.`,
}

func (m *MDMAndroidConfigProfile) ValidateUserProvided(isPremium bool) error {
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

		if !isPremium {
			if errMsg, ok := AndroidPremiumOnlyJSONKeys[key]; ok {
				return errors.New(errMsg)
			}
		}

		if !IsAndroidPolicyFieldValid(key) {
			return fmt.Errorf("Invalid JSON payload. Unknown key %q", key)
		}
	}

	if err := json.Unmarshal(m.RawJSON, &androidmanagement.Policy{}); err != nil {
		return parseAndroidProfileValidationError(err)
	}

	return nil
}

func parseAndroidProfileValidationError(err error) error {
	var typeErr *json.UnmarshalTypeError

	// Check for type mismatches (e.g., array where object expected)
	if errors.As(err, &typeErr) {
		fieldPath := typeErr.Field
		if fieldPath == "" {
			fieldPath = "<root>"
		}
		return fmt.Errorf("Invalid JSON payload. %q format is wrong.", fieldPath)
	}

	// Fallback for any other unexpected errors
	return errors.New("Invalid JSON payload.")
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
		Status:        p.Status.StringPtr(),
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

var (
	policyFieldsCache map[string]bool
	policyFieldsOnce  sync.Once
)

// Initialize the cache once, lazily, with only JSON tag names.
// Since we take in the JSON value.
func initPolicyFieldsCache() {
	policyFieldsCache = make(map[string]bool)
	policyType := reflect.TypeOf(androidmanagement.Policy{})

	for i := 0; i < policyType.NumField(); i++ {
		field := policyType.Field(i)

		// Add JSON tag name if it exists
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			tagName := strings.Split(jsonTag, ",")[0]
			if tagName != "" && tagName != "-" {
				policyFieldsCache[tagName] = true
			}
		}
	}
}

// Fast lookup using cached field names
func IsAndroidPolicyFieldValid(fieldName string) bool {
	policyFieldsOnce.Do(initPolicyFieldsCache)
	return policyFieldsCache[fieldName]
}

var validAndroidWorkProfileWidgets = map[string]struct{}{
	"WORK_PROFILE_WIDGETS_UNSPECIFIED": {},
	"WORK_PROFILE_WIDGETS_ALLOWED":     {},
	"WORK_PROFILE_WIDGETS_DISALLOWED":  {},
}

// ValidateAndroidAppConfiguration validates Android app configuration JSON.
// Configuration must be valid JSON with only "managedConfiguration" and/or
// "workProfileWidgets" as top-level keys. Empty configuration is not allowed.
func ValidateAndroidAppConfiguration(config json.RawMessage) error {
	if len(config) == 0 {
		return &BadRequestError{
			Message: "Couldn't update configuration. Invalid JSON.",
		}
	}

	type androidAppConfig struct {
		ManagedConfiguration json.RawMessage `json:"managedConfiguration"`
		WorkProfileWidgets   string          `json:"workProfileWidgets"`
	}

	var cfg androidAppConfig
	if err := JSONStrictDecode(bytes.NewReader(config), &cfg); err != nil {
		if strings.Contains(err.Error(), "unknown field") {
			return &BadRequestError{
				Message: `Couldn't update configuration. Only "managedConfiguration" and "workProfileWidgets" are supported as top-level keys.`,
			}
		}

		return &BadRequestError{
			Message: "Couldn't update configuration. Invalid JSON.",
		}
	}

	if _, validVal := validAndroidWorkProfileWidgets[cfg.WorkProfileWidgets]; cfg.WorkProfileWidgets != "" && !validVal {
		return &BadRequestError{Message: fmt.Sprintf(`Couldn't update configuration. "%s" is not a supported value for "workProfileWidget".`, cfg.WorkProfileWidgets)}
	}

	return nil
}
