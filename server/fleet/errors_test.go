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
