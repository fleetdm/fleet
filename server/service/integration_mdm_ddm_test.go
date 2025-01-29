package service

import (
	"bytes"
	"context"
	"crypto/md5" // nolint:gosec // used only for tests
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestAppleDDMBatchUpload() {
	t := s.T()
	tmpl := `
{
	"Type": "com.apple.configuration.decl%d",
	"Identifier": "com.fleet.config%d",
	"Payload": {
		"ServiceType": "com.apple.bash",
		"DataAssetReference": "com.fleet.asset.bash" %s
	}
}`

	newDeclBytes := func(i int, payload ...string) []byte {
		var p string
		if len(payload) > 0 {
			p = "," + strings.Join(payload, ",")
		}
		return []byte(fmt.Sprintf(tmpl, i, i, p))
	}

	var decls [][]byte

	for i := 0; i < 7; i++ {
		decls = append(decls, newDeclBytes(i))
	}

	// Non-configuration type should fail
	res := s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "bad", Contents: []byte(`{"Type": "com.apple.activation"}`)},
	}}, http.StatusUnprocessableEntity)

	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Only configuration declarations (com.apple.configuration) are supported")

	// "com.apple.configuration.softwareupdate.enforcement.specific" type should fail
	res = s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "bad2", Contents: []byte(`{"Type": "com.apple.configuration.softwareupdate.enforcement.specific"}`)},
	}}, http.StatusUnprocessableEntity)

	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Declaration profile can’t include OS updates settings. To control these settings, go to OS updates.")

	// Types from our list of forbidden types should fail
	for ft := range fleet.ForbiddenDeclTypes {
		res = s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "bad2", Contents: []byte(fmt.Sprintf(`{"Type": "%s"}`, ft))},
		}}, http.StatusUnprocessableEntity)

		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, "Only configuration declarations that don’t require an asset reference are supported.")
	}

	// "com.apple.configuration.management.status-subscriptions" type should fail
	res = s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "bad2", Contents: []byte(`{"Type": "com.apple.configuration.management.status-subscriptions"}`)},
	}}, http.StatusUnprocessableEntity)

	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Declaration profile can’t include status subscription type. To get host’s vitals, please use queries and policies.")

	// Two different payloads with the same name should fail
	res = s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "bad2", Contents: newDeclBytes(1, `"foo": "bar"`)},
		{Name: "bad2", Contents: newDeclBytes(2, `"baz": "bing"`)},
	}}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "More than one configuration profile have the same name")

	// Same identifier should fail
	res = s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: decls[0]},
		{Name: "N2", Contents: decls[0]},
	}}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "A declaration profile with this identifier already exists.")

	// Create 2 declarations
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: decls[0]},
		{Name: "N2", Contents: decls[1]},
	}}, http.StatusNoContent)

	var resp listMDMConfigProfilesResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles", &listMDMConfigProfilesRequest{}, http.StatusOK, &resp)

	require.Len(t, resp.Profiles, 2)
	require.Equal(t, "N1", resp.Profiles[0].Name)
	require.Equal(t, "darwin", resp.Profiles[0].Platform)
	require.Equal(t, "N2", resp.Profiles[1].Name)
	require.Equal(t, "darwin", resp.Profiles[1].Platform)

	// Create 2 new declarations. These should take the place of the first two.
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N3", Contents: decls[2]},
		{Name: "N4", Contents: decls[3]},
	}}, http.StatusNoContent)

	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles", &listMDMConfigProfilesRequest{}, http.StatusOK, &resp)

	require.Len(t, resp.Profiles, 2)
	require.Equal(t, "N3", resp.Profiles[0].Name)
	require.Equal(t, "darwin", resp.Profiles[0].Platform)
	require.Equal(t, "N4", resp.Profiles[1].Name)
	require.Equal(t, "darwin", resp.Profiles[1].Platform)

	// replace only 1 declaration, the other one should be the same

	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N3", Contents: decls[2]},
		{Name: "N5", Contents: decls[4]},
	}}, http.StatusNoContent)

	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles", &listMDMConfigProfilesRequest{}, http.StatusOK, &resp)

	require.Len(t, resp.Profiles, 2)
	require.Equal(t, "N3", resp.Profiles[0].Name)
	require.Equal(t, "darwin", resp.Profiles[0].Platform)
	require.Equal(t, "N5", resp.Profiles[1].Name)
	require.Equal(t, "darwin", resp.Profiles[1].Platform)

	// update the declarations

	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N3", Contents: newDeclBytes(2, `"foo": "bar"`)},
		{Name: "N5", Contents: newDeclBytes(4, `"bing": "baz"`)},
	}}, http.StatusNoContent)

	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles", &listMDMConfigProfilesRequest{}, http.StatusOK, &resp)

	require.Len(t, resp.Profiles, 2)
	require.Equal(t, "N3", resp.Profiles[0].Name)
	require.Equal(t, "darwin", resp.Profiles[0].Platform)
	require.Equal(t, "N5", resp.Profiles[1].Name)
	require.Equal(t, "darwin", resp.Profiles[1].Platform)

	var createResp createLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: "label_1", Query: "select 1"}, http.StatusOK, &createResp)
	require.NotZero(t, createResp.Label.ID)
	require.Equal(t, "label_1", createResp.Label.Name)
	lbl1 := createResp.Label.Label

	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: "label_2", Query: "select 1"}, http.StatusOK, &createResp)
	require.NotZero(t, createResp.Label.ID)
	require.Equal(t, "label_2", createResp.Label.Name)
	lbl2 := createResp.Label.Label

	// Add with the deprecated "labels" and the new LabelsIncludeAll field
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N5", Contents: decls[5], Labels: []string{lbl1.Name, lbl2.Name}},
		{Name: "N6", Contents: decls[6], LabelsIncludeAll: []string{lbl1.Name}},
	}}, http.StatusNoContent)

	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles", &listMDMConfigProfilesRequest{}, http.StatusOK, &resp)

	require.Len(t, resp.Profiles, 2)
	require.Equal(t, "N5", resp.Profiles[0].Name)
	require.Equal(t, "darwin", resp.Profiles[0].Platform)
	require.Equal(t, "N6", resp.Profiles[1].Name)
	require.Equal(t, "darwin", resp.Profiles[1].Platform)
	require.Len(t, resp.Profiles[0].LabelsIncludeAll, 2)
	require.Equal(t, lbl1.Name, resp.Profiles[0].LabelsIncludeAll[0].LabelName)
	require.Equal(t, lbl2.Name, resp.Profiles[0].LabelsIncludeAll[1].LabelName)
	require.Len(t, resp.Profiles[1].LabelsIncludeAll, 1)
	require.Equal(t, lbl1.Name, resp.Profiles[1].LabelsIncludeAll[0].LabelName)
}

