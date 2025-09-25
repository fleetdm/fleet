package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

func createAndroidService(t *testing.T) (android.Service, *AndroidMockDS) {
	androidAPIClient := android_mock.Client{}
	androidAPIClient.InitCommonMocks()
	logger := kitlog.NewLogfmtLogger(os.Stdout)
	mockDS := InitCommonDSMocks()
	fleetSvc := mockService{}
	svc, err := NewServiceWithClient(logger, mockDS, &androidAPIClient, &fleetSvc, "test-private-key")
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
			require.Error(t, err)
			require.Equal(t, "validation failed: android Android MDM is NOT configured", err.Error())
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

			mockDS.NewAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost) (*fleet.AndroidHost, error) {
				return nil, nil // We do not care about return value here
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

			mockDS.NewAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost) (*fleet.AndroidHost, error) {
				return nil, nil // We do not care about return value here
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
			})
			err = svc.ProcessPubSubPush(context.Background(), "value", enrollmentMessage)
			require.NoError(t, err)

			require.True(t, mockDS.AssociateHostMDMIdPAccountFuncInvoked)
			require.True(t, mockDS.NewAndroidHostFuncInvoked)
		})
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
	mockDS.UpdateAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, fromEnroll bool) error {
		return nil
	}

	t.Run("single install pending profile with empty compliance details", func(t *testing.T) {
		policyVersion := ptr.Int(1)

		installPendingProfile := &fleet.MDMAndroidProfilePayload{
			ProfileUUID:             uuid.NewString(),
			ProfileName:             "a",
			HostUUID:                androidDevice.UUID,
			Status:                  &fleet.MDMDeliveryPending,
			OperationType:           fleet.MDMOperationTypeInstall,
			IncludedInPolicyVersion: policyVersion,
		}
		mockDS.ListHostMDMAndroidProfilesPendingInstallWithVersionFunc = func(ctx context.Context, hostUUID string, version int64) ([]*fleet.MDMAndroidProfilePayload, error) {
			return []*fleet.MDMAndroidProfilePayload{
				installPendingProfile,
			}, nil
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

		require.True(t, mockDS.ListHostMDMAndroidProfilesPendingInstallWithVersionFuncInvoked)
		require.False(t, mockDS.GetAndroidPolicyRequestByUUIDFuncInvoked)
		require.True(t, mockDS.BulkUpsertMDMAndroidHostProfilesFuncInvoked)
		require.True(t, mockDS.BulkDeleteMDMAndroidHostProfilesFuncInvoked)
		mockDS.ListHostMDMAndroidProfilesPendingInstallWithVersionFuncInvoked = false
		mockDS.BulkDeleteMDMAndroidHostProfilesFuncInvoked = false
		mockDS.BulkUpsertMDMAndroidHostProfilesFuncInvoked = false
	})

	t.Run("compliance details has failure", func(t *testing.T) {
		policyVersion := ptr.Int(1)

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

		mockDS.GetAndroidPolicyRequestByUUIDFunc = func(ctx context.Context, id string) (*fleet.MDMAndroidPolicyRequest, error) {
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
				return &fleet.MDMAndroidPolicyRequest{
					Payload: payload,
				}, nil
			}

			return nil, errors.New("something went wrong")
		}

		mockDS.ListHostMDMAndroidProfilesPendingInstallWithVersionFunc = func(ctx context.Context, hostUUID string, version int64) ([]*fleet.MDMAndroidProfilePayload, error) {
			return []*fleet.MDMAndroidProfilePayload{
				installPendingProfile1,
				installPendingProfile2,
			}, nil
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
		require.True(t, mockDS.ListHostMDMAndroidProfilesPendingInstallWithVersionFuncInvoked)
		require.True(t, mockDS.BulkUpsertMDMAndroidHostProfilesFuncInvoked)
		require.True(t, mockDS.BulkDeleteMDMAndroidHostProfilesFuncInvoked)
		mockDS.ListHostMDMAndroidProfilesPendingInstallWithVersionFuncInvoked = false
		mockDS.BulkDeleteMDMAndroidHostProfilesFuncInvoked = false
		mockDS.BulkUpsertMDMAndroidHostProfilesFuncInvoked = false
		mockDS.GetAndroidPolicyRequestByUUIDFuncInvoked = false
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
	mockDS.NewAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost) (*fleet.AndroidHost, error) {
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
	device := androidmanagement.Device{
		Name:                 createAndroidDeviceId(name),
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
