package enforcement

import (
	"context"
	"testing"
	"time"

	enf "github.com/fleetdm/fleet/v4/orbit/pkg/enforcement"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestColumnsSchema(t *testing.T) {
	cols := Columns()
	require.Len(t, cols, 8)

	names := make([]string, len(cols))
	for i, c := range cols {
		names[i] = c.Name
	}
	assert.Contains(t, names, "setting_name")
	assert.Contains(t, names, "category")
	assert.Contains(t, names, "policy_name")
	assert.Contains(t, names, "cis_ref")
	assert.Contains(t, names, "desired_value")
	assert.Contains(t, names, "current_value")
	assert.Contains(t, names, "compliant")
	assert.Contains(t, names, "last_checked")
}

func TestGenerateNilCache(t *testing.T) {
	// Reset package-level cache to nil
	cache = nil

	rows, err := Generate(context.Background(), table.QueryContext{})
	require.NoError(t, err)
	require.Nil(t, rows)
}

func TestGenerateEmptyCache(t *testing.T) {
	c := enf.NewComplianceCache()
	SetCache(c)

	rows, err := Generate(context.Background(), table.QueryContext{})
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestGenerateWithRecords(t *testing.T) {
	c := enf.NewComplianceCache()
	SetCache(c)

	now := time.Date(2026, 3, 5, 10, 30, 0, 0, time.UTC)
	c.Update([]enf.ComplianceRecord{
		{
			SettingName:  "MaxPasswordAge",
			Category:     "secpol",
			PolicyName:   "cis-password",
			CISRef:       "1.1.2",
			DesiredValue: "60",
			CurrentValue: "90",
			Compliant:    false,
			LastChecked:  now,
		},
		{
			SettingName:  "AuditLogonEvents",
			Category:     "audit",
			PolicyName:   "cis-audit",
			CISRef:       "17.5.1",
			DesiredValue: "3",
			CurrentValue: "3",
			Compliant:    true,
			LastChecked:  now,
		},
	})

	rows, err := Generate(context.Background(), table.QueryContext{})
	require.NoError(t, err)
	require.Len(t, rows, 2)

	// First row: non-compliant
	assert.Equal(t, "MaxPasswordAge", rows[0]["setting_name"])
	assert.Equal(t, "secpol", rows[0]["category"])
	assert.Equal(t, "cis-password", rows[0]["policy_name"])
	assert.Equal(t, "1.1.2", rows[0]["cis_ref"])
	assert.Equal(t, "60", rows[0]["desired_value"])
	assert.Equal(t, "90", rows[0]["current_value"])
	assert.Equal(t, "0", rows[0]["compliant"])
	assert.Equal(t, "2026-03-05T10:30:00Z", rows[0]["last_checked"])

	// Second row: compliant
	assert.Equal(t, "AuditLogonEvents", rows[1]["setting_name"])
	assert.Equal(t, "audit", rows[1]["category"])
	assert.Equal(t, "cis-audit", rows[1]["policy_name"])
	assert.Equal(t, "17.5.1", rows[1]["cis_ref"])
	assert.Equal(t, "3", rows[1]["desired_value"])
	assert.Equal(t, "3", rows[1]["current_value"])
	assert.Equal(t, "1", rows[1]["compliant"])
	assert.Equal(t, "2026-03-05T10:30:00Z", rows[1]["last_checked"])
}

func TestSetCache(t *testing.T) {
	cache = nil
	require.Nil(t, cache)

	c := enf.NewComplianceCache()
	SetCache(c)
	require.NotNil(t, cache)

	// Verify it's the same cache
	c.Update([]enf.ComplianceRecord{{SettingName: "test"}})
	rows, err := Generate(context.Background(), table.QueryContext{})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "test", rows[0]["setting_name"])
}