func (s *integrationMDMTestSuite) TestMDMAppleDeviceManagementRequests() {
	t := s.T()
	_, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	calcChecksum := func(source []byte) string {
		csum := fmt.Sprintf("%x", md5.Sum(source)) //nolint:gosec
		return strings.ToUpper(csum)
	}

	insertDeclaration := func(t *testing.T, decl fleet.MDMAppleDeclaration) {
		stmt := `
INSERT INTO mdm_apple_declarations (
	declaration_uuid,
	team_id,
	identifier,
	name,
	raw_json,
	created_at,
	uploaded_at
) VALUES (?,?,?,?,?,?,?)`

		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), stmt,
				decl.DeclarationUUID,
				decl.TeamID,
				decl.Identifier,
				decl.Name,
				decl.RawJSON,
				decl.CreatedAt,
				decl.UploadedAt,
			)
			return err
		})
	}

	insertHostDeclaration := func(t *testing.T, hostUUID string, decl fleet.MDMAppleDeclaration) {
		stmt := `
INSERT INTO host_mdm_apple_declarations (
	host_uuid,
	status,
	operation_type,
	token,
	declaration_uuid
) VALUES (?,?,?,UNHEX(?),?)`

		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), stmt,
				hostUUID,
				fleet.MDMDeliveryPending,
				fleet.MDMOperationTypeInstall,
				calcChecksum(decl.RawJSON),
				decl.DeclarationUUID,
			)
			return err
		})
	}

	// initialize a time to use for our first declaration, subsequent declarations will be
	// incremented by a minute
	then := time.Now().UTC().Truncate(time.Second).Add(-1 * time.Hour)

	// insert a declaration with no team
	noTeamDeclsByUUID := map[string]fleet.MDMAppleDeclaration{
		"123": {
			DeclarationUUID: "123",
			TeamID:          ptr.Uint(0),
			Identifier:      "com.example",
			Name:            "Example",
			RawJSON: json.RawMessage(`{
				"Type": "com.apple.configuration.declaration-items.test",
				"Payload": {"foo":"bar"},
				"Identifier": "com.example"
			}`),
			CreatedAt:  then,
			UploadedAt: then,
		},
	}
	insertDeclaration(t, noTeamDeclsByUUID["123"])
	insertHostDeclaration(t, mdmDevice.UUID, noTeamDeclsByUUID["123"])

	mapDeclsByChecksum := func(byUUID map[string]fleet.MDMAppleDeclaration) map[string]fleet.MDMAppleDeclaration {
		byChecksum := make(map[string]fleet.MDMAppleDeclaration)
		for _, d := range byUUID {
			byChecksum[calcChecksum(d.RawJSON)] = byUUID[d.DeclarationUUID]
		}
		return byChecksum
	}

	assertDeclarationResponse := func(r *http.Response, expected fleet.MDMAppleDeclaration) {
		require.NotNil(t, r)

		// unmarsal the response and assert it's valid
		var wantParsed fleet.MDMAppleDDMDeclarationResponse
		require.NoError(t, json.Unmarshal(expected.RawJSON, &wantParsed))
		var gotParsed fleet.MDMAppleDDMDeclarationResponse
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotParsed))
		require.EqualValues(t, wantParsed.Payload, gotParsed.Payload)
		require.Equal(t, calcChecksum(expected.RawJSON), gotParsed.ServerToken)
		require.Equal(t, expected.Identifier, gotParsed.Identifier)
		// t.Logf("decoded: %+v", gotParsed)
	}

	checkTokensResp := func(t *testing.T, r fleet.MDMAppleDDMTokensResponse, expectedTimestamp time.Time, prevToken string) {
		require.Equal(t, expectedTimestamp, r.SyncTokens.Timestamp)
		require.NotEmpty(t, r.SyncTokens.DeclarationsToken)
		require.NotEqual(t, prevToken, r.SyncTokens.DeclarationsToken)
	}

	checkDeclarationItemsResp := func(t *testing.T, r fleet.MDMAppleDDMDeclarationItemsResponse, expectedDeclTok string, expectedDeclsByChecksum map[string]fleet.MDMAppleDeclaration) {
		require.Equal(t, expectedDeclTok, r.DeclarationsToken)
		// TODO(roberto): better assertions
		require.NotEmpty(t, r.Declarations.Activations)
		require.Empty(t, r.Declarations.Assets)
		require.Empty(t, r.Declarations.Management)
		require.Len(t, r.Declarations.Configurations, len(expectedDeclsByChecksum))
		for _, m := range r.Declarations.Configurations {
			d, ok := expectedDeclsByChecksum[m.ServerToken]
			require.True(t, ok)
			require.Equal(t, d.Identifier, m.Identifier)
		}
	}

	var currDeclToken string // we'll use this to track the expected token across tests

	t.Run("Tokens", func(t *testing.T) {
		// get tokens, timestamp should be the same as the declaration and token should be non-empty
		r, err := mdmDevice.DeclarativeManagement("tokens")
		require.NoError(t, err)
		parsed := parseTokensResp(t, r)
		checkTokensResp(t, parsed, then, "")
		currDeclToken = parsed.SyncTokens.DeclarationsToken

		// insert a new declaration
		noTeamDeclsByUUID["456"] = fleet.MDMAppleDeclaration{
			DeclarationUUID: "456",
			TeamID:          ptr.Uint(0),
			Identifier:      "com.example2",
			Name:            "Example2",
			RawJSON: json.RawMessage(`{
				"Type": "com.apple.configuration.declaration-items.test",
				"Payload": {"foo":"baz"},
				"Identifier": "com.example2"
			}`),
			CreatedAt:  then.Add(1 * time.Minute),
			UploadedAt: then.Add(1 * time.Minute),
		}
		insertDeclaration(t, noTeamDeclsByUUID["456"])
		insertHostDeclaration(t, mdmDevice.UUID, noTeamDeclsByUUID["456"])

		// get tokens again, timestamp and token should have changed
		r, err = mdmDevice.DeclarativeManagement("tokens")
		require.NoError(t, err)
		parsed = parseTokensResp(t, r)
		checkTokensResp(t, parsed, then.Add(1*time.Minute), currDeclToken)
		currDeclToken = parsed.SyncTokens.DeclarationsToken
	})

	t.Run("DeclarationItems", func(t *testing.T) {
		r, err := mdmDevice.DeclarativeManagement("declaration-items")
		require.NoError(t, err)
		checkDeclarationItemsResp(t, parseDeclarationItemsResp(t, r), currDeclToken, mapDeclsByChecksum(noTeamDeclsByUUID))

		// insert a new declaration
		noTeamDeclsByUUID["789"] = fleet.MDMAppleDeclaration{
			DeclarationUUID: "789",
			TeamID:          ptr.Uint(0),
			Identifier:      "com.example3",
			Name:            "Example3",
			RawJSON: json.RawMessage(`{
				"Type": "com.apple.configuration.declaration-items.test",
				"Payload": {"foo":"bang"},
				"Identifier": "com.example3"
			}`),
			CreatedAt:  then.Add(2 * time.Minute),
			UploadedAt: then.Add(2 * time.Minute),
		}
		insertDeclaration(t, noTeamDeclsByUUID["789"])
		insertHostDeclaration(t, mdmDevice.UUID, noTeamDeclsByUUID["789"])

		// get tokens again, timestamp and token should have changed
		r, err = mdmDevice.DeclarativeManagement("tokens")
		require.NoError(t, err)
		toks := parseTokensResp(t, r)
		checkTokensResp(t, toks, then.Add(2*time.Minute), currDeclToken)
		currDeclToken = toks.SyncTokens.DeclarationsToken

		r, err = mdmDevice.DeclarativeManagement("declaration-items")
		require.NoError(t, err)
		checkDeclarationItemsResp(t, parseDeclarationItemsResp(t, r), currDeclToken, mapDeclsByChecksum(noTeamDeclsByUUID))
	})

	t.Run("Status", func(t *testing.T) {
		_, err := mdmDevice.DeclarativeManagement("status", fleet.MDMAppleDDMStatusReport{})
		require.NoError(t, err)
	})

	t.Run("Declaration", func(t *testing.T) {
		want := noTeamDeclsByUUID["123"]
		declarationPath := fmt.Sprintf("declaration/%s/%s", "configuration", want.Identifier)
		r, err := mdmDevice.DeclarativeManagement(declarationPath)
		require.NoError(t, err)

		assertDeclarationResponse(r, want)

		// insert a new declaration
		noTeamDeclsByUUID["abc"] = fleet.MDMAppleDeclaration{
			DeclarationUUID: "abc",
			TeamID:          ptr.Uint(0),
			Identifier:      "com.example4",
			Name:            "Example4",
			RawJSON: json.RawMessage(`{
				"Type": "com.apple.configuration.test",
				"Payload": {"foo":"bar"},
				"Identifier": "com.example4"
			}`),
			CreatedAt:  then.Add(3 * time.Minute),
			UploadedAt: then.Add(3 * time.Minute),
		}
		insertDeclaration(t, noTeamDeclsByUUID["abc"])
		insertHostDeclaration(t, mdmDevice.UUID, noTeamDeclsByUUID["abc"])
		want = noTeamDeclsByUUID["abc"]
		r, err = mdmDevice.DeclarativeManagement(fmt.Sprintf("declaration/%s/%s", "configuration", want.Identifier))
		require.NoError(t, err)

		// try getting a non-existent declaration, should fail 404
		nonExistantDeclarationPath := fmt.Sprintf("declaration/%s/%s", "configuration", "nonexistent")
		_, err = mdmDevice.DeclarativeManagement(nonExistantDeclarationPath)
		require.Error(t, err)
		require.ErrorContains(t, err, "404 Not Found")

		// try getting an unsupported declaration, should fail 404
		unsupportedDeclarationPath := fmt.Sprintf("declaration/%s/%s", "asset", "nonexistent")
		_, err = mdmDevice.DeclarativeManagement(unsupportedDeclarationPath)
		require.Error(t, err)
		require.ErrorContains(t, err, "404 Not Found")

		// typo should fail as bad request
		typoDeclarationPath := fmt.Sprintf("declarations/%s/%s", "configurations", want.Identifier)
		_, err = mdmDevice.DeclarativeManagement(typoDeclarationPath)
		require.Error(t, err)
		require.ErrorContains(t, err, "400 Bad Request")

		assertDeclarationResponse(r, want)
	})
}

