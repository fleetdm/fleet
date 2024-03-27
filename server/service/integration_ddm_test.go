package service

import (
	"bytes"
	"context"
	"crypto/md5" // nolint:gosec // used only for tests
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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
	// TODO: figure out the best way to do this. We might even consider
	// starting a different test suite.
	t.Cleanup(func() { s.cleanupDeclarations(t) })

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
	require.Contains(t, errMsg, "A declaration profile with this name already exists.")

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
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: ptr.String("label_1"), Query: ptr.String("select 1")}, http.StatusOK, &createResp)
	require.NotZero(t, createResp.Label.ID)
	require.Equal(t, "label_1", createResp.Label.Name)
	lbl1 := createResp.Label.Label

	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: ptr.String("label_2"), Query: ptr.String("select 1")}, http.StatusOK, &createResp)
	require.NotZero(t, createResp.Label.ID)
	require.Equal(t, "label_2", createResp.Label.Name)
	lbl2 := createResp.Label.Label

	// Add with labels
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N5", Contents: decls[5], Labels: []string{lbl1.Name, lbl2.Name}},
		{Name: "N6", Contents: decls[6], Labels: []string{lbl1.Name}},
	}}, http.StatusNoContent)

	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles", &listMDMConfigProfilesRequest{}, http.StatusOK, &resp)

	require.Len(t, resp.Profiles, 2)
	require.Equal(t, "N5", resp.Profiles[0].Name)
	require.Equal(t, "darwin", resp.Profiles[0].Platform)
	require.Equal(t, "N6", resp.Profiles[1].Name)
	require.Equal(t, "darwin", resp.Profiles[1].Platform)
	require.Len(t, resp.Profiles[0].Labels, 2)
	require.Equal(t, lbl1.Name, resp.Profiles[0].Labels[0].LabelName)
	require.Equal(t, lbl2.Name, resp.Profiles[0].Labels[1].LabelName)
	require.Len(t, resp.Profiles[1].Labels, 1)
	require.Equal(t, lbl1.Name, resp.Profiles[1].Labels[0].LabelName)
}

