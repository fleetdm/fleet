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