func parseTokensResp(t *testing.T, r *http.Response) fleet.MDMAppleDDMTokensResponse {
	require.NotNil(t, r)
	b, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	defer r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(b))

	// unmarshal the response to make sure it's valid
	var tok fleet.MDMAppleDDMTokensResponse
	err = json.NewDecoder(r.Body).Decode(&tok)
	require.NoError(t, err)

	return tok
}

func parseDeclarationItemsResp(t *testing.T, r *http.Response) fleet.MDMAppleDDMDeclarationItemsResponse {
	require.NotNil(t, r)
	b, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	defer r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(b))

	// unmarshal the response to make sure it's valid
	var di fleet.MDMAppleDDMDeclarationItemsResponse
	err = json.NewDecoder(r.Body).Decode(&di)
	require.NoError(t, err)

	return di
}

func (s *integrationMDMTestSuite) TestAppleDDMSecretVariables() {
	t := s.T()
	_, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	checkDeclarationItemsResp := func(t *testing.T, r fleet.MDMAppleDDMDeclarationItemsResponse, expectedDeclTok string,
		expectedDeclsByToken map[string]fleet.MDMAppleDeclaration) {
		require.Equal(t, expectedDeclTok, r.DeclarationsToken)
		require.NotEmpty(t, r.Declarations.Activations)
		require.Empty(t, r.Declarations.Assets)
		require.Empty(t, r.Declarations.Management)
		require.Len(t, r.Declarations.Configurations, len(expectedDeclsByToken))
		for _, m := range r.Declarations.Configurations {
			d, ok := expectedDeclsByToken[m.ServerToken]
			if !ok {
				for k := range expectedDeclsByToken {
					t.Logf("expected token: %x", k)
				}
			}
			require.True(t, ok, "server token %x not found for %s", m.ServerToken, m.Identifier)
			require.Equal(t, d.Identifier, m.Identifier)
		}
	}

	tmpl := `
{
	"Type": "com.apple.configuration.decl%d",
	"Identifier": "com.fleet.config%d",
	"Payload": {
		"ServiceType": "com.apple.bash%d",
		"DataAssetReference": "com.fleet.asset.bash" %s
	}
}`

	newDeclBytes := func(i int, payload ...string) []byte {
		var p string
		if len(payload) > 0 {
			p = "," + strings.Join(payload, ",")
		}
		return []byte(fmt.Sprintf(tmpl, i, i, i, p))
	}

	var decls [][]byte
	for i := 0; i < 3; i++ {
		decls = append(decls, newDeclBytes(i))
	}
	// Use secrets
	myBash := "com.apple.bash1"
	decls[1] = []byte(strings.ReplaceAll(string(decls[1]), myBash, "$"+fleet.ServerSecretPrefix+"BASH"))
	secretProfile := decls[2]
	decls[2] = []byte("${" + fleet.ServerSecretPrefix + "PROFILE}")

	// Create declarations
	profilesReq := batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N0", Contents: decls[0]},
		{Name: "N1", Contents: decls[1]},
		{Name: "N2", Contents: decls[2]},
	}}
	// First dry run
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", profilesReq, http.StatusNoContent, "dry_run", "true")

	var resp listMDMConfigProfilesResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles", &listMDMConfigProfilesRequest{}, http.StatusOK, &resp)
	require.Empty(t, resp.Profiles)

	// Add secrets to server
	req := secretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  "FLEET_SECRET_BASH",
				Value: myBash,
			},
			{
				Name:  "FLEET_SECRET_PROFILE",
				Value: string(secretProfile),
			},
		},
	}
	secretResp := secretVariablesResponse{}
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &secretResp)

	// Now real run
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", profilesReq, http.StatusNoContent)
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles", &listMDMConfigProfilesRequest{}, http.StatusOK, &resp)

	require.Len(t, resp.Profiles, len(decls))
	checkedProfiles := 0
	for _, p := range resp.Profiles {
		switch p.Name {
		case "N0", "N1", "N2":
			require.Equal(t, "darwin", p.Platform)
			checkedProfiles++
		default:
			t.Logf("unexpected profile %s", p.Name)
		}
	}
	assert.Equal(t, len(decls), checkedProfiles)

	getDeclaration := func(t *testing.T, name string) fleet.MDMAppleDeclaration {
		stmt := `
SELECT
	declaration_uuid,
	team_id,
	identifier,
	name,
	raw_json,
	HEX(token) as token,
	created_at,
	uploaded_at,
	secrets_updated_at
FROM mdm_apple_declarations
WHERE name = ?`

		var decl fleet.MDMAppleDeclaration
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(context.Background(), q, &decl, stmt, name)
		})
		return decl
	}
	nameToIdentifier := make(map[string]string, 3)
	nameToUUID := make(map[string]string, 3)
	declsByToken := map[string]fleet.MDMAppleDeclaration{}
	decl := getDeclaration(t, "N0")
	nameToIdentifier["N0"] = decl.Identifier
	nameToUUID["N0"] = decl.DeclarationUUID
	declsByToken[decl.Token] = fleet.MDMAppleDeclaration{
		Identifier: "com.fleet.config0",
	}
	decl = getDeclaration(t, "N1")
	assert.NotContains(t, string(decl.RawJSON), myBash)
	assert.Contains(t, string(decl.RawJSON), "$"+fleet.ServerSecretPrefix+"BASH")
	nameToIdentifier["N1"] = decl.Identifier
	nameToUUID["N1"] = decl.DeclarationUUID
	n1Token := decl.Token
	declsByToken[decl.Token] = fleet.MDMAppleDeclaration{
		Identifier: "com.fleet.config1",
	}
	decl = getDeclaration(t, "N2")
	assert.Equal(t, string(decl.RawJSON), "${"+fleet.ServerSecretPrefix+"PROFILE}")
	nameToIdentifier["N2"] = decl.Identifier
	nameToUUID["N2"] = decl.DeclarationUUID
	declsByToken[decl.Token] = fleet.MDMAppleDeclaration{
		Identifier: "com.fleet.config2",
	}
	// trigger a profile sync
	s.awaitTriggerProfileSchedule(t)

	// get tokens again, timestamp and token should have changed
	r, err := mdmDevice.DeclarativeManagement("tokens")
	require.NoError(t, err)
	tokens := parseTokensResp(t, r)
	currDeclToken := tokens.SyncTokens.DeclarationsToken

	r, err = mdmDevice.DeclarativeManagement("declaration-items")
	require.NoError(t, err)
	itemsResp := parseDeclarationItemsResp(t, r)
	checkDeclarationItemsResp(t, itemsResp, currDeclToken, declsByToken)

	// Now, retrieve the declaration configuration profiles
	declarationPath := fmt.Sprintf("declaration/configuration/%s", nameToIdentifier["N0"])
	r, err = mdmDevice.DeclarativeManagement(declarationPath)
	require.NoError(t, err)
	var gotParsed fleet.MDMAppleDDMDeclarationResponse
	require.NoError(t, json.NewDecoder(r.Body).Decode(&gotParsed))
	assert.EqualValues(t, `{"DataAssetReference":"com.fleet.asset.bash","ServiceType":"com.apple.bash0"}`, gotParsed.Payload)

	declarationPath = fmt.Sprintf("declaration/configuration/%s", nameToIdentifier["N1"])
	r, err = mdmDevice.DeclarativeManagement(declarationPath)
	require.NoError(t, err)
	require.NoError(t, json.NewDecoder(r.Body).Decode(&gotParsed))
	assert.EqualValues(t, `{"DataAssetReference":"com.fleet.asset.bash","ServiceType":"com.apple.bash1"}`, gotParsed.Payload)

	declarationPath = fmt.Sprintf("declaration/configuration/%s", nameToIdentifier["N2"])
	r, err = mdmDevice.DeclarativeManagement(declarationPath)
	require.NoError(t, err)
	require.NoError(t, json.NewDecoder(r.Body).Decode(&gotParsed))
	assert.EqualValues(t, `{"DataAssetReference":"com.fleet.asset.bash","ServiceType":"com.apple.bash2"}`, gotParsed.Payload)

	// Upload the same profiles again -- nothing should change
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", profilesReq, http.StatusNoContent, "dry_run", "true")
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", profilesReq, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)
	// Get tokens again
	r, err = mdmDevice.DeclarativeManagement("tokens")
	require.NoError(t, err)
	tokens = parseTokensResp(t, r)
	currDeclToken = tokens.SyncTokens.DeclarationsToken
	// Get declaration items -- the checksums should be the same as before
	r, err = mdmDevice.DeclarativeManagement("declaration-items")
	require.NoError(t, err)
	itemsResp = parseDeclarationItemsResp(t, r)
	checkDeclarationItemsResp(t, itemsResp, currDeclToken, declsByToken)

	// Change the secrets.
	myBash = "my.new.bash"
	req = secretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  "FLEET_SECRET_BASH",
				Value: myBash, // changed
			},
			{
				Name:  "FLEET_SECRET_PROFILE",
				Value: string(secretProfile), // did not change
			},
		},
	}
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &secretResp)
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", profilesReq, http.StatusNoContent, "dry_run", "true")
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", profilesReq, http.StatusNoContent)
	// The token of the declaration with the updated secret should have changed.
	decl = getDeclaration(t, "N1")
	assert.NotContains(t, string(decl.RawJSON), myBash)
	assert.Contains(t, string(decl.RawJSON), "$"+fleet.ServerSecretPrefix+"BASH")
	nameToIdentifier["N1"] = decl.Identifier
	nameToUUID["N1"] = decl.DeclarationUUID
	assert.NotEqual(t, n1Token, decl.Token)
	// Update expected token
	delete(declsByToken, n1Token)
	declsByToken[decl.Token] = fleet.MDMAppleDeclaration{
		Identifier: "com.fleet.config1",
	}
	s.awaitTriggerProfileSchedule(t)

	// Get tokens again
	r, err = mdmDevice.DeclarativeManagement("tokens")
	require.NoError(t, err)
	tokens = parseTokensResp(t, r)
	currDeclToken = tokens.SyncTokens.DeclarationsToken
	// Only N1 should have changed
	r, err = mdmDevice.DeclarativeManagement("declaration-items")
	require.NoError(t, err)
	itemsResp = parseDeclarationItemsResp(t, r)
	checkDeclarationItemsResp(t, itemsResp, currDeclToken, declsByToken)

	// Now, retrieve the declaration configuration profiles
	declarationPath = fmt.Sprintf("declaration/configuration/%s", nameToIdentifier["N0"])
	r, err = mdmDevice.DeclarativeManagement(declarationPath)
	require.NoError(t, err)
	require.NoError(t, json.NewDecoder(r.Body).Decode(&gotParsed))
	assert.EqualValues(t, `{"DataAssetReference":"com.fleet.asset.bash","ServiceType":"com.apple.bash0"}`, gotParsed.Payload)

	declarationPath = fmt.Sprintf("declaration/configuration/%s", nameToIdentifier["N1"])
	r, err = mdmDevice.DeclarativeManagement(declarationPath)
	require.NoError(t, err)
	require.NoError(t, json.NewDecoder(r.Body).Decode(&gotParsed))
	assert.EqualValues(t, `{"DataAssetReference":"com.fleet.asset.bash","ServiceType":"my.new.bash"}`, gotParsed.Payload)

	declarationPath = fmt.Sprintf("declaration/configuration/%s", nameToIdentifier["N2"])
	r, err = mdmDevice.DeclarativeManagement(declarationPath)
	require.NoError(t, err)
	require.NoError(t, json.NewDecoder(r.Body).Decode(&gotParsed))
	assert.EqualValues(t, `{"DataAssetReference":"com.fleet.asset.bash","ServiceType":"com.apple.bash2"}`, gotParsed.Payload)

	// Delete the profiles
	s.Do("DELETE", "/api/latest/fleet/configuration_profiles/"+nameToUUID["N0"], nil, http.StatusOK)
	s.Do("DELETE", "/api/latest/fleet/configuration_profiles/"+nameToUUID["N1"], nil, http.StatusOK)
	s.Do("DELETE", "/api/latest/fleet/configuration_profiles/"+nameToUUID["N2"], nil, http.StatusOK)
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles", &listMDMConfigProfilesRequest{}, http.StatusOK, &resp)
	require.Empty(t, resp.Profiles)
}

