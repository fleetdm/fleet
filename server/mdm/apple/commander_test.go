package apple_mdm

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	nanomdm_pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
	mdmmock "github.com/fleetdm/fleet/v4/server/mock/mdm"
	svcmock "github.com/fleetdm/fleet/v4/server/service/mock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/nanolib/log/stdlogfmt"
	"github.com/micromdm/plist"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/require"
)

// mockConflictError is used in tests to simulate a conflict error
type mockConflictError struct {
	msg string
}

func (e *mockConflictError) Error() string {
	return e.msg
}

func (e *mockConflictError) IsConflict() bool {
	return true
}

func TestMDMAppleCommander(t *testing.T) {
	ctx := context.Background()
	mdmStorage := &mdmmock.MDMAppleStore{}
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		stdlogfmt.New(),
	)
	cmdr := NewMDMAppleCommander(mdmStorage, pusher)

	// TODO(roberto): there's a data race in the mock when more
	// than one host ID is provided because the pusher uses one
	// goroutine per uuid to send the commands
	hostUUIDs := []string{"A"}
	payloadName := "com.foo.bar"
	payloadIdentifier := "com-foo-bar"
	mc := mobileconfigForTest(payloadName, payloadIdentifier)

	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
		require.NotNil(t, cmd)
		require.Equal(t, cmd.Command.Command.RequestType, "InstallProfile")
		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
		p7, err := pkcs7.Parse(fullCmd.Command.InstallProfile.Payload)
		require.NoError(t, err)
		require.Equal(t, string(p7.Content), string(mc))
		return nil, nil
	}

	mdmStorage.RetrievePushInfoFunc = func(p0 context.Context, targetUUIDs []string) (map[string]*mdm.Push, error) {
		require.ElementsMatch(t, hostUUIDs, targetUUIDs)
		pushes := make(map[string]*mdm.Push, len(targetUUIDs))
		for _, uuid := range targetUUIDs {
			pushes[uuid] = &mdm.Push{
				PushMagic: "magic" + uuid,
				Token:     []byte("token" + uuid),
				Topic:     "topic" + uuid,
			}
		}

		return pushes, nil
	}

	mdmStorage.RetrievePushCertFunc = func(ctx context.Context, topic string) (*tls.Certificate, string, error) {
		cert, err := tls.LoadX509KeyPair("../../service/testdata/server.pem", "../../service/testdata/server.key")
		return &cert, "", err
	}
	mdmStorage.IsPushCertStaleFunc = func(ctx context.Context, topic string, staleToken string) (bool, error) {
		return false, nil
	}
	mdmStorage.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		certPEM, err := os.ReadFile("../../service/testdata/server.pem")
		require.NoError(t, err)
		keyPEM, err := os.ReadFile("../../service/testdata/server.key")
		require.NoError(t, err)
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert: {Value: certPEM},
			fleet.MDMAssetCAKey:  {Value: keyPEM},
		}, nil
	}

	cmdUUID := uuid.New().String()
	err := cmdr.InstallProfile(ctx, hostUUIDs, mc, cmdUUID)
	require.NoError(t, err)
	require.True(t, mdmStorage.EnqueueCommandFuncInvoked)
	mdmStorage.EnqueueCommandFuncInvoked = false
	require.True(t, mdmStorage.RetrievePushInfoFuncInvoked)
	mdmStorage.RetrievePushInfoFuncInvoked = false

	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
		require.NotNil(t, cmd)
		require.Equal(t, "RemoveProfile", cmd.Command.Command.RequestType)
		require.Contains(t, string(cmd.Raw), payloadIdentifier)
		return nil, nil
	}
	cmdUUID = uuid.New().String()
	err = cmdr.RemoveProfile(ctx, hostUUIDs, payloadIdentifier, cmdUUID)
	require.True(t, mdmStorage.EnqueueCommandFuncInvoked)
	mdmStorage.EnqueueCommandFuncInvoked = false
	require.True(t, mdmStorage.RetrievePushInfoFuncInvoked)
	mdmStorage.RetrievePushInfoFuncInvoked = false
	require.NoError(t, err)

	cmdUUID = uuid.New().String()
	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
		require.NotNil(t, cmd)
		require.Equal(t, "InstallEnterpriseApplication", cmd.Command.Command.RequestType)
		require.Contains(t, string(cmd.Raw), "http://test.example.com")
		require.Contains(t, string(cmd.Raw), cmdUUID)
		return nil, nil
	}
	err = cmdr.InstallEnterpriseApplication(ctx, hostUUIDs, "http://test.example.com", cmdUUID)
	require.NoError(t, err)
	require.True(t, mdmStorage.EnqueueCommandFuncInvoked)
	mdmStorage.EnqueueCommandFuncInvoked = false
	require.True(t, mdmStorage.RetrievePushInfoFuncInvoked)
	mdmStorage.RetrievePushInfoFuncInvoked = false

	host := &fleet.Host{ID: 1, UUID: "A", Platform: "darwin"}
	cmdUUID = uuid.New().String()

	// Mock GetPendingLockCommand to return nil (no pending command)
	mdmStorage.GetPendingLockCommandFunc = func(ctx context.Context, hostUUID string) (*mdm.Command, string, error) {
		return nil, "", nil
	}

	mdmStorage.EnqueueDeviceLockCommandFunc = func(ctx context.Context, gotHost *fleet.Host, cmd *mdm.Command, pin string) error {
		require.NotNil(t, gotHost)
		require.Equal(t, host.ID, gotHost.ID)
		require.Equal(t, host.UUID, gotHost.UUID)
		require.Equal(t, "DeviceLock", cmd.Command.RequestType)
		require.Contains(t, string(cmd.Raw), cmdUUID)
		require.Len(t, pin, 6)
		return nil
	}
	pin, err := cmdr.DeviceLock(ctx, host, cmdUUID)
	require.NoError(t, err)
	require.Len(t, pin, 6)
	require.True(t, mdmStorage.EnqueueDeviceLockCommandFuncInvoked)
	mdmStorage.EnqueueDeviceLockCommandFuncInvoked = false
	require.True(t, mdmStorage.RetrievePushInfoFuncInvoked)
	mdmStorage.RetrievePushInfoFuncInvoked = false

	mdmStorage.EnqueueDeviceUnlockCommandFunc = func(ctx context.Context, gotHost *fleet.Host, cmd *mdm.Command) error {
		require.NotNil(t, gotHost)
		require.Equal(t, host.ID, gotHost.ID)
		require.Equal(t, host.UUID, gotHost.UUID)
		require.Equal(t, "DisableLostMode", cmd.Command.RequestType)
		require.Contains(t, string(cmd.Raw), cmdUUID)
		return nil
	}
	err = cmdr.DisableLostMode(ctx, host, cmdUUID)
	require.NoError(t, err)
	require.True(t, mdmStorage.EnqueueDeviceUnlockCommandFuncInvoked)
	mdmStorage.EnqueueDeviceUnlockCommandFuncInvoked = false
	require.True(t, mdmStorage.RetrievePushInfoFuncInvoked)
	mdmStorage.RetrievePushInfoFuncInvoked = false

	cmdUUID = uuid.New().String()
	mdmStorage.EnqueueDeviceWipeCommandFunc = func(ctx context.Context, gotHost *fleet.Host, cmd *mdm.Command) error {
		require.NotNil(t, gotHost)
		require.Equal(t, host.ID, gotHost.ID)
		require.Equal(t, host.UUID, gotHost.UUID)
		require.Equal(t, "EraseDevice", cmd.Command.RequestType)
		require.Contains(t, string(cmd.Raw), cmdUUID)
		return nil
	}
	err = cmdr.EraseDevice(ctx, host, cmdUUID)
	require.NoError(t, err)
	require.True(t, mdmStorage.EnqueueDeviceWipeCommandFuncInvoked)
	mdmStorage.EnqueueDeviceWipeCommandFuncInvoked = false
	require.True(t, mdmStorage.RetrievePushInfoFuncInvoked)
	mdmStorage.RetrievePushInfoFuncInvoked = false
}

