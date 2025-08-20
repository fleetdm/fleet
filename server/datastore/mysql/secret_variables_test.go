package mysql

import (
	"context"
	"sort"
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
		{"CreateSecretVariable", testCreateSecretVariable},
		{"ListSecretVariables", testListSecretVariables},
		{"DeleteSecretVariable", testDeleteSecretVariable},
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

func testCreateSecretVariable(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		name := "test_secret_" + t.Name()
		value := "secret_value"

		id, err := ds.CreateSecretVariable(ctx, name, value)
		require.NoError(t, err)
		require.NotZero(t, id)

		var storedValue string
		err = ds.writer(ctx).QueryRowContext(ctx,
			`SELECT value FROM secret_variables WHERE id = ?`, id).Scan(&storedValue)
		require.NoError(t, err)
		require.NotEmpty(t, storedValue)
	})

	t.Run("duplicate name error", func(t *testing.T) {
		ctx := context.Background()
		name := "test_secret_duplicate_" + t.Name()
		value := "secret_value"

		id1, err := ds.CreateSecretVariable(ctx, name, value)
		require.NoError(t, err)
		require.NotZero(t, id1)

		id2, err := ds.CreateSecretVariable(ctx, name, value)
		require.Error(t, err)
		var aee fleet.AlreadyExistsError
		require.ErrorAs(t, err, &aee)
		require.Zero(t, id2)
	})
}

func testListSecretVariables(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	createTestSecret := func(name, value string) uint {
		id, err := ds.CreateSecretVariable(ctx, name, value)
		require.NoError(t, err)
		return id
	}

	t.Run("list all secrets", func(t *testing.T) {
		name1 := "test_secret1_" + t.Name()
		name2 := "test_secret2_" + t.Name()
		id1 := createTestSecret(name1, "value1")
		id2 := createTestSecret(name2, "value2")

		secrets, meta, count, err := ds.ListSecretVariables(ctx, fleet.ListOptions{})
		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.Nil(t, meta)
		require.Len(t, secrets, 2)
		sort.Slice(secrets, func(i, j int) bool {
			return secrets[i].ID < secrets[j].ID
		})
		require.Equal(t, id1, secrets[0].ID)
		require.Equal(t, name1, secrets[0].Name)
		require.NotZero(t, secrets[0].UpdatedAt)
		require.Equal(t, id2, secrets[1].ID)
		require.Equal(t, name2, secrets[1].Name)
		require.NotZero(t, secrets[1].UpdatedAt)

		_, err = ds.DeleteSecretVariable(ctx, id1)
		require.NoError(t, err)
		_, err = ds.DeleteSecretVariable(ctx, id2)
		require.NoError(t, err)
	})

	t.Run("list with search query", func(t *testing.T) {
		name1 := "test_secret_unique_" + t.Name()
		name2 := "test_other_" + t.Name()
		id1 := createTestSecret(name1, "value1")
		id2 := createTestSecret(name2, "value2")

		opt := fleet.ListOptions{MatchQuery: "unique"}
		secrets, meta, count, err := ds.ListSecretVariables(ctx, opt)

		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Nil(t, meta)
		require.Len(t, secrets, 1)
		require.Equal(t, id1, secrets[0].ID)
		require.Equal(t, name1, secrets[0].Name)
		require.NotZero(t, secrets[0].UpdatedAt)

		_, err = ds.DeleteSecretVariable(ctx, id1)
		require.NoError(t, err)
		_, err = ds.DeleteSecretVariable(ctx, id2)
		require.NoError(t, err)
	})

	t.Run("list with pagination", func(t *testing.T) {
		name1 := "test_secret_pag1_" + t.Name()
		name2 := "test_secret_pag2_" + t.Name()
		id1 := createTestSecret(name1, "value1")
		id2 := createTestSecret(name2, "value2")
		opt := fleet.ListOptions{
			PerPage:         1,
			Page:            0,
			IncludeMetadata: true,
		}
		secrets, meta, count, err := ds.ListSecretVariables(ctx, opt)
		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.NotNil(t, meta)
		require.True(t, meta.HasNextResults)
		require.False(t, meta.HasPreviousResults)
		require.EqualValues(t, meta.TotalResults, count)
		require.Len(t, secrets, 1)
		require.Equal(t, id1, secrets[0].ID)
		require.Equal(t, name1, secrets[0].Name)
		require.NotZero(t, secrets[0].UpdatedAt)

		opt = fleet.ListOptions{
			PerPage:         1,
			Page:            1,
			IncludeMetadata: true,
		}
		secrets, meta, count, err = ds.ListSecretVariables(ctx, opt)
		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.NotNil(t, meta)
		require.False(t, meta.HasNextResults)
		require.True(t, meta.HasPreviousResults)
		require.EqualValues(t, meta.TotalResults, count)
		require.Len(t, secrets, 1)
		require.Equal(t, id2, secrets[0].ID)
		require.Equal(t, name2, secrets[0].Name)
		require.NotZero(t, secrets[0].UpdatedAt)

		_, err = ds.DeleteSecretVariable(ctx, id1)
		require.NoError(t, err)
		_, err = ds.DeleteSecretVariable(ctx, id2)
		require.NoError(t, err)
	})

	t.Run("list empty result", func(t *testing.T) {
		opt := fleet.ListOptions{
			MatchQuery:      "nonexistent_" + t.Name(),
			IncludeMetadata: true,
		}
		secrets, meta, count, err := ds.ListSecretVariables(ctx, opt)
		require.NoError(t, err)
		require.Equal(t, 0, count)
		require.NotNil(t, meta)
		require.False(t, meta.HasPreviousResults)
		require.False(t, meta.HasNextResults)
		require.Zero(t, meta.TotalResults)
		require.Empty(t, secrets)
	})
}

func testDeleteSecretVariable(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	deletedName, err := ds.DeleteSecretVariable(ctx, 999)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
	require.Empty(t, deletedName)

	id1, err := ds.CreateSecretVariable(ctx, "name1", "value1")
	require.NoError(t, err)

	deletedName, err = ds.DeleteSecretVariable(ctx, 999)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
	require.Empty(t, deletedName)

	id2, err := ds.CreateSecretVariable(ctx, "name2", "value2")
	require.NoError(t, err)

	deletedName, err = ds.DeleteSecretVariable(ctx, id1)
	require.NoError(t, err)
	require.Equal(t, "name1", deletedName)

	secrets, _, _, err := ds.ListSecretVariables(ctx, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	require.Equal(t, secrets[0].ID, id2)
}
