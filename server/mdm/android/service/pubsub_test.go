package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

// sha256 of "TestBrand:test-serial". Will need to be updated if our test enrollment message changes
var testBrandTestSerialHashed = "9c311e05af14f958bd65188796e41fcc8a7b0ff913bfea4f11f31c96c6f052b0"

func createAndroidService(t *testing.T) (android.Service, *AndroidMockDS) {
	androidAPIClient := android_mock.Client{}
	androidAPIClient.InitCommonMocks()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockDS := InitCommonDSMocks()
	svc, err := NewServiceWithClient(logger, mockDS, &androidAPIClient, "test-private-key", &mockDS.DataStore, noopNewActivity, config.AndroidAgentConfig{})
	require.NoError(t, err)

	return svc, mockDS
}

func TestPubSubEnrollment(t *testing.T) {
	svc, mockDS := createAndroidService(t)

	globalSecret := "global"
	teamSecret := "team"
	teamID := uint(1)

	mockDS.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
		switch secret {
		case globalSecret:
			return &fleet.EnrollSecret{
				Secret: globalSecret,
				TeamID: nil,
			}, nil
		case teamSecret:
			return &fleet.EnrollSecret{
				Secret: teamSecret,
				TeamID: &teamID, // Remember to create the team in each test that uses it.
			}, nil
		}

		return nil, common_mysql.NotFound("enroll secret")
	}

	mockDS.AndroidHostLiteFunc = func(ctx context.Context, enterpriseSpecificID string) (*fleet.AndroidHost, error) {
		return nil, common_mysql.NotFound("android host lite mock")
	}

	t.Run("errors", func(t *testing.T) {
		t.Run("if android mdm is not configured", func(t *testing.T) {
			mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{
					MDM: fleet.MDM{
						AndroidEnabledAndConfigured: false,
					},
				}, nil
			}

			enrollmentToken := enrollmentTokenRequest{
				EnrollSecret: "invalid",
			}
			enrollTokenData, err := json.Marshal(enrollmentToken)
			require.NoError(t, err)
			enrollmentMessage := createEnrollmentMessage(t, androidmanagement.Device{
				Name:                createAndroidDeviceId("test-android"),
				EnrollmentTokenData: string(enrollTokenData),
			})
			err = svc.ProcessPubSubPush(context.Background(), "invalid", enrollmentMessage)
			require.Equal(t, "validation failed: android Android MDM is NOT configured", err.Error())
			sc, ok := err.(interface{ Status() int })
			require.True(t, ok, "error should implement Status() interface")
			require.Equal(t, http.StatusOK, sc.Status())
		})

		t.Run("if android token is invalid", func(t *testing.T) {
			mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{
					MDM: fleet.MDM{
						AndroidEnabledAndConfigured: true,
					},
				}, nil
			}

			enrollmentToken := enrollmentTokenRequest{
				EnrollSecret: "invalid",
			}
			enrollTokenData, err := json.Marshal(enrollmentToken)
			require.NoError(t, err)
			enrollmentMessage := createEnrollmentMessage(t, androidmanagement.Device{
				Name:                createAndroidDeviceId("test-android"),
				EnrollmentTokenData: string(enrollTokenData),
			})
			err = svc.ProcessPubSubPush(context.Background(), "invalid", enrollmentMessage)
			require.Error(t, err)
			require.Equal(t, "Authentication failed", err.Error())
		})

		t.Run("if enroll secret is invalid", func(t *testing.T) {
			mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{
					MDM: fleet.MDM{
						AndroidEnabledAndConfigured: true,
					},
				}, nil
			}

			enrollmentToken := enrollmentTokenRequest{
				EnrollSecret: "invalid",
			}
			enrollTokenData, err := json.Marshal(enrollmentToken)
			require.NoError(t, err)
			enrollmentMessage := createEnrollmentMessage(t, androidmanagement.Device{
				Name:                createAndroidDeviceId("test-android"),
				EnrollmentTokenData: string(enrollTokenData),
			})
			err = svc.ProcessPubSubPush(context.Background(), "value", enrollmentMessage)
			require.Error(t, err)
		})
	})

	t.Run("successfully enrolls", func(t *testing.T) {
		t.Run("device into a team with valid enroll secret", func(t *testing.T) {
			mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{
					MDM: fleet.MDM{
						AndroidEnabledAndConfigured: true,
					},
				}, nil
			}

			mockDS.NewAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, companyOwned bool) (*fleet.AndroidHost, error) {
				require.False(t, companyOwned)
				return &fleet.AndroidHost{Host: &fleet.Host{}}, nil
			}

			enrollmentToken := enrollmentTokenRequest{
				EnrollSecret: "global",
			}
			enrollTokenData, err := json.Marshal(enrollmentToken)
			require.NoError(t, err)
			enrollmentMessage := createEnrollmentMessage(t, androidmanagement.Device{
				Name:                createAndroidDeviceId("test-android"),
				EnrollmentTokenData: string(enrollTokenData),
			})
			err = svc.ProcessPubSubPush(context.Background(), "value", enrollmentMessage)
			require.NoError(t, err)

			require.False(t, mockDS.AssociateHostMDMIdPAccountFuncInvoked)
			require.True(t, mockDS.NewAndroidHostFuncInvoked)
		})

		t.Run("device into a team and associates idp if byod idp id is set", func(t *testing.T) {
			mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{
					MDM: fleet.MDM{
						AndroidEnabledAndConfigured: true,
					},
				}, nil
			}

			mockDS.NewAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, companyOwned bool) (*fleet.AndroidHost, error) {
				require.False(t, companyOwned)
				return &fleet.AndroidHost{Host: &fleet.Host{}}, nil
			}
			mockDS.AssociateHostMDMIdPAccountFunc = func(ctx context.Context, hostUUID, accountUUID string) error {
				return nil
			}
			mockDS.MaybeAssociateHostWithScimUserFunc = func(ctx context.Context, hostID uint) error {
				return nil
			}

			enrollmentToken := enrollmentTokenRequest{
				EnrollSecret: "global",
				IdpUUID:      "mock-id",
			}
			enrollTokenData, err := json.Marshal(enrollmentToken)
			require.NoError(t, err)
			enrollmentMessage := createEnrollmentMessage(t, androidmanagement.Device{
				Name:                createAndroidDeviceId("test-android"),
				EnrollmentTokenData: string(enrollTokenData),
			})
			err = svc.ProcessPubSubPush(t.Context(), "value", enrollmentMessage)
			require.NoError(t, err)

			require.True(t, mockDS.AssociateHostMDMIdPAccountFuncInvoked)
			require.True(t, mockDS.NewAndroidHostFuncInvoked)
			require.True(t, mockDS.MaybeAssociateHostWithScimUserFuncInvoked)
		})

		t.Run("associates scim user with correct host ID after idp association", func(t *testing.T) {
			mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{
					MDM: fleet.MDM{
						AndroidEnabledAndConfigured: true,
					},
				}, nil
			}

			expectedHostID := uint(42)
			mockDS.NewAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, companyOwned bool) (*fleet.AndroidHost, error) {
				return &fleet.AndroidHost{Host: &fleet.Host{ID: expectedHostID}}, nil
			}

			var capturedIdpHostUUID, capturedIdpAcctUUID string
			mockDS.AssociateHostMDMIdPAccountFunc = func(ctx context.Context, hostUUID, accountUUID string) error {
				capturedIdpHostUUID = hostUUID
				capturedIdpAcctUUID = accountUUID
				return nil
			}

			var capturedScimHostID uint
			mockDS.MaybeAssociateHostWithScimUserFunc = func(ctx context.Context, hostID uint) error {
				capturedScimHostID = hostID
				return nil
			}

			idpUUID := "test-idp-uuid"
			enrollmentToken := enrollmentTokenRequest{
				EnrollSecret: "global",
				IdpUUID:      idpUUID,
			}
			enrollTokenData, err := json.Marshal(enrollmentToken)
			require.NoError(t, err)
			enrollmentMessage := createEnrollmentMessage(t, androidmanagement.Device{
				Name:                createAndroidDeviceId("test-android-scim"),
				EnrollmentTokenData: string(enrollTokenData),
			})
			err = svc.ProcessPubSubPush(t.Context(), "value", enrollmentMessage)
			require.NoError(t, err)

			require.True(t, mockDS.AssociateHostMDMIdPAccountFuncInvoked)
			require.NotEmpty(t, capturedIdpHostUUID)
			require.Equal(t, idpUUID, capturedIdpAcctUUID)

			require.True(t, mockDS.MaybeAssociateHostWithScimUserFuncInvoked)
			require.Equal(t, expectedHostID, capturedScimHostID)
		})

		t.Run("populates operating_systems with Android name and version on enrollment", func(t *testing.T) {
			// Regression test for https://github.com/fleetdm/fleet/issues/45711.
			// Without this, filtering hosts with `os_name=Android&os_version=<v>`
			// returns nothing because the operating_systems table is empty for
			// Android hosts.
			mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{
					MDM: fleet.MDM{AndroidEnabledAndConfigured: true},
				}, nil
			}

			expectedHostID := uint(99)
			mockDS.NewAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, companyOwned bool) (*fleet.AndroidHost, error) {
				return &fleet.AndroidHost{Host: &fleet.Host{ID: expectedHostID}}, nil
			}
			var capturedHostID uint
			var capturedOS fleet.OperatingSystem
			mockDS.UpdateHostOperatingSystemFunc = func(ctx context.Context, hostID uint, hostOS fleet.OperatingSystem) error {
				capturedHostID = hostID
				capturedOS = hostOS
				return nil
			}

			enrollmentToken := enrollmentTokenRequest{EnrollSecret: "global"}
			enrollTokenData, err := json.Marshal(enrollmentToken)
			require.NoError(t, err)
			deviceInfo := androidmanagement.Device{
				Name:                createAndroidDeviceId("test-android-os"),
				EnrollmentTokenData: string(enrollTokenData),
			}
			enrollmentMessage := createEnrollmentMessage(t, deviceInfo)
			// createEnrollmentMessage sets AndroidVersion="1"; override to a more
			// realistic version so we verify it's passed through unchanged.
			data, err := base64.StdEncoding.DecodeString(enrollmentMessage.Data)
			require.NoError(t, err)
			var decoded androidmanagement.Device
			require.NoError(t, json.Unmarshal(data, &decoded))
			decoded.SoftwareInfo.AndroidVersion = "16"
			reEncoded, err := json.Marshal(decoded)
			require.NoError(t, err)
			enrollmentMessage.Data = base64.StdEncoding.EncodeToString(reEncoded)

			err = svc.ProcessPubSubPush(t.Context(), "value", enrollmentMessage)
			require.NoError(t, err)

			require.True(t, mockDS.UpdateHostOperatingSystemFuncInvoked)
			require.Equal(t, expectedHostID, capturedHostID)
			require.Equal(t, "Android", capturedOS.Name)
			require.Equal(t, "16", capturedOS.Version)
			require.Equal(t, "android", capturedOS.Platform)
		})

		t.Run("creates device as company-owned if specified in enrollment message", func(t *testing.T) {
			mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{
					MDM: fleet.MDM{
						AndroidEnabledAndConfigured: true,
					},
				}, nil
			}

			mockDS.NewAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, companyOwned bool) (*fleet.AndroidHost, error) {
				require.True(t, companyOwned)
				require.Equal(t, testBrandTestSerialHashed, host.UUID)
				return &fleet.AndroidHost{Host: &fleet.Host{}}, nil
			}
			mockDS.AssociateHostMDMIdPAccountFunc = func(ctx context.Context, hostUUID, accountUUID string) error {
				return nil
			}

			enrollmentToken := enrollmentTokenRequest{
				EnrollSecret: "global",
				IdpUUID:      "mock-id",
			}
			enrollTokenData, err := json.Marshal(enrollmentToken)
			require.NoError(t, err)
			enrollmentMessage := createEnrollmentMessage(t, androidmanagement.Device{
				Name:                createAndroidDeviceId("test-android"),
				EnrollmentTokenData: string(enrollTokenData),
				Ownership:           DeviceOwnershipCompanyOwned,
			})
			err = svc.ProcessPubSubPush(context.Background(), "value", enrollmentMessage)
			require.NoError(t, err)

			require.True(t, mockDS.AssociateHostMDMIdPAccountFuncInvoked)
			require.True(t, mockDS.NewAndroidHostFuncInvoked)
		})
	})

	t.Run("re-enrollment updates IdP association", func(t *testing.T) {
		mockDS.NewAndroidHostFuncInvoked = false
		mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				MDM: fleet.MDM{AndroidEnabledAndConfigured: true},
			}, nil
		}

		const existingHostUUID = "EXISTING-HOST-UUID"
		// Return an existing host so enrollHost takes the re-enrollment path
		mockDS.AndroidHostLiteFunc = func(ctx context.Context, esID string) (*fleet.AndroidHost, error) {
			return &fleet.AndroidHost{
				Host: &fleet.Host{
					ID:   10,
					UUID: existingHostUUID,
				},
				Device: &android.Device{
					HostID:               10,
					DeviceID:             "existing-device",
					EnterpriseSpecificID: new(existingHostUUID),
				},
			}, nil
		}

		mockDS.UpdateAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, fromEnroll, companyOwned bool) error {
			return nil
		}
		mockDS.DeleteAllHostCertificateTemplatesFunc = func(ctx context.Context, hostUUID string) error {
			return nil
		}

		var capturedHostUUID, capturedIdpUUID string
		mockDS.AssociateHostMDMIdPAccountFuncInvoked = false
		mockDS.AssociateHostMDMIdPAccountFunc = func(ctx context.Context, hostUUID, accountUUID string) error {
			capturedHostUUID = hostUUID
			capturedIdpUUID = accountUUID
			return nil
		}

		enrollmentToken := enrollmentTokenRequest{
			EnrollSecret: "global",
			IdpUUID:      "new-user-idp-uuid",
		}
		enrollTokenData, err := json.Marshal(enrollmentToken)
		require.NoError(t, err)
		enrollmentMessage := createEnrollmentMessage(t, androidmanagement.Device{
			Name:                createAndroidDeviceId("test-re-enroll"),
			EnrollmentTokenData: string(enrollTokenData),
		})

		err = svc.ProcessPubSubPush(t.Context(), "value", enrollmentMessage)
		require.NoError(t, err)

		require.True(t, mockDS.AssociateHostMDMIdPAccountFuncInvoked)
		require.Equal(t, existingHostUUID, capturedHostUUID)
		require.Equal(t, "new-user-idp-uuid", capturedIdpUUID)
		// Re-enrollment should update, not create a new host
		require.False(t, mockDS.NewAndroidHostFuncInvoked)
	})
}

