package datastore

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testApplyOsqueryOptions(t *testing.T, ds kolide.Datastore) {
	expectedOpts := &kolide.OptionsSpec{
		Config: json.RawMessage(`{"foo": "bar"}`),
		Overrides: kolide.OptionsOverrides{
			Platforms: map[string]json.RawMessage{
				"darwin": json.RawMessage(`{"froob": "ling"}`),
			},
		},
	}

	err := ds.ApplyOptions(expectedOpts)
	require.Nil(t, err)

	retrievedOpts, err := ds.GetOptions()
	require.Nil(t, err)
	assert.Equal(t, expectedOpts, retrievedOpts)

	// Re-apply and verify everything has been replaced.
	expectedOpts = &kolide.OptionsSpec{
		Config: json.RawMessage(`{"blue": "smurf"}`),
		Overrides: kolide.OptionsOverrides{
			Platforms: map[string]json.RawMessage{
				"linux": json.RawMessage(`{"transitive": "nightfall"}`),
			},
		},
	}

	err = ds.ApplyOptions(expectedOpts)
	require.Nil(t, err)

	retrievedOpts, err = ds.GetOptions()
	require.Nil(t, err)
	assert.Equal(t, expectedOpts, retrievedOpts)
}

func testApplyOsqueryOptionsNoOverrides(t *testing.T, ds kolide.Datastore) {
	expectedOpts := &kolide.OptionsSpec{
		Config: json.RawMessage(`{}`),
	}

	err := ds.ApplyOptions(expectedOpts)
	require.Nil(t, err)

	retrievedOpts, err := ds.GetOptions()
	require.Nil(t, err)
	assert.Equal(t, expectedOpts.Config, retrievedOpts.Config)
	assert.Empty(t, retrievedOpts.Overrides.Platforms)
}

func testOsqueryOptionsForHost(t *testing.T, ds kolide.Datastore) {
	defaultOpts := json.RawMessage(`{"foo": "bar"}`)
	darwinOpts := json.RawMessage(`{"darwin": "macintosh"}`)
	linuxOpts := json.RawMessage(`{"linux": "FOSS"}`)
	expectedOpts := &kolide.OptionsSpec{
		Config: defaultOpts,
		Overrides: kolide.OptionsOverrides{
			Platforms: map[string]json.RawMessage{
				"darwin": darwinOpts,
				"linux":  linuxOpts,
			},
		},
	}

	err := ds.ApplyOptions(expectedOpts)
	require.Nil(t, err)

	var testCases = []struct {
		host         kolide.Host
		expectedOpts json.RawMessage
	}{
		{kolide.Host{Platform: "windows"}, defaultOpts},
		{kolide.Host{Platform: "linux"}, linuxOpts},
		{kolide.Host{Platform: "darwin"}, darwinOpts},
		{kolide.Host{Platform: "some_other_platform"}, defaultOpts},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			opts, err := ds.OptionsForPlatform(tt.host.Platform)
			require.Nil(t, err)
			assert.Equal(t, tt.expectedOpts, opts)
		})
	}
}
