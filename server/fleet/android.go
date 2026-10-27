package fleet

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/platform/jsondecode"
	"github.com/fleetdm/fleet/v4/server/variables"
	"google.golang.org/api/androidmanagement/v1"
)

const AndroidWebAppPrefix = "com.google.enterprise.webapp"

// MDMAndroidConfigProfile represents an Android MDM profile in Fleet. This does not map
// directly to a specific policy in the Android API, rather the policy applied is the
// result of combining all applicable profiles.
type MDMAndroidConfigProfile struct {
	// ProfileUUID is the unique identifier of the configuration profile in
	// Fleet. For Android profiles, it is the letter "g" followed by a uuid.
	ProfileUUID      string                      `db:"profile_uuid" json:"profile_uuid"`
	TeamID           *uint                       `db:"team_id" json:"fleet_id"`
	Name             string                      `db:"name" json:"name"`
	Description      string                      `db:"description" json:"description"`
	RawJSON          []byte                      `db:"raw_json" json:"-"`
	AutoIncrement    int64                       `db:"auto_increment" json:"auto_increment"`
	Hidden           bool                        `db:"hidden" json:"hidden"`
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
	"applications":                  androidSoftwareKeyErrPremium("applications"),
	"appFunctions":                  androidSoftwareKeyErrPremium("appFunctions"),
	"playStoreMode":                 androidSoftwareKeyErrPremium("playStoreMode"),
	"installAppsDisabled":           androidSoftwareKeyErrPremium("installAppsDisabled"),
	"uninstallAppsDisabled":         androidSoftwareKeyErrPremium("uninstallAppsDisabled"),
	"blockApplicationsEnabled":      androidSoftwareKeyErrPremium("blockApplicationsEnabled"),
	"appAutoUpdatePolicy":           androidSoftwareKeyErrPremium("appAutoUpdatePolicy"),
	"kioskCustomLauncherEnabled":    `Android configuration profile can't include "kioskCustomLauncherEnabled" setting. Currently, only personal hosts are supported.`,
	"kioskCustomization":            `Android configuration profile can't include "kioskCustomization" setting. Currently, only personal hosts are supported.`,
	"persistentPreferredActivities": `Android configuration profile can't include "persistentPreferredActivities" setting. Currently, only personal hosts are supported.`,
	"setupActions":                  `Android configuration profile can't include "setupActions" setting. Currently, setup experience customization isn't supported.`,
	"encryptionPolicy":              `Android configuration profile can't include "encryptionPolicy" setting. Currently, disk encryption isn't supported.`,
}

// AndroidSoftwareJSONKeysFree are the software management keys from AndroidForbiddenJSONKeys with the
// error messages shown instead for non-Premium licenses, since software management is Premium only.
var AndroidSoftwareJSONKeysFree = map[string]string{
	"applications":             androidSoftwareKeyErrFree("applications"),
	"appFunctions":             androidSoftwareKeyErrFree("appFunctions"),
	"playStoreMode":            androidSoftwareKeyErrFree("playStoreMode"),
	"installAppsDisabled":      androidSoftwareKeyErrFree("installAppsDisabled"),
	"uninstallAppsDisabled":    androidSoftwareKeyErrFree("uninstallAppsDisabled"),
	"blockApplicationsEnabled": androidSoftwareKeyErrFree("blockApplicationsEnabled"),
	"appAutoUpdatePolicy":      androidSoftwareKeyErrFree("appAutoUpdatePolicy"),
}

func androidSoftwareKeyErrPremium(key string) string {
	return fmt.Sprintf(`Android configuration profile can't include %q setting. To manage Android software, add Play Store apps in Fleet. Learn more: https://fleetdm.com/learn-more-about/android-software`, key)
}