func (s *integrationMDMTestSuite) TestAppleDDMReconciliation() {
	t := s.T()
	ctx := context.Background()

	addDeclaration := func(identifier string, teamID uint, labelNames []string) string {
		fields := map[string][]string{
			"labels": labelNames,
		}
		if teamID > 0 {
			fields["team_id"] = []string{fmt.Sprintf("%d", teamID)}
		}
		body, headers := generateNewProfileMultipartRequest(
			t, identifier+".json", declarationForTest(identifier), s.token, fields,
		)
		res := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusOK, headers)
		var resp newMDMConfigProfileResponse
		err := json.NewDecoder(res.Body).Decode(&resp)
		require.NoError(t, err)
		require.NotEmpty(t, resp.ProfileUUID)
		require.Equal(t, "d", string(resp.ProfileUUID[0]))
		return resp.ProfileUUID
	}

	deleteDeclaration := func(declUUID string) {
		var deleteResp deleteMDMConfigProfileResponse
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", declUUID), nil, http.StatusOK, &deleteResp)
	}

	// create a team
	teamName := t.Name() + "team1"
	team := &fleet.Team{
		Name: teamName,
	}
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	team = createTeamResp.Team

	checkNoCommands := func(d *mdmtest.TestAppleMDMClient) {
		cmd, err := d.Idle()
		require.NoError(t, err)
		require.Nil(t, cmd)
	}

	checkDDMSync := func(d *mdmtest.TestAppleMDMClient) {
		cmd, err := d.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd)
		require.Equal(t, "DeclarativeManagement", cmd.Command.RequestType)
		cmd, err = d.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
		require.Nil(t, cmd)
		_, err = d.DeclarativeManagement("tokens")
		require.NoError(t, err)
	}

	// create a windows host
	_, err := s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            1,
		OsqueryHostID: ptr.String("non-macos-host"),
		NodeKey:       ptr.String("non-macos-host"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.non.macos", t.Name()),
		Platform:      "windows",
	})
	require.NoError(t, err)

	// create a windows host that's enrolled in MDM
	_, _ = createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	// create a linux host
	_, err = s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            2,
		OsqueryHostID: ptr.String("linux-host"),
		NodeKey:       ptr.String("linux-host"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.linux", t.Name()),
		Platform:      "linux",
	})
	require.NoError(t, err)

	// create a host that's not enrolled into MDM
	_, err = s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            2,
		OsqueryHostID: ptr.String("not-mdm-enrolled"),
		NodeKey:       ptr.String("not-mdm-enrolled"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.not.enrolled", t.Name()),
		Platform:      "darwin",
	})
	require.NoError(t, err)

	// create a host and then enroll in MDM.
	mdmHost, device := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	// Create and then immediately delete a declaration
	delUUID := addDeclaration("TestImmediateDelete", 0, nil)
	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", mdmHost.ID), nil, http.StatusOK, &hostResp)
	require.NotNil(t, hostResp.Host.MDM.Profiles)
	require.Len(t, *hostResp.Host.MDM.Profiles, 1)
	require.Equal(t, (*hostResp.Host.MDM.Profiles)[0].Name, "TestImmediateDelete")

	deleteDeclaration(delUUID)
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", mdmHost.ID), nil, http.StatusOK, &hostResp)
	require.Nil(t, hostResp.Host.MDM.Profiles)

	// trigger the reconciler, no error
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)

	// declarativeManagement command is not sent.
	checkNoCommands(device)

	// add global declarations
	d1UUID := addDeclaration("I1", 0, nil)
	addDeclaration("I2", 0, nil)

	// reconcile again, this time new declarations were added
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)

	// TODO: check command is pending

	// declarativeManagement command is sent
	checkDDMSync(device)

	// reconcile again, commands for the uploaded declarations are already sent
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	// no new commands are sent
	checkNoCommands(device)

	// delete a declaration
	deleteDeclaration(d1UUID)
	// reconcile again
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	// a DDM sync is triggered
	checkDDMSync(device)

	// add a new host
	_, deviceTwo := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	// reconcile again
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	// DDM sync is triggered only for the new host
	checkNoCommands(device)
	checkDDMSync(deviceTwo)

	// add device to the team
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &team.ID, HostIDs: []uint{mdmHost.ID}}, http.StatusOK)

	// reconcile
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)

	// DDM sync is triggered only for the transferred host
	// because the team doesn't have any declarations
	checkDDMSync(device)
	checkNoCommands(deviceTwo)

	// reconcile
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	// nobody receives commands this time
	checkNoCommands(device)
	checkNoCommands(deviceTwo)

	// add declarations to the team
	addDeclaration("I1", team.ID, nil)
	addDeclaration("I2", team.ID, nil)

	// reconcile
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	// DDM sync is triggered for the host in the team
	checkDDMSync(device)
	checkNoCommands(deviceTwo)

	// add a new host, this one belongs to the team
	mdmHostThree, deviceThree := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &team.ID, HostIDs: []uint{mdmHostThree.ID}}, http.StatusOK)

	// reconcile
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	// DDM sync is triggered only for the new host
	checkNoCommands(device)
	checkNoCommands(deviceTwo)
	checkDDMSync(deviceThree)

	// no new commands after another reconciliation
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	checkNoCommands(device)
	checkNoCommands(deviceTwo)
	checkNoCommands(deviceThree)

	label, err := s.ds.NewLabel(ctx, &fleet.Label{Name: t.Name(), Query: "select 1;"})
	require.NoError(t, err)
	// update label with host membership
	mysql.ExecAdhocSQL(
		t, s.ds, func(db sqlx.ExtContext) error {
			_, err := db.ExecContext(
				context.Background(),
				"INSERT IGNORE INTO label_membership (host_id, label_id) VALUES (?, ?)",
				mdmHostThree.ID,
				label.ID,
			)
			return err
		},
	)

	// add a new label + label declaration
	addDeclaration("I3", team.ID, []string{label.Name})

	// reconcile
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	// DDM sync is triggered only for the host with the label
	checkNoCommands(device)
	checkNoCommands(deviceTwo)
	checkDDMSync(deviceThree)
}