func TestMDMAppleCommanderConcurrentDeviceLock(t *testing.T) {
	ctx := context.Background()
	mdmStorage := &mdmmock.MDMAppleStore{}
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		stdlogfmt.New(),
	)
	cmdr := NewMDMAppleCommander(mdmStorage, pusher)

	host := &fleet.Host{ID: 1, UUID: "TEST-HOST", Platform: "darwin"}

	// Variables to track calls (with mutex for thread safety)
	var mu sync.Mutex
	var pendingCommand *mdm.Command
	var pendingPIN string
	enqueueCalls := 0
	getPendingCalls := 0

	// Mock GetPendingLockCommand
	// Need to track state across concurrent calls
	var commandCreated bool
	mdmStorage.GetPendingLockCommandFunc = func(ctx context.Context, hostUUID string) (*mdm.Command, string, error) {
		mu.Lock()
		defer mu.Unlock()
		getPendingCalls++
		require.Equal(t, host.UUID, hostUUID)
		// After the first command is enqueued, return it as pending
		if commandCreated && pendingCommand != nil {
			return pendingCommand, pendingPIN, nil
		}
		return nil, "", nil
	}

	// Mock EnqueueDeviceLockCommand
	mdmStorage.EnqueueDeviceLockCommandFunc = func(ctx context.Context, gotHost *fleet.Host, cmd *mdm.Command, pin string) error {
		mu.Lock()
		defer mu.Unlock()
		enqueueCalls++
		require.NotNil(t, gotHost)
		require.Equal(t, host.ID, gotHost.ID)
		require.Equal(t, host.UUID, gotHost.UUID)
		require.Equal(t, "DeviceLock", cmd.Command.RequestType)
		require.Len(t, pin, 6)
		// Store the first command as pending, reject others with conflict
		if !commandCreated {
			pendingCommand = cmd
			pendingPIN = pin
			commandCreated = true
			return nil
		}
		// Command already exists, return conflict error
		return &mockConflictError{msg: "host already has a pending lock command"}
	}

	// Mock RetrievePushInfo
	mdmStorage.RetrievePushInfoFunc = func(ctx context.Context, tokens []string) (map[string]*mdm.Push, error) {
		res := make(map[string]*mdm.Push)
		for _, token := range tokens {
			res[token] = &mdm.Push{
				PushMagic: "magic",
				Token:     []byte("token"),
				Topic:     "topic",
			}
		}
		return res, nil
	}

	// Mock RetrievePushCert
	mdmStorage.RetrievePushCertFunc = func(ctx context.Context, topic string) (*tls.Certificate, string, error) {
		// Return a mock certificate
		return &tls.Certificate{}, "staleToken", nil
	}

	// Mock IsPushCertStale - return false (cert is not stale)
	mdmStorage.IsPushCertStaleFunc = func(ctx context.Context, topic string, staleToken string) (bool, error) {
		return false, nil
	}

	// Simulate concurrent lock requests
	numGoroutines := 10
	results := make(chan string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			cmdUUID := fmt.Sprintf("cmd-uuid-%d", idx)
			pin, err := cmdr.DeviceLock(ctx, host, cmdUUID)
			if err != nil {
				errors <- err
			} else {
				results <- pin
			}
		}(i)
	}

	// Collect results
	var pins []string
	for i := 0; i < numGoroutines; i++ {
		select {
		case pin := <-results:
			pins = append(pins, pin)
		case err := <-errors:
			require.NoError(t, err)
		}
	}

	// Verify results
	require.Len(t, pins, numGoroutines, "All requests should succeed")

	// All PINs should be the same
	firstPIN := pins[0]
	for _, pin := range pins {
		require.Equal(t, firstPIN, pin, "All requests should return the same PIN")
	}

	// Due to race conditions, multiple goroutines may attempt to enqueue
	// but only one should succeed, the rest should get conflict errors.
	// The important thing is that all requests return the same PIN
	require.GreaterOrEqual(t, enqueueCalls, 1, "At least one enqueue attempt should be made")
	require.LessOrEqual(t, enqueueCalls, numGoroutines, "At most numGoroutines enqueue attempts")

	// GetPendingLockCommand should have been called multiple times
	// This includes both initial checks and post-conflict checks
	require.GreaterOrEqual(t, getPendingCalls, numGoroutines, "GetPendingLockCommand should be called at least once per request")
}

