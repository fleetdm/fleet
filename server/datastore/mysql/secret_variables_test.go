package mysql

import (
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
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
		{"ExpandHostSecrets", testExpandHostSecrets},
		{"CreateSecretVariable", testCreateSecretVariable},
		{"ListSecretVariables", testListSecretVariables},
		{"DeleteSecretVariable", testDeleteSecretVariable},
		{"DeleteUsedSecretVariable", testDeleteUsedSecretVariable},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testUpsertSecretVariables(t *testing.T, ds *Datastore) {
	ctx := t.Context()

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
	ctx := t.Context()

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

	xmlValidSecret := `<?xml>${FLEET_SECRET_VALID_XML}</xml>`                  //nolint:gosec // G101: test fixture, not a credential
	jsonValidSecret := `{"pwd":"${FLEET_SECRET_VALID_JSON}"}`                  //nolint:gosec // G101: test fixture, not a credential
	jsonValidSecretWhitespace := "\n  " + `{"pwd":"$FLEET_SECRET_VALID_JSON"}` //nolint:gosec // G101: test fixture, not a credential

	ctx := t.Context()

	secretMap := map[string]string{
		"VALID":      "testValue1",
		"ALSO_VALID": "testValue2",
		"VALID_XML":  "<tag>value & more</tag>",
		"VALID_JSON": `p"<&'\d`,
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

	expanded, err = ds.ExpandEmbeddedSecrets(ctx, xmlValidSecret)
	require.NoError(t, err)
	expectedXMLExpansion := `<?xml>&lt;tag&gt;value &amp; more&lt;/tag&gt;</xml>`
	require.Equal(t, expectedXMLExpansion, expanded)

	// JSON documents get JSON-escaped secret values so the result remains valid JSON.
	expanded, err = ds.ExpandEmbeddedSecrets(ctx, jsonValidSecret)
	require.NoError(t, err)
	var parsed map[string]string
	require.NoError(t, json.Unmarshal([]byte(expanded), &parsed))
	require.Equal(t, `p"<&'\d`, parsed["pwd"])

	// Leading whitespace before the opening brace should still be detected as JSON.
	expanded, err = ds.ExpandEmbeddedSecrets(ctx, jsonValidSecretWhitespace)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(expanded)), &parsed))
	require.Equal(t, `p"<&'\d`, parsed["pwd"])
}