func TestStatusReportPolicyValidation(t *testing.T) {
	svc, mockDS := createAndroidService(t)

	androidDevice := &fleet.AndroidHost{
		Host: &fleet.Host{
			UUID: uuid.NewString(),
		},
		Device: &android.Device{
			DeviceID: createAndroidDeviceId("test"),
		},
	}
	mockDS.AndroidHostLiteFunc = func(ctx context.Context, enterpriseSpecificID string) (*fleet.AndroidHost, error) {
		return androidDevice, nil
	}
	mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			MDM: fleet.MDM{
				AndroidEnabledAndConfigured: true,
			},
		}, nil
	}
	mockDS.UpdateAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, fromEnroll, companyOwned bool) error {
		return nil
	}

	t.Run("single install pending profile with empty compliance details", func(t *testing.T) {
		policyVersion := new(1)

		installPendingProfile := &fleet.MDMAndroidProfilePayload{
			ProfileUUID:             uuid.NewString(),
			ProfileName:             "a",
			HostUUID:                androidDevice.UUID,
			Status:                  &fleet.MDMDeliveryPending,
			OperationType:           fleet.MDMOperationTypeInstall,
			IncludedInPolicyVersion: policyVersion,
		}
		mockDS.ListHostMDMAndroidProfilesPendingOrFailedInstallWithVersionFunc = func(ctx context.Context, hostUUID string, version int64) ([]*fleet.MDMAndroidProfilePayload, error) {
			return []*fleet.MDMAndroidProfilePayload{
				installPendingProfile,
			}, nil
		}
		mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFunc = func(ctx context.Context, hostUUID string, version int64) ([]*fleet.HostAndroidVPPSoftwareInstall, error) {
			return nil, nil
		}
		mockDS.BulkUpsertMDMAndroidHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAndroidProfilePayload) error {
			require.Len(t, payload, 1)
			require.Equal(t, installPendingProfile.ProfileUUID, payload[0].ProfileUUID)
			require.Equal(t, fleet.MDMDeliveryVerified, *payload[0].Status)
			return nil
		}
		mockDS.BulkDeleteMDMAndroidHostProfilesFunc = func(ctx context.Context, hostUUID string, policyVersionID int64) error {
			return nil
		}

		enrollmentMessage := createStatusReportMessage(t, androidDevice.UUID, "test", createAndroidDeviceId("test-policy"), policyVersion, nil)

		err := svc.ProcessPubSubPush(context.Background(), "value", &enrollmentMessage)
		require.NoError(t, err)

		require.True(t, mockDS.ListHostMDMAndroidProfilesPendingOrFailedInstallWithVersionFuncInvoked)
		require.False(t, mockDS.GetAndroidPolicyRequestByUUIDFuncInvoked)
		require.True(t, mockDS.BulkUpsertMDMAndroidHostProfilesFuncInvoked)
		require.True(t, mockDS.BulkDeleteMDMAndroidHostProfilesFuncInvoked)
		mockDS.ListHostMDMAndroidProfilesPendingOrFailedInstallWithVersionFuncInvoked = false
		mockDS.BulkDeleteMDMAndroidHostProfilesFuncInvoked = false
		mockDS.BulkUpsertMDMAndroidHostProfilesFuncInvoked = false
	})

	t.Run("compliance details has failure", func(t *testing.T) {
		policyVersion := new(1)

		policyRequestUUID := uuid.NewString()
		installPendingProfile1 := &fleet.MDMAndroidProfilePayload{
			ProfileUUID:             uuid.NewString(),
			ProfileName:             "a",
			HostUUID:                androidDevice.UUID,
			Status:                  &fleet.MDMDeliveryPending,
			OperationType:           fleet.MDMOperationTypeInstall,
			IncludedInPolicyVersion: policyVersion,
			PolicyRequestUUID:       &policyRequestUUID,
		}

		installPendingProfile2 := &fleet.MDMAndroidProfilePayload{
			ProfileUUID:             uuid.NewString(),
			ProfileName:             "b",
			HostUUID:                androidDevice.UUID,
			Status:                  &fleet.MDMDeliveryPending,
			OperationType:           fleet.MDMOperationTypeInstall,
			IncludedInPolicyVersion: policyVersion,
			PolicyRequestUUID:       &policyRequestUUID,
		}

		mockDS.GetAndroidPolicyRequestByUUIDFunc = func(ctx context.Context, id string) (*android.MDMAndroidPolicyRequest, error) {
			if id == policyRequestUUID {
				payload, err := json.Marshal(map[string]any{
					"policy": map[string]any{
						"DefaultPermissionPolicy": "invalid",
						"cameraDisabled":          true,
					},
					"metadata": map[string]any{
						"settings_origin": map[string]string{
							"DefaultPermissionPolicy": installPendingProfile1.ProfileUUID,
							"cameraDisabled":          installPendingProfile2.ProfileUUID,
						},
					},
				})
				require.NoError(t, err)
				return &android.MDMAndroidPolicyRequest{
					Payload: payload,
				}, nil
			}

			return nil, errors.New("something went wrong")
		}

		mockDS.ListHostMDMAndroidProfilesPendingOrFailedInstallWithVersionFunc = func(ctx context.Context, hostUUID string, version int64) ([]*fleet.MDMAndroidProfilePayload, error) {
			return []*fleet.MDMAndroidProfilePayload{
				installPendingProfile1,
				installPendingProfile2,
			}, nil
		}
		mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFunc = func(ctx context.Context, hostUUID string, version int64) ([]*fleet.HostAndroidVPPSoftwareInstall, error) {
			return nil, nil
		}
		mockDS.BulkUpsertMDMAndroidHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAndroidProfilePayload) error {
			require.Len(t, payload, 2)
			for _, profile := range payload {
				switch profile.ProfileUUID {
				case installPendingProfile1.ProfileUUID:
					require.Equal(t, installPendingProfile1.ProfileUUID, profile.ProfileUUID)
					require.Equal(t, fleet.MDMDeliveryFailed, *profile.Status)
				case installPendingProfile2.ProfileUUID:
					require.Equal(t, installPendingProfile2.ProfileUUID, profile.ProfileUUID)
					require.Equal(t, fleet.MDMDeliveryVerified, *profile.Status)
				default:
					require.Fail(t, "All profiles upserted should have an if statement verifying status.")
				}
			}

			return nil
		}
		mockDS.BulkDeleteMDMAndroidHostProfilesFunc = func(ctx context.Context, hostUUID string, policyVersionID int64) error {
			return nil
		}

		enrollmentMessage := createStatusReportMessage(t, androidDevice.UUID, "test", createAndroidDeviceId("test-policy"), policyVersion, []*androidmanagement.NonComplianceDetail{
			{
				SettingName:         "DefaultPermissionPolicy",
				NonComplianceReason: "INVALID_VALUE",
			},
		})

		err := svc.ProcessPubSubPush(context.Background(), "value", &enrollmentMessage)
		require.NoError(t, err)

		require.True(t, mockDS.GetAndroidPolicyRequestByUUIDFuncInvoked)
		require.True(t, mockDS.ListHostMDMAndroidProfilesPendingOrFailedInstallWithVersionFuncInvoked)
		require.True(t, mockDS.BulkUpsertMDMAndroidHostProfilesFuncInvoked)
		require.True(t, mockDS.BulkDeleteMDMAndroidHostProfilesFuncInvoked)
		mockDS.ListHostMDMAndroidProfilesPendingOrFailedInstallWithVersionFuncInvoked = false
		mockDS.BulkDeleteMDMAndroidHostProfilesFuncInvoked = false
		mockDS.BulkUpsertMDMAndroidHostProfilesFuncInvoked = false
		mockDS.GetAndroidPolicyRequestByUUIDFuncInvoked = false
	})

	t.Run("profile failed due to non-compliance but is reverified", func(t *testing.T) {
		policyVersion := new(1)

		policyRequestUUID := uuid.NewString()
		installPendingProfile1 := &fleet.MDMAndroidProfilePayload{
			ProfileUUID:             uuid.NewString(),
			ProfileName:             "a",
			HostUUID:                androidDevice.UUID,
			Status:                  &fleet.MDMDeliveryPending,
			OperationType:           fleet.MDMOperationTypeInstall,
			IncludedInPolicyVersion: policyVersion,
			PolicyRequestUUID:       &policyRequestUUID,
		}

		installPendingProfile2 := &fleet.MDMAndroidProfilePayload{
			ProfileUUID:             uuid.NewString(),
			ProfileName:             "b",
			HostUUID:                androidDevice.UUID,
			Status:                  &fleet.MDMDeliveryPending,
			OperationType:           fleet.MDMOperationTypeInstall,
			IncludedInPolicyVersion: policyVersion,
			PolicyRequestUUID:       &policyRequestUUID,
		}

		installPendingProfile3 := &fleet.MDMAndroidProfilePayload{
			ProfileUUID:             uuid.NewString(),
			ProfileName:             "c",
			HostUUID:                androidDevice.UUID,
			Status:                  &fleet.MDMDeliveryPending,
			OperationType:           fleet.MDMOperationTypeInstall,
			IncludedInPolicyVersion: policyVersion,
			PolicyRequestUUID:       &policyRequestUUID,
		}

		mockDS.GetAndroidPolicyRequestByUUIDFunc = func(ctx context.Context, id string) (*android.MDMAndroidPolicyRequest, error) {
			if id == policyRequestUUID {
				payload, err := json.Marshal(map[string]any{
					"policy": map[string]any{
						"DefaultPermissionPolicy": true,
						"passwordPolicies":        []map[string]any{},
						"cameraDisabled":          true,
					},
					"metadata": map[string]any{
						"settings_origin": map[string]string{
							"DefaultPermissionPolicy": installPendingProfile1.ProfileUUID,
							"passwordPolicies":        installPendingProfile2.ProfileUUID,
							"cameraDisabled":          installPendingProfile3.ProfileUUID,
						},
					},
				})
				require.NoError(t, err)
				return &android.MDMAndroidPolicyRequest{
					Payload: payload,
				}, nil
			}

			return nil, errors.New("something went wrong")
		}

		mockDS.ListHostMDMAndroidProfilesPendingOrFailedInstallWithVersionFunc = func(ctx context.Context, hostUUID string, version int64) ([]*fleet.MDMAndroidProfilePayload, error) {
			return []*fleet.MDMAndroidProfilePayload{
				installPendingProfile1,
				installPendingProfile2,
				installPendingProfile3,
			}, nil
		}
		mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFunc = func(ctx context.Context, hostUUID string, version int64) ([]*fleet.HostAndroidVPPSoftwareInstall, error) {
			return nil, nil
		}

		wantedReason1 := fleet.MDMDeliveryVerified
		wantedReason2 := fleet.MDMDeliveryFailed
		wantedReason3 := fleet.MDMDeliveryVerified

		mockDS.BulkUpsertMDMAndroidHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAndroidProfilePayload) error {
			require.Len(t, payload, 3)
			for _, profile := range payload {
				switch profile.ProfileUUID {
				case installPendingProfile1.ProfileUUID:
					require.Equal(t, wantedReason1, *profile.Status)
				case installPendingProfile2.ProfileUUID:
					require.Equal(t, wantedReason2, *profile.Status)
				case installPendingProfile3.ProfileUUID:
					require.Equal(t, wantedReason3, *profile.Status)
				}
			}
			return nil
		}
		mockDS.BulkDeleteMDMAndroidHostProfilesFunc = func(ctx context.Context, hostUUID string, policyVersionID int64) error {
			return nil
		}

		// the two pending profiles will be set to verified, and the non-compliant profile will be set to failed
		enrollmentMessage := createStatusReportMessage(t, androidDevice.UUID, "test", createAndroidDeviceId("test-policy"), policyVersion,
			[]*androidmanagement.NonComplianceDetail{{SettingName: "passwordPolicies", NonComplianceReason: "USER_ACTION"}})

		err := svc.ProcessPubSubPush(context.Background(), "value", &enrollmentMessage)
		require.NoError(t, err)

		require.True(t, mockDS.ListHostMDMAndroidProfilesPendingOrFailedInstallWithVersionFuncInvoked)
		require.True(t, mockDS.BulkUpsertMDMAndroidHostProfilesFuncInvoked)
		require.True(t, mockDS.BulkDeleteMDMAndroidHostProfilesFuncInvoked)
		mockDS.ListHostMDMAndroidProfilesPendingOrFailedInstallWithVersionFuncInvoked = false
		mockDS.BulkDeleteMDMAndroidHostProfilesFuncInvoked = false
		mockDS.BulkUpsertMDMAndroidHostProfilesFuncInvoked = false

		// the failed profile will now be verified because it is no longer in non compliance details
		enrollmentMessage = createStatusReportMessage(t, androidDevice.UUID, "test", createAndroidDeviceId("test-policy"), policyVersion,
			[]*androidmanagement.NonComplianceDetail{})
		wantedReason2 = fleet.MDMDeliveryVerified

		err = svc.ProcessPubSubPush(context.Background(), "value", &enrollmentMessage)
		require.NoError(t, err)

		require.True(t, mockDS.ListHostMDMAndroidProfilesPendingOrFailedInstallWithVersionFuncInvoked)
		require.True(t, mockDS.BulkUpsertMDMAndroidHostProfilesFuncInvoked)
		require.True(t, mockDS.BulkDeleteMDMAndroidHostProfilesFuncInvoked)
	})
}

