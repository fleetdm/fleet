package service

// Secret variable tests for the core (no-license) suite.
//
// Belongs here: secret variable CRUD, applying them via GitOps, detecting which
// are in use, and the permissions required to read or write them.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationTestSuite) TestSecretVariablesGitOps() {
	t := s.T()
	ctx := context.Background()

	u := &fleet.User{
		Name:  "GitOps",
		Email: "gitops1@example.com",
		// GitOps role is premium only, so we use the global admin role.
		GlobalRole: new(fleet.RoleAdmin),
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.ds.NewUser(ctx, u)
	require.NoError(t, err)
	s.setTokenForTest(t, "gitops1@example.com", test.GoodPassword)

	// Empty request
	req := fleet.CreateSecretVariablesRequest{}
	var resp fleet.CreateSecretVariablesResponse
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &resp)

	// Secret variable name too long
	req = fleet.CreateSecretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  strings.Repeat("a", 256),
				Value: "value",
			},
		},
	}
	httpResp := s.Do("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusUnprocessableEntity)
	assertBodyContains(t, httpResp, "secret variable name is too long")

	// Secret variable name empty
	req = fleet.CreateSecretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  "  ",
				Value: "value",
			},
		},
	}
	httpResp = s.Do("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusUnprocessableEntity)
	assertBodyContains(t, httpResp, "secret variable name cannot be empty")

	validName := strings.Repeat("G", 255)
	req = fleet.CreateSecretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  "FLEET_SECRET_" + validName,
				Value: "value",
			},
		},
	}
	// Do dry run
	req.DryRun = true
	idBeforeDryRun := s.lastActivityMatches("", "", 0)
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &resp)

	secrets, err := s.ds.GetSecretVariables(ctx, []string{validName})
	require.NoError(t, err)
	require.Empty(t, secrets)
	// A dry run persists nothing, so it must not emit any activity.
	require.Equal(t, idBeforeDryRun, s.lastActivityMatches("", "", 0))

	// Do real run: creating the variable emits a created_custom_variable activity.
	req.DryRun = false
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &resp)
	secrets, err = s.ds.GetSecretVariables(ctx, []string{validName})
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	assert.Equal(t, "value", secrets[0].Value)
	s.lastActivityMatches(
		fleet.ActivityCreatedCustomVariable{}.ActivityName(),
		fmt.Sprintf(`{"custom_variable_id":0,"custom_variable_name":%q}`, validName),
		0,
	)

	// Re-applying the same spec is a no-op and must not emit any activity.
	idAfterCreate := s.lastActivityMatches("", "", 0)
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &resp)
	require.Equal(t, idAfterCreate, s.lastActivityMatches("", "", 0))

	// Changing the value via the spec endpoint emits an updated_custom_variable activity.
	req.SecretVariables[0].Value = "new-value"
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &resp)
	secrets, err = s.ds.GetSecretVariables(ctx, []string{validName})
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	assert.Equal(t, "new-value", secrets[0].Value)
	s.lastActivityMatches(
		fleet.ActivityUpdatedCustomVariable{}.ActivityName(),
		fmt.Sprintf(`{"custom_variable_name":%q}`, validName),
		0,
	)
}

