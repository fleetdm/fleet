package service

import (
	"context"
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
)

func newVerifyTestService(t *testing.T, opts ...ServiceOption) (*Service, *AndroidMockDS, *android_mock.Client) {
	t.Helper()
	client := &android_mock.Client{}
	client.InitCommonMocks()
	client.InitDeviceLookupMocks()
	mockDS := InitCommonDSMocks()
	mockDS.Store.GetEnterpriseFunc = func(ctx context.Context) (*android.Enterprise, error) {
		return &android.Enterprise{EnterpriseID: "mock-enterprise-id"}, nil
	}
	svc, err := NewServiceWithClient(slog.New(slog.NewTextHandler(os.Stdout, nil)), mockDS, client, "test-private-key",
		&mockDS.DataStore, noopNewActivity, config.AndroidAgentConfig{}, opts...)
	require.NoError(t, err)
	return svc.(*Service), mockDS, client
}

func decodePubSubDevice(t *testing.T, msg *android.PubSubMessage) androidmanagement.Device {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(msg.Data)
	require.NoError(t, err)
	var device androidmanagement.Device
	require.NoError(t, json.Unmarshal(raw, &device))
	return device
}

func amapiNotFound() error {
	return &googleapi.Error{Code: http.StatusNotFound, Message: "Requested entity was not found."}
}

// requireAckedAsUnverified checks that a rejected notification is acked (HTTP 200), so
// Pub/Sub does not keep redelivering it.
func requireAckedAsUnverified(t *testing.T, err error) {
	t.Helper()
	require.ErrorContains(t, err, "was not processed")
	statusErr, ok := errors.AsType[interface {
		error
		Status() int
	}](err)
	require.True(t, ok, "the error must carry an HTTP status")
	require.Equal(t, http.StatusOK, statusErr.Status())
}

func TestPubSubTokenComparison(t *testing.T) {
	svc, mockDS, _ := newVerifyTestService(t)
	mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: true}}, nil
	}

	// InitCommonDSMocks stores "value" as the token.
	for _, tc := range []struct {
		name  string
		token string
		ok    bool
	}{
		{"matching token", "value", true},
		{"same length, different token", "valuf", false},
		{"prefix of the token", "val", false},
		{"token with a suffix", "value1", false},
		{"empty token", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.authenticatePubSub(t.Context(), tc.token)
			if tc.ok {
				require.NoError(t, err)
				return
			}
			var authErr *fleet.AuthFailedError
			require.ErrorAs(t, err, &authErr)
		})
	}

	t.Run("an empty stored token matches nothing", func(t *testing.T) {
		mockDS.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
			queryerContext sqlx.QueryerContext,
		) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return map[fleet.MDMAssetName]fleet.MDMConfigAsset{fleet.MDMAssetAndroidPubSubToken: {Value: []byte{}}}, nil
		}
		var authErr *fleet.AuthFailedError
		require.ErrorAs(t, svc.authenticatePubSub(t.Context(), ""), &authErr)
	})
}

func TestPubSubEventTimeCap(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	svc, mockDS, _ := newVerifyTestService(t, WithClock(clock.NewMockClock(now)))

	t.Run("event times too far in the future are ignored", func(t *testing.T) {
		for _, tc := range []struct {
			name       string
			deviceTime string
			publish    string
			want       *time.Time
		}{
			{"past device time is used", "2026-11-01T00:00:00Z", "", new(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))},
			{"time within the skew is used", "2026-11-02T12:04:00Z", "", new(time.Date(2026, 11, 2, 12, 4, 0, 0, time.UTC))},
			{"future device time falls back to publish time", "2099-01-01T00:00:00Z", "2026-11-02T11:59:00Z", new(time.Date(2026, 11, 2, 11, 59, 0, 0, time.UTC))},
			{"future publish time alone gives no time", "", "2099-01-01T00:00:00Z", nil},
			{"both in the future gives no time", "2099-01-01T00:00:00Z", "2099-01-01T00:00:00Z", nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				got := svc.pubSubEventTime(tc.deviceTime, tc.publish)
				if tc.want == nil {
					require.Nil(t, got)
					return
				}
				require.NotNil(t, got)
				require.True(t, tc.want.Equal(*got), "got %s, want %s", got, tc.want)
			})
		}
	})

	t.Run("a stored future event time does not make later notifications stale", func(t *testing.T) {
		stored := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
		mockDS.GetAndroidPubSubDedupStateFunc = func(ctx context.Context, hostID uint) (string, *time.Time, error) {
			return "future-msg", &stored, nil
		}
		skip, err := svc.isDuplicateOrStalePubSub(t.Context(), 1, "real-msg", new(now.Add(-time.Minute)))
		require.NoError(t, err)
		require.False(t, skip)
	})

	t.Run("a stored past event time still drops older notifications", func(t *testing.T) {
		stored := now.Add(-time.Minute)
		mockDS.GetAndroidPubSubDedupStateFunc = func(ctx context.Context, hostID uint) (string, *time.Time, error) {
			return "last-msg", &stored, nil
		}
		skip, err := svc.isDuplicateOrStalePubSub(t.Context(), 1, "old-msg", new(now.Add(-time.Hour)))
		require.NoError(t, err)
		require.True(t, skip)
	})
}

