package service

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluateMDMPolicy(t *testing.T) {
	now := time.Now()

	// helper to build a DeviceStateEntry
	entry := func(value string) fleet.DeviceStateEntry {
		return fleet.DeviceStateEntry{Value: value, Source: "mdm_poll", ObservedAt: now}
	}

	tests := []struct {
		name       string
		checks     []fleet.MDMPolicyCheck
		deviceData map[string]fleet.DeviceStateEntry
		wantPasses bool
	}{
		{
			name:       "empty_checks_passes",
			checks:     nil,
			deviceData: map[string]fleet.DeviceStateEntry{"OSVersion": entry("17.4")},
			wantPasses: true,
		},
		{
			name:       "string_eq_pass",
			checks:     []fleet.MDMPolicyCheck{{Field: "OSVersion", Operator: fleet.MDMPolicyCheckEq, Expected: "17.4"}},
			deviceData: map[string]fleet.DeviceStateEntry{"OSVersion": entry("17.4")},
			wantPasses: true,
		},
		{
			name:       "string_eq_fail",
			checks:     []fleet.MDMPolicyCheck{{Field: "OSVersion", Operator: fleet.MDMPolicyCheckEq, Expected: "17.4"}},
			deviceData: map[string]fleet.DeviceStateEntry{"OSVersion": entry("16.0")},
			wantPasses: false,
		},
		{
			name:       "string_neq_pass",
			checks:     []fleet.MDMPolicyCheck{{Field: "ProductName", Operator: fleet.MDMPolicyCheckNeq, Expected: "iPhone"}},
			deviceData: map[string]fleet.DeviceStateEntry{"ProductName": entry("iPad")},
			wantPasses: true,
		},
		{
			name:       "string_neq_fail",
			checks:     []fleet.MDMPolicyCheck{{Field: "ProductName", Operator: fleet.MDMPolicyCheckNeq, Expected: "iPhone"}},
			deviceData: map[string]fleet.DeviceStateEntry{"ProductName": entry("iPhone")},
			wantPasses: false,
		},
		{
			name:       "numeric_gt_pass",
			checks:     []fleet.MDMPolicyCheck{{Field: "Capacity", Operator: fleet.MDMPolicyCheckGt, Expected: "10.0"}},
			deviceData: map[string]fleet.DeviceStateEntry{"Capacity": entry("25.5")},
			wantPasses: true,
		},
		{
			name:       "numeric_gt_fail",
			checks:     []fleet.MDMPolicyCheck{{Field: "Capacity", Operator: fleet.MDMPolicyCheckGt, Expected: "10.0"}},
			deviceData: map[string]fleet.DeviceStateEntry{"Capacity": entry("5.0")},
			wantPasses: false,
		},
		{
			name:       "numeric_gt_boundary_fail",
			checks:     []fleet.MDMPolicyCheck{{Field: "Capacity", Operator: fleet.MDMPolicyCheckGt, Expected: "10.0"}},
			deviceData: map[string]fleet.DeviceStateEntry{"Capacity": entry("10.0")},
			wantPasses: false,
		},
		{
			name:       "numeric_gte_boundary_pass",
			checks:     []fleet.MDMPolicyCheck{{Field: "Capacity", Operator: fleet.MDMPolicyCheckGte, Expected: "10.0"}},
			deviceData: map[string]fleet.DeviceStateEntry{"Capacity": entry("10.0")},
			wantPasses: true,
		},
		{
			name:       "numeric_lt_pass",
			checks:     []fleet.MDMPolicyCheck{{Field: "Capacity", Operator: fleet.MDMPolicyCheckLt, Expected: "50.0"}},
			deviceData: map[string]fleet.DeviceStateEntry{"Capacity": entry("25.5")},
			wantPasses: true,
		},
		{
			name:       "numeric_lt_boundary_fail",
			checks:     []fleet.MDMPolicyCheck{{Field: "Capacity", Operator: fleet.MDMPolicyCheckLt, Expected: "50.0"}},
			deviceData: map[string]fleet.DeviceStateEntry{"Capacity": entry("50.0")},
			wantPasses: false,
		},
		{
			name:       "numeric_lte_boundary_pass",
			checks:     []fleet.MDMPolicyCheck{{Field: "Capacity", Operator: fleet.MDMPolicyCheckLte, Expected: "50.0"}},
			deviceData: map[string]fleet.DeviceStateEntry{"Capacity": entry("50.0")},
			wantPasses: true,
		},
		{
			name:       "bool_eq_true_pass",
			checks:     []fleet.MDMPolicyCheck{{Field: "Passcode", Operator: fleet.MDMPolicyCheckEq, Expected: "true"}},
			deviceData: map[string]fleet.DeviceStateEntry{"Passcode": entry("true")},
			wantPasses: true,
		},
		{
			name:       "bool_eq_false_fail",
			checks:     []fleet.MDMPolicyCheck{{Field: "Passcode", Operator: fleet.MDMPolicyCheckEq, Expected: "true"}},
			deviceData: map[string]fleet.DeviceStateEntry{"Passcode": entry("false")},
			wantPasses: false,
		},
		{
			name:       "contains_pass",
			checks:     []fleet.MDMPolicyCheck{{Field: "Name", Operator: fleet.MDMPolicyCheckContains, Expected: "Fleet"}},
			deviceData: map[string]fleet.DeviceStateEntry{"Name": entry("Fleet-iPad")},
			wantPasses: true,
		},
		{
			name:       "contains_fail",
			checks:     []fleet.MDMPolicyCheck{{Field: "Name", Operator: fleet.MDMPolicyCheckContains, Expected: "Fleet"}},
			deviceData: map[string]fleet.DeviceStateEntry{"Name": entry("Corp-iPad")},
			wantPasses: false,
		},
		{
			name:       "not_contains_pass",
			checks:     []fleet.MDMPolicyCheck{{Field: "Name", Operator: fleet.MDMPolicyCheckNotContains, Expected: "personal"}},
			deviceData: map[string]fleet.DeviceStateEntry{"Name": entry("Fleet-iPad")},
			wantPasses: true,
		},
		{
			name:       "not_contains_fail",
			checks:     []fleet.MDMPolicyCheck{{Field: "Name", Operator: fleet.MDMPolicyCheckNotContains, Expected: "Fleet"}},
			deviceData: map[string]fleet.DeviceStateEntry{"Name": entry("Fleet-iPad")},
			wantPasses: false,
		},
		{
			name:       "version_gte_equal",
			checks:     []fleet.MDMPolicyCheck{{Field: "OS", Operator: fleet.MDMPolicyCheckVersionGte, Expected: "17.4"}},
			deviceData: map[string]fleet.DeviceStateEntry{"OS": entry("17.4")},
			wantPasses: true,
		},
		{
			name:       "version_gte_higher_minor",
			checks:     []fleet.MDMPolicyCheck{{Field: "OS", Operator: fleet.MDMPolicyCheckVersionGte, Expected: "17.4"}},
			deviceData: map[string]fleet.DeviceStateEntry{"OS": entry("17.5")},
			wantPasses: true,
		},
		{
			name:       "version_gte_higher_patch",
			checks:     []fleet.MDMPolicyCheck{{Field: "OS", Operator: fleet.MDMPolicyCheckVersionGte, Expected: "17.4"}},
			deviceData: map[string]fleet.DeviceStateEntry{"OS": entry("17.4.1")},
			wantPasses: true,
		},
		{
			name:       "version_gte_lower",
			checks:     []fleet.MDMPolicyCheck{{Field: "OS", Operator: fleet.MDMPolicyCheckVersionGte, Expected: "17.4"}},
			deviceData: map[string]fleet.DeviceStateEntry{"OS": entry("16.7.2")},
			wantPasses: false,
		},
		{
			name:       "version_gte_major_higher",
			checks:     []fleet.MDMPolicyCheck{{Field: "OS", Operator: fleet.MDMPolicyCheckVersionGte, Expected: "17.0"}},
			deviceData: map[string]fleet.DeviceStateEntry{"OS": entry("18.1")},
			wantPasses: true,
		},
		{
			name:       "version_gte_patch_lower",
			checks:     []fleet.MDMPolicyCheck{{Field: "OS", Operator: fleet.MDMPolicyCheckVersionGte, Expected: "17.4.2"}},
			deviceData: map[string]fleet.DeviceStateEntry{"OS": entry("17.4.1")},
			wantPasses: false,
		},
		{
			name:       "version_lte_pass",
			checks:     []fleet.MDMPolicyCheck{{Field: "OS", Operator: fleet.MDMPolicyCheckVersionLte, Expected: "18.0"}},
			deviceData: map[string]fleet.DeviceStateEntry{"OS": entry("17.4")},
			wantPasses: true,
		},
		{
			name:       "version_lte_equal",
			checks:     []fleet.MDMPolicyCheck{{Field: "OS", Operator: fleet.MDMPolicyCheckVersionLte, Expected: "17.4"}},
			deviceData: map[string]fleet.DeviceStateEntry{"OS": entry("17.4")},
			wantPasses: true,
		},
		{
			name:       "version_lte_fail",
			checks:     []fleet.MDMPolicyCheck{{Field: "OS", Operator: fleet.MDMPolicyCheckVersionLte, Expected: "17.0"}},
			deviceData: map[string]fleet.DeviceStateEntry{"OS": entry("17.4")},
			wantPasses: false,
		},
		{
			name:       "exists_pass",
			checks:     []fleet.MDMPolicyCheck{{Field: "WiFiMAC", Operator: fleet.MDMPolicyCheckExists, Expected: ""}},
			deviceData: map[string]fleet.DeviceStateEntry{"WiFiMAC": entry("AA:BB:CC:DD:EE:FF")},
			wantPasses: true,
		},
		{
			name:       "exists_fail",
			checks:     []fleet.MDMPolicyCheck{{Field: "WiFiMAC", Operator: fleet.MDMPolicyCheckExists, Expected: ""}},
			deviceData: map[string]fleet.DeviceStateEntry{"OtherField": entry("value")},
			wantPasses: false,
		},
		{
			name:       "not_exists_pass",
			checks:     []fleet.MDMPolicyCheck{{Field: "MissingField", Operator: fleet.MDMPolicyCheckNotExists, Expected: ""}},
			deviceData: map[string]fleet.DeviceStateEntry{"OtherField": entry("value")},
			wantPasses: true,
		},
		{
			name:       "not_exists_fail",
			checks:     []fleet.MDMPolicyCheck{{Field: "PresentField", Operator: fleet.MDMPolicyCheckNotExists, Expected: ""}},
			deviceData: map[string]fleet.DeviceStateEntry{"PresentField": entry("value")},
			wantPasses: false,
		},
		{
			name: "and_all_pass",
			checks: []fleet.MDMPolicyCheck{
				{Field: "OSVersion", Operator: fleet.MDMPolicyCheckEq, Expected: "17.4"},
				{Field: "Passcode", Operator: fleet.MDMPolicyCheckEq, Expected: "true"},
				{Field: "Encrypted", Operator: fleet.MDMPolicyCheckEq, Expected: "true"},
			},
			deviceData: map[string]fleet.DeviceStateEntry{
				"OSVersion": entry("17.4"),
				"Passcode":  entry("true"),
				"Encrypted": entry("true"),
			},
			wantPasses: true,
		},
		{
			name: "and_one_fails",
			checks: []fleet.MDMPolicyCheck{
				{Field: "OSVersion", Operator: fleet.MDMPolicyCheckEq, Expected: "17.4"},
				{Field: "Passcode", Operator: fleet.MDMPolicyCheckEq, Expected: "true"},
				{Field: "Encrypted", Operator: fleet.MDMPolicyCheckEq, Expected: "true"},
			},
			deviceData: map[string]fleet.DeviceStateEntry{
				"OSVersion": entry("17.4"),
				"Passcode":  entry("false"),
				"Encrypted": entry("true"),
			},
			wantPasses: false,
		},
		{
			name:       "missing_field_fails",
			checks:     []fleet.MDMPolicyCheck{{Field: "MissingField", Operator: fleet.MDMPolicyCheckEq, Expected: "value"}},
			deviceData: map[string]fleet.DeviceStateEntry{},
			wantPasses: false,
		},
		{
			name:       "nil_device_data",
			checks:     []fleet.MDMPolicyCheck{{Field: "OSVersion", Operator: fleet.MDMPolicyCheckEq, Expected: "17.4"}},
			deviceData: nil,
			wantPasses: false,
		},
		{
			name:       "empty_device_data",
			checks:     []fleet.MDMPolicyCheck{{Field: "OSVersion", Operator: fleet.MDMPolicyCheckEq, Expected: "17.4"}},
			deviceData: map[string]fleet.DeviceStateEntry{},
			wantPasses: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := fleet.MDMPolicyDefinition{Checks: tt.checks}
			result := EvaluateMDMPolicy(1, 42, def, tt.deviceData)

			assert.Equal(t, tt.wantPasses, result.Passes, "expected passes=%v, got passes=%v", tt.wantPasses, result.Passes)
			assert.Equal(t, uint(42), result.HostID)
			assert.Equal(t, uint(1), result.PolicyID)
			assert.False(t, result.Timestamp.IsZero())
			if tt.wantPasses {
				require.NoError(t, result.Err)
			}
		})
	}
}

func TestEvaluateMDMPolicy_UnsupportedOperator(t *testing.T) {
	def := fleet.MDMPolicyDefinition{
		Checks: []fleet.MDMPolicyCheck{
			{Field: "Foo", Operator: "bogus_op", Expected: "bar"},
		},
	}
	deviceData := map[string]fleet.DeviceStateEntry{
		"Foo": {Value: "bar", Source: "test", ObservedAt: time.Now()},
	}

	result := EvaluateMDMPolicy(1, 1, def, deviceData)
	assert.False(t, result.Passes)
	assert.Error(t, result.Err)
	assert.Contains(t, result.Err.Error(), "unsupported operator")
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"17.4", "17.4", 0},
		{"17.5", "17.4", 1},
		{"17.3", "17.4", -1},
		{"17.4.1", "17.4", 1},
		{"17.4", "17.4.1", -1},
		{"18.0", "17.9.9", 1},
		{"1.0.0", "1.0.0", 0},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got := compareVersions(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}
