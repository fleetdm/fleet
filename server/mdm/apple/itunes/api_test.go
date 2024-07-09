package itunes

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetBaseURL(t *testing.T) {
	t.Run("Default URL", func(t *testing.T) {
		os.Setenv("FLEET_DEV_ITUNES_URL", "")
		require.Equal(t, "https://itunes.apple.com/lookup", getBaseURL())
	})

	t.Run("Custom URL", func(t *testing.T) {
		customURL := "http://localhost:8000"
		os.Setenv("FLEET_DEV_ITUNES_URL", customURL)
		require.Equal(t, customURL, getBaseURL())
	})
}
