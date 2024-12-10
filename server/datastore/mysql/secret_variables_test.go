package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretVariables(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"UpsertSecretVariables", testUpsertSecretVariables},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testUpsertSecretVariables(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	err := ds.UpsertSecretVariables(ctx, nil)
	assert.NoError(t, err)
	results, err := ds.GetSecretVariables(ctx, nil)
	assert.NoError(t, err)
	assert.Empty(t, results)

	secretMap := map[string]string{
		"test1": "testValue1",
		"test2": "testValue2",
		"test3": "testValue3",
	}
	createExpectedSecrets := func() []fleet.SecretVariable {
		secrets := make([]fleet.SecretVariable, 0, len(secretMap))
		for name, value := range secretMap {
			secrets = append(secrets, fleet.SecretVariable{Name: name, Value: value})
		}
		return secrets
	}
	secrets := createExpectedSecrets()
	err = ds.UpsertSecretVariables(ctx, secrets)
	assert.NoError(t, err)

	results, err = ds.GetSecretVariables(ctx, []string{"test1", "test2", "test3"})
	assert.NoError(t, err)
	assert.Len(t, results, 3)
	for _, result := range results {
		assert.Equal(t, secretMap[result.Name], result.Value)
	}

	// Update a secret
	secretMap["test2"] = "newTestValue2"
	err = ds.UpsertSecretVariables(ctx, []fleet.SecretVariable{
		{Name: "test2", Value: secretMap["test2"]},
	})
	assert.NoError(t, err)
	results, err = ds.GetSecretVariables(ctx, []string{"test2"})
	assert.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "test2", results[0].Name)
	assert.Equal(t, secretMap[results[0].Name], results[0].Value)

}
