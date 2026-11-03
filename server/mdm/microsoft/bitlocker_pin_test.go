package microsoft_mdm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateBitLockerPIN(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		pin     string
		wantErr string
	}{
		{name: "minimum length", pin: "123456"},
		{name: "maximum length", pin: "12345678901234567890"},
		{name: "leading zeros are fine", pin: "000000"},
		// Fleet enables enhanced PINs, so letters, spaces, and symbols are accepted.
		{name: "letters", pin: "abcDEF"},
		{name: "spaces", pin: "123 456"},
		{name: "symbols", pin: "!\"#$%&'()*+,-./:;<=>"},
		{name: "brackets, backslash, backtick, and a trailing space", pin: "?@[\\]^_`{|}~ "},
		{name: "empty", pin: "", wantErr: BitLockerPINLengthMessage},
		{name: "one character short", pin: "12345", wantErr: BitLockerPINLengthMessage},
		{name: "one character long", pin: "123456789012345678901", wantErr: BitLockerPINLengthMessage},
		// The pre-boot screen uses the US English keyboard layout, so these could be set but never typed at startup.
		{name: "an accented letter", pin: "123456é", wantErr: BitLockerPINInvalidCharsMessage},
		{name: "a tab", pin: "123\t456", wantErr: BitLockerPINInvalidCharsMessage},
		// A multi-byte character must not be able to pass a byte-counted length check.
		{name: "multi-byte characters", pin: "１２３４５６", wantErr: BitLockerPINInvalidCharsMessage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateBitLockerPIN(tc.pin)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
