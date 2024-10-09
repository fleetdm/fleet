package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Implement fleet.Lock interface
type mockLock struct {
	AcquireLockFn func(ctx context.Context, key string, value string, expireMs uint64) (ok bool, err error)
	GetFn         func(ctx context.Context, key string) (*string, error)
	AddToSetFn    func(ctx context.Context, key string, value string) error
}

func (m *mockLock) SetIfNotExist(ctx context.Context, key string, value string, expireMs uint64) (ok bool, err error) {
	return m.AcquireLockFn(ctx, key, value, expireMs)
}

func (m *mockLock) ReleaseLock(ctx context.Context, key string, value string) (ok bool, err error) {
	panic("implement me")
}

func (m *mockLock) Get(ctx context.Context, key string) (*string, error) {
	return m.GetFn(ctx, key)
}

func (m *mockLock) GetAndDelete(ctx context.Context, key string) (*string, error) {
	panic("implement me")
}

func (m *mockLock) AddToSet(ctx context.Context, key string, value string) error {
	return m.AddToSetFn(ctx, key, value)
}

func (m *mockLock) RemoveFromSet(ctx context.Context, key string, value string) error {
	panic("implement me")
}

func (m *mockLock) GetSet(ctx context.Context, key string) ([]string, error) {
	panic("implement me")
}

var calendarTestSetup = func(t *testing.T) (*mockLock, *Service) {
	lock := &mockLock{}
	svc := &Service{
		distributedLock: lock,
	}
	return lock, svc
}

func TestGetCalendarLock(t *testing.T) {
	lock, svc := calendarTestSetup(t)
	ctx := context.Background()
	eventUUID := "testUUID"
	lock.AcquireLockFn = func(ctx context.Context, key string, value string, expireMs uint64) (ok bool, err error) {
		return true, nil
	}
	lock.GetFn = func(ctx context.Context, key string) (*string, error) {
		// not reserved
		return nil, nil
	}
	lockValue, reserved, err := svc.getCalendarLock(ctx, eventUUID, false)
	require.NoError(t, err)
	assert.False(t, reserved)
	assert.NotEmpty(t, lockValue)

	// Make sure lock value is empty if we don't acquire the lock.
	lock.AcquireLockFn = func(ctx context.Context, key string, value string, expireMs uint64) (ok bool, err error) {
		return false, nil
	}
	lock.GetFn = func(ctx context.Context, key string) (*string, error) {
		value := "value"
		return &value, nil
	}
	lockValue, reserved, err = svc.getCalendarLock(ctx, eventUUID, false)
	require.NoError(t, err)
	assert.True(t, reserved)
	assert.Empty(t, lockValue)

	addedToSet := false
	lock.AddToSetFn = func(ctx context.Context, key string, value string) error {
		addedToSet = true
		return nil
	}
	lockValue, reserved, err = svc.getCalendarLock(ctx, eventUUID, true)
	require.NoError(t, err)
	assert.True(t, reserved)
	assert.Empty(t, lockValue)
	assert.True(t, addedToSet)

	addedToSet = false
	lock.GetFn = func(ctx context.Context, key string) (*string, error) {
		// not reserved
		return nil, nil
	}
	lockValue, reserved, err = svc.getCalendarLock(ctx, eventUUID, false)
	require.NoError(t, err)
	assert.False(t, reserved)
	assert.Empty(t, lockValue)
	assert.False(t, addedToSet)

	addedToSet = false
	lockValue, reserved, err = svc.getCalendarLock(ctx, eventUUID, true)
	require.NoError(t, err)
	assert.False(t, reserved)
	assert.Empty(t, lockValue)
	assert.True(t, addedToSet)

}