func TestMDMAppleCommanderDeviceLockPushNotificationFailure(t *testing.T) {
	ctx := context.Background()
	mdmStorage := &mdmmock.MDMAppleStore{}

	// Create a mock push provider that will fail
	pushProvider := &svcmock.APNSPushProvider{}
	pushFactory := &svcmock.APNSPushProviderFactory{}
	pushFactory.NewPushProviderFunc = func(*tls.Certificate) (push.PushProvider, error) {
		return pushProvider, nil
	}

	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		stdlogfmt.New(),
	)
	cmdr := NewMDMAppleCommander(mdmStorage, pusher)

	host := &fleet.Host{ID: 1, UUID: "TEST-HOST-PUSH-FAIL", Platform: "darwin"}

	// Track whether we're on the first or second request
	var requestCount int
	var existingCommand *mdm.Command
	var existingPIN string

	// Mock GetPendingLockCommand
	mdmStorage.GetPendingLockCommandFunc = func(ctx context.Context, hostUUID string) (*mdm.Command, string, error) {
		requestCount++
		require.Equal(t, host.UUID, hostUUID)

		switch requestCount {
		case 1:
			// First request - no pending command
			return nil, "", nil
		case 2:
			// Second request initial check - still no pending command
			// (hasn't been created yet)
			return nil, "", nil
		case 3:
			// Second request after conflict - return the existing command
			return existingCommand, existingPIN, nil
		default:
			t.Fatalf("Unexpected call to GetPendingLockCommand: %d", requestCount)
			return nil, "", nil
		}
	}

	// Mock EnqueueDeviceLockCommand
	var enqueueCalls int
	mdmStorage.EnqueueDeviceLockCommandFunc = func(ctx context.Context, gotHost *fleet.Host, cmd *mdm.Command, pin string) error {
		enqueueCalls++
		require.NotNil(t, gotHost)
		require.Equal(t, host.ID, gotHost.ID)
		require.Equal(t, "DeviceLock", cmd.Command.RequestType)

		switch enqueueCalls {
		case 1:
			// First request succeeds
			existingCommand = cmd
			existingPIN = pin
			return nil
		case 2:
			// Second request gets conflict
			return &mockConflictError{msg: "host already has a pending lock command"}
		default:
			t.Fatalf("Unexpected call to EnqueueDeviceLockCommand: %d", enqueueCalls)
			return nil
		}
	}

	// Mock RetrievePushInfo
	mdmStorage.RetrievePushInfoFunc = func(ctx context.Context, tokens []string) (map[string]*mdm.Push, error) {
		res := make(map[string]*mdm.Push)
		for _, token := range tokens {
			res[token] = &mdm.Push{
				PushMagic: "magic",
				Token:     []byte("token"),
				Topic:     "topic",
			}
		}
		return res, nil
	}

	// Mock RetrievePushCert
	mdmStorage.RetrievePushCertFunc = func(ctx context.Context, topic string) (*tls.Certificate, string, error) {
		return &tls.Certificate{}, "staleToken", nil
	}

	// Mock IsPushCertStale
	mdmStorage.IsPushCertStaleFunc = func(ctx context.Context, topic string, staleToken string) (bool, error) {
		return false, nil
	}

	// Configure push provider to fail on conflict scenario
	var pushAttempts int
	pushProvider.PushFunc = func(ctx context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		pushAttempts++

		switch pushAttempts {
		case 1:
			// First request - push succeeds
			return mockSuccessfulPush(ctx, pushes)
		case 2:
			// Second request during conflict handling - push fails
			// This simulates a network error or push service issue
			return nil, errors.New("push notification service unavailable")
		default:
			t.Fatalf("Unexpected push attempt: %d", pushAttempts)
			return nil, nil
		}
	}

	// First request - should succeed normally
	pin1, err := cmdr.DeviceLock(ctx, host, "cmd-uuid-1")
	require.NoError(t, err)
	require.NotEmpty(t, pin1)
	require.Len(t, pin1, 6)

	// Reset request count for second request
	requestCount = 1

	// Second concurrent request - should get conflict but still return PIN despite push failure
	pin2, err := cmdr.DeviceLock(ctx, host, "cmd-uuid-2")
	require.NoError(t, err, "Should not return error even when push notification fails")
	require.NotEmpty(t, pin2)
	require.Equal(t, pin1, pin2, "Should return the same PIN as first request")

	// Verify the expected number of calls
	require.Equal(t, 2, enqueueCalls, "Should have attempted to enqueue twice")
	require.Equal(t, 2, pushAttempts, "Should have attempted push twice")
	require.Equal(t, 3, requestCount, "Should have called GetPendingLockCommand three times")
}

