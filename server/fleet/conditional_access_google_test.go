package fleet

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoogleConditionalAccessIntegrationValidate(t *testing.T) {
	errNames := func(invalid *InvalidArgumentError) []string {
		var names []string
		for _, e := range invalid.Errors {
			names = append(names, e.name)
		}
		return names
	}

	t.Run("both fields are required", func(t *testing.T) {
		invalid := &InvalidArgumentError{}
		(&GoogleConditionalAccessIntegration{}).Validate(invalid)
		require.ElementsMatch(t, []string{"customer_id", "admin_email"}, errNames(invalid))
	})

	t.Run("customer ID must start with C", func(t *testing.T) {
		for _, tc := range []struct {
			in    string
			valid bool
		}{
			{"C02cajzvg", true},
			{"Cabcdefgh", true},
			{"02cajzvg", false},
			{"c02cajzvg", false},
			{"C", false},
			{"C0123", false},
			{"C02-cajzvg", false},
			{"customers/C02cajzvg", false},
		} {
			invalid := &InvalidArgumentError{}
			(&GoogleConditionalAccessIntegration{CustomerID: tc.in, AdminEmail: "admin@example.com"}).Validate(invalid)
			require.Equal(t, !tc.valid, invalid.HasErrors(), tc.in)
		}
	})

	t.Run("trims", func(t *testing.T) {
		invalid := &InvalidArgumentError{}
		intg := &GoogleConditionalAccessIntegration{CustomerID: " C02cajzvg", AdminEmail: " admin@example.com "}
		intg.Validate(invalid)
		require.False(t, invalid.HasErrors())
		require.Equal(t, "C02cajzvg", intg.CustomerID)
		require.Equal(t, "admin@example.com", intg.AdminEmail)
	})

	t.Run("malformed admin email", func(t *testing.T) {
		for _, email := range []string{"Admin <admin@example.com>", "admin"} {
			invalid := &InvalidArgumentError{}
			(&GoogleConditionalAccessIntegration{CustomerID: "C02cajzvg", AdminEmail: email}).Validate(invalid)
			require.Equal(t, []string{"admin_email"}, errNames(invalid), email)
		}
	})
}

func TestValidateGoogleConditionalAccessIntegration(t *testing.T) {
	invalid := &InvalidArgumentError{}
	ValidateGoogleConditionalAccessIntegration(nil, invalid)
	require.False(t, invalid.HasErrors())

	intg := &GoogleConditionalAccessIntegration{CustomerID: " C02cajzvg ", AdminEmail: "nope"}
	ValidateGoogleConditionalAccessIntegration(intg, invalid)
	require.Equal(t, []map[string]string{
		{"name": "integrations.google_conditional_access.admin_email", "reason": "must be a valid email address"},
	}, invalid.Invalid())
	require.Equal(t, "C02cajzvg", intg.CustomerID)
}

func TestGoogleServiceAccountKeyJSON(t *testing.T) {
	var intg GoogleConditionalAccessIntegration
	require.NoError(t, json.Unmarshal([]byte(`{"customer_id": "C02cajzvg", "api_key_json": {"client_email": "sa@example.com", "private_key": "pk"}}`), &intg))
	require.True(t, intg.APIKeyJSON.Provided())
	require.JSONEq(t, `{"client_email": "sa@example.com", "private_key": "pk"}`, string(intg.APIKeyJSON))

	intg = GoogleConditionalAccessIntegration{}
	require.NoError(t, json.Unmarshal([]byte(`{"api_key_json": "********"}`), &intg))
	require.False(t, intg.APIKeyJSON.Provided())
	require.Equal(t, GoogleServiceAccountKey(MaskedPassword), intg.APIKeyJSON)

	intg = GoogleConditionalAccessIntegration{}
	require.NoError(t, json.Unmarshal([]byte(`{"api_key_json": null}`), &intg))
	require.False(t, intg.APIKeyJSON.Provided())

	require.Error(t, json.Unmarshal([]byte(`{"api_key_json": "a key"}`), &intg))
	require.Error(t, json.Unmarshal([]byte(`{"api_key_json": [1]}`), &intg))

	b, err := json.Marshal(GoogleConditionalAccessIntegration{CustomerID: "C02cajzvg"})
	require.NoError(t, err)
	require.JSONEq(t, `{"customer_id": "C02cajzvg", "admin_email": ""}`, string(b))
	b, err = json.Marshal(GoogleConditionalAccessIntegration{APIKeyJSON: MaskedPassword})
	require.NoError(t, err)
	require.JSONEq(t, `{"customer_id": "", "admin_email": "", "api_key_json": "********"}`, string(b))
}

func TestGoogleConditionalAccessIntegrationValidateKey(t *testing.T) {
	intg := &GoogleConditionalAccessIntegration{CustomerID: "C02cajzvg", AdminEmail: "admin@example.com"}
	intg.APIKeyJSON = `{"client_email": "sa@example.com"}`
	invalid := &InvalidArgumentError{}
	ValidateGoogleConditionalAccessIntegration(intg, invalid)
	require.Equal(t, []map[string]string{
		{"name": "integrations.google_conditional_access.api_key_json", "reason": "api_key_json must include client_email and private_key"},
	}, invalid.Invalid())

	invalid = &InvalidArgumentError{}
	intg.APIKeyJSON = MaskedPassword
	ValidateGoogleConditionalAccessIntegration(intg, invalid)
	require.False(t, invalid.HasErrors())
}

func TestGoogleConditionalAccessDesiredClientState(t *testing.T) {
	for _, state := range []GoogleConditionalAccessDeviceUserState{
		GoogleConditionalAccessDeviceUserPending,
		GoogleConditionalAccessDeviceUserManaged,
		GoogleConditionalAccessDeviceUserStale,
	} {
		du := GoogleConditionalAccessDeviceUser{HostUUID: "host-uuid-1", Name: "devices/d/deviceUsers/u", State: state}
		require.Equal(t, GoogleConditionalAccessClientState{Managed: new(true), CustomID: new("host-uuid-1")},
			GoogleConditionalAccessDesiredClientState(du), state)
	}

	du := GoogleConditionalAccessDeviceUser{HostUUID: "host-uuid-1", Name: "devices/d/deviceUsers/u", State: GoogleConditionalAccessDeviceUserUnmanaging}
	require.Equal(t, GoogleConditionalAccessClientState{Managed: new(false)}, GoogleConditionalAccessDesiredClientState(du))
}
