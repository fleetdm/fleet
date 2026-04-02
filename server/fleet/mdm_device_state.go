package fleet

import "sync"

// InMemoryDeviceStateStore is a thread-safe in-memory implementation of DeviceStateStore.
type InMemoryDeviceStateStore struct {
	mu    sync.RWMutex
	store map[string]map[string]DeviceStateEntry
}

// NewInMemoryDeviceStateStore creates a new in-memory device state store.
func NewInMemoryDeviceStateStore() *InMemoryDeviceStateStore {
	return &InMemoryDeviceStateStore{
		store: make(map[string]map[string]DeviceStateEntry),
	}
}

// UpdateDeviceState merges entries into the device state for the given host.
// Existing keys are overwritten; new keys are added.
func (s *InMemoryDeviceStateStore) UpdateDeviceState(hostUUID string, entries map[string]DeviceStateEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store[hostUUID] == nil {
		s.store[hostUUID] = make(map[string]DeviceStateEntry)
	}
	for k, v := range entries {
		s.store[hostUUID][k] = v
	}
	return nil
}

// GetDeviceState returns a copy of all state entries for the given host.
// Returns an empty map (not nil) for unknown hosts.
func (s *InMemoryDeviceStateStore) GetDeviceState(hostUUID string) (map[string]DeviceStateEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hostData, ok := s.store[hostUUID]
	if !ok {
		return map[string]DeviceStateEntry{}, nil
	}
	// Return a copy to prevent concurrent modification
	result := make(map[string]DeviceStateEntry, len(hostData))
	for k, v := range hostData {
		result[k] = v
	}
	return result, nil
}
