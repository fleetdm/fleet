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
	enrolled, err := IsEnrolledIntoMatchingURL("https://test.example.com")
	require.ErrorIs(t, ErrNotImplemented, err)
	require.False(t, enrolled)
}

func TestCheckAssignedEnrollmentProfile(t *testing.T) {
	err := CheckAssignedEnrollmentProfile("https://test.example.com")
	require.ErrorIs(t, ErrNotImplemented, err)
}
