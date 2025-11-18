package fleet

import "testing"

func TestHostCertificateTemplate(t *testing.T) {
	t.Run("ToHostMDMProfile", func(t *testing.T) {
		tests := []struct {
			name     string
			template *HostCertificateTemplate
			expected HostMDMProfile
		}{
			{
				name:     "nil template",
				template: nil,
				expected: HostMDMProfile{},
			},
			{
				name: "verified status",
				template: &HostCertificateTemplate{
					HostUUID: "1234",
					Name:     "HostCertificate",
					Status:   MDMDeliveryVerified,
				},
				expected: HostMDMProfile{
					HostUUID: "1234",
					Name:     "HostCertificate",
					Platform: "android",
					Status:   &MDMDeliveryVerified,
				},
			},
			{
				name: "verifying status",
				template: &HostCertificateTemplate{
					HostUUID: "5678",
					Name:     "VerifyingHost",
					Status:   MDMDeliveryVerifying,
				},
				expected: HostMDMProfile{
					HostUUID: "5678",
					Name:     "VerifyingHost",
					Platform: "android",
					Status:   &MDMDeliveryVerifying,
				},
			},
			{
				name: "pending status",
				template: &HostCertificateTemplate{
					HostUUID: "91011",
					Name:     "PendingHost",
					Status:   MDMDeliveryPending,
				},
				expected: HostMDMProfile{
					HostUUID: "91011",
					Name:     "PendingHost",
					Platform: "android",
					Status:   &MDMDeliveryPending,
				},
			},
			{
				name: "failed status",
				template: &HostCertificateTemplate{
					HostUUID: "121314",
					Name:     "FailedHost",
					Status:   MDMDeliveryFailed,
				},
				expected: HostMDMProfile{
					HostUUID: "121314",
					Name:     "FailedHost",
					Platform: "android",
					Status:   &MDMDeliveryFailed,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.template.ToHostMDMProfile()
				if result != tt.expected {
					t.Errorf("unexpected result: got %v, want %v", result, tt.expected)
				}
			})
		}
	})
}
