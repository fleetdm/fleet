package service

// Apple (macOS) MDM enrollment tests for the MDM suite.
//
// Belongs here: Apple MDM and APNs setup (CSR, certificates, ABM tokens and assets), manual and
// automatic macOS enrollment and unenrollment, enrollment profiles (including the OTA profile) and
// their signatures and SCEP challenges, check-in messages (authenticate, user authenticate,
// bootstrap token), service discovery, the fleetd configuration profile and install command, and
// APNs pushes.
//
// Does not belong here: ADE/DEP profile assignment and release (integration_mdm_dep_test.go),
// iOS/iPadOS and BYOD enrollment (integration_mdm_ios_ipados_test.go), migration from another MDM
// (integration_mdm_migration_test.go), macOS setup experience (integration_mdm_macos_setup_test.go),
// and MDM lifecycle events (integration_mdm_lifecycle_test.go).

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	shared_mdm "github.com/fleetdm/fleet/v4/pkg/mdm"
	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/service/contract"
	"github.com/fleetdm/fleet/v4/server/service/redis_key_value"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/micromdm/plist"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestUserAuthenticate() {
	t := s.T()
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	err = mdmDevice.UserAuthenticate()
	require.ErrorContains(t, err, "410 Gone")
}

func (s *integrationMDMTestSuite) TestGetBootstrapToken() {
	// see https://developer.apple.com/documentation/devicemanagement/get_bootstrap_token
	t := s.T()
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	checkStoredCertAuthAssociation := func(id string, expectedCount uint) {
		// confirm expected cert auth association
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			var ct uint
			// query duplicates the logic in nanomdm/storage/mysql/certauth.go
			if err := sqlx.GetContext(context.Background(), q, &ct, "SELECT COUNT(*) FROM nano_cert_auth_associations WHERE id = ?", mdmDevice.UUID); err != nil {
				return err
			}
			require.Equal(t, expectedCount, ct)
			return nil
		})
	}
	checkStoredCertAuthAssociation(mdmDevice.UUID, 1)

	checkStoredBootstrapToken := func(id string, expectedToken *string, expectedErr error) {
		// confirm expected bootstrap token
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			var tok *string
			err := sqlx.GetContext(context.Background(), q, &tok, "SELECT bootstrap_token_b64 FROM nano_devices WHERE id = ?", mdmDevice.UUID)
			if err != nil || expectedErr != nil {
				require.ErrorIs(t, err, expectedErr)
			} else {
				require.NoError(t, err)
			}

			if expectedToken != nil {
				require.NotEmpty(t, tok)
				decoded, err := base64.StdEncoding.DecodeString(*tok)
				require.NoError(t, err)
				require.Equal(t, *expectedToken, string(decoded))
			} else {
				require.Empty(t, tok)
			}
			return nil
		})
	}

	t.Run("bootstrap token not set", func(t *testing.T) {
		// device record exists, but bootstrap token not set
		checkStoredBootstrapToken(mdmDevice.UUID, nil, nil)

		// if token not set, server returns empty body and no error (see https://github.com/micromdm/nanomdm/pull/63)
		res, err := mdmDevice.GetBootstrapToken()
		require.NoError(t, err)
		require.Nil(t, res)
	})

	t.Run("bootstrap token set", func(t *testing.T) {
		// device record exists, set bootstrap token
		token := base64.StdEncoding.EncodeToString([]byte("testtoken"))
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), "UPDATE nano_devices SET bootstrap_token_b64 = ? WHERE id = ?", base64.StdEncoding.EncodeToString([]byte(token)), mdmDevice.UUID)
			require.NoError(t, err)
			return nil
		})
		checkStoredBootstrapToken(mdmDevice.UUID, &token, nil)

		// if token set, server returns token
		res, err := mdmDevice.GetBootstrapToken()
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, token, string(res))
	})

	t.Run("no device record", func(t *testing.T) {
		// delete the entire device record
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), "DELETE FROM nano_devices WHERE id = ?", mdmDevice.UUID)
			require.NoError(t, err)
			return nil
		})
		checkStoredBootstrapToken(mdmDevice.UUID, nil, sql.ErrNoRows)

		// if not found, server returns empty body and no error (see https://github.com/fleetdm/nanomdm/pull/8)
		res, err := mdmDevice.GetBootstrapToken()
		require.NoError(t, err)
		require.Nil(t, res)
	})

	t.Run("no cert auth association", func(t *testing.T) {
		// on mdm checkout, nano soft deletes by calling storage.Disable, which leaves the cert auth
		// association in place, so what if we hard delete instead?
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), "DELETE FROM nano_cert_auth_associations WHERE id = ?", mdmDevice.UUID)
			require.NoError(t, err)
			return nil
		})
		checkStoredCertAuthAssociation(mdmDevice.UUID, 0)

		// server returns 403 Forbidden for cert auth failures (enrollment not associated with cert)
		res, err := mdmDevice.GetBootstrapToken()
		require.ErrorContains(t, err, "403") // getbootstraptoken service: cert auth: existing enrollment: enrollment not associated with cert
		require.Nil(t, res)
	})
}

func (s *integrationMDMTestSuite) TestAppleGetAppleMDM() {
	t := s.T()

	var mdmResp getAppleMDMResponse
	s.DoJSON("GET", "/api/latest/fleet/apns", nil, http.StatusOK, &mdmResp)
	// returned values are dummy, this is a test certificate
	require.Equal(t, "Fleet", mdmResp.Issuer)
	require.NotEmpty(t, mdmResp.SerialNumber)
	require.Equal(t, "Fleet", mdmResp.CommonName)
	require.NotZero(t, mdmResp.RenewDate)

	var countTokensResp countABMTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/abm_tokens/count", nil, http.StatusOK, &countTokensResp)
	assert.Equal(t, 0, countTokensResp.Count)

	// set up multiple ABM tokens with different org names
	defaultOrgName := "fleet_test"
	s.enableABM(defaultOrgName)
	tmOrgName := t.Name()
	s.enableABM(tmOrgName)

	var tokensResp listABMTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/abm_tokens", nil, http.StatusOK, &tokensResp)

	// for t.Name()
	tok := s.getABMTokenByName(defaultOrgName, tokensResp.Tokens)
	require.NotNil(t, tok)
	require.False(t, tok.TermsExpired)
	require.Equal(t, "abc", tok.AppleID)
	require.Equal(t, defaultOrgName, tok.OrganizationName)
	require.Equal(t, s.server.URL+"/mdm/apple/mdm", tok.MDMServerURL)
	require.Equal(t, fleet.TeamNameNoTeam, tok.MacOSTeam.Name)
	require.Equal(t, fleet.TeamNameNoTeam, tok.IOSTeam.Name)
	require.Equal(t, fleet.TeamNameNoTeam, tok.IPadOSTeam.Name)

	// for tmOrgName
	tok = s.getABMTokenByName(tmOrgName, tokensResp.Tokens)
	require.NotNil(t, tok)
	require.False(t, tok.TermsExpired)
	require.Equal(t, "abc", tok.AppleID)
	require.Equal(t, tmOrgName, tok.OrganizationName)
	require.Equal(t, s.server.URL+"/mdm/apple/mdm", tok.MDMServerURL)
	require.Equal(t, fleet.TeamNameNoTeam, tok.MacOSTeam.Name)
	require.Equal(t, fleet.TeamNameNoTeam, tok.IOSTeam.Name)
	require.Equal(t, fleet.TeamNameNoTeam, tok.IPadOSTeam.Name)

	s.DoJSON("GET", "/api/latest/fleet/abm_tokens/count", nil, http.StatusOK, &countTokensResp)
	assert.Equal(t, 2, countTokensResp.Count)

	// create a new team
	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)
	// set the default bm assignment for that token to that team
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"apple_business_manager": [{
			  "organization_name": %q,
			  "macos_team": %q,
			  "ios_team": %q,
			  "ipados_team": %q
			}]
		}
	}`, tmOrgName, tm.Name, tm.Name, tm.Name)), http.StatusOK, &acResp)
	t.Cleanup(func() {
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"apple_business_manager": []
			}
		}`), http.StatusOK, &acResp)
	})

	// try again, this time we get team assignments in the response
	tokensResp = listABMTokensResponse{}
	s.DoJSON("GET", "/api/latest/fleet/abm_tokens", nil, http.StatusOK, &tokensResp)

	tok = s.getABMTokenByName(tmOrgName, tokensResp.Tokens)
	require.NotNil(t, tok)
	require.False(t, tok.TermsExpired)
	require.Equal(t, "abc", tok.AppleID)
	require.Equal(t, tmOrgName, tok.OrganizationName)
	require.Equal(t, s.server.URL+"/mdm/apple/mdm", tok.MDMServerURL)
	require.Equal(t, tm.Name, tok.MacOSTeam.Name)
	require.Equal(t, tm.Name, tok.IOSTeam.Name)
	require.Equal(t, tm.Name, tok.IPadOSTeam.Name)

	// Reset the teams via app config
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"apple_business_manager": []
			}
		}`), http.StatusOK, &acResp)

	tokensResp = listABMTokensResponse{}
	s.DoJSON("GET", "/api/latest/fleet/abm_tokens", nil, http.StatusOK, &tokensResp)
	tok = s.getABMTokenByName(tmOrgName, tokensResp.Tokens)
	require.NotNil(t, tok)
	require.False(t, tok.TermsExpired)
	require.Equal(t, "abc", tok.AppleID)
	require.Equal(t, tmOrgName, tok.OrganizationName)
	require.Equal(t, s.server.URL+"/mdm/apple/mdm", tok.MDMServerURL)
	require.Equal(t, fleet.TeamNameNoTeam, tok.MacOSTeam.Name)
	require.Equal(t, fleet.TeamNameNoTeam, tok.IOSTeam.Name)
	require.Equal(t, fleet.TeamNameNoTeam, tok.IPadOSTeam.Name)
}

func (s *integrationMDMTestSuite) getABMTokenByName(orgName string, tokens []*fleet.ABMToken) *fleet.ABMToken {
	for _, tok := range tokens {
		if tok.OrganizationName == orgName {
			return tok
		}
	}

	return nil
}

func (s *integrationMDMTestSuite) TestABMTeamPersistsOnConfigChange() {
	t := s.T()

	s.enableABM(t.Name())

	// Same tests, but for teams
	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)

	tokensResp := listABMTokensResponse{}
	s.DoJSON("GET", "/api/latest/fleet/abm_tokens", nil, http.StatusOK, &tokensResp)
	tok := s.getABMTokenByName(t.Name(), tokensResp.Tokens)
	require.NotNil(t, tok)

	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/abm_tokens/%d/teams", tok.ID), json.RawMessage(fmt.Sprintf(`{
    "ios_team_id": %d
	}`, team.ID)), http.StatusOK, &updateABMTokenTeamsResponse{})

	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
    "org_info": {"org_name": "New name"}
	}`), http.StatusOK, &appConfigResponse{})

	s.DoJSON("GET", "/api/latest/fleet/abm_tokens", nil, http.StatusOK, &tokensResp)

	require.Len(t, tokensResp.Tokens, 1)
	assert.Equal(t, team.Name, tokensResp.Tokens[0].IOSTeam.Name)
	assert.Equal(t, team.ID, tokensResp.Tokens[0].IOSTeam.ID)
}

