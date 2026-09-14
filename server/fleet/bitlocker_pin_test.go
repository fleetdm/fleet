package fleet

import (
	"testing"
	"time"

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
			t.Parallel()

			err := ValidateBitLockerPIN(tc.pin)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestBitLockerPINRequestExpired(t *testing.T) {
	t.Parallel()

	now := time.Now()

	for _, tc := range []struct {
		name string
		req  *HostBitLockerPINRequest
		want bool
	}{
		{name: "nil request", req: nil},
		{
			name: "fresh pending",
			req:  &HostBitLockerPINRequest{Status: BitLockerPINRequestPending, CreatedAt: now.Add(-time.Second)},
		},
		{
			name: "stale pending",
			req:  &HostBitLockerPINRequest{Status: BitLockerPINRequestPending, CreatedAt: now.Add(-BitLockerPINRequestTTL - time.Second)},
			want: true,
		},
		// Once the agent has the PIN the outcome is its to report, so age stops mattering.
		{
			name: "old delivered",
			req:  &HostBitLockerPINRequest{Status: BitLockerPINRequestDelivered, CreatedAt: now.Add(-time.Hour)},
		},
		{
			name: "old set",
			req:  &HostBitLockerPINRequest{Status: BitLockerPINRequestSet, CreatedAt: now.Add(-time.Hour)},
		},
		{
			name: "old failed",
			req:  &HostBitLockerPINRequest{Status: BitLockerPINRequestFailed, CreatedAt: now.Add(-time.Hour)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, tc.req.Expired(now))
		})
	}
}

func TestHostNeedsBitLockerPIN(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		de   *HostMDMDiskEncryption
		want bool
	}{
		{name: "no disk encryption status"},
		{name: "no action required", de: &HostMDMDiskEncryption{}},
		{
			name: "create pin",
			de:   &HostMDMDiskEncryption{ActionRequired: new(ActionRequiredCreatePIN)},
			want: true,
		},
		// A host waiting on a restart is action-required for a different reason, and asking it for a PIN would be
		// asking for something Windows will not accept yet.
		{
			name: "restart required",
			de:   &HostMDMDiskEncryption{ActionRequired: new(ActionRequiredRestart)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, HostNeedsBitLockerPIN(tc.de))
		})
	}
}