func (s *integrationMDMTestSuite) TestAppleDDMStatusReport() {
	t := s.T()
	ctx := context.Background()

	assertHostDeclarations := func(hostUUID string, wantDecls []*fleet.MDMAppleHostDeclaration) {
		var gotDecls []*fleet.MDMAppleHostDeclaration
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(context.Background(), q, &gotDecls, `SELECT declaration_identifier, status, operation_type FROM host_mdm_apple_declarations WHERE host_uuid = ?`, hostUUID)
		})
		require.ElementsMatch(t, wantDecls, gotDecls)
	}

	// create a host and then enroll in MDM.
	mdmHost, device := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	declarations := []fleet.MDMProfileBatchPayload{
		{Name: "N1.json", Contents: declarationForTest("I1")},
		{Name: "N2.json", Contents: declarationForTest("I2")},
	}
	// add global declarations
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: declarations}, http.StatusNoContent)

	// reconcile profiles
	err := ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)

	// declarations are ("install", "pending") after the cron run
	assertHostDeclarations(mdmHost.UUID, []*fleet.MDMAppleHostDeclaration{
		{Identifier: "I1", Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		{Identifier: "I2", Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
	})

	// host gets a DDM sync call
	cmd, err := device.Idle()
	require.NoError(t, err)
	require.Equal(t, "DeclarativeManagement", cmd.Command.RequestType)
	_, err = device.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)

	r, err := device.DeclarativeManagement("declaration-items")
	require.NoError(t, err)
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	var items fleet.MDMAppleDDMDeclarationItemsResponse
	require.NoError(t, json.Unmarshal(body, &items))

	var i1ServerToken, i2ServerToken string
	for _, d := range items.Declarations.Configurations {
		switch d.Identifier {
		case "I1":
			i1ServerToken = d.ServerToken
		case "I2":
			i2ServerToken = d.ServerToken
		}
	}

	// declarations are ("install", "verifying") after the ack
	assertHostDeclarations(mdmHost.UUID, []*fleet.MDMAppleHostDeclaration{
		{Identifier: "I1", Status: &fleet.MDMDeliveryVerifying, OperationType: fleet.MDMOperationTypeInstall},
		{Identifier: "I2", Status: &fleet.MDMDeliveryVerifying, OperationType: fleet.MDMOperationTypeInstall},
	})

	// host sends a partial DDM report
	report := fleet.MDMAppleDDMStatusReport{}
	report.StatusItems.Management.Declarations.Configurations = []fleet.MDMAppleDDMStatusDeclaration{
		{Active: true, Valid: fleet.MDMAppleDeclarationValid, Identifier: "I1", ServerToken: i1ServerToken},
	}
	_, err = device.DeclarativeManagement("status", report)
	require.NoError(t, err)
	assertHostDeclarations(mdmHost.UUID, []*fleet.MDMAppleHostDeclaration{
		{Identifier: "I1", Status: &fleet.MDMDeliveryVerified, OperationType: fleet.MDMOperationTypeInstall},
		{Identifier: "I2", Status: &fleet.MDMDeliveryVerifying, OperationType: fleet.MDMOperationTypeInstall},
	})

	// host sends a report with a wrong (could be old) server token for I2, nothing changes
	report = fleet.MDMAppleDDMStatusReport{}
	report.StatusItems.Management.Declarations.Configurations = []fleet.MDMAppleDDMStatusDeclaration{
		{Active: true, Valid: fleet.MDMAppleDeclarationValid, Identifier: "I2", ServerToken: "foo"},
	}
	_, err = device.DeclarativeManagement("status", report)
	require.NoError(t, err)
	assertHostDeclarations(mdmHost.UUID, []*fleet.MDMAppleHostDeclaration{
		{Identifier: "I1", Status: &fleet.MDMDeliveryVerified, OperationType: fleet.MDMOperationTypeInstall},
		{Identifier: "I2", Status: &fleet.MDMDeliveryVerifying, OperationType: fleet.MDMOperationTypeInstall},
	})

	// host sends a full report, declaration I2 is invalid
	report = fleet.MDMAppleDDMStatusReport{}
	report.StatusItems.Management.Declarations.Configurations = []fleet.MDMAppleDDMStatusDeclaration{
		{Active: true, Valid: fleet.MDMAppleDeclarationValid, Identifier: "I1", ServerToken: i1ServerToken},
		{Active: false, Valid: fleet.MDMAppleDeclarationInvalid, Identifier: "I2", ServerToken: i2ServerToken},
	}
	_, err = device.DeclarativeManagement("status", report)
	require.NoError(t, err)
	assertHostDeclarations(mdmHost.UUID, []*fleet.MDMAppleHostDeclaration{
		{Identifier: "I1", Status: &fleet.MDMDeliveryVerified, OperationType: fleet.MDMOperationTypeInstall},
		{Identifier: "I2", Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall},
	})

	// do a batch request, this time I2 is deleted
	declarations = []fleet.MDMProfileBatchPayload{
		{Name: "N1.json", Contents: declarationForTest("I1")},
	}
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: declarations}, http.StatusNoContent)

	// reconcile profiles
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	assertHostDeclarations(mdmHost.UUID, []*fleet.MDMAppleHostDeclaration{
		{Identifier: "I1", Status: &fleet.MDMDeliveryVerified, OperationType: fleet.MDMOperationTypeInstall},
		{Identifier: "I2", Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
	})

	// host sends a report, declaration I2 is removed from the hosts_* table
	report = fleet.MDMAppleDDMStatusReport{}
	report.StatusItems.Management.Declarations.Configurations = []fleet.MDMAppleDDMStatusDeclaration{
		{Active: true, Valid: fleet.MDMAppleDeclarationValid, Identifier: "I1", ServerToken: i1ServerToken},
	}
	_, err = device.DeclarativeManagement("status", report)
	require.NoError(t, err)
	assertHostDeclarations(mdmHost.UUID, []*fleet.MDMAppleHostDeclaration{
		{Identifier: "I1", Status: &fleet.MDMDeliveryVerified, OperationType: fleet.MDMOperationTypeInstall},
	})

	// host sends a report, declaration I1 is failing after a while
	report = fleet.MDMAppleDDMStatusReport{}
	report.StatusItems.Management.Declarations.Configurations = []fleet.MDMAppleDDMStatusDeclaration{
		{Active: false, Valid: fleet.MDMAppleDeclarationInvalid, Identifier: "I1", ServerToken: i1ServerToken},
	}
	_, err = device.DeclarativeManagement("status", report)
	require.NoError(t, err)
	assertHostDeclarations(mdmHost.UUID, []*fleet.MDMAppleHostDeclaration{
		{Identifier: "I1", Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall},
	})
}