func androidSoftwareKeyErrFree(key string) string {
	return fmt.Sprintf(`Android configuration profile can't include %q setting. Android software management is Fleet Premium only.`, key)
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
	if strings.TrimSpace(m.Name) == "" {
		return errors.New("Profile name can't be empty.")
	}
	fleetNames := mdm.FleetReservedProfileNames()
	if _, ok := fleetNames[m.Name]; ok {
		return fmt.Errorf("Profile name %q is not allowed.", m.Name)
	}
	type jsonObj map[string]any
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
		if !isPremium {
			if errMsg, ok := AndroidSoftwareJSONKeysFree[key]; ok {
				return errors.New(errMsg)
			}
		}

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

	// This only checks that every value has the right type.
	if err := jsondecode.Unmarshal(m.RawJSON, &androidmanagement.Policy{}); err != nil {
		return parseAndroidProfileValidationError(err)
	}

	if err := validateAndroidProfileFleetVariables(m.RawJSON, profileKeyMap); err != nil {
		return err
	}

	return nil
}

func parseAndroidProfileValidationError(err error) error {
	// Check for type mismatches (e.g., array where object expected)
	if jsondecode.IsTypeError(err) {
		fieldPath := jsondecode.FieldPath(err)
		if fieldPath == "" {
			fieldPath = "<root>"
		}
		return fmt.Errorf("Invalid JSON payload. %q format is wrong.", fieldPath)
	}

	// Fallback for any other unexpected errors
	return errors.New("Invalid JSON payload.")
}

func validateAndroidProfileFleetVariables(rawJSON []byte, decoded map[string]any) error {
	contents := string(rawJSON)

	// Malformed vital refs (e.g. a typo like $FLEET_HOST_VITAL_asset_tag) are
	// rejected here rather than left solely to the service-layer existence
	// check (ds.ValidateReferencedCustomHostVitals), so this function doesn't
	// depend on callers always pairing it with that check to catch a malformed
	// reference — mirrors the same rejection in ValidateAndroidAppConfiguration.
	if malformed := ContainsMalformedCustomHostVitalRefs(contents); len(malformed) > 0 {
		return errors.New((&InvalidCustomHostVitalRefError{Refs: malformed}).Error())
	}

	// Custom host vitals ($FLEET_HOST_VITAL_<id>) are validated for existence
	// at the service layer via ds.ValidateReferencedCustomHostVitals, same as
	// Apple/Windows profiles. Here we only enforce that the token sits inside a
	// JSON string value, mirroring the $FLEET_VAR_* check below, since a token
	// used as a JSON key would corrupt the profile structure once substituted
	// at delivery time.
	vitalIDs := FindCustomHostVitalIDs(contents)

	varNames := variables.Find(contents)
	if len(varNames) == 0 && len(vitalIDs) == 0 {
		return nil
	}

	if len(varNames) > 0 {
		if name := FindUnsupportedAndroidFleetVar(contents); name != "" {
			return fmt.Errorf("Unsupported Fleet variable $FLEET_VAR_%s.", name)
		}

		keyVars := make(map[string]struct{})
		stringVars := make(map[string]struct{})
		walkJSONForVars(decoded, variables.Find, keyVars, stringVars)
		for _, name := range varNames {
			if _, inKey := keyVars[name]; inKey {
				return fmt.Errorf("Fleet variable $FLEET_VAR_%s must be inside a JSON string value.", name)
			}
			if _, inStr := stringVars[name]; !inStr {
				return fmt.Errorf("Fleet variable $FLEET_VAR_%s must be inside a JSON string value.", name)
			}
		}
	}

	if len(vitalIDs) > 0 {
		vitalKeyVars := make(map[string]struct{})
		vitalStringVars := make(map[string]struct{})
		walkJSONForVars(decoded, findCustomHostVitalTokens, vitalKeyVars, vitalStringVars)
		for _, id := range vitalIDs {
			token := fmt.Sprintf("%s%d", CustomHostVitalPrefix, id)
			if _, inKey := vitalKeyVars[token]; inKey {
				return fmt.Errorf("Custom host vital $%s must be inside a JSON string value.", token)
			}
			if _, inStr := vitalStringVars[token]; !inStr {
				return fmt.Errorf("Custom host vital $%s must be inside a JSON string value.", token)
			}
		}
	}

	return nil
}

