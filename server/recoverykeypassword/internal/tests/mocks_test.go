package tests

import "context"

// mockCommander implements recoverykeypassword.MDMCommander for testing.
// This mocks the external MDM/APNs infrastructure.
type mockCommander struct {
	EnqueueCommandFunc     func(ctx context.Context, hostUUIDs []string, rawCommand string) error
	SendNotificationsFunc  func(ctx context.Context, hostUUIDs []string) error
	EnqueueCommandCalls    []enqueueCommandCall
	SendNotificationsCalls [][]string
}

type enqueueCommandCall struct {
	HostUUIDs  []string
	RawCommand string
}

func (m *mockCommander) EnqueueCommand(ctx context.Context, hostUUIDs []string, rawCommand string) error {
	m.EnqueueCommandCalls = append(m.EnqueueCommandCalls, enqueueCommandCall{
		HostUUIDs:  hostUUIDs,
		RawCommand: rawCommand,
	})
	if m.EnqueueCommandFunc != nil {
		return m.EnqueueCommandFunc(ctx, hostUUIDs, rawCommand)
	}
	return nil
}

func (m *mockCommander) SendNotifications(ctx context.Context, hostUUIDs []string) error {
	m.SendNotificationsCalls = append(m.SendNotificationsCalls, hostUUIDs)
	if m.SendNotificationsFunc != nil {
		return m.SendNotificationsFunc(ctx, hostUUIDs)
	}
	return nil
}