func TestUpdateHostEmptyUUIDGetsPopulated(t *testing.T) {
	svc, mockDS := createAndroidService(t)
	enterpriseSpecificID := "SHOULD-BE-THIS-UUID"
	const deviceName = "test-empty-uuid-bug"

	// Mock AppConfig
	mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			MDM: fleet.MDM{
				AndroidEnabledAndConfigured: true,
			},
		}, nil
	}

	// Create existing host with blank UUID
	existingHostWithEmptyUUID := &fleet.AndroidHost{
		Host: &fleet.Host{
			ID:       200,
			UUID:     "",
			Hostname: "buggy-hostname",
			TeamID:   nil,
		},
		Device: &android.Device{
			HostID:               200,
			DeviceID:             "buggy-device",
			EnterpriseSpecificID: &enterpriseSpecificID,
		},
	}
	existingHostWithEmptyUUID.SetNodeKey(enterpriseSpecificID)

	// Mock AndroidHostLite returns host with blank UUID
	mockDS.AndroidHostLiteFunc = func(ctx context.Context, esID string) (*fleet.AndroidHost, error) {
		if esID == enterpriseSpecificID {
			// Return host with empty UUID
			return &fleet.AndroidHost{
				Host: &fleet.Host{
					ID:       existingHostWithEmptyUUID.Host.ID,
					UUID:     "",
					Hostname: existingHostWithEmptyUUID.Host.Hostname,
					TeamID:   existingHostWithEmptyUUID.Host.TeamID,
				},
				Device: existingHostWithEmptyUUID.Device,
			}, nil
		}
		return nil, common_mysql.NotFound("android host")
	}

	// Capture what gets sent to UpdateAndroidHost (and thus to the API)
	var capturedHost *fleet.AndroidHost
	mockDS.UpdateAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, fromEnroll, companyOwned bool) error {
		capturedHost = host
		return nil
	}

	// status report with valid EnterpriseSpecificId
	device := androidmanagement.Device{
		Name: createAndroidDeviceId(deviceName),
		HardwareInfo: &androidmanagement.HardwareInfo{
			EnterpriseSpecificId: enterpriseSpecificID,
			Brand:                "TestBrand",
			Model:                "TestModel",
		},
		SoftwareInfo: &androidmanagement.SoftwareInfo{
			AndroidVersion: "14",
		},
		MemoryInfo: &androidmanagement.MemoryInfo{
			TotalRam: int64(8 * 1024 * 1024 * 1024),
		},
		LastStatusReportTime: "2024-01-01T12:00:00Z",
	}

	deviceBytes, err := json.Marshal(device)
	require.NoError(t, err)
	encodedData := base64.StdEncoding.EncodeToString(deviceBytes)
	message := &android.PubSubMessage{
		Attributes: map[string]string{
			"notificationType": string(android.PubSubStatusReport),
		},
		Data: encodedData,
	}

	// Process the status report
	err = svc.ProcessPubSubPush(context.Background(), "value", message)
	require.NoError(t, err)

	require.True(t, mockDS.UpdateAndroidHostFuncInvoked)
	require.NotNil(t, capturedHost)

	require.Equal(t, enterpriseSpecificID, capturedHost.Host.UUID,
		"Host UUID is properly populated from device.EnterpriseSpecificId")
}

// TestStatusReportPopulatesOperatingSystem is a regression test for
// https://github.com/fleetdm/fleet/issues/45711. Status reports for existing
// Android hosts must populate the operating_systems table so the hosts can be
// filtered via GET /api/v1/fleet/hosts?os_name=Android&os_version=<v>.
func TestStatusReportPopulatesOperatingSystem(t *testing.T) {
	svc, mockDS := createAndroidService(t)
	const enterpriseSpecificID = "ESI-OS-TEST"

	mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			MDM: fleet.MDM{AndroidEnabledAndConfigured: true},
		}, nil
	}
	expectedHostID := uint(321)
	mockDS.AndroidHostLiteFunc = func(ctx context.Context, esID string) (*fleet.AndroidHost, error) {
		return &fleet.AndroidHost{
			Host: &fleet.Host{
				ID:   expectedHostID,
				UUID: enterpriseSpecificID,
			},
			Device: &android.Device{
				HostID:               expectedHostID,
				DeviceID:             "device",
				EnterpriseSpecificID: new(enterpriseSpecificID),
			},
		}, nil
	}
	mockDS.UpdateAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, fromEnroll, companyOwned bool) error {
		return nil
	}

	var capturedHostID uint
	var capturedOS fleet.OperatingSystem
	mockDS.UpdateHostOperatingSystemFunc = func(ctx context.Context, hostID uint, hostOS fleet.OperatingSystem) error {
		capturedHostID = hostID
		capturedOS = hostOS
		return nil
	}

	device := androidmanagement.Device{
		Name: createAndroidDeviceId("test-android-status-os"),
		HardwareInfo: &androidmanagement.HardwareInfo{
			EnterpriseSpecificId: enterpriseSpecificID,
			Brand:                "Google",
			Model:                "Pixel 8a",
		},
		SoftwareInfo: &androidmanagement.SoftwareInfo{
			AndroidVersion: "16",
		},
		MemoryInfo: &androidmanagement.MemoryInfo{
			TotalRam: int64(8 * 1024 * 1024 * 1024),
		},
		LastStatusReportTime: "2024-01-01T12:00:00Z",
	}
	deviceBytes, err := json.Marshal(device)
	require.NoError(t, err)
	message := &android.PubSubMessage{
		Attributes: map[string]string{"notificationType": string(android.PubSubStatusReport)},
		Data:       base64.StdEncoding.EncodeToString(deviceBytes),
	}

	err = svc.ProcessPubSubPush(t.Context(), "value", message)
	require.NoError(t, err)

	require.True(t, mockDS.UpdateHostOperatingSystemFuncInvoked)
	require.Equal(t, expectedHostID, capturedHostID)
	require.Equal(t, "Android", capturedOS.Name)
	require.Equal(t, "16", capturedOS.Version)
	require.Equal(t, "android", capturedOS.Platform)
}

func TestHostPayloadUUIDForFrontend(t *testing.T) {
	svc, mockDS := createAndroidService(t)
	enterpriseSpecificID := "ANDROID-DEVICE-UUID-123"
	const deviceName = "test-frontend-payload"

	// Mock AppConfig
	mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			MDM: fleet.MDM{
				AndroidEnabledAndConfigured: true,
			},
		}, nil
	}

	testCases := []struct {
		name         string
		existingUUID string
		expectedUUID string
		description  string
	}{
		{
			name:         "empty_uuid_gets_populated",
			existingUUID: "",
			expectedUUID: enterpriseSpecificID,
			description:  "Empty UUID is properly populated from EnterpriseSpecificId",
		},
		{
			name:         "existing_uuid_gets_updated",
			existingUUID: "OLD-UUID-456",
			expectedUUID: enterpriseSpecificID,
			description:  "Existing UUID is updated to match current EnterpriseSpecificId",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock AndroidHostLite returns host with specified UUID
			mockDS.AndroidHostLiteFunc = func(ctx context.Context, esID string) (*fleet.AndroidHost, error) {
				if esID == enterpriseSpecificID {
					return &fleet.AndroidHost{
						Host: &fleet.Host{
							ID:       100,
							UUID:     tc.existingUUID,
							Hostname: "test-host",
							TeamID:   nil,
						},
						Device: &android.Device{
							HostID:               100,
							DeviceID:             "test-device",
							EnterpriseSpecificID: &enterpriseSpecificID,
						},
					}, nil
				}
				return nil, common_mysql.NotFound("android host")
			}

			// Capture the host payload that would be sent to frontend
			var hostPayload *fleet.AndroidHost
			mockDS.UpdateAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, fromEnroll, companyOwned bool) error {
				hostPayload = host
				return nil
			}

			// Create status report from Google
			device := androidmanagement.Device{
				Name: createAndroidDeviceId(deviceName),
				HardwareInfo: &androidmanagement.HardwareInfo{
					EnterpriseSpecificId: enterpriseSpecificID,
					Brand:                "Google",
					Model:                "Pixel",
				},
				SoftwareInfo: &androidmanagement.SoftwareInfo{
					AndroidVersion: "14",
				},
				MemoryInfo: &androidmanagement.MemoryInfo{
					TotalRam: int64(8 * 1024 * 1024 * 1024),
				},
				LastStatusReportTime: "2024-01-01T12:00:00Z",
			}

			deviceBytes, err := json.Marshal(device)
			require.NoError(t, err)
			encodedData := base64.StdEncoding.EncodeToString(deviceBytes)
			message := &android.PubSubMessage{
				Attributes: map[string]string{
					"notificationType": string(android.PubSubStatusReport),
				},
				Data: encodedData,
			}

			// Process the status report
			err = svc.ProcessPubSubPush(context.Background(), "value", message)
			require.NoError(t, err)

			require.NotNil(t, hostPayload)
			require.Equal(t, tc.expectedUUID, hostPayload.Host.UUID, tc.description)
		})
	}
}