func TestPubSubReenrollmentVerifiedWithAMAPI(t *testing.T) {
	const existingHostID uint = 42
	const enrollmentTime = "2026-11-01T10:00:00Z"

	newReenroll := func(t *testing.T) (*Service, *AndroidMockDS, *android_mock.Client, *android.PubSubMessage, androidmanagement.Device) {
		svc, mockDS, client := newVerifyTestService(t)
		setupReenrollMocks(t, mockDS, &fleet.AndroidHost{
			Host: &fleet.Host{ID: existingHostID, UUID: testBrandTestSerialHashed},
			Device: &android.Device{
				HostID:               existingHostID,
				DeviceID:             "device-old",
				EnterpriseSpecificID: new(testBrandTestSerialHashed),
			},
		})
		mockDS.AndroidResetOnReenrollmentFunc = func(ctx context.Context, hostID uint, hostUUID string, preserve bool) ([]*fleet.User, []fleet.ActivityDetails, error) {
			return nil, nil, nil
		}
		enrollTokenData, err := json.Marshal(enrollmentTokenRequest{EnrollSecret: "global"})
		require.NoError(t, err)
		msg := createEnrollmentMessage(t, androidmanagement.Device{
			Name:                createAndroidDeviceId("verify-reenroll"),
			EnrollmentTokenData: string(enrollTokenData),
			EnrollmentTime:      enrollmentTime,
			Ownership:           DeviceOwnershipCompanyOwned,
		})
		return svc, mockDS, client, msg, decodePubSubDevice(t, msg)
	}

	t.Run("matching AMAPI record resets the host", func(t *testing.T) {
		svc, mockDS, client, msg, payload := newReenroll(t)
		client.EnterprisesDevicesGetFunc = func(ctx context.Context, name string) (*androidmanagement.Device, error) {
			require.True(t, androidmgmt.RetryDisabled(ctx), "a lookup must not wait out a 429 past Pub/Sub's ack deadline")
			return &payload, nil
		}
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), "value", msg))
		require.True(t, client.EnterprisesDevicesGetFuncInvoked)
		require.True(t, mockDS.AndroidResetOnReenrollmentFuncInvoked)
		require.True(t, mockDS.UpdateAndroidHostFuncInvoked)
	})

	for _, tc := range []struct {
		name  string
		amapi func(payload androidmanagement.Device) (*androidmanagement.Device, error)
	}{
		{"device unknown to AMAPI", func(androidmanagement.Device) (*androidmanagement.Device, error) {
			return nil, amapiNotFound()
		}},
		{"device deleted in AMAPI", func(d androidmanagement.Device) (*androidmanagement.Device, error) {
			d.State = string(android.DeviceStateDeleted)
			return &d, nil
		}},
		{"different hardware identity", func(d androidmanagement.Device) (*androidmanagement.Device, error) {
			hw := *d.HardwareInfo
			hw.SerialNumber = "other-serial"
			d.HardwareInfo = &hw
			return &d, nil
		}},
		{"no hardware info in AMAPI", func(d androidmanagement.Device) (*androidmanagement.Device, error) {
			d.HardwareInfo = nil
			return &d, nil
		}},
		{"different enrollment time", func(d androidmanagement.Device) (*androidmanagement.Device, error) {
			d.EnrollmentTime = "2026-10-01T10:00:00Z"
			return &d, nil
		}},
	} {
		t.Run(tc.name+" leaves pending commands in place", func(t *testing.T) {
			svc, mockDS, client, msg, payload := newReenroll(t)
			client.EnterprisesDevicesGetFunc = func(ctx context.Context, name string) (*androidmanagement.Device, error) {
				require.Equal(t, payload.Name, name)
				return tc.amapi(payload)
			}
			requireAckedAsUnverified(t, svc.ProcessPubSubPush(t.Context(), "value", msg))
			require.False(t, mockDS.AndroidResetOnReenrollmentFuncInvoked, "an unverified re-enrollment must not clear pending commands")
			require.False(t, mockDS.UpdateAndroidHostFuncInvoked)
			require.False(t, mockDS.SetAndroidPubSubDedupStateFuncInvoked)
		})
	}

	t.Run("same enrollment time written differently resets the host", func(t *testing.T) {
		svc, mockDS, client, msg, payload := newReenroll(t)
		client.EnterprisesDevicesGetFunc = func(ctx context.Context, name string) (*androidmanagement.Device, error) {
			d := payload
			d.EnrollmentTime = "2026-11-01T10:00:00.000Z"
			return &d, nil
		}
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), "value", msg))
		require.True(t, mockDS.AndroidResetOnReenrollmentFuncInvoked)
	})

	t.Run("device outside this enterprise is not looked up", func(t *testing.T) {
		svc, mockDS, client, _, payload := newReenroll(t)
		payload.Name = "enterprises/other-enterprise/devices/verify-reenroll"
		android_mock.RegisterDevice(payload)
		requireAckedAsUnverified(t, svc.ProcessPubSubPush(t.Context(), "value", pubSubMessageFor(t, android.PubSubEnrollment, payload)))
		require.False(t, client.EnterprisesDevicesGetFuncInvoked)
		require.False(t, mockDS.AndroidResetOnReenrollmentFuncInvoked)
	})

	t.Run("AMAPI error is retried", func(t *testing.T) {
		svc, mockDS, client, msg, _ := newReenroll(t)
		client.EnterprisesDevicesGetFunc = func(ctx context.Context, name string) (*androidmanagement.Device, error) {
			return nil, &googleapi.Error{Code: http.StatusInternalServerError}
		}
		err := svc.ProcessPubSubPush(t.Context(), "value", msg)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "was not processed")
		require.False(t, mockDS.AndroidResetOnReenrollmentFuncInvoked)
	})

	t.Run("a device new to Fleet is not looked up", func(t *testing.T) {
		svc, mockDS, client, msg, _ := newReenroll(t)
		mockDS.AndroidHostLiteFunc = func(ctx context.Context, esID string) (*fleet.AndroidHost, error) {
			return nil, common_mysql.NotFound("android host")
		}
		mockDS.NewAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, companyOwned bool) (*fleet.AndroidHost, error) {
			host.Host.ID = existingHostID
			return host, nil
		}
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), "value", msg))
		require.False(t, client.EnterprisesDevicesGetFuncInvoked)
		require.True(t, mockDS.NewAndroidHostFuncInvoked)
	})

	// Creating a host for a device ID Fleet already has moves that device record to the new host.
	for _, tc := range []struct {
		name        string
		amapiDevice func(payload androidmanagement.Device) *androidmanagement.Device
		wantCreated bool
	}{
		{"device ID of an existing host with another hardware identity is dropped", func(d androidmanagement.Device) *androidmanagement.Device {
			hw := *d.HardwareInfo
			hw.EnterpriseSpecificId = "the-real-device"
			d.HardwareInfo = &hw
			return &d
		}, false},
		{"device ID of an existing host that AMAPI confirms creates the host", func(d androidmanagement.Device) *androidmanagement.Device {
			return &d
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, mockDS, client, msg, payload := newReenroll(t)
			mockDS.AndroidHostLiteFunc = func(ctx context.Context, esID string) (*fleet.AndroidHost, error) {
				return nil, common_mysql.NotFound("android host")
			}
			mockDS.AndroidDeviceExistsByDeviceIDFunc = func(ctx context.Context, deviceID string) (bool, error) {
				require.Equal(t, "verify-reenroll", deviceID)
				return true, nil
			}
			mockDS.NewAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, companyOwned bool) (*fleet.AndroidHost, error) {
				host.Host.ID = existingHostID + 1
				return host, nil
			}
			client.EnterprisesDevicesGetFunc = func(ctx context.Context, name string) (*androidmanagement.Device, error) {
				require.Equal(t, payload.Name, name)
				return tc.amapiDevice(payload), nil
			}
			err := svc.ProcessPubSubPush(t.Context(), "value", msg)
			if tc.wantCreated {
				require.NoError(t, err)
			} else {
				requireAckedAsUnverified(t, err)
			}
			require.Equal(t, tc.wantCreated, mockDS.NewAndroidHostFuncInvoked)
		})
	}
}