// findCustomHostVitalTokens returns custom host vital token strings without
// the leading '$' (e.g. "FLEET_HOST_VITAL_7"). Unlike variables.Find, which
// strips the entire "$FLEET_VAR_" prefix down to a bare name (e.g.
// "HOST_UUID"), this keeps the "FLEET_HOST_VITAL_" prefix in each token. It
// matches the func(string) []string shape walkJSONForVars expects, so it can
// drive the same walk variables.Find does for $FLEET_VAR_*.
func findCustomHostVitalTokens(s string) []string {
	ids := FindCustomHostVitalIDs(s)
	tokens := make([]string, len(ids))
	for i, id := range ids {
		tokens[i] = fmt.Sprintf("%s%d", CustomHostVitalPrefix, id)
	}
	return tokens
}

// walkJSONForVars recursively walks a decoded JSON value and collects
// variable-like tokens found in string values and in map keys separately.
// find extracts tokens from a raw string ($FLEET_VAR_* names or
// $FLEET_HOST_VITAL_<id> tokens).
func walkJSONForVars(v any, find func(string) []string, keyVars, stringVars map[string]struct{}) {
	switch t := v.(type) {
	case string:
		for _, name := range find(t) {
			stringVars[name] = struct{}{}
		}
	case map[string]any:
		for k, val := range t {
			for _, name := range find(k) {
				keyVars[name] = struct{}{}
			}
			walkJSONForVars(val, find, keyVars, stringVars)
		}
	case []any:
		for _, val := range t {
			walkJSONForVars(val, find, keyVars, stringVars)
		}
	}
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
	Checksum                []byte             `db:"checksum"`
	LastErrorDetails        string             `db:"last_error_details"`
	CanReverify             bool               `db:"can_reverify"`
}

