package service

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/go-json-experiment/json"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

const dedupToken = "value"

// wireDedupHost configures mockDS so both the ENROLLMENT (re-enroll) and
// STATUS_REPORT full-processing paths succeed for a single existing host, and
// returns that host. Individual tests override GetAndroidPubSubDedupStateFunc and
// the invocation flags they assert on.
func wireDedupHost(t *testing.T, mockDS *AndroidMockDS, hostID uint, hostUUID string) *fleet.AndroidHost {
	t.Helper()
	host := &fleet.AndroidHost{
		Host: &fleet.Host{ID: hostID, UUID: hostUUID},
		Device: &android.Device{
			HostID:               hostID,
			DeviceID:             "existing-device",
			EnterpriseSpecificID: &hostUUID,
		},
	}
	mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: true}}, nil
	}
	mockDS.AndroidHostLiteFunc = func(ctx context.Context, esID string) (*fleet.AndroidHost, error) {
		return host, nil
	}
	mockDS.UpdateAndroidHostFunc = func(ctx context.Context, h *fleet.AndroidHost, fromEnroll, companyOwned bool) error {
		return nil
	}
	mockDS.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
		return &fleet.EnrollSecret{}, nil
	}
	mockDS.DeleteAllHostCertificateTemplatesFunc = func(ctx context.Context, hostUUID string) error { return nil }
	mockDS.ClearHostMDMActionsFunc = func(ctx context.Context, id uint) error { return nil }
	mockDS.ScimUserByHostIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
		return nil, common_mysql.NotFound("scim user")
	}
	mockDS.ListHostDeviceMappingFunc = func(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
		return nil, nil
	}
	// STATUS_REPORT DELETED path.
	mockDS.SetAndroidHostUnenrolledFunc = func(ctx context.Context, id uint) (bool, error) { return true, nil }
	mockDS.GetHostMDMFunc = func(ctx context.Context, id uint) (*fleet.HostMDM, error) {
		return &fleet.HostMDM{IsPersonalEnrollment: true}, nil
	}
	mockDS.MarkAllPendingVPPInstallsAsFailedForAndroidHostFunc = func(ctx context.Context, id uint) ([]*fleet.User, []fleet.ActivityDetails, error) {
		return nil, nil, nil
	}
	mockDS.ListHostsLiteByIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.Host, error) {
		return []*fleet.Host{{ID: hostID}}, nil
	}
	return host
}

// makeEnrollmentEnvelope builds an ENROLLMENT PubSub message for a fixed device
// with the given Google envelope messageId/publishTime.
func makeEnrollmentEnvelope(t *testing.T, esID, messageID, publishTime string) *android.PubSubMessage {
	msg := createEnrollmentMessage(t, androidmanagement.Device{
		Name:                createAndroidDeviceId("dedup"),
		EnrollmentTokenData: `{"enroll_secret":"global"}`,
	})
	msg.MessageID = messageID
	msg.PublishTime = publishTime
	return msg
}

// makeStatusEnvelope builds a STATUS_REPORT PubSub message (optionally in the
// DELETED state) for a fixed device with the given envelope fields.
func makeStatusEnvelope(t *testing.T, esID, messageID, publishTime string, deleted bool) *android.PubSubMessage {
	device := androidmanagement.Device{
		Name: createAndroidDeviceId("dedup"),
		HardwareInfo: &androidmanagement.HardwareInfo{
			EnterpriseSpecificId: esID,
			Brand:                "TestBrand",
			Model:                "TestModel",
			SerialNumber:         "test-serial",
			Hardware:             "test-hardware",
		},
		SoftwareInfo: &androidmanagement.SoftwareInfo{AndroidBuildNumber: "test-build", AndroidVersion: "1"},
		MemoryInfo:   &androidmanagement.MemoryInfo{TotalRam: 8 * 1024 * 1024 * 1024},
	}
	if deleted {
		device.AppliedState = string(android.DeviceStateDeleted)
	}
	data, err := json.Marshal(device)
	require.NoError(t, err)
	return &android.PubSubMessage{
		Attributes:  map[string]string{"notificationType": string(android.PubSubStatusReport)},
		Data:        base64.StdEncoding.EncodeToString(data),
		MessageID:   messageID,
		PublishTime: publishTime,
	}
}

