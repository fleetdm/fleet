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
		_ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
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
		// Store the first command as pending
		if !commandCreated {
			pendingCommand = cmd
			pendingPIN = pin
			commandCreated = true
		}
		return nil
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

	// Only one command should have been enqueued
	require.Equal(t, 1, enqueueCalls, "Only one lock command should be enqueued")

	// GetPendingLockCommand should have been called multiple times
	require.Greater(t, getPendingCalls, 1, "GetPendingLockCommand should be called multiple times")
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