// HostMDMAndroidProfile represents the status of an MDM profile for a Android host.
type HostMDMAndroidProfile struct {
	HostUUID      string             `db:"host_uuid" json:"host_uuid"`
	ProfileUUID   string             `db:"profile_uuid" json:"profile_uuid"`
	Name          string             `db:"name" json:"name"`
	Status        *MDMDeliveryStatus `db:"status" json:"status"`
	OperationType MDMOperationType   `db:"operation_type" json:"operation_type"`
	Detail        string             `db:"detail" json:"detail"`
	Hidden        bool               `db:"hidden" json:"hidden"`
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
		Hidden:        p.Hidden,
		SelfService:   false,
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

// FindUnsupportedAndroidFleetVar returns the name of the first $FLEET_VAR_*
// token in content that is not in the Android allow-list, or "" if all are
// supported.
func FindUnsupportedAndroidFleetVar(content string) string {
	for _, name := range variables.Find(content) {
		if !slices.Contains(FleetVarsSupportedInAndroidAppConfig, FleetVarName(name)) {
			return name
		}
	}
	return ""
}

// FleetVarsSupportedInAndroidAppConfig is the allow-list of Fleet variables that
// can appear in an Android managed app configuration or configuration profile JSON.
var FleetVarsSupportedInAndroidAppConfig = []FleetVarName{
	FleetVarHostUUID,
	FleetVarHostHardwareSerial,
	FleetVarHostPlatform,
	FleetVarHostEndUserEmailIDP,
	FleetVarHostEndUserIDPUsername,
	FleetVarHostEndUserIDPUsernameLocalPart,
	FleetVarHostEndUserIDPGroups,
	FleetVarHostEndUserIDPDepartment,
	FleetVarHostEndUserIDPFullname,
	FleetVarHostEndUserIDPExternalID,
}

// androidAppPolicyEnums lists the values Google accepts for each enum field reachable from
// ApplicationPolicy (other than the denied installType), keyed by field path. Google rejects the
// whole policy request when one app has a bad value, so they're checked at save time.
// TestAndroidAppPolicyEnumsMatchDiscoveryDoc keeps this in sync with the vendored
// google.golang.org/api version.
var androidAppPolicyEnums = map[string][]string{
	"alwaysOnVpnLockdownExemption": {"ALWAYS_ON_VPN_LOCKDOWN_EXEMPTION_UNSPECIFIED", "VPN_LOCKDOWN_ENFORCED", "VPN_LOCKDOWN_EXEMPTION"},
	"autoUpdateMode":               {"AUTO_UPDATE_MODE_UNSPECIFIED", "AUTO_UPDATE_DEFAULT", "AUTO_UPDATE_POSTPONED", "AUTO_UPDATE_HIGH_PRIORITY"},
	"connectedWorkAndPersonalApp":  {"CONNECTED_WORK_AND_PERSONAL_APP_UNSPECIFIED", "CONNECTED_WORK_AND_PERSONAL_APP_DISALLOWED", "CONNECTED_WORK_AND_PERSONAL_APP_ALLOWED"},
	"credentialProviderPolicy":     {"CREDENTIAL_PROVIDER_POLICY_UNSPECIFIED", "CREDENTIAL_PROVIDER_ALLOWED"},
	"customAppConfig.userUninstallSettings": {
		"USER_UNINSTALL_SETTINGS_UNSPECIFIED", "DISALLOW_UNINSTALL_BY_USER", "ALLOW_UNINSTALL_BY_USER",
	},
	"defaultPermissionPolicy": {"PERMISSION_POLICY_UNSPECIFIED", "PROMPT", "GRANT", "DENY"},
	"delegatedScopes": {
		"DELEGATED_SCOPE_UNSPECIFIED", "CERT_INSTALL", "MANAGED_CONFIGURATIONS", "BLOCK_UNINSTALL", "PERMISSION_GRANT",
		"PACKAGE_ACCESS", "ENABLE_SYSTEM_APP", "NETWORK_ACTIVITY_LOGS", "SECURITY_LOGS", "CERT_SELECTION",
	},
	"installConstraint.chargingConstraint": {
		"CHARGING_CONSTRAINT_UNSPECIFIED", "CHARGING_NOT_REQUIRED", "INSTALL_ONLY_WHEN_CHARGING",
	},
	"installConstraint.deviceIdleConstraint": {
		"DEVICE_IDLE_CONSTRAINT_UNSPECIFIED", "DEVICE_IDLE_NOT_REQUIRED", "INSTALL_ONLY_WHEN_DEVICE_IDLE",
	},
	"installConstraint.networkTypeConstraint": {
		"NETWORK_TYPE_CONSTRAINT_UNSPECIFIED", "INSTALL_ON_ANY_NETWORK", "INSTALL_ONLY_ON_UNMETERED_NETWORK",
	},
	"permissionGrants.policy": {"PERMISSION_POLICY_UNSPECIFIED", "PROMPT", "GRANT", "DENY"},
	"preferentialNetworkId": {
		"PREFERENTIAL_NETWORK_ID_UNSPECIFIED", "NO_PREFERENTIAL_NETWORK", "PREFERENTIAL_NETWORK_ID_ONE",
		"PREFERENTIAL_NETWORK_ID_TWO", "PREFERENTIAL_NETWORK_ID_THREE", "PREFERENTIAL_NETWORK_ID_FOUR",
		"PREFERENTIAL_NETWORK_ID_FIVE",
	},
	"roles.roleType": {
		"ROLE_TYPE_UNSPECIFIED", "COMPANION_APP", "KIOSK", "MOBILE_THREAT_DEFENSE_ENDPOINT_DETECTION_RESPONSE",
		"SYSTEM_HEALTH_MONITORING",
	},
	"userControlSettings": {"USER_CONTROL_SETTINGS_UNSPECIFIED", "USER_CONTROL_ALLOWED", "USER_CONTROL_DISALLOWED"},
	"workProfileWidgets":  {"WORK_PROFILE_WIDGETS_UNSPECIFIED", "WORK_PROFILE_WIDGETS_ALLOWED", "WORK_PROFILE_WIDGETS_DISALLOWED"},
}

// validateAndroidAppPolicyValues checks the enum values and limits Google documents for a single
// app. Rules that span the policy's apps (e.g. one KIOSK app per policy) are left to Google.
func validateAndroidAppPolicyValues(cfg *androidmanagement.ApplicationPolicy) error {
	// An empty string is omitted when the policy is sent, so Google applies the default. List
	// entries aren't omitted, so empty or null ones are rejected below.
	checkEnum := func(enumKey, fieldPath, value string) error {
		if value == "" || slices.Contains(androidAppPolicyEnums[enumKey], value) {
			return nil
		}
		return &BadRequestError{Message: fmt.Sprintf("Couldn't update configuration. %q is not a supported value for %q.", value, fieldPath)}
	}

	for _, field := range []struct{ path, value string }{
		{"alwaysOnVpnLockdownExemption", cfg.AlwaysOnVpnLockdownExemption},
		{"autoUpdateMode", cfg.AutoUpdateMode},
		{"connectedWorkAndPersonalApp", cfg.ConnectedWorkAndPersonalApp},
		{"credentialProviderPolicy", cfg.CredentialProviderPolicy},
		{"defaultPermissionPolicy", cfg.DefaultPermissionPolicy},
		{"preferentialNetworkId", cfg.PreferentialNetworkId},
		{"userControlSettings", cfg.UserControlSettings},
		{"workProfileWidgets", cfg.WorkProfileWidgets},
	} {
		if err := checkEnum(field.path, field.path, field.value); err != nil {
			return err
		}
	}
	// Google accepts customAppConfig and the KIOSK role only with installType CUSTOM (or
	// REQUIRED_FOR_SETUP for KIOSK), which Fleet never sends.
	if cfg.CustomAppConfig != nil {
		return &BadRequestError{Message: `Couldn't update configuration. "customAppConfig" is only supported for custom apps, which Fleet doesn't install.`}
	}
	// Google reads managedConfiguration as a protobuf Struct, which only a JSON object (or null) fits.
	if managedConfig := bytes.TrimSpace(cfg.ManagedConfiguration); len(managedConfig) > 0 && managedConfig[0] != '{' && !bytes.Equal(managedConfig, []byte("null")) {
		return &BadRequestError{Message: `Couldn't update configuration. "managedConfiguration" must be a JSON object.`}
	}
	emptyEntryErr := func(fieldPath string) error {
		return &BadRequestError{Message: fmt.Sprintf("Couldn't update configuration. %q can't be empty.", fieldPath)}
	}
	for i, scope := range cfg.DelegatedScopes {
		if scope == "" {
			return emptyEntryErr(fmt.Sprintf("delegatedScopes.%d", i))
		}
		if err := checkEnum("delegatedScopes", fmt.Sprintf("delegatedScopes.%d", i), scope); err != nil {
			return err
		}
	}
	for i, grant := range cfg.PermissionGrants {
		if grant == nil {
			return emptyEntryErr(fmt.Sprintf("permissionGrants.%d", i))
		}
		if err := checkEnum("permissionGrants.policy", fmt.Sprintf("permissionGrants.%d.policy", i), grant.Policy); err != nil {
			return err
		}
	}

	if len(cfg.InstallConstraint) > 1 {
		return &BadRequestError{Message: `Couldn't update configuration. "installConstraint" can have at most one entry.`}
	}
	for i, constraint := range cfg.InstallConstraint {
		if constraint == nil {
			return emptyEntryErr(fmt.Sprintf("installConstraint.%d", i))
		}
		hasConstraint := false
		for _, field := range []struct{ name, value string }{
			{"chargingConstraint", constraint.ChargingConstraint},
			{"deviceIdleConstraint", constraint.DeviceIdleConstraint},
			{"networkTypeConstraint", constraint.NetworkTypeConstraint},
		} {
			if err := checkEnum("installConstraint."+field.name, fmt.Sprintf("installConstraint.%d.%s", i, field.name), field.value); err != nil {
				return err
			}
			// Google treats the UNSPECIFIED value as unset and requires at least one field set.
			if field.value != "" && !strings.HasSuffix(field.value, "_UNSPECIFIED") {
				hasConstraint = true
			}
		}
		if !hasConstraint {
			return &BadRequestError{Message: fmt.Sprintf("Couldn't update configuration. %q must set at least one constraint.", fmt.Sprintf("installConstraint.%d", i))}
		}
	}

	seenRoles := make(map[string]struct{}, len(cfg.Roles))
	for i, role := range cfg.Roles {
		if role == nil {
			return emptyEntryErr(fmt.Sprintf("roles.%d", i))
		}
		fieldPath := fmt.Sprintf("roles.%d.roleType", i)
		if err := checkEnum("roles.roleType", fieldPath, role.RoleType); err != nil {
			return err
		}
		// Google rejects a role without a type, which an omitted roleType also is.
		if role.RoleType == "" || role.RoleType == "ROLE_TYPE_UNSPECIFIED" {
			return &BadRequestError{Message: fmt.Sprintf("Couldn't update configuration. %q is required.", fieldPath)}
		}
		if role.RoleType == "KIOSK" {
			return &BadRequestError{Message: fmt.Sprintf(`Couldn't update configuration. "KIOSK" isn't supported for %q.`, fieldPath)}
		}
		if _, dup := seenRoles[role.RoleType]; dup {
			return &BadRequestError{Message: fmt.Sprintf(`Couldn't update configuration. %q is listed more than once in "roles".`, role.RoleType)}
		}
		seenRoles[role.RoleType] = struct{}{}
	}
	if cfg.UserControlSettings == "USER_CONTROL_ALLOWED" && len(seenRoles) > 0 {
		return &BadRequestError{Message: `Couldn't update configuration. "userControlSettings" can't be "USER_CONTROL_ALLOWED" for an app with "roles".`}
	}
	if _, companion := seenRoles["COMPANION_APP"]; companion && cfg.ExtensionConfig != nil {
		return &BadRequestError{Message: `Couldn't update configuration. "extensionConfig" can't be set for an app with the "COMPANION_APP" role.`}
	}

	for i, cert := range cfg.SigningKeyCerts {
		if cert == nil {
			return emptyEntryErr(fmt.Sprintf("signingKeyCerts.%d", i))
		}
		if !isBase64SHA256(cert.SigningKeyCertFingerprintSha256) {
			return &BadRequestError{Message: fmt.Sprintf("Couldn't update configuration. %q must be a base64-encoded SHA-256 hash.", fmt.Sprintf("signingKeyCerts.%d.signingKeyCertFingerprintSha256", i))}
		}
	}
	if cfg.ExtensionConfig != nil {
		for i, fingerprint := range cfg.ExtensionConfig.SigningKeyFingerprintsSha256 {
			if decoded, err := hex.DecodeString(fingerprint); err != nil || len(decoded) != 32 {
				return &BadRequestError{Message: fmt.Sprintf("Couldn't update configuration. %q must be a 64-character hex SHA-256 hash.", fmt.Sprintf("extensionConfig.signingKeyFingerprintsSha256.%d", i))}
			}
		}
	}

	if cfg.InstallPriority < 0 || cfg.InstallPriority > 10_000 {
		return &BadRequestError{Message: `Couldn't update configuration. "installPriority" must be between 0 and 10000.`}
	}
	// Google stores minimumVersionCode as an int32, but the Go client decodes it into an int64.
	if cfg.MinimumVersionCode < math.MinInt32 || cfg.MinimumVersionCode > math.MaxInt32 {
		return &BadRequestError{Message: fmt.Sprintf(`Couldn't update configuration. "minimumVersionCode" must be between %d and %d.`, math.MinInt32, math.MaxInt32)}
	}

	return nil
}

// isBase64SHA256 reports whether s decodes to 32 bytes in any of the base64 forms protobuf JSON
// accepts for bytes fields.
func isBase64SHA256(s string) bool {
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if decoded, err := enc.DecodeString(s); err == nil && len(decoded) == 32 {
			return true
		}
	}
	return false
}