func (s *integrationMDMTestSuite) TestABMTokenDefault() {
	t := s.T()

	orgA := t.Name() + "-org-a"
	orgB := t.Name() + "-org-b"
	s.enableABM(orgA)
	s.enableABM(orgB)

	tokensResp := listABMTokensResponse{}
	s.DoJSON("GET", "/api/latest/fleet/ab_tokens", nil, http.StatusOK, &tokensResp)
	require.Len(t, tokensResp.Tokens, 2)
	tokA := s.getABMTokenByName(orgA, tokensResp.Tokens)
	tokB := s.getABMTokenByName(orgB, tokensResp.Tokens)
	require.NotNil(t, tokA)
	require.NotNil(t, tokB)

	// the first token became the default when it was the only one
	require.True(t, tokA.IsDefault)
	require.False(t, tokB.IsDefault)

	countDefaults := func() (n int, defaultOrg string) {
		s.DoJSON("GET", "/api/latest/fleet/ab_tokens", nil, http.StatusOK, &tokensResp)
		for _, tok := range tokensResp.Tokens {
			if tok.IsDefault {
				n++
				defaultOrg = tok.OrganizationName
			}
		}
		return n, defaultOrg
	}
	appCfgDefaults := func() map[string]bool {
		acResp := appConfigResponse{}
		s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
		m := make(map[string]bool)
		for _, e := range acResp.MDM.AppleBusinessManager.Value {
			m[e.OrganizationName] = e.Default
		}
		return m
	}

	// setting B as default moves it off A in one call
	var setResp setABMTokenDefaultResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/ab_tokens/%d/default", tokB.ID),
		json.RawMessage(`{"default": true}`), http.StatusOK, &setResp)
	require.NotNil(t, setResp.ABMToken)
	assert.True(t, setResp.ABMToken.IsDefault)
	n, defaultOrg := countDefaults()
	assert.Equal(t, 1, n)
	assert.Equal(t, orgB, defaultOrg)
	// GET /config mirrors it on B's entry and drops it from A's
	cfgDefaults := appCfgDefaults()
	assert.True(t, cfgDefaults[orgB])
	assert.False(t, cfgDefaults[orgA])

	// unsetting the default clears it everywhere
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/ab_tokens/%d/default", tokB.ID),
		json.RawMessage(`{"default": false}`), http.StatusOK, &setResp)
	assert.False(t, setResp.ABMToken.IsDefault)
	n, _ = countDefaults()
	assert.Equal(t, 0, n)
	cfgDefaults = appCfgDefaults()
	assert.False(t, cfgDefaults[orgA])
	assert.False(t, cfgDefaults[orgB])

	// unsetting a non-default token is a no-op
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/ab_tokens/%d/default", tokA.ID),
		json.RawMessage(`{"default": false}`), http.StatusOK, &setResp)
	assert.False(t, setResp.ABMToken.IsDefault)

	// unknown token id
	s.Do("PATCH", "/api/latest/fleet/ab_tokens/999999/default",
		json.RawMessage(`{"default": true}`), http.StatusNotFound)

	// omitting "default" is rejected instead of silently clearing the default
	res := s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/ab_tokens/%d/default", tokA.ID),
		json.RawMessage(`{}`), http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "missing required argument")
}

func (s *integrationMDMTestSuite) TestListABMTokensRoles() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	newUser := func(globalRole, teamRole string) fleet.User {
		u := fleet.User{
			Name:  t.Name() + globalRole + teamRole,
			Email: fmt.Sprintf("abm-list-%s%s-%s@example.com", globalRole, teamRole, uuid.NewString()[:8]),
		}
		if globalRole != "" {
			u.GlobalRole = new(globalRole)
		} else {
			u.Teams = []fleet.UserTeam{{Team: *team, Role: teamRole}}
		}
		require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
		_, err := s.ds.NewUser(ctx, &u)
		require.NoError(t, err)
		return u
	}

	cases := []struct {
		name string
		user fleet.User
		want int
	}{
		{"global gitops", newUser(fleet.RoleGitOps, ""), http.StatusOK},
		{"global maintainer", newUser(fleet.RoleMaintainer, ""), http.StatusForbidden},
		{"global observer", newUser(fleet.RoleObserver, ""), http.StatusForbidden},
		{"global technician", newUser(fleet.RoleTechnician, ""), http.StatusForbidden},
		{"fleet gitops", newUser("", fleet.RoleGitOps), http.StatusForbidden},
		{"fleet admin", newUser("", fleet.RoleAdmin), http.StatusForbidden},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s.setTokenForTest(t, c.user.Email, test.GoodPassword)
			var resp listABMTokensResponse
			s.DoJSON("GET", "/api/latest/fleet/ab_tokens", nil, c.want, &resp)
		})
	}
}

