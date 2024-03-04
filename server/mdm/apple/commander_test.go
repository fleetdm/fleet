package apple_mdm

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/log/stdlogfmt"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	nanomdm_pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
	"github.com/fleetdm/fleet/v4/server/mock"
	svcmock "github.com/fleetdm/fleet/v4/server/service/mock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMDMAppleCommander(t *testing.T) {
	ctx := context.Background()
	mdmStorage := &mock.MDMAppleStore{}
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

	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
		require.NotNil(t, cmd)
		require.Equal(t, cmd.Command.RequestType, "InstallProfile")
		require.Contains(t, string(cmd.Raw), base64.StdEncoding.EncodeToString(mc))
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

	cmdUUID := uuid.New().String()
	err := cmdr.InstallProfile(ctx, hostUUIDs, mc, cmdUUID)
	require.NoError(t, err)
	require.True(t, mdmStorage.EnqueueCommandFuncInvoked)
	mdmStorage.EnqueueCommandFuncInvoked = false
	require.True(t, mdmStorage.RetrievePushInfoFuncInvoked)
	mdmStorage.RetrievePushInfoFuncInvoked = false

	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
		require.NotNil(t, cmd)
		require.Equal(t, "RemoveProfile", cmd.Command.RequestType)
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
	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
		require.NotNil(t, cmd)
		require.Equal(t, "InstallEnterpriseApplication", cmd.Command.RequestType)
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
	err = cmdr.DeviceLock(ctx, host, cmdUUID)
	require.NoError(t, err)
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

func newMockAPNSPushProviderFactory() (*svcmock.APNSPushProviderFactory, *svcmock.APNSPushProvider) {
	provider := &svcmock.APNSPushProvider{}
	provider.PushFunc = mockSuccessfulPush
	factory := &svcmock.APNSPushProviderFactory{}
	factory.NewPushProviderFunc = func(*tls.Certificate) (push.PushProvider, error) {
		return provider, nil
	}

	return factory, provider
}

func mockSuccessfulPush(pushes []*mdm.Push) (map[string]*push.Response, error) {
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
