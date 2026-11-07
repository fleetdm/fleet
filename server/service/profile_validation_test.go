package service

import (
	"testing"

	pv "github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"github.com/stretchr/testify/require"
)

func TestProfileValidationSentence(t *testing.T) {
	t.Parallel()

	mismatch := func(key string, expected ...string) pv.Finding {
		return pv.Finding{Code: pv.CodeTypeMismatch, KeyPath: key, Expected: expected}
	}

	cases := []struct {
		name    string
		finding pv.Finding
		windows bool
		want    string
	}{
		// type words from the Contract
		{"apple string", mismatch("k", "string"), false, `"k" expects a string.`},
		{"apple integer", mismatch("k", "integer"), false, `"k" expects a number.`},
		{"apple real", mismatch("k", "real"), false, `"k" expects a number.`},
		{"apple boolean", mismatch("allowAirDrop", "boolean"), false, `"allowAirDrop" expects a boolean.`},
		{"apple date", mismatch("k", "date"), false, `"k" expects a date.`},
		{"apple data", mismatch("k", "data"), false, `"k" expects base64 data.`},
		{"apple array", mismatch("k", "array"), false, `"k" expects an array.`},
		{"apple dictionary", mismatch("k", "dictionary"), false, `"k" expects a dictionary.`},
		{"windows int", mismatch("./Device/A", "int"), true, `"./Device/A" expects a number.`},
		{"windows bool", mismatch("./Device/A", "bool"), true, `"./Device/A" expects a boolean.`},
		{"windows b64", mismatch("./Device/A", "b64"), true, `"./Device/A" expects base64 data.`},
		{"windows xml", mismatch("./Device/A", "xml"), true, `"./Device/A" expects XML.`},

		// joining
		{"or join", mismatch("k", "real", "string"), false, `"k" expects a number or a string.`},
		{"or join dedupes", mismatch("k", "integer", "real"), false, `"k" expects a number.`},
		{"unknown type name passes through", mismatch("k", "uuid"), false, `"k" expects uuid.`},
		{"dotted key path", mismatch("PayloadContent.0.Rules", "array"), false, `"PayloadContent.0.Rules" expects an array.`},

		// strict-only Windows formats
		{
			"format mismatch extra formats",
			pv.Finding{Code: pv.CodeFormatMismatch, KeyPath: "./Device/A", Expected: []string{"chr", "float", "time", "bin", "node", "null"}},
			true, `"./Device/A" expects a string or a number or a time or binary data or a node or no value.`,
		},

		// finding sentences
		{
			"unknown key",
			pv.Finding{Code: pv.CodeUnknownKey, PayloadType: "com.apple.applicationaccess", KeyPath: "allowFoo"},
			false, `"allowFoo" isn't valid for the "com.apple.applicationaccess" profile type.`,
		},
		{
			"unknown key at root",
			pv.Finding{Code: pv.CodeUnknownKey, PayloadType: pv.TopLevelPayloadType, KeyPath: "Foo"},
			false, `"Foo" isn't valid for the "Configuration" profile type.`,
		},
		{
			"unknown payload type",
			pv.Finding{Code: pv.CodeUnknownType, PayloadType: "com.example.nope"},
			false, `"com.example.nope" isn't a valid profile type.`,
		},
		{
			"unknown LocURI",
			pv.Finding{Code: pv.CodeUnknownKey, PayloadType: "Nope", KeyPath: "./Device/Vendor/MSFT/Policy/Config/Nope/Thing"},
			true, `"./Device/Vendor/MSFT/Policy/Config/Nope/Thing" isn't a valid Windows setting.`,
		},
		{
			"meta format mismatch",
			pv.Finding{Code: pv.CodeFormatMismatch, KeyPath: "./Device/A", Expected: []string{"int"}, Actual: "chr"},
			true, `"./Device/A" expects a number.`,
		},
		{
			"wrong scope user",
			pv.Finding{Code: pv.CodeWrongScope, KeyPath: "./User/A", Expected: []string{"device"}, Actual: "user"},
			true, `"./User/A" can't be set for users.`,
		},
		{
			"wrong scope device",
			pv.Finding{Code: pv.CodeWrongScope, KeyPath: "./Device/A", Expected: []string{"user"}, Actual: "device"},
			true, `"./Device/A" can't be set for devices.`,
		},
		{
			"missing PayloadVersion",
			pv.Finding{Code: pv.CodeMissingPayloadVersion, PayloadType: "com.apple.wifi.managed"},
			false, `A "com.apple.wifi.managed" payload is missing PayloadVersion.`,
		},
		{
			"missing PayloadVersion at root",
			pv.Finding{Code: pv.CodeMissingPayloadVersion, PayloadType: pv.TopLevelPayloadType},
			false, `A "Configuration" payload is missing PayloadVersion.`,
		},
		{
			"invalid PayloadUUID",
			pv.Finding{Code: pv.CodeMalformedUUID, PayloadType: "com.apple.wifi.managed"},
			false, `A "com.apple.wifi.managed" payload has an invalid PayloadUUID.`,
		},
		{
			"invalid PayloadIdentifier",
			pv.Finding{Code: pv.CodeMalformedIdentifier, PayloadType: "com.apple.wifi.managed"},
			false, `A "com.apple.wifi.managed" payload has an invalid PayloadIdentifier.`,
		},
		{
			"unknown code fallback",
			pv.Finding{Code: "SOMETHING_NEW", KeyPath: "k"},
			false, `"k" has a SOMETHING_NEW problem.`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, profileValidationSentence(c.finding, c.windows))
		})
	}
}

func TestProfileValidationMessages(t *testing.T) {
	t.Parallel()

	f := pv.Finding{Code: pv.CodeTypeMismatch, PayloadType: "com.apple.applicationaccess", KeyPath: "allowAirDrop", Expected: []string{"boolean"}}

	require.Equal(t,
		`Couldn't add profile. "allowAirDrop" expects a boolean. To skip validation, an admin can turn it off in Organization settings > Advanced options.`,
		profileValidationAddMessage(f, false))
	require.Equal(t,
		`Couldn't edit profile. "allowAirDrop" expects a boolean. To skip validation, an admin can turn it off in Organization settings > Advanced options.`,
		profileValidationEditMessage(f, false))
	require.Equal(t,
		"Couldn't edit configuration_profiles. \"Disable AirDrop\": \"allowAirDrop\" expects a boolean. To skip validation, set `enable_profile_validation` to false in your GitOps configuration.",
		profileValidationBatchMessage("Disable AirDrop", f, false))
}
