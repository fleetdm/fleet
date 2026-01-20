package apple_mdm

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestMDMAppleEnrollURL(t *testing.T) {
	cases := []struct {
		appConfig   *fleet.AppConfig
		expectedURL string
	}{
		{
			appConfig: &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					ServerURL: "https://foo.example.com",
				},
			},
			expectedURL: "https://foo.example.com/api/mdm/apple/enroll?token=tok",
		},
		{
			appConfig: &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					ServerURL: "https://foo.example.com/",
				},
			},
			expectedURL: "https://foo.example.com/api/mdm/apple/enroll?token=tok",
		},
	}

	for _, tt := range cases {
		enrollURL, err := EnrollURL("tok", tt.appConfig)
		require.NoError(t, err)
		require.Equal(t, tt.expectedURL, enrollURL)
	}
}

func TestGenerateRandomPin(t *testing.T) {
	for i := 1; i <= 100; i++ {
		pin, err := GenerateRandomPin(i)
		require.NoError(t, err)
		require.Len(t, pin, i)
	}
}