// androidAppConfigDeniedKeys are ApplicationPolicy fields Fleet sets itself: packageName is the
// software title's identity and installType is how Fleet drives self-service vs. setup experience.
// Keys are lowercased because encoding/json, which the worker uses to read stored configs, matches
// field names case-insensitively.
var androidAppConfigDeniedKeys = map[string]struct{}{
	"packagename": {},
	"installtype": {},
}

// androidApplicationPolicyFields returns the lowercased JSON names of
// androidmanagement.ApplicationPolicy's fields. Allowed keys follow the vendored
// google.golang.org/api version, so a field Google adds later is rejected until that dependency is
// bumped.
var androidApplicationPolicyFields = sync.OnceValue(func() map[string]struct{} {
	fields := make(map[string]struct{})
	for field := range reflect.TypeFor[androidmanagement.ApplicationPolicy]().Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name != "" && name != "-" {
			fields[strings.ToLower(name)] = struct{}{}
		}
	}
	return fields
})

var unknownJSONFieldRegexp = regexp.MustCompile(`unknown field ("(?:[^"\\]|\\.)*")`)

// ValidateAndroidAppConfiguration validates Android app configuration JSON.
// Configuration must be a valid JSON object whose top-level keys are
// ApplicationPolicy fields, other than "packageName" and "installType".
// Empty configuration is not allowed.
func ValidateAndroidAppConfiguration(config json.RawMessage) error {
	if len(config) == 0 {
		return &BadRequestError{
			Message: "Couldn't update configuration. Invalid JSON.",
		}
	}

	var topLevel map[string]json.RawMessage
	if err := json.Unmarshal(config, &topLevel); err != nil || topLevel == nil {
		return &BadRequestError{
			Message: "Couldn't update configuration. Invalid JSON.",
		}
	}

	keys := slices.Sorted(maps.Keys(topLevel))
	for _, key := range keys {
		if _, denied := androidAppConfigDeniedKeys[strings.ToLower(key)]; denied {
			return &BadRequestError{
				Message: `Couldn't update configuration. "packageName" and "installType" are not supported as top-level keys.`,
			}
		}
	}
	allowedFields := androidApplicationPolicyFields()
	for _, key := range keys {
		if _, ok := allowedFields[strings.ToLower(key)]; !ok {
			return &BadRequestError{Message: fmt.Sprintf("Couldn't update configuration. Unknown top-level key %q.", key)}
		}
	}

	// managedConfiguration is a RawMessage, so the strict decode leaves app-defined keys alone.
	var cfg androidmanagement.ApplicationPolicy
	if err := JSONStrictDecode(bytes.NewReader(config), &cfg); err != nil {
		if fieldPath := jsondecode.FieldPath(err); jsondecode.IsTypeError(err) && fieldPath != "" {
			return &BadRequestError{Message: fmt.Sprintf("Couldn't update configuration. %q format is wrong.", fieldPath)}
		}
		if m := unknownJSONFieldRegexp.FindStringSubmatch(err.Error()); m != nil {
			if name, uerr := strconv.Unquote(m[1]); uerr == nil {
				return &BadRequestError{Message: fmt.Sprintf("Couldn't update configuration. Unknown key %q.", name)}
			}
		}
		return &BadRequestError{
			Message: "Couldn't update configuration. Invalid JSON.",
		}
	}

	if err := validateAndroidAppPolicyValues(&cfg); err != nil {
		return err
	}

	if name := FindUnsupportedAndroidFleetVar(string(config)); name != "" {
		return &BadRequestError{
			Message: fmt.Sprintf("Couldn't update configuration. Unsupported variable $FLEET_VAR_%s.", name),
		}
	}

	// Malformed custom host vital refs (e.g. a typo like $FLEET_HOST_VITAL_asset_tag)
	// are rejected here since this validation also runs client-side (fleetctl),
	// where a database existence check isn't possible. Existence of well-formed
	// refs is validated server-side via ds.ValidateReferencedCustomHostVitals.
	if malformed := ContainsMalformedCustomHostVitalRefs(string(config)); len(malformed) > 0 {
		return &BadRequestError{
			Message: fmt.Sprintf("Couldn't update configuration. %s", (&InvalidCustomHostVitalRefError{Refs: malformed}).Error()),
		}
	}

	return nil
}
