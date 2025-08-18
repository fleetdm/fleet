package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
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
		TotalRam: 1000,
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

func createAndroidDeviceId(name string) string {
	return "enterprises/mock-enterprise-id/devices/" + name
}
