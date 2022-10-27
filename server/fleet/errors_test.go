package fleet

import (
	"fmt"
	"io"
	"strings"
	"testing"

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
