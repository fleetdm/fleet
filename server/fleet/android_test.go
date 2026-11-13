package fleet

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsAndroidPolicyFieldValid(t *testing.T) {
	isValid := IsAndroidPolicyFieldValid("bogusKeyThatWillNeverExist")
	require.False(t, isValid)

	isValid = IsAndroidPolicyFieldValid("name") // "name" is a valid top-level policy field, that we assume will exist forever
	require.True(t, isValid)
}

func TestValidateAndroidAppConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		config      json.RawMessage
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid - both keys",
			config:      json.RawMessage(`{"managedConfiguration": {"key": "value"}, "workProfileWidgets": ""}`),
			expectError: false,
		},
		{
			name:        "valid - managedConfiguration only",
			config:      json.RawMessage(`{"managedConfiguration": {"key": "value"}}`),
			expectError: false,
		},
		{
			name:        "valid - workProfileWidgets only",
			config:      json.RawMessage(`{"workProfileWidgets": "WORK_PROFILE_WIDGETS_ALLOWED"}`),
			expectError: false,
		},
		{
			name:        "invalid - workProfileWidgets bad type",
			config:      json.RawMessage(`{"workProfileWidgets": false}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "workProfileWidgets" format is wrong.`,
		},
		{
			name:        "invalid - workProfileWidgets bad value",
			config:      json.RawMessage(`{"workProfileWidgets": "NO_SUCH_VALUE"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "NO_SUCH_VALUE" is not a supported value for "workProfileWidgets".`,
		},
		{
			name:        "valid - credentialProviderPolicy only",
			config:      json.RawMessage(`{"credentialProviderPolicy": "CREDENTIAL_PROVIDER_ALLOWED"}`),
			expectError: false,
		},
		{
			name:        "valid - credentialProviderPolicy unspecified value",
			config:      json.RawMessage(`{"credentialProviderPolicy": "CREDENTIAL_PROVIDER_POLICY_UNSPECIFIED"}`),
			expectError: false,
		},
		{
			name:        "valid - credentialProviderPolicy empty string",
			config:      json.RawMessage(`{"credentialProviderPolicy": ""}`),
			expectError: false,
		},
		{
			name:        "valid - all three keys",
			config:      json.RawMessage(`{"managedConfiguration": {"key": "value"}, "workProfileWidgets": "WORK_PROFILE_WIDGETS_ALLOWED", "credentialProviderPolicy": "CREDENTIAL_PROVIDER_ALLOWED"}`),
			expectError: false,
		},
		{
			name:        "invalid - credentialProviderPolicy bad value",
			config:      json.RawMessage(`{"credentialProviderPolicy": "CREDENTIAL_PROVIDER_DISALLOWED"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "CREDENTIAL_PROVIDER_DISALLOWED" is not a supported value for "credentialProviderPolicy".`,
		},
		{
			name:        "invalid - credentialProviderPolicy bad type",
			config:      json.RawMessage(`{"credentialProviderPolicy": true}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "credentialProviderPolicy" format is wrong.`,
		},
		{
			name:        "valid - empty object",
			config:      json.RawMessage(`{}`),
			expectError: false,
		},
		{
			name:        "valid - nested managedConfiguration",
			config:      json.RawMessage(`{"managedConfiguration": {"nested": {"key": "value"}}}`),
			expectError: false,
		},
		{
			name:        "valid - managedConfiguration with array",
			config:      json.RawMessage(`{"managedConfiguration": {"items": [1, 2, 3]}}`),
			expectError: false,
		},
		{
			name:        "invalid - empty string",
			config:      json.RawMessage(""),
			expectError: true,
			errorMsg:    "Couldn't update configuration. Invalid JSON.",
		},
		{
			name:        "invalid - null",
			config:      nil,
			expectError: true,
			errorMsg:    "Couldn't update configuration. Invalid JSON.",
		},
		{
			name:        "invalid - invalid JSON syntax",
			config:      json.RawMessage(`{invalid json}`),
			expectError: true,
			errorMsg:    "Couldn't update configuration. Invalid JSON.",
		},
		{
			name:        "invalid - not an object",
			config:      json.RawMessage(`"string"`),
			expectError: true,
			errorMsg:    "Couldn't update configuration. Invalid JSON.",
		},
		{
			name:        "invalid - array instead of object",
			config:      json.RawMessage(`[1, 2, 3]`),
			expectError: true,
			errorMsg:    "Couldn't update configuration. Invalid JSON.",
		},
		{
			name:        "invalid - literal null",
			config:      json.RawMessage(`null`),
			expectError: true,
			errorMsg:    "Couldn't update configuration. Invalid JSON.",
		},
		{
			name:        "invalid - unknown top-level key",
			config:      json.RawMessage(`{"invalidKey": "value"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. Unknown top-level key "invalidKey".`,
		},
		{
			name:        "invalid - extra key with valid keys",
			config:      json.RawMessage(`{"managedConfiguration": {}, "extraKey": "value"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. Unknown top-level key "extraKey".`,
		},
		{
			name:        "invalid - multiple invalid keys reports the first in sorted order",
			config:      json.RawMessage(`{"key2": "value2", "key1": "value1"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. Unknown top-level key "key1".`,
		},
		{
			name:        "invalid - misspelled ApplicationPolicy field",
			config:      json.RawMessage(`{"autoUpdteMode": "AUTO_UPDATE_HIGH_PRIORITY"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. Unknown top-level key "autoUpdteMode".`,
		},
		// Denied keys
		{
			name:        "invalid - packageName",
			config:      json.RawMessage(`{"packageName": "com.example.other"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "packageName" and "installType" are not supported as top-level keys.`,
		},
		{
			name:        "invalid - installType",
			config:      json.RawMessage(`{"installType": "FORCE_INSTALLED"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "packageName" and "installType" are not supported as top-level keys.`,
		},
		{
			name:        "invalid - installType with valid keys",
			config:      json.RawMessage(`{"managedConfiguration": {"key": "value"}, "installType": "KIOSK"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "packageName" and "installType" are not supported as top-level keys.`,
		},
		{
			name:        "invalid - denied key takes precedence over unknown key",
			config:      json.RawMessage(`{"aaaUnknown": 1, "packageName": "com.example.other"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "packageName" and "installType" are not supported as top-level keys.`,
		},
		{
			name:        "valid - packageName nested in managedConfiguration",
			config:      json.RawMessage(`{"managedConfiguration": {"packageName": "com.example.other", "installType": "x"}}`),
			expectError: false,
		},
		// Other ApplicationPolicy fields
		{
			name:        "valid - autoUpdateMode",
			config:      json.RawMessage(`{"autoUpdateMode": "AUTO_UPDATE_HIGH_PRIORITY"}`),
			expectError: false,
		},
		{
			name:        "valid - permissionGrants",
			config:      json.RawMessage(`{"permissionGrants": [{"permission": "android.permission.CAMERA", "policy": "GRANT"}]}`),
			expectError: false,
		},
		{
			name:        "valid - several ApplicationPolicy fields",
			config:      json.RawMessage(`{"managedConfiguration": {"key": "value"}, "minimumVersionCode": 42, "disabled": false, "installConstraint": [{"chargingConstraint": "CHARGING_NOT_REQUIRED"}], "delegatedScopes": ["MANAGED_CONFIGURATIONS"]}`),
			expectError: false,
		},
		{
			name:        "invalid - wrong type for minimumVersionCode",
			config:      json.RawMessage(`{"minimumVersionCode": "abc"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "minimumVersionCode" format is wrong.`,
		},
		{
			name:        "invalid - wrong type nested in permissionGrants",
			config:      json.RawMessage(`{"permissionGrants": [{"permission": 5}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "permissionGrants.0.permission" format is wrong.`,
		},
		{
			name:        "invalid - capitalized denied key",
			config:      json.RawMessage(`{"PackageName": "com.example.other"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "packageName" and "installType" are not supported as top-level keys.`,
		},
		{
			name:        "valid - capitalized ApplicationPolicy field",
			config:      json.RawMessage(`{"ManagedConfiguration": {"key": "value"}}`),
			expectError: false,
		},
		{
			name:        "invalid - Go-only ApplicationPolicy field",
			config:      json.RawMessage(`{"ForceSendFields": ["Disabled"]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. Unknown top-level key "ForceSendFields".`,
		},
		{
			name:        "invalid - misspelled nested key",
			config:      json.RawMessage(`{"permissionGrants": [{"permision": "android.permission.CAMERA", "policy": "GRANT"}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. Unknown key "permision".`,
		},
		{
			name:        "valid - arbitrary keys nested in managedConfiguration",
			config:      json.RawMessage(`{"managedConfiguration": {"anything": {"goes": [1, {"here": true}]}}}`),
			expectError: false,
		},
		{
			name:        "valid - Fleet variable in a non-managedConfiguration field",
			config:      json.RawMessage(`{"permissionGrants": [{"permission": "$FLEET_VAR_HOST_UUID", "policy": "GRANT"}]}`),
			expectError: false,
		},
		// Fleet variable tests
		{
			name:        "valid - supported Fleet variable HOST_UUID",
			config:      json.RawMessage(`{"managedConfiguration": {"deviceId": "$FLEET_VAR_HOST_UUID"}}`),
			expectError: false,
		},
		{
			name:        "valid - supported Fleet variable with braces",
			config:      json.RawMessage(`{"managedConfiguration": {"deviceId": "${FLEET_VAR_HOST_UUID}"}}`),
			expectError: false,
		},
		{
			name:        "valid - multiple supported Fleet variables",
			config:      json.RawMessage(`{"managedConfiguration": {"uuid": "$FLEET_VAR_HOST_UUID", "serial": "$FLEET_VAR_HOST_HARDWARE_SERIAL", "user": "$FLEET_VAR_HOST_END_USER_IDP_USERNAME"}}`),
			expectError: false,
		},
		{
			name:        "valid - HOST_PLATFORM variable",
			config:      json.RawMessage(`{"managedConfiguration": {"platform": "$FLEET_VAR_HOST_PLATFORM"}}`),
			expectError: false,
		},
		{
			name:        "valid - all IDP variables",
			config:      json.RawMessage(`{"managedConfiguration": {"email": "$FLEET_VAR_HOST_END_USER_EMAIL_IDP", "user": "$FLEET_VAR_HOST_END_USER_IDP_USERNAME", "local": "$FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART", "groups": "$FLEET_VAR_HOST_END_USER_IDP_GROUPS", "dept": "$FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT", "name": "$FLEET_VAR_HOST_END_USER_IDP_FULL_NAME", "externalId": "$FLEET_VAR_HOST_END_USER_IDP_EXTERNAL_ID"}}`),
			expectError: false,
		},
		{
			name:        "invalid - unsupported Fleet variable NDES_SCEP_CHALLENGE",
			config:      json.RawMessage(`{"managedConfiguration": {"challenge": "$FLEET_VAR_NDES_SCEP_CHALLENGE"}}`),
			expectError: true,
			errorMsg:    "Couldn't update configuration. Unsupported variable $FLEET_VAR_NDES_SCEP_CHALLENGE.",
		},
		{
			name:        "invalid - unsupported Fleet variable CUSTOM_SCEP",
			config:      json.RawMessage(`{"managedConfiguration": {"url": "$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_MyCA"}}`),
			expectError: true,
			errorMsg:    "Couldn't update configuration. Unsupported variable $FLEET_VAR_CUSTOM_SCEP_PROXY_URL_MyCA.",
		},
		{
			name:        "valid - custom host vital",
			config:      json.RawMessage(`{"managedConfiguration": {"assetTag": "$FLEET_HOST_VITAL_7"}}`),
			expectError: false,
		},
		{
			name:        "invalid - malformed custom host vital reference",
			config:      json.RawMessage(`{"managedConfiguration": {"assetTag": "$FLEET_HOST_VITAL_asset_tag"}}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. Invalid custom host vital reference "$FLEET_HOST_VITAL_asset_tag"; the value after $FLEET_HOST_VITAL_ must be a custom host vital ID`,
		},
		// Values Google rejects
		{
			name:        "valid - every enum field with a supported value",
			config:      json.RawMessage(`{"alwaysOnVpnLockdownExemption": "VPN_LOCKDOWN_EXEMPTION", "autoUpdateMode": "AUTO_UPDATE_HIGH_PRIORITY", "connectedWorkAndPersonalApp": "CONNECTED_WORK_AND_PERSONAL_APP_ALLOWED", "credentialProviderPolicy": "CREDENTIAL_PROVIDER_ALLOWED", "defaultPermissionPolicy": "GRANT", "delegatedScopes": ["CERT_INSTALL", "SECURITY_LOGS"], "installConstraint": [{"chargingConstraint": "INSTALL_ONLY_WHEN_CHARGING", "deviceIdleConstraint": "INSTALL_ONLY_WHEN_DEVICE_IDLE", "networkTypeConstraint": "INSTALL_ONLY_ON_UNMETERED_NETWORK"}], "permissionGrants": [{"permission": "android.permission.CAMERA", "policy": "DENY"}], "preferentialNetworkId": "PREFERENTIAL_NETWORK_ID_TWO", "roles": [{"roleType": "MOBILE_THREAT_DEFENSE_ENDPOINT_DETECTION_RESPONSE"}, {"roleType": "SYSTEM_HEALTH_MONITORING"}], "userControlSettings": "USER_CONTROL_DISALLOWED", "workProfileWidgets": "WORK_PROFILE_WIDGETS_DISALLOWED"}`),
			expectError: false,
		},
		{
			name:        "valid - unspecified enum values",
			config:      json.RawMessage(`{"autoUpdateMode": "AUTO_UPDATE_MODE_UNSPECIFIED", "permissionGrants": [{"permission": "android.permission.CAMERA", "policy": "PERMISSION_POLICY_UNSPECIFIED"}]}`),
			expectError: false,
		},
		{
			name:        "invalid - autoUpdateMode bad value",
			config:      json.RawMessage(`{"autoUpdateMode": "NOT_A_MODE"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "NOT_A_MODE" is not a supported value for "autoUpdateMode".`,
		},
		{
			name:        "invalid - enum value with the wrong case",
			config:      json.RawMessage(`{"autoUpdateMode": "auto_update_default"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "auto_update_default" is not a supported value for "autoUpdateMode".`,
		},
		{
			name:        "invalid - alwaysOnVpnLockdownExemption bad value",
			config:      json.RawMessage(`{"alwaysOnVpnLockdownExemption": "EXEMPT"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "EXEMPT" is not a supported value for "alwaysOnVpnLockdownExemption".`,
		},
		{
			name:        "invalid - connectedWorkAndPersonalApp bad value",
			config:      json.RawMessage(`{"connectedWorkAndPersonalApp": "ALLOWED"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "ALLOWED" is not a supported value for "connectedWorkAndPersonalApp".`,
		},
		{
			name:        "invalid - defaultPermissionPolicy bad value",
			config:      json.RawMessage(`{"defaultPermissionPolicy": "ALLOW"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "ALLOW" is not a supported value for "defaultPermissionPolicy".`,
		},
		{
			name:        "invalid - preferentialNetworkId bad value",
			config:      json.RawMessage(`{"preferentialNetworkId": "PREFERENTIAL_NETWORK_ID_SIX"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "PREFERENTIAL_NETWORK_ID_SIX" is not a supported value for "preferentialNetworkId".`,
		},
		{
			name:        "invalid - userControlSettings bad value",
			config:      json.RawMessage(`{"userControlSettings": "ALLOWED"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "ALLOWED" is not a supported value for "userControlSettings".`,
		},
		{
			name:        "invalid - customAppConfig",
			config:      json.RawMessage(`{"customAppConfig": {"userUninstallSettings": "ALLOW_UNINSTALL_BY_USER"}}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "customAppConfig" is only supported for custom apps, which Fleet doesn't install.`,
		},
		{
			name:        "invalid - delegatedScopes bad value",
			config:      json.RawMessage(`{"delegatedScopes": ["CERT_INSTALL", "ROOT"]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "ROOT" is not a supported value for "delegatedScopes.1".`,
		},
		{
			name:        "invalid - delegatedScopes empty entry",
			config:      json.RawMessage(`{"delegatedScopes": ["CERT_INSTALL", ""]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "delegatedScopes.1" can't be empty.`,
		},
		{
			name:        "invalid - delegatedScopes null entry",
			config:      json.RawMessage(`{"delegatedScopes": [null]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "delegatedScopes.0" can't be empty.`,
		},
		{
			name:        "invalid - permissionGrants null entry",
			config:      json.RawMessage(`{"permissionGrants": [{"permission": "android.permission.CAMERA", "policy": "DENY"}, null]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "permissionGrants.1" can't be empty.`,
		},
		{
			name:        "invalid - installConstraint null entry",
			config:      json.RawMessage(`{"installConstraint": [null]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "installConstraint.0" can't be empty.`,
		},
		{
			name:        "invalid - roles null entry",
			config:      json.RawMessage(`{"roles": [null]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "roles.0" can't be empty.`,
		},
		{
			name:        "invalid - permissionGrants policy bad value",
			config:      json.RawMessage(`{"permissionGrants": [{"permission": "android.permission.CAMERA", "policy": "GRANT"}, {"permission": "android.permission.RECORD_AUDIO", "policy": "ALLOW"}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "ALLOW" is not a supported value for "permissionGrants.1.policy".`,
		},
		{
			name:        "invalid - installConstraint chargingConstraint bad value",
			config:      json.RawMessage(`{"installConstraint": [{"chargingConstraint": "ONLY_WHEN_CHARGING"}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "ONLY_WHEN_CHARGING" is not a supported value for "installConstraint.0.chargingConstraint".`,
		},
		{
			name:        "invalid - installConstraint deviceIdleConstraint bad value",
			config:      json.RawMessage(`{"installConstraint": [{"deviceIdleConstraint": "IDLE"}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "IDLE" is not a supported value for "installConstraint.0.deviceIdleConstraint".`,
		},
		{
			name:        "invalid - installConstraint networkTypeConstraint bad value",
			config:      json.RawMessage(`{"installConstraint": [{"networkTypeConstraint": "WIFI"}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "WIFI" is not a supported value for "installConstraint.0.networkTypeConstraint".`,
		},
		{
			name:        "invalid - installConstraint with no constraint",
			config:      json.RawMessage(`{"installConstraint": [{}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "installConstraint.0" must set at least one constraint.`,
		},
		{
			name:        "invalid - installConstraint with only unspecified constraints",
			config:      json.RawMessage(`{"installConstraint": [{"chargingConstraint": "CHARGING_CONSTRAINT_UNSPECIFIED", "networkTypeConstraint": "NETWORK_TYPE_CONSTRAINT_UNSPECIFIED"}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "installConstraint.0" must set at least one constraint.`,
		},
		{
			name:        "valid - installConstraint with one constraint among unspecified ones",
			config:      json.RawMessage(`{"installConstraint": [{"chargingConstraint": "CHARGING_CONSTRAINT_UNSPECIFIED", "deviceIdleConstraint": "DEVICE_IDLE_NOT_REQUIRED"}]}`),
			expectError: false,
		},
		{
			name:        "invalid - two installConstraint entries",
			config:      json.RawMessage(`{"installConstraint": [{"chargingConstraint": "INSTALL_ONLY_WHEN_CHARGING"}, {"networkTypeConstraint": "INSTALL_ON_ANY_NETWORK"}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "installConstraint" can have at most one entry.`,
		},
		{
			name:        "invalid - roles roleType bad value",
			config:      json.RawMessage(`{"roles": [{"roleType": "ADMIN"}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "ADMIN" is not a supported value for "roles.0.roleType".`,
		},
		{
			name:        "invalid - roles roleType unspecified",
			config:      json.RawMessage(`{"roles": [{"roleType": "SYSTEM_HEALTH_MONITORING"}, {"roleType": "ROLE_TYPE_UNSPECIFIED"}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "roles.1.roleType" is required.`,
		},
		{
			name:        "invalid - roles roleType missing",
			config:      json.RawMessage(`{"roles": [{}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "roles.0.roleType" is required.`,
		},
		{
			name:        "invalid - duplicate roles roleType",
			config:      json.RawMessage(`{"roles": [{"roleType": "SYSTEM_HEALTH_MONITORING"}, {"roleType": "SYSTEM_HEALTH_MONITORING"}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "SYSTEM_HEALTH_MONITORING" is listed more than once in "roles".`,
		},
		{
			name:        "invalid - KIOSK role",
			config:      json.RawMessage(`{"roles": [{"roleType": "KIOSK"}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "KIOSK" isn't supported for "roles.0.roleType".`,
		},
		{
			name:        "invalid - COMPANION_APP role",
			config:      json.RawMessage(`{"roles": [{"roleType": "SYSTEM_HEALTH_MONITORING"}, {"roleType": "COMPANION_APP"}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "COMPANION_APP" isn't supported for "roles.1.roleType".`,
		},
		{
			name:        "valid - USER_CONTROL_ALLOWED without roles",
			config:      json.RawMessage(`{"userControlSettings": "USER_CONTROL_ALLOWED"}`),
			expectError: false,
		},
		{
			name:        "invalid - USER_CONTROL_ALLOWED with roles",
			config:      json.RawMessage(`{"userControlSettings": "USER_CONTROL_ALLOWED", "roles": [{"roleType": "SYSTEM_HEALTH_MONITORING"}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "userControlSettings" can't be "USER_CONTROL_ALLOWED" for an app with "roles".`,
		},
		{
			name:        "valid - managedConfiguration null",
			config:      json.RawMessage(`{"managedConfiguration": null}`),
			expectError: false,
		},
		{
			name:        "invalid - managedConfiguration number",
			config:      json.RawMessage(`{"managedConfiguration": 5}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "managedConfiguration" must be a JSON object.`,
		},
		{
			name:        "invalid - managedConfiguration array",
			config:      json.RawMessage(`{"managedConfiguration": [{"key": "value"}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "managedConfiguration" must be a JSON object.`,
		},
		{
			name:        "invalid - managedConfiguration string",
			config:      json.RawMessage(`{"managedConfiguration": "key=value"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "managedConfiguration" must be a JSON object.`,
		},
		{
			name:        "valid - signingKeyCerts base64 fingerprints",
			config:      json.RawMessage(`{"signingKeyCerts": [{"signingKeyCertFingerprintSha256": "XrLOKRx9In3WhOyD+d3AV3bi/poMTmKSe0WSOD5m+yg="}, {"signingKeyCertFingerprintSha256": "XrLOKRx9In3WhOyD-d3AV3bi_poMTmKSe0WSOD5m-yg"}]}`),
			expectError: false,
		},
		{
			name:        "invalid - signingKeyCerts null entry",
			config:      json.RawMessage(`{"signingKeyCerts": [null]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "signingKeyCerts.0" can't be empty.`,
		},
		{
			name:        "invalid - signingKeyCerts missing fingerprint",
			config:      json.RawMessage(`{"signingKeyCerts": [{}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "signingKeyCerts.0.signingKeyCertFingerprintSha256" must be a base64-encoded SHA-256 hash.`,
		},
		{
			name:        "invalid - signingKeyCerts fingerprint not base64",
			config:      json.RawMessage(`{"signingKeyCerts": [{"signingKeyCertFingerprintSha256": "XrLOKRx9In3WhOyD+d3AV3bi/poMTmKSe0WSOD5m+yg="}, {"signingKeyCertFingerprintSha256": "not base64!!"}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "signingKeyCerts.1.signingKeyCertFingerprintSha256" must be a base64-encoded SHA-256 hash.`,
		},
		{
			name:        "invalid - signingKeyCerts fingerprint wrong length",
			config:      json.RawMessage(`{"signingKeyCerts": [{"signingKeyCertFingerprintSha256": "XrLOKRx9In3WhOyD+d3AV3bi/poMTmKSe0WSOD5m+w=="}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "signingKeyCerts.0.signingKeyCertFingerprintSha256" must be a base64-encoded SHA-256 hash.`,
		},
		{
			name:        "invalid - signingKeyCerts hex fingerprint",
			config:      json.RawMessage(`{"signingKeyCerts": [{"signingKeyCertFingerprintSha256": "5eb2ce291c7d227dd684ec83f9ddc05776e2fe9a0c4e62927b4592383e66fb28"}]}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "signingKeyCerts.0.signingKeyCertFingerprintSha256" must be a base64-encoded SHA-256 hash.`,
		},
		{
			name:        "invalid - extensionConfig",
			config:      json.RawMessage(`{"extensionConfig": {"notificationReceiver": "com.example.Receiver"}}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "extensionConfig" isn't supported.`,
		},
		{
			name:        "valid - extensionConfig null",
			config:      json.RawMessage(`{"extensionConfig": null}`),
			expectError: false,
		},
		{
			name:        "valid - installPriority bounds",
			config:      json.RawMessage(`{"installPriority": 10000}`),
			expectError: false,
		},
		{
			name:        "valid - installPriority zero",
			config:      json.RawMessage(`{"installPriority": 0}`),
			expectError: false,
		},
		{
			name:        "invalid - installPriority above range",
			config:      json.RawMessage(`{"installPriority": 20000}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "installPriority" must be between 0 and 10000.`,
		},
		{
			name:        "invalid - installPriority just above range",
			config:      json.RawMessage(`{"installPriority": 10001}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "installPriority" must be between 0 and 10000.`,
		},
		{
			name:        "invalid - installPriority negative",
			config:      json.RawMessage(`{"installPriority": -1}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "installPriority" must be between 0 and 10000.`,
		},
		{
			name:        "valid - minimumVersionCode at the int32 maximum",
			config:      json.RawMessage(`{"minimumVersionCode": 2147483647}`),
			expectError: false,
		},
		{
			name:        "invalid - minimumVersionCode above the int32 maximum",
			config:      json.RawMessage(`{"minimumVersionCode": 2147483648}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "minimumVersionCode" must be between -2147483648 and 2147483647.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAndroidAppConfiguration(tt.config)
			if tt.expectError {
				require.Error(t, err)
				var badReqErr *BadRequestError
				require.ErrorAs(t, err, &badReqErr)
				require.Equal(t, tt.errorMsg, badReqErr.Message)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateUserProvided_FleetVariables(t *testing.T) {
	tests := []struct {
		name      string
		rawJSON   string
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "supported variable HOST_UUID in string value",
			rawJSON: `{"name": "$FLEET_VAR_HOST_UUID"}`,
			wantErr: false,
		},
		{
			name:    "supported variable with braces",
			rawJSON: `{"name": "${FLEET_VAR_HOST_HARDWARE_SERIAL}"}`,
			wantErr: false,
		},
		{
			name:    "multiple supported variables",
			rawJSON: `{"name": "$FLEET_VAR_HOST_UUID $FLEET_VAR_HOST_END_USER_IDP_USERNAME"}`,
			wantErr: false,
		},
		{
			name:    "no variables at all",
			rawJSON: `{"name": "plain-value"}`,
			wantErr: false,
		},
		{
			name:      "unsupported variable NDES_SCEP_CHALLENGE",
			rawJSON:   `{"name": "$FLEET_VAR_NDES_SCEP_CHALLENGE"}`,
			wantErr:   true,
			errSubstr: "Unsupported Fleet variable $FLEET_VAR_NDES_SCEP_CHALLENGE",
		},
		{
			name:      "unsupported variable CUSTOM_SCEP",
			rawJSON:   `{"name": "$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_MyCA"}`,
			wantErr:   true,
			errSubstr: "Unsupported Fleet variable",
		},
		{
			name:      "variable not in JSON string value — in key",
			rawJSON:   `{"$FLEET_VAR_HOST_UUID": "value"}`,
			wantErr:   true,
			errSubstr: "Unknown key",
		},
		{
			// maximumTimeToLock expects a number; a string with a variable is
			// caught by the Policy struct validation before our variable check.
			name:      "variable in number field is rejected by policy validation",
			rawJSON:   `{"name": "ok", "maximumTimeToLock": "$FLEET_VAR_HOST_UUID"}`,
			wantErr:   true,
			errSubstr: "Invalid JSON payload",
		},
		{
			name:    "custom host vital in JSON string value is allowed",
			rawJSON: `{"name": "$FLEET_HOST_VITAL_7"}`,
			wantErr: false,
		},
		{
			name:    "custom host vital alongside a supported Fleet variable is allowed",
			rawJSON: `{"name": "$FLEET_VAR_HOST_UUID $FLEET_HOST_VITAL_7"}`,
			wantErr: false,
		},
		{
			name:      "custom host vital in a nested JSON key must be inside a string value",
			rawJSON:   `{"name": "ok", "passwordRequirements": {"$FLEET_HOST_VITAL_7": "value"}}`,
			wantErr:   true,
			errSubstr: "Custom host vital $FLEET_HOST_VITAL_7 must be inside a JSON string value",
		},
		{
			// A zero-padded ID normalizes to the same vital ("007" -> 7); the
			// string-value-position check must recognize that rather than
			// falsely reporting it as not found in a string value.
			name:    "custom host vital with a zero-padded ID in a string value is allowed",
			rawJSON: `{"name": "$FLEET_HOST_VITAL_007"}`,
			wantErr: false,
		},
		{
			name:      "malformed custom host vital reference is rejected",
			rawJSON:   `{"name": "$FLEET_HOST_VITAL_asset_tag"}`,
			wantErr:   true,
			errSubstr: "Invalid custom host vital reference",
		},
		{
			// A vital token present in the raw upload but shadowed by a
			// duplicate JSON key (json.Unmarshal keeps only the last value for
			// a repeated key) ends up neither a decoded key nor a decoded
			// string value, exercising the "not in string value" branch
			// distinctly from the "used as a key" case above.
			name:      "custom host vital shadowed by a duplicate JSON key is rejected",
			rawJSON:   `{"name": "ok", "passwordRequirements": {"a": "$FLEET_HOST_VITAL_7", "a": "no-vital-here"}}`,
			wantErr:   true,
			errSubstr: "Custom host vital $FLEET_HOST_VITAL_7 must be inside a JSON string value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prof := &MDMAndroidConfigProfile{
				Name:    "test-profile",
				RawJSON: []byte(tt.rawJSON),
			}
			err := prof.ValidateUserProvided(true)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errSubstr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateUserProvided_SoftwareKeys(t *testing.T) {
	for key := range AndroidSoftwareJSONKeysFree {
		require.Contains(t, AndroidForbiddenJSONKeys, key)

		prof := MDMAndroidConfigProfile{Name: "test", RawJSON: []byte(`{"` + key + `": true}`)}

		err := prof.ValidateUserProvided(true)
		require.EqualError(t, err, AndroidForbiddenJSONKeys[key])
		require.Contains(t, err.Error(), "https://fleetdm.com/learn-more-about/android-software")

		err = prof.ValidateUserProvided(false)
		require.EqualError(t, err, AndroidSoftwareJSONKeysFree[key])
		require.Contains(t, err.Error(), "Fleet Premium only")
	}
}

// TestAndroidAppPolicyEnumsMatchDiscoveryDoc fails when a google.golang.org/api bump changes the
// enum values Google accepts, so androidAppPolicyEnums never rejects a value Google allows.
func TestAndroidAppPolicyEnumsMatchDiscoveryDoc(t *testing.T) {
	out, err := exec.CommandContext(t.Context(), "go", "list", "-m", "-f", "{{.Dir}}", "google.golang.org/api").Output()
	require.NoError(t, err, "locating google.golang.org/api")
	docPath := filepath.Join(strings.TrimSpace(string(out)), "androidmanagement", "v1", "androidmanagement-api.json")
	docJSON, err := os.ReadFile(docPath)
	require.NoError(t, err)

	type property struct {
		Ref   string    `json:"$ref"`
		Enum  []string  `json:"enum"`
		Items *property `json:"items"`
	}
	var doc struct {
		Schemas map[string]struct {
			Properties map[string]property `json:"properties"`
		} `json:"schemas"`
	}
	require.NoError(t, json.Unmarshal(docJSON, &doc))

	discovered := make(map[string][]string)
	arrayPaths := make(map[string]struct{})
	var walk func(schema, prefix string, seen map[string]bool)
	walk = func(schema, prefix string, seen map[string]bool) {
		if seen[schema] {
			return
		}
		seen = maps.Clone(seen)
		seen[schema] = true
		for name, prop := range doc.Schemas[schema].Properties {
			if prop.Items != nil {
				arrayPaths[prefix+name] = struct{}{}
				prop = *prop.Items
			}
			if len(prop.Enum) > 0 {
				discovered[prefix+name] = prop.Enum
			}
			if prop.Ref != "" {
				walk(prop.Ref, prefix+name+".", seen)
			}
		}
	}
	walk("ApplicationPolicy", "", map[string]bool{})
	require.NotEmpty(t, discovered)
	delete(discovered, "installType")

	require.Equal(t, discovered, androidAppPolicyEnums)

	// Every enum in the table must also be checked, so a field added on a dependency bump isn't
	// listed here but left unvalidated.
	for enumPath := range androidAppPolicyEnums {
		t.Run(enumPath, func(t *testing.T) {
			segments := strings.Split(enumPath, ".")
			var value any = "NOT_A_SUPPORTED_VALUE"
			for i, segment := range slices.Backward(segments) {
				if _, isArray := arrayPaths[strings.Join(segments[:i+1], ".")]; isArray {
					value = []any{value}
				}
				if i > 0 {
					value = map[string]any{segment: value}
				}
			}
			var fieldPath []string
			for i, segment := range segments {
				fieldPath = append(fieldPath, segment)
				if _, isArray := arrayPaths[strings.Join(segments[:i+1], ".")]; isArray {
					fieldPath = append(fieldPath, "0")
				}
			}
			expectedMsg := fmt.Sprintf("Couldn't update configuration. %q is not a supported value for %q.", "NOT_A_SUPPORTED_VALUE", strings.Join(fieldPath, "."))
			if segments[0] == "customAppConfig" {
				// customAppConfig is rejected as a whole, so its enums are never reached.
				expectedMsg = `Couldn't update configuration. "customAppConfig" is only supported for custom apps, which Fleet doesn't install.`
			}
			config, err := json.Marshal(map[string]any{segments[0]: value})
			require.NoError(t, err)
			require.EqualError(t, ValidateAndroidAppConfiguration(config), expectedMsg, string(config))
		})
	}
}