func (s *integrationTestSuite) TestSecretVariables() {
	t := s.T()
	ctx := context.Background()

	// Create a single secret variable.
	var csvr fleet.CreateSecretVariableResponse
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "NAME1",
		Value: "value1",
	}, http.StatusOK, &csvr)
	firstVariableID := csvr.ID
	require.NotZero(t, firstVariableID)

	// List (no-filtering).
	var lsvr fleet.ListSecretVariablesResponse
	s.DoJSON("GET", "/api/latest/fleet/custom_variables", fleet.ListSecretVariablesRequest{}, http.StatusOK, &lsvr)
	require.Equal(t, 1, lsvr.Count)
	require.Len(t, lsvr.CustomVariables, 1)
	require.NotZero(t, lsvr.CustomVariables[0].ID)
	require.Equal(t, "NAME1", lsvr.CustomVariables[0].Name)
	require.NotEmpty(t, lsvr.CustomVariables[0].UpdatedAt)

	// Make sure we can access the value internally.
	secretVariables, err := s.ds.GetSecretVariables(ctx, []string{"NAME1"})
	require.NoError(t, err)
	require.Len(t, secretVariables, 1)
	require.Equal(t, "NAME1", secretVariables[0].Name)
	require.Equal(t, "value1", secretVariables[0].Value)
	require.NotZero(t, secretVariables[0].UpdatedAt)

	// Creating the same variable should fail with conflict.
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "NAME1",
		Value: "value1",
	}, http.StatusConflict, &csvr)

	// Creating a variable with invalid name should fail with 422.
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "lowercase",
		Value: "foobar",
	}, http.StatusUnprocessableEntity, &csvr)
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "",
		Value: "foobar",
	}, http.StatusUnprocessableEntity, &csvr)
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  strings.Repeat("HA ", 255/3+1),
		Value: "foobar",
	}, http.StatusUnprocessableEntity, &csvr)
	// No server private key configured, should fail with 400.
	testSetEmptyPrivateKey = true
	defer func() {
		testSetEmptyPrivateKey = false
	}()
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "NAME2",
		Value: "foobar",
	}, http.StatusBadRequest, &csvr)

	testSetEmptyPrivateKey = false

	// Creating a variable with empty value should fail with 422.
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "ANOTHER_NAME",
		Value: "",
	}, http.StatusUnprocessableEntity, &csvr)

	// Creating a second variable.
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "ANOTHER_NAME",
		Value: "value2",
	}, http.StatusOK, &csvr)
	secondVariableID := csvr.ID
	require.NotZero(t, secondVariableID)

	// List (no-filtering) with pagination (first page).
	lsvr = fleet.ListSecretVariablesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/custom_variables", nil, http.StatusOK, &lsvr, "per_page", "1", "page", "0")
	require.Equal(t, 2, lsvr.Count)
	require.NotNil(t, lsvr.Meta)
	require.False(t, lsvr.Meta.HasPreviousResults)
	require.True(t, lsvr.Meta.HasNextResults)
	require.Len(t, lsvr.CustomVariables, 1)
	require.Equal(t, secondVariableID, lsvr.CustomVariables[0].ID)
	require.Equal(t, "ANOTHER_NAME", lsvr.CustomVariables[0].Name)
	require.NotEmpty(t, lsvr.CustomVariables[0].UpdatedAt)
	// List (no-filtering) with pagination (second page).
	lsvr = fleet.ListSecretVariablesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/custom_variables", nil, http.StatusOK, &lsvr, "per_page", "1", "page", "1")
	require.Equal(t, 2, lsvr.Count)
	require.NotNil(t, lsvr.Meta)
	require.True(t, lsvr.Meta.HasPreviousResults)
	require.False(t, lsvr.Meta.HasNextResults)
	require.Len(t, lsvr.CustomVariables, 1)
	require.Equal(t, firstVariableID, lsvr.CustomVariables[0].ID)
	require.Equal(t, "NAME1", lsvr.CustomVariables[0].Name)
	require.NotEmpty(t, lsvr.CustomVariables[0].UpdatedAt)
	// List (no-filtering) with pagination (one page, two secrets).
	// Must be ordered alphabetically.
	lsvr = fleet.ListSecretVariablesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/custom_variables", nil, http.StatusOK, &lsvr, "per_page", "20", "page", "0")
	require.Equal(t, 2, lsvr.Count)
	require.NotNil(t, lsvr.Meta)
	require.False(t, lsvr.Meta.HasPreviousResults)
	require.False(t, lsvr.Meta.HasNextResults)
	require.Len(t, lsvr.CustomVariables, 2)
	require.Equal(t, secondVariableID, lsvr.CustomVariables[0].ID)
	require.Equal(t, "ANOTHER_NAME", lsvr.CustomVariables[0].Name)
	require.NotEmpty(t, lsvr.CustomVariables[0].UpdatedAt)
	require.Equal(t, firstVariableID, lsvr.CustomVariables[1].ID)
	require.Equal(t, "NAME1", lsvr.CustomVariables[1].Name)
	require.NotEmpty(t, lsvr.CustomVariables[1].UpdatedAt)

	// Test deletion of non-existent ID
	var dsvr fleet.DeleteSecretVariableResponse
	s.DoJSON("DELETE", "/api/latest/fleet/custom_variables/999", nil, http.StatusNotFound, &dsvr)

	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/custom_variables/%d", firstVariableID), nil, http.StatusOK, &dsvr)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/custom_variables/%d", secondVariableID), nil, http.StatusOK, &dsvr)

	// List after deletions should be empty.
	lsvr = fleet.ListSecretVariablesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/custom_variables", nil, http.StatusOK, &lsvr)
	require.Equal(t, 0, lsvr.Count)
	require.Empty(t, lsvr.CustomVariables)
}

