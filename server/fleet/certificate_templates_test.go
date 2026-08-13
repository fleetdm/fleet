package fleet

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
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
					require.Nil(t, profile.CertificateTemplateID)
					require.Nil(t, profile.RetryCount)
					require.Nil(t, profile.MaxRetries)
				},
			},
			{
				name: "maps fields correctly",
				template: &HostCertificateTemplate{
					HostUUID:      "1234",
					Name:          "HostCertificate",
					Status:        CertificateTemplateVerified,
					OperationType: MDMOperationTypeInstall,
				},
				expectation: func(t *testing.T, profile HostMDMProfile) {
					require.Equal(t, "1234", profile.HostUUID)
					require.Equal(t, "HostCertificate", profile.Name)
					require.Equal(t, "android", profile.Platform)
					require.EqualValues(t, CertificateTemplateVerified, *profile.Status)
					require.Equal(t, MDMOperationTypeInstall, profile.OperationType)
					require.Equal(t, AndroidCertificateTemplateProfileID, profile.ProfileUUID)
					require.Empty(t, profile.Detail)
				},
			},
			{
				name: "maps certificate_template_id correctly",
				template: &HostCertificateTemplate{
					CertificateTemplateID: 42,
				},
				expectation: func(t *testing.T, profile HostMDMProfile) {
					require.NotNil(t, profile.CertificateTemplateID)
					require.EqualValues(t, 42, *profile.CertificateTemplateID)
				},
			},
			{
				name: "maps detail correctly",
				template: &HostCertificateTemplate{
					Detail: ptr.String("some error"),
				},
				expectation: func(t *testing.T, profile HostMDMProfile) {
					require.Equal(t, "some error", profile.Detail)
				},
			},
			{
				name: "reports the retry allowance on a first attempt",
				template: &HostCertificateTemplate{
					Status:        CertificateTemplateDelivered,
					OperationType: MDMOperationTypeInstall,
				},
				expectation: func(t *testing.T, profile HostMDMProfile) {
					require.NotNil(t, profile.RetryCount)
					require.EqualValues(t, 0, *profile.RetryCount)
					require.NotNil(t, profile.MaxRetries)
					require.Equal(t, MaxCertificateInstallRetries, *profile.MaxRetries)
				},
			},
			{
				name: "reports an in-progress retry after a failure",
				template: &HostCertificateTemplate{
					Status:        CertificateTemplateDelivered,
					OperationType: MDMOperationTypeInstall,
					Detail:        new("Network error during SCEP enrollment"),
					RetryCount:    1,
				},
				expectation: func(t *testing.T, profile HostMDMProfile) {
					// The status stays in-progress while Fleet retries, so the retry count and
					// the preserved detail are the only signal that the last attempt failed.
					require.EqualValues(t, CertificateTemplateDelivered, *profile.Status)
					require.Equal(t, "Network error during SCEP enrollment", profile.Detail)
					require.NotNil(t, profile.RetryCount)
					require.EqualValues(t, 1, *profile.RetryCount)
					require.NotNil(t, profile.MaxRetries)
					require.Equal(t, MaxCertificateInstallRetries, *profile.MaxRetries)
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tt.expectation(t, tt.template.ToHostMDMProfile())
			})
		}
	})

	// A retry and a manual resend both leave a retry count on an in-progress certificate. The only
	// thing separating them is that a resend clears the detail to NULL while a retry always writes
	// one, even an empty string. That distinction does not survive into the API response, so it has
	// to be resolved here.
	t.Run("IsRetrying", func(t *testing.T) {
		tests := []struct {
			name     string
			template *HostCertificateTemplate
			expected bool
		}{
			{
				name:     "nil template",
				template: nil,
			},
			{
				name: "first delivery, never failed",
				template: &HostCertificateTemplate{
					Status: CertificateTemplateDelivered, OperationType: MDMOperationTypeInstall,
				},
			},
			{
				name: "retry carrying the reported failure",
				template: &HostCertificateTemplate{
					Status: CertificateTemplateDelivered, OperationType: MDMOperationTypeInstall,
					RetryCount: 1, Detail: new("SCEP failure"),
				},
				expected: true,
			},
			{
				name: "retry where the host reported no detail",
				template: &HostCertificateTemplate{
					Status: CertificateTemplatePending, OperationType: MDMOperationTypeInstall,
					RetryCount: 1, Detail: new(""),
				},
				expected: true,
			},
			{
				name: "final retry, at the maximum with no detail",
				template: &HostCertificateTemplate{
					Status: CertificateTemplateDelivered, OperationType: MDMOperationTypeInstall,
					RetryCount: MaxCertificateInstallRetries, Detail: new(""),
				},
				expected: true,
			},
			{
				name: "manual resend, at the maximum with the detail cleared",
				template: &HostCertificateTemplate{
					Status: CertificateTemplatePending, OperationType: MDMOperationTypeInstall,
					RetryCount: MaxCertificateInstallRetries, Detail: nil,
				},
			},
			{
				name: "terminally failed",
				template: &HostCertificateTemplate{
					Status: CertificateTemplateFailed, OperationType: MDMOperationTypeInstall,
					RetryCount: MaxCertificateInstallRetries, Detail: new("SCEP failure"),
				},
			},
			{
				name: "verified after a retry succeeded",
				template: &HostCertificateTemplate{
					Status: CertificateTemplateVerified, OperationType: MDMOperationTypeInstall,
					RetryCount: 1, Detail: new("SCEP failure"),
				},
			},
			{
				name: "removals are never retried",
				template: &HostCertificateTemplate{
					Status: CertificateTemplateDelivered, OperationType: MDMOperationTypeRemove,
					RetryCount: 1, Detail: new("SCEP failure"),
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				require.Equal(t, tt.expected, tt.template.IsRetrying())

				profile := tt.template.ToHostMDMProfile()
				if tt.template == nil {
					require.Nil(t, profile.Retrying)
					return
				}
				require.NotNil(t, profile.Retrying)
				require.Equal(t, tt.expected, *profile.Retrying)
			})
		}
	})

	// The retry fields describe Fleet's automatic retry of an Android certificate install, which
	// no other platform has. They must stay absent from every other profile's JSON rather than
	// showing up as a misleading zero.
	t.Run("retry fields are Android certificate only", func(t *testing.T) {
		for _, profile := range []HostMDMProfile{
			HostMDMWindowsProfile{Name: "Windows profile"}.ToHostMDMProfile(),
			HostMDMAndroidProfile{Name: "Android ONC profile"}.ToHostMDMProfile(),
			{Name: "Apple profile", Platform: "darwin"},
		} {
			t.Run(profile.Name, func(t *testing.T) {
				require.Nil(t, profile.RetryCount)
				require.Nil(t, profile.MaxRetries)

				encoded, err := json.Marshal(profile)
				require.NoError(t, err)
				require.NotContains(t, string(encoded), "retry_count")
				require.NotContains(t, string(encoded), "retrying")
				require.NotContains(t, string(encoded), "max_retries")
			})
		}
	})

	// The Android certificate rows do carry them, including a zero retry count on a first
	// delivery, which the UI needs in order to tell a first attempt from a retry.
	t.Run("retry fields are always present for Android certificates", func(t *testing.T) {
		template := &HostCertificateTemplate{Name: "BeyondCorp", Status: CertificateTemplateDelivered}

		encoded, err := json.Marshal(template.ToHostMDMProfile())
		require.NoError(t, err)
		require.Contains(t, string(encoded), `"retry_count":0`)
		require.Contains(t, string(encoded), `"max_retries":3`)
	})
}
