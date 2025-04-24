package setupexperience

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImageSize(t *testing.T) {
	url := "https://fleetdm.com/images/logo-blue-118x41@2x.png"

	size, err := getIconSize(url)
	require.NoError(t, err)
	require.Equal(t, wideIconSize, size)

	url = "https://fleetdm.com/legal/privacy"

	size, err = getIconSize(url)
	require.ErrorContains(t, err, "unknown format")
	require.Zero(t, size)
}
