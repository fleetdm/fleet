package recoverykeypassword

import "context"

// mockDatastore implements Datastore for unit testing results_handler.
type mockDatastore struct {
	GetHostIDByVerifyCommandUUIDFunc func(ctx context.Context, verifyCommandUUID string) (uint, error)
	SetRecoveryLockVerifiedFunc      func(ctx context.Context, hostID uint) error
	SetRecoveryLockFailedFunc        func(ctx context.Context, hostID uint, errorMsg string) error

	// Track invocations
	GetHostIDByVerifyCommandUUIDFuncInvoked bool
	SetRecoveryLockVerifiedFuncInvoked      bool
	SetRecoveryLockFailedFuncInvoked        bool
	LastFailedErrorMsg                      string
}

func (m *mockDatastore) GetHostsForRecoveryLockAction(ctx context.Context) ([]HostNeedingRecoveryLock, error) {
	return nil, nil
}

func (m *mockDatastore) SetHostRecoveryKeyPassword(ctx context.Context, hostID uint) (string, error) {
	return "", nil
}

func (m *mockDatastore) GetHostRecoveryKeyPassword(ctx context.Context, hostID uint) (*HostRecoveryKeyPassword, error) {
	return nil, nil
}

func (m *mockDatastore) SetRecoveryLockPending(ctx context.Context, hostID uint, setCommandUUID string) error {
	return nil
}

func (m *mockDatastore) SetRecoveryLockVerifying(ctx context.Context, hostID uint, verifyCommandUUID string) error {
	return nil
}

func (m *mockDatastore) SetRecoveryLockVerified(ctx context.Context, hostID uint) error {
	m.SetRecoveryLockVerifiedFuncInvoked = true
	if m.SetRecoveryLockVerifiedFunc != nil {
		return m.SetRecoveryLockVerifiedFunc(ctx, hostID)
	}
	return nil
}

func (m *mockDatastore) SetRecoveryLockFailed(ctx context.Context, hostID uint, errorMsg string) error {
	m.SetRecoveryLockFailedFuncInvoked = true
	m.LastFailedErrorMsg = errorMsg
	if m.SetRecoveryLockFailedFunc != nil {
		return m.SetRecoveryLockFailedFunc(ctx, hostID, errorMsg)
	}
	return nil
}

func (m *mockDatastore) GetPendingRecoveryLockHosts(ctx context.Context) ([]HostPendingRecoveryLock, error) {
	return nil, nil
}

func (m *mockDatastore) GetHostIDByVerifyCommandUUID(ctx context.Context, verifyCommandUUID string) (uint, error) {
	m.GetHostIDByVerifyCommandUUIDFuncInvoked = true
	if m.GetHostIDByVerifyCommandUUIDFunc != nil {
		return m.GetHostIDByVerifyCommandUUIDFunc(ctx, verifyCommandUUID)
	}
	return 0, nil
}