func testExpandHostSecrets(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create a host
	host, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("host-secrets-test"),
		NodeKey:         ptr.String("host-secrets-test-key"),
		UUID:            "host-secrets-test-uuid",
		Hostname:        "host-secrets-test-hostname",
	})
	require.NoError(t, err)

	// Set a recovery lock password for this host
	password := "TEST-PASS-1234"
	err = ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{
		{HostUUID: host.UUID, Password: password},
	})
	require.NoError(t, err)

	t.Run("no host secrets in document", func(t *testing.T) {
		doc := "This document has no host secrets. $FLEET_SECRET_SOMETHING ${OTHER_VAR}"
		expanded, err := ds.ExpandHostSecrets(ctx, doc, host.UUID)
		require.NoError(t, err)
		assert.Equal(t, doc, expanded) // unchanged
	})

	t.Run("expand recovery lock password", func(t *testing.T) {
		doc := `<dict><key>NewPassword</key><string>$FLEET_HOST_SECRET_RECOVERY_LOCK_PASSWORD</string></dict>`
		expected := `<dict><key>NewPassword</key><string>TEST-PASS-1234</string></dict>`
		expanded, err := ds.ExpandHostSecrets(ctx, doc, host.UUID)
		require.NoError(t, err)
		assert.Equal(t, expected, expanded)
	})

	t.Run("expand with braces syntax", func(t *testing.T) {
		doc := `Password: ${FLEET_HOST_SECRET_RECOVERY_LOCK_PASSWORD}`
		expected := `Password: TEST-PASS-1234`
		expanded, err := ds.ExpandHostSecrets(ctx, doc, host.UUID)
		require.NoError(t, err)
		assert.Equal(t, expected, expanded)
	})

	t.Run("unknown host secret type", func(t *testing.T) {
		doc := `<key>Value</key><string>$FLEET_HOST_SECRET_UNKNOWN_TYPE</string>`
		_, err := ds.ExpandHostSecrets(ctx, doc, host.UUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown host secret type")
	})

	t.Run("non-existent host", func(t *testing.T) {
		doc := `<string>$FLEET_HOST_SECRET_RECOVERY_LOCK_PASSWORD</string>`
		_, err := ds.ExpandHostSecrets(ctx, doc, "non-existent-uuid")
		require.Error(t, err)
	})

	t.Run("host without recovery lock password", func(t *testing.T) {
		// Create another host without a recovery lock password
		host2, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   ptr.String("host-no-password"),
			NodeKey:         ptr.String("host-no-password-key"),
			UUID:            "host-no-password-uuid",
			Hostname:        "host-no-password-hostname",
		})
		require.NoError(t, err)

		doc := `<string>$FLEET_HOST_SECRET_RECOVERY_LOCK_PASSWORD</string>`
		_, err = ds.ExpandHostSecrets(ctx, doc, host2.UUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "getting recovery lock password")
	})

	t.Run("expand recovery lock password with XML special characters", func(t *testing.T) {
		// Create a host with a password containing XML special characters
		hostXML, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   ptr.String("host-xml-escape-test"),
			NodeKey:         ptr.String("host-xml-escape-test-key"),
			UUID:            "host-xml-escape-test-uuid",
			Hostname:        "host-xml-escape-test-hostname",
		})
		require.NoError(t, err)

		// Set a password with XML special characters: & < > " '
		passwordWithSpecialChars := `Pass&word<with>special"chars'`
		err = ds.SetHostsRecoveryLockPasswords(ctx, []fleet.HostRecoveryLockPasswordPayload{
			{HostUUID: hostXML.UUID, Password: passwordWithSpecialChars},
		})
		require.NoError(t, err)

		// When expanded in an XML document, special characters should be escaped
		doc := `<dict><key>NewPassword</key><string>$FLEET_HOST_SECRET_RECOVERY_LOCK_PASSWORD</string></dict>`
		expected := `<dict><key>NewPassword</key><string>Pass&amp;word&lt;with&gt;special&#34;chars&#39;</string></dict>`
		expanded, err := ds.ExpandHostSecrets(ctx, doc, hostXML.UUID)
		require.NoError(t, err)
		assert.Equal(t, expected, expanded)

		// Non-XML documents should not escape the characters
		docNonXML := `Password: $FLEET_HOST_SECRET_RECOVERY_LOCK_PASSWORD`
		expandedNonXML, err := ds.ExpandHostSecrets(ctx, docNonXML, hostXML.UUID)
		require.NoError(t, err)
		assert.Equal(t, `Password: Pass&word<with>special"chars'`, expandedNonXML)
	})

	t.Run("mdm unlock token expansion", func(t *testing.T) {
		// Create a host with an MDM unlock token
		hostMDM, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   ptr.String("host-mdm-unlock-token-test"),
			NodeKey:         ptr.String("host-mdm-unlock-token-test-key"),
			UUID:            "host-mdm-unlock-token-test-uuid",
			Hostname:        "host-mdm-unlock-token-test-hostname",
			Platform:        "ios",
		})
		require.NoError(t, err)

		unlockToken := "TEST-MDM-UNLOCK-TOKEN" // nolint:gosec // G101: this is a constant identifier, not a credential
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `INSERT INTO nano_devices (id, unlock_token, authenticate, platform) VALUES (?, ?, 'fake-auth', 'ios')`, hostMDM.UUID, unlockToken)
			require.NoError(t, err)
			_, err = q.ExecContext(ctx, `INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, last_seen_at) VALUES (?, ?, 'Device', 'fake-topic', 'fake-push-magic', 'fake-token-hex', NOW())`, hostMDM.UUID, hostMDM.UUID)
			return err
		})

		b64Encoded := base64.StdEncoding.EncodeToString([]byte(unlockToken))
		doc := `<string>$FLEET_HOST_SECRET_MDM_UNLOCK_TOKEN</string>`
		expected := `<string>` + b64Encoded + `</string>`
		expanded, err := ds.ExpandHostSecrets(ctx, doc, hostMDM.UUID)
		require.NoError(t, err)
		assert.Equal(t, expected, expanded)
	})
}