func (s *integrationMDMTestSuite) TestABMExpiredToken() {
	t := s.T()

	s.enableABM(t.Name())

	var returnType string
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch returnType {
		case "not_signed":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"code": "T_C_NOT_SIGNED"}`))
		case "unauthorized":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{}`))
		case "success":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"auth_session_token": "abcd"}`))
		default:
			assert.Fail(t, fmt.Sprintf("unexpected return type: %s", returnType))
		}
	}))

	config := s.getConfig()
	require.False(t, config.MDM.AppleBMTermsExpired)

	ctx := context.Background()
	fleetSyncer := apple_mdm.NewDEPService(s.ds, s.depStorage, s.logger)

	// not signed error flips the AppleBMTermsExpired flag
	returnType = "not_signed"
	err := fleetSyncer.RunAssigner(ctx)
	require.ErrorContains(t, err, "T_C_NOT_SIGNED")
	var tokensResp listABMTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/abm_tokens", nil, http.StatusOK, &tokensResp)
	tok := s.getABMTokenByName(t.Name(), tokensResp.Tokens)
	require.NotNil(t, tok)
	require.True(t, tok.TermsExpired)
	config = s.getConfig()
	require.True(t, config.MDM.AppleBMTermsExpired)

	// a successful call clears it
	returnType = "success"
	err = fleetSyncer.RunAssigner(ctx)
	require.NoError(t, err)
	tokensResp = listABMTokensResponse{}
	s.DoJSON("GET", "/api/latest/fleet/abm_tokens", nil, http.StatusOK, &tokensResp)
	tok = s.getABMTokenByName(t.Name(), tokensResp.Tokens)
	require.NotNil(t, tok)
	require.False(t, tok.TermsExpired)

	config = s.getConfig()
	require.False(t, config.MDM.AppleBMTermsExpired)

	// an unauthorized call does not flip the terms expired flag
	returnType = "unauthorized"
	err = fleetSyncer.RunAssigner(ctx)
	require.ErrorContains(t, err, "DEP auth error")
	tokensResp = listABMTokensResponse{}
	s.DoJSON("GET", "/api/latest/fleet/abm_tokens", nil, http.StatusOK, &tokensResp)
	tok = s.getABMTokenByName(t.Name(), tokensResp.Tokens)
	require.NotNil(t, tok)
	require.False(t, tok.TermsExpired)

	config = s.getConfig()
	require.False(t, config.MDM.AppleBMTermsExpired)
}

func (s *integrationMDMTestSuite) TestDeviceMDMManualEnroll() {
	t := s.T()

	token := "token_test_manual_enroll"
	createHostAndDeviceToken(t, s.ds, token)

	// invalid token fails
	s.DoRaw("GET", "/api/latest/fleet/device/invalid_token/mdm/apple/manual_enrollment_profile", nil, http.StatusUnauthorized)

	// valid token downloads the profile
	var r1 getDeviceMDMManualEnrollProfileResponse
	s.DoJSON("GET", "/api/latest/fleet/device/"+token+"/mdm/apple/manual_enrollment_profile", nil, http.StatusOK, &r1)
	require.NotEmpty(t, r1.EnrollURL)
	require.Contains(t, r1.EnrollURL, s.server.URL+"/enroll?enroll_secret=")
	u, err := url.Parse(r1.EnrollURL)
	require.NoError(t, err)
	q, err := url.ParseQuery(u.RawQuery)
	require.NoError(t, err)
	require.NotEmpty(t, q.Get("enroll_secret"))

	// set end user auth enabled
	appConfig, err := s.ds.AppConfig(t.Context())
	require.NoError(t, err)
	appConfig.MDM.MacOSSetup.EnableEndUserAuthentication = true

	err = s.ds.SaveAppConfig(t.Context(), appConfig)
	require.NoError(t, err)

	// works with end user auth enabled
	var r2 getDeviceMDMManualEnrollProfileResponse
	s.DoJSON("GET", "/api/latest/fleet/device/"+token+"/mdm/apple/manual_enrollment_profile", nil, http.StatusOK, &r2)
	require.NotEmpty(t, r2.EnrollURL)
	require.Contains(t, r2.EnrollURL, s.server.URL+"/enroll?enroll_secret=")
	u2, err := url.Parse(r2.EnrollURL)
	require.NoError(t, err)
	q2, err := url.ParseQuery(u2.RawQuery)
	require.NoError(t, err)
	require.NotEmpty(t, q2.Get("enroll_secret"))
}

func (s *integrationMDMTestSuite) TestAppleMDMDeviceEnrollment() {
	t := s.T()
	// Clear out all activities to other test leaking activities into this one.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(), `DELETE FROM activity_past`)
		return err
	})

	// Enroll two devices into MDM
	mdmEnrollInfo := mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}
	mdmDeviceA := mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")
	err := mdmDeviceA.Enroll()
	require.NoError(t, err)
	hostA, err := s.ds.HostByIdentifier(context.Background(), mdmDeviceA.SerialNumber)
	require.NoError(t, err)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeMDMEnrolled{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_serial": "%s", "enrollment_id": null, "host_display_name": "%s (%s)", "installed_from_dep": false, "mdm_platform": "apple", "platform": "darwin"}`, hostA.ID, mdmDeviceA.SerialNumber, mdmDeviceA.Model, mdmDeviceA.SerialNumber), 0)
	mdmDeviceB := mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")
	err = mdmDeviceB.Enroll()
	require.NoError(t, err)
	hostB, err := s.ds.HostByIdentifier(context.Background(), mdmDeviceB.SerialNumber)
	require.NoError(t, err)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeMDMEnrolled{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_serial": "%s", "enrollment_id": null, "host_display_name": "%s (%s)", "installed_from_dep": false, "mdm_platform": "apple", "platform": "darwin"}`, hostB.ID, mdmDeviceB.SerialNumber, mdmDeviceB.Model, mdmDeviceB.SerialNumber), 0)
	// Find the ID of Fleet's MDM solution
	var mdmID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &mdmID,
			`SELECT id FROM mobile_device_management_solutions WHERE name = ?`, fleet.WellKnownMDMFleet)
	})

	// Check that both devices are returned by the /hosts endpoint
	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes, "mdm_id", fmt.Sprint(mdmID))
	require.Len(t, listHostsRes.Hosts, 2)
	require.Equal(
		t,
		[]string{mdmDeviceA.UUID, mdmDeviceB.UUID},
		[]string{listHostsRes.Hosts[0].UUID, listHostsRes.Hosts[1].UUID},
	)

	var targetHostID uint
	var lastEnroll time.Time
	for _, host := range listHostsRes.Hosts {
		if host.UUID == mdmDeviceA.UUID {
			targetHostID = host.ID
			lastEnroll = host.LastEnrolledAt
			break
		}
	}

	// set an enroll secret
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: t.Name()}},
		},
	}, http.StatusOK, &applyResp)

	// simulate a matching host enrolling via osquery
	j, err := json.Marshal(&contract.EnrollOsqueryAgentRequest{
		EnrollSecret:   t.Name(),
		HostIdentifier: mdmDeviceA.UUID,
	})
	require.NoError(t, err)
	var enrollResp contract.EnrollOsqueryAgentResponse
	hres := s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusOK)
	defer hres.Body.Close()
	require.NoError(t, json.NewDecoder(hres.Body).Decode(&enrollResp))
	require.NotEmpty(t, enrollResp.NodeKey)

	// query all hosts
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	// we still have only two hosts
	require.Len(t, listHostsRes.Hosts, 2)

	// LastEnrolledAt should have been updated
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", targetHostID), nil, http.StatusOK, &getHostResp)
	require.Greater(t, getHostResp.Host.LastEnrolledAt, lastEnroll)

	// Unenroll a device
	err = mdmDeviceA.Checkout()
	require.NoError(t, err)

	// An activity is created
	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities)

	found := false
	for _, activity := range activities.Activities {
		if activity.Type == "mdm_unenrolled" {
			found = true
			require.Nil(t, activity.ActorID)
			require.Nil(t, activity.ActorFullName)
			require.JSONEq(t, fmt.Sprintf(`{"host_id": %d, "enrollment_id": null, "host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": false, "platform": "darwin"}`, targetHostID, mdmDeviceA.SerialNumber, mdmDeviceA.Model, mdmDeviceA.SerialNumber), string(*activity.Details))
		}
	}
	require.True(t, found)

	// The unenroll activity must also show on the host details page activity timeline.
	s.lastHostActivityMatches(targetHostID, "mdm_unenrolled",
		fmt.Sprintf(`{"host_id": %d, "enrollment_id": null, "host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": false, "platform": "darwin"}`,
			targetHostID, mdmDeviceA.SerialNumber, mdmDeviceA.Model, mdmDeviceA.SerialNumber), 0)
}

func (s *integrationMDMTestSuite) TestDeviceMultipleAuthMessages() {
	t := s.T()

	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	s.Require().Len(listHostsRes.Hosts, 1)

	// send the auth message again, we still have only one host
	err = mdmDevice.Authenticate()
	require.NoError(t, err)
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	s.Require().Len(listHostsRes.Hosts, 1)
}

func (s *integrationMDMTestSuite) TestAppleMDMCSRRequest() {
	t := s.T()

	var errResp validationErrResp
	// missing arguments
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/request_csr", requestMDMAppleCSRRequest{}, http.StatusUnprocessableEntity, &errResp)
	require.Len(t, errResp.Errors, 1)
	require.Equal(t, "email_address", errResp.Errors[0].Name)

	// invalid email address
	errResp = validationErrResp{}
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/request_csr", requestMDMAppleCSRRequest{EmailAddress: "abc", Organization: "def"}, http.StatusUnprocessableEntity, &errResp)
	require.Len(t, errResp.Errors, 1)
	require.Equal(t, "email_address", errResp.Errors[0].Name)

	// missing organization
	errResp = validationErrResp{}
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/request_csr", requestMDMAppleCSRRequest{EmailAddress: "a@b.c", Organization: ""}, http.StatusUnprocessableEntity, &errResp)
	require.Len(t, errResp.Errors, 1)
	require.Equal(t, "organization", errResp.Errors[0].Name)

	// fleetdm CSR request failed
	s.FailNextCSRRequestWith(http.StatusBadRequest)
	errResp = validationErrResp{}
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/request_csr", requestMDMAppleCSRRequest{EmailAddress: "a@gmail.com", Organization: "test"}, http.StatusUnprocessableEntity, &errResp)
	require.Len(t, errResp.Errors, 1)
	require.Contains(t, errResp.Errors[0].Reason, "CSR request failed. Email domain '@gmail.com' is not permitted for APNS certificate signing. Please use a corporate or organization email address.")

	s.FailNextCSRRequestWith(http.StatusInternalServerError)
	errResp = validationErrResp{}
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/request_csr", requestMDMAppleCSRRequest{EmailAddress: "a@b.c", Organization: "test"}, http.StatusBadGateway, &errResp)
	require.Len(t, errResp.Errors, 1)
	require.Contains(t, errResp.Errors[0].Reason, "FleetDM CSR request failed")

	var reqCSRResp requestMDMAppleCSRResponse
	// fleetdm CSR request succeeds
	s.SucceedNextCSRRequest()
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/request_csr", requestMDMAppleCSRRequest{EmailAddress: "a@b.c", Organization: "test"}, http.StatusOK, &reqCSRResp)
	require.Contains(t, string(reqCSRResp.APNsKey), "-----BEGIN RSA PRIVATE KEY-----\n")
	require.Contains(t, string(reqCSRResp.SCEPCert), "-----BEGIN CERTIFICATE-----\n")
	require.Contains(t, string(reqCSRResp.SCEPKey), "-----BEGIN RSA PRIVATE KEY-----\n")
}

