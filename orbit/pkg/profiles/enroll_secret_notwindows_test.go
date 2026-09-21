//go:build !windows

package profiles

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnrollSecretNotImplementedOutsideWindows(t *testing.T) {
	secret, err := GetEnrollSecret()
	require.ErrorIs(t, err, ErrNotImplemented)
	require.Empty(t, secret)

	require.ErrorIs(t, ClearEnrollSecret(), ErrNotImplemented)
}