func (s *integrationMDMTestSuite) TestMDMAppleDeviceManagementRequests() {
	t := s.T()
	_, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	calcChecksum := func(source []byte) string {
		csum := fmt.Sprintf("%x", md5.Sum(source)) //nolint:gosec
		return strings.ToUpper(csum)
	}

	t.Cleanup(func() { s.cleanupDeclarations(t) })

	insertDeclaration := func(t *testing.T, decl fleet.MDMAppleDeclaration) {
		stmt := `
INSERT INTO mdm_apple_declarations (
	declaration_uuid,
	team_id,
	identifier,
	name,
	raw_json,
	checksum,
	created_at,
	uploaded_at
) VALUES (?,?,?,?,?,UNHEX(?),?,?)`

		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), stmt,
				decl.DeclarationUUID,
				decl.TeamID,
				decl.Identifier,
				decl.Name,
				decl.RawJSON,
				calcChecksum(decl.RawJSON),
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
	checksum,
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

	parseTokensResp := func(r *http.Response) fleet.MDMAppleDDMTokensResponse {
		require.NotNil(t, r)
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()
		r.Body = io.NopCloser(bytes.NewBuffer(b))
		// t.Log("body", string(b))

		// unmarsal the response to make sure it's valid
		var tok fleet.MDMAppleDDMTokensResponse
		err = json.NewDecoder(r.Body).Decode(&tok)
		require.NoError(t, err)
		// t.Log("decoded", tok)

		return tok
	}

	parseDeclarationItemsResp := func(r *http.Response) fleet.MDMAppleDDMDeclarationItemsResponse {
		require.NotNil(t, r)
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()
		r.Body = io.NopCloser(bytes.NewBuffer(b))
		// t.Log("body", string(b))

		// unmarsal the response to make sure it's valid
		var di fleet.MDMAppleDDMDeclarationItemsResponse
		err = json.NewDecoder(r.Body).Decode(&di)
		require.NoError(t, err)
		// t.Log("decoded", di)

		return di
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

	checkRequestsDatabase := func(t *testing.T, messageType, enrollmentID string, expectedCount int) {
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			var count int
			if err := sqlx.GetContext(
				context.Background(),
				q,
				&count,
				"SELECT count(*) AS count FROM mdm_apple_declarative_requests WHERE enrollment_id = ? AND message_type = ?",
				enrollmentID,
				messageType,
			); err != nil {
				return err
			}

			require.Equal(t, expectedCount, count, "unexpected db row count for declaration requests")

			return nil
		})
	}

	var currDeclToken string // we'll use this to track the expected token across tests

	t.Run("Tokens", func(t *testing.T) {
		checkRequestsDatabase(t, "tokens", mdmDevice.UUID, 0)
		// get tokens, timestamp should be the same as the declaration and token should be non-empty
		r, err := mdmDevice.DeclarativeManagement("tokens")
		require.NoError(t, err)
		parsed := parseTokensResp(r)
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
		checkRequestsDatabase(t, "tokens", mdmDevice.UUID, 1)

		// get tokens again, timestamp and token should have changed
		r, err = mdmDevice.DeclarativeManagement("tokens")
		require.NoError(t, err)
		parsed = parseTokensResp(r)
		checkTokensResp(t, parsed, then.Add(1*time.Minute), currDeclToken)
		currDeclToken = parsed.SyncTokens.DeclarationsToken
		checkRequestsDatabase(t, "tokens", mdmDevice.UUID, 2)
	})

	t.Run("DeclarationItems", func(t *testing.T) {
		checkRequestsDatabase(t, "declaration-items", mdmDevice.UUID, 0)
		r, err := mdmDevice.DeclarativeManagement("declaration-items")
		require.NoError(t, err)
		checkDeclarationItemsResp(t, parseDeclarationItemsResp(r), currDeclToken, mapDeclsByChecksum(noTeamDeclsByUUID))

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
		checkRequestsDatabase(t, "declaration-items", mdmDevice.UUID, 1)

		// get tokens again, timestamp and token should have changed
		r, err = mdmDevice.DeclarativeManagement("tokens")
		require.NoError(t, err)
		toks := parseTokensResp(r)
		checkTokensResp(t, toks, then.Add(2*time.Minute), currDeclToken)
		currDeclToken = toks.SyncTokens.DeclarationsToken

		r, err = mdmDevice.DeclarativeManagement("declaration-items")
		require.NoError(t, err)
		checkDeclarationItemsResp(t, parseDeclarationItemsResp(r), currDeclToken, mapDeclsByChecksum(noTeamDeclsByUUID))
		checkRequestsDatabase(t, "declaration-items", mdmDevice.UUID, 2)
	})

	t.Run("Status", func(t *testing.T) {
		checkRequestsDatabase(t, "status", mdmDevice.UUID, 0)
		_, err := mdmDevice.DeclarativeManagement("status", fleet.MDMAppleDDMStatusReport{})
		require.NoError(t, err)
		checkRequestsDatabase(t, "status", mdmDevice.UUID, 1)
	})

	t.Run("Declaration", func(t *testing.T) {
		want := noTeamDeclsByUUID["123"]
		declarationPath := fmt.Sprintf("declaration/%s/%s", "configuration", want.Identifier)
		checkRequestsDatabase(t, declarationPath, mdmDevice.UUID, 0)
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
		checkRequestsDatabase(t, declarationPath, mdmDevice.UUID, 1)

		// try getting a non-existent declaration, should fail 404
		nonExistantDeclarationPath := fmt.Sprintf("declaration/%s/%s", "configuration", "nonexistent")
		checkRequestsDatabase(t, nonExistantDeclarationPath, mdmDevice.UUID, 0)
		_, err = mdmDevice.DeclarativeManagement(nonExistantDeclarationPath)
		require.Error(t, err)
		require.ErrorContains(t, err, "404 Not Found")
		checkRequestsDatabase(t, nonExistantDeclarationPath, mdmDevice.UUID, 1)

		// typo should fail as bad request
		typoDeclarationPath := fmt.Sprintf("declarations/%s/%s", "configurations", want.Identifier)
		checkRequestsDatabase(t, typoDeclarationPath, mdmDevice.UUID, 0)
		_, err = mdmDevice.DeclarativeManagement(typoDeclarationPath)
		require.Error(t, err)
		require.ErrorContains(t, err, "400 Bad Request")
		checkRequestsDatabase(t, typoDeclarationPath, mdmDevice.UUID, 1)

		assertDeclarationResponse(r, want)
	})
}

