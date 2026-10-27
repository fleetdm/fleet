package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_google_proxy"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

type googleCheckDelegationCall struct {
	key        []byte
	adminEmail string
	customerID string
}

// mockedConditionalAccessGoogle stands in for both fleetdm.com and Google, recording every call.
type mockedConditionalAccessGoogle struct {
	mu sync.Mutex

	checkErr  error
	createErr error
	deleteErr error

	checks  []googleCheckDelegationCall
	creates []string
	deletes [][2]string // customer ID, secret
}

func (m *mockedConditionalAccessGoogle) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkErr, m.createErr, m.deleteErr = nil, nil, nil
	m.checks, m.creates, m.deletes = nil, nil, nil
}

func (m *mockedConditionalAccessGoogle) Create(_ context.Context, customerID string) (*conditional_access_google_proxy.CreateResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.creates = append(m.creates, customerID)
	return &conditional_access_google_proxy.CreateResponse{
		FleetServerSecret: fmt.Sprintf("secret-%d", len(m.creates)),
		OAuthClientID:     "client.apps.googleusercontent.com",
	}, nil
}

func (m *mockedConditionalAccessGoogle) Delete(_ context.Context, customerID, secret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletes = append(m.deletes, [2]string{customerID, secret})
	return m.deleteErr
}

type mockedConditionalAccessGoogleClient struct {
	m    *mockedConditionalAccessGoogle
	call googleCheckDelegationCall
}

func (c *mockedConditionalAccessGoogleClient) CheckDelegation(context.Context) error {
	c.m.mu.Lock()
	defer c.m.mu.Unlock()
	c.m.checks = append(c.m.checks, c.call)
	return c.m.checkErr
}

func (m *mockedConditionalAccessGoogle) clientFactory(_ context.Context, key []byte, adminEmail, customerID string) (ConditionalAccessGoogleClient, error) {
	return &mockedConditionalAccessGoogleClient{m: m, call: googleCheckDelegationCall{key: key, adminEmail: adminEmail, customerID: customerID}}, nil
}

// counts returns the number of delegation checks, registrations, and deregistrations so far.
func (m *mockedConditionalAccessGoogle) counts() (checks, creates, deletes int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.checks), len(m.creates), len(m.deletes)
}

var mockedConditionalAccessGoogleInstance = &mockedConditionalAccessGoogle{}

type googleProxyNotFoundError struct{}

func (googleProxyNotFoundError) Error() string    { return "not found" }
func (googleProxyNotFoundError) IsNotFound() bool { return true }

func googleConditionalAccessPayload(t *testing.T, entries ...map[string]any) json.RawMessage {
	if entries == nil {
		entries = []map[string]any{}
	}
	b, err := json.Marshal(map[string]any{"integrations": map[string]any{"google_conditional_access": entries}})
	require.NoError(t, err)
	return b
}