func (s *integrationMDMTestSuite) TestGetMDMCSR() {
	t := s.T()
	ctx := context.Background()

	// Validate errors if no private key is set
	testSetEmptyPrivateKey = true
	t.Cleanup(func() { testSetEmptyPrivateKey = false })
	s.uploadDataViaForm("/api/latest/fleet/mdm/apple/apns_certificate", "certificate", "certificate.pem", []byte("-----BEGIN CERTIFICATE-----\nZm9vCg==\n-----END CERTIFICATE-----"), http.StatusInternalServerError, "Couldn't add APNs certificate. Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key", nil)

	r := s.Do("GET", "/api/latest/fleet/mdm/apple/request_csr", getMDMAppleCSRRequest{}, http.StatusInternalServerError)
	require.Contains(t, extractServerErrorText(r.Body), "Couldn't download signed CSR. Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
	testSetEmptyPrivateKey = false

	// ensure we leave everything in a clean state for other tests
	t.Cleanup(s.appleCoreCertsSetup)

	// Delete APNS cert, should soft delete all certs and keys created in this test
	s.Do("DELETE", "/api/latest/fleet/mdm/apple/apns_certificate", nil, http.StatusOK)

	assets, err := s.ds.GetAllMDMConfigAssetsByName(ctx,
		[]fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey, fleet.MDMAssetAPNSKey, fleet.MDMAssetAPNSCert}, nil)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, assets)

	// trying to upload a certificate without generating a private key first is not allowed
	s.uploadDataViaForm("/api/latest/fleet/mdm/apple/apns_certificate", "certificate", "certificate.pem", []byte("-----BEGIN CERTIFICATE-----\nZm9vCg==\n-----END CERTIFICATE-----"), http.StatusBadRequest, "Couldn't connect. Please download the certificate signing request (CSR) first, then upload APNs certificate.", nil)

	// Check that we return bad gateway if the website API errors
	s.FailNextCSRRequestWith(http.StatusInternalServerError)
	errResp := validationErrResp{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/request_csr", getMDMAppleCSRRequest{}, http.StatusBadGateway, &errResp)
	require.Len(t, errResp.Errors, 1)
	require.Contains(t, errResp.Errors[0].Reason, "FleetDM CSR request failed")

	// Check that we return bad request if the website API does (it will do this in case of an
	// invalid email address
	s.FailNextCSRRequestWith(http.StatusUnprocessableEntity)
	errResp = validationErrResp{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/request_csr", getMDMAppleCSRRequest{}, http.StatusUnprocessableEntity, &errResp)
	require.Len(t, errResp.Errors, 1)
	require.Contains(t, errResp.Errors[0].Reason, "CSR request failed. Email domain '@example.com' is not permitted for APNS certificate signing. Please use a corporate or organization email address.")

	// Invalid APNS cert upload attempt
	s.uploadDataViaForm("/api/latest/fleet/mdm/apple/apns_certificate", "certificate", "certificate.pem", []byte("invalid-cert"), http.StatusUnprocessableEntity, "Invalid certificate. Please provide a valid certificate from Apple Push Certificate Portal.", nil)

	// simulate a renew flow
	s.appleCoreCertsSetup()
}

func (s *integrationMDMTestSuite) TestMDMAppleUnenroll() {
	t := s.T()

	// Enroll a device into MDM.
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// set an enroll secret
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: t.Name()}},
		},
	}, http.StatusOK, &applyResp)

	// simulate a matching host enrolling via osquery
	j, err := json.Marshal(&contract.EnrollOsqueryAgentRequest{
		EnrollSecret:   t.Name(),
		HostIdentifier: mdmDevice.UUID,
	})
	require.NoError(t, err)
	var enrollResp contract.EnrollOsqueryAgentResponse
	hres := s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusOK)
	defer hres.Body.Close()
	require.NoError(t, json.NewDecoder(hres.Body).Decode(&enrollResp))
	require.NotEmpty(t, enrollResp.NodeKey)

	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)
	h := listHostsRes.Hosts[0]

	// assign profiles to the host
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
		mobileconfigForTest("N1", "I1"),
		mobileconfigForTest("N2", "I2"),
		mobileconfigForTest("N3", "I3"),
	}}, http.StatusNoContent)

	// trigger a sync and verify that there are profiles assigned to the host
	s.awaitTriggerProfileSchedule(t)

	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", h.ID), getHostRequest{}, http.StatusOK, &hostResp)
	// 3 profiles added + 1 profile with fleetd configuration + 1 root CA config
	require.Len(t, *hostResp.Host.MDM.Profiles, 5)

	// The error cases below have to run while the host is still enrolled: a
	// failed push returns before MDM is turned off, so the host stays enrolled
	// and each case gets a fresh attempt. Once a turn off succeeds, further
	// requests are rejected as a conflict.

	// we're going to modify this mock, make sure we restore its default
	originalPushMock := s.pushProvider.PushFunc
	defer func() { s.pushProvider.PushFunc = originalPushMock }()

	// if there's an error coming from APNs servers
	s.pushProvider.PushFunc = func(_ context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		return map[string]*push.Response{
			pushes[0].Token.String(): {
				Id:  uuid.New().String(),
				Err: errors.New("test"),
			},
		}, nil
	}
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", h.ID), nil, http.StatusBadGateway)

	// if there was an error unrelated to APNs
	s.pushProvider.PushFunc = func(_ context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		res := map[string]*push.Response{
			pushes[0].Token.String(): {
				Id:  uuid.New().String(),
				Err: nil,
			},
		}
		return res, errors.New("baz")
	}
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", h.ID), nil, http.StatusInternalServerError)

	// try again, but this time the host is online and answers
	var checkoutErr error
	s.pushProvider.PushFunc = func(ctx context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		res, err := mockSuccessfulPush(ctx, pushes)
		checkoutErr = mdmDevice.Checkout()
		return res, err
	}
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", h.ID), nil, http.StatusNoContent)
	// trying again fails with 409 as it is alreayd unenrolled
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", h.ID), nil, http.StatusConflict)

	require.NoError(t, checkoutErr)

	// profiles are removed and the host is no longer enrolled
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", h.ID), getHostRequest{}, http.StatusOK, &hostResp)
	require.Nil(t, hostResp.Host.MDM.Profiles)
	require.Empty(t, hostResp.Host.MDM.Name)
}

func (s *integrationMDMTestSuite) TestEnrollOrbitAfterDEPSync() {
	t := s.T()
	ctx := context.Background()

	// create a host with minimal information and the serial, no uuid/osquery id
	// (as when created via DEP sync).
	dbZeroTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	h, err := s.ds.NewHost(ctx, &fleet.Host{
		HardwareSerial:   uuid.New().String(),
		Platform:         "darwin",
		LastEnrolledAt:   dbZeroTime,
		DetailUpdatedAt:  dbZeroTime,
		RefetchRequested: true,
	})
	require.NoError(t, err)
	require.Equal(t, dbZeroTime, h.LastEnrolledAt)

	// create an enroll secret
	secret := uuid.New().String()
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: secret}},
		},
	}, http.StatusOK, &applyResp)

	// enroll the host from orbit, it should match the host above via the serial
	var resp enrollOrbitResponse
	hostUUID := uuid.New().String()
	h.ComputerName = "My Mac"
	h.HardwareModel = "MacBook Pro"
	s.DoJSON("POST", "/api/fleet/orbit/enroll", fleet.EnrollOrbitRequest{
		EnrollSecret:   secret,
		HardwareUUID:   hostUUID, // will not match any existing host
		HardwareSerial: h.HardwareSerial,
		ComputerName:   h.ComputerName,
		HardwareModel:  h.HardwareModel,
	}, http.StatusOK, &resp)
	require.NotEmpty(t, resp.OrbitNodeKey)

	// fetch the host, it will match the one created above
	// (NOTE: cannot check the returned OrbitNodeKey, this field is not part of the response)
	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID), nil, http.StatusOK, &hostResp)
	require.Equal(t, h.ID, hostResp.Host.ID)
	require.NotEqual(t, dbZeroTime, hostResp.Host.LastEnrolledAt)
	assert.Equal(t, h.ComputerName, hostResp.Host.ComputerName)
	assert.Equal(t, h.HardwareModel, hostResp.Host.HardwareModel)
	assert.Equal(t, h.HardwareSerial, hostResp.Host.HardwareSerial)
	assert.Equal(t, h.DisplayName(), hostResp.Host.DisplayName)

	got, err := s.ds.LoadHostByOrbitNodeKey(ctx, resp.OrbitNodeKey)
	require.NoError(t, err)
	require.Equal(t, h.ID, got.ID)

	s.lastActivityMatches(
		"fleet_enrolled",
		fmt.Sprintf(`{"host_id": %v, "host_display_name": "%s", "host_serial": "%s"}`, h.ID, h.DisplayName(), h.HardwareSerial),
		0,
	)

	// enroll the host from osquery, it should match the same host
	var osqueryResp contract.EnrollOsqueryAgentResponse
	osqueryID := uuid.New().String()
	s.DoJSON("POST", "/api/osquery/enroll", contract.EnrollOsqueryAgentRequest{
		EnrollSecret:   secret,
		HostIdentifier: osqueryID, // osquery host_identifier may not be the same as the host UUID, simulate that here
		HostDetails: map[string]map[string]string{
			"system_info": {
				"uuid":            hostUUID,
				"hardware_serial": h.HardwareSerial,
			},
		},
	}, http.StatusOK, &osqueryResp)
	require.NotEmpty(t, osqueryResp.NodeKey)

	// load the host by osquery node key, should match the initial host
	got, err = s.ds.LoadHostByNodeKey(ctx, osqueryResp.NodeKey)
	require.NoError(t, err)
	require.Equal(t, h.ID, got.ID)
}

func (s *integrationMDMTestSuite) TestFleetdConfiguration() {
	t := s.T()
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetdConfigPayloadIdentifier, false)

	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: t.Name()}},
		},
	}, http.StatusOK, &applyResp)

	// a new fleetd configuration profile for "no team" is created
	s.awaitTriggerProfileSchedule(t)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetdConfigPayloadIdentifier, true)

	// create a new team
	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)
	s.assertConfigProfilesByIdentifier(&tm.ID, mobileconfig.FleetdConfigPayloadIdentifier, false)

	// upload an ABM token
	s.enableABM(t.Name())

	// set the default bm assignment to that team
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"apple_business_manager": [{
			  "organization_name": %q,
			  "macos_team": %q,
			  "ios_team": %q,
			  "ipados_team": %q
			}]
		}
	}`, t.Name(), tm.Name, tm.Name, tm.Name)), http.StatusOK, &acResp)
	t.Cleanup(func() {
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"apple_business_manager": []
			}
		}`), http.StatusOK, &acResp)
	})

	// the team doesn't have any enroll secrets yet, a profile is created using the global enroll secret
	s.awaitTriggerProfileSchedule(t)
	p := s.assertConfigProfilesByIdentifier(&tm.ID, mobileconfig.FleetdConfigPayloadIdentifier, true)
	require.Contains(t, string(p.Mobileconfig), t.Name())

	// create an enroll secret for the team
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name:    tm.Name,
		Secrets: &[]fleet.EnrollSecret{{Secret: t.Name() + "team-secret"}},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// a new fleetd configuration profile for that team is created
	s.awaitTriggerProfileSchedule(t)
	p = s.assertConfigProfilesByIdentifier(&tm.ID, mobileconfig.FleetdConfigPayloadIdentifier, true)
	require.Contains(t, string(p.Mobileconfig), t.Name()+"team-secret") // nolint:nilaway // assertConfigProfilesByIdentifier fails the test when the profile doesn't exist

	// the old configuration profile is kept
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetdConfigPayloadIdentifier, true)
}

