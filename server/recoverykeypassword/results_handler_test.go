package recoverykeypassword

import (
	"context"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyRecoveryLockResultsHandler_NonVerifyCommand(t *testing.T) {
	ctx := context.Background()
	mockDS := &mockDatastore{}
	logger := slog.New(slog.DiscardHandler)

	handler := NewVerifyRecoveryLockResultsHandler(mockDS, logger)

	// Command without the verify prefix should be skipped
	result := NewRecoveryLockResult(&mdm.CommandResults{
		CommandUUID: "some-other-command-uuid",
		Status:      fleet.MDMAppleStatusAcknowledged,
	})

	err := handler(ctx, result)
	require.NoError(t, err)
	assert.False(t, mockDS.GetHostIDByVerifyCommandUUIDFuncInvoked)
}

func TestVerifyRecoveryLockResultsHandler_Acknowledged_PasswordVerified(t *testing.T) {
	ctx := context.Background()
	mockDS := &mockDatastore{
		GetHostIDByVerifyCommandUUIDFunc: func(ctx context.Context, verifyCommandUUID string) (uint, error) {
			return 123, nil
		},
		SetRecoveryLockVerifiedFunc: func(ctx context.Context, hostID uint) error {
			return nil
		},
	}
	logger := slog.New(slog.DiscardHandler)

	handler := NewVerifyRecoveryLockResultsHandler(mockDS, logger)

	// Plist response with PasswordVerified=true
	rawResponse := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CommandUUID</key>
	<string>VERIFY-RECOVERY-LOCK-test-uuid</string>
	<key>Status</key>
	<string>Acknowledged</string>
	<key>PasswordVerified</key>
	<true/>
	<key>UDID</key>
	<string>host-uuid-123</string>
</dict>
</plist>`)

	result := NewRecoveryLockResult(&mdm.CommandResults{
		CommandUUID: VerifyRecoveryLockCommandPrefix + "test-uuid",
		Status:      fleet.MDMAppleStatusAcknowledged,
		Raw:         rawResponse,
	})

	err := handler(ctx, result)
	require.NoError(t, err)
	assert.True(t, mockDS.GetHostIDByVerifyCommandUUIDFuncInvoked)
	assert.True(t, mockDS.SetRecoveryLockVerifiedFuncInvoked)
	assert.False(t, mockDS.SetRecoveryLockFailedFuncInvoked)
}

func TestVerifyRecoveryLockResultsHandler_Acknowledged_PasswordNotVerified(t *testing.T) {
	ctx := context.Background()
	mockDS := &mockDatastore{
		GetHostIDByVerifyCommandUUIDFunc: func(ctx context.Context, verifyCommandUUID string) (uint, error) {
			return 123, nil
		},
		SetRecoveryLockFailedFunc: func(ctx context.Context, hostID uint, errorMsg string) error {
			return nil
		},
	}
	logger := slog.New(slog.DiscardHandler)

	handler := NewVerifyRecoveryLockResultsHandler(mockDS, logger)

	// Plist response with PasswordVerified=false
	rawResponse := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CommandUUID</key>
	<string>VERIFY-RECOVERY-LOCK-test-uuid</string>
	<key>Status</key>
	<string>Acknowledged</string>
	<key>PasswordVerified</key>
	<false/>
	<key>UDID</key>
	<string>host-uuid-123</string>
</dict>
</plist>`)

	result := NewRecoveryLockResult(&mdm.CommandResults{
		CommandUUID: VerifyRecoveryLockCommandPrefix + "test-uuid",
		Status:      fleet.MDMAppleStatusAcknowledged,
		Raw:         rawResponse,
	})

	err := handler(ctx, result)
	require.NoError(t, err)
	assert.True(t, mockDS.GetHostIDByVerifyCommandUUIDFuncInvoked)
	assert.False(t, mockDS.SetRecoveryLockVerifiedFuncInvoked)
	assert.True(t, mockDS.SetRecoveryLockFailedFuncInvoked)
	assert.Contains(t, mockDS.LastFailedErrorMsg, "password does not match")
}