func TestUpdateHost(t *testing.T) {
	svc, mockDS := createAndroidService(t)

	enterpriseSpecificID := "TEST-UUID-12345"
	const deviceName = "test-update-host"

	// Mock AppConfig
	mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			MDM: fleet.MDM{
				AndroidEnabledAndConfigured: true,
			},
		}, nil
	}

	// Create an existing Android host with empty UUID
	existingHost := &fleet.AndroidHost{
		Host: &fleet.Host{
			ID:       1,
			UUID:     "",
			Hostname: "old-hostname",
			TeamID:   nil,
		},
		Device: &android.Device{
			HostID:               1,
			DeviceID:             "old-device-id",
			EnterpriseSpecificID: &enterpriseSpecificID,
		},
	}
	existingHost.SetNodeKey(enterpriseSpecificID)

	// Mock AndroidHostLite returns the existing host
	mockDS.AndroidHostLiteFunc = func(ctx context.Context, esID string) (*fleet.AndroidHost, error) {
		if esID == enterpriseSpecificID {
			return existingHost, nil
		}
		return nil, common_mysql.NotFound("android host")
	}

	// verify UUID is set correctly
	var capturedHost *fleet.AndroidHost
	mockDS.UpdateAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, fromEnroll, companyOwned bool) error {
		// Validate that the update always updates the label updated at value with a recent one
		require.Greater(t, host.LabelUpdatedAt, time.Now().Add(-5*time.Second))
		capturedHost = host
		return nil
	}

	t.Run("UUID gets populated from EnterpriseSpecificId", func(t *testing.T) {
		// Create status report message that triggers updateHost
		device := androidmanagement.Device{
			Name: createAndroidDeviceId(deviceName),
			HardwareInfo: &androidmanagement.HardwareInfo{
				EnterpriseSpecificId: enterpriseSpecificID,
				Brand:                "UpdatedBrand",
				Model:                "UpdatedModel",
				SerialNumber:         "updated-serial",
				Hardware:             "updated-hardware",
			},
			SoftwareInfo: &androidmanagement.SoftwareInfo{
				AndroidBuildNumber: "updated-build",
				AndroidVersion:     "15",
			},
			MemoryInfo: &androidmanagement.MemoryInfo{
				TotalRam:             int64(16 * 1024 * 1024 * 1024),  // 16GB RAM
				TotalInternalStorage: int64(128 * 1024 * 1024 * 1024), // 128GB storage
			},
			LastStatusReportTime: "2024-01-01T12:00:00Z",
		}

		// Create message
		deviceBytes, err := json.Marshal(device)
		require.NoError(t, err)
		encodedData := base64.StdEncoding.EncodeToString(deviceBytes)
		message := &android.PubSubMessage{
			Attributes: map[string]string{
				"notificationType": string(android.PubSubStatusReport),
			},
			Data: encodedData,
		}

		// Process the message
		err = svc.ProcessPubSubPush(context.Background(), "value", message)
		require.NoError(t, err)

		// Verify UpdateAndroidHost was called
		require.True(t, mockDS.UpdateAndroidHostFuncInvoked)
		require.NotNil(t, capturedHost)

		// UUID is properly set from device data
		require.Equal(t, enterpriseSpecificID, capturedHost.Host.UUID, "UUID is properly set from EnterpriseSpecificId")

		// Other fields were updated (no IdP account, so falls back to hardware model)
		require.Equal(t, "Updatedbrand UpdatedModel", capturedHost.Host.ComputerName)
		require.Equal(t, "Updatedbrand UpdatedModel", capturedHost.Host.Hostname)
		require.Equal(t, "Updatedbrand UpdatedModel", capturedHost.Host.HardwareModel)
		require.Equal(t, "Android 15", capturedHost.Host.OSVersion)
	})

	t.Run("UUID is set from EnterpriseSpecificId", func(t *testing.T) {
		mockDS.UpdateAndroidHostFuncInvoked = false
		capturedHost = nil

		// Create a status report message
		device := androidmanagement.Device{
			Name: createAndroidDeviceId(deviceName),
			HardwareInfo: &androidmanagement.HardwareInfo{
				EnterpriseSpecificId: enterpriseSpecificID,
				Brand:                "UpdatedBrand",
				Model:                "UpdatedModel",
			},
			SoftwareInfo: &androidmanagement.SoftwareInfo{
				AndroidVersion: "15",
			},
			MemoryInfo: &androidmanagement.MemoryInfo{
				TotalRam: int64(16 * 1024 * 1024 * 1024),
			},
			LastStatusReportTime: "2024-01-01T12:00:00Z",
		}

		deviceBytes, err := json.Marshal(device)
		require.NoError(t, err)
		encodedData := base64.StdEncoding.EncodeToString(deviceBytes)
		message := &android.PubSubMessage{
			Attributes: map[string]string{
				"notificationType": string(android.PubSubStatusReport),
			},
			Data: encodedData,
		}

		err = svc.ProcessPubSubPush(context.Background(), "value", message)
		require.NoError(t, err)

		// WITH THE FIX: UUID should be set to EnterpriseSpecificId
		require.Equal(t, enterpriseSpecificID, capturedHost.Host.UUID, "UUID should be set from device data")
	})

	t.Run("Company-owned device update should update the host", func(t *testing.T) {
		// Reset the mock invocation flag
		mockDS.UpdateAndroidHostFuncInvoked = false
		capturedHost = nil

		// Create a device with empty EnterpriseSpecificId (edge case)
		device := androidmanagement.Device{
			Name: createAndroidDeviceId(deviceName + "-empty"),
			HardwareInfo: &androidmanagement.HardwareInfo{
				EnterpriseSpecificId: "", // Empty UUID
				Brand:                "TestBrand",
				Model:                "TestModel",
				SerialNumber:         "test-serial",
			},
			SoftwareInfo: &androidmanagement.SoftwareInfo{
				AndroidBuildNumber: "test-build",
				AndroidVersion:     "14",
			},
			MemoryInfo: &androidmanagement.MemoryInfo{
				TotalRam: int64(8 * 1024 * 1024 * 1024),
			},
			LastStatusReportTime: "2024-01-01T12:00:00Z",
		}

		// Mock to return host with empty enterprise ID
		mockDS.AndroidHostLiteFunc = func(ctx context.Context, esID string) (*fleet.AndroidHost, error) {
			return &fleet.AndroidHost{
				Host: &fleet.Host{
					ID:   2,
					UUID: testBrandTestSerialHashed,
				},
				Device: &android.Device{
					HostID:               2,
					DeviceID:             "device-2",
					EnterpriseSpecificID: &testBrandTestSerialHashed,
				},
			}, nil
		}

		// Create and process message
		deviceBytes, err := json.Marshal(device)
		require.NoError(t, err)
		encodedData := base64.StdEncoding.EncodeToString(deviceBytes)
		message := &android.PubSubMessage{
			Attributes: map[string]string{
				"notificationType": string(android.PubSubStatusReport),
			},
			Data: encodedData,
		}

		err = svc.ProcessPubSubPush(context.Background(), "value", message)
		require.NoError(t, err)

		// Verify UpdateAndroidHost was called
		require.True(t, mockDS.UpdateAndroidHostFuncInvoked)
		require.NotNil(t, capturedHost)

		require.Equal(t, testBrandTestSerialHashed, capturedHost.Host.UUID)
	})
}

func TestAndroidHostDisplayNameWithIdP(t *testing.T) {
	t.Run("updateHost uses IdP first name in computer name", func(t *testing.T) {
		svc, mockDS := createAndroidService(t)

		const enterpriseSpecificID = "IDP-TEST-UUID"

		mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				MDM: fleet.MDM{AndroidEnabledAndConfigured: true},
			}, nil
		}

		mockDS.AndroidHostLiteFunc = func(ctx context.Context, esID string) (*fleet.AndroidHost, error) {
			return &fleet.AndroidHost{
				Host: &fleet.Host{
					ID:   1,
					UUID: enterpriseSpecificID,
				},
				Device: &android.Device{
					HostID:               1,
					DeviceID:             "device-1",
					EnterpriseSpecificID: new(enterpriseSpecificID),
				},
			}, nil
		}

		// Mock IdP account lookup to return a user with a full name
		mockDS.GetMDMIdPAccountByHostUUIDFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMIdPAccount, error) {
			return &fleet.MDMIdPAccount{
				UUID:     "idp-acct-uuid",
				Fullname: "John Smith",
				Email:    "ksykulev@test.com",
			}, nil
		}

		var capturedHost *fleet.AndroidHost
		mockDS.UpdateAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, fromEnroll, companyOwned bool) error {
			capturedHost = host
			return nil
		}

		device := androidmanagement.Device{
			Name: createAndroidDeviceId("test-idp-display"),
			HardwareInfo: &androidmanagement.HardwareInfo{
				EnterpriseSpecificId: enterpriseSpecificID,
				Brand:                "samsung",
				Model:                "SM-A176U1",
			},
			SoftwareInfo:         &androidmanagement.SoftwareInfo{AndroidVersion: "15"},
			MemoryInfo:           &androidmanagement.MemoryInfo{TotalRam: int64(8 * 1024 * 1024 * 1024)},
			LastStatusReportTime: "2024-01-01T12:00:00Z",
		}

		deviceBytes, err := json.Marshal(device)
		require.NoError(t, err)
		message := &android.PubSubMessage{
			Attributes: map[string]string{"notificationType": string(android.PubSubStatusReport)},
			Data:       base64.StdEncoding.EncodeToString(deviceBytes),
		}

		err = svc.ProcessPubSubPush(t.Context(), "value", message)
		require.NoError(t, err)
		require.NotNil(t, capturedHost)

		require.Equal(t, "John Smith's Samsung SM-A176U1", capturedHost.Host.ComputerName)
		require.Equal(t, "John Smith's Samsung SM-A176U1", capturedHost.Host.Hostname)
		require.Equal(t, "Samsung SM-A176U1", capturedHost.Host.HardwareModel)
	})

	t.Run("updateHost falls back to hardware model when no IdP account", func(t *testing.T) {
		svc, mockDS := createAndroidService(t)

		const enterpriseSpecificID = "NO-IDP-UUID"

		mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				MDM: fleet.MDM{AndroidEnabledAndConfigured: true},
			}, nil
		}

		mockDS.AndroidHostLiteFunc = func(ctx context.Context, esID string) (*fleet.AndroidHost, error) {
			return &fleet.AndroidHost{
				Host: &fleet.Host{
					ID:   2,
					UUID: enterpriseSpecificID,
				},
				Device: &android.Device{
					HostID:               2,
					DeviceID:             "device-2",
					EnterpriseSpecificID: new(enterpriseSpecificID),
				},
			}, nil
		}

		var capturedHost *fleet.AndroidHost
		mockDS.UpdateAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, fromEnroll, companyOwned bool) error {
			capturedHost = host
			return nil
		}

		device := androidmanagement.Device{
			Name: createAndroidDeviceId("test-no-idp"),
			HardwareInfo: &androidmanagement.HardwareInfo{
				EnterpriseSpecificId: enterpriseSpecificID,
				Brand:                "google",
				Model:                "Pixel 7",
			},
			SoftwareInfo:         &androidmanagement.SoftwareInfo{AndroidVersion: "14"},
			MemoryInfo:           &androidmanagement.MemoryInfo{TotalRam: int64(8 * 1024 * 1024 * 1024)},
			LastStatusReportTime: "2024-01-01T12:00:00Z",
		}

		deviceBytes, err := json.Marshal(device)
		require.NoError(t, err)
		message := &android.PubSubMessage{
			Attributes: map[string]string{"notificationType": string(android.PubSubStatusReport)},
			Data:       base64.StdEncoding.EncodeToString(deviceBytes),
		}

		err = svc.ProcessPubSubPush(t.Context(), "value", message)
		require.NoError(t, err)
		require.NotNil(t, capturedHost)

		require.Equal(t, "Google Pixel 7", capturedHost.Host.ComputerName)
		require.Equal(t, "Google Pixel 7", capturedHost.Host.Hostname)
		require.Equal(t, "Google Pixel 7", capturedHost.Host.HardwareModel)
	})

	t.Run("addNewHost uses IdP name when IdP UUID is present", func(t *testing.T) {
		svc, mockDS := createAndroidService(t)

		mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				MDM: fleet.MDM{AndroidEnabledAndConfigured: true},
			}, nil
		}

		mockDS.AndroidHostLiteFunc = func(ctx context.Context, esID string) (*fleet.AndroidHost, error) {
			return nil, common_mysql.NotFound("android host")
		}

		mockDS.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
			return &fleet.EnrollSecret{Secret: secret}, nil
		}

		mockDS.GetMDMIdPAccountByUUIDFunc = func(ctx context.Context, uuid string) (*fleet.MDMIdPAccount, error) {
			return &fleet.MDMIdPAccount{
				UUID:     uuid,
				Fullname: "Jane Doe",
				Email:    "jane@test.com",
			}, nil
		}

		var capturedHost *fleet.AndroidHost
		mockDS.NewAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, companyOwned bool) (*fleet.AndroidHost, error) {
			capturedHost = host
			return &fleet.AndroidHost{Host: &fleet.Host{}}, nil
		}

		mockDS.AssociateHostMDMIdPAccountFunc = func(ctx context.Context, hostUUID, accountUUID string) error {
			return nil
		}
		mockDS.MaybeAssociateHostWithScimUserFunc = func(ctx context.Context, hostID uint) error {
			return nil
		}

		enrollmentToken := enrollmentTokenRequest{
			EnrollSecret: "global",
			IdpUUID:      "jane-idp-uuid",
		}
		enrollTokenData, err := json.Marshal(enrollmentToken)
		require.NoError(t, err)

		enrollmentMessage := createEnrollmentMessage(t, androidmanagement.Device{
			Name:                createAndroidDeviceId("test-idp-enroll"),
			EnrollmentTokenData: string(enrollTokenData),
		})

		err = svc.ProcessPubSubPush(t.Context(), "value", enrollmentMessage)
		require.NoError(t, err)
		require.NotNil(t, capturedHost)

		require.Equal(t, "Jane Doe's Testbrand TestModel", capturedHost.Host.ComputerName)
		require.Equal(t, "Jane Doe's Testbrand TestModel", capturedHost.Host.Hostname)
		require.Equal(t, "Testbrand TestModel", capturedHost.Host.HardwareModel)
	})

	t.Run("addNewHost falls back when IdP fullname is empty", func(t *testing.T) {
		svc, mockDS := createAndroidService(t)

		mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				MDM: fleet.MDM{AndroidEnabledAndConfigured: true},
			}, nil
		}

		mockDS.AndroidHostLiteFunc = func(ctx context.Context, esID string) (*fleet.AndroidHost, error) {
			return nil, common_mysql.NotFound("android host")
		}

		mockDS.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
			return &fleet.EnrollSecret{Secret: secret}, nil
		}

		mockDS.GetMDMIdPAccountByUUIDFunc = func(ctx context.Context, uuid string) (*fleet.MDMIdPAccount, error) {
			return &fleet.MDMIdPAccount{
				UUID:     uuid,
				Fullname: "",
				Email:    "nofullname@test.com",
			}, nil
		}

		var capturedHost *fleet.AndroidHost
		mockDS.NewAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, companyOwned bool) (*fleet.AndroidHost, error) {
			capturedHost = host
			return &fleet.AndroidHost{Host: &fleet.Host{}}, nil
		}

		mockDS.AssociateHostMDMIdPAccountFunc = func(ctx context.Context, hostUUID, accountUUID string) error {
			return nil
		}
		mockDS.MaybeAssociateHostWithScimUserFunc = func(ctx context.Context, hostID uint) error {
			return nil
		}

		enrollmentToken := enrollmentTokenRequest{
			EnrollSecret: "global",
			IdpUUID:      "empty-name-uuid",
		}
		enrollTokenData, err := json.Marshal(enrollmentToken)
		require.NoError(t, err)

		enrollmentMessage := createEnrollmentMessage(t, androidmanagement.Device{
			Name:                createAndroidDeviceId("test-empty-name-enroll"),
			EnrollmentTokenData: string(enrollTokenData),
		})

		err = svc.ProcessPubSubPush(t.Context(), "value", enrollmentMessage)
		require.NoError(t, err)
		require.NotNil(t, capturedHost)

		// Falls back to Brand + Model when fullname is empty
		require.Equal(t, "Testbrand TestModel", capturedHost.Host.ComputerName)
		require.Equal(t, "Testbrand TestModel", capturedHost.Host.Hostname)
	})
}