func (s *integrationMDMTestSuite) downloadAndVerifyOTAEnrollmentProfile(path string) {
	t := s.T()

	resp := s.DoRaw("GET", path, nil, http.StatusOK)
	rawProfile, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	require.Contains(t, resp.Header, "Content-Disposition")
	require.Contains(t, resp.Header, "Content-Type")
	require.Contains(t, resp.Header, "X-Content-Type-Options")
	require.Contains(t, resp.Header.Get("Content-Disposition"), "attachment;")
	require.Contains(t, resp.Header.Get("Content-Type"), "application/x-apple-aspen-config")
	require.Contains(t, resp.Header.Get("X-Content-Type-Options"), "nosniff")
	headerLen, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	require.NoError(t, err)
	require.Equal(t, len(rawProfile), headerLen)

	p7, err := pkcs7.Parse(rawProfile)
	require.NoError(t, err)
	rootCA := x509.NewCertPool()

	assets, err := s.ds.GetAllMDMConfigAssetsByName(context.Background(), []fleet.MDMAssetName{
		fleet.MDMAssetCACert,
	}, nil)
	require.NoError(t, err)

	require.True(t, rootCA.AppendCertsFromPEM(assets[fleet.MDMAssetCACert].Value))
	require.NoError(t, p7.VerifyWithChain(rootCA))

	var otaEnrollmentProfile struct {
		PayloadContent struct {
			URL string `plist:"URL"`
		} `plist:"PayloadContent"`
	}
	err = plist.Unmarshal(p7.Content, &otaEnrollmentProfile)
	require.NoError(t, err)
	require.Contains(t, otaEnrollmentProfile.PayloadContent.URL, s.getConfig().ServerSettings.ServerURL+"/api/v1/fleet/ota_enrollment")
}

func (s *integrationMDMTestSuite) downloadAndVerifyEnrollmentProfileManual(t *testing.T) {
	resp := s.DoRaw("GET", "/api/latest/fleet/enrollment_profiles/manual", nil, http.StatusOK)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, resp.Header, "Content-Disposition")
	require.Contains(t, resp.Header, "Content-Type")
	require.Contains(t, resp.Header, "X-Content-Type-Options")
	require.Contains(t, resp.Header.Get("Content-Disposition"), "attachment;")
	require.Contains(t, resp.Header.Get("Content-Type"), "application/x-apple-aspen-config")
	require.Contains(t, resp.Header.Get("X-Content-Type-Options"), "nosniff")
	headerLen, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	require.NoError(t, err)
	require.Equal(t, len(body), headerLen)

	s.verifyEnrollmentProfile(body, "", "")
}

func (s *integrationMDMTestSuite) TestManualEnrollmentCommands() {
	t := s.T()

	// ensure fleet profiles
	s.awaitTriggerProfileSchedule(t)

	// create a device that's not enrolled into Fleet, it should get a command to
	// install fleetd
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	err := mdmDevice.Enroll()
	require.NoError(t, err)
	s.awaitRunAppleMDMWorkerSchedule()
	checkInstallFleetdCommandSent(t, mdmDevice, true)

	// create a device that's enrolled into Fleet before turning on MDM features,
	// it should still get the command to install fleetd if turns on MDM.
	desktopToken := uuid.New().String()
	host := createOrbitEnrolledHost(t, "darwin", "h1", s.ds)
	err = s.ds.SetOrUpdateDeviceAuthToken(context.Background(), host.ID, desktopToken)
	require.NoError(t, err)
	mdmDevice = mdmtest.NewTestMDMClientAppleDesktopManual(s.server.URL, desktopToken)
	mdmDevice.UUID = host.UUID
	mdmDevice.SerialNumber = host.HardwareSerial
	err = mdmDevice.Enroll()
	require.NoError(t, err)
	s.awaitRunAppleMDMWorkerSchedule()
	checkInstallFleetdCommandSent(t, mdmDevice, true)
}

func (s *integrationMDMTestSuite) TestABMAssetManagement() {
	t := s.T()
	ctx := context.Background()

	s.enableABM(t.Name())

	// Validate error when server private key not set
	testSetEmptyPrivateKey = true
	t.Cleanup(func() { testSetEmptyPrivateKey = false })

	r := s.Do("GET", "/api/latest/fleet/mdm/apple/ab_public_key", generateABMKeyPairResponse{}, http.StatusInternalServerError)
	require.Contains(t, extractServerErrorText(r.Body), "Couldn't download public key. Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
	testSetEmptyPrivateKey = false

	// grab the current public key
	var abmResp generateABMKeyPairResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/ab_public_key", nil, http.StatusOK, &abmResp)
	require.NoError(t, abmResp.Err)
	require.NotEmpty(t, abmResp.PublicKey)

	// the deprecated abm_public_key path still resolves to the same endpoint
	var deprecatedResp generateABMKeyPairResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/abm_public_key", nil, http.StatusOK, &deprecatedResp)
	require.NoError(t, deprecatedResp.Err)
	require.NotEmpty(t, deprecatedResp.PublicKey)

	var tokensResp listABMTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/abm_tokens", nil, http.StatusOK, &tokensResp)
	tok := s.getABMTokenByName(t.Name(), tokensResp.Tokens)
	require.NotNil(t, tok)

	// disable ABM
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/abm_tokens/%d", tok.ID), nil, http.StatusNoContent)
	tok, err := s.ds.GetABMTokenByOrgName(ctx, t.Name())
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, tok)

	// try to upload an invalid token
	s.uploadABMToken([]byte("foo"), http.StatusBadRequest, "Please provide a valid token from Apple Business")

	// enable ABM again
	var newABMResp generateABMKeyPairResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/ab_public_key", nil, http.StatusOK, &newABMResp)
	require.NoError(t, newABMResp.Err)
	require.NotEmpty(t, newABMResp.PublicKey)
	block, _ := pem.Decode(newABMResp.PublicKey)
	require.NotNil(t, block)
	require.Equal(t, "CERTIFICATE", block.Type)

	// we should always return the same values to support renewing the token
	var renewABMResp generateABMKeyPairResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/ab_public_key", nil, http.StatusOK, &renewABMResp)
	require.NoError(t, renewABMResp.Err)
	require.NotEmpty(t, renewABMResp.PublicKey)
	require.Equal(t, renewABMResp.PublicKey, newABMResp.PublicKey)

	// simulate a renew flow
	s.enableABM(t.Name())
}

