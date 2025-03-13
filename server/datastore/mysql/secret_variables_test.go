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
		{"ValidateEmbeddedSecrets", testValidateEmbeddedSecrets},
		{"ExpandEmbeddedSecrets", testExpandEmbeddedSecrets},
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

	// Update a secret and insert a new one
	secretMap["test2"] = "newTestValue2"
	secretMap["test4"] = "testValue4"
	err = ds.UpsertSecretVariables(ctx, []fleet.SecretVariable{
		{Name: "test2", Value: secretMap["test2"]},
		{Name: "test4", Value: secretMap["test4"]},
	})
	assert.NoError(t, err)
	results, err = ds.GetSecretVariables(ctx, []string{"test2", "test4"})
	assert.NoError(t, err)
	require.Len(t, results, 2)
	for _, result := range results {
		assert.Equal(t, secretMap[result.Name], result.Value)
	}

	// Make sure updated_at timestamp does not change when we update a secret with the same value
	original, err := ds.GetSecretVariables(ctx, []string{"test1"})
	require.NoError(t, err)
	require.Len(t, original, 1)
	err = ds.UpsertSecretVariables(ctx, []fleet.SecretVariable{
		{Name: "test1", Value: secretMap["test1"]},
	})
	require.NoError(t, err)
	updated, err := ds.GetSecretVariables(ctx, []string{"test1"})
	require.NoError(t, err)
	require.Len(t, original, 1)
	assert.Equal(t, original[0], updated[0])
}

func testValidateEmbeddedSecrets(t *testing.T, ds *Datastore) {
	noSecrets := `
This document contains to fleet secrets.
$FLEET_VAR_XX $HOSTNAME ${SOMETHING_ELSE}
`

	validSecret := `
This document contains a valid ${FLEET_SECRET_VALID}.
Another $FLEET_SECRET_ALSO_VALID.
`

	invalidSecret := `
This document contains a secret not stored in the database.
Hello doc${FLEET_SECRET_INVALID}. $FLEET_SECRET_ALSO_INVALID
`
	ctx := context.Background()
	secretMap := map[string]string{
		"VALID":      "testValue1",
		"ALSO_VALID": "testValue2",
	}

	secrets := make([]fleet.SecretVariable, 0, len(secretMap))
	for name, value := range secretMap {
		secrets = append(secrets, fleet.SecretVariable{Name: name, Value: value})
	}

	err := ds.UpsertSecretVariables(ctx, secrets)
	require.NoError(t, err)

	err = ds.ValidateEmbeddedSecrets(ctx, []string{noSecrets})
	require.NoError(t, err)

	err = ds.ValidateEmbeddedSecrets(ctx, []string{validSecret})
	require.NoError(t, err)

	err = ds.ValidateEmbeddedSecrets(ctx, []string{noSecrets, validSecret})
	require.NoError(t, err)

	err = ds.ValidateEmbeddedSecrets(ctx, []string{invalidSecret})
	require.ErrorContains(t, err, "$FLEET_SECRET_INVALID")
	require.ErrorContains(t, err, "$FLEET_SECRET_ALSO_INVALID")

	err = ds.ValidateEmbeddedSecrets(ctx, []string{noSecrets, validSecret, invalidSecret})
	require.ErrorContains(t, err, "$FLEET_SECRET_INVALID")
	require.ErrorContains(t, err, "$FLEET_SECRET_ALSO_INVALID")
}

func testExpandEmbeddedSecrets(t *testing.T, ds *Datastore) {
	noSecrets := `
This document contains no fleet secrets.
$FLEET_VAR_XX $HOSTNAME ${SOMETHING_ELSE}
`

	validSecret := `
This document contains a valid ${FLEET_SECRET_VALID}.
Another $FLEET_SECRET_ALSO_VALID.
`
	validSecretExpanded := `
This document contains a valid testValue1.
Another testValue2.
`

	invalidSecret := `
This document contains a secret not stored in the database.
Hello doc${FLEET_SECRET_INVALID}. $FLEET_SECRET_ALSO_INVALID
`

	ctx := context.Background()
	secretMap := map[string]string{
		"VALID":      "testValue1",
		"ALSO_VALID": "testValue2",
	}

	secrets := make([]fleet.SecretVariable, 0, len(secretMap))
	for name, value := range secretMap {
		secrets = append(secrets, fleet.SecretVariable{Name: name, Value: value})
	}

	err := ds.UpsertSecretVariables(ctx, secrets)
	require.NoError(t, err)

	expanded, err := ds.ExpandEmbeddedSecrets(ctx, noSecrets)
	require.NoError(t, err)
	require.Equal(t, noSecrets, expanded)
	expanded, secretsUpdatedAt, err := ds.ExpandEmbeddedSecretsAndUpdatedAt(ctx, noSecrets)
	require.NoError(t, err)
	require.Equal(t, noSecrets, expanded)
	assert.Nil(t, secretsUpdatedAt)

	expanded, err = ds.ExpandEmbeddedSecrets(ctx, validSecret)
	require.NoError(t, err)
	require.Equal(t, validSecretExpanded, expanded)
	expanded, secretsUpdatedAt, err = ds.ExpandEmbeddedSecretsAndUpdatedAt(ctx, validSecret)
	require.NoError(t, err)
	require.Equal(t, validSecretExpanded, expanded)
	assert.NotNil(t, secretsUpdatedAt)

	_, err = ds.ExpandEmbeddedSecrets(ctx, invalidSecret)
	require.ErrorContains(t, err, "$FLEET_SECRET_INVALID")
	require.ErrorContains(t, err, "$FLEET_SECRET_ALSO_INVALID")
}