func TestAndroidStorageExtraction(t *testing.T) {
	svc, mockDS := createAndroidService(t)

	mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			MDM: fleet.MDM{AndroidEnabledAndConfigured: true},
		}, nil
	}

	mockDS.AndroidHostLiteFunc = func(ctx context.Context, enterpriseSpecificID string) (*fleet.AndroidHost, error) {
		return nil, common_mysql.NotFound("android host lite mock")
	}

	mockDS.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
		return &fleet.EnrollSecret{Secret: "global"}, nil
	}

	var createdHost *fleet.AndroidHost
	mockDS.NewAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, companyOwned bool) (*fleet.AndroidHost, error) {
		createdHost = host
		return host, nil
	}

	t.Run("extracts storage data from AMAPI device", func(t *testing.T) {
		createdHost = nil // Reset

		enrollmentMessage := createEnrollmentMessage(t, androidmanagement.Device{
			Name:                createAndroidDeviceId("storage-test"),
			EnrollmentTokenData: `{"enroll_secret": "global"}`,
		})

		err := svc.ProcessPubSubPush(context.Background(), "value", enrollmentMessage)
		require.NoError(t, err)

		require.NotNil(t, createdHost)
		require.NotNil(t, createdHost.Host)

		// Total: 64GB (internal) + 64GB (external) = 128GB
		// Available: 10GB (internal free) + 25GB (external free) = 35GB
		// Percentage: 35/128 * 100 = 27.34%
		require.Equal(t, 128.0, createdHost.Host.GigsTotalDiskSpace, "should calculate total storage (64GB internal + 64GB external)")
		require.Equal(t, 35.0, createdHost.Host.GigsDiskSpaceAvailable, "should calculate total available storage (10GB + 25GB)")
		require.InDelta(t, 27.34, createdHost.Host.PercentDiskSpaceAvailable, 0.1, "should calculate percentage (35/128*100=27.34%)")
	})

	t.Run("storage not supported when missing MEASURED events", func(t *testing.T) {
		createdHost = nil // Reset

		enrollmentMessage := createEnrollmentMessageWithoutMeasuredEvents(t, androidmanagement.Device{
			Name:                createAndroidDeviceId("work-profile-test"),
			EnrollmentTokenData: `{"enroll_secret": "global"}`,
		})

		err := svc.ProcessPubSubPush(context.Background(), "value", enrollmentMessage)
		require.NoError(t, err)

		require.NotNil(t, createdHost)
		require.NotNil(t, createdHost.Host)

		// Total storage is calculated from DETECTED events
		require.Equal(t, 128.0, createdHost.Host.GigsTotalDiskSpace, "should calculate total storage from DETECTED events")

		// Available storage and percentage should be -1 (not supported) when only DETECTED events are present
		require.Equal(t, float64(-1), createdHost.Host.GigsDiskSpaceAvailable, "should set available storage to -1 when MEASURED events are missing")
		require.Equal(t, float64(-1), createdHost.Host.PercentDiskSpaceAvailable, "should set percent available to -1 when MEASURED events are missing")
	})

	t.Run("uses only latest EXTERNAL_STORAGE_DETECTED event", func(t *testing.T) {
		createdHost = nil // Reset

		enrollmentMessage := createEnrollmentMessageWithMultipleExternalDetectedEvents(t, androidmanagement.Device{
			Name:                createAndroidDeviceId("multiple-external-test"),
			EnrollmentTokenData: `{"enroll_secret": "global"}`,
		})

		err := svc.ProcessPubSubPush(context.Background(), "value", enrollmentMessage)
		require.NoError(t, err)

		require.NotNil(t, createdHost)
		require.NotNil(t, createdHost.Host)

		// Should use only the latest EXTERNAL_STORAGE_DETECTED event (120GB)
		// Total: 1GB (internal) + 120GB (latest external) = 121GB
		require.Equal(t, 121.0, createdHost.Host.GigsTotalDiskSpace, "should use only latest EXTERNAL_STORAGE_DETECTED event")

		// Available: 96GB (from EXTERNAL_STORAGE_MEASURED)
		require.InDelta(t, 96.0, createdHost.Host.GigsDiskSpaceAvailable, 0.1, "should calculate available storage")

		// Percentage: 96/121 * 100 = 79.34%
		require.InDelta(t, 79.34, createdHost.Host.PercentDiskSpaceAvailable, 0.1, "should calculate percentage correctly")
	})
}

func createEnrollmentMessage(t *testing.T, deviceInfo androidmanagement.Device) *android.PubSubMessage {
	deviceInfo.HardwareInfo = &androidmanagement.HardwareInfo{
		Brand:    "TestBrand",
		Model:    "TestModel",
		Hardware: "test-hardware",
	}
	// default to personally owned for tests if not specified
	if deviceInfo.Ownership == "" {
		deviceInfo.Ownership = DeviceOwnershipPersonallyOwned
	}
	if deviceInfo.Ownership == DeviceOwnershipCompanyOwned {
		deviceInfo.HardwareInfo.SerialNumber = "test-serial"
	}
	if deviceInfo.Ownership == DeviceOwnershipPersonallyOwned {
		deviceInfo.HardwareInfo.EnterpriseSpecificId = strings.ToUpper(uuid.New().String())
		deviceInfo.HardwareInfo.SerialNumber = deviceInfo.HardwareInfo.EnterpriseSpecificId
	}
	deviceInfo.SoftwareInfo = &androidmanagement.SoftwareInfo{
		AndroidBuildNumber: "test-build",
		AndroidVersion:     "1",
	}
	deviceInfo.MemoryInfo = &androidmanagement.MemoryInfo{
		TotalRam:             int64(8 * 1024 * 1024 * 1024),  // 8GB RAM in bytes
		TotalInternalStorage: int64(64 * 1024 * 1024 * 1024), // 64GB system partition
	}

	deviceInfo.MemoryEvents = []*androidmanagement.MemoryEvent{
		{
			EventType:  "EXTERNAL_STORAGE_DETECTED",
			ByteCount:  int64(64 * 1024 * 1024 * 1024), // 64GB external/built-in storage total capacity
			CreateTime: "2024-01-15T09:00:00Z",
		},
		{
			EventType:  "INTERNAL_STORAGE_MEASURED",
			ByteCount:  int64(10 * 1024 * 1024 * 1024), // 10GB free in system partition
			CreateTime: "2024-01-15T10:00:00Z",
		},
		{
			EventType:  "EXTERNAL_STORAGE_MEASURED",
			ByteCount:  int64(25 * 1024 * 1024 * 1024), // 25GB free in external/built-in storage
			CreateTime: "2024-01-15T10:00:00Z",
		},
	}

	data, err := json.Marshal(deviceInfo)
	require.NoError(t, err)

	encodedData := base64.StdEncoding.EncodeToString(data)

	return &android.PubSubMessage{
		Attributes: map[string]string{
			"notificationType": string(android.PubSubEnrollment),
		},
		Data: encodedData,
	}
}

func createEnrollmentMessageWithoutMeasuredEvents(t *testing.T, deviceInfo androidmanagement.Device) *android.PubSubMessage {
	deviceInfo.HardwareInfo = &androidmanagement.HardwareInfo{
		EnterpriseSpecificId: strings.ToUpper(uuid.New().String()),
		Brand:                "TestBrand",
		Model:                "TestModel",
		SerialNumber:         "test-serial",
		Hardware:             "test-hardware",
	}
	deviceInfo.SoftwareInfo = &androidmanagement.SoftwareInfo{
		AndroidBuildNumber: "test-build",
		AndroidVersion:     "1",
	}
	deviceInfo.MemoryInfo = &androidmanagement.MemoryInfo{
		TotalRam:             int64(8 * 1024 * 1024 * 1024),  // 8GB RAM in bytes
		TotalInternalStorage: int64(64 * 1024 * 1024 * 1024), // 64GB system partition
	}

	// Only DETECTED events, not MEASURED events
	deviceInfo.MemoryEvents = []*androidmanagement.MemoryEvent{
		{
			EventType:  "EXTERNAL_STORAGE_DETECTED",
			ByteCount:  int64(64 * 1024 * 1024 * 1024), // 64GB external/built-in storage total capacity
			CreateTime: "2024-01-15T09:00:00Z",
		},
		// No INTERNAL_STORAGE_MEASURED or EXTERNAL_STORAGE_MEASURED events
	}

	data, err := json.Marshal(deviceInfo)
	require.NoError(t, err)

	encodedData := base64.StdEncoding.EncodeToString(data)

	return &android.PubSubMessage{
		Attributes: map[string]string{
			"notificationType": string(android.PubSubEnrollment),
		},
		Data: encodedData,
	}
}

