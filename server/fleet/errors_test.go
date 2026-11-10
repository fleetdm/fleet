package fleet

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserMessageErrors(t *testing.T) {
	var barString struct {
		Bar string `json:"bar"`
	}

	type inner struct {
		Foo       int      `json:"foo"`
		Strings   []string `json:"strings"`
		IntsInts  [][]int  `json:"ints_ints"`
		StringPtr *string  `json:"string_ptr"`
	}
	type outer struct {
		Inner inner `json:"inner"`
	}
	var nestedFoo outer

	cases := []struct {
		in  error
		out string
	}{
		{io.EOF, "EOF"},
		{JSONStrictDecode(strings.NewReader(`{"foo":1}`), &barString), `unsupported key provided: "foo"`},
		{JSONStrictDecode(strings.NewReader(`{"bar":1}`), &barString), `invalid value type at 'bar': expected string but got number`},
		{JSONStrictDecode(strings.NewReader(`{"inner":{"foo":"bar"}}`), &nestedFoo), `invalid value type at 'inner.foo': expected int but got string`},
		{JSONStrictDecode(strings.NewReader(`{"inner":{"strings":true}}`), &nestedFoo), `invalid value type at 'inner.strings': expected array of strings but got bool`},
		{JSONStrictDecode(strings.NewReader(`{"inner":{"ints_ints":true}}`), &nestedFoo), `invalid value type at 'inner.ints_ints': expected array of array of ints but got bool`},
		{JSONStrictDecode(strings.NewReader(`{"inner":{"string_ptr":true}}`), &nestedFoo), `invalid value type at 'inner.string_ptr': expected string but got bool`},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%T: %[1]q", c.in), func(t *testing.T) {
			ume := NewUserMessageError(c.in, 0)
			got := ume.UserMessage()
			require.Contains(t, got, c.out)
		})
	}
}

func TestFleetdErrors(t *testing.T) {
	testTime, err := time.Parse(time.RFC3339, "1969-06-19T21:44:05Z")
	require.NoError(t, err)
	ferr := FleetdError{
		ErrorSource:         "orbit",
		ErrorSourceVersion:  "1.1.1",
		ErrorTimestamp:      testTime,
		ErrorMessage:        "test message",
		ErrorAdditionalInfo: map[string]any{"foo": "bar"},
	}

	require.Equal(t, "test message", ferr.Error())
	assert.Equal(t, map[string]any{
		"vital":                 ferr.Vital,
		"error_source":          ferr.ErrorSource,
		"error_source_version":  ferr.ErrorSourceVersion,
		"error_timestamp":       ferr.ErrorTimestamp,
		"error_message":         ferr.ErrorMessage,
		"error_additional_info": ferr.ErrorAdditionalInfo,
	}, ferr.ToMap())

	logBuf := bytes.NewBuffer(nil)
	logger := zerolog.New(logBuf)
	zevent := logger.Log()
	ferr.MarshalZerologObject(zevent)
	zevent.Send()
	assert.JSONEq(t,
		`{"error_source":"orbit","error_source_version":"1.1.1","error_timestamp":"1969-06-19T21:44:05Z","error_message":"test message","error_additional_info":{"foo":"bar"},"vital":false}`,
		logBuf.String())
}

func TestParamRequiredDeprecationError(t *testing.T) {
	err := NewParamRequiredDeprecationError("host_identifier")
	require.Equal(t, `The "host_identifier" parameter is now required. Learn more: https://fleetdm.com/learn-more-about/fleet-5-migration`, err.Error())

	// Encoded as a bad request (400) and logged as a client error.
	var asError error = err
	_, isBadRequest := asError.(interface{ BadRequestError() []map[string]string })
	require.True(t, isBadRequest)
	clientErr, ok := asError.(interface{ IsClientError() bool })
	require.True(t, ok)
	require.True(t, clientErr.IsClientError())
}

func TestRemovedFleetVariable(t *testing.T) {
	const removedMsg = `The "$FLEET_VAR_HOST_END_USER_EMAIL_IDP" variable is not available anymore. ` +
		`What to use instead: https://fleetdm.com/learn-more-about/fleet-5-migration`

	require.NoError(t, FindRemovedFleetVariable())
	require.NoError(t, FindRemovedFleetVariable("HOST_UUID", "NOT_A_VARIABLE", "HOST_END_USER_EMAIL_IDP_X"))
	err := FindRemovedFleetVariable("HOST_UUID", "HOST_END_USER_EMAIL_IDP")
	require.EqualError(t, err, removedMsg)

	// Encoded as a validation error (422) and logged as a client error.
	invalid, ok := err.(interface{ Invalid() []map[string]string })
	require.True(t, ok)
	require.Equal(t, []map[string]string{{"name": "base", "reason": removedMsg}}, invalid.Invalid())
	clientErr, ok := err.(interface{ IsClientError() bool })
	require.True(t, ok)
	require.True(t, clientErr.IsClientError())

	for _, v := range []string{"$FLEET_VAR_HOST_END_USER_EMAIL_IDP", "${FLEET_VAR_HOST_END_USER_EMAIL_IDP}"} {
		androidProfile := MDMAndroidConfigProfile{Name: "p", RawJSON: []byte(`{"name": "` + v + `"}`)}
		for name, err := range map[string]error{
			"android profile":    androidProfile.ValidateUserProvided(true),
			"android app config": ValidateAndroidAppConfiguration([]byte(`{"managedConfiguration": {"email": "` + v + `"}}`)),
			"apple app config": ValidateAppleAppConfiguration([]byte(
				`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict><key>email</key><string>` + v + `</string></dict></plist>`)),
		} {
			var removedErr *RemovedFleetVariableError
			require.ErrorAs(t, err, &removedErr, "%s with %s", name, v)
			require.EqualError(t, err, removedMsg, "%s with %s", name, v)
		}
	}

	// The tree walk of an Apple app config catches a variable the raw scan can't see.
	err = ValidateAppleAppConfiguration([]byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict>` +
		`<key>email</key><string>&#36;FLEET_VAR_HOST_END_USER_EMAIL_IDP</string></dict></plist>`))
	require.EqualError(t, err, removedMsg)

	// The removed variable wins over another unsupported one, whatever the order.
	err = ValidateAppleAppConfiguration([]byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict>` +
		`<key>a</key><string>$FLEET_VAR_BOZO</string><key>b</key><string>$FLEET_VAR_HOST_END_USER_EMAIL_IDP</string></dict></plist>`))
	require.EqualError(t, err, removedMsg)

	// A longer name isn't the removed variable.
	err = ValidateAndroidAppConfiguration([]byte(`{"managedConfiguration": {"email": "$FLEET_VAR_HOST_END_USER_EMAIL_IDP_X"}}`))
	require.EqualError(t, err, "Couldn't update configuration. Unsupported variable $FLEET_VAR_HOST_END_USER_EMAIL_IDP_X.")
}