func TestPubSubDeletedVerifiedWithAMAPI(t *testing.T) {
	const hostID = uint(10)
	const hostUUID = "VERIFY-DELETED-HOST"

	newDeleted := func(t *testing.T) (*Service, *AndroidMockDS, *android_mock.Client) {
		svc, mockDS, client := newVerifyTestService(t)
		wireDedupHost(t, mockDS, hostID, hostUUID) // the host's current AMAPI device is "dedup"
		mockDS.GetAndroidPubSubDedupStateFunc = func(ctx context.Context, id uint) (string, *time.Time, error) {
			return "", nil, nil
		}
		return svc, mockDS, client
	}
	statusDeleted := func(t *testing.T) *android.PubSubMessage {
		return makeStatusEnvelope(t, hostUUID, "delete-msg", "2026-07-22T10:00:00Z", true)
	}
	enrollmentDeleted := func(t *testing.T) *android.PubSubMessage {
		msg := makeStatusEnvelope(t, hostUUID, "delete-msg", "2026-07-22T10:00:00Z", true)
		msg.Attributes["notificationType"] = string(android.PubSubEnrollment)
		return msg
	}

	for _, notification := range []struct {
		name string
		msg  func(t *testing.T) *android.PubSubMessage
	}{
		{"STATUS_REPORT", statusDeleted},
		{"ENROLLMENT", enrollmentDeleted},
	} {
		t.Run(notification.name, func(t *testing.T) {
			t.Run("device gone from AMAPI unenrolls the host", func(t *testing.T) {
				svc, mockDS, client := newDeleted(t)
				client.EnterprisesDevicesGetFunc = func(ctx context.Context, name string) (*androidmanagement.Device, error) {
					require.Equal(t, createAndroidDeviceId("dedup"), name)
					return nil, amapiNotFound()
				}
				require.NoError(t, svc.ProcessPubSubPush(t.Context(), dedupToken, notification.msg(t)))
				require.True(t, mockDS.SetAndroidHostUnenrolledFuncInvoked)
			})

			t.Run("device reported DELETED by AMAPI unenrolls the host", func(t *testing.T) {
				svc, mockDS, client := newDeleted(t)
				client.EnterprisesDevicesGetFunc = func(ctx context.Context, name string) (*androidmanagement.Device, error) {
					return &androidmanagement.Device{Name: name, State: string(android.DeviceStateDeleted)}, nil
				}
				require.NoError(t, svc.ProcessPubSubPush(t.Context(), dedupToken, notification.msg(t)))
				require.True(t, mockDS.SetAndroidHostUnenrolledFuncInvoked)
			})

			t.Run("device still active in AMAPI keeps the host enrolled", func(t *testing.T) {
				svc, mockDS, client := newDeleted(t)
				client.EnterprisesDevicesGetFunc = func(ctx context.Context, name string) (*androidmanagement.Device, error) {
					return &androidmanagement.Device{Name: name, State: "ACTIVE"}, nil
				}
				requireAckedAsUnverified(t, svc.ProcessPubSubPush(t.Context(), dedupToken, notification.msg(t)))
				require.False(t, mockDS.SetAndroidHostUnenrolledFuncInvoked)
				require.False(t, mockDS.ClearHostMDMActionsFuncInvoked, "the host's wipe state must be left alone")
				require.False(t, mockDS.SetAndroidPubSubDedupStateFuncInvoked)
			})

			t.Run("DELETED for a device that is not the host's current one is dropped", func(t *testing.T) {
				svc, mockDS, client := newDeleted(t)
				host := wireDedupHost(t, mockDS, hostID, hostUUID)
				host.Device.DeviceID = "current-device"
				client.EnterprisesDevicesGetFunc = func(ctx context.Context, name string) (*androidmanagement.Device, error) {
					return nil, amapiNotFound()
				}
				requireAckedAsUnverified(t, svc.ProcessPubSubPush(t.Context(), dedupToken, notification.msg(t)))
				require.False(t, mockDS.SetAndroidHostUnenrolledFuncInvoked)
				require.False(t, client.EnterprisesDevicesGetFuncInvoked, "the device binding is checked before calling AMAPI")
			})

			t.Run("DELETED naming the host's device under another enterprise is dropped", func(t *testing.T) {
				svc, mockDS, client := newDeleted(t)
				// The proxy answers 404 for an enterprise it does not know.
				client.EnterprisesDevicesGetFunc = func(ctx context.Context, name string) (*androidmanagement.Device, error) {
					return nil, amapiNotFound()
				}
				msg := notification.msg(t)
				device := decodePubSubDevice(t, msg)
				device.Name = "enterprises/other-enterprise/devices/dedup"
				msg.Data = pubSubMessageFor(t, android.PubSubStatusReport, device).Data
				requireAckedAsUnverified(t, svc.ProcessPubSubPush(t.Context(), dedupToken, msg))
				require.False(t, mockDS.SetAndroidHostUnenrolledFuncInvoked)
			})

			t.Run("DELETED for a host with no recorded device is dropped", func(t *testing.T) {
				svc, mockDS, _ := newDeleted(t)
				host := wireDedupHost(t, mockDS, hostID, hostUUID)
				host.Device.DeviceID = ""
				// A name with an empty device ID would otherwise equal the expected name.
				msg := notification.msg(t)
				device := decodePubSubDevice(t, msg)
				device.Name = createAndroidDeviceId("")
				msg.Data = pubSubMessageFor(t, android.PubSubStatusReport, device).Data
				requireAckedAsUnverified(t, svc.ProcessPubSubPush(t.Context(), dedupToken, msg))
				require.False(t, mockDS.SetAndroidHostUnenrolledFuncInvoked)
			})
		})
	}
}

