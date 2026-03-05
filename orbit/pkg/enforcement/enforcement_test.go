package enforcement

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrNotSupported(t *testing.T) {
	assert.Error(t, ErrNotSupported)
	assert.Contains(t, ErrNotSupported.Error(), "not supported")
}

func TestStubHandlers(t *testing.T) {
	ctx := context.Background()

	handlers := []Handler{
		NewRegistryHandler(),
		NewServiceHandler(),
		NewAuditHandler(),
		NewSecpolHandler(),
	}

	expectedNames := []string{"registry", "service", "audit", "secpol"}

	for i, h := range handlers {
		t.Run(expectedNames[i], func(t *testing.T) {
			assert.Equal(t, expectedNames[i], h.Name())

			diffResults, err := h.Diff(ctx, []byte(`{}`))
			if errors.Is(err, ErrNotSupported) {
				// Stub implementation on non-Windows
				assert.Nil(t, diffResults)
			} else {
				// Windows implementation would return real results or nil
				require.NoError(t, err)
			}

			applyResults, err := h.Apply(ctx, []byte(`{}`))
			if errors.Is(err, ErrNotSupported) {
				assert.Nil(t, applyResults)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestComplianceCacheEmpty(t *testing.T) {
	cache := NewComplianceCache()
	records := cache.Records()
	assert.NotNil(t, records)
	assert.Empty(t, records)
}

func TestComplianceCacheUpdateAndRetrieve(t *testing.T) {
	cache := NewComplianceCache()
	now := time.Now()

	input := []ComplianceRecord{
		{
			SettingName:  "EnableLUA",
			Category:     "registry",
			PolicyName:   "cis-uac",
			CISRef:       "2.3.17.7",
			DesiredValue: "1",
			CurrentValue: "1",
			Compliant:    true,
			LastChecked:  now,
		},
		{
			SettingName:  "LmCompatibilityLevel",
			Category:     "registry",
			PolicyName:   "cis-ntlm",
			CISRef:       "2.3.11.7",
			DesiredValue: "5",
			CurrentValue: "3",
			Compliant:    false,
			LastChecked:  now,
		},
	}

	cache.Update(input)
	records := cache.Records()
	require.Len(t, records, 2)
	assert.Equal(t, "EnableLUA", records[0].SettingName)
	assert.True(t, records[0].Compliant)
	assert.Equal(t, "LmCompatibilityLevel", records[1].SettingName)
	assert.False(t, records[1].Compliant)
	assert.Equal(t, "2.3.11.7", records[1].CISRef)
}

func TestComplianceCacheCopyOnRead(t *testing.T) {
	cache := NewComplianceCache()
	cache.Update([]ComplianceRecord{
		{SettingName: "original", Compliant: true},
	})

	// Modify returned slice
	records := cache.Records()
	records[0].SettingName = "mutated"
	records[0].Compliant = false

	// Original should be unchanged
	original := cache.Records()
	assert.Equal(t, "original", original[0].SettingName)
	assert.True(t, original[0].Compliant)
}

func TestComplianceCacheReplace(t *testing.T) {
	cache := NewComplianceCache()
	cache.Update([]ComplianceRecord{
		{SettingName: "first"},
		{SettingName: "second"},
	})
	assert.Len(t, cache.Records(), 2)

	// Replace with single record
	cache.Update([]ComplianceRecord{
		{SettingName: "replacement"},
	})
	records := cache.Records()
	require.Len(t, records, 1)
	assert.Equal(t, "replacement", records[0].SettingName)

	// Replace with empty
	cache.Update(nil)
	assert.Empty(t, cache.Records())
}

func TestNewRunner(t *testing.T) {
	cache := NewComplianceCache()
	handlers := map[string]Handler{
		"test": &mockHandler{name: "test"},
	}
	runner := NewRunner(handlers, cache)
	require.NotNil(t, runner)
}

func TestRunnerDiffError(t *testing.T) {
	handler := &mockHandler{
		name:    "failing",
		diffErr: errors.New("diff failed"),
	}

	cache := NewComplianceCache()
	runner := NewRunner(map[string]Handler{"failing": handler}, cache)

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

	// Diff error should not crash the runner
	err := runner.Run(cfg)
	require.NoError(t, err)
	// No records should be cached since diff failed
	assert.Empty(t, cache.Records())
}

func TestRunnerApplyError(t *testing.T) {
	handler := &mockHandler{
		name: "apply-fail",
		diffResults: []DiffResult{
			{SettingName: "s1", Compliant: false},
		},
		applyErr: errors.New("apply failed"),
	}

	cache := NewComplianceCache()
	runner := NewRunner(map[string]Handler{"apply-fail": handler}, cache)

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
	assert.Equal(t, 1, handler.applyCalls)
}

func TestRunnerApplyWithFailedSettings(t *testing.T) {
	diffCall := 0
	handler := &mockHandler{
		name: "partial",
	}
	// Override Diff to return different results on second call
	originalDiff := handler.diffResults
	handler.diffResults = []DiffResult{
		{SettingName: "s1", Compliant: false, Category: "registry"},
		{SettingName: "s2", Compliant: true, Category: "registry"},
	}
	handler.applyResults = []ApplyResult{
		{SettingName: "s1", Success: false, Error: "access denied"},
	}
	_ = originalDiff
	_ = diffCall

	cache := NewComplianceCache()
	runner := NewRunner(map[string]Handler{"partial": handler}, cache)

	cfg := &fleet.OrbitConfig{
		Notifications: fleet.OrbitConfigNotifications{
			PendingWindowsEnforcementHash: "hash1",
		},
		WindowsEnforcement: &fleet.OrbitWindowsEnforcement{
			Policies: []fleet.OrbitEnforcementPolicy{
				{ProfileUUID: "e-uuid-1", Name: "cis-policy", RawPolicy: []byte(`{}`)},
			},
		},
	}

	err := runner.Run(cfg)
	require.NoError(t, err)
	assert.Equal(t, 1, handler.applyCalls)

	// Records should be updated despite partial failure
	records := cache.Records()
	assert.GreaterOrEqual(t, len(records), 1)
}

func TestRunnerEmptyPolicies(t *testing.T) {
	cache := NewComplianceCache()
	runner := NewRunner(map[string]Handler{}, cache)

	cfg := &fleet.OrbitConfig{
		Notifications: fleet.OrbitConfigNotifications{
			PendingWindowsEnforcementHash: "hash1",
		},
		WindowsEnforcement: &fleet.OrbitWindowsEnforcement{
			Policies: []fleet.OrbitEnforcementPolicy{},
		},
	}

	err := runner.Run(cfg)
	require.NoError(t, err)
}

func TestRunnerMultipleHandlers(t *testing.T) {
	registryHandler := &mockHandler{
		name: "registry",
		diffResults: []DiffResult{
			{SettingName: "EnableLUA", Category: "registry", Compliant: true},
		},
	}
	serviceHandler := &mockHandler{
		name: "service",
		diffResults: []DiffResult{
			{SettingName: "sshd", Category: "service", Compliant: true},
		},
	}

	cache := NewComplianceCache()
	runner := NewRunner(map[string]Handler{
		"registry": registryHandler,
		"service":  serviceHandler,
	}, cache)

	cfg := &fleet.OrbitConfig{
		Notifications: fleet.OrbitConfigNotifications{
			PendingWindowsEnforcementHash: "hash1",
		},
		WindowsEnforcement: &fleet.OrbitWindowsEnforcement{
			Policies: []fleet.OrbitEnforcementPolicy{
				{ProfileUUID: "e-uuid-1", Name: "policy1", RawPolicy: []byte(`{}`)},
			},
		},
	}

	err := runner.Run(cfg)
	require.NoError(t, err)

	records := cache.Records()
	assert.Len(t, records, 2)
}

func TestDiffResultFields(t *testing.T) {
	r := DiffResult{
		SettingName:  "EnableLUA",
		Category:     "registry",
		PolicyName:   "cis-uac",
		CISRef:       "2.3.17.7",
		DesiredValue: "1",
		CurrentValue: "0",
		Compliant:    false,
	}
	assert.Equal(t, "EnableLUA", r.SettingName)
	assert.Equal(t, "registry", r.Category)
	assert.False(t, r.Compliant)
}

func TestApplyResultFields(t *testing.T) {
	r := ApplyResult{
		SettingName: "EnableLUA",
		Category:    "registry",
		Success:     false,
		Error:       "access denied",
	}
	assert.Equal(t, "EnableLUA", r.SettingName)
	assert.False(t, r.Success)
	assert.Equal(t, "access denied", r.Error)
}

func TestRunnerAllCompliant(t *testing.T) {
	handler := &mockHandler{
		name: "registry",
		diffResults: []DiffResult{
			{SettingName: "EnableLUA", Category: "registry", Compliant: true, DesiredValue: "1", CurrentValue: "1"},
			{SettingName: "LmLevel", Category: "registry", Compliant: true, DesiredValue: "5", CurrentValue: "5"},
		},
	}

	cache := NewComplianceCache()
	runner := NewRunner(map[string]Handler{"registry": handler}, cache)

	cfg := &fleet.OrbitConfig{
		Notifications: fleet.OrbitConfigNotifications{
			PendingWindowsEnforcementHash: "hash-all-compliant",
		},
		WindowsEnforcement: &fleet.OrbitWindowsEnforcement{
			Policies: []fleet.OrbitEnforcementPolicy{
				{ProfileUUID: "e-uuid-1", Name: "cis-policy", RawPolicy: []byte(`{}`)},
			},
		},
	}

	err := runner.Run(cfg)
	require.NoError(t, err)

	// Apply should NOT be called since all are compliant
	assert.Equal(t, 0, handler.applyCalls)

	// Records should still be cached
	records := cache.Records()
	require.Len(t, records, 2)
	assert.True(t, records[0].Compliant)
	assert.True(t, records[1].Compliant)
	assert.Equal(t, "cis-policy", records[0].PolicyName)
	assert.Equal(t, "1", records[0].DesiredValue)
	assert.Equal(t, "1", records[0].CurrentValue)
}

func TestRunnerMultiplePolicies(t *testing.T) {
	handler := &mockHandler{
		name: "registry",
		diffResults: []DiffResult{
			{SettingName: "s1", Category: "registry", Compliant: true},
		},
	}

	cache := NewComplianceCache()
	runner := NewRunner(map[string]Handler{"registry": handler}, cache)

	cfg := &fleet.OrbitConfig{
		Notifications: fleet.OrbitConfigNotifications{
			PendingWindowsEnforcementHash: "hash-multi",
		},
		WindowsEnforcement: &fleet.OrbitWindowsEnforcement{
			Policies: []fleet.OrbitEnforcementPolicy{
				{ProfileUUID: "e-uuid-1", Name: "policy-a", RawPolicy: []byte(`{}`)},
				{ProfileUUID: "e-uuid-2", Name: "policy-b", RawPolicy: []byte(`{}`)},
				{ProfileUUID: "e-uuid-3", Name: "policy-c", RawPolicy: []byte(`{}`)},
			},
		},
	}

	err := runner.Run(cfg)
	require.NoError(t, err)

	// Should have 3 records (one per policy since handler returns 1 result)
	records := cache.Records()
	require.Len(t, records, 3)

	// Each record should have the corresponding policy name
	policyNames := map[string]bool{}
	for _, r := range records {
		policyNames[r.PolicyName] = true
	}
	assert.True(t, policyNames["policy-a"])
	assert.True(t, policyNames["policy-b"])
	assert.True(t, policyNames["policy-c"])
}

func TestRunnerPostApplyDiffError(t *testing.T) {
	diffCallCount := 0
	handler := &mockHandler{
		name: "test",
	}
	// Override with a custom diff that fails on second call
	originalDiff := handler.Diff
	_ = originalDiff

	// First diff returns non-compliant, second diff (post-apply) fails
	customHandler := &mockHandlerWithPostDiffError{
		name:      "failing-post-diff",
		diffCount: 0,
		firstDiff: []DiffResult{
			{SettingName: "s1", Compliant: false},
		},
		applyResults: []ApplyResult{
			{SettingName: "s1", Success: true},
		},
	}
	_ = diffCallCount

	cache := NewComplianceCache()
	runner := NewRunner(map[string]Handler{"failing-post-diff": customHandler}, cache)

	cfg := &fleet.OrbitConfig{
		Notifications: fleet.OrbitConfigNotifications{
			PendingWindowsEnforcementHash: "hash-post-diff-err",
		},
		WindowsEnforcement: &fleet.OrbitWindowsEnforcement{
			Policies: []fleet.OrbitEnforcementPolicy{
				{ProfileUUID: "e-uuid-1", Name: "test-policy", RawPolicy: []byte(`{}`)},
			},
		},
	}

	err := runner.Run(cfg)
	require.NoError(t, err)
	assert.Equal(t, 1, customHandler.applyCalls)

	// No records cached because post-apply diff failed
	assert.Empty(t, cache.Records())
}

func TestRunnerApplySuccessWithLogs(t *testing.T) {
	handler := &mockHandler{
		name: "registry",
		diffResults: []DiffResult{
			{SettingName: "EnableLUA", Category: "registry", Compliant: false, DesiredValue: "1", CurrentValue: "0"},
		},
		applyResults: []ApplyResult{
			{SettingName: "EnableLUA", Category: "registry", Success: true},
		},
	}

	cache := NewComplianceCache()
	runner := NewRunner(map[string]Handler{"registry": handler}, cache)

	cfg := &fleet.OrbitConfig{
		Notifications: fleet.OrbitConfigNotifications{
			PendingWindowsEnforcementHash: "hash-apply-success",
		},
		WindowsEnforcement: &fleet.OrbitWindowsEnforcement{
			Policies: []fleet.OrbitEnforcementPolicy{
				{ProfileUUID: "e-uuid-1", Name: "cis-uac", RawPolicy: []byte(`{}`)},
			},
		},
	}

	err := runner.Run(cfg)
	require.NoError(t, err)
	assert.Equal(t, 1, handler.applyCalls)

	// After apply, a re-diff happens; since our mock returns the same results,
	// we'll get them in the cache
	records := cache.Records()
	require.Len(t, records, 1)
	assert.Equal(t, "cis-uac", records[0].PolicyName)
}

func TestRunnerApplyWithPartialFailures(t *testing.T) {
	handler := &mockHandler{
		name: "registry",
		diffResults: []DiffResult{
			{SettingName: "s1", Category: "registry", Compliant: false},
			{SettingName: "s2", Category: "registry", Compliant: false},
		},
		applyResults: []ApplyResult{
			{SettingName: "s1", Category: "registry", Success: true},
			{SettingName: "s2", Category: "registry", Success: false, Error: "access denied"},
		},
	}

	cache := NewComplianceCache()
	runner := NewRunner(map[string]Handler{"registry": handler}, cache)

	cfg := &fleet.OrbitConfig{
		Notifications: fleet.OrbitConfigNotifications{
			PendingWindowsEnforcementHash: "hash-partial",
		},
		WindowsEnforcement: &fleet.OrbitWindowsEnforcement{
			Policies: []fleet.OrbitEnforcementPolicy{
				{ProfileUUID: "e-uuid-1", Name: "mixed-results", RawPolicy: []byte(`{}`)},
			},
		},
	}

	err := runner.Run(cfg)
	require.NoError(t, err)
	assert.Equal(t, 1, handler.applyCalls)
}

func TestComplianceRecordFields(t *testing.T) {
	now := time.Now()
	r := ComplianceRecord{
		SettingName:  "EnableLUA",
		Category:     "registry",
		PolicyName:   "cis-uac",
		CISRef:       "2.3.17.7",
		DesiredValue: "1",
		CurrentValue: "0",
		Compliant:    false,
		LastChecked:  now,
	}
	assert.Equal(t, "EnableLUA", r.SettingName)
	assert.Equal(t, "registry", r.Category)
	assert.Equal(t, "cis-uac", r.PolicyName)
	assert.Equal(t, "2.3.17.7", r.CISRef)
	assert.Equal(t, "1", r.DesiredValue)
	assert.Equal(t, "0", r.CurrentValue)
	assert.False(t, r.Compliant)
	assert.Equal(t, now, r.LastChecked)
}

// mockHandlerWithPostDiffError is a handler that returns an error on the second Diff call
// (post-apply verification), simulating a transient failure.
type mockHandlerWithPostDiffError struct {
	name         string
	diffCount    int
	firstDiff    []DiffResult
	applyResults []ApplyResult
	applyCalls   int
}

func (h *mockHandlerWithPostDiffError) Name() string { return h.name }
func (h *mockHandlerWithPostDiffError) Diff(ctx context.Context, rawPolicy []byte) ([]DiffResult, error) {
	h.diffCount++
	if h.diffCount > 1 {
		return nil, errors.New("post-apply diff failed")
	}
	return h.firstDiff, nil
}
func (h *mockHandlerWithPostDiffError) Apply(ctx context.Context, rawPolicy []byte) ([]ApplyResult, error) {
	h.applyCalls++
	return h.applyResults, nil
}
