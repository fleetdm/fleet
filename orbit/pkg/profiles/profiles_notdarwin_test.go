//go:build !darwin

package profiles

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetFleetdConfig(t *testing.T) {
	config, err := GetFleetdConfig()
	require.ErrorIs(t, ErrNotImplemented, err)
	require.Nil(t, config)
}

func TestIsEnrolledIntoMatchingURL(t *testing.T) {
	enrolled, err := IsEnrolledIntoMatchingURL()
	require.ErrorIs(t, ErrNotImplemented, err)
	require.False(t, enrolled)
}