func createEnrollmentMessageWithMultipleExternalDetectedEvents(t *testing.T, deviceInfo androidmanagement.Device) *android.PubSubMessage {
	deviceInfo.HardwareInfo = &androidmanagement.HardwareInfo{
		EnterpriseSpecificId: strings.ToUpper(uuid.New().String()),
		Brand:                "Google",
		Model:                "Pixel 8a",
		SerialNumber:         "test-serial",
		Hardware:             "test-hardware",
	}
	deviceInfo.SoftwareInfo = &androidmanagement.SoftwareInfo{
		AndroidBuildNumber: "test-build",
		AndroidVersion:     "16",
	}
	deviceInfo.MemoryInfo = &androidmanagement.MemoryInfo{
		TotalRam:             int64(8 * 1024 * 1024 * 1024), // 8GB RAM in bytes
		TotalInternalStorage: int64(1 * 1024 * 1024 * 1024), // 1GB work profile partition (simulates PROFILE_OWNER)
	}

	// Simulate with multiple work profile EXTERNAL_STORAGE_DETECTED events
	deviceInfo.MemoryEvents = []*androidmanagement.MemoryEvent{
		{
			EventType:  "INTERNAL_STORAGE_MEASURED",
			CreateTime: "2024-01-15T09:00:00Z",
			// No byteCount for work profiles
		},
		{
			EventType:  "EXTERNAL_STORAGE_MEASURED",
			ByteCount:  int64(96 * 1024 * 1024 * 1024), // 96GB available
			CreateTime: "2024-01-15T09:00:01Z",
		},
		// Multiple EXTERNAL_STORAGE_DETECTED events (simulating the bug scenario)
		{
			EventType:  "EXTERNAL_STORAGE_DETECTED",
			ByteCount:  int64(110 * 1024 * 1024 * 1024), // 110GB (older event)
			CreateTime: "2024-01-15T09:00:02Z",
		},
		{
			EventType:  "EXTERNAL_STORAGE_DETECTED",
			ByteCount:  int64(110 * 1024 * 1024 * 1024), // 110GB
			CreateTime: "2024-01-15T09:05:00Z",
		},
		{
			EventType:  "EXTERNAL_STORAGE_DETECTED",
			ByteCount:  int64(110 * 1024 * 1024 * 1024), // 110GB
			CreateTime: "2024-01-15T09:10:00Z",
		},
		{
			EventType:  "EXTERNAL_STORAGE_DETECTED",
			ByteCount:  int64(120 * 1024 * 1024 * 1024), // 120GB (latest, different value)
			CreateTime: "2024-01-15T09:15:00Z",
		},
		{
			EventType:  "EXTERNAL_STORAGE_DETECTED",
			ByteCount:  int64(110 * 1024 * 1024 * 1024), // 110GB (older timestamp than above)
			CreateTime: "2024-01-15T09:14:00Z",
		},
	}

	data, err := json.Marshal(deviceInfo)
	require.NoError(t, err)

	encodedData := base64.StdEncoding.EncodeToString(data)

	return &android.PubSubMessage{
		Attributes: map[string]string{
			"notificationType": string(android.PubSubEnrollment),
		},
		Data: encodedData,
	}
}

func createStatusReportMessage(t *testing.T, deviceId, name, policyName string, policyVersion *int, nonComplianceDetails []*androidmanagement.NonComplianceDetail) android.PubSubMessage {
	return createStatusAppReportMessage(t, deviceId, name, policyName, policyVersion, nil, nonComplianceDetails)
}

func createStatusAppReportMessage(t *testing.T, deviceId, name, policyName string, policyVersion *int, appReports []*androidmanagement.ApplicationReport, nonComplianceDetails []*androidmanagement.NonComplianceDetail) android.PubSubMessage {
	device := androidmanagement.Device{
		Name:                 createAndroidDeviceId(name),
		ApplicationReports:   appReports,
		NonComplianceDetails: nonComplianceDetails,
		HardwareInfo: &androidmanagement.HardwareInfo{
			EnterpriseSpecificId: deviceId,
			Brand:                "TestBrand",
			Model:                "TestModel",
			SerialNumber:         "test-serial",
			Hardware:             "test-hardware",
		},
		SoftwareInfo: &androidmanagement.SoftwareInfo{
			AndroidBuildNumber: "test-build",
			AndroidVersion:     "1",
		},
		MemoryInfo: &androidmanagement.MemoryInfo{
			TotalRam:             int64(8 * 1024 * 1024 * 1024),  // 8GB RAM in bytes
			TotalInternalStorage: int64(64 * 1024 * 1024 * 1024), // 64GB system partition
		},
		AppliedPolicyName:    policyName,
		AppliedPolicyVersion: int64(*policyVersion),
		LastPolicySyncTime:   "2001-01-01T00:00:00Z",
	}

	data, err := json.Marshal(device)
	require.NoError(t, err)

	encodedData := base64.StdEncoding.EncodeToString(data)

	return android.PubSubMessage{
		Attributes: map[string]string{
			"notificationType": string(android.PubSubStatusReport),
		},
		Data: encodedData,
	}
}

func createAndroidDeviceId(name string) string {
	return "enterprises/mock-enterprise-id/devices/" + name
}

func TestBuildNonComplianceErrorMessage(t *testing.T) {
	testCases := []struct {
		name                 string
		nonCompliance        []*androidmanagement.NonComplianceDetail
		expectedErrorMessage string
	}{
		{
			name:                 "nil non-compliance detail",
			nonCompliance:        nil,
			expectedErrorMessage: "Settings couldn't apply to a host for unknown reasons.",
		},
		{
			name:                 "no non-compliance detail",
			nonCompliance:        []*androidmanagement.NonComplianceDetail{},
			expectedErrorMessage: "Settings couldn't apply to a host for unknown reasons.",
		},
		{
			name: "single non-compliance detail",
			nonCompliance: []*androidmanagement.NonComplianceDetail{
				{
					SettingName:         "bluetoothDisabled",
					NonComplianceReason: "MANAGEMENT_MODE",
				},
			},
			expectedErrorMessage: "\"bluetoothDisabled\" setting couldn't apply to a host.\nReason: MANAGEMENT_MODE. Other settings are applied.",
		},
		{
			name: "two non-compliance details",
			nonCompliance: []*androidmanagement.NonComplianceDetail{
				{
					SettingName:         "bluetoothDisabled",
					NonComplianceReason: "MANAGEMENT_MODE",
				},
				{
					SettingName:         "cameraDisabled",
					NonComplianceReason: "API_LEVEL",
				},
			},
			expectedErrorMessage: "\"bluetoothDisabled\", and \"cameraDisabled\" settings couldn't apply to a host.\nReasons: MANAGEMENT_MODE, and API_LEVEL. Other settings are applied.",
		},
		{
			name: "three non-compliance details",
			nonCompliance: []*androidmanagement.NonComplianceDetail{
				{
					SettingName:         "bluetoothDisabled",
					NonComplianceReason: "MANAGEMENT_MODE",
				},
				{
					SettingName:         "cameraDisabled",
					NonComplianceReason: "API_LEVEL",
				},
				{
					SettingName:         "someCoolNewSetting",
					NonComplianceReason: "UNSUPPORTED",
				},
			},
			expectedErrorMessage: "\"bluetoothDisabled\", \"cameraDisabled\", and \"someCoolNewSetting\" settings couldn't apply to a host.\nReasons: MANAGEMENT_MODE, API_LEVEL, and UNSUPPORTED. Other settings are applied.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualMessage := buildNonComplianceErrorMessage(tc.nonCompliance)
			require.Equal(t, tc.expectedErrorMessage, actualMessage)
		})
	}
}