func pubSubMessageFor(t *testing.T, notificationType android.NotificationType, device androidmanagement.Device) *android.PubSubMessage {
	t.Helper()
	data, err := json.Marshal(device)
	require.NoError(t, err)
	return &android.PubSubMessage{
		Attributes: map[string]string{"notificationType": string(notificationType)},
		Data:       base64.StdEncoding.EncodeToString(data),
	}
}

// TestPubSubStatusReportDeviceChangeVerifiedWithAMAPI covers a STATUS_REPORT that names a different
// AMAPI device than the host's. The new device is only stored if AMAPI has it, so a later DELETED
// for a device AMAPI does not have cannot unenroll the host.
func TestPubSubStatusReportDeviceChangeVerifiedWithAMAPI(t *testing.T) {
	const hostID = uint(11)
	const hostUUID = "VERIFY-DEVICE-CHANGE-HOST"

	newDeviceChange := func(t *testing.T) (*Service, *AndroidMockDS, *android_mock.Client, *fleet.AndroidHost) {
		svc, mockDS, client := newVerifyTestService(t)
		host := wireDedupHost(t, mockDS, hostID, hostUUID)
		host.Device.DeviceID = "current-device"
		mockDS.GetAndroidPubSubDedupStateFunc = func(ctx context.Context, id uint) (string, *time.Time, error) {
			return "", nil, nil
		}
		mockDS.SetAndroidHostEnrolledFunc = func(ctx context.Context, id uint) (bool, error) { return false, nil }
		return svc, mockDS, client, host
	}
	report := func(t *testing.T, deviceName string) (*android.PubSubMessage, androidmanagement.Device) {
		msg := makeStatusEnvelope(t, hostUUID, "report-"+deviceName, "2026-07-22T10:00:00Z", false)
		device := decodePubSubDevice(t, msg)
		device.Name = createAndroidDeviceId(deviceName)
		return pubSubMessageFor(t, android.PubSubStatusReport, device), device
	}

	t.Run("report for the host's current device is not looked up", func(t *testing.T) {
		svc, mockDS, client, _ := newDeviceChange(t)
		msg, _ := report(t, "current-device")
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), dedupToken, msg))
		require.False(t, client.EnterprisesDevicesGetFuncInvoked)
		require.True(t, mockDS.UpdateAndroidHostFuncInvoked)
	})

	t.Run("report for a new device AMAPI has is processed", func(t *testing.T) {
		svc, mockDS, client, _ := newDeviceChange(t)
		msg, device := report(t, "new-device")
		client.EnterprisesDevicesGetFunc = func(ctx context.Context, name string) (*androidmanagement.Device, error) {
			require.Equal(t, device.Name, name)
			return &device, nil
		}
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), dedupToken, msg))
		require.True(t, mockDS.UpdateAndroidHostFuncInvoked)
	})

	t.Run("report for a name AMAPI resolves to another name is dropped", func(t *testing.T) {
		svc, mockDS, client, _ := newDeviceChange(t)
		msg, device := report(t, "current-device?")
		client.EnterprisesDevicesGetFunc = func(ctx context.Context, name string) (*androidmanagement.Device, error) {
			resolved := device
			resolved.Name = createAndroidDeviceId("current-device")
			return &resolved, nil
		}
		requireAckedAsUnverified(t, svc.ProcessPubSubPush(t.Context(), dedupToken, msg))
		require.False(t, mockDS.UpdateAndroidHostFuncInvoked)
	})

	t.Run("report and DELETED for a device AMAPI does not have leave the host enrolled", func(t *testing.T) {
		svc, mockDS, client, host := newDeviceChange(t)
		client.EnterprisesDevicesGetFunc = func(ctx context.Context, name string) (*androidmanagement.Device, error) {
			return nil, amapiNotFound()
		}
		mockDS.UpdateAndroidHostFunc = func(ctx context.Context, h *fleet.AndroidHost, fromEnroll, companyOwned bool) error {
			host.Device.DeviceID = h.Device.DeviceID
			return nil
		}

		msg, _ := report(t, "unknown-device")
		requireAckedAsUnverified(t, svc.ProcessPubSubPush(t.Context(), dedupToken, msg))
		require.False(t, mockDS.UpdateAndroidHostFuncInvoked, "a device AMAPI does not have must not be stored")
		require.Equal(t, "current-device", host.Device.DeviceID)

		deleted := makeStatusEnvelope(t, hostUUID, "delete-unknown-device", "2026-07-22T10:01:00Z", true)
		device := decodePubSubDevice(t, deleted)
		device.Name = createAndroidDeviceId("unknown-device")
		requireAckedAsUnverified(t, svc.ProcessPubSubPush(t.Context(), dedupToken,
			pubSubMessageFor(t, android.PubSubStatusReport, device)))
		require.False(t, mockDS.SetAndroidHostUnenrolledFuncInvoked)
	})
}

