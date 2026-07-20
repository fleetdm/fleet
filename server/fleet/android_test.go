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