func TestPubSubDedupAndStaleness(t *testing.T) {
	const hostID = uint(10)
	const hostUUID = "DEDUP-HOST-UUID"

	t.Run("duplicate ENROLLMENT messageId is a no-op", func(t *testing.T) {
		svc, mockDS := createAndroidService(t)
		wireDedupHost(t, mockDS, hostID, hostUUID)
		mockDS.GetAndroidPubSubDedupStateFunc = func(ctx context.Context, id uint) (string, *time.Time, error) {
			return "msg-dup", nil, nil
		}

		msg := makeEnrollmentEnvelope(t, hostUUID, "msg-dup", "2026-07-22T10:00:00Z")
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), dedupToken, msg))

		require.False(t, mockDS.UpdateAndroidHostFuncInvoked, "duplicate enrollment must not re-run updateHost")
		require.False(t, mockDS.NewJobFuncInvoked, "duplicate enrollment must not re-queue setup experience")
		require.False(t, mockDS.SetAndroidPubSubDedupStateFuncInvoked, "no state should be recorded on a skipped message")
	})

	t.Run("stale ENROLLMENT event time is skipped", func(t *testing.T) {
		svc, mockDS := createAndroidService(t)
		wireDedupHost(t, mockDS, hostID, hostUUID)
		stored := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
		mockDS.GetAndroidPubSubDedupStateFunc = func(ctx context.Context, id uint) (string, *time.Time, error) {
			return "other-msg", &stored, nil
		}

		// publishTime older than the stored event time -> stale.
		msg := makeEnrollmentEnvelope(t, hostUUID, "msg-new", "2020-01-01T00:00:00Z")
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), dedupToken, msg))

		require.False(t, mockDS.UpdateAndroidHostFuncInvoked, "stale enrollment must not re-run updateHost")
		require.False(t, mockDS.NewJobFuncInvoked, "stale enrollment must not re-queue setup experience")
	})

	t.Run("re-enrollment records dedup state", func(t *testing.T) {
		svc, mockDS := createAndroidService(t)
		wireDedupHost(t, mockDS, hostID, hostUUID)
		mockDS.GetAndroidPubSubDedupStateFunc = func(ctx context.Context, id uint) (string, *time.Time, error) {
			return "older-msg", nil, nil
		}
		var recordedID string
		var recordedHostID uint
		mockDS.SetAndroidPubSubDedupStateFunc = func(ctx context.Context, id uint, messageID string, eventTime *time.Time) error {
			recordedHostID = id
			recordedID = messageID
			return nil
		}

		msg := makeEnrollmentEnvelope(t, hostUUID, "msg-reenroll", "2026-07-22T10:00:00Z")
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), dedupToken, msg))

		require.True(t, mockDS.UpdateAndroidHostFuncInvoked, "a non-duplicate re-enrollment must run updateHost")
		require.True(t, mockDS.SetAndroidPubSubDedupStateFuncInvoked, "successful enrollment must record dedup state")
		require.Equal(t, "msg-reenroll", recordedID)
		require.Equal(t, hostID, recordedHostID)
	})

	t.Run("duplicate STATUS_REPORT messageId is a no-op", func(t *testing.T) {
		svc, mockDS := createAndroidService(t)
		wireDedupHost(t, mockDS, hostID, hostUUID)
		mockDS.GetAndroidPubSubDedupStateFunc = func(ctx context.Context, id uint) (string, *time.Time, error) {
			return "msg-dup", nil, nil
		}

		msg := makeStatusEnvelope(t, hostUUID, "msg-dup", "2026-07-22T10:00:00Z", false)
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), dedupToken, msg))

		require.False(t, mockDS.UpdateAndroidHostFuncInvoked, "duplicate status report must not re-run updateHost")
		require.False(t, mockDS.SetAndroidHostEnrolledFuncInvoked, "duplicate status report must not touch enrollment")
	})

	t.Run("out-of-order DELETED is skipped by staleness", func(t *testing.T) {
		svc, mockDS := createAndroidService(t)
		wireDedupHost(t, mockDS, hostID, hostUUID)
		// A more-recent re-enrollment was already processed.
		stored := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
		mockDS.GetAndroidPubSubDedupStateFunc = func(ctx context.Context, id uint) (string, *time.Time, error) {
			return "recent-enroll-msg", &stored, nil
		}

		// A stale DELETED redelivered out of order (older publishTime).
		msg := makeStatusEnvelope(t, hostUUID, "stale-delete-msg", "2020-01-01T00:00:00Z", true)
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), dedupToken, msg))

		require.False(t, mockDS.SetAndroidHostUnenrolledFuncInvoked, "a stale DELETED must not unenroll a live host")
	})

	t.Run("STATUS_REPORT recovers a wrongly-unenrolled host", func(t *testing.T) {
		svc, mockDS := createAndroidService(t)
		wireDedupHost(t, mockDS, hostID, hostUUID)
		mockDS.GetAndroidPubSubDedupStateFunc = func(ctx context.Context, id uint) (string, *time.Time, error) {
			return "", nil, nil // no prior state -> processed normally
		}
		var enrolledHostID uint
		mockDS.SetAndroidHostEnrolledFunc = func(ctx context.Context, id uint) (bool, error) {
			enrolledHostID = id
			return true, nil
		}

		msg := makeStatusEnvelope(t, hostUUID, "live-msg", "2026-07-22T10:00:00Z", false)
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), dedupToken, msg))

		require.True(t, mockDS.UpdateAndroidHostFuncInvoked, "a live status report must run updateHost")
		require.True(t, mockDS.SetAndroidHostEnrolledFuncInvoked, "a live status report must attempt enrollment recovery")
		require.Equal(t, hostID, enrolledHostID)
		require.True(t, mockDS.SetAndroidPubSubDedupStateFuncInvoked, "successful status report must record dedup state")
	})

	t.Run("stale STATUS_REPORT does not trigger enrollment recovery", func(t *testing.T) {
		// Mirror of the bug being fixed: a stale STATUS_REPORT (older than the last
		// processed event, e.g. one published before a legitimate unenroll) must not
		// re-enroll the host. Staleness must short-circuit before SetAndroidHostEnrolled.
		svc, mockDS := createAndroidService(t)
		wireDedupHost(t, mockDS, hostID, hostUUID)
		stored := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
		mockDS.GetAndroidPubSubDedupStateFunc = func(ctx context.Context, id uint) (string, *time.Time, error) {
			return "unenroll-msg", &stored, nil
		}

		msg := makeStatusEnvelope(t, hostUUID, "stale-report-msg", "2020-01-01T00:00:00Z", false)
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), dedupToken, msg))

		require.False(t, mockDS.UpdateAndroidHostFuncInvoked, "stale status report must not run updateHost")
		require.False(t, mockDS.SetAndroidHostEnrolledFuncInvoked, "stale status report must not re-enroll the host")
	})

	t.Run("equal event time with a different messageId is processed", func(t *testing.T) {
		// A distinct message with the same timestamp is not a duplicate and not stale
		// (staleness is strict "older than"), so it must be processed.
		svc, mockDS := createAndroidService(t)
		wireDedupHost(t, mockDS, hostID, hostUUID)
		sameTime := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
		mockDS.GetAndroidPubSubDedupStateFunc = func(ctx context.Context, id uint) (string, *time.Time, error) {
			return "stored-msg", &sameTime, nil
		}

		msg := makeStatusEnvelope(t, hostUUID, "different-msg", "2026-07-22T10:00:00Z", false)
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), dedupToken, msg))

		require.True(t, mockDS.UpdateAndroidHostFuncInvoked, "a distinct, non-stale message must be processed")
	})

	t.Run("WIPE ack records dedup state to block a later stale STATUS_REPORT", func(t *testing.T) {
		svc, mockDS := createAndroidService(t)
		mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: true}}, nil
		}
		stored := &android.MDMAndroidCommand{
			CommandUUID:   "cmd-wipe",
			HostUUID:      hostUUID,
			OperationName: "enterprises/E/devices/D/operations/wipe-ack",
			CommandType:   string(android.MDMAndroidCommandTypeWipe),
			Status:        string(android.MDMAndroidCommandStatusPending),
		}
		mockDS.GetMDMAndroidCommandByOperationNameFunc = func(ctx context.Context, opName string) (*android.MDMAndroidCommand, error) {
			return stored, nil
		}
		mockDS.UpdateMDMAndroidCommandStatusFunc = func(ctx context.Context, commandUUID, status string, errorCode, errorMessage *string) error {
			return nil
		}
		mockDS.AndroidHostLiteByHostUUIDFunc = func(ctx context.Context, hUUID string) (*fleet.AndroidHost, error) {
			return &fleet.AndroidHost{Host: &fleet.Host{ID: hostID, UUID: hUUID}}, nil
		}
		mockDS.GetHostMDMFunc = func(ctx context.Context, id uint) (*fleet.HostMDM, error) {
			return &fleet.HostMDM{IsPersonalEnrollment: false}, nil
		}
		mockDS.ClearHostMDMActionsFunc = func(ctx context.Context, id uint) error { return nil }
		mockDS.SetAndroidHostUnenrolledFunc = func(ctx context.Context, id uint) (bool, error) { return true, nil }
		mockDS.ListHostsLiteByIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.Host, error) {
			return []*fleet.Host{{ID: hostID}}, nil
		}
		var recordedID string
		var recordedTime *time.Time
		mockDS.SetAndroidPubSubDedupStateFunc = func(ctx context.Context, id uint, messageID string, eventTime *time.Time) error {
			recordedID = messageID
			recordedTime = eventTime
			return nil
		}

		body, err := json.Marshal(androidmanagement.Operation{Name: stored.OperationName, Done: true})
		require.NoError(t, err)
		msg := &android.PubSubMessage{
			Attributes:  map[string]string{"notificationType": string(android.PubSubCommand)},
			Data:        base64.StdEncoding.EncodeToString(body),
			MessageID:   "wipe-msg",
			PublishTime: "2026-07-22T12:00:00Z",
		}
		require.NoError(t, svc.ProcessPubSubPush(t.Context(), dedupToken, msg))

		require.True(t, mockDS.SetAndroidPubSubDedupStateFuncInvoked, "WIPE ack unenroll must record dedup state")
		require.Equal(t, "wipe-msg", recordedID)
		require.NotNil(t, recordedTime, "WIPE ack must record the notification publish time as the event time")
		require.Equal(t, time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC), recordedTime.UTC())
	})
}