func TestPubSubCommandVerifiedWithAMAPI(t *testing.T) {
	const opName = "enterprises/E/devices/D/operations/wipe-1"
	const deviceName = "enterprises/E/devices/D"

	newCommand := func(t *testing.T, commandType android.MDMAndroidCommandType, currentDeviceID string) (*Service, *AndroidMockDS, *android_mock.Client, *string) {
		svc, mockDS, client := newVerifyTestService(t)
		mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: true}}, nil
		}
		mockDS.GetMDMAndroidCommandByOperationNameFunc = func(ctx context.Context, name string) (*android.MDMAndroidCommand, error) {
			return &android.MDMAndroidCommand{
				CommandUUID:   "cmd-wipe",
				HostUUID:      "host-uuid",
				OperationName: opName,
				CommandType:   string(commandType),
				Status:        string(android.MDMAndroidCommandStatusPending),
			}, nil
		}
		mockDS.AndroidHostLiteByHostUUIDFunc = func(ctx context.Context, hostUUID string) (*fleet.AndroidHost, error) {
			return &fleet.AndroidHost{
				Host:   &fleet.Host{ID: 7, UUID: hostUUID},
				Device: &android.Device{HostID: 7, DeviceID: currentDeviceID},
			}, nil
		}
		mockDS.GetHostMDMFunc = func(ctx context.Context, id uint) (*fleet.HostMDM, error) {
			return &fleet.HostMDM{}, nil
		}
		mockDS.SetAndroidHostUnenrolledFunc = func(ctx context.Context, id uint) (bool, error) { return false, nil }
		var status string
		mockDS.UpdateMDMAndroidCommandStatusFunc = func(ctx context.Context, commandUUID, s string, errorCode, errorMessage, rawResult *string) error {
			status = s
			return nil
		}
		return svc, mockDS, client, &status
	}
	commandMsg := func(t *testing.T, op androidmanagement.Operation) *android.PubSubMessage {
		body, err := json.Marshal(op)
		require.NoError(t, err)
		return &android.PubSubMessage{
			Attributes: map[string]string{"notificationType": string(android.PubSubCommand)},
			Data:       base64.StdEncoding.EncodeToString(body),
		}
	}
	notifiedSuccess := androidmanagement.Operation{Name: opName, Done: true}

	t.Run("AMAPI's result is used instead of the notification's", func(t *testing.T) {
		svc, _, client, status := newCommand(t, android.MDMAndroidCommandTypeLock, "D")
		client.EnterprisesDevicesOperationsGetFunc = func(ctx context.Context, name string) (*androidmanagement.Operation, error) {
			require.Equal(t, opName, name)
			return &androidmanagement.Operation{Name: name, Done: true, Error: &androidmanagement.Status{Code: 9, Message: "failed"}}, nil
		}
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), "value", commandMsg(t, notifiedSuccess)))
		require.Equal(t, string(android.MDMAndroidCommandStatusError), *status)
	})

	// AMAPI never marks a WIPE operation done, because the device is erased before it can report
	// back, so a wipe whose device is gone counts as finished.
	for _, tc := range []struct {
		name            string
		commandType     android.MDMAndroidCommandType
		currentDeviceID string
		deviceActive    bool
		wantAcked       bool
	}{
		{"operation not done in AMAPI leaves the command pending", android.MDMAndroidCommandTypeLock, "D", false, false},
		{"wipe not done in AMAPI while its device is active is left pending", android.MDMAndroidCommandTypeWipe, "D", true, false},
		{"wipe not done in AMAPI whose device is gone is acknowledged", android.MDMAndroidCommandTypeWipe, "D", false, true},
		{"wipe not done in AMAPI of a gone device that is not the host's current one is left pending", android.MDMAndroidCommandTypeWipe, "new-device", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, mockDS, client, status := newCommand(t, tc.commandType, tc.currentDeviceID)
			client.EnterprisesDevicesOperationsGetFunc = func(ctx context.Context, name string) (*androidmanagement.Operation, error) {
				return &androidmanagement.Operation{Name: name, Done: false}, nil
			}
			client.EnterprisesDevicesGetFunc = func(ctx context.Context, name string) (*androidmanagement.Device, error) {
				require.Equal(t, deviceName, name)
				if !tc.deviceActive {
					return nil, amapiNotFound()
				}
				return &androidmanagement.Device{Name: name, State: "ACTIVE"}, nil
			}
			require.NoError(t, svc.ProcessPubSubPush(t.Context(), "value", commandMsg(t, notifiedSuccess)))
			require.Equal(t, tc.wantAcked, mockDS.UpdateMDMAndroidCommandStatusFuncInvoked)
			require.Equal(t, tc.wantAcked, mockDS.SetAndroidHostUnenrolledFuncInvoked)
			if tc.wantAcked {
				require.Equal(t, string(android.MDMAndroidCommandStatusAcknowledged), *status)
			}
		})
	}

	t.Run("notification error is not used for a wipe AMAPI reports as not done", func(t *testing.T) {
		svc, _, client, status := newCommand(t, android.MDMAndroidCommandTypeWipe, "D")
		client.EnterprisesDevicesOperationsGetFunc = func(ctx context.Context, name string) (*androidmanagement.Operation, error) {
			return &androidmanagement.Operation{Name: name, Done: false}, nil
		}
		client.EnterprisesDevicesGetFunc = func(ctx context.Context, name string) (*androidmanagement.Device, error) {
			return nil, amapiNotFound()
		}
		notifiedFailure := androidmanagement.Operation{Name: opName, Done: true, Error: &androidmanagement.Status{Code: 9, Message: "failed"}}
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), "value", commandMsg(t, notifiedFailure)))
		require.Equal(t, string(android.MDMAndroidCommandStatusAcknowledged), *status)
	})

	// AMAPI drops operations it has finished with, so a missing operation is not checked unless
	// the result would act on a host it may no longer belong to.
	for _, tc := range []struct {
		name            string
		commandType     android.MDMAndroidCommandType
		currentDeviceID string
		deviceActive    bool
		wantStatus      android.MDMAndroidCommandStatus // empty: the command is left as is
		wantRetry       bool
	}{
		{"lock dropped by AMAPI for an active device uses the notification's result", android.MDMAndroidCommandTypeLock, "D", true, android.MDMAndroidCommandStatusAcknowledged, false},
		{"wipe unknown to AMAPI for an active device is retried", android.MDMAndroidCommandTypeWipe, "D", true, "", true},
		{"operation and device gone from AMAPI use the notification's result", android.MDMAndroidCommandTypeWipe, "D", false, android.MDMAndroidCommandStatusAcknowledged, false},
		{"result for a gone device that is not the host's current one is dropped", android.MDMAndroidCommandTypeWipe, "new-device", false, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, mockDS, client, status := newCommand(t, tc.commandType, tc.currentDeviceID)
			client.EnterprisesDevicesOperationsGetFunc = func(ctx context.Context, name string) (*androidmanagement.Operation, error) {
				return nil, amapiNotFound()
			}
			client.EnterprisesDevicesGetFunc = func(ctx context.Context, name string) (*androidmanagement.Device, error) {
				require.Equal(t, deviceName, name)
				if !tc.deviceActive {
					return nil, amapiNotFound()
				}
				return &androidmanagement.Device{Name: name, State: "ACTIVE"}, nil
			}
			err := svc.ProcessPubSubPush(t.Context(), "value", commandMsg(t, notifiedSuccess))
			if tc.wantRetry {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantStatus != "", mockDS.UpdateMDMAndroidCommandStatusFuncInvoked)
			if tc.wantStatus != "" {
				require.Equal(t, string(tc.wantStatus), *status)
			}
			require.Equal(t, tc.commandType == android.MDMAndroidCommandTypeWipe && tc.wantStatus != "", mockDS.SetAndroidHostUnenrolledFuncInvoked)
		})
	}

	t.Run("AMAPI error is retried", func(t *testing.T) {
		svc, mockDS, client, _ := newCommand(t, android.MDMAndroidCommandTypeLock, "D")
		client.EnterprisesDevicesOperationsGetFunc = func(ctx context.Context, name string) (*androidmanagement.Operation, error) {
			return nil, &googleapi.Error{Code: http.StatusInternalServerError}
		}
		require.Error(t, svc.ProcessPubSubPush(t.Context(), "value", commandMsg(t, notifiedSuccess)))
		require.False(t, mockDS.UpdateMDMAndroidCommandStatusFuncInvoked)
	})
}

