package bitlocker

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// A TPM-only protector unseals the volume with no user input. Adding one to a volume that already has a TPM+PIN
// protector therefore removes pre-boot authentication silently, because the PIN protector stays present and the host
// keeps reporting a PIN as set. See #52159.
func TestEnsureBootUnsealProtector(t *testing.T) {
	listErr := errors.New("WMI unavailable")
	addErr := errors.New("policy forbids a TPM-only protector")

	for _, tc := range []struct {
		name       string
		hasBoot    bool
		hasBootErr error
		addErr     error
		wantAdd    bool
		wantErr    error
	}{
		{
			name:    "no boot protector, so one is added",
			hasBoot: false,
			wantAdd: true,
		},
		{
			name:    "a boot protector already exists, so none is added",
			hasBoot: true,
			wantAdd: false,
		},
		{
			name:       "an unreadable protector list adds nothing",
			hasBootErr: listErr,
			wantAdd:    false,
			wantErr:    listErr,
		},
		{
			name:    "a failed add is reported",
			hasBoot: false,
			addErr:  addErr,
			wantAdd: true,
			wantErr: addErr,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var added bool
			err := ensureBootUnsealProtector(
				func() (bool, error) { return tc.hasBoot, tc.hasBootErr },
				func() error {
					added = true
					return tc.addErr
				},
			)

			require.Equal(t, tc.wantAdd, added, "add-protector expectation")
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Every protector that can release the volume master key at boot has to be in this set. One missing from it makes
// ensureBootUnsealProtector add a redundant TPM-only protector, which is the bypass above.
func TestTPMFamilyProtectorTypes(t *testing.T) {
	require.ElementsMatch(t, []int32{
		KeyProtectorTypeTPM,
		KeyProtectorTypeTPMAndPIN,
		KeyProtectorTypeTPMAndStartupKey,
		KeyProtectorTypeTPMAndPINAndStartupKey,
	}, TPMFamilyProtectorTypes)
}