func (s *integrationMDMTestSuite) TestAppleDDMReconciliation() {
	t := s.T()
	ctx := context.Background()
	// TODO: use config logger or take into account FLEET_INTEGRATION_TESTS_DISABLE_LOG
	logger := kitlog.NewJSONLogger(os.Stdout)

	// TODO: use endpoints once those are available.
	addDeclaration := func(identifier string, teamID uint) {
		stmt := `
		  INSERT INTO mdm_apple_declarations
		    (declaration_uuid, team_id, identifier, name, raw_json, checksum)
		  VALUES
		    (UUID(), ?, ?, UUID(), ?, HEX(MD5(raw_json)) )`
		mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, stmt, teamID, identifier, declarationForTest(identifier))
			return err
		})
	}

	deleteDeclaration := func(identifier string, teamID uint) {
		mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, "DELETE FROM mdm_apple_declarations WHERE team_id = ? AND identifier = ?", teamID, identifier)
			return err
		})
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

	// TODO: figure out the best way to do this. We might even consider
	// starting a different test suite.
	t.Cleanup(func() { s.cleanupDeclarations(t) })

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

	// trigger the reconciler, no error
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, logger)
	require.NoError(t, err)

	// declarativeManagement command is not sent.
	checkNoCommands(device)

	// add global declarations
	addDeclaration("I1", 0)
	addDeclaration("I2", 0)

	// reconcile again, this time new declarations were added
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, logger)
	require.NoError(t, err)

	// TODO: check command is pending

	// declarativeManagement command is sent
	checkDDMSync(device)

	// reconcile again, commands for the uploaded declarations are already sent
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, logger)
	require.NoError(t, err)
	// no new commands are sent
	checkNoCommands(device)

	// delete a declaration
	deleteDeclaration("I1", 0)
	// reconcile again
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, logger)
	require.NoError(t, err)
	// a DDM sync is triggered
	checkDDMSync(device)

	// add a new host
	_, deviceTwo := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	// reconcile again
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, logger)
	require.NoError(t, err)
	// DDM sync is triggered only for the new host
	checkNoCommands(device)
	checkDDMSync(deviceTwo)

	// add device to the team
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &team.ID, HostIDs: []uint{mdmHost.ID}}, http.StatusOK)

	// reconcile
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, logger)
	require.NoError(t, err)

	// DDM sync is triggered only for the transferred host
	// because the team doesn't have any declarations
	checkDDMSync(device)
	checkNoCommands(deviceTwo)

	// reconcile
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, logger)
	require.NoError(t, err)
	// nobody receives commands this time
	checkNoCommands(device)
	checkNoCommands(deviceTwo)

	// add declarations to the team
	addDeclaration("I1", team.ID)
	addDeclaration("I2", team.ID)

	// reconcile
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, logger)
	require.NoError(t, err)
	// DDM sync is triggered for the host in the team
	checkDDMSync(device)
	checkNoCommands(deviceTwo)

	// add a new host, this one belongs to the team
	mdmHostThree, deviceThree := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &team.ID, HostIDs: []uint{mdmHostThree.ID}}, http.StatusOK)

	// reconcile
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, logger)
	require.NoError(t, err)
	// DDM sync is triggered only for the new host
	checkNoCommands(device)
	checkNoCommands(deviceTwo)
	checkDDMSync(deviceThree)

	// no new commands after another reconciliation
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, logger)
	require.NoError(t, err)
	checkNoCommands(device)
	checkNoCommands(deviceTwo)
	checkNoCommands(deviceThree)

	// TODO: use proper APIs for this
	// add a new label + label declaration
	addDeclaration("I3", team.ID)
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

	// update declaration <-> label mapping
	mysql.ExecAdhocSQL(
		t, s.ds, func(db sqlx.ExtContext) error {
			_, err := db.ExecContext(
				context.Background(),
				`INSERT INTO
				  mdm_declaration_labels (apple_declaration_uuid, label_name, label_id)
				  VALUES ((SELECT declaration_uuid FROM mdm_apple_declarations WHERE team_id = ? and identifier = ?), ?, ?)`,
				team.ID,
				"I3",
				label.Name,
				label.ID,
			)
			return err
		},
	)

	// reconcile
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, logger)
	require.NoError(t, err)
	// DDM sync is triggered only for the host with the label
	checkNoCommands(device)
	checkNoCommands(deviceTwo)
	checkDDMSync(deviceThree)
}

func (s *integrationMDMTestSuite) TestAppleDDMStatusReport() {
	t := s.T()
	ctx := context.Background()
	// TODO: use config logger or take into account FLEET_INTEGRATION_TESTS_DISABLE_LOG
	logger := kitlog.NewJSONLogger(os.Stdout)

	// TODO: figure out the best way to do this. We might even consider
	// starting a different test suite.
	t.Cleanup(func() { s.cleanupDeclarations(t) })

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
	err := ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, logger)
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
	err = ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, logger)
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

func (s *integrationMDMTestSuite) cleanupDeclarations(t *testing.T) {
	ctx := context.Background()
	// TODO: figure out the best way to do this. We might even consider
	// starting a different test suite.
	// delete declarations to not affect other tests
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM mdm_apple_declarations")
		return err
	})
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM host_mdm_apple_declarations")
		return err
	})

}