// TestPubSubAcknowledgedWipeRedelivery covers a COMMAND for a WIPE that is already acknowledged.
// Fleet re-runs the unenroll so a failed first delivery recovers, but an acknowledged WIPE survives
// a re-enrollment, so a redelivery of one issued to an earlier enrollment must not unenroll the host.
func TestPubSubAcknowledgedWipeRedelivery(t *testing.T) {
	const opName = "enterprises/E/devices/old-device/operations/wipe-1"

	for _, tc := range []struct {
		name            string
		currentDeviceID string
		deviceActive    bool
		wantUnenroll    bool
	}{
		{"wipe for the host's current device is re-run", "old-device", false, true},
		{"wipe for an earlier enrollment is ignored", "new-device", false, false},
		{"wipe for a device AMAPI still has active is ignored", "old-device", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, mockDS, client := newVerifyTestService(t)
			client.EnterprisesDevicesGetFunc = func(ctx context.Context, name string) (*androidmanagement.Device, error) {
				require.Equal(t, "enterprises/E/devices/old-device", name)
				if !tc.deviceActive {
					return nil, amapiNotFound()
				}
				return &androidmanagement.Device{Name: name, State: "ACTIVE"}, nil
			}
			mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: true}}, nil
			}
			mockDS.GetMDMAndroidCommandByOperationNameFunc = func(ctx context.Context, name string) (*android.MDMAndroidCommand, error) {
				return &android.MDMAndroidCommand{
					CommandUUID:   "cmd-wipe",
					HostUUID:      "host-uuid",
					OperationName: opName,
					CommandType:   string(android.MDMAndroidCommandTypeWipe),
					Status:        string(android.MDMAndroidCommandStatusAcknowledged),
				}, nil
			}
			mockDS.AndroidHostLiteByHostUUIDFunc = func(ctx context.Context, hostUUID string) (*fleet.AndroidHost, error) {
				return &fleet.AndroidHost{
					Host:   &fleet.Host{ID: 7, UUID: hostUUID},
					Device: &android.Device{HostID: 7, DeviceID: tc.currentDeviceID},
				}, nil
			}
			mockDS.GetHostMDMFunc = func(ctx context.Context, id uint) (*fleet.HostMDM, error) {
				return &fleet.HostMDM{}, nil
			}
			mockDS.SetAndroidHostUnenrolledFunc = func(ctx context.Context, id uint) (bool, error) { return false, nil }

			body, err := json.Marshal(androidmanagement.Operation{Name: opName, Done: true})
			require.NoError(t, err)
			require.NoError(t, svc.ProcessPubSubPush(t.Context(), "value", &android.PubSubMessage{
				Attributes: map[string]string{"notificationType": string(android.PubSubCommand)},
				Data:       base64.StdEncoding.EncodeToString(body),
			}))
			require.Equal(t, tc.wantUnenroll, mockDS.SetAndroidHostUnenrolledFuncInvoked)
		})
	}
}
