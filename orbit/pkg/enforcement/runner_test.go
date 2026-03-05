package enforcement

import (
	"context"

	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHandler is a test handler that tracks calls and returns configurable results.
type mockHandler struct {
	name        string
	diffResults []DiffResult
	diffErr     error
	applyCalls  int
	applyResults []ApplyResult
	applyErr    error
}

func (h *mockHandler) Name() string { return h.name }
func (h *mockHandler) Diff(ctx context.Context, rawPolicy []byte) ([]DiffResult, error) {
	return h.diffResults, h.diffErr
}
func (h *mockHandler) Apply(ctx context.Context, rawPolicy []byte) ([]ApplyResult, error) {
	h.applyCalls++
	return h.applyResults, h.applyErr
}

func TestRunnerSkipsWhenHashUnchanged(t *testing.T) {
	handler := &mockHandler{
		name: "test",
		diffResults: []DiffResult{
			{SettingName: "test-setting", Compliant: true},
		},
	}

	cache := NewComplianceCache()
	runner := NewRunner(
		map[string]Handler{"test": handler},
		cache,
	)

	cfg := &fleet.OrbitConfig{
		Notifications: fleet.OrbitConfigNotifications{
			PendingWindowsEnforcementHash: "hash1",
		},
		WindowsEnforcement: &fleet.OrbitWindowsEnforcement{
			Policies: []fleet.OrbitEnforcementPolicy{
				{ProfileUUID: "e-uuid-1", Name: "test-policy", RawPolicy: []byte(`{}`)},
			},
		},
	}

	// First run should process
	err := runner.Run(cfg)
	require.NoError(t, err)

	records := cache.Records()
	assert.Len(t, records, 1)
	assert.Equal(t, "test-setting", records[0].SettingName)
	assert.True(t, records[0].Compliant)

	// Second run with same hash should skip
	handler.diffResults = []DiffResult{
		{SettingName: "changed", Compliant: false},
	}
	err = runner.Run(cfg)
	require.NoError(t, err)

	// Cache should still have the old results
	records = cache.Records()
	assert.Len(t, records, 1)
	assert.Equal(t, "test-setting", records[0].SettingName)
}

func TestRunnerProcessesOnHashChange(t *testing.T) {
	handler := &mockHandler{
		name: "test",
		diffResults: []DiffResult{
			{SettingName: "s1", Compliant: true},
		},
	}

	cache := NewComplianceCache()
	runner := NewRunner(
		map[string]Handler{"test": handler},
		cache,
	)

	cfg := &fleet.OrbitConfig{
		Notifications: fleet.OrbitConfigNotifications{
			PendingWindowsEnforcementHash: "hash1",
		},
		WindowsEnforcement: &fleet.OrbitWindowsEnforcement{
			Policies: []fleet.OrbitEnforcementPolicy{
				{ProfileUUID: "e-uuid-1", Name: "test-policy", RawPolicy: []byte(`{}`)},
			},
		},
	}

	err := runner.Run(cfg)
	require.NoError(t, err)
	assert.Len(t, cache.Records(), 1)

	// Change hash and results
	cfg.Notifications.PendingWindowsEnforcementHash = "hash2"
	handler.diffResults = []DiffResult{
		{SettingName: "s2", Compliant: false},
		{SettingName: "s3", Compliant: true},
	}
	handler.applyResults = []ApplyResult{
		{SettingName: "s2", Success: true},
	}

	err = runner.Run(cfg)
	require.NoError(t, err)

	// Apply should have been called once (re-diff after apply means we get the new results)
	assert.Equal(t, 1, handler.applyCalls)
}

func TestRunnerSkipsEmptyHash(t *testing.T) {
	cache := NewComplianceCache()
	runner := NewRunner(
		map[string]Handler{},
		cache,
	)

	cfg := &fleet.OrbitConfig{
		Notifications: fleet.OrbitConfigNotifications{
			PendingWindowsEnforcementHash: "",
		},
	}

	err := runner.Run(cfg)
	require.NoError(t, err)
	assert.Empty(t, cache.Records())
}

func TestRunnerSkipsNilEnforcement(t *testing.T) {
	cache := NewComplianceCache()
	runner := NewRunner(
		map[string]Handler{},
		cache,
	)

	cfg := &fleet.OrbitConfig{
		Notifications: fleet.OrbitConfigNotifications{
			PendingWindowsEnforcementHash: "some-hash",
		},
		WindowsEnforcement: nil,
	}

	err := runner.Run(cfg)
	require.NoError(t, err)
}

func TestComplianceCache(t *testing.T) {
	cache := NewComplianceCache()

	// Empty cache
	records := cache.Records()
	assert.Empty(t, records)

	// Update cache
	cache.Update([]ComplianceRecord{
		{SettingName: "s1", Compliant: true},
		{SettingName: "s2", Compliant: false},
	})

	records = cache.Records()
	assert.Len(t, records, 2)

	// Verify it's a copy (modifying returned slice doesn't affect cache)
	records[0].SettingName = "modified"
	records2 := cache.Records()
	assert.Equal(t, "s1", records2[0].SettingName)

	// Replace cache
	cache.Update([]ComplianceRecord{
		{SettingName: "s3", Compliant: true},
	})
	records3 := cache.Records()
	assert.Len(t, records3, 1)
	assert.Equal(t, "s3", records3[0].SettingName)
}