func (s *integrationMDMTestSuite) TestAPNsPushCron() {
	t := s.T()
	ctx := context.Background()
	kv := redis_key_value.New(s.redisPool)

	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: "N2", Contents: syncMLForTest("./Foo/Bar")},
		{Name: "N4", Contents: declarationForTest("D1")},
	}}, http.StatusNoContent)

	// macOS host, MDM on
	macHost, macDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	// windows host, MDM on
	createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	// linux and darwin, MDM off
	createOrbitEnrolledHost(t, "linux", "linux_host", s.ds)
	createOrbitEnrolledHost(t, "darwin", "mac_not_enrolled", s.ds)

	// Delete the key to let the reconciler pick up the profiles
	err := kv.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+macHost.UUID)
	require.NoError(t, err)

	// we're going to modify this mock, make sure we restore its default
	originalPushMock := s.pushProvider.PushFunc
	defer func() { s.pushProvider.PushFunc = originalPushMock }()

	var recordedPushes []*mdm.Push
	var mu sync.Mutex
	s.pushProvider.PushFunc = func(ctx context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		recordedPushes = pushes
		return mockSuccessfulPush(ctx, pushes)
	}

	// trigger the reconciliation schedule
	err = ReconcileAppleProfilesBatched(ctx, s.ds, s.mdmCommander, kv, s.logger, 0, false)
	require.NoError(t, err)
	require.Len(t, recordedPushes, 1)
	recordedPushes = nil

	// triggering the schedule again doesn't send any more pushes
	err = ReconcileAppleProfilesBatched(ctx, s.ds, s.mdmCommander, kv, s.logger, 0, false)
	require.NoError(t, err)
	require.Empty(t, recordedPushes)
	recordedPushes = nil

	// the cron to trigger pushes sends a new push request each time it
	// runs if there are pending commands
	for range 3 {
		err := SendPushesToPendingDevices(ctx, s.ds, s.mdmCommander, s.logger)
		require.NoError(t, err)
		require.Len(t, recordedPushes, 1)
		recordedPushes = nil
	}

	// device acknowledges the commands
	cmd, err := macDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	for cmd != nil {
		cmd, err = macDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	// no more pushes are enqueued
	err = SendPushesToPendingDevices(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	require.Empty(t, recordedPushes)
}

func (s *integrationMDMTestSuite) TestAPNsPushWithNotNow() {
	t := s.T()
	ctx := context.Background()
	kv := redis_key_value.New(s.redisPool)

	// macOS host, MDM on
	_, macDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	// windows host, MDM on
	createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	// linux and darwin, MDM off
	createOrbitEnrolledHost(t, "linux", "linux_host", s.ds)
	createOrbitEnrolledHost(t, "darwin", "mac_not_enrolled", s.ds)

	// Delete the key to let the reconciler pick up the profiles
	err := kv.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+macDevice.UUID)
	require.NoError(t, err)

	// we're going to modify this mock, make sure we restore its default
	originalPushMock := s.pushProvider.PushFunc
	defer func() { s.pushProvider.PushFunc = originalPushMock }()

	var recordedPushes []*mdm.Push
	var mu sync.Mutex
	s.pushProvider.PushFunc = func(ctx context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		recordedPushes = pushes
		return mockSuccessfulPush(ctx, pushes)
	}

	// trigger the reconciliation schedule
	err = ReconcileAppleProfilesBatched(ctx, s.ds, s.mdmCommander, kv, s.logger, 0, false)
	require.NoError(t, err)
	require.Len(t, recordedPushes, 1)
	recordedPushes = nil

	// Flush any existing profiles.
	cmd, err := macDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		t.Logf("Received: %s %s", cmd.CommandUUID, cmd.Command.RequestType)
		cmd, err = macDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	// Load new profiles
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: "N2", Contents: syncMLForTest("./Foo/Bar")},
	}}, http.StatusNoContent)

	// trigger the reconciliation schedule
	err = ReconcileAppleProfilesBatched(ctx, s.ds, s.mdmCommander, kv, s.logger, 0, false)
	require.NoError(t, err)
	require.Len(t, recordedPushes, 1)
	recordedPushes = nil

	// The cron to trigger pushes sends a new push request each time it runs.
	err = SendPushesToPendingDevices(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	require.Len(t, recordedPushes, 1)
	recordedPushes = nil

	// device sends 'NotNow'
	cmd, err = macDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	cmd, err = macDevice.NotNow(cmd.CommandUUID)
	require.NoError(t, err)
	assert.Nil(t, cmd)

	// A 'NotNow' command will not trigger a new push. Device is expected to check in again when conditions change.
	err = ReconcileAppleProfilesBatched(ctx, s.ds, s.mdmCommander, kv, s.logger, 0, false)
	require.NoError(t, err)
	require.Empty(t, recordedPushes)
	recordedPushes = nil

	// device acknowledges the commands
	cmd, err = macDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	cmd, err = macDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)
	assert.Nil(t, cmd)

	// no more pushes are enqueued
	err = SendPushesToPendingDevices(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	assert.Zero(t, recordedPushes)
}

func (s *integrationMDMTestSuite) TestMDMRequestWithoutCerts() {
	t := s.T()
	res := s.DoRawNoAuth("PUT", "/mdm/apple/mdm", nil, http.StatusBadRequest)
	require.NoError(t, res.Body.Close())
}

func (s *integrationMDMTestSuite) TestMDMEnrollDoesntClearLastEnrolledAtForMacOS() {
	t := s.T()

	// Enroll to Fleet with fleetd first.
	desktopToken := uuid.New().String()
	mdmDevice := mdmtest.NewTestMDMClientAppleDesktopManual(s.server.URL, desktopToken)
	fleetHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		LastEnrolledAt:  time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name() + uuid.New().String()),
		NodeKey:         new(t.Name() + uuid.New().String()),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",

		UUID:           mdmDevice.UUID,
		HardwareSerial: mdmDevice.SerialNumber,
	})
	require.NoError(t, err)
	err = s.ds.SetOrUpdateDeviceAuthToken(context.Background(), fleetHost.ID, desktopToken)
	require.NoError(t, err)
	hostResp := getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", fleetHost.ID), nil, http.StatusOK, &hostResp)
	require.NotNil(t, hostResp.Host)
	lastEnrolledAt := hostResp.Host.LastEnrolledAt

	assert.Nil(t, hostResp.Host.LastMDMEnrolledAt)
	assert.Nil(t, hostResp.Host.LastMDMCheckedInAt)

	// Add the following here for debug: time.Sleep(2 * time.Second)

	// Enroll with MDM manually after.
	enrollTime := time.Now().UTC().Truncate(time.Second)
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", fleetHost.ID), nil, http.StatusOK, &hostResp)
	require.NotNil(t, hostResp.Host)
	assert.Equal(t, lastEnrolledAt, hostResp.Host.LastEnrolledAt)
	assert.NotNil(t, hostResp.Host.LastMDMEnrolledAt)
	assert.GreaterOrEqual(t, *hostResp.Host.LastMDMEnrolledAt, enrollTime)
	assert.NotNil(t, hostResp.Host.LastMDMCheckedInAt)
	assert.GreaterOrEqual(t, *hostResp.Host.LastMDMCheckedInAt, enrollTime)
}

func (s *integrationMDMTestSuite) TestConnectedToFleetWithoutCheckout() {
	t := s.T()
	ctx := context.Background()

	host := createOrbitEnrolledHost(t, "darwin", t.Name(), s.ds)

	// simulate that's actually enrolled to Fleet under the hood
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	mdmDevice.UUID = host.UUID
	mdmDevice.SerialNumber = host.HardwareSerial
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// but, set the host as enrolled in a third-party MDM
	err = s.ds.SetOrUpdateMDMData(ctx, host.ID, true, true, "https://foo.com", false, fleet.WellKnownMDMSimpleMDM, "", fleet.PersonalEnrollmentTypeNone)
	require.NoError(t, err)

	// host is connected to Fleet
	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.NotNil(t, hostResp.Host)
	require.NotNil(t, hostResp.Host.MDM.ConnectedToFleet)
	require.True(t, *hostResp.Host.MDM.ConnectedToFleet)

	// now simulate an un-enrollment without checkout, in this case, osquery reports the host as not-enrolled
	err = s.ds.SetOrUpdateMDMData(ctx, host.ID, false, false, "", false, "", "", fleet.PersonalEnrollmentTypeNone)
	require.NoError(t, err)

	// host is not connected to Fleet anymore
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.NotNil(t, hostResp.Host)
	require.NotNil(t, hostResp.Host.MDM.ConnectedToFleet)
	require.False(t, *hostResp.Host.MDM.ConnectedToFleet)
}

func (s *integrationMDMTestSuite) TestEnrollmentProfilesWithSpecialChars() {
	t := s.T()
	// machine info blobs in this test are signed with a throwaway cert, not an
	// Apple device identity
	apple_mdm.SetMachineInfoVerificationForTest(t, false)
	ctx := context.Background()

	initialConfig, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	initialSecrets, err := s.ds.GetEnrollSecrets(ctx, nil)
	require.NoError(t, err)

	nameWithInvalidChars := "Fleet & Device <3 Management"
	/* #nosec G101 -- this is a made up value for tests */
	enrollSecretWithInvalidChars := "1<2>3&4&/"

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"org_info": {
			"org_name": %q
		}
	}`, nameWithInvalidChars)), http.StatusOK, &acResp)
	enrollSecretResp := applyEnrollSecretSpecResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: enrollSecretWithInvalidChars}},
		},
	}, http.StatusOK, &enrollSecretResp)
	t.Cleanup(func() {
		acResp := appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		  "org_info": {
			  "org_name": %q
		  }
		}`, initialConfig.OrgInfo.OrgName)), http.StatusOK, &acResp)
		s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
			Spec: &fleet.EnrollSecretSpec{
				Secrets: initialSecrets,
			},
		}, http.StatusOK, &enrollSecretResp)
	})

	// manual enrollment from My Device
	token := "token_test_manual_enroll"
	host := createHostAndDeviceToken(t, s.ds, token)
	var r getDeviceMDMManualEnrollProfileResponse
	s.DoJSON("GET", "/api/latest/fleet/device/"+token+"/mdm/apple/manual_enrollment_profile", nil, http.StatusOK, &r)
	u, err := url.Parse(r.EnrollURL)
	require.NoError(t, err)
	q, err := url.ParseQuery(u.RawQuery)
	require.NoError(t, err)
	require.Equal(t, enrollSecretWithInvalidChars, q.Get("enroll_secret"))
	s.downloadAndVerifyOTAEnrollmentProfile("/api/latest/fleet/enrollment_profiles/ota?enroll_secret=" + url.QueryEscape(q.Get("enroll_secret")))

	// automatic enrollment by token
	rawMsg := json.RawMessage(`{"allow_pairing": true}`)
	_, err = s.ds.NewMDMAppleEnrollmentProfile(ctx, fleet.MDMAppleEnrollmentProfilePayload{
		Type:       "automatic",
		DEPProfile: &rawMsg,
		Token:      "abcd",
	})
	require.NoError(t, err)

	s.createDEPAssignmentForHost(t, host)
	di, err := mdmtest.EncodeDeviceInfo(fleet.MDMAppleMachineInfo{
		Serial:                 host.HardwareSerial,
		UDID:                   host.UUID,
		Product:                "Mac13,1",
		SoftwareUpdateDeviceID: "bogus-update-id",
	})
	require.NoError(t, err)
	s.downloadAndVerifyEnrollmentProfile(t, optsDownloadEnrollProf{
		basePath: apple_mdm.EnrollPath,
		token:    "abcd",
		diParam:  di,
	})

	// unsigned manual enrollment profile for IT admins
	s.downloadAndVerifyEnrollmentProfileManual(t)

	// Verify that macOS enrollment inserts entry into os updates tracking
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var count int
		err := sqlx.GetContext(ctx, q, &count, `SELECT COUNT(*) FROM host_mdm_apple_os_updates WHERE host_uuid = ?`, host.UUID)
		require.NoError(t, err)
		require.Equal(t, 1, count)
		return nil
	})

	// ensure the fleetd profile sends a good enroll secret too
	s.awaitTriggerProfileSchedule(t)
	prof := s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetdConfigPayloadIdentifier, true)

	type fleetdPlist struct {
		PayloadContent []struct {
			EnrollSecret string `plist:"EnrollSecret"`
		} `plist:"PayloadContent"`
	}

	// Parse the plist data
	var parsedData fleetdPlist
	err = plist.NewDecoder(bytes.NewReader(prof.Mobileconfig)).Decode(&parsedData)
	require.NoError(t, err)
	require.Equal(t, enrollSecretWithInvalidChars, parsedData.PayloadContent[0].EnrollSecret)
}