func (s *integrationMDMTestSuite) TestDDMUnsupportedDevice() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := context.Background()
	fleetHost, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	getProfiles := func(h *fleet.Host) map[string]*fleet.HostMDMAppleProfile {
		profs, err := s.ds.GetHostMDMAppleProfiles(ctx, h.UUID)
		require.NoError(t, err)
		out := make(map[string]*fleet.HostMDMAppleProfile, len(profs))
		for _, p := range profs {
			p := p
			out[p.Identifier] = &p
		}

		return out
	}

	declarations := []fleet.MDMProfileBatchPayload{
		{Name: "N1.json", Contents: declarationForTest("I1")},
		{Name: "N2.json", Contents: declarationForTest("I2")},
	}
	// add global declarations
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: declarations}, http.StatusNoContent)

	// reconcile declarations
	err := ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)

	// declaration is pending
	profs := getProfiles(fleetHost)
	require.Equal(t, &fleet.MDMDeliveryPending, profs["I1"].Status)
	require.Equal(t, &fleet.MDMDeliveryPending, profs["I2"].Status)

	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	require.Equal(t, "DeclarativeManagement", cmd.Command.RequestType)

	// simulate an error returned by devices that don't support DDM
	errChain := []mdm.ErrorChain{
		{
			ErrorCode:            4,
			ErrorDomain:          "RMErrorDomain",
			LocalizedDescription: "Feature Disabled: DeclarativeManagement is disabled.",
		},
	}
	cmd, err = mdmDevice.Err(cmd.CommandUUID, errChain)
	require.NoError(t, err)
	require.Nil(t, cmd)

	// profiles are failed
	profs = getProfiles(fleetHost)
	require.Equal(t, &fleet.MDMDeliveryFailed, profs["I1"].Status)
	require.Contains(t, profs["I1"].Detail, "Feature Disabled")
	require.Equal(t, &fleet.MDMDeliveryFailed, profs["I2"].Status)
	require.Contains(t, profs["I2"].Detail, "Feature Disabled")
}