func TestStatusReportAppInstallVerification(t *testing.T) {
	svc, mockDS := createAndroidService(t)

	androidDevice := &fleet.AndroidHost{
		Host: &fleet.Host{
			UUID: uuid.NewString(),
		},
		Device: &android.Device{
			DeviceID: createAndroidDeviceId("test"),
		},
	}
	mockDS.AndroidHostLiteFunc = func(ctx context.Context, enterpriseSpecificID string) (*fleet.AndroidHost, error) {
		return androidDevice, nil
	}
	mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			MDM: fleet.MDM{
				AndroidEnabledAndConfigured: true,
			},
		}, nil
	}
	mockDS.UpdateAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, fromEnroll, companyOwned bool) error {
		return nil
	}
	mockDS.ListHostMDMAndroidProfilesPendingOrFailedInstallWithVersionFunc = func(ctx context.Context, hostUUID string, version int64) ([]*fleet.MDMAndroidProfilePayload, error) {
		return nil, nil
	}
	mockDS.BulkUpsertMDMAndroidHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAndroidProfilePayload) error {
		return nil
	}
	mockDS.BulkDeleteMDMAndroidHostProfilesFunc = func(ctx context.Context, hostUUID string, policyVersionID int64) error {
		return nil
	}
	mockDS.UpdateHostSoftwareFunc = func(ctx context.Context, hostID uint, software []fleet.Software) (*fleet.UpdateHostSoftwareDBResult, error) {
		return &fleet.UpdateHostSoftwareDBResult{}, nil
	}
	mockDS.GetAndroidPolicyRequestByUUIDFunc = func(ctx context.Context, id string) (*android.MDMAndroidPolicyRequest, error) {
		return nil, &notFoundError{}
	}
	mockDS.GetPastActivityDataForAndroidVPPAppInstallFunc = func(ctx context.Context, cmdUUID string, status fleet.SoftwareInstallerStatus) (*fleet.User, *fleet.ActivityInstalledAppStoreApp, error) {
		return nil, nil, nil
	}

	t.Run("no pending app install", func(t *testing.T) {
		t.Cleanup(func() {
			mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFuncInvoked = false
			mockDS.BulkSetVPPInstallsAsVerifiedFuncInvoked = false
			mockDS.BulkSetVPPInstallsAsFailedFuncInvoked = false
		})

		policyVersion := new(1)

		mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFunc = func(ctx context.Context, hostUUID string, version int64) ([]*fleet.HostAndroidVPPSoftwareInstall, error) {
			return nil, nil
		}
		mockDS.BulkSetVPPInstallsAsVerifiedFunc = func(ctx context.Context, hostID uint, cmdUUIDs []string) error {
			return nil
		}
		mockDS.BulkSetVPPInstallsAsFailedFunc = func(ctx context.Context, hostID uint, cmdUUIDs []string) error {
			return nil
		}

		enrollmentMessage := createStatusReportMessage(t, androidDevice.UUID, "test", createAndroidDeviceId("test"), policyVersion, nil)
		err := svc.ProcessPubSubPush(context.Background(), "value", &enrollmentMessage)
		require.NoError(t, err)

		require.True(t, mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFuncInvoked)
		require.False(t, mockDS.BulkSetVPPInstallsAsVerifiedFuncInvoked)
		require.False(t, mockDS.BulkSetVPPInstallsAsFailedFuncInvoked)
	})

	t.Run("pending app but in a future version", func(t *testing.T) {
		t.Cleanup(func() {
			mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFuncInvoked = false
			mockDS.BulkSetVPPInstallsAsVerifiedFuncInvoked = false
			mockDS.BulkSetVPPInstallsAsFailedFuncInvoked = false
		})

		pendingApp := &fleet.HostAndroidVPPSoftwareInstall{
			AdamID:            "com.example.app",
			CommandUUID:       "a",
			AssociatedEventID: "2", // future policy version
		}
		mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFunc = func(ctx context.Context, hostUUID string, version int64) ([]*fleet.HostAndroidVPPSoftwareInstall, error) {
			appVersion, _ := strconv.Atoi(pendingApp.AssociatedEventID)
			if int64(appVersion) <= version {
				return []*fleet.HostAndroidVPPSoftwareInstall{pendingApp}, nil
			}
			return nil, nil
		}
		mockDS.BulkSetVPPInstallsAsVerifiedFunc = func(ctx context.Context, hostID uint, cmdUUIDs []string) error {
			return nil
		}
		mockDS.BulkSetVPPInstallsAsFailedFunc = func(ctx context.Context, hostID uint, cmdUUIDs []string) error {
			return nil
		}

		policyVersion := new(1)
		enrollmentMessage := createStatusReportMessage(t, androidDevice.UUID, "test", createAndroidDeviceId("test"), policyVersion, nil)
		err := svc.ProcessPubSubPush(context.Background(), "value", &enrollmentMessage)
		require.NoError(t, err)

		require.True(t, mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFuncInvoked)
		require.False(t, mockDS.BulkSetVPPInstallsAsVerifiedFuncInvoked)
		require.False(t, mockDS.BulkSetVPPInstallsAsFailedFuncInvoked)
	})

	t.Run("pending app verified", func(t *testing.T) {
		t.Cleanup(func() {
			mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFuncInvoked = false
			mockDS.BulkSetVPPInstallsAsVerifiedFuncInvoked = false
			mockDS.BulkSetVPPInstallsAsFailedFuncInvoked = false
		})

		pendingApp := &fleet.HostAndroidVPPSoftwareInstall{
			AdamID:            "com.example.app",
			CommandUUID:       "a",
			AssociatedEventID: "2",
		}
		mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFunc = func(ctx context.Context, hostUUID string, version int64) ([]*fleet.HostAndroidVPPSoftwareInstall, error) {
			appVersion, _ := strconv.Atoi(pendingApp.AssociatedEventID)
			if int64(appVersion) <= version {
				return []*fleet.HostAndroidVPPSoftwareInstall{pendingApp}, nil
			}
			return nil, nil
		}
		mockDS.BulkSetVPPInstallsAsVerifiedFunc = func(ctx context.Context, hostID uint, cmdUUIDs []string) error {
			require.Equal(t, []string{pendingApp.CommandUUID}, cmdUUIDs)
			return nil
		}
		mockDS.BulkSetVPPInstallsAsFailedFunc = func(ctx context.Context, hostID uint, cmdUUIDs []string) error {
			require.Empty(t, cmdUUIDs)
			return nil
		}

		policyVersion := new(2)
		enrollmentMessage := createStatusAppReportMessage(t, androidDevice.UUID, "test", createAndroidDeviceId("test"), policyVersion, []*androidmanagement.ApplicationReport{
			{PackageName: pendingApp.AdamID, State: "INSTALLED"},
		}, nil)
		err := svc.ProcessPubSubPush(context.Background(), "value", &enrollmentMessage)
		require.NoError(t, err)

		require.True(t, mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFuncInvoked)
		require.True(t, mockDS.BulkSetVPPInstallsAsVerifiedFuncInvoked)
		require.True(t, mockDS.BulkSetVPPInstallsAsFailedFuncInvoked)
	})

	t.Run("pending app verified with unrelated non-compliance event", func(t *testing.T) {
		t.Cleanup(func() {
			mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFuncInvoked = false
			mockDS.BulkSetVPPInstallsAsVerifiedFuncInvoked = false
			mockDS.BulkSetVPPInstallsAsFailedFuncInvoked = false
		})

		pendingApp := &fleet.HostAndroidVPPSoftwareInstall{
			AdamID:            "com.example.app",
			CommandUUID:       "a",
			AssociatedEventID: "2",
		}
		mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFunc = func(ctx context.Context, hostUUID string, version int64) ([]*fleet.HostAndroidVPPSoftwareInstall, error) {
			appVersion, _ := strconv.Atoi(pendingApp.AssociatedEventID)
			if int64(appVersion) <= version {
				return []*fleet.HostAndroidVPPSoftwareInstall{pendingApp}, nil
			}
			return nil, nil
		}
		mockDS.BulkSetVPPInstallsAsVerifiedFunc = func(ctx context.Context, hostID uint, cmdUUIDs []string) error {
			require.Equal(t, []string{pendingApp.CommandUUID}, cmdUUIDs)
			return nil
		}
		mockDS.BulkSetVPPInstallsAsFailedFunc = func(ctx context.Context, hostID uint, cmdUUIDs []string) error {
			require.Empty(t, cmdUUIDs)
			return nil
		}

		policyVersion := new(2)
		enrollmentMessage := createStatusAppReportMessage(t, androidDevice.UUID, "test", createAndroidDeviceId("test"), policyVersion, []*androidmanagement.ApplicationReport{
			{PackageName: pendingApp.AdamID, State: "INSTALLED"},
		}, []*androidmanagement.NonComplianceDetail{
			{SettingName: "DefaultPermissionPolicy", NonComplianceReason: "INVALID_VALUE"},
		})
		err := svc.ProcessPubSubPush(context.Background(), "value", &enrollmentMessage)
		require.NoError(t, err)

		require.True(t, mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFuncInvoked)
		require.True(t, mockDS.BulkSetVPPInstallsAsVerifiedFuncInvoked)
		require.True(t, mockDS.BulkSetVPPInstallsAsFailedFuncInvoked)
	})

	t.Run("pending app failed with non-compliance", func(t *testing.T) {
		t.Cleanup(func() {
			mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFuncInvoked = false
			mockDS.BulkSetVPPInstallsAsVerifiedFuncInvoked = false
			mockDS.BulkSetVPPInstallsAsFailedFuncInvoked = false
		})

		pendingApp := &fleet.HostAndroidVPPSoftwareInstall{
			AdamID:            "com.example.app",
			CommandUUID:       "a",
			AssociatedEventID: "2",
		}
		mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFunc = func(ctx context.Context, hostUUID string, version int64) ([]*fleet.HostAndroidVPPSoftwareInstall, error) {
			appVersion, _ := strconv.Atoi(pendingApp.AssociatedEventID)
			if int64(appVersion) <= version {
				return []*fleet.HostAndroidVPPSoftwareInstall{pendingApp}, nil
			}
			return nil, nil
		}
		mockDS.BulkSetVPPInstallsAsVerifiedFunc = func(ctx context.Context, hostID uint, cmdUUIDs []string) error {
			require.Empty(t, cmdUUIDs)
			return nil
		}
		mockDS.BulkSetVPPInstallsAsFailedFunc = func(ctx context.Context, hostID uint, cmdUUIDs []string) error {
			require.Equal(t, []string{pendingApp.CommandUUID}, cmdUUIDs)
			return nil
		}

		policyVersion := new(2)
		enrollmentMessage := createStatusAppReportMessage(t, androidDevice.UUID, "test", createAndroidDeviceId("test"), policyVersion, []*androidmanagement.ApplicationReport{
			{PackageName: pendingApp.AdamID, State: "APPLICATION_STATE_UNSPECIFIED"},
		}, []*androidmanagement.NonComplianceDetail{
			{PackageName: pendingApp.AdamID, NonComplianceReason: "APP_NOT_INSTALLED", InstallationFailureReason: "NOT_FOUND"},
		})
		err := svc.ProcessPubSubPush(context.Background(), "value", &enrollmentMessage)
		require.NoError(t, err)

		require.True(t, mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFuncInvoked)
		require.True(t, mockDS.BulkSetVPPInstallsAsVerifiedFuncInvoked)
		require.True(t, mockDS.BulkSetVPPInstallsAsFailedFuncInvoked)
	})

	t.Run("pending app in progress", func(t *testing.T) {
		t.Cleanup(func() {
			mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFuncInvoked = false
			mockDS.BulkSetVPPInstallsAsVerifiedFuncInvoked = false
			mockDS.BulkSetVPPInstallsAsFailedFuncInvoked = false
		})

		pendingApp := &fleet.HostAndroidVPPSoftwareInstall{
			AdamID:            "com.example.app",
			CommandUUID:       "a",
			AssociatedEventID: "2",
		}
		mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFunc = func(ctx context.Context, hostUUID string, version int64) ([]*fleet.HostAndroidVPPSoftwareInstall, error) {
			appVersion, _ := strconv.Atoi(pendingApp.AssociatedEventID)
			if int64(appVersion) <= version {
				return []*fleet.HostAndroidVPPSoftwareInstall{pendingApp}, nil
			}
			return nil, nil
		}
		mockDS.BulkSetVPPInstallsAsVerifiedFunc = func(ctx context.Context, hostID uint, cmdUUIDs []string) error {
			require.Empty(t, cmdUUIDs)
			return nil
		}
		mockDS.BulkSetVPPInstallsAsFailedFunc = func(ctx context.Context, hostID uint, cmdUUIDs []string) error {
			require.Empty(t, cmdUUIDs)
			return nil
		}

		policyVersion := new(2)
		enrollmentMessage := createStatusAppReportMessage(t, androidDevice.UUID, "test", createAndroidDeviceId("test"), policyVersion, nil, []*androidmanagement.NonComplianceDetail{
			{PackageName: pendingApp.AdamID, NonComplianceReason: "PENDING", InstallationFailureReason: "IN_PROGRESS"},
		})
		err := svc.ProcessPubSubPush(context.Background(), "value", &enrollmentMessage)
		require.NoError(t, err)

		require.True(t, mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFuncInvoked)
		require.True(t, mockDS.BulkSetVPPInstallsAsVerifiedFuncInvoked)
		require.True(t, mockDS.BulkSetVPPInstallsAsFailedFuncInvoked)
	})

	t.Run("multiple apps in various states", func(t *testing.T) {
		t.Cleanup(func() {
			mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFuncInvoked = false
			mockDS.BulkSetVPPInstallsAsVerifiedFuncInvoked = false
			mockDS.BulkSetVPPInstallsAsFailedFuncInvoked = false
			mockDS.GetPastActivityDataForAndroidVPPAppInstallFuncInvoked = false
		})

		// 4 apps installed in the same policy, so same version,
		// and 1 more in a future policy (not possible for setup experience, but
		// tests the logic)
		pendingApps := []*fleet.HostAndroidVPPSoftwareInstall{
			{
				AdamID:            "com.example.app1",
				CommandUUID:       "a",
				AssociatedEventID: "2",
			},
			{
				AdamID:            "com.example.app2",
				CommandUUID:       "b",
				AssociatedEventID: "2",
			},
			{
				AdamID:            "com.example.app3",
				CommandUUID:       "c",
				AssociatedEventID: "2",
			},
			{
				AdamID:            "com.example.app4",
				CommandUUID:       "d",
				AssociatedEventID: "2",
			},
			{
				AdamID:            "com.example.app5",
				CommandUUID:       "e",
				AssociatedEventID: "3",
			},
		}
		mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFunc = func(ctx context.Context, hostUUID string, version int64) ([]*fleet.HostAndroidVPPSoftwareInstall, error) {
			switch version {
			case 0, 1:
				return nil, nil
			case 2:
				return pendingApps[:4], nil
			default:
				return pendingApps, nil
			}
		}
		commandsToStatus := map[string]fleet.SoftwareInstallerStatus{
			"a": fleet.SoftwareInstalled,
			"b": fleet.SoftwareInstalled,
			"c": fleet.SoftwareInstallFailed,
			"d": fleet.SoftwareInstallFailed,
		}
		mockDS.BulkSetVPPInstallsAsVerifiedFunc = func(ctx context.Context, hostID uint, cmdUUIDs []string) error {
			require.ElementsMatch(t, []string{"a", "b"}, cmdUUIDs)
			return nil
		}
		mockDS.BulkSetVPPInstallsAsFailedFunc = func(ctx context.Context, hostID uint, cmdUUIDs []string) error {
			require.ElementsMatch(t, []string{"c", "d"}, cmdUUIDs)
			return nil
		}
		mockDS.GetPastActivityDataForAndroidVPPAppInstallFunc = func(ctx context.Context, cmdUUID string, status fleet.SoftwareInstallerStatus) (*fleet.User, *fleet.ActivityInstalledAppStoreApp, error) {
			want, ok := commandsToStatus[cmdUUID]
			require.True(t, ok, "unexpected command UUID: %s", cmdUUID)
			require.Equal(t, want, status)
			return &fleet.User{}, &fleet.ActivityInstalledAppStoreApp{CommandUUID: cmdUUID, Status: string(status)}, nil
		}

		policyVersion := new(2)
		// app1 and app2 verified, app3 not reported at all so failed, app4 failed with compliance report
		enrollmentMessage := createStatusAppReportMessage(t, androidDevice.UUID, "test", createAndroidDeviceId("test"), policyVersion, []*androidmanagement.ApplicationReport{
			{PackageName: pendingApps[0].AdamID, State: "INSTALLED"},
			{PackageName: pendingApps[1].AdamID, State: "INSTALLED"},
		}, []*androidmanagement.NonComplianceDetail{
			{PackageName: pendingApps[3].AdamID, NonComplianceReason: "APP_NOT_INSTALLED", InstallationFailureReason: "NOT_APPROVED"},
		})
		err := svc.ProcessPubSubPush(context.Background(), "value", &enrollmentMessage)
		require.NoError(t, err)

		require.True(t, mockDS.ListHostMDMAndroidVPPAppsPendingInstallWithVersionFuncInvoked)
		require.True(t, mockDS.BulkSetVPPInstallsAsVerifiedFuncInvoked)
		require.True(t, mockDS.BulkSetVPPInstallsAsFailedFuncInvoked)
		require.True(t, mockDS.GetPastActivityDataForAndroidVPPAppInstallFuncInvoked)
	})
}