func (s *integrationMDMTestSuite) TestMachineInfoSignatureEnforcement() {
	t := s.T()

	// signed with a throwaway cert, not an Apple device identity, so signature
	// verification must fail
	di, err := mdmtest.EncodeDeviceInfo(fleet.MDMAppleMachineInfo{
		Serial:    "FAKESERIAL123",
		UDID:      "00000000-0000-0000-0000-000000000000",
		Product:   "MacBookPro16,1",
		OSVersion: "15.0",
	})
	require.NoError(t, err)

	t.Run("enroll endpoint rejects unverifiable deviceinfo", func(t *testing.T) {
		res := s.DoRawNoAuth("GET", apple_mdm.EnrollPath+"?token=unused&deviceinfo="+di, nil, http.StatusBadRequest)
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, "unable to verify deviceinfo signature")
		require.NoError(t, res.Body.Close())
	})

	t.Run("sso middleware rejects unverifiable deviceinfo header", func(t *testing.T) {
		request, err := http.NewRequest("GET", s.server.URL+"/mdm/sso", nil)
		require.NoError(t, err)
		request.Header.Set("x-apple-aspen-deviceinfo", di)
		// nolint:gosec // this client is used for testing only
		cc := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
			InsecureSkipVerify: true,
		}))
		response, err := cc.Do(request)
		require.NoError(t, err)
		defer response.Body.Close()
		require.Equal(t, http.StatusForbidden, response.StatusCode)
	})

	t.Run("sso middleware rejects an unparseable deviceinfo header", func(t *testing.T) {
		request, err := http.NewRequest("GET", s.server.URL+"/mdm/sso", nil)
		require.NoError(t, err)
		request.Header.Set("x-apple-aspen-deviceinfo", "not-base64-or-pkcs7!!!")
		// nolint:gosec // this client is used for testing only
		cc := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
			InsecureSkipVerify: true,
		}))
		response, err := cc.Do(request)
		require.NoError(t, err)
		defer response.Body.Close()
		require.Equal(t, http.StatusForbidden, response.StatusCode)
	})

	t.Run("account-driven enroll rejects unverifiable deviceinfo", func(t *testing.T) {
		body, err := mdmtest.AccountDrivenUserEnrollDeviceInfoAsPKCS7(fleet.MDMAppleAccountDrivenUserEnrollDeviceInfo{
			Product: "iPhone14,5",
			Version: "22A3351",
		})
		require.NoError(t, err)
		path := strings.Replace(apple_mdm.AccountDrivenEnrollTokenPath, "{token}", "unused", 1)
		res := s.DoRawNoAuth("POST", path, body, http.StatusBadRequest)
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, "unable to verify deviceinfo signature")
		require.NoError(t, res.Body.Close())
	})

	t.Run("audit mode logs but does not block", func(t *testing.T) {
		apple_mdm.SetMachineInfoVerificationForTest(t, false)
		// the request proceeds past signature verification and fails later on
		// the unknown enrollment token instead
		res := s.DoRawNoAuth("GET", apple_mdm.EnrollPath+"?token=unused&deviceinfo="+di, nil, http.StatusUnauthorized)
		require.NoError(t, res.Body.Close())
	})
}

func (s *integrationMDMTestSuite) TestServiceDiscovery() {
	t := s.T()

	type serviceDiscoveryServer struct {
		Version string `json:"Version"`
		BaseURL string `json:"BaseURL"`
	}
	type serviceDiscoveryResponse struct {
		Servers []serviceDiscoveryServer `json:"Servers"`
	}

	res := serviceDiscoveryResponse{}
	s.DoJSON("GET", "/mdm/apple/service_discovery", nil, http.StatusOK, &res)
	require.Contains(t, res.Servers[0].BaseURL, apple_mdm.AccountDrivenEnrollPath) // nolint:staticcheck // kept for backwards compatible testing
	require.Equal(t, "mdm-byod", res.Servers[0].Version)

	// Verify /{token} path works and passes it through
	s.DoJSON("GET", "/mdm/apple/service_discovery/fake-token", nil, http.StatusOK, &res)
	require.Contains(t, res.Servers[0].BaseURL, strings.Replace(apple_mdm.AccountDrivenEnrollTokenPath, "{token}", "fake-token", 1))
	require.Equal(t, "mdm-byod", res.Servers[0].Version)
}