func (s *integrationMDMTestSuite) TestDDMNoDeclarationsLeft() {
	t := s.T()
	_, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	res, err := mdmDevice.DeclarativeManagement("tokens")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	var tok fleet.MDMAppleDDMTokensResponse
	err = json.NewDecoder(res.Body).Decode(&tok)
	require.NoError(t, err)
	require.Empty(t, tok.SyncTokens.DeclarationsToken)
	require.NotEmpty(t, tok.SyncTokens.Timestamp)

	res, err = mdmDevice.DeclarativeManagement("declaration-items")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	var items fleet.MDMAppleDDMDeclarationItemsResponse
	err = json.NewDecoder(res.Body).Decode(&items)
	require.NoError(t, err)
	require.Empty(t, items.DeclarationsToken)
	require.Empty(t, items.Declarations.Activations)
	require.Empty(t, items.Declarations.Configurations)
	require.Empty(t, items.Declarations.Assets)
	require.Empty(t, items.Declarations.Management)
}

func (s *integrationMDMTestSuite) TestDDMTransactionRecording() {
	t := s.T()
	ctx := context.Background()

	type record struct {
		EnrollmentID string           `db:"enrollment_id"`
		MessageType  string           `db:"message_type"`
		RawJSON      *json.RawMessage `db:"raw_json"`
	}
	verifyTransactionRecord := func(want record) {
		var got record
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(
				ctx, q, &got,
				`SELECT
				    enrollment_id, message_type, raw_json
				 FROM mdm_apple_declarative_requests
				 ORDER BY id DESC
				 LIMIT 1`,
			)
		})
		if got.RawJSON != nil {
			fmt.Println(string(*got.RawJSON))
		}
		require.Equal(t, want, got)
	}

	declarations := []fleet.MDMProfileBatchPayload{
		{Name: "N1.json", Contents: declarationForTest("I1")},
		{Name: "N2.json", Contents: declarationForTest("I2")},
	}
	// add global declarations
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: declarations}, http.StatusNoContent)

	// reconcile declarations
	err := ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)

	_, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	_, err = mdmDevice.DeclarativeManagement("tokens")
	require.NoError(t, err)
	verifyTransactionRecord(record{
		MessageType:  "tokens",
		EnrollmentID: mdmDevice.UUID,
		RawJSON:      nil,
	})

	res, err := mdmDevice.DeclarativeManagement("declaration-items")
	require.NoError(t, err)
	verifyTransactionRecord(record{
		MessageType:  "declaration-items",
		EnrollmentID: mdmDevice.UUID,
		RawJSON:      nil,
	})

	var items fleet.MDMAppleDDMDeclarationItemsResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&items))
	var i1ServerToken string
	for _, d := range items.Declarations.Configurations {
		if d.Identifier == "I1" {
			i1ServerToken = d.ServerToken
		}
	}

	// a second device requests tokens
	_, mdmDeviceTwo := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)

	_, err = mdmDeviceTwo.DeclarativeManagement("tokens")
	require.NoError(t, err)
	verifyTransactionRecord(record{
		MessageType:  "tokens",
		EnrollmentID: mdmDeviceTwo.UUID,
		RawJSON:      nil,
	})

	_, err = mdmDevice.DeclarativeManagement("declaration/configuration/I1")
	require.NoError(t, err)
	verifyTransactionRecord(record{
		MessageType:  "declaration/configuration/I1",
		EnrollmentID: mdmDevice.UUID,
		RawJSON:      nil,
	})

	report := fleet.MDMAppleDDMStatusReport{}
	report.StatusItems.Management.Declarations.Configurations = []fleet.MDMAppleDDMStatusDeclaration{
		{Active: true, Valid: fleet.MDMAppleDeclarationValid, Identifier: "I1", ServerToken: i1ServerToken},
	}
	_, err = mdmDevice.DeclarativeManagement("status", report)
	require.NoError(t, err)
	verifyTransactionRecord(record{
		MessageType:  "status",
		EnrollmentID: mdmDevice.UUID,
		RawJSON: ptr.RawMessage(
			json.RawMessage(
				fmt.Sprintf(
					`{"StatusItems":{"management":{"declarations":{"activations":null,"configurations":[{"active":true,"identifier":"I1","valid":"valid","server-token":"%s"}],"assets":null,"management":null}}},"Errors":null}`,
					i1ServerToken,
				),
			),
		),
	})
}

func declarationForTest(identifier string) []byte {
	return []byte(fmt.Sprintf(`
{
    "Type": "com.apple.configuration.management.test",
    "Payload": {
        "Echo": "foo"
    },
    "Identifier": "%s"
}`, identifier))
}

func declarationForTestWithType(identifier string, dType string) []byte {
	return []byte(fmt.Sprintf(`
{
    "Type": "%s",
    "Payload": {
        "Echo": "foo"
    },
    "Identifier": "%s"
}`, dType, identifier))
}
