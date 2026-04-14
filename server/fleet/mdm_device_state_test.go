package fleet

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryDeviceStateStore_UpdateAndGet(t *testing.T) {
	store := NewInMemoryDeviceStateStore()
	now := time.Now()
	entries := map[string]DeviceStateEntry{
		"DeviceInformation.OSVersion":  {Value: "17.4", Source: "mdm_poll", ObservedAt: now},
		"DeviceInformation.DeviceName": {Value: "Fleet-iPad", Source: "mdm_poll", ObservedAt: now},
		"SecurityInfo.PasscodePresent": {Value: "true", Source: "mdm_poll", ObservedAt: now},
	}
	err := store.UpdateDeviceState("host-1", entries)
	require.NoError(t, err)

	result, err := store.GetDeviceState("host-1")
	require.NoError(t, err)
	require.Len(t, result, 3)
	assert.Equal(t, "17.4", result["DeviceInformation.OSVersion"].Value)
	assert.Equal(t, "Fleet-iPad", result["DeviceInformation.DeviceName"].Value)
	assert.Equal(t, "true", result["SecurityInfo.PasscodePresent"].Value)
}

func TestInMemoryDeviceStateStore_UpdateMerges(t *testing.T) {
	store := NewInMemoryDeviceStateStore()
	now := time.Now()

	// First update: A and B
	err := store.UpdateDeviceState("host-1", map[string]DeviceStateEntry{
		"A": {Value: "1", Source: "mdm_poll", ObservedAt: now},
		"B": {Value: "old", Source: "mdm_poll", ObservedAt: now},
	})
	require.NoError(t, err)

	// Second update: B (new value) and C
	later := now.Add(time.Minute)
	err = store.UpdateDeviceState("host-1", map[string]DeviceStateEntry{
		"B": {Value: "new", Source: "ddm", ObservedAt: later},
		"C": {Value: "3", Source: "ddm", ObservedAt: later},
	})
	require.NoError(t, err)

	result, err := store.GetDeviceState("host-1")
	require.NoError(t, err)
	require.Len(t, result, 3)
	assert.Equal(t, "1", result["A"].Value)
	assert.Equal(t, "new", result["B"].Value)
	assert.Equal(t, "ddm", result["B"].Source)
	assert.Equal(t, "3", result["C"].Value)
}

func TestInMemoryDeviceStateStore_GetUnknownHost(t *testing.T) {
	store := NewInMemoryDeviceStateStore()
	result, err := store.GetDeviceState("unknown-host")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestInMemoryDeviceStateStore_ConcurrentAccess(t *testing.T) {
	store := NewInMemoryDeviceStateStore()
	var wg sync.WaitGroup
	now := time.Now()

	// 10 writers
	for range 10 {
		wg.Go(func() {
			for range 100 {
				_ = store.UpdateDeviceState("host-1", map[string]DeviceStateEntry{
					"field": {Value: "val", Source: "mdm_poll", ObservedAt: now},
				})
			}
		})
	}

	// 10 readers
	for range 10 {
		wg.Go(func() {
			for range 100 {
				_, _ = store.GetDeviceState("host-1")
			}
		})
	}

	wg.Wait()
	// If we get here without a race condition panic, the test passes
	result, err := store.GetDeviceState("host-1")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}
