package mysql

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSoftwareDiffDoesNothingIfNothingChanges(t *testing.T) {
	results, err := softwareDiff(
		1,
		[]fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
			{Name: "towel", Version: "42.0.0", Source: "apps"},
		},
		[]fleet.Software{
			{ID: 1, Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{ID: 2, Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
			{ID: 3, Name: "towel", Version: "42.0.0", Source: "apps"},
		},
		[]fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
			{Name: "towel", Version: "42.0.0", Source: "apps"},
		},
	)
	require.NoError(t, err)
	assert.Len(t, results.insertsSoftware, 0)
	assert.Len(t, results.insertsHostSoftware, 0)
	assert.Len(t, results.deletesHostSoftware, 0)
}

func TestSoftwareDiffAddsHostAndSoftware(t *testing.T) {
	results, err := softwareDiff(
		1,
		[]fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
			{Name: "towel", Version: "42.0.0", Source: "apps"},
		},
		[]fleet.Software{
			{ID: 1, Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{ID: 2, Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		},
		[]fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		},
	)
	require.NoError(t, err)
	assert.Len(t, results.insertsSoftware, 1)
	assert.Len(t, results.insertsHostSoftware, 0)
	assert.Len(t, results.deletesHostSoftware, 0)

	assert.Equal(t, results.insertsSoftware, [][]interface{}{[]interface{}{"towel", "42.0.0", "apps"}})
}

func TestSoftwareDiffAddsHostOnly(t *testing.T) {
	results, err := softwareDiff(
		1,
		[]fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
			{Name: "towel", Version: "42.0.0", Source: "apps"},
		},
		[]fleet.Software{
			{ID: 1, Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{ID: 2, Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
			{ID: 3, Name: "towel", Version: "42.0.0", Source: "apps"},
		},
		[]fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		},
	)
	require.NoError(t, err)
	assert.Len(t, results.insertsSoftware, 0)
	assert.Len(t, results.insertsHostSoftware, 2)
	assert.Len(t, results.deletesHostSoftware, 0)

	assert.Equal(t, results.insertsHostSoftware, []interface{}{uint(1), uint(3)})
}

func TestSoftwareDiffRemovesHost(t *testing.T) {
	results, err := softwareDiff(
		1,
		[]fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		},
		[]fleet.Software{
			{ID: 1, Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{ID: 2, Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
			{ID: 3, Name: "towel", Version: "42.0.0", Source: "apps"},
		},
		[]fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
			{Name: "towel", Version: "42.0.0", Source: "apps"},
		},
	)
	require.NoError(t, err)
	assert.Len(t, results.insertsSoftware, 0)
	assert.Len(t, results.insertsHostSoftware, 0)
	assert.Len(t, results.deletesHostSoftware, 1)

	assert.Equal(t, results.deletesHostSoftware, []interface{}{uint(3)})
}
