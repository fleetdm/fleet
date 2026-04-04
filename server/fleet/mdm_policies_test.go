package fleet

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMDMPolicyCheckOperator_ValidConstants(t *testing.T) {
	operators := []MDMPolicyCheckOperator{
		MDMPolicyCheckEq, MDMPolicyCheckNeq,
		MDMPolicyCheckGt, MDMPolicyCheckLt, MDMPolicyCheckGte, MDMPolicyCheckLte,
		MDMPolicyCheckContains, MDMPolicyCheckNotContains,
		MDMPolicyCheckVersionGte, MDMPolicyCheckVersionLte,
		MDMPolicyCheckExists, MDMPolicyCheckNotExists,
	}
	require.Len(t, operators, 12)
	seen := make(map[MDMPolicyCheckOperator]struct{})
	for _, op := range operators {
		assert.NotEmpty(t, string(op))
		_, dup := seen[op]
		assert.False(t, dup, "duplicate operator: %s", op)
		seen[op] = struct{}{}
	}
}

func TestMDMPolicyCheckSource_ValidConstants(t *testing.T) {
	sources := []MDMPolicyCheckSource{
		MDMPolicySourceDeviceInformation,
		MDMPolicySourceSecurityInfo,
		MDMPolicySourceInstalledApplicationList,
	}
	require.Len(t, sources, 3)
	seen := make(map[MDMPolicyCheckSource]struct{})
	for _, src := range sources {
		assert.NotEmpty(t, string(src))
		_, dup := seen[src]
		assert.False(t, dup, "duplicate source: %s", src)
		seen[src] = struct{}{}
	}
}

func TestMDMPolicyDefinition_JSONRoundTrip(t *testing.T) {
	def := MDMPolicyDefinition{
		Checks: []MDMPolicyCheck{
			{Field: "DeviceInformation.OSVersion", Operator: MDMPolicyCheckVersionGte, Expected: "17.0", Source: MDMPolicySourceDeviceInformation},
			{Field: "SecurityInfo.PasscodePresent", Operator: MDMPolicyCheckEq, Expected: "true", Source: MDMPolicySourceSecurityInfo},
			{Field: "InstalledApplicationList.com.apple.Safari.ShortVersion", Operator: MDMPolicyCheckExists, Expected: "", Source: MDMPolicySourceInstalledApplicationList},
		},
	}
	data, err := json.Marshal(def)
	require.NoError(t, err)

	var decoded MDMPolicyDefinition
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, def, decoded)
}

func TestMDMPolicyDefinition_EmptyChecks_JSONRoundTrip(t *testing.T) {
	def := MDMPolicyDefinition{Checks: []MDMPolicyCheck{}}
	data, err := json.Marshal(def)
	require.NoError(t, err)

	var decoded MDMPolicyDefinition
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Empty(t, decoded.Checks)
}

func TestDeviceStateEntry_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	entry := DeviceStateEntry{
		Value:      "17.4.1",
		Source:     "mdm_poll",
		ObservedAt: now,
	}
	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var decoded DeviceStateEntry
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, entry.Value, decoded.Value)
	assert.Equal(t, entry.Source, decoded.Source)
	assert.True(t, entry.ObservedAt.Equal(decoded.ObservedAt))
}