func (s *integrationEnterpriseTestSuite) TestConditionalAccessGoogle() {
	t := s.T()
	ctx := t.Context()
	m := mockedConditionalAccessGoogleInstance
	m.reset()

	key := map[string]string{
		"type":         "service_account",
		"client_email": "sa@example.iam.gserviceaccount.com",
		"private_key":  "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----",
	}
	keyJSON, err := json.Marshal(key)
	require.NoError(t, err)
	entry := func(customerID, adminEmail string, apiKey any) map[string]any {
		e := map[string]any{"customer_id": customerID, "admin_email": adminEmail}
		if apiKey != nil {
			e["api_key_json"] = apiKey
		}
		return e
	}

	t.Cleanup(func() {
		ctx := context.WithoutCancel(ctx)
		m.reset()
		s.token = s.getTestAdminToken()
		require.NoError(t, s.ds.ConditionalAccessGoogleDelete(ctx))
		require.NoError(t, s.ds.DeleteMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetGoogleConditionalAccessServiceAccountKey}))
		s.Do("PATCH", "/api/latest/fleet/config", googleConditionalAccessPayload(t), http.StatusOK)
	})

	getConfig := func() (entries []*fleet.GoogleConditionalAccessIntegration, ca fleet.ConditionalAccessSettings, rawEntries []map[string]any) {
		var raw map[string]json.RawMessage
		s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &raw)
		var intgs struct {
			GoogleConditionalAccess json.RawMessage `json:"google_conditional_access"`
		}
		require.NoError(t, json.Unmarshal(raw["integrations"], &intgs))
		if len(intgs.GoogleConditionalAccess) > 0 {
			require.NoError(t, json.Unmarshal(intgs.GoogleConditionalAccess, &entries))
			require.NoError(t, json.Unmarshal(intgs.GoogleConditionalAccess, &rawEntries))
		}
		require.NoError(t, json.Unmarshal(raw["conditional_access"], &ca))
		return entries, ca, rawEntries
	}
	storedKey := func() []byte {
		assets, err := s.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetGoogleConditionalAccessServiceAccountKey}, nil)
		if fleet.IsNotFound(err) {
			return nil
		}
		require.NoError(t, err)
		return assets[fleet.MDMAssetGoogleConditionalAccessServiceAccountKey].Value
	}
	requireNotConnected := func() {
		_, err := s.ds.ConditionalAccessGoogleGet(ctx)
		require.True(t, fleet.IsNotFound(err))
		require.Nil(t, storedKey())
		entries, ca, _ := getConfig()
		require.Empty(t, entries)
		require.Empty(t, ca.GoogleCustomerID)
		require.False(t, ca.GoogleConnectionConfigured)
	}
	added := func() int {
		return s.countActivitiesOfType(fleet.ActivityTypeAddedConditionalAccessIntegrationGoogle{}.ActivityName())
	}
	deleted := func() int {
		return s.countActivitiesOfType(fleet.ActivityTypeDeletedConditionalAccessIntegrationGoogle{}.ActivityName())
	}
	addedBefore, deletedBefore := added(), deleted()

	// Validation (covered case by case in the fleet package) returns 422 before any Google or fleetdm.com call.
	for _, tc := range []struct {
		entries []map[string]any
		want    string
	}{
		{[]map[string]any{entry("01234abc", "admin@example.com", key)}, `customer_id must be a Google customer ID starting with "C"`},
		{[]map[string]any{entry("C01234abc", "admin@example.com", fleet.MaskedPassword)}, "api_key_json is required"},
	} {
		res := s.Do("PATCH", "/api/latest/fleet/config", googleConditionalAccessPayload(t, tc.entries...), http.StatusUnprocessableEntity)
		require.Contains(t, extractServerErrorText(res.Body), tc.want)
	}
	checks, creates, deletes := m.counts()
	require.Zero(t, checks+creates+deletes)
	requireNotConnected()

	// A failed delegation check returns Google's error and stores nothing.
	m.checkErr = errors.New("googleapi: Error 403: Not Authorized to access this resource/api, forbidden")
	res := s.Do("PATCH", "/api/latest/fleet/config", googleConditionalAccessPayload(t, entry("C01234abc", "admin@example.com", key)), http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "Couldn't connect to Google: googleapi: Error 403: Not Authorized")
	checks, creates, _ = m.counts()
	require.Equal(t, 1, checks)
	require.Zero(t, creates)
	requireNotConnected()
	require.Equal(t, addedBefore, added())
	m.reset()

	// Connect.
	s.Do("PATCH", "/api/latest/fleet/config", googleConditionalAccessPayload(t, entry("C01234abc", "admin@example.com", key)), http.StatusOK)
	require.Equal(t, []googleCheckDelegationCall{{key: keyJSON, adminEmail: "admin@example.com", customerID: "C01234abc"}}, m.checks)
	require.Equal(t, []string{"C01234abc"}, m.creates)
	require.Empty(t, m.deletes)
	intg, err := s.ds.ConditionalAccessGoogleGet(ctx)
	require.NoError(t, err)
	require.Equal(t, &fleet.ConditionalAccessGoogleIntegration{
		CustomerID: "C01234abc", ProxyServerSecret: "secret-1", OAuthClientID: "client.apps.googleusercontent.com", SetupDone: true,
	}, intg)
	require.JSONEq(t, string(keyJSON), string(storedKey()))
	entries, ca, rawEntries := getConfig()
	require.Len(t, entries, 1)
	require.Equal(t, "C01234abc", entries[0].CustomerID)
	require.Equal(t, "admin@example.com", entries[0].AdminEmail)
	require.Equal(t, []map[string]any{{
		"customer_id": "C01234abc", "admin_email": "admin@example.com", "api_key_json": fleet.MaskedPassword,
	}}, rawEntries)
	require.Equal(t, "C01234abc", ca.GoogleCustomerID)
	require.True(t, ca.GoogleConnectionConfigured)
	require.Equal(t, addedBefore+1, added())

	// The key is never saved in the app config JSON.
	appConfig, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	require.Len(t, appConfig.Integrations.GoogleConditionalAccess, 1)
	require.True(t, appConfig.Integrations.GoogleConditionalAccess[0].ApiKey.IsEmpty())

	// Saving the same settings again (with the key, masked, or omitted) or leaving the setting
	// out makes no Google or fleetdm.com call and logs nothing. The response reports the
	// connection like GET does, so the UI doesn't show it as disconnected after an unrelated save.
	for _, payload := range []json.RawMessage{
		googleConditionalAccessPayload(t, entry("C01234abc", "admin@example.com", key)),
		googleConditionalAccessPayload(t, entry("C01234abc", "admin@example.com", fleet.MaskedPassword)),
		googleConditionalAccessPayload(t, entry("C01234abc", "admin@example.com", nil)),
		json.RawMessage(`{"integrations": {"google_conditional_access": null}}`),
		json.RawMessage(`{"org_info": {"org_name": "Fleet"}}`),
	} {
		var patchRes struct {
			ConditionalAccess fleet.ConditionalAccessSettings `json:"conditional_access"`
		}
		s.DoJSON("PATCH", "/api/latest/fleet/config", payload, http.StatusOK, &patchRes)
		require.Equal(t, "C01234abc", patchRes.ConditionalAccess.GoogleCustomerID)
		require.True(t, patchRes.ConditionalAccess.GoogleConnectionConfigured)
	}
	checks, creates, deletes = m.counts()
	require.Equal(t, 1, checks)
	require.Equal(t, 1, creates)
	require.Zero(t, deletes)
	require.Equal(t, addedBefore+1, added())
	entries, _, _ = getConfig()
	require.Len(t, entries, 1)

	// Changing only the admin email re-runs the delegation check without re-registering.
	s.Do("PATCH", "/api/latest/fleet/config", googleConditionalAccessPayload(t, entry("C01234abc", "other-admin@example.com", nil)), http.StatusOK)
	checks, creates, deletes = m.counts()
	require.Equal(t, 2, checks)
	require.Equal(t, "other-admin@example.com", m.checks[1].adminEmail)
	require.JSONEq(t, string(keyJSON), string(m.checks[1].key), "the stored key is used")
	require.Equal(t, 1, creates)
	require.Zero(t, deletes)
	intg, err = s.ds.ConditionalAccessGoogleGet(ctx)
	require.NoError(t, err)
	require.Equal(t, "secret-1", intg.ProxyServerSecret)

	// A new key is verified and replaces the stored one.
	newKey := map[string]string{"client_email": "sa2@example.iam.gserviceaccount.com", "private_key": "other"}
	s.Do("PATCH", "/api/latest/fleet/config", googleConditionalAccessPayload(t, entry("C01234abc", "other-admin@example.com", newKey)), http.StatusOK)
	checks, creates, _ = m.counts()
	require.Equal(t, 3, checks)
	require.Equal(t, 1, creates)
	require.JSONEq(t, `{"client_email": "sa2@example.iam.gserviceaccount.com", "private_key": "other"}`, string(storedKey()))

	// A failed registration of a new customer ID leaves the current integration and key untouched.
	m.createErr = errors.New("fleetdm.com is down")
	s.Do("PATCH", "/api/latest/fleet/config", googleConditionalAccessPayload(t, entry("C05678def", "other-admin@example.com", key)), http.StatusInternalServerError)
	m.createErr = nil
	require.Equal(t, []string{"C01234abc"}, m.creates)
	require.Empty(t, m.deletes)
	intg, err = s.ds.ConditionalAccessGoogleGet(ctx)
	require.NoError(t, err)
	require.Equal(t, "C01234abc", intg.CustomerID)
	require.JSONEq(t, `{"client_email": "sa2@example.iam.gserviceaccount.com", "private_key": "other"}`, string(storedKey()))
	_, ca, _ = getConfig()
	require.Equal(t, "C01234abc", ca.GoogleCustomerID)

	// Changing the customer ID registers the new one and deregisters the old one.
	s.Do("PATCH", "/api/latest/fleet/config", googleConditionalAccessPayload(t, entry("C05678def", "other-admin@example.com", nil)), http.StatusOK)
	require.Equal(t, [][2]string{{"C01234abc", "secret-1"}}, m.deletes)
	require.Equal(t, []string{"C01234abc", "C05678def"}, m.creates)
	intg, err = s.ds.ConditionalAccessGoogleGet(ctx)
	require.NoError(t, err)
	require.Equal(t, "C05678def", intg.CustomerID)
	require.Equal(t, "secret-2", intg.ProxyServerSecret)
	_, ca, _ = getConfig()
	require.Equal(t, "C05678def", ca.GoogleCustomerID)
	require.Equal(t, addedBefore+2, added())

	// An empty array disconnects.
	s.Do("PATCH", "/api/latest/fleet/config", googleConditionalAccessPayload(t), http.StatusOK)
	require.Equal(t, [][2]string{{"C01234abc", "secret-1"}, {"C05678def", "secret-2"}}, m.deletes)
	requireNotConnected()
	require.Equal(t, deletedBefore+1, deleted())

	// The DELETE endpoint does the same, and a record missing on fleetdm.com doesn't block it.
	s.Do("PATCH", "/api/latest/fleet/config", googleConditionalAccessPayload(t, entry("C01234abc", "admin@example.com", key)), http.StatusOK)
	m.deleteErr = googleProxyNotFoundError{}
	s.Do("DELETE", "/api/latest/fleet/conditional-access/google", nil, http.StatusOK)
	requireNotConnected()
	require.Equal(t, deletedBefore+2, deleted())
	res = s.Do("DELETE", "/api/latest/fleet/conditional-access/google", nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "integration not found")
	require.Equal(t, deletedBefore+2, deleted())

	// Any other fleetdm.com error blocks removal, so the registration isn't orphaned.
	m.deleteErr = nil
	s.Do("PATCH", "/api/latest/fleet/config", googleConditionalAccessPayload(t, entry("C01234abc", "admin@example.com", key)), http.StatusOK)
	m.deleteErr = errors.New("fleetdm.com is down")
	s.Do("DELETE", "/api/latest/fleet/conditional-access/google", nil, http.StatusInternalServerError)
	_, err = s.ds.ConditionalAccessGoogleGet(ctx)
	require.NoError(t, err)
	m.deleteErr = nil
	s.Do("DELETE", "/api/latest/fleet/conditional-access/google", nil, http.StatusOK)
	requireNotConnected()

	// Only global admins can use the DELETE endpoint; global maintainers and team admins can't.
	s.Do("PATCH", "/api/latest/fleet/config", googleConditionalAccessPayload(t, entry("C01234abc", "admin@example.com", key)), http.StatusOK)
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	for _, u := range []*fleet.User{
		{Name: "gca maintainer", Email: "gca-maintainer@example.com", GlobalRole: new(fleet.RoleMaintainer)},
		{Name: "gca team admin", Email: "gca-team-admin@example.com", Teams: []fleet.UserTeam{{Team: *team, Role: fleet.RoleAdmin}}},
	} {
		require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
		_, err := s.ds.NewUser(ctx, u)
		require.NoError(t, err)
		s.token = s.getTestToken(u.Email, test.GoodPassword)
		s.Do("DELETE", "/api/latest/fleet/conditional-access/google", nil, http.StatusForbidden)
		s.Do("PATCH", "/api/latest/fleet/config", googleConditionalAccessPayload(t), http.StatusForbidden)
	}
	s.token = s.getTestAdminToken()
	_, err = s.ds.ConditionalAccessGoogleGet(ctx)
	require.NoError(t, err)

	// A gitops-role token connects and disconnects through the config, as `fleetctl gitops` does.
	gitops := &fleet.User{Name: "gca gitops", Email: "gca-gitops@example.com", GlobalRole: new(fleet.RoleGitOps), APIOnly: true}
	require.NoError(t, gitops.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(ctx, gitops)
	require.NoError(t, err)
	s.token = s.getTestToken(gitops.Email, test.GoodPassword)
	s.Do("PATCH", "/api/latest/fleet/config", googleConditionalAccessPayload(t), http.StatusOK)
	s.token = s.getTestAdminToken()
	requireNotConnected()
	s.token = s.getTestToken(gitops.Email, test.GoodPassword)
	s.Do("PATCH", "/api/latest/fleet/config", googleConditionalAccessPayload(t, entry("C05678def", "admin@example.com", key)), http.StatusOK)
	s.token = s.getTestAdminToken()
	intg, err = s.ds.ConditionalAccessGoogleGet(ctx)
	require.NoError(t, err)
	require.Equal(t, "C05678def", intg.CustomerID)
}