func testCreateSecretVariable(t *testing.T, ds *Datastore) {
	t.Run("successful creation", func(t *testing.T) {
		ctx := t.Context()

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
		ctx := t.Context()

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
	ctx := t.Context()

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
	ctx := t.Context()

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

func testDeleteUsedSecretVariable(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	foobarTeam, err := ds.NewTeam(ctx, &fleet.Team{
		Name: "Foobar",
	})
	require.NoError(t, err)

	id, err := ds.CreateSecretVariable(ctx, "FOOBAR", "123")
	require.NoError(t, err)
	id2, err := ds.CreateSecretVariable(ctx, "OTHER", "123")
	require.NoError(t, err)

	t.Run("apple configuration profiles", func(t *testing.T) {
		// Create Apple configuration profile in "No team" that uses the variable.
		appleProfile, err := ds.NewMDMAppleConfigProfile(ctx,
			fleet.MDMAppleConfigProfile{
				Name:         "Name0",
				Identifier:   "Identifier0",
				Mobileconfig: []byte("$FLEET_SECRET_FOOBAR"),
			},
			nil,
		)
		require.NoError(t, err)

		// Attempt to delete the variable, should fail.
		_, err = ds.DeleteSecretVariable(ctx, id)
		require.Error(t, err)
		var s *fleet.SecretUsedError
		require.ErrorAs(t, err, &s)
		require.Equal(t, "FOOBAR", s.SecretName)
		require.Equal(t, "apple_profile", s.Entity.Type)
		require.Equal(t, "Name0", s.Entity.Name)
		require.Equal(t, "No team", s.Entity.TeamName)

		// Deleting an unused variable is allowed.
		_, err = ds.DeleteSecretVariable(ctx, id2)
		require.NoError(t, err)

		// Delete the profile.
		err = ds.DeleteMDMAppleConfigProfile(ctx, appleProfile.ProfileUUID)
		require.NoError(t, err)

		// Create Apple configuration profile in team "Foobar" that uses the variable.
		appleProfile, err = ds.NewMDMAppleConfigProfile(ctx,
			fleet.MDMAppleConfigProfile{
				Name:         "Name0",
				Identifier:   "Identifier0",
				Mobileconfig: []byte("$FLEET_SECRET_FOOBAR"),
				TeamID:       &foobarTeam.ID,
			},
			nil,
		)
		require.NoError(t, err)

		// Attempt to delete the variable, should fail.
		_, err = ds.DeleteSecretVariable(ctx, id)
		require.Error(t, err)
		s = &fleet.SecretUsedError{}
		require.ErrorAs(t, err, &s)
		require.Equal(t, "FOOBAR", s.SecretName)
		require.Equal(t, "apple_profile", s.Entity.Type)
		require.Equal(t, "Name0", s.Entity.Name)
		require.Equal(t, "Foobar", s.Entity.TeamName)

		// Delete the profile.
		err = ds.DeleteMDMAppleConfigProfile(ctx, appleProfile.ProfileUUID)
		require.NoError(t, err)
	})

	t.Run("apple declarations", func(t *testing.T) {
		// Create Apple declaration "No team" that uses the variable.
		appleDeclaration, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			Identifier: "decl-1",
			Name:       "decl-1",
			RawJSON:    json.RawMessage(`{"Identifier": "${FLEET_SECRET_FOOBAR}"}`),
		}, nil)
		require.NoError(t, err)

		// Attempt to delete the variable, should fail.
		_, err = ds.DeleteSecretVariable(ctx, id)
		require.Error(t, err)
		s := &fleet.SecretUsedError{}
		require.ErrorAs(t, err, &s)
		require.Equal(t, "FOOBAR", s.SecretName)
		require.Equal(t, "apple_declaration", s.Entity.Type)
		require.Equal(t, "decl-1", s.Entity.Name)
		require.Equal(t, "No team", s.Entity.TeamName)

		err = ds.DeleteMDMAppleDeclaration(ctx, appleDeclaration.DeclarationUUID)
		require.NoError(t, err)

		// Create Apple declaration "Foobar" that uses the variable.
		appleDeclaration, err = ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			Identifier: "decl-1",
			Name:       "decl-1",
			RawJSON:    json.RawMessage(`{"Identifier": "${FLEET_SECRET_FOOBAR}"}`),
			TeamID:     &foobarTeam.ID,
		}, nil)
		require.NoError(t, err)

		// Attempt to delete the variable, should fail.
		_, err = ds.DeleteSecretVariable(ctx, id)
		require.Error(t, err)
		s = &fleet.SecretUsedError{}
		require.ErrorAs(t, err, &s)
		require.Equal(t, "FOOBAR", s.SecretName)
		require.Equal(t, "apple_declaration", s.Entity.Type)
		require.Equal(t, "decl-1", s.Entity.Name)
		require.Equal(t, "Foobar", s.Entity.TeamName)

		err = ds.DeleteMDMAppleDeclaration(ctx, appleDeclaration.DeclarationUUID)
		require.NoError(t, err)
	})

	t.Run("windows profiles", func(t *testing.T) {
		// Create Windows profile "No team" that uses the variable.
		windowsProfile, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
			Name:   "zoo",
			TeamID: nil,
			SyncML: []byte("<Replace>$FLEET_SECRET_FOOBAR</Replace>"),
		}, nil)
		require.NoError(t, err)

		// Attempt to delete the variable, should fail.
		_, err = ds.DeleteSecretVariable(ctx, id)
		require.Error(t, err)
		s := &fleet.SecretUsedError{}
		require.ErrorAs(t, err, &s)
		require.Equal(t, "FOOBAR", s.SecretName)
		require.Equal(t, "windows_profile", s.Entity.Type)
		require.Equal(t, "zoo", s.Entity.Name)
		require.Equal(t, "No team", s.Entity.TeamName)

		err = ds.DeleteMDMWindowsConfigProfile(ctx, windowsProfile.ProfileUUID)
		require.NoError(t, err)

		// Create Windows profile in "Foobar" team that uses the variable.
		windowsProfile, err = ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
			Name:   "zoo",
			TeamID: &foobarTeam.ID,
			SyncML: []byte("<Replace>$FLEET_SECRET_FOOBAR</Replace>"),
		}, nil)
		require.NoError(t, err)

		// Attempt to delete the variable, should fail.
		_, err = ds.DeleteSecretVariable(ctx, id)
		require.Error(t, err)
		s = &fleet.SecretUsedError{}
		require.ErrorAs(t, err, &s)
		require.Equal(t, "FOOBAR", s.SecretName)
		require.Equal(t, "windows_profile", s.Entity.Type)
		require.Equal(t, "zoo", s.Entity.Name)
		require.Equal(t, "Foobar", s.Entity.TeamName)

		err = ds.DeleteMDMWindowsConfigProfile(ctx, windowsProfile.ProfileUUID)
		require.NoError(t, err)
	})

	t.Run("scripts", func(t *testing.T) {
		// Create a script in "No team" that uses a variable
		script, err := ds.NewScript(ctx, &fleet.Script{
			Name:           "foobar.sh",
			ScriptContents: "echo $FLEET_SECRET_FOOBAR",
		})
		require.NoError(t, err)

		// Attempt to delete the variable, should fail.
		_, err = ds.DeleteSecretVariable(ctx, id)
		require.Error(t, err)
		s := &fleet.SecretUsedError{}
		require.ErrorAs(t, err, &s)
		require.Equal(t, "FOOBAR", s.SecretName)
		require.Equal(t, "script", s.Entity.Type)
		require.Equal(t, "foobar.sh", s.Entity.Name)
		require.Equal(t, "No team", s.Entity.TeamName)

		err = ds.DeleteScript(ctx, script.ID)
		require.NoError(t, err)

		// Create a script in team "Foobar" that uses a variable
		script, err = ds.NewScript(ctx, &fleet.Script{
			Name:           "foobar.sh",
			ScriptContents: "echo $FLEET_SECRET_FOOBAR",
			TeamID:         &foobarTeam.ID,
		})
		require.NoError(t, err)

		// Attempt to delete the variable, should fail.
		_, err = ds.DeleteSecretVariable(ctx, id)
		require.Error(t, err)
		s = &fleet.SecretUsedError{}
		require.ErrorAs(t, err, &s)
		require.Equal(t, "FOOBAR", s.SecretName)
		require.Equal(t, "script", s.Entity.Type)
		require.Equal(t, "foobar.sh", s.Entity.Name)
		require.Equal(t, "Foobar", s.Entity.TeamName)

		err = ds.DeleteScript(ctx, script.ID)
		require.NoError(t, err)
	})

	// Finally attempt to delete the secret again now that no entity is using it.
	_, err = ds.DeleteSecretVariable(ctx, id)
	require.NoError(t, err)
}