func (s *integrationTestSuite) TestSecretVariablesInUse() {
	t := s.T()
	ctx := context.Background()

	foobarTeam, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Foobar",
	})
	require.NoError(t, err)

	// Create a single secret variable.
	var csvr fleet.CreateSecretVariableResponse
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "NAME1",
		Value: "value1",
	}, http.StatusOK, &csvr)
	firstVariableID := csvr.ID
	require.NotZero(t, firstVariableID)

	// Create Apple configuration profile in "No team" that uses the variable.
	appleProfile, err := s.ds.NewMDMAppleConfigProfile(
		ctx,
		fleet.MDMAppleConfigProfile{
			Name:         "Name0",
			Identifier:   "Identifier0",
			Mobileconfig: []byte("$FLEET_SECRET_NAME1"),
		},
		nil,
	)
	require.NoError(t, err)

	res := s.DoRaw("DELETE", fmt.Sprintf("/api/latest/fleet/custom_variables/%d", firstVariableID), nil, http.StatusConflict)
	errorMsg := extractServerErrorText(res.Body)
	require.Contains(
		t,
		errorMsg,
		"Couldn't delete. NAME1 is used by the \"Name0\" configuration profile in the \"No team\" team. Please delete the configuration profile and try again.",
	)

	err = s.ds.DeleteMDMAppleConfigProfile(ctx, appleProfile.ProfileUUID)
	require.NoError(t, err)

	// Create Apple declaration in "Foobar" team that uses the variable.
	appleDeclaration, err := s.ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Identifier: "decl-1",
		Name:       "decl-1",
		RawJSON:    json.RawMessage(`{"Identifier": "${FLEET_SECRET_NAME1}"}`),
		TeamID:     &foobarTeam.ID,
	}, nil)
	require.NoError(t, err)

	res = s.DoRaw("DELETE", fmt.Sprintf("/api/latest/fleet/custom_variables/%d", firstVariableID), nil, http.StatusConflict)
	errorMsg = extractServerErrorText(res.Body)
	require.Contains(
		t,
		errorMsg,
		"Couldn't delete. NAME1 is used by the \"decl-1\" configuration profile in the \"Foobar\" team. Please delete the configuration profile and try again.",
	)

	err = s.ds.DeleteMDMAppleDeclaration(ctx, appleDeclaration.DeclarationUUID)
	require.NoError(t, err)

	// Create Windows profile in "Foobar" team that uses the variable.
	windowsProfile, err := s.ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
		Name:   "zoo",
		TeamID: &foobarTeam.ID,
		SyncML: []byte("<Replace>$FLEET_SECRET_NAME1</Replace>"),
	}, nil)
	require.NoError(t, err)

	res = s.DoRaw("DELETE", fmt.Sprintf("/api/latest/fleet/custom_variables/%d", firstVariableID), nil, http.StatusConflict)
	errorMsg = extractServerErrorText(res.Body)
	require.Contains(
		t,
		errorMsg,
		"Couldn't delete. NAME1 is used by the \"zoo\" configuration profile in the \"Foobar\" team. Please delete the configuration profile and try again.",
	)

	err = s.ds.DeleteMDMWindowsConfigProfile(ctx, windowsProfile.ProfileUUID)
	require.NoError(t, err)

	// Create a script in "No team" that uses a variable
	script, err := s.ds.NewScript(ctx, &fleet.Script{
		Name:           "foobar.sh",
		ScriptContents: "echo $FLEET_SECRET_NAME1",
	})
	require.NoError(t, err)

	res = s.DoRaw("DELETE", fmt.Sprintf("/api/latest/fleet/custom_variables/%d", firstVariableID), nil, http.StatusConflict)
	errorMsg = extractServerErrorText(res.Body)
	require.Contains(
		t,
		errorMsg,
		"Couldn't delete. NAME1 is used by the \"foobar.sh\" script in the \"No team\" team. Please edit or delete the script and try again.",
	)

	err = s.ds.DeleteScript(ctx, script.ID)
	require.NoError(t, err)

	// Finally, delete now should work.
	var dsvr fleet.DeleteSecretVariableResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/custom_variables/%d", firstVariableID), nil, http.StatusOK, &dsvr)
}

func (s *integrationTestSuite) TestSecretVariablesPermissions() {
	t := s.T()
	ctx := context.Background()

	// Create a single secret variable.
	var csvr fleet.CreateSecretVariableResponse
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "NAME1",
		Value: "foobar",
	}, http.StatusOK, &csvr)

	// Create a global observer user which should be allowed to read but not create secret variables.
	u := &fleet.User{
		Name:       "Observer",
		Email:      "observer@example.com",
		GlobalRole: new(fleet.RoleObserver),
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.ds.NewUser(ctx, u)
	require.NoError(t, err)
	s.setTokenForTest(t, "observer@example.com", test.GoodPassword)

	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "NAME1",
		Value: "foobar",
	}, http.StatusForbidden, &csvr)

	// List (no-filtering) should work for non-admins.
	var lsvr fleet.ListSecretVariablesResponse
	s.DoJSON("GET", "/api/latest/fleet/custom_variables", fleet.ListSecretVariablesRequest{}, http.StatusOK, &lsvr)
	require.Equal(t, 1, lsvr.Count)
	require.Len(t, lsvr.CustomVariables, 1)
	require.NotZero(t, lsvr.CustomVariables[0].ID)
	require.Equal(t, "NAME1", lsvr.CustomVariables[0].Name)
	require.NotEmpty(t, lsvr.CustomVariables[0].UpdatedAt)
}
