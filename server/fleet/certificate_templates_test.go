package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHostCertificateTemplate(t *testing.T) {
	t.Run("ToHostMDMProfile", func(t *testing.T) {
		tests := []struct {
			name        string
			template    *HostCertificateTemplate
			expectation func(*testing.T, HostMDMProfile)
		}{
			{
				name:     "nil template",
				template: nil,
				expectation: func(t *testing.T, profile HostMDMProfile) {
					require.Equal(t, "", profile.HostUUID)
					require.Equal(t, "", profile.Name)
					require.Equal(t, "", profile.Platform)
					require.Nil(t, profile.Status)
				},
			},
			{
				name: "maps fields correctly",
				template: &HostCertificateTemplate{
					HostUUID: "1234",
					Name:     "HostCertificate",
					Status:   MDMDeliveryVerified,
				},
				expectation: func(t *testing.T, profile HostMDMProfile) {
					require.Equal(t, "1234", profile.HostUUID)
					require.Equal(t, "HostCertificate", profile.Name)
					require.Equal(t, "android", profile.Platform)
					require.Equal(t, MDMDeliveryVerified, *profile.Status)

				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tt.expectation(t, tt.template.ToHostMDMProfile())
			})
		}
	})
}
