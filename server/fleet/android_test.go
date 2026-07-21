package fleet

import (
	"encoding/json"
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
			errorMsg:    "Couldn't update configuration. Invalid JSON.",
		},
		{
			name:        "invalid - workProfileWidgets bad value",
			config:      json.RawMessage(`{"workProfileWidgets": "NO_SUCH_VALUE"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. "NO_SUCH_VALUE" is not a supported value for "workProfileWidget".`,
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
			name:        "invalid - unknown top-level key",
			config:      json.RawMessage(`{"invalidKey": "value"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. Only "managedConfiguration" and "workProfileWidgets" are supported as top-level keys.`,
		},
		{
			name:        "invalid - extra key with valid keys",
			config:      json.RawMessage(`{"managedConfiguration": {}, "extraKey": "value"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. Only "managedConfiguration" and "workProfileWidgets" are supported as top-level keys.`,
		},
		{
			name:        "invalid - multiple invalid keys",
			config:      json.RawMessage(`{"key1": "value1", "key2": "value2"}`),
			expectError: true,
			errorMsg:    `Couldn't update configuration. Only "managedConfiguration" and "workProfileWidgets" are supported as top-level keys.`,
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
			config:      json.RawMessage(`{"managedConfiguration": {"email": "$FLEET_VAR_HOST_END_USER_EMAIL_IDP", "user": "$FLEET_VAR_HOST_END_USER_IDP_USERNAME", "local": "$FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART", "groups": "$FLEET_VAR_HOST_END_USER_IDP_GROUPS", "dept": "$FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT", "name": "$FLEET_VAR_HOST_END_USER_IDP_FULL_NAME"}}`),
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
