package bitlocker

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// A TPM-only protector unseals the volume with no user input, so adding one to a volume that already has a TPM+PIN
// protector silently removes pre-boot authentication. That is the bug this function exists to prevent, and it cannot
// be caught on a non-Windows CI runner any other way. See #52159.
func TestEnsureBootUnsealProtector(t *testing.T) {
	listErr := errors.New("WMI unavailable")

	for _, tc := range []struct {
		name       string
		hasBoot    bool
		hasBootErr error
		wantAdd    bool
	}{
		{name: "no boot protector, so one is added", wantAdd: true},
		{name: "a boot protector already exists, so none is added", hasBoot: true},
		// Adding one blind risks the bypass above, so an unreadable list is the safer failure.
		{name: "an unreadable protector list adds nothing", hasBootErr: listErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var added bool
			err := ensureBootUnsealProtector(
				func() (bool, error) { return tc.hasBoot, tc.hasBootErr },
				func() error { added = true; return nil },
			)

			require.Equal(t, tc.wantAdd, added)
			if tc.hasBootErr != nil {
				require.ErrorIs(t, err, tc.hasBootErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
