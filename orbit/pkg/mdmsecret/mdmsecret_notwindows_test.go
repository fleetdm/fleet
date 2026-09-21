//go:build !windows

package mdmsecret

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNotImplementedOutsideWindows(t *testing.T) {
	secret, err := Read()
	require.ErrorIs(t, err, ErrNotImplemented)
	require.Empty(t, secret)

	require.ErrorIs(t, Clear(), ErrNotImplemented)
}