func newMockAPNSPushProviderFactory() (*svcmock.APNSPushProviderFactory, *svcmock.APNSPushProvider) {
	provider := &svcmock.APNSPushProvider{}
	provider.PushFunc = mockSuccessfulPush
	factory := &svcmock.APNSPushProviderFactory{}
	factory.NewPushProviderFunc = func(*tls.Certificate) (push.PushProvider, error) {
		return provider, nil
	}

	return factory, provider
}

func mockSuccessfulPush(_ context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
	res := make(map[string]*push.Response, len(pushes))
	for _, p := range pushes {
		res[p.Token.String()] = &push.Response{
			Id:  uuid.New().String(),
			Err: nil,
		}
	}
	return res, nil
}

func mobileconfigForTest(name, identifier string) []byte {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array/>
	<key>PayloadDisplayName</key>
	<string>%s</string>
	<key>PayloadIdentifier</key>
	<string>%s</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>%s</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
`, name, identifier, uuid.New().String()))
}

func TestAPNSDeliveryError(t *testing.T) {
	tests := []struct {
		name                string
		errorsByUUID        map[string]error
		expectedError       string
		expectedFailedUUIDs []string
		expectedStatusCode  int
	}{
		{
			name: "single error",
			errorsByUUID: map[string]error{
				"uuid1": errors.New("network error"),
			},
			expectedError: `APNS delivery failed with the following errors:
UUID: uuid1, Error: network error`,
			expectedFailedUUIDs: []string{"uuid1"},
			expectedStatusCode:  http.StatusBadGateway,
		},
		{
			name: "multiple errors, sorted",
			errorsByUUID: map[string]error{
				"uuid3": errors.New("timeout error"),
				"uuid1": errors.New("network error"),
				"uuid2": errors.New("certificate error"),
			},
			expectedError: `APNS delivery failed with the following errors:
UUID: uuid1, Error: network error
UUID: uuid2, Error: certificate error
UUID: uuid3, Error: timeout error`,
			expectedFailedUUIDs: []string{"uuid1", "uuid2", "uuid3"},
			expectedStatusCode:  http.StatusBadGateway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apnsErr := &APNSDeliveryError{
				errorsByUUID: tt.errorsByUUID,
			}

			require.Equal(t, tt.expectedError, apnsErr.Error())
			require.Equal(t, tt.expectedFailedUUIDs, apnsErr.FailedUUIDs())
			require.Equal(t, tt.expectedStatusCode, apnsErr.StatusCode())
		})
	}
}