func TestVerifyRecoveryLockResultsHandler_NotNow(t *testing.T) {
	ctx := context.Background()
	mockDS := &mockDatastore{
		GetHostIDByVerifyCommandUUIDFunc: func(ctx context.Context, verifyCommandUUID string) (uint, error) {
			return 123, nil
		},
	}
	logger := slog.New(slog.DiscardHandler)

	handler := NewVerifyRecoveryLockResultsHandler(mockDS, logger)

	result := NewRecoveryLockResult(&mdm.CommandResults{
		CommandUUID: VerifyRecoveryLockCommandPrefix + "test-uuid",
		Status:      fleet.MDMAppleStatusNotNow,
	})

	err := handler(ctx, result)
	require.NoError(t, err)
	assert.True(t, mockDS.GetHostIDByVerifyCommandUUIDFuncInvoked)
	// Status should remain verifying, no updates
	assert.False(t, mockDS.SetRecoveryLockVerifiedFuncInvoked)
	assert.False(t, mockDS.SetRecoveryLockFailedFuncInvoked)
}

func TestVerifyRecoveryLockResultsHandler_Error(t *testing.T) {
	ctx := context.Background()
	mockDS := &mockDatastore{
		GetHostIDByVerifyCommandUUIDFunc: func(ctx context.Context, verifyCommandUUID string) (uint, error) {
			return 123, nil
		},
		SetRecoveryLockFailedFunc: func(ctx context.Context, hostID uint, errorMsg string) error {
			return nil
		},
	}
	logger := slog.New(slog.DiscardHandler)

	handler := NewVerifyRecoveryLockResultsHandler(mockDS, logger)

	result := NewRecoveryLockResult(&mdm.CommandResults{
		CommandUUID: VerifyRecoveryLockCommandPrefix + "test-uuid",
		Status:      fleet.MDMAppleStatusError,
		ErrorChain: []mdm.ErrorChain{
			{ErrorCode: 12021, ErrorDomain: "MCMDMErrorDomain", LocalizedDescription: "Test error"},
		},
	})

	err := handler(ctx, result)
	require.NoError(t, err)
	assert.True(t, mockDS.GetHostIDByVerifyCommandUUIDFuncInvoked)
	assert.True(t, mockDS.SetRecoveryLockFailedFuncInvoked)
}

// testNotFoundError is a simple implementation of fleet.NotFoundError for testing.
type testNotFoundError struct {
	msg string
}

func (e *testNotFoundError) Error() string {
	return e.msg
}

func (e *testNotFoundError) IsNotFound() bool {
	return true
}

func (e *testNotFoundError) Message() string {
	return e.msg
}

func (e *testNotFoundError) StatusCode() int {
	return 404
}

func (e *testNotFoundError) IsClientError() bool {
	return true
}

func TestVerifyRecoveryLockResultsHandler_CommandNotFound(t *testing.T) {
	ctx := context.Background()
	mockDS := &mockDatastore{
		GetHostIDByVerifyCommandUUIDFunc: func(ctx context.Context, verifyCommandUUID string) (uint, error) {
			return 0, &testNotFoundError{msg: "not found"}
		},
	}
	logger := slog.New(slog.DiscardHandler)

	handler := NewVerifyRecoveryLockResultsHandler(mockDS, logger)

	result := NewRecoveryLockResult(&mdm.CommandResults{
		CommandUUID: VerifyRecoveryLockCommandPrefix + "unknown-uuid",
		Status:      fleet.MDMAppleStatusAcknowledged,
	})

	// Should not return error for unknown command (just skip)
	err := handler(ctx, result)
	require.NoError(t, err)
	assert.True(t, mockDS.GetHostIDByVerifyCommandUUIDFuncInvoked)
	assert.False(t, mockDS.SetRecoveryLockVerifiedFuncInvoked)
}