// Status report arrives for an Android host that was deleted from Fleet: the
// handler must re-enroll the host and then read it back from primary (issue #42494).
func TestPubSubStatusReportHostDeletedFromFleet(t *testing.T) {
	svc, mockDS := createAndroidService(t)

	mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: true}}, nil
	}

	// AndroidHostLite returns not-found initially (host was deleted from Fleet) and
	// returns the created host only after enrollHost has run.
	var createdHost *fleet.AndroidHost
	mockDS.AndroidHostLiteFunc = func(ctx context.Context, enterpriseSpecificID string) (*fleet.AndroidHost, error) {
		if createdHost != nil && ctxdb.IsPrimaryRequired(ctx) {
			return createdHost, nil
		}
		return nil, common_mysql.NotFound("android host lite mock")
	}
	mockDS.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
		return &fleet.EnrollSecret{Secret: "global"}, nil
	}
	mockDS.NewAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, companyOwned bool) (*fleet.AndroidHost, error) {
		createdHost = host
		return host, nil
	}
	mockDS.UpdateAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, fromEnroll, companyOwned bool) error {
		return nil
	}

	// Minimal device: no AppliedPolicyName / ApplicationReports, so the status report
	// handler stays on the simple path (skips policy + software verification).
	// EnrollmentTokenData is present because production AMAPI payloads include it;
	// without it, enrollHost would fail to unmarshal and never exercise the fix.
	device := androidmanagement.Device{
		Name:                createAndroidDeviceId("deleted-from-fleet"),
		EnrollmentTokenData: `{"enroll_secret": "global"}`,
		HardwareInfo: &androidmanagement.HardwareInfo{
			EnterpriseSpecificId: strings.ToUpper(uuid.New().String()),
			Brand:                "TestBrand",
			Model:                "TestModel",
			SerialNumber:         "test-serial",
			Hardware:             "test-hardware",
		},
		SoftwareInfo: &androidmanagement.SoftwareInfo{AndroidBuildNumber: "test-build", AndroidVersion: "1"},
		MemoryInfo: &androidmanagement.MemoryInfo{
			TotalRam:             int64(8 * 1024 * 1024 * 1024),
			TotalInternalStorage: int64(64 * 1024 * 1024 * 1024),
		},
	}
	data, err := json.Marshal(device)
	require.NoError(t, err)
	statusReport := &android.PubSubMessage{
		Attributes: map[string]string{"notificationType": string(android.PubSubStatusReport)},
		Data:       base64.StdEncoding.EncodeToString(data),
	}

	err = svc.ProcessPubSubPush(context.Background(), "value", statusReport)
	require.NoError(t, err)
	require.True(t, mockDS.NewAndroidHostFuncInvoked, "re-enrollment should create the host")
	require.True(t, mockDS.UpdateAndroidHostFuncInvoked, "status report update should run against the re-enrolled host")
}

func TestPubSubStatusReport_DoesNotPanicWhenHardwareInfoMissing(t *testing.T) {
	svc, mockDS := createAndroidService(t)

	mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: true}}, nil
	}

	deviceJSON := `{"name":"enterprises/E1/devices/abc123","appliedState":"ACTIVE"}`
	msg := &android.PubSubMessage{
		Attributes: map[string]string{"notificationType": string(android.PubSubStatusReport)},
		Data:       base64.StdEncoding.EncodeToString([]byte(deviceJSON)),
	}

	require.NotPanics(t, func() {
		err := svc.ProcessPubSubPush(t.Context(), "value", msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing hardware info")
	})
}

func TestPubSubCommand(t *testing.T) {
	const validToken = "value"
	makeMessage := func(t *testing.T, op androidmanagement.Operation) *android.PubSubMessage {
		body, err := json.Marshal(op)
		require.NoError(t, err)
		return &android.PubSubMessage{
			Attributes: map[string]string{"notificationType": string(android.PubSubCommand)},
			Data:       base64.StdEncoding.EncodeToString(body),
		}
	}

	// newSvc builds the per-subtest fixture: a service whose mock datastore reports Android MDM as enabled. Subtests override
	// individual *Func fields on mockDS to shape the specific branch they're exercising.
	newSvc := func(t *testing.T) (android.Service, *AndroidMockDS) {
		svc, mockDS := createAndroidService(t)
		mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: true}}, nil
		}
		return svc, mockDS
	}
	// commandNotFound shapes the "unknown operation" branches: the datastore lookup by operation_name returns NotFound, simulating
	// either a race (row not yet committed) or an orphan (device deleted from Fleet).
	commandNotFound := func(mockDS *AndroidMockDS) {
		mockDS.GetMDMAndroidCommandByOperationNameFunc = func(ctx context.Context, opName string) (*android.MDMAndroidCommand, error) {
			return nil, common_mysql.NotFound("MDMAndroidCommand").WithName(opName)
		}
	}

	t.Run("pending -> acknowledged on op with no error", func(t *testing.T) {
		svc, mockDS := newSvc(t)

		stored := &android.MDMAndroidCommand{
			CommandUUID:   "cmd-uuid-ack",
			HostUUID:      "host-uuid",
			OperationName: "enterprises/E/devices/D/operations/ack-1",
			CommandType:   string(android.MDMAndroidCommandTypeLock),
			Status:        string(android.MDMAndroidCommandStatusPending),
		}
		mockDS.GetMDMAndroidCommandByOperationNameFunc = func(ctx context.Context, opName string) (*android.MDMAndroidCommand, error) {
			require.Equal(t, stored.OperationName, opName)
			return stored, nil
		}
		var capturedStatus string
		var capturedErrCode, capturedErrMsg *string
		mockDS.UpdateMDMAndroidCommandStatusFunc = func(ctx context.Context, commandUUID, status string, errorCode, errorMessage *string) error {
			require.Equal(t, stored.CommandUUID, commandUUID)
			capturedStatus = status
			capturedErrCode = errorCode
			capturedErrMsg = errorMessage
			return nil
		}

		msg := makeMessage(t, androidmanagement.Operation{Name: stored.OperationName, Done: true})
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), validToken, msg))

		require.True(t, mockDS.UpdateMDMAndroidCommandStatusFuncInvoked)
		require.Equal(t, string(android.MDMAndroidCommandStatusAcknowledged), capturedStatus)
		require.Nil(t, capturedErrCode)
		require.Nil(t, capturedErrMsg)
	})

	t.Run("pending -> error on op.Error set", func(t *testing.T) {
		svc, mockDS := newSvc(t)

		stored := &android.MDMAndroidCommand{
			CommandUUID:   "cmd-uuid-err",
			HostUUID:      "host-uuid",
			OperationName: "enterprises/E/devices/D/operations/err-1",
			CommandType:   string(android.MDMAndroidCommandTypeWipe),
			Status:        string(android.MDMAndroidCommandStatusPending),
		}
		mockDS.GetMDMAndroidCommandByOperationNameFunc = func(ctx context.Context, opName string) (*android.MDMAndroidCommand, error) {
			return stored, nil
		}
		var capturedStatus, capturedCode, capturedMsg string
		mockDS.UpdateMDMAndroidCommandStatusFunc = func(ctx context.Context, commandUUID, status string, errorCode, errorMessage *string) error {
			capturedStatus = status
			if errorCode != nil {
				capturedCode = *errorCode
			}
			if errorMessage != nil {
				capturedMsg = *errorMessage
			}
			return nil
		}

		msg := makeMessage(t, androidmanagement.Operation{
			Name:  stored.OperationName,
			Done:  true,
			Error: &androidmanagement.Status{Code: 13, Message: "device does not support WIPE"},
		})
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), validToken, msg))

		require.Equal(t, string(android.MDMAndroidCommandStatusError), capturedStatus)
		require.Equal(t, "13", capturedCode)
		require.Equal(t, "device does not support WIPE", capturedMsg)
	})

	t.Run("already-terminal status is not re-transitioned", func(t *testing.T) {
		svc, mockDS := newSvc(t)

		stored := &android.MDMAndroidCommand{
			CommandUUID:   "cmd-uuid-ack-already",
			OperationName: "enterprises/E/devices/D/operations/redelivered",
			CommandType:   string(android.MDMAndroidCommandTypeLock),
			Status:        string(android.MDMAndroidCommandStatusAcknowledged),
		}
		mockDS.GetMDMAndroidCommandByOperationNameFunc = func(ctx context.Context, opName string) (*android.MDMAndroidCommand, error) {
			return stored, nil
		}
		mockDS.UpdateMDMAndroidCommandStatusFunc = func(ctx context.Context, commandUUID, status string, errorCode, errorMessage *string) error {
			t.Fatalf("UpdateMDMAndroidCommandStatus should not be called for terminal row")
			return nil
		}

		msg := makeMessage(t, androidmanagement.Operation{Name: stored.OperationName, Done: true})
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), validToken, msg))
		require.False(t, mockDS.UpdateMDMAndroidCommandStatusFuncInvoked)
	})

	t.Run("unknown operation, host deleted from Fleet -> ack", func(t *testing.T) {
		// COBO unenroll / manual cleanup: the host is gone from Fleet, the command row is gone from mdm_android_commands, but Pub/Sub is
		// still trying to deliver the original notification. A false from AndroidDeviceExistsByDeviceID confirms the orphan; ack.
		svc, mockDS := newSvc(t)
		commandNotFound(mockDS)
		mockDS.AndroidDeviceExistsByDeviceIDFunc = func(ctx context.Context, deviceID string) (bool, error) {
			require.Equal(t, "D", deviceID, "handler should pass the AMAPI device_id parsed from op.Name")
			return false, nil
		}

		msg := makeMessage(t, androidmanagement.Operation{Name: "enterprises/E/devices/D/operations/orphan", Done: true})
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), validToken, msg))
	})

	t.Run("unknown operation, host still exists in Fleet -> retry (race window)", func(t *testing.T) {
		// AMAPI delivered the COMMAND notification before our IssueCommand-then-insert sequence committed the row. The DB check confirms
		// the device is still in Fleet; returning error makes Pub/Sub retry until the row commits.
		svc, mockDS := newSvc(t)
		commandNotFound(mockDS)
		mockDS.AndroidDeviceExistsByDeviceIDFunc = func(ctx context.Context, deviceID string) (bool, error) {
			return true, nil
		}

		msg := makeMessage(t, androidmanagement.Operation{Name: "enterprises/E/devices/D/operations/racy", Done: true})
		require.Error(t, svc.ProcessPubSubPush(t.Context(), validToken, msg))
	})

	t.Run("unknown operation, malformed op.Name (no /devices/ segment) -> ack", func(t *testing.T) {
		// If we can't parse a device_id from op.Name we can't probe Fleet's DB; ack rather than retry forever.
		svc, mockDS := newSvc(t)
		commandNotFound(mockDS)
		mockDS.AndroidDeviceExistsByDeviceIDFunc = func(ctx context.Context, deviceID string) (bool, error) {
			t.Fatalf("device existence check should not run when op.Name is malformed")
			return false, nil
		}

		msg := makeMessage(t, androidmanagement.Operation{Name: "not-a-real-operation-name", Done: true})
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), validToken, msg))
	})

	t.Run("non-done operation is acked without state transition", func(t *testing.T) {
		// AMAPI long-running Operations use done=false for in-progress states. The handler must not transition state on a non-terminal
		// payload.
		svc, mockDS := newSvc(t)
		mockDS.GetMDMAndroidCommandByOperationNameFunc = func(ctx context.Context, opName string) (*android.MDMAndroidCommand, error) {
			t.Fatalf("lookup should be skipped when op.Done is false")
			return nil, nil
		}
		mockDS.UpdateMDMAndroidCommandStatusFunc = func(ctx context.Context, commandUUID, status string, errorCode, errorMessage *string) error {
			t.Fatalf("status update should be skipped when op.Done is false")
			return nil
		}

		msg := makeMessage(t, androidmanagement.Operation{Name: "enterprises/E/devices/D/operations/in-flight", Done: false})
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), validToken, msg))
	})

	t.Run("empty op.Name is acked without error", func(t *testing.T) {
		svc, mockDS := newSvc(t)
		mockDS.GetMDMAndroidCommandByOperationNameFunc = func(ctx context.Context, opName string) (*android.MDMAndroidCommand, error) {
			t.Fatalf("lookup should be skipped when op.Name is empty")
			return nil, nil
		}

		msg := makeMessage(t, androidmanagement.Operation{Done: true})
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), validToken, msg))
	})
}

func TestPubSubEnrollment_DoesNotPanicWhenHardwareInfoMissing(t *testing.T) {
	svc, mockDS := createAndroidService(t)

	mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: true}}, nil
	}

	deviceJSON := `{"name":"enterprises/E1/devices/abc123","appliedState":"DELETED"}`
	msg := &android.PubSubMessage{
		Attributes: map[string]string{"notificationType": string(android.PubSubEnrollment)},
		Data:       base64.StdEncoding.EncodeToString([]byte(deviceJSON)),
	}

	require.NotPanics(t, func() {
		err := svc.ProcessPubSubPush(t.Context(), "value", msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing hardware info")
	})
}