func (s *integrationMDMTestSuite) TestErrorOnEnrollmentInstallProfileProducesActivity() {
	t := s.T()
	ctx := t.Context()

	// Enrolling the host populates nano_cert_auth_associations for it.
	host, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	setRenewCommandUUID := func(cmdUUID *string) {
		t.Helper()
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(
				ctx,
				`UPDATE nano_cert_auth_associations SET renew_command_uuid = ? WHERE id = ?`,
				cmdUUID, host.UUID,
			)
			return err
		})
	}

	// nano_cert_auth_associations.renew_command_uuid references nano_commands,
	// so we have to insert a placeholder command before pointing at it.
	insertNanoCommand := func(cmdUUID string) {
		t.Helper()
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(
				ctx,
				`INSERT INTO nano_commands (command_uuid, request_type, command) VALUES (?, 'InstallProfile', '<?xml version="1.0"?>')`,
				cmdUUID,
			)
			return err
		})
	}

	activityName := fleet.ActivityTypeFailedEnrollmentProfileRenewal{}.ActivityName()

	countRenewalActivitiesForCmd := func(cmdUUID string) int {
		t.Helper()
		var n int
		for _, act := range s.listActivities() {
			if act.Type != activityName || act.Details == nil {
				continue
			}
			var d struct {
				CommandUUID string `json:"command_uuid"`
			}
			require.NoError(t, json.Unmarshal(*act.Details, &d))
			if d.CommandUUID == cmdUUID {
				n++
			}
		}
		return n
	}

	failed := fleet.MDMDeliveryFailed
	verifying := fleet.MDMDeliveryVerifying

	// Case 1: failure for a command that has no host_mdm_apple_profiles row and
	// no renew_command_uuid pointing at it — no activity should be produced.
	// HandleHostMDMProfileInstallResult will surface the underlying not-found
	// error from the retry-count lookup; we deliberately ignore it.
	setRenewCommandUUID(nil)
	case1Cmd := uuid.NewString()
	_ = apple_mdm.HandleHostMDMProfileInstallResult(ctx, s.ds, host.UUID, case1Cmd, &failed, "boom", s.fleetSvc.NewActivity)
	require.Zero(t, countRenewalActivitiesForCmd(case1Cmd))

	// Case 2: failure with a renew_command_uuid set on the host, but the
	// failing command's UUID does not match it — no activity.
	case2RenewCmd := uuid.NewString()
	insertNanoCommand(case2RenewCmd)
	setRenewCommandUUID(&case2RenewCmd)
	case2OtherCmd := uuid.NewString()
	_ = apple_mdm.HandleHostMDMProfileInstallResult(ctx, s.ds, host.UUID, case2OtherCmd, &failed, "boom", s.fleetSvc.NewActivity)
	require.Zero(t, countRenewalActivitiesForCmd(case2OtherCmd))
	require.Zero(t, countRenewalActivitiesForCmd(case2RenewCmd))

	// Case 3: failure where the command UUID matches the renew_command_uuid —
	// activity is produced.
	case3RenewCmd := uuid.NewString()
	insertNanoCommand(case3RenewCmd)
	setRenewCommandUUID(&case3RenewCmd)
	require.NoError(t, apple_mdm.HandleHostMDMProfileInstallResult(ctx, s.ds, host.UUID, case3RenewCmd, &failed, "boom", s.fleetSvc.NewActivity))
	require.Equal(t, 1, countRenewalActivitiesForCmd(case3RenewCmd))

	hostLite, err := s.ds.HostLiteByIdentifier(ctx, host.UUID)
	require.NoError(t, err)
	expected := fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "command_uuid": %q}`,
		hostLite.ID, hostLite.DisplayName(), case3RenewCmd)
	s.lastActivityOfTypeMatches(activityName, expected, 0)

	// Case 4: a successful (verifying) install profile result whose command
	// UUID matches the renew_command_uuid — no activity, because only failures
	// take the renewal path.
	case4RenewCmd := uuid.NewString()
	insertNanoCommand(case4RenewCmd)
	setRenewCommandUUID(&case4RenewCmd)
	require.NoError(t, apple_mdm.HandleHostMDMProfileInstallResult(ctx, s.ds, host.UUID, case4RenewCmd, &verifying, "", s.fleetSvc.NewActivity))
	require.Zero(t, countRenewalActivitiesForCmd(case4RenewCmd))
}

// TestAppleEnrollmentProfilesCarryBoundSCEPChallenges checks that each lane enrolls with the dynamic challenge
// from its profile. The static challenge is rotated first, so neither the original nor the new static value
// could have been used at SCEP.
func (s *integrationMDMTestSuite) TestAppleEnrollmentProfilesCarryBoundSCEPChallenges() {
	t := s.T()
	ctx := t.Context()

	rotated := "rotated-" + uuid.NewString()
	require.NoError(t, s.ds.ReplaceMDMConfigAssets(ctx, []fleet.MDMConfigAsset{{Name: fleet.MDMAssetSCEPChallenge, Value: []byte(rotated)}}, nil))
	t.Cleanup(func() {
		require.NoError(t, s.ds.ReplaceMDMConfigAssets(context.Background(),
			[]fleet.MDMConfigAsset{{Name: fleet.MDMAssetSCEPChallenge, Value: []byte(s.scepChallenge)}}, nil))
	})

	// requireBound checks the challenge was minted for want, was used for exactly one identity certificate,
	// and that certificate carries the matching binding.
	requireBound := func(t *testing.T, challenge string, want fleet.AppleSCEPChallengeInfo) {
		require.NotEmpty(t, challenge)
		require.NotEqual(t, s.scepChallenge, challenge)
		require.NotEqual(t, rotated, challenge)

		var row struct {
			fleet.AppleSCEPChallengeInfo
			Consumed   bool  `db:"consumed"`
			CertSerial int64 `db:"issued_cert_serial"`
		}
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &row, `SELECT purpose, host_uuid, hardware_serial, idp_account_uuid,
				consumed_at IS NOT NULL AS consumed, COALESCE(issued_cert_serial, 0) AS issued_cert_serial
				FROM mdm_apple_scep_challenges WHERE challenge = ?`, challenge)
		})
		require.Equal(t, want, row.AppleSCEPChallengeInfo)
		require.True(t, row.Consumed)
		require.NotZero(t, row.CertSerial)

		var certPEM string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &certPEM, `SELECT certificate_pem FROM identity_certificates WHERE serial = ?`, row.CertSerial)
		})
		block, _ := pem.Decode([]byte(certPEM))
		require.NotNil(t, block)
		cert, err := x509.ParseCertificate(block.Bytes)
		require.NoError(t, err)
		binding, err := apple_mdm.ParseAppleMDMCertificateBindingExtension(cert)
		require.NoError(t, err)
		require.NotNil(t, binding)
		require.Equal(t, want.Purpose, binding.Purpose)
	}

	// ade
	rawProfile := json.RawMessage(`{}`)
	_, err := s.ds.NewMDMAppleEnrollmentProfile(ctx, fleet.MDMAppleEnrollmentProfilePayload{
		Type: "automatic", DEPProfile: &rawProfile, Token: uuid.NewString(),
	})
	require.NoError(t, err)
	serial := mdmtest.RandSerialNumber()
	_, adeDevice := s.createAppleMobileHostThenDEPEnrollMDM("ios", serial)
	requireBound(t, adeDevice.EnrollInfo.SCEPChallenge, fleet.AppleSCEPChallengeInfo{
		Purpose: fleet.AppleMDMCertPurposeADE, UUID: new(adeDevice.UUID), Serial: new(serial),
	})
	// a challenge gives exactly one certificate
	require.Error(t, adeDevice.SCEPEnroll())

	// ota
	secret := "bound-challenge-secret"
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{Secrets: []*fleet.EnrollSecret{{Secret: secret}}},
	}, http.StatusOK, &applyResp)

	otaDevice := mdmtest.NewTestMDMClientAppleOTA(s.server.URL, secret, "MacBookPro16,1")
	require.NoError(t, otaDevice.Enroll())
	identity := fleet.AppleSCEPChallengeInfo{UUID: new(otaDevice.UUID), Serial: new(otaDevice.SerialNumber)}

	phaseTwo := identity
	phaseTwo.Purpose = fleet.AppleMDMCertPurposeOTAPhaseTwo
	requireBound(t, otaDevice.EnrollInfo.SCEPChallenge, phaseTwo)

	var phaseOneChallenge string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &phaseOneChallenge, `SELECT challenge FROM mdm_apple_scep_challenges
			WHERE purpose = ? AND host_uuid = ? AND hardware_serial = ?`,
			fleet.AppleMDMCertPurposeOTAPhaseOne, otaDevice.UUID, otaDevice.SerialNumber)
	})
	phaseOne := identity
	phaseOne.Purpose = fleet.AppleMDMCertPurposeOTAPhaseOne
	requireBound(t, phaseOneChallenge, phaseOne)
	require.NotEqual(t, phaseOneChallenge, otaDevice.EnrollInfo.SCEPChallenge)

	// adue
	account := &fleet.MDMIdPAccount{UUID: uuid.NewString(), Username: "bound-adue", Email: "bound-adue@example.com"}
	require.NoError(t, s.ds.InsertMDMIdPAccount(ctx, account))
	token, err := s.ds.InsertADUEEnrollmentChallenge(ctx, nil, account.UUID, time.Hour)
	require.NoError(t, err)

	adueDevice := mdmtest.NewTestMDMClientAppleAccountDrivenUserEnrollment(s.server.URL, "iPhone14,5", token)
	require.NoError(t, adueDevice.Enroll())
	requireBound(t, adueDevice.EnrollInfo.SCEPChallenge, fleet.AppleSCEPChallengeInfo{
		Purpose: fleet.AppleMDMCertPurposeADUE, IDPAccountUUID: new(account.UUID),
	})
}

func (s *integrationMDMTestSuite) TestOTAProfile() {
	t := s.T()
	ctx := context.Background()

	// Getting profile for non-existent secret it's ok
	s.Do("GET", "/api/latest/fleet/enrollment_profiles/ota", getOTAProfileRequest{}, http.StatusOK, "enroll_secret", "not-real")

	// Create an enroll secret; has some special characters that should be escaped in the profile
	globalEnrollSec := "global_enroll+_/sec"
	escSec := url.QueryEscape(globalEnrollSec)
	s.Do("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: globalEnrollSec}},
		},
	}, http.StatusOK)

	cfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)

	t.Run("gets profile with idp uuid included if boyd cookie is set", func(t *testing.T) {
		// Get profile with that enroll secret
		j, err := json.Marshal(getOTAProfileRequest{})
		require.NoError(t, err)

		idpUUID := uuid.New()
		sessionID := s.mustBYODIdPSession(t, idpUUID.String())
		resp := s.DoRawWithHeaders("GET", "/api/latest/fleet/enrollment_profiles/ota", j, http.StatusOK, map[string]string{
			"Cookie": fmt.Sprintf("%s=%s", shared_mdm.BYODIdpCookieName, sessionID),
		}, "enroll_secret", globalEnrollSec)

		require.NotZero(t, resp.ContentLength)
		require.Contains(t, resp.Header.Get("Content-Disposition"), `attachment;filename="fleet-mdm-enrollment-profile.mobileconfig"`)
		require.Contains(t, resp.Header.Get("Content-Type"), "application/x-apple-aspen-config")
		require.Contains(t, resp.Header.Get("X-Content-Type-Options"), "nosniff")
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, resp.ContentLength, int64(len(b)))
		require.Contains(t, string(b), "com.fleetdm.fleet.mdm.apple.ota")
		require.Contains(t, string(b), fmt.Sprintf("%s/api/v1/fleet/ota_enrollment?enroll_secret=%s&amp;idp_session=%s", cfg.ServerSettings.ServerURL, escSec, url.QueryEscape(sessionID)))
		require.NotContains(t, string(b), idpUUID.String())

		// the raw account reference is not a session, so the profile carries no account
		resp = s.DoRawWithHeaders("GET", "/api/latest/fleet/enrollment_profiles/ota", j, http.StatusOK, map[string]string{
			"Cookie": fmt.Sprintf("%s=%s", shared_mdm.BYODIdpCookieName, idpUUID.String()),
		}, "enroll_secret", globalEnrollSec)
		defer resp.Body.Close()
		b, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NotContains(t, string(b), idpUUID.String())
		require.Contains(t, string(b), cfg.OrgInfo.OrgName)
	})

	t.Run("does not include idp_uuid in the url if cookie is not set", func(t *testing.T) {
		resp := s.Do("GET", "/api/latest/fleet/enrollment_profiles/ota", &getOTAProfileRequest{}, http.StatusOK, "enroll_secret", globalEnrollSec)
		require.NotZero(t, resp.ContentLength)
		require.Contains(t, resp.Header.Get("Content-Disposition"), `attachment;filename="fleet-mdm-enrollment-profile.mobileconfig"`)
		require.Contains(t, resp.Header.Get("Content-Type"), "application/x-apple-aspen-config")
		require.Contains(t, resp.Header.Get("X-Content-Type-Options"), "nosniff")
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, resp.ContentLength, int64(len(b)))
		require.Contains(t, string(b), "com.fleetdm.fleet.mdm.apple.ota")
		require.Contains(t, string(b), fmt.Sprintf("%s/api/v1/fleet/ota_enrollment?enroll_secret=%s", cfg.ServerSettings.ServerURL, escSec))
		require.NotContains(t, string(b), "idp_session=")
		require.Contains(t, string(b), cfg.OrgInfo.OrgName)
	})

	t.Run("returns 403 if no idp_uuid is set when required by config", func(t *testing.T) {
		// update config to require idp_uuid
		cfg.MDM.MacOSSetup.EnableEndUserAuthentication = true
		err := s.ds.SaveAppConfig(ctx, cfg)
		require.NoError(t, err)
		t.Cleanup(func() {
			cfg.MDM.MacOSSetup.EnableEndUserAuthentication = false
			err := s.ds.SaveAppConfig(ctx, cfg)
			require.NoError(t, err)
		})
		s.Do("GET", "/api/latest/fleet/enrollment_profiles/ota", &getOTAProfileRequest{}, http.StatusForbidden, "enroll_secret", globalEnrollSec)
	})
}
