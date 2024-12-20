package service

import (
	"bytes"
	"context"
	"crypto/md5" // nolint:gosec // used only for tests
	"crypto/x509"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	servermdm "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) signedProfilesMatch(want, got [][]byte) {
	t := s.T()
	rootCA := x509.NewCertPool()

	assets, err := s.ds.GetAllMDMConfigAssetsByName(context.Background(), []fleet.MDMAssetName{
		fleet.MDMAssetCACert,
	}, nil)
	require.NoError(t, err)

	require.True(t, rootCA.AppendCertsFromPEM(assets[fleet.MDMAssetCACert].Value))

	// verify that all the profiles were signed usign the SCEP certificate,
	// and grab their contents
	signedContents := [][]byte{}
	for _, prof := range got {
		p7, err := pkcs7.Parse(prof)
		require.NoError(t, err)
		require.NoError(t, p7.VerifyWithChain(rootCA))
		signedContents = append(signedContents, p7.Content)
	}

	// verify that contents match
	require.ElementsMatch(t, want, signedContents)
}

func (s *integrationMDMTestSuite) TestAppleProfileManagement() {
	t := s.T()
	ctx := context.Background()

	err := s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}})
	require.NoError(t, err)

	globalProfiles := [][]byte{
		mobileconfigForTest("N1", "I1"),
		mobileconfigForTest("N2", "I2"),
	}
	wantGlobalProfiles := globalProfiles
	wantGlobalProfiles = append(
		wantGlobalProfiles,
		setupExpectedFleetdProfile(t, s.server.URL, t.Name(), nil),
	)

	// add global profiles
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)

	// invalid secrets
	var invalidSecretsProfile = []byte(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array/>
	<key>PayloadDisplayName</key>
	<string>$FLEET_SECRET_INVALID</string>
	<key>PayloadIdentifier</key>
	<string>N3</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>601E0B42-0989-4FAD-A61B-18656BA3670E</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
`)

	res := s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{invalidSecretsProfile}}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "$FLEET_SECRET_INVALID")

	// create a new team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "batch_set_mdm_profiles"})
	require.NoError(t, err)

	// add an enroll secret so the fleetd profiles differ
	var teamResp teamEnrollSecretsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", tm.ID),
		modifyTeamEnrollSecretsRequest{
			Secrets: []fleet.EnrollSecret{{Secret: "team1_enroll_sec"}},
		}, http.StatusOK, &teamResp)

	teamProfiles := [][]byte{
		mobileconfigForTest("N3", "I3"),
	}
	wantTeamProfiles := teamProfiles
	wantTeamProfiles = append(
		wantTeamProfiles,
		setupExpectedFleetdProfile(t, s.server.URL, "team1_enroll_sec", &tm.ID),
	)
	// add profiles to the team
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: teamProfiles}, http.StatusNoContent,
		"team_id", fmt.Sprint(tm.ID))

	// create a non-macOS host
	_, err = s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            1,
		OsqueryHostID: ptr.String("non-macos-host"),
		NodeKey:       ptr.String("non-macos-host"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.non.macos", t.Name()),
		Platform:      "windows",
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

	// Create a host and then enroll to MDM.
	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setupPusher(s, t, mdmDevice)

	// trigger a profile sync
	s.awaitTriggerProfileSchedule(t)
	installs, removes := checkNextPayloads(t, mdmDevice, false)
	// verify that we received all profiles
	s.signedProfilesMatch(
		append(wantGlobalProfiles, setupExpectedCAProfile(t, s.ds)),
		installs,
	)
	require.Empty(t, removes)

	expectedNoTeamSummary := fleet.MDMProfilesSummary{
		Pending:   0,
		Failed:    0,
		Verifying: 1,
		Verified:  0,
	}
	expectedTeamSummary := fleet.MDMProfilesSummary{}
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamSummary, &expectedNoTeamSummary)
	s.checkMDMProfilesSummaries(t, &tm.ID, expectedTeamSummary, &expectedTeamSummary) // empty because no hosts in team

	// add the host to a team
	err = s.ds.AddHostsToTeam(ctx, &tm.ID, []uint{host.ID})
	require.NoError(t, err)

	// trigger a profile sync
	s.awaitTriggerProfileSchedule(t)
	installs, removes = checkNextPayloads(t, mdmDevice, false)
	// verify that we should install the team profile
	s.signedProfilesMatch(wantTeamProfiles, installs)
	// verify that we should delete both profiles
	require.ElementsMatch(t, []string{"I1", "I2"}, removes)

	expectedNoTeamSummary = fleet.MDMProfilesSummary{}
	expectedTeamSummary = fleet.MDMProfilesSummary{
		Pending:   0,
		Failed:    0,
		Verifying: 1,
		Verified:  0,
	}
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamSummary, &expectedNoTeamSummary) // empty because host was transferred
	s.checkMDMProfilesSummaries(t, &tm.ID, expectedTeamSummary, &expectedTeamSummary)  // host now verifying team profiles

	// Use secret variables in a profile
	secretIdentifier := "secret-identifier-1"
	secretType := "secret.type.1"
	secretName := "secretName"
	secretProfile := string(mobileconfigForTest("NS1", "IS1"))
	req := secretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  "FLEET_SECRET_IDENTIFIER",
				Value: secretIdentifier,
			},
			{
				Name:  "FLEET_SECRET_TYPE",
				Value: secretType,
			},
			{
				Name:  "FLEET_SECRET_NAME",
				Value: secretName,
			},
			{
				Name:  "FLEET_SECRET_PROFILE",
				Value: secretProfile,
			},
		},
	}
	secretResp := secretVariablesResponse{}
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &secretResp)

	// set new team profiles (delete + addition)
	teamProfiles = [][]byte{
		mobileconfigForTest("N4", "I4"),
		mobileconfigForTestWithContent("N5", "I5", "$FLEET_SECRET_IDENTIFIER", "${FLEET_SECRET_TYPE}",
			"$FLEET_SECRET_NAME"),
		// The whole profile is one big secret.
		[]byte("$FLEET_SECRET_PROFILE"),
	}
	// We deep copy one of the team profiles because we will modify the slice in place, and we want to keep the originals for later.
	wantTeamProfiles = [][]byte{
		teamProfiles[0],
		make([]byte, len(teamProfiles[1])),
		{},
	}
	copy(wantTeamProfiles[1], teamProfiles[1])
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: teamProfiles}, http.StatusNoContent,
		"team_id", fmt.Sprint(tm.ID))

	// trigger a profile sync
	s.awaitTriggerProfileSchedule(t)
	installs, removes = checkNextPayloads(t, mdmDevice, false)
	// Manually replace the expected secret variables in the profile
	wantTeamProfiles[1] = []byte(strings.ReplaceAll(string(wantTeamProfiles[1]), "$FLEET_SECRET_IDENTIFIER", secretIdentifier))
	wantTeamProfiles[1] = []byte(strings.ReplaceAll(string(wantTeamProfiles[1]), "${FLEET_SECRET_TYPE}", secretType))
	wantTeamProfiles[1] = []byte(strings.ReplaceAll(string(wantTeamProfiles[1]), "$FLEET_SECRET_NAME", secretName))
	wantTeamProfiles[2] = []byte(secretProfile)
	// verify that we should install the team profiles
	s.signedProfilesMatch(wantTeamProfiles, installs)
	// verify that we should delete the old team profiles
	require.ElementsMatch(t, []string{"I3"}, removes)

	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamSummary, &expectedNoTeamSummary) // empty because host was transferred
	s.checkMDMProfilesSummaries(t, &tm.ID, expectedTeamSummary, &expectedTeamSummary)  // host still verifying team profiles

	// with no changes
	s.awaitTriggerProfileSchedule(t)
	installs, removes = checkNextPayloads(t, mdmDevice, false)
	require.Empty(t, installs)
	require.Empty(t, removes)

	// Clear the profiles using the new (non-deprecated) endpoint.
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: nil}, http.StatusNoContent, "team_id",
		fmt.Sprint(tm.ID), "dry_run", "true")
	s.assertConfigProfilesByIdentifier(&tm.ID, "IS1", true)
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: nil}, http.StatusNoContent, "team_id",
		fmt.Sprint(tm.ID), "dry_run", "false")
	s.assertConfigProfilesByIdentifier(&tm.ID, "IS1", false)
	s.awaitTriggerProfileSchedule(t)
	installs, removes = checkNextPayloads(t, mdmDevice, false)
	require.Empty(t, installs)
	assert.Len(t, removes, 3)

	// And reapply the same profiles using the new (non-deprecated) endpoint.
	batchRequest := batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N4", Contents: teamProfiles[0]},
		{Name: "N5", Contents: teamProfiles[1]},
		{Name: "NS1", Contents: teamProfiles[2]},
	}}
	t.Logf("VICTOR: %s", string(teamProfiles[2]))
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchRequest, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID), "dry_run", "true")
	s.assertConfigProfilesByIdentifier(&tm.ID, "I4", false)
	s.assertConfigProfilesByIdentifier(&tm.ID, "I5", false)
	s.assertConfigProfilesByIdentifier(&tm.ID, "IS1", false)
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchRequest, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))
	s.assertConfigProfilesByIdentifier(&tm.ID, "IS1", true)
	s.awaitTriggerProfileSchedule(t)
	installs, removes = checkNextPayloads(t, mdmDevice, false)
	assert.Empty(t, removes)
	// verify that we should install the team profiles
	s.signedProfilesMatch(wantTeamProfiles, installs)

	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID), getHostRequest{}, http.StatusOK, &hostResp)
	require.NotEmpty(t, hostResp.Host.MDM.Profiles)
	resProfiles := *hostResp.Host.MDM.Profiles
	// two extra profiles: fleetd config and root CA
	require.Len(t, resProfiles, len(wantTeamProfiles)+2)

	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamSummary, &expectedNoTeamSummary) // empty because host was transferred
	s.checkMDMProfilesSummaries(t, &tm.ID, expectedTeamSummary, &expectedTeamSummary)  // host still verifying team profiles

	// add a new profile to the team
	mcUUID := "a" + uuid.NewString()
	prof := mcBytesForTest("name-"+mcUUID, "idenfifer-"+mcUUID, mcUUID)
	wantTeamProfiles = append(wantTeamProfiles, prof)
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `INSERT INTO mdm_apple_configuration_profiles (profile_uuid, team_id, name, identifier, mobileconfig, checksum, uploaded_at) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);`
		_, err := q.ExecContext(context.Background(), stmt, mcUUID, tm.ID, "name-"+mcUUID, "identifier-"+mcUUID, prof, []byte("checksum-"+mcUUID))
		return err
	})
	s.awaitTriggerProfileSchedule(t)
	installs, removes = checkNextPayloads(t, mdmDevice, false)
	require.Len(t, installs, 1)
	s.signedProfilesMatch([][]byte{prof}, installs)
	require.Empty(t, removes)
	s.checkMDMProfilesSummaries(t, &tm.ID, fleet.MDMProfilesSummary{Verifying: 1}, nil)

	// can't resend profile while verifying
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, mcUUID), nil, http.StatusConflict)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn’t resend. Configuration profiles with “pending” or “verifying” status can’t be resent.")

	// set the profile to pending, can't resend
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_apple_profiles SET status = ? WHERE profile_uuid = ? AND host_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, fleet.MDMDeliveryPending, mcUUID, host.UUID)
		return err
	})
	s.checkMDMProfilesSummaries(t, &tm.ID, fleet.MDMProfilesSummary{Pending: 1}, nil)
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, mcUUID), nil, http.StatusConflict)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn’t resend. Configuration profiles with “pending” or “verifying” status can’t be resent.")

	// set the profile to failed, can resend
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_apple_profiles SET status = ? WHERE profile_uuid = ? AND host_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, fleet.MDMDeliveryFailed, mcUUID, host.UUID)
		return err
	})
	s.checkMDMProfilesSummaries(t, &tm.ID, fleet.MDMProfilesSummary{Failed: 1}, nil)
	_ = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, mcUUID), nil, http.StatusAccepted)
	s.awaitTriggerProfileSchedule(t)
	installs, removes = checkNextPayloads(t, mdmDevice, false)
	require.Len(t, installs, 1)
	s.signedProfilesMatch([][]byte{prof}, installs)
	require.Empty(t, removes)
	s.checkMDMProfilesSummaries(t, &tm.ID, fleet.MDMProfilesSummary{Verifying: 1}, nil)

	// can't resend profile while verifying
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, mcUUID), nil, http.StatusConflict)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn’t resend. Configuration profiles with “pending” or “verifying” status can’t be resent.")

	// set the profile to verified, can resend
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_apple_profiles SET status = ? WHERE profile_uuid = ? AND host_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, fleet.MDMDeliveryVerified, mcUUID, host.UUID)
		return err
	})
	_ = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, mcUUID), nil, http.StatusAccepted)
	s.awaitTriggerProfileSchedule(t)
	installs, removes = checkNextPayloads(t, mdmDevice, false)
	require.Len(t, installs, 1)
	s.signedProfilesMatch([][]byte{prof}, installs)
	require.Empty(t, removes)
	s.checkMDMProfilesSummaries(t, &tm.ID, fleet.MDMProfilesSummary{Verifying: 1}, nil)
	s.lastActivityMatches(
		fleet.ActivityTypeResentConfigurationProfile{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "profile_name": %q}`, host.ID, host.DisplayName(), "name-"+mcUUID),
		0)

	// add a declaration to the team
	declIdent := "decl-ident-" + uuid.NewString()
	fields := map[string][]string{
		"team_id": {fmt.Sprintf("%d", tm.ID)},
	}
	body, headers := generateNewProfileMultipartRequest(
		t, "some-declaration.json", declarationForTest(declIdent), s.token, fields,
	)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusOK, headers)
	var resp newMDMConfigProfileResponse
	err = json.NewDecoder(res.Body).Decode(&resp)
	require.NoError(t, err)
	require.NotEmpty(t, resp.ProfileUUID)
	require.Equal(t, "d", string(resp.ProfileUUID[0]))
	declUUID := resp.ProfileUUID

	checkDDMSync := func(d *mdmtest.TestAppleMDMClient) {
		require.NoError(t, ReconcileAppleDeclarations(ctx, s.ds, s.mdmCommander, s.logger))
		cmd, err := d.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd)
		require.Equal(t, "DeclarativeManagement", cmd.Command.RequestType)
		cmd, err = d.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
		require.Nil(t, cmd, fmt.Sprintf("expected no more commands, but got: %+v", cmd))
		_, err = d.DeclarativeManagement("tokens")
		require.NoError(t, err)
	}
	checkDDMSync(mdmDevice)
	s.checkMDMProfilesSummaries(t, &tm.ID, fleet.MDMProfilesSummary{Verifying: 1}, nil)

	// can't resend declaration while verifying
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, declUUID), nil, http.StatusConflict)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn’t resend. Configuration profiles with “pending” or “verifying” status can’t be resent.")

	// set the declaration to verified, can resend
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_apple_declarations SET status = ? WHERE declaration_uuid = ? AND host_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, fleet.MDMDeliveryVerified, declUUID, host.UUID)
		return err
	})
	_ = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, declUUID), nil, http.StatusAccepted)
	checkDDMSync(mdmDevice)
	s.checkMDMProfilesSummaries(t, &tm.ID, fleet.MDMProfilesSummary{Verifying: 1}, nil)
	s.lastActivityMatches(
		fleet.ActivityTypeResentConfigurationProfile{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "profile_name": "some-declaration"}`, host.ID, host.DisplayName()),
		0)

	// transfer the host to the global team
	err = s.ds.AddHostsToTeam(ctx, nil, []uint{host.ID})
	require.NoError(t, err)

	s.awaitTriggerProfileSchedule(t)
	installs, removes = checkNextPayloads(t, mdmDevice, false)
	require.Len(t, installs, len(wantGlobalProfiles))
	s.signedProfilesMatch(wantGlobalProfiles, installs)
	require.Len(t, removes, len(wantTeamProfiles))
	expectedNoTeamSummary = fleet.MDMProfilesSummary{Verifying: 1}
	expectedTeamSummary = fleet.MDMProfilesSummary{}
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamSummary, &expectedNoTeamSummary) // host now verifying global profiles
	s.checkMDMProfilesSummaries(t, &tm.ID, expectedTeamSummary, &expectedTeamSummary)

	// can't resend profile from another team
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, mcUUID), nil, http.StatusNotFound)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Unable to match profile to host")

	// add a Windows profile, resend not supported when host is macOS
	wpUUID := mysql.InsertWindowsProfileForTest(t, s.ds, 0)
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, wpUUID), nil, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Profile is not compatible with host platform")

	// invalid profile UUID prefix should return 404
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, "z"+uuid.NewString()), nil, http.StatusNotFound)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Invalid profile UUID prefix")

	// set OS updates settings for no-team and team, should not change the
	// summaries as this profile is ignored.
	s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"macos_updates": {
				"deadline": "2023-12-31",
				"minimum_version": "13.3.7"
			}
		}
	}`), http.StatusOK)
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			MacOSUpdates: &fleet.AppleOSUpdateSettings{
				Deadline:       optjson.SetString("1992-01-01"),
				MinimumVersion: optjson.SetString("13.1.1"),
			},
		},
	}, http.StatusOK)
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamSummary, &expectedNoTeamSummary)
	s.checkMDMProfilesSummaries(t, &tm.ID, expectedTeamSummary, &expectedTeamSummary)

	// it should also not show up in the host's profiles list
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID), getHostRequest{}, http.StatusOK, &hostResp)
	require.NotEmpty(t, hostResp.Host.MDM.Profiles)
	resProfiles = *hostResp.Host.MDM.Profiles
	// two extra profiles: fleetd config and root CA
	require.Len(t, resProfiles, len(wantGlobalProfiles)+2)
}

func (s *integrationMDMTestSuite) TestAppleProfileRetries() {
	t := s.T()
	ctx := context.Background()

	enrollSecret := "test-profile-retries-secret"
	err := s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: enrollSecret}})
	require.NoError(t, err)

	testProfiles := [][]byte{
		mobileconfigForTest("N1", "I1"),
		mobileconfigForTest("N2", "I2"),
	}
	initialExpectedProfiles := testProfiles
	initialExpectedProfiles = append(
		initialExpectedProfiles,
		setupExpectedFleetdProfile(t, s.server.URL, enrollSecret, nil),
		setupExpectedCAProfile(t, s.ds),
	)

	h, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setupPusher(s, t, mdmDevice)

	expectedProfileStatuses := map[string]fleet.MDMDeliveryStatus{
		"I1": fleet.MDMDeliveryVerifying,
		"I2": fleet.MDMDeliveryVerifying,
		mobileconfig.FleetdConfigPayloadIdentifier:      fleet.MDMDeliveryVerifying,
		mobileconfig.FleetCARootConfigPayloadIdentifier: fleet.MDMDeliveryVerifying,
	}
	checkProfilesStatus := func(t *testing.T) {
		storedProfs, err := s.ds.GetHostMDMAppleProfiles(ctx, h.UUID)
		require.NoError(t, err)
		require.Len(t, storedProfs, len(expectedProfileStatuses))
		for _, p := range storedProfs {
			want, ok := expectedProfileStatuses[p.Identifier]
			require.True(t, ok, "unexpected profile: %s", p.Identifier)
			require.Equal(t, want, *p.Status, "expected status %s but got %s for profile: %s", want, *p.Status, p.Identifier)
		}
	}

	expectedRetryCounts := map[string]uint{
		"I1": 0,
		"I2": 0,
		mobileconfig.FleetdConfigPayloadIdentifier:      0,
		mobileconfig.FleetCARootConfigPayloadIdentifier: 0,
	}
	checkRetryCounts := func(t *testing.T) {
		counts, err := s.ds.GetHostMDMProfilesRetryCounts(ctx, h)
		require.NoError(t, err)
		require.Len(t, counts, len(expectedRetryCounts))
		for _, c := range counts {
			want, ok := expectedRetryCounts[c.ProfileIdentifier]
			require.True(t, ok, "unexpected profile: %s", c.ProfileIdentifier)
			require.Equal(t, want, c.Retries, "expected retry count %d but got %d for profile: %s", want, c.Retries, c.ProfileIdentifier)
		}
	}

	hostProfsByIdent := map[string]*fleet.HostMacOSProfile{
		"I1": {
			Identifier:  "I1",
			DisplayName: "N1",
			InstallDate: time.Now().Add(15 * time.Minute),
		},
		"I2": {
			Identifier:  "I2",
			DisplayName: "N2",
			InstallDate: time.Now().Add(15 * time.Minute),
		},
		mobileconfig.FleetdConfigPayloadIdentifier: {
			Identifier:  mobileconfig.FleetdConfigPayloadIdentifier,
			DisplayName: "Fleetd configuration",
			InstallDate: time.Now().Add(15 * time.Minute),
		},
	}
	reportHostProfs := func(t *testing.T, identifiers ...string) {
		report := make(map[string]*fleet.HostMacOSProfile, len(hostProfsByIdent))
		for _, ident := range identifiers {
			report[ident] = hostProfsByIdent[ident]
		}
		require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, s.ds, h, report))
	}

	setProfileUploadedAt := func(t *testing.T, uploadedAt time.Time, identifiers ...interface{}) {
		bindVars := strings.TrimSuffix(strings.Repeat("?, ", len(identifiers)), ", ")
		stmt := fmt.Sprintf("UPDATE mdm_apple_configuration_profiles SET uploaded_at = ? WHERE identifier IN(%s)", bindVars)
		args := append([]interface{}{uploadedAt}, identifiers...)
		mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, stmt, args...)
			return err
		})
	}

	t.Run("retry after verifying", func(t *testing.T) {
		// upload test profiles then simulate expired grace period by setting updated_at timestamp of profiles back by 48 hours
		s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		setProfileUploadedAt(t, time.Now().Add(-48*time.Hour), "I1", "I2", mobileconfig.FleetdConfigPayloadIdentifier)

		// trigger initial profile sync and confirm that we received all profiles
		s.awaitTriggerProfileSchedule(t)
		installs, removes := checkNextPayloads(t, mdmDevice, false)
		s.signedProfilesMatch(initialExpectedProfiles, installs)
		require.Empty(t, removes)

		checkProfilesStatus(t) // all profiles verifying
		checkRetryCounts(t)    // no retries yet

		// report osquery results with I2 missing and confirm I2 marked as pending and other profiles are marked as verified
		reportHostProfs(t, "I1", mobileconfig.FleetdConfigPayloadIdentifier)
		expectedProfileStatuses["I2"] = fleet.MDMDeliveryPending
		expectedProfileStatuses["I1"] = fleet.MDMDeliveryVerified
		expectedProfileStatuses[mobileconfig.FleetdConfigPayloadIdentifier] = fleet.MDMDeliveryVerified
		checkProfilesStatus(t)
		expectedRetryCounts["I2"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile command for I2 was resent
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		s.signedProfilesMatch([][]byte{initialExpectedProfiles[1]}, installs)
		require.Empty(t, removes)

		// report osquery results with I2 present and confirm that all profiles are verified
		reportHostProfs(t, "I1", "I2", mobileconfig.FleetdConfigPayloadIdentifier)
		expectedProfileStatuses["I2"] = fleet.MDMDeliveryVerified
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that no profiles were sent
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		require.Empty(t, installs)
		require.Empty(t, removes)
	})

	t.Run("retry after verification", func(t *testing.T) {
		// report osquery results with I1 missing and confirm that the I1 marked as pending (initial retry)
		reportHostProfs(t, "I2", mobileconfig.FleetdConfigPayloadIdentifier)
		expectedProfileStatuses["I1"] = fleet.MDMDeliveryPending
		checkProfilesStatus(t)
		expectedRetryCounts["I1"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile command for I1 was resent
		s.awaitTriggerProfileSchedule(t)
		installs, removes := checkNextPayloads(t, mdmDevice, false)
		s.signedProfilesMatch([][]byte{initialExpectedProfiles[0]}, installs)
		require.Empty(t, removes)

		// report osquery results with I1 missing again and confirm that the I1 marked as failed (max retries exceeded)
		reportHostProfs(t, "I2", mobileconfig.FleetdConfigPayloadIdentifier)
		expectedProfileStatuses["I1"] = fleet.MDMDeliveryFailed
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for I1 was not resent
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		require.Empty(t, installs)
		require.Empty(t, removes)
	})

	t.Run("retry after device error", func(t *testing.T) {
		// add another profile and set the updated_at timestamp back by 48 hours
		newProfile := mobileconfigForTest("N3", "I3")
		testProfiles = append(testProfiles, newProfile)
		s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		setProfileUploadedAt(t, time.Now().Add(-48*time.Hour), "I1", "I2", mobileconfig.FleetdConfigPayloadIdentifier, "I3")

		// trigger a profile sync and confirm that the install profile command for I3 was sent and
		// simulate a device error
		s.awaitTriggerProfileSchedule(t)
		installs, removes := checkNextPayloads(t, mdmDevice, true)
		s.signedProfilesMatch([][]byte{newProfile}, installs)
		require.Empty(t, removes)
		expectedProfileStatuses["I3"] = fleet.MDMDeliveryPending
		checkProfilesStatus(t)
		expectedRetryCounts["I3"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile command for I3 was sent and
		// simulate a device ack
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		s.signedProfilesMatch([][]byte{newProfile}, installs)
		require.Empty(t, removes)
		expectedProfileStatuses["I3"] = fleet.MDMDeliveryVerifying
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// report osquery results with I3 missing and confirm that the I3 marked as failed (max
		// retries exceeded)
		reportHostProfs(t, "I2", mobileconfig.FleetdConfigPayloadIdentifier)
		expectedProfileStatuses["I3"] = fleet.MDMDeliveryFailed
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for I3 was not resent
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		require.Empty(t, installs)
		require.Empty(t, removes)
	})

	t.Run("repeated device error", func(t *testing.T) {
		// add another profile and set the updated_at timestamp back by 48 hours
		newProfile := mobileconfigForTest("N4", "I4")
		testProfiles = append(testProfiles, newProfile)
		s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		setProfileUploadedAt(t, time.Now().Add(-48*time.Hour), "I1", "I2", mobileconfig.FleetdConfigPayloadIdentifier, "I3", "I4")

		// trigger a profile sync and confirm that the install profile command for I3 was sent and
		// simulate a device error
		s.awaitTriggerProfileSchedule(t)
		installs, removes := checkNextPayloads(t, mdmDevice, true)
		s.signedProfilesMatch([][]byte{newProfile}, installs)
		require.Empty(t, removes)
		expectedProfileStatuses["I4"] = fleet.MDMDeliveryPending
		checkProfilesStatus(t)
		expectedRetryCounts["I4"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile command for I4 was sent and
		// simulate a second device error
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, true)
		s.signedProfilesMatch([][]byte{newProfile}, installs)
		require.Empty(t, removes)
		expectedProfileStatuses["I4"] = fleet.MDMDeliveryFailed
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for I3 was not resent
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		require.Empty(t, installs)
		require.Empty(t, removes)
	})

	t.Run("retry count does not reset", func(t *testing.T) {
		// add another profile and set the updated_at timestamp back by 48 hours
		newProfile := mobileconfigForTest("N5", "I5")
		testProfiles = append(testProfiles, newProfile)
		hostProfsByIdent["I5"] = &fleet.HostMacOSProfile{Identifier: "I5", DisplayName: "N5", InstallDate: time.Now()}
		s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		setProfileUploadedAt(t, time.Now().Add(-48*time.Hour), "I1", "I2", mobileconfig.FleetdConfigPayloadIdentifier, "I3", "I4", "I5")

		// trigger a profile sync and confirm that the install profile command for I3 was sent and
		// simulate a device error
		s.awaitTriggerProfileSchedule(t)
		installs, removes := checkNextPayloads(t, mdmDevice, true)
		s.signedProfilesMatch([][]byte{newProfile}, installs)
		require.Empty(t, removes)
		expectedProfileStatuses["I5"] = fleet.MDMDeliveryPending
		checkProfilesStatus(t)
		expectedRetryCounts["I5"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile command for I5 was sent and
		// simulate a device ack
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		s.signedProfilesMatch([][]byte{newProfile}, installs)
		require.Empty(t, removes)
		expectedProfileStatuses["I5"] = fleet.MDMDeliveryVerifying
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// report osquery results with I5 found and confirm that the I5 marked as verified
		reportHostProfs(t, "I2", mobileconfig.FleetdConfigPayloadIdentifier, "I5")
		expectedProfileStatuses["I5"] = fleet.MDMDeliveryVerified
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for I5 was not resent
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		require.Empty(t, installs)
		require.Empty(t, removes)

		// report osquery results again, this time I5 is missing and confirm that the I5 marked as
		// failed (max retries exceeded)
		reportHostProfs(t, "I2", mobileconfig.FleetdConfigPayloadIdentifier)
		expectedProfileStatuses["I5"] = fleet.MDMDeliveryFailed
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for I5 was not resent
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		require.Empty(t, installs)
		require.Empty(t, removes)
	})
}

func (s *integrationMDMTestSuite) TestWindowsProfileRetries() {
	t := s.T()
	ctx := context.Background()

	testProfiles := []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: syncml.ForTestWithData(map[string]string{"L1": "D1"})},
		{Name: "N2", Contents: syncml.ForTestWithData(map[string]string{"L2": "D2", "L3": "D3"})},
	}

	h, mdmDevice := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	expectedProfileStatuses := map[string]fleet.MDMDeliveryStatus{
		"N1": fleet.MDMDeliveryVerifying,
		"N2": fleet.MDMDeliveryVerifying,
	}
	checkProfilesStatus := func(t *testing.T) {
		storedProfs, err := s.ds.GetHostMDMWindowsProfiles(ctx, h.UUID)
		require.NoError(t, err)
		require.Len(t, storedProfs, len(expectedProfileStatuses))
		for _, p := range storedProfs {
			want, ok := expectedProfileStatuses[p.Name]
			require.True(t, ok, "unexpected profile: %s", p.Name)
			require.Equal(t, want, *p.Status, "expected status %s but got %s for profile: %s", want, *p.Status, p.Name)
		}
	}

	expectedRetryCounts := map[string]uint{
		"N1": 0,
		"N2": 0,
	}
	checkRetryCounts := func(t *testing.T) {
		counts, err := s.ds.GetHostMDMProfilesRetryCounts(ctx, h)
		require.NoError(t, err)
		require.Len(t, counts, len(expectedRetryCounts))
		for _, c := range counts {
			want, ok := expectedRetryCounts[c.ProfileName]
			require.True(t, ok, "unexpected profile: %s", c.ProfileName)
			require.Equal(t, want, c.Retries, "expected retry count %d but got %d for profile: %s", want, c.Retries, c.ProfileName)
		}
	}

	type profileData struct {
		Status string
		LocURI string
		Data   string
	}
	hostProfileReports := map[string][]profileData{
		"N1": {{"200", "L1", "D1"}},
		"N2": {{"200", "L2", "D2"}, {"200", "L3", "D3"}},
	}
	reportHostProfs := func(t *testing.T, profileNames ...string) {
		var responseOps []*fleet.SyncMLCmd
		for _, profileName := range profileNames {
			report, ok := hostProfileReports[profileName]
			require.True(t, ok)

			for _, p := range report {
				ref := microsoft_mdm.HashLocURI(profileName, p.LocURI)
				responseOps = append(responseOps, &fleet.SyncMLCmd{
					XMLName: xml.Name{Local: fleet.CmdStatus},
					CmdID:   fleet.CmdID{Value: uuid.NewString()},
					CmdRef:  &ref,
					Data:    ptr.String(p.Status),
				})

				// the protocol can respond with only a `Status`
				// command if the status failed
				if p.Status != "200" || p.Data != "" {
					responseOps = append(responseOps, &fleet.SyncMLCmd{
						XMLName: xml.Name{Local: fleet.CmdResults},
						CmdID:   fleet.CmdID{Value: uuid.NewString()},
						CmdRef:  &ref,
						Items: []fleet.CmdItem{
							{Target: ptr.String(p.LocURI), Data: &fleet.RawXmlData{Content: p.Data}},
						},
					})
				}
			}
		}

		msg, err := createSyncMLMessage("2", "2", "foo", "bar", responseOps)
		require.NoError(t, err)
		out, err := xml.Marshal(msg)
		require.NoError(t, err)
		require.NoError(t, microsoft_mdm.VerifyHostMDMProfiles(ctx, s.ds, h, out))
	}

	verifyCommands := func(wantProfileInstalls int, status string) {
		s.awaitTriggerProfileSchedule(t)
		cmds, err := mdmDevice.StartManagementSession()
		require.NoError(t, err)
		// profile installs + 2 protocol commands acks
		require.Len(t, cmds, wantProfileInstalls+2)
		msgID, err := mdmDevice.GetCurrentMsgID()
		require.NoError(t, err)
		atomicCmds := 0
		for _, c := range cmds {
			if c.Verb == "Atomic" {
				atomicCmds++
			}
			mdmDevice.AppendResponse(fleet.SyncMLCmd{
				XMLName: xml.Name{Local: fleet.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  ptr.String(c.Cmd.CmdID.Value),
				Cmd:     ptr.String(c.Verb),
				Data:    ptr.String(status),
				Items:   nil,
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			})
		}
		require.Equal(t, wantProfileInstalls, atomicCmds)
		cmds, err = mdmDevice.SendResponse()
		require.NoError(t, err)
		// the ack of the message should be the only returned command
		require.Len(t, cmds, 1)
	}

	t.Run("retry after verifying", func(t *testing.T) {
		// upload test profiles then simulate expired grace period by setting updated_at timestamp of profiles back by 48 hours
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		// profiles to install + 2 boilerplate <Status>
		verifyCommands(len(testProfiles), syncml.CmdStatusOK)
		checkProfilesStatus(t) // all profiles verifying
		checkRetryCounts(t)    // no retries yet

		// report osquery results with N2 missing and confirm N2 marked
		// as verifying and other profiles are marked as verified
		reportHostProfs(t, "N1")
		expectedProfileStatuses["N2"] = fleet.MDMDeliveryPending
		expectedProfileStatuses["N1"] = fleet.MDMDeliveryVerified
		checkProfilesStatus(t)
		expectedRetryCounts["N2"] = 1
		checkRetryCounts(t)

		// report osquery results with N2 present and confirm that all profiles are verified
		verifyCommands(1, syncml.CmdStatusOK)
		reportHostProfs(t, "N1", "N2")
		expectedProfileStatuses["N2"] = fleet.MDMDeliveryVerified
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that no profiles were sent
		verifyCommands(0, syncml.CmdStatusOK)
	})

	t.Run("retry after verification", func(t *testing.T) {
		// report osquery results with N1 missing and confirm that the N1 marked as pending (initial retry)
		reportHostProfs(t, "N2")
		expectedProfileStatuses["N1"] = fleet.MDMDeliveryPending
		checkProfilesStatus(t)
		expectedRetryCounts["N1"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile command for N1 was resent
		verifyCommands(1, syncml.CmdStatusOK)

		// report osquery results with N1 missing again and confirm that the N1 marked as failed (max retries exceeded)
		reportHostProfs(t, "N2")
		expectedProfileStatuses["N1"] = fleet.MDMDeliveryFailed
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for N1 was not resent
		verifyCommands(0, syncml.CmdStatusOK)
	})

	t.Run("retry after device error", func(t *testing.T) {
		// add another profile
		newProfile := syncml.ForTestWithData(map[string]string{"L3": "D3"})
		testProfiles = append(testProfiles, fleet.MDMProfileBatchPayload{
			Name:     "N3",
			Contents: newProfile,
		})
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		// trigger a profile sync and confirm that the install profile command for N3 was sent and
		// simulate a device error
		verifyCommands(1, syncml.CmdStatusAtomicFailed)
		expectedProfileStatuses["N3"] = fleet.MDMDeliveryPending
		checkProfilesStatus(t)
		expectedRetryCounts["N3"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile command for N3 was sent and
		// simulate a device ack
		verifyCommands(1, syncml.CmdStatusOK)
		expectedProfileStatuses["N3"] = fleet.MDMDeliveryVerifying
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// report osquery results with N3 missing and confirm that the N3 marked as failed (max
		// retries exceeded)
		reportHostProfs(t, "N2")
		expectedProfileStatuses["N3"] = fleet.MDMDeliveryFailed
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for N3 was not resent
		verifyCommands(0, syncml.CmdStatusOK)
	})

	t.Run("repeated device error", func(t *testing.T) {
		// add another profile
		testProfiles = append(testProfiles, fleet.MDMProfileBatchPayload{
			Name:     "N4",
			Contents: syncml.ForTestWithData(map[string]string{"L4": "D4"}),
		})
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		// trigger a profile sync and confirm that the install profile command for N4 was sent and
		// simulate a device error
		verifyCommands(1, syncml.CmdStatusAtomicFailed)
		expectedProfileStatuses["N4"] = fleet.MDMDeliveryPending
		checkProfilesStatus(t)
		expectedRetryCounts["N4"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile
		// command for N4 was sent and simulate a second device error
		verifyCommands(1, syncml.CmdStatusAtomicFailed)
		expectedProfileStatuses["N4"] = fleet.MDMDeliveryFailed
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile
		// command for N4 was not resent
		verifyCommands(0, syncml.CmdStatusOK)
	})

	t.Run("retry count does not reset", func(t *testing.T) {
		// add another profile
		testProfiles = append(testProfiles, fleet.MDMProfileBatchPayload{
			Name:     "N5",
			Contents: syncml.ForTestWithData(map[string]string{"L5": "D5"}),
		})
		// hostProfsByIdent["N5"] = &fleet.HostMacOSProfile{Identifier: "N5", DisplayName: "N5", InstallDate: time.Now()}
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		// trigger a profile sync and confirm that the install profile
		// command for N5 was sent and simulate a device error
		verifyCommands(1, syncml.CmdStatusAtomicFailed)
		expectedProfileStatuses["N5"] = fleet.MDMDeliveryPending
		checkProfilesStatus(t)
		expectedRetryCounts["N5"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile
		// command for N5 was sent and simulate a device ack
		verifyCommands(1, syncml.CmdStatusOK)
		expectedProfileStatuses["N5"] = fleet.MDMDeliveryVerifying
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// report osquery results with N5 found and confirm that the N5 marked as verified
		hostProfileReports["N5"] = []profileData{{"200", "L5", "D5"}}
		reportHostProfs(t, "N2", "N5")
		expectedProfileStatuses["N5"] = fleet.MDMDeliveryVerified
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for N5 was not resent
		verifyCommands(0, syncml.CmdStatusOK)

		// report osquery results again, this time N5 is missing and confirm that the N5 marked as
		// failed (max retries exceeded)
		reportHostProfs(t, "N2")
		expectedProfileStatuses["N5"] = fleet.MDMDeliveryFailed
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for N5 was not resent
		verifyCommands(0, syncml.CmdStatusOK)
	})
}

func (s *integrationMDMTestSuite) TestPuppetMatchPreassignProfiles() {
	ctx := context.Background()
	t := s.T()

	// before we switch to a gitops token, ensure ABM is setup
	s.enableABM(t.Name())

	// Use a gitops user for all Puppet actions
	u := &fleet.User{
		Name:       "GitOps",
		Email:      "gitops-TestPuppetMatchPreassignProfiles@example.com",
		GlobalRole: ptr.String(fleet.RoleGitOps),
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)
	s.setTokenForTest(t, "gitops-TestPuppetMatchPreassignProfiles@example.com", test.GoodPassword)

	runWithAdminToken := func(cb func()) {
		s.token = s.getTestAdminToken()
		cb()
		s.token = s.getCachedUserToken("gitops-TestPuppetMatchPreassignProfiles@example.com", test.GoodPassword)
	}

	// create a host enrolled in fleet
	mdmHost, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	// create a host that's not enrolled into MDM
	nonMDMHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID: ptr.String("not-mdm-enrolled"),
		NodeKey:       ptr.String("not-mdm-enrolled"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.not.enrolled", t.Name()),
		Platform:      "darwin",
	})
	require.NoError(t, err)

	// create a setup assistant for no team, for this we need to:
	// 1. mock the ABM API, as it gets called to set the profile
	// 2. run the DEP schedule, as this registers the default profile
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
	}))
	s.runDEPSchedule()
	noTeamProf := `{"x": 1}`
	var globalAsstResp createMDMAppleSetupAssistantResponse
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "no-team",
		EnrollmentProfile: json.RawMessage(noTeamProf),
	}, http.StatusOK, &globalAsstResp)

	// set the global Enable Release Device manually setting to true,
	// will be inherited by teams created via preassign/match.
	s.Do("PATCH", "/api/latest/fleet/setup_experience",
		json.RawMessage(jsonMustMarshal(t, map[string]any{"enable_release_device_manually": true})),
		http.StatusNoContent)

	s.runWorker()

	// preassign an empty profile, fails
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "empty", HostUUID: nonMDMHost.UUID, Profile: nil}}, http.StatusUnprocessableEntity)

	// preassign a valid profile to the MDM host
	prof1 := mobileconfigForTest("n1", "i1")
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "mdm1", HostUUID: mdmHost.UUID, Profile: prof1}}, http.StatusNoContent)

	// preassign another valid profile to the MDM host
	prof2 := mobileconfigForTest("n2", "i2")
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "mdm1", HostUUID: mdmHost.UUID, Profile: prof2, Group: "g1"}}, http.StatusNoContent)

	// preassign a valid profile to the non-MDM host, still works as the host is not validated in this call
	prof3 := mobileconfigForTest("n3", "i3")
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "non-mdm", HostUUID: nonMDMHost.UUID, Profile: prof3, Group: "g2"}}, http.StatusNoContent)

	// match with an invalid external host id, succeeds as it is the same as if
	// there was no matching to do (no preassignment was done)
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentRequest{ExternalHostIdentifier: "no-such-id"}, http.StatusNoContent)

	// match with the non-mdm host fails
	res := s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentRequest{ExternalHostIdentifier: "non-mdm"}, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "host is not enrolled in Fleet MDM")

	// match with the mdm host succeeds and creates a team based on the group labels
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentRequest{ExternalHostIdentifier: "mdm1"}, http.StatusNoContent)

	// the host is now part of that team
	h, err := s.ds.Host(ctx, mdmHost.ID)
	require.NoError(t, err)
	require.NotNil(t, h.TeamID)
	tm1, err := s.ds.Team(ctx, *h.TeamID)
	require.NoError(t, err)
	require.Equal(t, "g1", tm1.Name)
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)
	require.True(t, tm1.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)

	runWithAdminToken(func() {
		// it create activities for the new team, the profiles assigned to it,
		// the host moved to it, and setup assistant
		s.lastActivityOfTypeMatches(
			fleet.ActivityTypeCreatedTeam{}.ActivityName(),
			fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, tm1.ID, tm1.Name),
			0)
		s.lastActivityOfTypeMatches(
			fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
			fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, tm1.ID, tm1.Name),
			0)
		s.lastActivityOfTypeMatches(
			fleet.ActivityTypeTransferredHostsToTeam{}.ActivityName(),
			fmt.Sprintf(`{"team_id": %d, "team_name": %q, "host_ids": [%d], "host_display_names": [%q]}`,
				tm1.ID, tm1.Name, h.ID, h.DisplayName()),
			0)
		s.lastActivityOfTypeMatches(
			fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
			fmt.Sprintf(`{"team_id": %d, "name": %q, "team_name": %q}`,
				tm1.ID, globalAsstResp.Name, tm1.Name),
			0)
	})

	// and the team has the expected profiles (prof1 and prof2)
	profs, err := s.ds.ListMDMAppleConfigProfiles(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Len(t, profs, 2)
	// order is guaranteed by profile name
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	// setup assistant settings are copyied from "no team"
	teamAsst, err := s.ds.GetMDMAppleSetupAssistant(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Equal(t, globalAsstResp.Name, teamAsst.Name)
	require.JSONEq(t, string(globalAsstResp.Profile), string(teamAsst.Profile))

	// trigger the schedule so profiles are set in their state
	s.awaitTriggerProfileSchedule(t)
	s.runWorker()

	// the mdm host has the same profiles (i1, i2, plus fleetd config and disk encryption)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		mdmHost: {
			{Identifier: "i1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "i2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetFileVaultPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	// create a team and set profiles to it (note that it doesn't have disk encryption enabled)
	tm2, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:    "g1 - g4",
		Secrets: []*fleet.EnrollSecret{{Secret: "tm2secret"}},
	})
	require.NoError(t, err)
	prof4 := mobileconfigForTest("n4", "i4")
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
		prof1, prof4,
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm2.ID))
	// tm2 has disk encryption and release device manually disabled
	require.False(t, tm2.Config.MDM.EnableDiskEncryption)
	require.False(t, tm2.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)

	// create another team with a superset of profiles
	tm3, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:    "team3_" + t.Name(),
		Secrets: []*fleet.EnrollSecret{{Secret: "tm3secret"}},
	})
	require.NoError(t, err)
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
		prof1, prof2, prof4,
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm3.ID))

	// and yet another team with the same profiles as tm3
	tm4, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:    "team4_" + t.Name(),
		Secrets: []*fleet.EnrollSecret{{Secret: "tm4secret"}},
	})
	require.NoError(t, err)
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
		prof1, prof2, prof4,
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm4.ID))

	// preassign the MDM host to prof1 and prof4, should match existing team tm2
	//
	// additionally, use external host identifiers with different
	// suffixes to simulate real world distributed scenarios where more
	// than one puppet server might be running at the time.
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "6f36ab2c-1a40-429b-9c9d-07c9029f4aa8-puppetcompiler06.test.example.com", HostUUID: mdmHost.UUID, Profile: prof1, Group: "g1"}}, http.StatusNoContent)
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "6f36ab2c-1a40-429b-9c9d-07c9029f4aa8-puppetcompiler01.test.example.com", HostUUID: mdmHost.UUID, Profile: prof4, Group: "g4"}}, http.StatusNoContent)

	// match with the mdm host succeeds and assigns it to tm2
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentRequest{ExternalHostIdentifier: "6f36ab2c-1a40-429b-9c9d-07c9029f4aa8-puppetcompiler03.test.example.com"}, http.StatusNoContent)

	// the host is now part of that team
	h, err = s.ds.Host(ctx, mdmHost.ID)
	require.NoError(t, err)
	require.NotNil(t, h.TeamID)
	require.Equal(t, tm2.ID, *h.TeamID)
	// tm2 still has disk encryption and release device manually disabled
	tm2, err = s.ds.Team(ctx, *h.TeamID)
	require.NoError(t, err)
	require.False(t, tm2.Config.MDM.EnableDiskEncryption)
	require.False(t, tm2.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)

	// the host's profiles are:
	// - the same as the team's and are pending (prof1 + prof4)
	// - prof2 + old filevault are pending removal
	// - fleetd config being reinstalled (for new enroll secret)
	s.awaitTriggerProfileSchedule(t)

	// useful for debugging
	// mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
	// 	mysql.DumpTable(t, q, "host_mdm_apple_profiles")
	// 	return nil
	// })
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		mdmHost: {
			{Identifier: "i1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "i4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	// create a new mdm host enrolled in fleet
	mdmHost2, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	// make it part of team 2
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &tm2.ID, HostIDs: []uint{mdmHost2.ID}}, http.StatusOK)

	// simulate having its profiles installed
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		res, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET status = ? WHERE host_uuid = ?`, fleet.OSSettingsVerifying, mdmHost2.UUID)
		n, _ := res.RowsAffected()
		require.Equal(t, 4, int(n))
		return err
	})

	// preassign the MDM host using "g1" and "g4", should match existing
	// team tm2, and nothing be done since the host is already in tm2
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "mdm2", HostUUID: mdmHost2.UUID, Profile: prof1, Group: "g1"}}, http.StatusNoContent)
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "mdm2", HostUUID: mdmHost2.UUID, Profile: prof4, Group: "g4"}}, http.StatusNoContent)
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentRequest{ExternalHostIdentifier: "mdm2"}, http.StatusNoContent)

	// the host is still part of tm2
	h, err = s.ds.Host(ctx, mdmHost2.ID)
	require.NoError(t, err)
	require.NotNil(t, h.TeamID)
	require.Equal(t, tm2.ID, *h.TeamID)

	// and its profiles have been left untouched
	s.awaitTriggerProfileSchedule(t)

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		mdmHost2: {
			{Identifier: "i1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "i4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
}

// while s.TestPuppetMatchPreassignProfiles focuses on many edge cases/extra
// checks around profile assignment, this test is mainly focused on
// simulating a few puppet runs in scenarios we want to support, and ensuring that:
//
// - different hosts end up in the right teams
// - teams get edited as expected
// - commands to add/remove profiles are issued adequately
func (s *integrationMDMTestSuite) TestPuppetRun() {
	t := s.T()
	ctx := context.Background()

	// define a few profiles
	prof1, prof2, prof3, prof4 := mobileconfigForTest("n1", "i1"),
		mobileconfigForTest("n2", "i2"),
		mobileconfigForTest("n3", "i3"),
		mobileconfigForTest("n4", "i4")

	// create three hosts
	host1, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	host2, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	host3, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.runWorker()

	// Set up a mock Apple DEP API
	s.enableABM(t.Name())
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "session123"}`))
		case "/account":
			_, _ = w.Write([]byte(fmt.Sprintf(`{"admin_id": "admin123", "org_name": "%s"}`, "foo")))
		case "/profile":
			w.WriteHeader(http.StatusOK)
			require.NoError(t, encoder.Encode(godep.ProfileResponse{ProfileUUID: "profile123"}))
		}
	}))

	// Use a gitops user for all Puppet actions
	u := &fleet.User{
		Name:       "GitOps",
		Email:      "gitops-TestPuppetRun@example.com",
		GlobalRole: ptr.String(fleet.RoleGitOps),
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)
	s.setTokenForTest(t, "gitops-TestPuppetRun@example.com", test.GoodPassword)

	// preassignAndMatch simulates the puppet module doing all the
	// preassign/match calls for a given set of profiles.
	preassignAndMatch := func(profs []fleet.MDMApplePreassignProfilePayload) {
		require.NotEmpty(t, profs)
		for _, prof := range profs {
			s.Do(
				"POST",
				"/api/latest/fleet/mdm/apple/profiles/preassign",
				preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: prof},
				http.StatusNoContent,
			)
		}
		s.Do(
			"POST",
			"/api/latest/fleet/mdm/apple/profiles/match",
			matchMDMApplePreassignmentRequest{ExternalHostIdentifier: profs[0].ExternalHostIdentifier},
			http.StatusNoContent,
		)
	}

	// node default {
	//   fleetdm::profile { 'n1':
	//     template => template('n1.mobileconfig.erb'),
	//     group    => 'base',
	//   }
	//
	//   fleetdm::profile { 'n2':
	//     template => template('n2.mobileconfig.erb'),
	//     group    => 'workstations',
	//   }
	//
	//   fleetdm::profile { 'n3':
	//     template => template('n3.mobileconfig.erb'),
	//     group    => 'workstations',
	//   }
	//
	//   if $facts['system_profiler']['hardware_uuid'] == 'host_2_uuid' {
	//       fleetdm::profile { 'n4':
	//         template => template('fleetdm/n4.mobileconfig.erb'),
	//         group    => 'kiosks',
	//       }
	//   }
	puppetRun := func(host *fleet.Host) {
		payload := []fleet.MDMApplePreassignProfilePayload{
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof1,
				Group:                  "base",
			},
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof2,
				Group:                  "workstations",
			},
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof3,
				Group:                  "workstations",
			},
		}

		if host.UUID == host2.UUID {
			payload = append(payload, fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof4,
				Group:                  "kiosks",
			})
		}

		preassignAndMatch(payload)
	}

	// host1 checks in
	puppetRun(host1)

	// the host now belongs to a team
	h1, err := s.ds.Host(ctx, host1.ID)
	require.NoError(t, err)
	require.NotNil(t, h1.TeamID)

	// the team has the right name
	tm1, err := s.ds.Team(ctx, *h1.TeamID)
	require.NoError(t, err)
	require.Equal(t, "base - workstations", tm1.Name)
	// and the right profiles
	profs, err := s.ds.ListMDMAppleConfigProfiles(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Len(t, profs, 3)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof3, []byte(profs[2].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// host2 checks in
	puppetRun(host2)
	// a new team is created
	h2, err := s.ds.Host(ctx, host2.ID)
	require.NoError(t, err)
	require.NotNil(t, h2.TeamID)

	// the team has the right name
	tm2, err := s.ds.Team(ctx, *h2.TeamID)
	require.NoError(t, err)
	require.Equal(t, "base - kiosks - workstations", tm2.Name)
	// and the right profiles
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm2.ID)
	require.NoError(t, err)
	require.Len(t, profs, 4)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof3, []byte(profs[2].Mobileconfig))
	require.Equal(t, prof4, []byte(profs[3].Mobileconfig))
	require.True(t, tm2.Config.MDM.EnableDiskEncryption)

	// host3 checks in
	puppetRun(host3)
	// it belongs to the same team as host1
	h3, err := s.ds.Host(ctx, host3.ID)
	require.NoError(t, err)
	require.Equal(t, h1.TeamID, h3.TeamID)

	// prof2 is edited
	oldProf2 := prof2
	prof2 = mobileconfigForTest("n2", "i2-v2")
	// host3 checks in again
	puppetRun(host3)
	// still belongs to the same team
	h3, err = s.ds.Host(ctx, host3.ID)
	require.NoError(t, err)
	require.Equal(t, tm1.ID, *h3.TeamID)

	// but the team has prof2 updated
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Len(t, profs, 3)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof3, []byte(profs[2].Mobileconfig))
	require.NotEqual(t, oldProf2, []byte(profs[1].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// host2 checks in, still belongs to the same team
	puppetRun(host2)
	h2, err = s.ds.Host(ctx, host2.ID)
	require.NoError(t, err)
	require.Equal(t, tm2.ID, *h2.TeamID)

	// but the team has prof2 updated as well
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm2.ID)
	require.NoError(t, err)
	require.Len(t, profs, 4)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof3, []byte(profs[2].Mobileconfig))
	require.Equal(t, prof4, []byte(profs[3].Mobileconfig))
	require.NotEqual(t, oldProf2, []byte(profs[1].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// the puppet manifest is changed, and prof3 is removed
	// node default {
	//   fleetdm::profile { 'n1':
	//     template => template('n1.mobileconfig.erb'),
	//     group    => 'base',
	//   }
	//
	//   fleetdm::profile { 'n2':
	//     template => template('n2.mobileconfig.erb'),
	//     group    => 'workstations',
	//   }
	//
	//   if $facts['system_profiler']['hardware_uuid'] == 'host_2_uuid' {
	//       fleetdm::profile { 'n4':
	//         template => template('fleetdm/n4.mobileconfig.erb'),
	//         group    => 'kiosks',
	//       }
	//   }
	puppetRun = func(host *fleet.Host) {
		payload := []fleet.MDMApplePreassignProfilePayload{
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof1,
				Group:                  "base",
			},
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof2,
				Group:                  "workstations",
			},
		}

		if host.UUID == host2.UUID {
			payload = append(payload, fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof4,
				Group:                  "kiosks",
			})
		}

		preassignAndMatch(payload)
	}

	// host1 checks in again
	puppetRun(host1)
	// still belongs to the same team
	h1, err = s.ds.Host(ctx, host1.ID)
	require.NoError(t, err)
	require.Equal(t, tm1.ID, *h1.TeamID)

	// but the team doesn't have prof3 anymore
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Len(t, profs, 2)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// same for host2
	puppetRun(host2)
	h2, err = s.ds.Host(ctx, host2.ID)
	require.NoError(t, err)
	require.Equal(t, tm2.ID, *h2.TeamID)
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm2.ID)
	require.NoError(t, err)
	require.Len(t, profs, 3)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof4, []byte(profs[2].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// The puppet manifest is drastically updated, this time to use exclusions on host3:
	//
	// node default {
	//   fleetdm::profile { 'n1':
	//     template => template('n1.mobileconfig.erb'),
	//     group    => 'base',
	//   }
	//
	//   fleetdm::profile { 'n2':
	//     template => template('n2.mobileconfig.erb'),
	//     group    => 'workstations',
	//   }
	//
	//   if $facts['system_profiler']['hardware_uuid'] == 'host_3_uuid' {
	//       fleetdm::profile { 'n3':
	//         template => template('fleetdm/n3.mobileconfig.erb'),
	//         group    => 'no-nudge',
	//       }
	//   } else {
	//       fleetdm::profile { 'n3':
	//         ensure => absent,
	//         template => template('fleetdm/n3.mobileconfig.erb'),
	//         group    => 'workstations',
	//       }
	//   }
	// }
	puppetRun = func(host *fleet.Host) {
		manifest := []fleet.MDMApplePreassignProfilePayload{
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof1,
				Group:                  "base",
			},
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof2,
				Group:                  "workstations",
			},
		}

		if host.UUID == host3.UUID {
			manifest = append(manifest, fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof3,
				Group:                  "no-nudge",
				Exclude:                true,
			})
		} else {
			manifest = append(manifest, fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof3,
				Group:                  "workstations",
			})
		}

		preassignAndMatch(manifest)
	}

	// host1 checks in
	puppetRun(host1)

	// the host belongs to the same team
	h1, err = s.ds.Host(ctx, host1.ID)
	require.NoError(t, err)
	require.Equal(t, tm1.ID, *h1.TeamID)

	// the team has the right profiles
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Len(t, profs, 3)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof3, []byte(profs[2].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// host2 checks in
	puppetRun(host2)
	// it is assigned to tm1
	h2, err = s.ds.Host(ctx, host2.ID)
	require.NoError(t, err)
	require.Equal(t, tm1.ID, *h2.TeamID)

	// host3 checks in
	puppetRun(host3)

	// it is assigned to a new team
	h3, err = s.ds.Host(ctx, host3.ID)
	require.NoError(t, err)
	require.NotNil(t, h3.TeamID)
	require.NotEqual(t, tm1.ID, *h3.TeamID)
	require.NotEqual(t, tm2.ID, *h3.TeamID)

	// a new team is created
	tm3, err := s.ds.Team(ctx, *h3.TeamID)
	require.NoError(t, err)
	require.Equal(t, "base - no-nudge - workstations", tm3.Name)
	// and the right profiles
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm3.ID)
	require.NoError(t, err)
	require.Len(t, profs, 2)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.True(t, tm3.Config.MDM.EnableDiskEncryption)
}

func (s *integrationMDMTestSuite) TestMDMAppleListConfigProfiles() {
	t := s.T()
	ctx := context.Background()

	testTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "TestTeam"})
	require.NoError(t, err)

	mdmHost, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.runWorker()

	t.Run("no profiles", func(t *testing.T) {
		var listResp listMDMAppleConfigProfilesResponse
		s.DoJSON("GET", "/api/v1/fleet/mdm/apple/profiles", nil, http.StatusOK, &listResp)
		require.NotNil(t, listResp.ConfigProfiles) // expect empty slice instead of nil
		require.Len(t, listResp.ConfigProfiles, 0)

		listResp = listMDMAppleConfigProfilesResponse{}
		s.DoJSON("GET", fmt.Sprintf(`/api/v1/fleet/mdm/apple/profiles?team_id=%d`, testTeam.ID), nil, http.StatusOK, &listResp)
		require.NotNil(t, listResp.ConfigProfiles) // expect empty slice instead of nil
		require.Len(t, listResp.ConfigProfiles, 0)

		var hostProfilesResp getHostProfilesResponse
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d/configuration_profiles", mdmHost.ID), nil, http.StatusOK, &hostProfilesResp)
		require.NotNil(t, hostProfilesResp.Profiles) // expect empty slice instead of nil
		require.Len(t, hostProfilesResp.Profiles, 0)
		require.EqualValues(t, mdmHost.ID, hostProfilesResp.HostID)
	})

	t.Run("with profiles", func(t *testing.T) {
		p1, err := fleet.NewMDMAppleConfigProfile(mcBytesForTest("p1", "p1.identifier", "p1.uuid"), nil)
		require.NoError(t, err)
		_, err = s.ds.NewMDMAppleConfigProfile(ctx, *p1)
		require.NoError(t, err)

		p2, err := fleet.NewMDMAppleConfigProfile(mcBytesForTest("p2", "p2.identifier", "p2.uuid"), &testTeam.ID)
		require.NoError(t, err)
		_, err = s.ds.NewMDMAppleConfigProfile(ctx, *p2)
		require.NoError(t, err)

		var resp listMDMAppleConfigProfilesResponse
		s.DoJSON("GET", "/api/latest/fleet/mdm/apple/profiles", listMDMAppleConfigProfilesRequest{TeamID: 0}, http.StatusOK, &resp)
		require.NotNil(t, resp.ConfigProfiles)
		require.Len(t, resp.ConfigProfiles, 1)
		require.Equal(t, p1.Name, resp.ConfigProfiles[0].Name)
		require.Equal(t, p1.Identifier, resp.ConfigProfiles[0].Identifier)

		resp = listMDMAppleConfigProfilesResponse{}
		s.DoJSON("GET", fmt.Sprintf(`/api/v1/fleet/mdm/apple/profiles?team_id=%d`, testTeam.ID), nil, http.StatusOK, &resp)
		require.NotNil(t, resp.ConfigProfiles)
		require.Len(t, resp.ConfigProfiles, 1)
		require.Equal(t, p2.Name, resp.ConfigProfiles[0].Name)
		require.Equal(t, p2.Identifier, resp.ConfigProfiles[0].Identifier)

		p3, err := fleet.NewMDMAppleConfigProfile(mcBytesForTest("p3", "p3.identifier", "p3.uuid"), &testTeam.ID)
		require.NoError(t, err)
		_, err = s.ds.NewMDMAppleConfigProfile(ctx, *p3)
		require.NoError(t, err)

		resp = listMDMAppleConfigProfilesResponse{}
		s.DoJSON("GET", fmt.Sprintf(`/api/v1/fleet/mdm/apple/profiles?team_id=%d`, testTeam.ID), nil, http.StatusOK, &resp)
		require.NotNil(t, resp.ConfigProfiles)
		require.Len(t, resp.ConfigProfiles, 2)
		for _, p := range resp.ConfigProfiles {
			if p.Name == p2.Name { //nolint:gocritic // ignore ifElseChain
				require.Equal(t, p2.Identifier, p.Identifier)
			} else if p.Name == p3.Name {
				require.Equal(t, p3.Identifier, p.Identifier)
			} else {
				require.Fail(t, "unexpected profile name")
			}
		}

		var hostProfilesResp getHostProfilesResponse
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d/configuration_profiles", mdmHost.ID), nil, http.StatusOK, &hostProfilesResp)
		require.NotNil(t, hostProfilesResp.Profiles)
		require.Len(t, hostProfilesResp.Profiles, 1)
		require.Equal(t, p1.Name, hostProfilesResp.Profiles[0].Name)
		require.Equal(t, p1.Identifier, hostProfilesResp.Profiles[0].Identifier)
		require.EqualValues(t, mdmHost.ID, hostProfilesResp.HostID)

		// add the host to a team
		err = s.ds.AddHostsToTeam(ctx, &testTeam.ID, []uint{mdmHost.ID})
		require.NoError(t, err)

		hostProfilesResp = getHostProfilesResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d/configuration_profiles", mdmHost.ID), nil, http.StatusOK, &hostProfilesResp)
		require.NotNil(t, hostProfilesResp.Profiles)
		require.Len(t, hostProfilesResp.Profiles, 2)
		require.EqualValues(t, mdmHost.ID, hostProfilesResp.HostID)
	})
}

func (s *integrationMDMTestSuite) TestAppConfigMDMCustomSettings() {
	t := s.T()

	// set the macos custom settings fields with the deprecated Labels field
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
		  "macos_settings": {
				"custom_settings": [
					{"path": "foo", "labels": ["baz"]},
					{"path": "bar"}
				]
			}
		}
  }`), http.StatusOK, &acResp)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "foo", LabelsIncludeAll: []string{"baz"}}, {Path: "bar"}}, acResp.MDM.MacOSSettings.CustomSettings)

	// check that they are returned by a GET /config
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "foo", LabelsIncludeAll: []string{"baz"}}, {Path: "bar"}}, acResp.MDM.MacOSSettings.CustomSettings)

	// set the windows custom settings fields with included/excluded labels
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"windows_settings": {
				"custom_settings": [
					{"path": "foo", "labels_exclude_any": ["x", "y"]},
					{"path": "bar", "labels_include_all": ["a", "b"]},
					{"path": "baz", "labels": ["c"]}
				]
			}
		}
  }`), http.StatusOK, &acResp)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "foo", LabelsIncludeAll: []string{"baz"}}, {Path: "bar"}}, acResp.MDM.MacOSSettings.CustomSettings)
	assert.Equal(t, optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "foo", LabelsExcludeAny: []string{"x", "y"}}, {Path: "bar", LabelsIncludeAll: []string{"a", "b"}}, {Path: "baz", LabelsIncludeAll: []string{"c"}}}), acResp.MDM.WindowsSettings.CustomSettings)

	// check that they are returned by a GET /config
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "foo", LabelsIncludeAll: []string{"baz"}}, {Path: "bar"}}, acResp.MDM.MacOSSettings.CustomSettings)
	assert.Equal(t, optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "foo", LabelsExcludeAny: []string{"x", "y"}}, {Path: "bar", LabelsIncludeAll: []string{"a", "b"}}, {Path: "baz", LabelsIncludeAll: []string{"c"}}}), acResp.MDM.WindowsSettings.CustomSettings)

	// patch without specifying the windows/macos custom settings fields and an unrelated
	// field, should not remove them
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "foo", LabelsIncludeAll: []string{"baz"}}, {Path: "bar"}}, acResp.MDM.MacOSSettings.CustomSettings)
	assert.Equal(t, optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "foo", LabelsExcludeAny: []string{"x", "y"}}, {Path: "bar", LabelsIncludeAll: []string{"a", "b"}}, {Path: "baz", LabelsIncludeAll: []string{"c"}}}), acResp.MDM.WindowsSettings.CustomSettings)

	// patch with explicitly empty macos/windows custom settings fields, would remove
	// them but this is a dry-run
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"macos_settings": { "custom_settings": null },
			"windows_settings": { "custom_settings": null }
		}
  }`), http.StatusOK, &acResp, "dry_run", "true")
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "foo", LabelsIncludeAll: []string{"baz"}}, {Path: "bar"}}, acResp.MDM.MacOSSettings.CustomSettings)
	assert.Equal(t, optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "foo", LabelsExcludeAny: []string{"x", "y"}}, {Path: "bar", LabelsIncludeAll: []string{"a", "b"}}, {Path: "baz", LabelsIncludeAll: []string{"c"}}}), acResp.MDM.WindowsSettings.CustomSettings)

	// patch with explicitly empty macos custom settings fields, removes them
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"macos_settings": { "custom_settings": null },
			"windows_settings": { "custom_settings": null }
		}
  }`), http.StatusOK, &acResp)
	assert.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)
	assert.Equal(t, optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}}, acResp.MDM.WindowsSettings.CustomSettings)

	// mix of labels fields returns an error
	res := s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
		  "macos_settings": {
				"custom_settings": [
					{"path": "foo", "labels": ["a"], "labels_exclude_any": ["b"]}
				]
			}
		}
  }`), http.StatusUnprocessableEntity)
	msg := extractServerErrorText(res.Body)
	require.Contains(t, msg, `For each profile, only one of "labels_exclude_any", "labels_include_all", "labels_include_any" or "labels" can be included.`)

	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
		  "windows_settings": {
				"custom_settings": [
					{"path": "foo", "labels_include_all": ["a"], "labels_exclude_any": ["b"]}
				]
			}
		}
  }`), http.StatusUnprocessableEntity)
	msg = extractServerErrorText(res.Body)
	require.Contains(t, msg, `For each profile, only one of "labels_exclude_any", "labels_include_all", "labels_include_any" or "labels" can be included.`)

	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
		  "windows_settings": {
				"custom_settings": [
					{"path": "foo", "labels_include_any": ["a"], "labels_exclude_any": ["b"]}
				]
			}
		}
  }`), http.StatusUnprocessableEntity)
	msg = extractServerErrorText(res.Body)
	require.Contains(t, msg, `For each profile, only one of "labels_exclude_any", "labels_include_all", "labels_include_any" or "labels" can be included.`)

	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
		  "windows_settings": {
				"custom_settings": [
					{"path": "foo", "labels": ["a"], "labels_include_any": ["b"]}
				]
			}
		}
  }`), http.StatusUnprocessableEntity)
	msg = extractServerErrorText(res.Body)
	require.Contains(t, msg, `For each profile, only one of "labels_exclude_any", "labels_include_all", "labels_include_any" or "labels" can be included.`)
}

func (s *integrationMDMTestSuite) TestApplyTeamsMDMAppleProfiles() {
	t := s.T()

	// create a team through the service so it initializes the agent ops
	teamName := t.Name() + "team1"
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc team1",
	}
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	team = createTeamResp.Team

	// apply with custom macos settings
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{
				"custom_settings": []map[string]interface{}{
					{"path": "foo", "labels": []string{"a", "b"}},
					{"path": "bar", "labels_exclude_any": []string{"c"}},
				},
			},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// retrieving the team returns the custom macos settings
	var teamResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []fleet.MDMProfileSpec{
		{Path: "foo", LabelsIncludeAll: []string{"a", "b"}},
		{Path: "bar", LabelsExcludeAny: []string{"c"}},
	}, teamResp.Team.Config.MDM.MacOSSettings.CustomSettings)

	// apply with invalid macos settings subfield should fail
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{"foo_bar": 123},
		},
	}}}
	res := s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, `unsupported key provided: "foo_bar"`)

	// apply with some good and some bad macos settings subfield should fail
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{"custom_settings": []interface{}{"A", true}},
		},
	}}}
	res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, `invalid value type at 'macos_settings.custom_settings': expected array of MDMProfileSpecs but got bool`)

	// apply without custom macos settings specified and unrelated field, should
	// not replace existing settings
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(false),
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []fleet.MDMProfileSpec{
		{Path: "foo", LabelsIncludeAll: []string{"a", "b"}},
		{Path: "bar", LabelsExcludeAny: []string{"c"}},
	}, teamResp.Team.Config.MDM.MacOSSettings.CustomSettings)

	// apply with explicitly empty custom macos settings would clear the existing
	// settings, but dry-run
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{"custom_settings": []map[string]interface{}{}},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "dry_run", "true")
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []fleet.MDMProfileSpec{
		{Path: "foo", LabelsIncludeAll: []string{"a", "b"}},
		{Path: "bar", LabelsExcludeAny: []string{"c"}},
	}, teamResp.Team.Config.MDM.MacOSSettings.CustomSettings)

	// apply with explicitly empty custom macos settings clears the existing settings
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{"custom_settings": []map[string]interface{}{}},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []fleet.MDMProfileSpec{}, teamResp.Team.Config.MDM.MacOSSettings.CustomSettings)

	// apply with invalid mix of labels fails
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{
				"custom_settings": []map[string]interface{}{
					{"path": "bar", "labels": []string{"x"}},
					{"path": "foo", "labels": []string{"a", "b"}, "labels_include_all": []string{"c"}},
				},
			},
		},
	}}}
	res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, `For each profile, only one of "labels_exclude_any", "labels_include_all", "labels_include_any" or "labels" can be included.`)
}

func (s *integrationMDMTestSuite) TestBatchSetMDMAppleProfiles() {
	t := s.T()
	ctx := context.Background()

	bigString := strings.Repeat("a", 1024*1024+1)

	// create a new team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "batch_set_mdm_profiles"})
	require.NoError(t, err)

	// apply an empty set to no-team
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: nil}, http.StatusNoContent)
	s.lastActivityMatches(
		fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null}`,
		0,
	)

	// apply to both team id and name
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: nil},
		http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID), "team_name", tm.Name)

	// invalid team name
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: nil},
		http.StatusNotFound, "team_name", uuid.New().String())

	// Profile is too big
	resp := s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{[]byte(bigString)}},
		http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(resp.Body), "maximum configuration profile file size is 1 MB")

	// duplicate profile names
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
		mobileconfigForTest("N1", "I1"),
		mobileconfigForTest("N1", "I2"),
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))

	// profiles with reserved identifiers
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		res := s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
			mobileconfigForTest("N1", "I1"),
			mobileconfigForTest(p, p),
		}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: payload identifier %s is not allowed", p))
	}

	// payloads with reserved types
	for p := range mobileconfig.FleetPayloadTypes() {
		res := s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
			mobileconfigForTestWithContent("N1", "I1", "II1", p, ""),
		}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadType(s): %s", p))
	}

	// payloads with reserved identifiers
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		res := s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
			mobileconfigForTestWithContent("N1", "I1", p, "random", ""),
		}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadIdentifier(s): %s", p))
	}

	// successfully apply a profile for the team
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
		mobileconfigForTest("N1", "I1"),
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))
	s.lastActivityMatches(
		fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, tm.ID, tm.Name),
		0,
	)
}

func (s *integrationMDMTestSuite) TestHostMDMAppleProfilesStatus() {
	t := s.T()
	ctx := context.Background()

	createManualMDMEnrollWithOrbit := func(secret string) *fleet.Host {
		// orbit enrollment happens before mdm enrollment, otherwise the host would
		// always receive the "no team" profiles on mdm enrollment since it would
		// not be part of any team yet (team assignment is done when it enrolls
		// with orbit).
		mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
			SCEPChallenge: s.scepChallenge,
			SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
			MDMURL:        s.server.URL + apple_mdm.MDMPath,
		}, "MacBookPro16,1")

		// enroll the device with orbit
		var resp EnrollOrbitResponse
		s.DoJSON("POST", "/api/fleet/orbit/enroll", EnrollOrbitRequest{
			EnrollSecret:   secret,
			HardwareUUID:   mdmDevice.UUID, // will not match any existing host
			HardwareSerial: mdmDevice.SerialNumber,
		}, http.StatusOK, &resp)
		require.NotEmpty(t, resp.OrbitNodeKey)
		orbitNodeKey := resp.OrbitNodeKey
		h, err := s.ds.LoadHostByOrbitNodeKey(ctx, orbitNodeKey)
		require.NoError(t, err)
		h.OrbitNodeKey = &orbitNodeKey
		h.Platform = "darwin"

		err = mdmDevice.Enroll()
		require.NoError(t, err)

		return h
	}

	triggerReconcileProfiles := func() {
		s.awaitTriggerProfileSchedule(t)
		// this will only mark them as "pending", as the response to confirm
		// profile deployment is asynchronous, so we simulate it here by
		// updating any "pending" (not NULL) profiles to "verifying"
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET status = ? WHERE status = ?`, fleet.OSSettingsVerifying, fleet.OSSettingsPending)
			return err
		})
	}

	assignHostToTeam := func(h *fleet.Host, teamID *uint) {
		var moveHostResp addHostsToTeamResponse
		s.DoJSON("POST", "/api/v1/fleet/hosts/transfer",
			addHostsToTeamRequest{TeamID: teamID, HostIDs: []uint{h.ID}}, http.StatusOK, &moveHostResp)

		h.TeamID = teamID
	}

	// add a couple global profiles
	globalProfiles := [][]byte{
		mobileconfigForTest("G1", "G1"),
		mobileconfigForTest("G2", "G2"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)
	// create the no-team enroll secret
	var applyResp applyEnrollSecretSpecResponse
	globalEnrollSec := "global_enroll_sec"
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret",
		applyEnrollSecretSpecRequest{
			Spec: &fleet.EnrollSecretSpec{
				Secrets: []*fleet.EnrollSecret{{Secret: globalEnrollSec}},
			},
		}, http.StatusOK, &applyResp)

	// create a team with a couple profiles
	tm1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team_profiles_status_1"})
	require.NoError(t, err)
	tm1Profiles := [][]byte{
		mobileconfigForTest("T1.1", "T1.1"),
		mobileconfigForTest("T1.2", "T1.2"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{Profiles: tm1Profiles}, http.StatusNoContent,
		"team_id", fmt.Sprint(tm1.ID))
	// create the team 1 enroll secret
	var teamResp teamEnrollSecretsResponse
	tm1EnrollSec := "team1_enroll_sec"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", tm1.ID),
		modifyTeamEnrollSecretsRequest{
			Secrets: []fleet.EnrollSecret{{Secret: tm1EnrollSec}},
		}, http.StatusOK, &teamResp)

	// create another team with different profiles
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team_profiles_status_2"})
	require.NoError(t, err)
	tm2Profiles := [][]byte{
		mobileconfigForTest("T2.1", "T2.1"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{Profiles: tm2Profiles}, http.StatusNoContent,
		"team_id", fmt.Sprint(tm2.ID))

	// enroll a couple hosts in no team
	h1 := createManualMDMEnrollWithOrbit(globalEnrollSec)
	require.Nil(t, h1.TeamID)
	h2 := createManualMDMEnrollWithOrbit(globalEnrollSec)
	require.Nil(t, h2.TeamID)
	// run the cron
	s.awaitTriggerProfileSchedule(t)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		h2: {
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	// enroll a couple hosts in team 1
	h3 := createManualMDMEnrollWithOrbit(tm1EnrollSec)
	require.NotNil(t, h3.TeamID)
	require.Equal(t, tm1.ID, *h3.TeamID)
	h4 := createManualMDMEnrollWithOrbit(tm1EnrollSec)
	require.NotNil(t, h4.TeamID)
	require.Equal(t, tm1.ID, *h4.TeamID)
	// run the cron
	s.awaitTriggerProfileSchedule(t)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h3: {
			{Identifier: "T1.1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T1.2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		h4: {
			{Identifier: "T1.1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T1.2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	// apply the pending profiles
	triggerReconcileProfiles()

	// switch a no team host (h1) to a team (tm2)
	var moveHostResp addHostsToTeamResponse
	s.DoJSON("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &tm2.ID, HostIDs: []uint{h1.ID}}, http.StatusOK, &moveHostResp)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h2: {
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// switch a team host (h3) to another team (tm2)
	s.DoJSON("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &tm2.ID, HostIDs: []uint{h3.ID}}, http.StatusOK, &moveHostResp)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h3: {
			{Identifier: "T1.1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T1.2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h4: {
			{Identifier: "T1.1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "T1.2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// switch a team host (h4) to no team
	s.DoJSON("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: nil, HostIDs: []uint{h4.ID}}, http.StatusOK, &moveHostResp)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h3: {
			{Identifier: "T1.1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T1.2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h4: {
			{Identifier: "T1.1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T1.2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// apply the pending profiles
	triggerReconcileProfiles()

	// add a profile to no team (h2 and h4 are now part of no team)
	body, headers := generateNewProfileMultipartRequest(t,
		"some_name", mobileconfigForTest("G3", "G3"), s.token, nil)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusOK, headers)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h2: {
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		h4: {
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// add a profile to team 2 (h1 and h3 are now part of team 2)
	body, headers = generateNewProfileMultipartRequest(t,
		"some_name", mobileconfigForTest("T2.2", "T2.2"), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm2.ID)}})
	s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusOK, headers)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "T2.1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "T2.2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "T2.1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "T2.2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// apply the pending profiles
	triggerReconcileProfiles()

	// delete a no team profile
	noTeamProfs, err := s.ds.ListMDMAppleConfigProfiles(ctx, nil)
	require.NoError(t, err)
	var g1ProfID uint
	for _, p := range noTeamProfs {
		if p.Identifier == "G1" {
			g1ProfID = p.ProfileID
			break
		}
	}
	require.NotZero(t, g1ProfID)
	var delProfResp deleteMDMAppleConfigProfileResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", g1ProfID),
		deleteMDMAppleConfigProfileRequest{}, http.StatusOK, &delProfResp)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h2: {
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h4: {
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// delete a team profile
	tm2Profs, err := s.ds.ListMDMAppleConfigProfiles(ctx, &tm2.ID)
	require.NoError(t, err)
	var tm21ProfID uint
	for _, p := range tm2Profs {
		if p.Identifier == "T2.1" {
			tm21ProfID = p.ProfileID
			break
		}
	}
	require.NotZero(t, tm21ProfID)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", tm21ProfID),
		deleteMDMAppleConfigProfileRequest{}, http.StatusOK, &delProfResp)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "T2.1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "T2.1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// apply the pending profiles
	triggerReconcileProfiles()

	// bulk-set profiles for no team, with add/delete/edit
	g2Edited := mobileconfigForTest("G2b", "G2b")
	g4Content := mobileconfigForTest("G4", "G4")
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{
			Profiles: [][]byte{
				g2Edited,
				// G3 is deleted
				g4Content,
			},
		}, http.StatusNoContent)

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h2: {
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G3", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h4: {
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G3", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// bulk-set profiles for a team, with add/delete/edit
	t22Edited := mobileconfigForTest("T2.2b", "T2.2b")
	t23Content := mobileconfigForTest("T2.3", "T2.3")
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{
			Profiles: [][]byte{
				t22Edited,
				t23Content,
			},
		}, http.StatusNoContent, "team_id", fmt.Sprint(tm2.ID))
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "T2.2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "T2.2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// apply the pending profiles
	triggerReconcileProfiles()

	// bulk-set profiles for no team and team 2, without changes, and team 1 added (but no host affected)
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{
			Profiles: [][]byte{
				g2Edited,
				g4Content,
			},
		}, http.StatusNoContent)
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{
			Profiles: [][]byte{
				t22Edited,
				t23Content,
			},
		}, http.StatusNoContent, "team_id", fmt.Sprint(tm2.ID))
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{
			Profiles: [][]byte{
				mobileconfigForTest("T1.3", "T1.3"),
			},
		}, http.StatusNoContent, "team_id", fmt.Sprint(tm1.ID))
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "T2.2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "T2.3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h2: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "T2.2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "T2.3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h4: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// delete team 2 (h1 and h3 are part of that team)
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d", tm2.ID), nil, http.StatusOK)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "T2.2b", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.3", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "T2.2b", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.3", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// apply the pending profiles
	triggerReconcileProfiles()

	// all profiles now verifying
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h2: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h4: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// h1 verified one of the profiles
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(context.Background(), s.ds, h1, map[string]*fleet.HostMacOSProfile{
		"G2b": {Identifier: "G2b", DisplayName: "G2b", InstallDate: time.Now()},
	}))
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerified},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h2: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h4: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// switch a team host (h1) to another team (tm1)
	assignHostToTeam(h1, &tm1.ID)

	// Create a new profile that will be labeled
	body, headers = generateNewProfileMultipartRequest(
		t,
		"label_prof",
		mobileconfigForTest("label_prof", "label_prof"),
		s.token,
		map[string][]string{"team_id": {fmt.Sprintf("%d", tm1.ID)}},
	)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusOK, headers)

	var uid string
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &uid, `SELECT profile_uuid FROM mdm_apple_configuration_profiles WHERE identifier = ?`, "label_prof")
	})

	label, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "test label 1", Query: "select 1;"})
	require.NoError(t, err)

	// Update label with host membership
	mysql.ExecAdhocSQL(
		t, s.ds, func(db sqlx.ExtContext) error {
			_, err := db.ExecContext(
				context.Background(),
				"INSERT IGNORE INTO label_membership (host_id, label_id) VALUES (?, ?)",
				h1.ID,
				label.ID,
			)
			return err
		},
	)

	// Update profile <-> label mapping
	mysql.ExecAdhocSQL(
		t, s.ds, func(db sqlx.ExtContext) error {
			_, err := db.ExecContext(
				context.Background(),
				"INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, ?, ?)",
				uid,
				label.Name,
				label.ID,
			)
			return err
		},
	)

	triggerReconcileProfiles()

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "T1.3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "label_prof", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h2: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h4: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(context.Background(), s.ds, h1, map[string]*fleet.HostMacOSProfile{
		"label_prof": {Identifier: "label_prof", DisplayName: "label_prof", InstallDate: time.Now()},
	}))

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "T1.3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "label_prof", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerified},
		},
		h2: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h4: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
}

func (s *integrationMDMTestSuite) TestMDMConfigProfileCRUD() {
	t := s.T()
	ctx := context.Background()

	testTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "TestTeam"})
	require.NoError(t, err)

	// NOTE: label names starting with "-" are sent as "labels_excluding_any"
	// (and the leading "-" is removed from the name). Names starting with
	// "!" are sent as the deprecated "labels" field (and the "!" is removed).
	// Names starting with a "~" prefix are sent as "labels_include_any"
	// (and the leading "~" is removed.
	addLabelsFields := func(labelNames []string) map[string][]string {
		var deprLabels, inclAllLabels, inclAnyLabels, exclLabels []string
		for _, lbl := range labelNames {
			switch {
			case strings.HasPrefix(lbl, "~"):
				inclAnyLabels = append(inclAnyLabels, strings.TrimPrefix(lbl, "~"))
			case strings.HasPrefix(lbl, "-"):
				exclLabels = append(exclLabels, strings.TrimPrefix(lbl, "-"))
			case strings.HasPrefix(lbl, "!"):
				deprLabels = append(deprLabels, strings.TrimPrefix(lbl, "!"))
			default:
				inclAllLabels = append(inclAllLabels, lbl)
			}
		}

		fields := make(map[string][]string)
		if len(deprLabels) > 0 {
			fields["labels"] = deprLabels
		}
		if len(inclAllLabels) > 0 {
			fields["labels_include_all"] = inclAllLabels
		}
		if len(exclLabels) > 0 {
			fields["labels_exclude_any"] = exclLabels
		}
		if len(inclAnyLabels) > 0 {
			fields["labels_include_any"] = inclAnyLabels
		}
		return fields
	}

	assertAppleProfile := func(filename, name, ident string, teamID uint, labelNames []string, wantStatus int, wantErrMsg string) string {
		fields := addLabelsFields(labelNames)
		if teamID > 0 {
			fields["team_id"] = []string{fmt.Sprintf("%d", teamID)}
		}
		body, headers := generateNewProfileMultipartRequest(
			t, filename, mobileconfigForTest(name, ident), s.token, fields,
		)
		res := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), wantStatus, headers)

		if wantErrMsg != "" {
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, wantErrMsg)
			return ""
		}

		var resp newMDMConfigProfileResponse
		err := json.NewDecoder(res.Body).Decode(&resp)
		require.NoError(t, err)
		require.NotEmpty(t, resp.ProfileUUID)
		require.Equal(t, "a", string(resp.ProfileUUID[0]))
		return resp.ProfileUUID
	}
	assertAppleDeclaration := func(filename, ident string, teamID uint, labelNames []string, wantStatus int, wantErrMsg string) string {
		fields := addLabelsFields(labelNames)
		if teamID > 0 {
			fields["team_id"] = []string{fmt.Sprintf("%d", teamID)}
		}

		bytes := []byte(fmt.Sprintf(`{
  "Type": "com.apple.configuration.foo",
  "Payload": {
    "Echo": "f1337"
  },
  "Identifier": "%s"
}`, ident))

		body, headers := generateNewProfileMultipartRequest(t, filename, bytes, s.token, fields)
		res := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), wantStatus, headers)

		if wantErrMsg != "" {
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, wantErrMsg)
			return ""
		}

		var resp newMDMConfigProfileResponse
		err := json.NewDecoder(res.Body).Decode(&resp)
		require.NoError(t, err)
		require.NotEmpty(t, resp.ProfileUUID)
		require.Equal(t, fleet.MDMAppleDeclarationUUIDPrefix, string(resp.ProfileUUID[0]))
		return resp.ProfileUUID
	}

	createAppleProfile := func(name, ident string, teamID uint, labelNames []string) string {
		uid := assertAppleProfile(name+".mobileconfig", name, ident, teamID, labelNames, http.StatusOK, "")

		var wantJSON string
		if teamID == 0 {
			wantJSON = fmt.Sprintf(`{"team_id": null, "team_name": null, "profile_name": %q, "profile_identifier": %q}`, name, ident)
		} else {
			wantJSON = fmt.Sprintf(`{"team_id": %d, "team_name": %q, "profile_name": %q, "profile_identifier": %q}`, teamID, testTeam.Name, name, ident)
		}
		s.lastActivityOfTypeMatches(fleet.ActivityTypeCreatedMacosProfile{}.ActivityName(), wantJSON, 0)

		return uid
	}

	createAppleDeclaration := func(name, ident string, teamID uint, labelNames []string) string {
		uid := assertAppleDeclaration(name+".json", ident, teamID, labelNames, http.StatusOK, "")

		var wantJSON string
		if teamID == 0 {
			wantJSON = fmt.Sprintf(`{"team_id": null, "team_name": null, "profile_name": %q, "identifier": %q}`, name, ident)
		} else {
			wantJSON = fmt.Sprintf(`{"team_id": %d, "team_name": %q, "profile_name": %q, "identifier": %q}`, teamID, testTeam.Name, name, ident)
		}
		s.lastActivityOfTypeMatches(fleet.ActivityTypeCreatedDeclarationProfile{}.ActivityName(), wantJSON, 0)

		return uid
	}

	assertWindowsProfile := func(filename, locURI string, teamID uint, labelNames []string, wantStatus int, wantErrMsg string) string {
		fields := addLabelsFields(labelNames)
		if teamID > 0 {
			fields["team_id"] = []string{fmt.Sprintf("%d", teamID)}
		}
		body, headers := generateNewProfileMultipartRequest(
			t,
			filename,
			[]byte(fmt.Sprintf(`<Add><Item><Target><LocURI>%s</LocURI></Target></Item></Add><Replace><Item><Target><LocURI>%s</LocURI></Target></Item></Replace>`, locURI, locURI)),
			s.token,
			fields,
		)
		res := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), wantStatus, headers)

		if wantErrMsg != "" {
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, wantErrMsg)
			return ""
		}

		var resp newMDMConfigProfileResponse
		err := json.NewDecoder(res.Body).Decode(&resp)
		require.NoError(t, err)
		require.NotEmpty(t, resp.ProfileUUID)
		require.Equal(t, "w", string(resp.ProfileUUID[0]))
		return resp.ProfileUUID
	}
	createWindowsProfile := func(name string, teamID uint, labels []string) string {
		uid := assertWindowsProfile(name+".xml", "./Test", teamID, labels, http.StatusOK, "")

		var wantJSON string
		if teamID == 0 {
			wantJSON = fmt.Sprintf(`{"team_id": null, "team_name": null, "profile_name": %q}`, name)
		} else {
			wantJSON = fmt.Sprintf(`{"team_id": %d, "team_name": %q, "profile_name": %q}`, teamID, testTeam.Name, name)
		}
		s.lastActivityOfTypeMatches(fleet.ActivityTypeCreatedWindowsProfile{}.ActivityName(), wantJSON, 0)

		return uid
	}

	// create a couple Apple profiles for no-team and team
	noTeamAppleProfUUID := createAppleProfile("apple-global-profile", "test-global-ident", 0, nil)
	teamAppleProfUUID := createAppleProfile("apple-team-profile", "test-team-ident", testTeam.ID, nil)
	// create a couple Windows profiles for no-team and team
	noTeamWinProfUUID := createWindowsProfile("win-global-profile", 0, nil)
	teamWinProfUUID := createWindowsProfile("win-team-profile", testTeam.ID, nil)

	// Windows profile name conflicts with Apple's for no team
	assertWindowsProfile("apple-global-profile.xml", "./Test", 0, nil, http.StatusConflict, "Couldn't upload. A configuration profile with this name already exists.")
	// but no conflict for team 1
	assertWindowsProfile("apple-global-profile.xml", "./Test", testTeam.ID, nil, http.StatusOK, "")
	// Apple profile name conflicts with Windows' for no team
	assertAppleProfile("win-global-profile.mobileconfig", "win-global-profile", "test-global-ident-2", 0, nil, http.StatusConflict, "Couldn't upload. A configuration profile with this name already exists.")
	// but no conflict for team 1
	assertAppleProfile("win-global-profile.mobileconfig", "win-global-profile", "test-global-ident-2", testTeam.ID, nil, http.StatusOK, "")
	// Windows profile name conflicts with Apple's for team 1
	assertWindowsProfile("apple-team-profile.xml", "./Test", testTeam.ID, nil, http.StatusConflict, "Couldn't upload. A configuration profile with this name already exists.")
	// but no conflict for no-team
	assertWindowsProfile("apple-team-profile.xml", "./Test", 0, nil, http.StatusOK, "")
	// Apple profile name conflicts with Windows' for team 1
	assertAppleProfile("win-team-profile.mobileconfig", "win-team-profile", "test-team-ident-2", testTeam.ID, nil, http.StatusConflict, "Couldn't upload. A configuration profile with this name already exists.")
	// but no conflict for no-team
	assertAppleProfile("win-team-profile.mobileconfig", "win-team-profile", "test-team-ident-2", 0, nil, http.StatusOK, "")

	// add some macOS declarations
	createAppleDeclaration("apple-declaration", "test-declaration-ident", 0, nil)
	// identifier must be unique, it conflicts with existing declaration
	assertAppleDeclaration("apple-declaration.json", "test-declaration-ident", 0, nil, http.StatusConflict, "test-declaration-ident already exists")
	// name is pulled from filename, it conflicts with existing declaration
	assertAppleDeclaration("apple-declaration.json", "test-declaration-ident-2", 0, nil, http.StatusConflict, "apple-declaration already exists")
	// uniqueness is checked only within team, so it's fine to have the same name and identifier in different teams
	assertAppleDeclaration("apple-declaration.json", "test-declaration-ident", testTeam.ID, nil, http.StatusOK, "")
	// name is pulled from filename, it conflicts with existing macOS config profile
	assertAppleDeclaration("apple-global-profile.json", "test-declaration-ident-2", 0, nil, http.StatusConflict, "apple-global-profile already exists")
	// name is pulled from filename, it conflicts with existing macOS config profile
	assertAppleDeclaration("win-global-profile.json", "test-declaration-ident-2", 0, nil, http.StatusConflict, "win-global-profile already exists")
	// windows profile name conflicts with existing declaration
	assertWindowsProfile("apple-declaration.xml", "./Test", 0, nil, http.StatusConflict, "Couldn't upload. A configuration profile with this name already exists.")
	// macOS profile name conflicts with existing declaration
	assertAppleProfile("apple-declaration.mobileconfig", "apple-declaration", "test-declaration-ident", 0, nil, http.StatusConflict, "Couldn't upload. A configuration profile with this name already exists.")

	// not an xml nor mobileconfig file
	assertWindowsProfile("foo.txt", "./Test", 0, nil, http.StatusBadRequest, "Couldn't add profile. The file should be a .mobileconfig, XML, or JSON file.")
	assertAppleProfile("foo.txt", "foo", "foo-ident", 0, nil, http.StatusBadRequest, "Couldn't add profile. The file should be a .mobileconfig, XML, or JSON file.")
	assertAppleDeclaration("foo.txt", "foo-ident", 0, nil, http.StatusBadRequest, "Couldn't add profile. The file should be a .mobileconfig, XML, or JSON file.")

	// Windows-reserved LocURI
	assertWindowsProfile("bitlocker.xml", syncml.FleetBitLockerTargetLocURI, 0, nil, http.StatusBadRequest, "Couldn't upload. Custom configuration profiles can't include BitLocker settings.")
	assertWindowsProfile("updates.xml", syncml.FleetOSUpdateTargetLocURI, testTeam.ID, nil, http.StatusBadRequest, "Couldn't upload. Custom configuration profiles can't include Windows updates settings.")

	// Fleet-reserved profiles
	for name := range servermdm.FleetReservedProfileNames() {
		assertAppleProfile(name+".mobileconfig", name, name+"-ident", 0, nil, http.StatusBadRequest, fmt.Sprintf(`name %s is not allowed`, name))
		assertAppleDeclaration(name+".json", name+"-ident", 0, nil, http.StatusBadRequest, fmt.Sprintf(`name %q is not allowed`, name))
		assertWindowsProfile(name+".xml", "./Test", 0, nil, http.StatusBadRequest, fmt.Sprintf(`Couldn't upload. Profile name %q is not allowed.`, name))
	}

	// profiles with non-existent labels
	assertAppleProfile("apple-profile-with-labels.mobileconfig", "apple-profile-with-labels", "ident-with-labels", 0, []string{"does-not-exist"}, http.StatusBadRequest, "some or all the labels provided don't exist")
	assertAppleDeclaration("apple-declaration-with-labels.json", "ident-with-labels", 0, []string{"does-not-exist"}, http.StatusBadRequest, "some or all the labels provided don't exist")
	assertWindowsProfile("win-profile-with-labels.xml", "./Test", 0, []string{"does-not-exist"}, http.StatusBadRequest, "some or all the labels provided don't exist")

	// create a couple of labels
	labelFoo := &fleet.Label{Name: "foo", Query: "select * from foo;"}
	labelFoo, err = s.ds.NewLabel(context.Background(), labelFoo)
	require.NoError(t, err)
	labelBar := &fleet.Label{Name: "bar", Query: "select * from bar;"}
	labelBar, err = s.ds.NewLabel(context.Background(), labelBar)
	require.NoError(t, err)

	// profiles mixing existent and non-existent labels
	assertAppleProfile("apple-profile-with-labels.mobileconfig", "apple-profile-with-labels", "ident-with-labels", 0, []string{"does-not-exist", "foo"}, http.StatusBadRequest, "some or all the labels provided don't exist")
	assertAppleDeclaration("apple-declaration-with-labels.json", "ident-with-labels", 0, []string{"does-not-exist", "foo"}, http.StatusBadRequest, "some or all the labels provided don't exist")
	assertWindowsProfile("win-profile-with-labels.xml", "./Test", 0, []string{"does-not-exist", "bar"}, http.StatusBadRequest, "some or all the labels provided don't exist")

	// profiles with invalid mix of labels
	assertAppleProfile("apple-invalid-profile-with-labels.mobileconfig", "apple-invalid-profile-with-labels", "ident-with-labels", 0, []string{"foo", "!bar"}, http.StatusBadRequest, `Only one of "labels_exclude_any", "labels_include_all", "labels_include_any", or "labels" can be included.`)
	assertAppleProfile("apple-invalid-profile-with-labels.mobileconfig", "apple-invalid-profile-with-labels", "ident-with-labels", 0, []string{"foo", "~bar"}, http.StatusBadRequest, `Only one of "labels_exclude_any", "labels_include_all", "labels_include_any", or "labels" can be included.`)
	assertAppleDeclaration("apple-invalid-decl-with-labels.json", "ident-decl-with-labels", 0, []string{"foo", "-bar"}, http.StatusBadRequest, `Only one of "labels_exclude_any", "labels_include_all", "labels_include_any", or "labels" can be included.`)
	assertAppleDeclaration("apple-invalid-decl-with-labels.json", "ident-decl-with-labels", 0, []string{"foo", "~bar"}, http.StatusBadRequest, `Only one of "labels_exclude_any", "labels_include_all", "labels_include_any", or "labels" can be included.`)
	assertWindowsProfile("win-invalid-profile-with-labels.xml", "./Test", 0, []string{"-foo", "!bar"}, http.StatusBadRequest, `Only one of "labels_exclude_any", "labels_include_all", "labels_include_any", or "labels" can be included.`)
	assertWindowsProfile("win-invalid-profile-with-labels.xml", "./Test", 0, []string{"-foo", "~bar"}, http.StatusBadRequest, `Only one of "labels_exclude_any", "labels_include_all", "labels_include_any", or "labels" can be included.`)

	// profiles with valid labels
	uuidAppleWithLabel := assertAppleProfile("apple-profile-with-labels.mobileconfig", "apple-profile-with-labels", "ident-with-labels", 0, []string{"!foo"}, http.StatusOK, "")
	uuidAppleWithInclAnyLabel := assertAppleProfile("apple-profile-with-incl-any-labels.mobileconfig", "apple-profile-with-incl-any-labels", "ident-with-incl-any-labels", 0, []string{"~foo", "~bar"}, http.StatusOK, "")
	uuidAppleDDMWithLabel := createAppleDeclaration("apple-decl-with-labels", "ident-decl-with-labels", 0, []string{"foo"})
	uuidWindowsWithLabel := assertWindowsProfile("win-profile-with-labels.xml", "./Test", 0, []string{"-foo", "-bar"}, http.StatusOK, "")
	uuidAppleDDMTeamWithLabel := createAppleDeclaration("apple-team-decl-with-labels", "ident-team-decl-with-labels", testTeam.ID, []string{"-foo"})
	uuidWindowsTeamWithLabel := assertWindowsProfile("win-team-profile-with-labels.xml", "./Test", testTeam.ID, []string{"foo", "bar"}, http.StatusOK, "")
	uuidWindowsTeamWithInclAnyLabel := assertWindowsProfile("win-team-profile-with-incl-any-labels.xml", "./Test", testTeam.ID, []string{"foo", "bar"}, http.StatusOK, "")

	// Windows invalid content
	body, headers := generateNewProfileMultipartRequest(t, "win.xml", []byte("\x00\x01\x02"), s.token, nil)
	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusBadRequest, headers)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't upload. The file should include valid XML:")

	// Apple invalid mobileconfig content
	body, headers = generateNewProfileMultipartRequest(t,
		"apple.mobileconfig", []byte("\x00\x01\x02"), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusBadRequest, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "mobileconfig is not XML nor PKCS7 parseable")

	// Apple invalid json declaration
	body, headers = generateNewProfileMultipartRequest(t,
		"apple.json", []byte("{"), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusBadRequest, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't upload. The file should include valid JSON:")

	// get the existing profiles work
	expectedProfiles := []fleet.MDMConfigProfilePayload{
		{ProfileUUID: noTeamAppleProfUUID, Platform: "darwin", Name: "apple-global-profile", Identifier: "test-global-ident", TeamID: nil},
		{ProfileUUID: teamAppleProfUUID, Platform: "darwin", Name: "apple-team-profile", Identifier: "test-team-ident", TeamID: &testTeam.ID},
		{ProfileUUID: noTeamWinProfUUID, Platform: "windows", Name: "win-global-profile", TeamID: nil},
		{ProfileUUID: teamWinProfUUID, Platform: "windows", Name: "win-team-profile", TeamID: &testTeam.ID},
		{
			ProfileUUID: uuidAppleDDMWithLabel, Platform: "darwin", Name: "apple-decl-with-labels", Identifier: "ident-decl-with-labels", TeamID: nil,
			LabelsIncludeAll: []fleet.ConfigurationProfileLabel{
				{LabelID: labelFoo.ID, LabelName: labelFoo.Name},
			},
		},
		{
			ProfileUUID: uuidAppleWithLabel, Platform: "darwin", Name: "apple-profile-with-labels", Identifier: "ident-with-labels", TeamID: nil,
			LabelsIncludeAll: []fleet.ConfigurationProfileLabel{
				{LabelID: labelFoo.ID, LabelName: labelFoo.Name},
			},
		},
		{
			ProfileUUID: uuidAppleWithInclAnyLabel, Platform: "darwin", Name: "apple-profile-with-incl-any-labels", Identifier: "ident-with-incl-any-labels", TeamID: nil,
			LabelsIncludeAny: []fleet.ConfigurationProfileLabel{
				{LabelID: labelBar.ID, LabelName: labelBar.Name},
				{LabelID: labelFoo.ID, LabelName: labelFoo.Name},
			},
		},
		{
			ProfileUUID: uuidWindowsWithLabel, Platform: "windows", Name: "win-profile-with-labels", TeamID: nil,
			LabelsExcludeAny: []fleet.ConfigurationProfileLabel{
				{LabelID: labelBar.ID, LabelName: labelBar.Name},
				{LabelID: labelFoo.ID, LabelName: labelFoo.Name},
			},
		},
		{
			ProfileUUID: uuidAppleDDMTeamWithLabel, Platform: "darwin", Name: "apple-team-decl-with-labels", Identifier: "ident-team-decl-with-labels", TeamID: &testTeam.ID,
			LabelsExcludeAny: []fleet.ConfigurationProfileLabel{
				{LabelID: labelFoo.ID, LabelName: labelFoo.Name},
			},
		},
		{
			ProfileUUID: uuidWindowsTeamWithLabel, Platform: "windows", Name: "win-team-profile-with-labels", TeamID: &testTeam.ID,
			LabelsIncludeAll: []fleet.ConfigurationProfileLabel{
				{LabelID: labelBar.ID, LabelName: labelBar.Name},
				{LabelID: labelFoo.ID, LabelName: labelFoo.Name},
			},
		},
		{
			ProfileUUID: uuidWindowsTeamWithInclAnyLabel, Platform: "windows", Name: "win-team-profile-with-incl-any-labels", TeamID: &testTeam.ID,
			LabelsIncludeAll: []fleet.ConfigurationProfileLabel{
				{LabelID: labelBar.ID, LabelName: labelBar.Name},
				{LabelID: labelFoo.ID, LabelName: labelFoo.Name},
			},
		},
	}
	for _, prof := range expectedProfiles {
		var getResp getMDMConfigProfileResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", prof.ProfileUUID), nil, http.StatusOK, &getResp)
		require.NotZero(t, getResp.CreatedAt)
		require.NotZero(t, getResp.UploadedAt)
		if getResp.Platform == "darwin" {
			require.Len(t, getResp.Checksum, 16)
		} else {
			require.Empty(t, getResp.Checksum)
		}
		getResp.CreatedAt, getResp.UploadedAt = time.Time{}, time.Time{}
		getResp.Checksum = nil
		// sort the labels by name
		sort.Slice(getResp.LabelsIncludeAll, func(i, j int) bool {
			return getResp.LabelsIncludeAll[i].LabelName < getResp.LabelsIncludeAll[j].LabelName
		})
		sort.Slice(getResp.LabelsExcludeAny, func(i, j int) bool {
			return getResp.LabelsExcludeAny[i].LabelName < getResp.LabelsExcludeAny[j].LabelName
		})
		sort.Slice(getResp.LabelsIncludeAny, func(i, j int) bool {
			return getResp.LabelsIncludeAny[i].LabelName < getResp.LabelsIncludeAny[j].LabelName
		})
		require.Equal(t, prof, *getResp.MDMConfigProfilePayload)

		resp := s.Do("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", prof.ProfileUUID), nil, http.StatusOK, "alt", "media")
		require.NotZero(t, resp.ContentLength)
		require.Contains(t, resp.Header.Get("Content-Disposition"), "attachment;")
		if strings.HasPrefix(prof.ProfileUUID, "a") { //nolint:gocritic // ignore ifElseChain
			require.Contains(t, resp.Header.Get("Content-Type"), "application/x-apple-aspen-config")
		} else if strings.HasPrefix(prof.ProfileUUID, fleet.MDMAppleDeclarationUUIDPrefix) {
			require.Contains(t, resp.Header.Get("Content-Type"), "application/json")
		} else {
			require.Contains(t, resp.Header.Get("Content-Type"), "application/octet-stream")
		}
		require.Contains(t, resp.Header.Get("X-Content-Type-Options"), "nosniff")

		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, resp.ContentLength, int64(len(b)))
	}

	var getResp getMDMConfigProfileResponse
	// get an unknown Apple profile
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", "ano-such-profile"), nil, http.StatusNotFound, &getResp)
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", "ano-such-profile"), nil, http.StatusNotFound, "alt", "media")
	// get an unknown Apple declaration
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", fmt.Sprintf("%sno-such-profile", fleet.MDMAppleDeclarationUUIDPrefix)), nil, http.StatusNotFound, &getResp)
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", fmt.Sprintf("%sno-such-profile", fleet.MDMAppleDeclarationUUIDPrefix)), nil, http.StatusNotFound, "alt", "media")
	// get an unknown Windows profile
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", "wno-such-profile"), nil, http.StatusNotFound, &getResp)
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", "wno-such-profile"), nil, http.StatusNotFound, "alt", "media")

	var deleteResp deleteMDMConfigProfileResponse
	// delete existing Apple profiles
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", noTeamAppleProfUUID), nil, http.StatusOK, &deleteResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", teamAppleProfUUID), nil, http.StatusOK, &deleteResp)
	// delete non-existing Apple profile
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", "ano-such-profile"), nil, http.StatusNotFound, &deleteResp)

	// delete existing Apple declaration
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", uuidAppleDDMWithLabel), nil, http.StatusOK, &deleteResp)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDeletedDeclarationProfile{}.ActivityName(),
		`{"profile_name": "apple-decl-with-labels", "identifier": "ident-decl-with-labels", "team_id": null, "team_name": null}`,
		0,
	)
	// delete non-existing Apple declaration
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", fmt.Sprintf("%sno-such-profile", fleet.MDMAppleDeclarationUUIDPrefix)), nil, http.StatusNotFound, &deleteResp)
	// delete existing Windows profiles
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", noTeamWinProfUUID), nil, http.StatusOK, &deleteResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", teamWinProfUUID), nil, http.StatusOK, &deleteResp)
	// delete non-existing Windows profile
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", "wno-such-profile"), nil, http.StatusNotFound, &deleteResp)

	// trying to create/delete profiles managed by Fleet fails
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		assertAppleProfile("foo.mobileconfig", p, p, 0, nil, http.StatusBadRequest, fmt.Sprintf("payload identifier %s is not allowed", p))

		// create it directly in the DB to test deletion
		uid := "a" + uuid.NewString()
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			mc := mcBytesForTest(p, p, uuid.New().String())
			_, err := q.ExecContext(ctx,
				"INSERT INTO mdm_apple_configuration_profiles (profile_uuid, identifier, name, mobileconfig, checksum, team_id, uploaded_at) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP())",
				uid, p, p, mc, "1234", 0)
			return err
		})

		var deleteResp deleteMDMConfigProfileResponse
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", uid), nil, http.StatusBadRequest, &deleteResp)

		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				"DELETE FROM mdm_apple_configuration_profiles WHERE profile_uuid = ?",
				uid)
			return err
		})
	}
	// TODO: Add tests for create/delete forbidden declaration types?

	// make fleet add a FileVault profile
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	profile := s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// try to delete the profile
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", profile.ProfileUUID), nil, http.StatusBadRequest, &deleteResp)

	// make fleet add a Windows OS Updates profile
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_updates": {"deadline_days": 1, "grace_period_days": 1} }
  }`), http.StatusOK, &acResp)
	profUUID := checkWindowsOSUpdatesProfile(t, s.ds, nil, &fleet.WindowsUpdates{DeadlineDays: optjson.SetInt(1), GracePeriodDays: optjson.SetInt(1)})

	// try to delete the profile
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", profUUID), nil, http.StatusBadRequest, &deleteResp)

	// TODO: Add tests for OS updates declaration when implemented.
}

func (s *integrationMDMTestSuite) TestListMDMConfigProfiles() {
	t := s.T()
	ctx := context.Background()

	// create some teams
	tm1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	tm3, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team3"})
	require.NoError(t, err)

	// create 5 profiles for no team and team 1, names are A, B, C ... for global and
	// tA, tB, tC ... for team 1. Alternate macOS and Windows profiles.
	for i := 0; i < 5; i++ {
		name := string('A' + byte(i))
		if i%2 == 0 {
			prof, err := fleet.NewMDMAppleConfigProfile(mcBytesForTest(name, name+".identifier", name+".uuid"), nil)
			require.NoError(t, err)
			_, err = s.ds.NewMDMAppleConfigProfile(ctx, *prof)
			require.NoError(t, err)

			tprof, err := fleet.NewMDMAppleConfigProfile(mcBytesForTest("t"+name, "t"+name+".identifier", "t"+name+".uuid"), nil)
			require.NoError(t, err)
			tprof.TeamID = &tm1.ID
			_, err = s.ds.NewMDMAppleConfigProfile(ctx, *tprof)
			require.NoError(t, err)
		} else {
			_, err = s.ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: name, SyncML: []byte(`<Replace></Replace>`)})
			require.NoError(t, err)
			_, err = s.ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "t" + name, TeamID: &tm1.ID, SyncML: []byte(`<Replace></Replace>`)})
			require.NoError(t, err)
		}
	}

	lblFoo, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "foo", Query: "select 1"})
	require.NoError(t, err)
	lblBar, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "bar", Query: "select 1"})
	require.NoError(t, err)
	lblBaz, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "baz", Query: "select 1"})
	require.NoError(t, err)

	// create a couple profiles (Win and mac) for team 2, and none for team 3
	tprof, err := fleet.NewMDMAppleConfigProfile(mcBytesForTest("tF", "tF.identifier", "tF.uuid"), nil)
	require.NoError(t, err)
	tprof.TeamID = &tm2.ID
	// make tm2ProfF a "exclude-any" label-based profile
	tprof.LabelsExcludeAny = []fleet.ConfigurationProfileLabel{
		{LabelID: lblFoo.ID, LabelName: lblFoo.Name},
		{LabelID: lblBar.ID, LabelName: lblBar.Name},
	}
	tm2ProfF, err := s.ds.NewMDMAppleConfigProfile(ctx, *tprof)
	require.NoError(t, err)
	// checksum is not returned by New..., so compute it manually
	checkSum := md5.Sum(tm2ProfF.Mobileconfig) // nolint:gosec // used only for test
	tm2ProfF.Checksum = checkSum[:]

	// make tm2ProfG a "include-all" label-based profile
	tm2ProfG, err := s.ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
		Name:   "tG",
		TeamID: &tm2.ID,
		SyncML: []byte(`<Add></Add>`),
		LabelsIncludeAll: []fleet.ConfigurationProfileLabel{
			{LabelID: lblFoo.ID, LabelName: lblFoo.Name},
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
		},
	})
	require.NoError(t, err)

	// make tm2ProfH a "include-any" label-based profile
	tm2ProfH, err := s.ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
		Name:   "tH",
		TeamID: &tm2.ID,
		SyncML: []byte(`<Add></Add>`),
		LabelsIncludeAny: []fleet.ConfigurationProfileLabel{
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
			{LabelID: lblBaz.ID, LabelName: lblBaz.Name},
		},
	})
	require.NoError(t, err)

	// break lblFoo by deleting it
	require.NoError(t, s.ds.DeleteLabel(ctx, lblFoo.Name))

	// test that all fields are correctly returned with team 2
	var listResp listMDMConfigProfilesResponse
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", nil, http.StatusOK, &listResp, "team_id", fmt.Sprint(tm2.ID))
	require.Len(t, listResp.Profiles, 3)
	require.NotZero(t, listResp.Profiles[0].CreatedAt)
	require.NotZero(t, listResp.Profiles[0].UploadedAt)
	require.NotZero(t, listResp.Profiles[1].CreatedAt)
	require.NotZero(t, listResp.Profiles[1].UploadedAt)
	listResp.Profiles[0].CreatedAt, listResp.Profiles[0].UploadedAt = time.Time{}, time.Time{}
	listResp.Profiles[1].CreatedAt, listResp.Profiles[1].UploadedAt = time.Time{}, time.Time{}
	listResp.Profiles[2].CreatedAt, listResp.Profiles[2].UploadedAt = time.Time{}, time.Time{}
	require.Equal(t, &fleet.MDMConfigProfilePayload{
		ProfileUUID: tm2ProfF.ProfileUUID,
		TeamID:      tm2ProfF.TeamID,
		Name:        tm2ProfF.Name,
		Platform:    "darwin",
		Identifier:  tm2ProfF.Identifier,
		Checksum:    tm2ProfF.Checksum,
		// labels are ordered by name
		LabelsExcludeAny: []fleet.ConfigurationProfileLabel{
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
			{LabelID: 0, LabelName: lblFoo.Name, Broken: true},
		},
	}, listResp.Profiles[0])
	require.Equal(t, &fleet.MDMConfigProfilePayload{
		ProfileUUID: tm2ProfG.ProfileUUID,
		TeamID:      tm2ProfG.TeamID,
		Name:        tm2ProfG.Name,
		Platform:    "windows",
		// labels are ordered by name
		LabelsIncludeAll: []fleet.ConfigurationProfileLabel{
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
			{LabelID: 0, LabelName: lblFoo.Name, Broken: true},
		},
	}, listResp.Profiles[1])
	require.Equal(t, &fleet.MDMConfigProfilePayload{
		ProfileUUID: tm2ProfH.ProfileUUID,
		TeamID:      tm2ProfH.TeamID,
		Name:        tm2ProfH.Name,
		Platform:    "windows",
		// labels are ordered by name
		LabelsIncludeAny: []fleet.ConfigurationProfileLabel{
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
			{LabelID: lblBaz.ID, LabelName: lblBaz.Name},
		},
	}, listResp.Profiles[2])

	// get the specific include-all label-based profile returns the information
	var getProfResp getMDMConfigProfileResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles/"+tm2ProfG.ProfileUUID, nil, http.StatusOK, &getProfResp)
	getProfResp.CreatedAt, getProfResp.UploadedAt = time.Time{}, time.Time{}
	require.Equal(t, &fleet.MDMConfigProfilePayload{
		ProfileUUID: tm2ProfG.ProfileUUID,
		TeamID:      tm2ProfG.TeamID,
		Name:        tm2ProfG.Name,
		Platform:    "windows",
		// labels are ordered by name
		LabelsIncludeAll: []fleet.ConfigurationProfileLabel{
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
			{LabelID: 0, LabelName: lblFoo.Name, Broken: true},
		},
	}, getProfResp.MDMConfigProfilePayload)

	// get the specific exclude-any label-based profile returns the information
	getProfResp = getMDMConfigProfileResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles/"+tm2ProfF.ProfileUUID, nil, http.StatusOK, &getProfResp)
	getProfResp.CreatedAt, getProfResp.UploadedAt = time.Time{}, time.Time{}
	require.Equal(t, &fleet.MDMConfigProfilePayload{
		ProfileUUID: tm2ProfF.ProfileUUID,
		TeamID:      tm2ProfF.TeamID,
		Name:        tm2ProfF.Name,
		Platform:    "darwin",
		Identifier:  tm2ProfF.Identifier,
		Checksum:    tm2ProfF.Checksum,
		// labels are ordered by name
		LabelsExcludeAny: []fleet.ConfigurationProfileLabel{
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
			{LabelID: 0, LabelName: lblFoo.Name, Broken: true},
		},
	}, getProfResp.MDMConfigProfilePayload)

	// get the specific include-any label-based profile returns the information
	getProfResp = getMDMConfigProfileResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles/"+tm2ProfH.ProfileUUID, nil, http.StatusOK, &getProfResp)
	getProfResp.CreatedAt, getProfResp.UploadedAt = time.Time{}, time.Time{}
	require.Equal(t, &fleet.MDMConfigProfilePayload{
		ProfileUUID: tm2ProfH.ProfileUUID,
		TeamID:      tm2ProfH.TeamID,
		Name:        tm2ProfH.Name,
		Platform:    "windows",
		// labels are ordered by name
		LabelsIncludeAny: []fleet.ConfigurationProfileLabel{
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
			{LabelID: lblBaz.ID, LabelName: lblBaz.Name},
		},
	}, getProfResp.MDMConfigProfilePayload)
	// list for a non-existing team returns 404
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", nil, http.StatusNotFound, &listResp, "team_id", "99999")

	cases := []struct {
		queries   []string // alternate query name and value
		teamID    *uint
		wantNames []string
		wantMeta  *fleet.PaginationMetadata
	}{
		{
			wantNames: []string{"A", "B", "C", "D", "E"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2"},
			wantNames: []string{"A", "B"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2", "page", "1"},
			wantNames: []string{"C", "D"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "2", "page", "2"},
			wantNames: []string{"E"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "3"},
			teamID:    &tm1.ID,
			wantNames: []string{"tA", "tB", "tC"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "3", "page", "1"},
			teamID:    &tm1.ID,
			wantNames: []string{"tD", "tE"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "3", "page", "2"},
			teamID:    &tm1.ID,
			wantNames: nil,
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "3"},
			teamID:    &tm2.ID,
			wantNames: []string{"tF", "tG", "tH"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2"},
			teamID:    &tm3.ID,
			wantNames: nil,
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%v: %#v", c.teamID, c.queries), func(t *testing.T) {
			var listResp listMDMConfigProfilesResponse
			queryArgs := c.queries
			if c.teamID != nil {
				queryArgs = append(queryArgs, "team_id", fmt.Sprint(*c.teamID))
			}
			s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", nil, http.StatusOK, &listResp, queryArgs...)

			require.Equal(t, len(c.wantNames), len(listResp.Profiles))
			require.Equal(t, c.wantMeta, listResp.Meta)

			var gotNames []string
			if len(listResp.Profiles) > 0 {
				gotNames = make([]string, len(listResp.Profiles))
				for i, p := range listResp.Profiles {
					gotNames[i] = p.Name
					if p.Name == "tG" {
						require.Len(t, p.LabelsIncludeAll, 2)
					} else {
						require.Nil(t, p.LabelsIncludeAll)
					}
					if p.Name == "tF" {
						require.Len(t, p.LabelsExcludeAny, 2)
					} else {
						require.Nil(t, p.LabelsExcludeAny)
					}
					if c.teamID == nil {
						// we set it to 0 for global
						require.NotNil(t, p.TeamID)
						require.Zero(t, *p.TeamID)
					} else {
						require.NotNil(t, p.TeamID)
						require.Equal(t, *c.teamID, *p.TeamID)
					}
					require.NotEmpty(t, p.Platform)
				}
			}
			require.Equal(t, c.wantNames, gotNames)
		})
	}
}

func (s *integrationMDMTestSuite) TestWindowsProfileManagement() {
	t := s.T()
	ctx := context.Background()

	err := s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}})
	require.NoError(t, err)

	globalProfiles := []string{
		mysql.InsertWindowsProfileForTest(t, s.ds, 0),
		mysql.InsertWindowsProfileForTest(t, s.ds, 0),
		mysql.InsertWindowsProfileForTest(t, s.ds, 0),
	}

	// create a new team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "batch_set_mdm_profiles"})
	require.NoError(t, err)
	teamProfiles := []string{
		mysql.InsertWindowsProfileForTest(t, s.ds, tm.ID),
		mysql.InsertWindowsProfileForTest(t, s.ds, tm.ID),
	}

	// create a non-Windows host
	_, err = s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            1,
		OsqueryHostID: ptr.String("non-windows-host"),
		NodeKey:       ptr.String("non-windows-host"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.non.windows", t.Name()),
		Platform:      "darwin",
	})
	require.NoError(t, err)

	// create a Windows host that's not enrolled into MDM
	_, err = s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            2,
		OsqueryHostID: ptr.String("not-mdm-enrolled"),
		NodeKey:       ptr.String("not-mdm-enrolled"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.not.enrolled", t.Name()),
		Platform:      "windows",
	})
	require.NoError(t, err)

	verifyHostProfileStatus := func(cmds []fleet.ProtoCmdOperation, wantStatus string) {
		for _, cmd := range cmds {
			var gotProfile struct {
				Status  string `db:"status"`
				Retries int    `db:"retries"`
			}
			mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				stmt := `
				SELECT COALESCE(status, 'pending') as status, retries
				FROM host_mdm_windows_profiles
				WHERE command_uuid = ?`
				return sqlx.GetContext(context.Background(), q, &gotProfile, stmt, cmd.Cmd.CmdID.Value)
			})

			wantDeliveryStatus := fleet.WindowsResponseToDeliveryStatus(wantStatus)
			if gotProfile.Retries <= servermdm.MaxProfileRetries && wantDeliveryStatus == fleet.MDMDeliveryFailed {
				require.EqualValues(t, "pending", gotProfile.Status, "command_uuid", cmd.Cmd.CmdID.Value)
			} else {
				require.EqualValues(t, wantDeliveryStatus, gotProfile.Status, "command_uuid", cmd.Cmd.CmdID.Value)
			}
		}
	}

	verifyProfiles := func(device *mdmtest.TestWindowsMDMClient, n int, fail bool) {
		mdmResponseStatus := syncml.CmdStatusOK
		if fail {
			mdmResponseStatus = syncml.CmdStatusAtomicFailed
		}
		s.awaitTriggerProfileSchedule(t)
		cmds, err := device.StartManagementSession()
		require.NoError(t, err)
		// 2 Status + n profiles
		require.Len(t, cmds, n+2)

		var atomicCmds []fleet.ProtoCmdOperation
		msgID, err := device.GetCurrentMsgID()
		require.NoError(t, err)
		for _, c := range cmds {
			cmdID := c.Cmd.CmdID
			status := syncml.CmdStatusOK
			if c.Verb == "Atomic" {
				atomicCmds = append(atomicCmds, c)
				status = mdmResponseStatus
				require.NotEmpty(t, c.Cmd.ReplaceCommands)
				for _, rc := range c.Cmd.ReplaceCommands {
					require.NotEmpty(t, rc.CmdID)
				}
			}
			device.AppendResponse(fleet.SyncMLCmd{
				XMLName: xml.Name{Local: fleet.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  &cmdID.Value,
				Cmd:     ptr.String(c.Verb),
				Data:    &status,
				Items:   nil,
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			})
		}
		// TODO: verify profile contents as well
		require.Len(t, atomicCmds, n)

		// before we send the response, commands should be "pending"
		verifyHostProfileStatus(atomicCmds, "")

		cmds, err = device.SendResponse()
		require.NoError(t, err)
		// the ack of the message should be the only returned command
		require.Len(t, cmds, 1)

		// verify that we updated status in the db
		verifyHostProfileStatus(atomicCmds, mdmResponseStatus)
	}

	checkHostsProfilesMatch := func(host *fleet.Host, wantUUIDs []string) {
		var gotUUIDs []string
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			stmt := `SELECT profile_uuid FROM host_mdm_windows_profiles WHERE host_uuid = ?`
			return sqlx.SelectContext(context.Background(), q, &gotUUIDs, stmt, host.UUID)
		})
		require.ElementsMatch(t, wantUUIDs, gotUUIDs)
	}

	checkHostDetails := func(t *testing.T, host *fleet.Host, wantProfs []string, wantStatus fleet.MDMDeliveryStatus) {
		var gotHostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID), nil, http.StatusOK, &gotHostResp)
		require.NotNil(t, gotHostResp.Host.MDM.Profiles)
		var gotProfs []string
		require.Len(t, *gotHostResp.Host.MDM.Profiles, len(wantProfs))
		for _, p := range *gotHostResp.Host.MDM.Profiles {
			gotProfs = append(gotProfs, strings.Replace(p.Name, "name-", "", 1))
			require.NotNil(t, p.Status)
			require.Equal(t, wantStatus, *p.Status, "profile", p.Name)
			require.Equal(t, "windows", p.Platform)
			// Fleet reserved profiles (e.g., OS updates) should be screened from the host details response
			require.NotContains(t, servermdm.ListFleetReservedWindowsProfileNames(), p.Name)
		}
		require.ElementsMatch(t, wantProfs, gotProfs)
	}

	checkHostsFilteredByOSSettingsStatus := func(t *testing.T, wantHosts []string, wantStatus fleet.MDMDeliveryStatus, teamID *uint, labels ...*fleet.Label) {
		var teamFilter string
		if teamID != nil {
			teamFilter = fmt.Sprintf("&team_id=%d", *teamID)
		}
		var gotHostsResp listHostsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts?os_settings=%s%s", wantStatus, teamFilter), nil, http.StatusOK, &gotHostsResp)
		require.NotNil(t, gotHostsResp.Hosts)
		var gotHosts []string
		for _, h := range gotHostsResp.Hosts {
			gotHosts = append(gotHosts, h.Hostname)
		}
		require.ElementsMatch(t, wantHosts, gotHosts)

		var countHostsResp countHostsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/count?os_settings=%s%s", wantStatus, teamFilter), nil, http.StatusOK, &countHostsResp)
		require.Equal(t, len(wantHosts), countHostsResp.Count)

		for _, l := range labels {
			gotHostsResp = listHostsResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/labels/%d/hosts?os_settings=%s%s", l.ID, wantStatus, teamFilter), nil, http.StatusOK, &gotHostsResp)
			require.NotNil(t, gotHostsResp.Hosts)
			gotHosts = []string{}
			for _, h := range gotHostsResp.Hosts {
				gotHosts = append(gotHosts, h.Hostname)
			}
			require.ElementsMatch(t, wantHosts, gotHosts, "label", l.Name)

			countHostsResp = countHostsResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/count?label_id=%d&os_settings=%s%s", l.ID, wantStatus, teamFilter), nil, http.StatusOK, &countHostsResp)
		}
	}

	getProfileUUID := func(t *testing.T, profName string, teamID *uint) string {
		var profUUID string
		mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
			var globalOrTeamID uint
			if teamID != nil {
				globalOrTeamID = *teamID
			}
			return sqlx.GetContext(ctx, tx, &profUUID, `SELECT profile_uuid FROM mdm_windows_configuration_profiles WHERE team_id = ? AND name = ?`, globalOrTeamID, profName)
		})
		require.NotNil(t, profUUID)
		return profUUID
	}

	checkHostProfileStatus := func(t *testing.T, hostUUID string, profUUID string, wantStatus fleet.MDMDeliveryStatus) {
		var gotStatus fleet.MDMDeliveryStatus
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			stmt := `SELECT status FROM host_mdm_windows_profiles WHERE host_uuid = ? AND profile_uuid = ?`
			err := sqlx.GetContext(context.Background(), q, &gotStatus, stmt, hostUUID, profUUID)
			return err
		})
		require.Equal(t, wantStatus, gotStatus)
	}

	// Create a host and then enroll to MDM.
	host, mdmDevice := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	// trigger a profile sync
	verifyProfiles(mdmDevice, 3, false)
	checkHostsProfilesMatch(host, globalProfiles)
	checkHostDetails(t, host, globalProfiles, fleet.MDMDeliveryVerifying)

	// can't resend a profile while it is verifying
	res := s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, globalProfiles[0]), nil, http.StatusConflict)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn’t resend. Configuration profiles with “pending” or “verifying” status can’t be resent.")

	// create new label that includes host
	label := &fleet.Label{
		Name:  t.Name() + "foo",
		Query: "select * from foo;",
	}
	label, err = s.ds.NewLabel(context.Background(), label)
	require.NoError(t, err)
	require.NoError(t, s.ds.RecordLabelQueryExecutions(ctx, host, map[uint]*bool{label.ID: ptr.Bool(true)}, time.Now(), false))

	// simulate osquery reporting host mdm details (host_mdm.enrolled = 1 is condition for
	// hosts filtering by os settings status and generating mdm profiles summaries)
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, host.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, ""))
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryVerifying, nil, label)
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{
		Verifying: 1,
	}, nil)

	// another sync shouldn't return profiles
	verifyProfiles(mdmDevice, 0, false)

	// make fleet add a Windows OS Updates profile
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"mdm": { "windows_updates": {"deadline_days": 1, "grace_period_days": 1} }}`), http.StatusOK, &acResp)
	osUpdatesProf := getProfileUUID(t, servermdm.FleetWindowsOSUpdatesProfileName, nil)

	// os updates is sent via a profiles commands
	verifyProfiles(mdmDevice, 1, false)
	checkHostsProfilesMatch(host, append(globalProfiles, osUpdatesProf))
	// but is hidden from host details response
	checkHostDetails(t, host, globalProfiles, fleet.MDMDeliveryVerifying)

	// os updates profile status doesn't matter for filtered hosts results or summaries
	checkHostProfileStatus(t, host.UUID, osUpdatesProf, fleet.MDMDeliveryVerifying)
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryVerifying, nil, label)
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{
		Verifying: 1,
	}, nil)
	// force os updates profile to failed, doesn't impact filtered hosts results or summaries
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_windows_profiles SET status = 'failed' WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, osUpdatesProf)
		return err
	})
	checkHostProfileStatus(t, host.UUID, osUpdatesProf, fleet.MDMDeliveryFailed)
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryVerifying, nil, label)
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{
		Verifying: 1,
	}, nil)
	// force another profile to failed, does impact filtered hosts results and summaries
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_windows_profiles SET status = 'failed' WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, globalProfiles[0])
		return err
	})
	checkHostProfileStatus(t, host.UUID, globalProfiles[0], fleet.MDMDeliveryFailed)
	checkHostsFilteredByOSSettingsStatus(t, []string{}, fleet.MDMDeliveryVerifying, nil, label)           // expect no hosts
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryFailed, nil, label) // expect host
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{
		Failed:    1,
		Verifying: 0,
	}, nil)

	// can resend a profile after it has failed
	// purposefully using deprecated path for backwards compatibility
	_ = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/resend/%s", host.ID, globalProfiles[0]), nil,
		http.StatusAccepted)
	verifyProfiles(mdmDevice, 1, false)                                                 // trigger a profile sync, device gets the profile resent
	checkHostProfileStatus(t, host.UUID, globalProfiles[0], fleet.MDMDeliveryVerifying) // profile was resent, so it back to verifying

	// add the host to a team
	err = s.ds.AddHostsToTeam(ctx, &tm.ID, []uint{host.ID})
	require.NoError(t, err)

	// trigger a profile sync, device gets the team profile
	verifyProfiles(mdmDevice, 2, false)
	checkHostsProfilesMatch(host, teamProfiles)
	checkHostDetails(t, host, teamProfiles, fleet.MDMDeliveryVerifying)

	// set new team profiles (delete + addition)
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `DELETE FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, teamProfiles[1])
		return err
	})
	teamProfiles = []string{
		teamProfiles[0],
		mysql.InsertWindowsProfileForTest(t, s.ds, tm.ID),
	}

	// trigger a profile sync, device gets the team profile
	verifyProfiles(mdmDevice, 1, false)

	// check that we deleted the old profile in the DB
	checkHostsProfilesMatch(host, teamProfiles)
	checkHostDetails(t, host, teamProfiles, fleet.MDMDeliveryVerifying)

	// can't resend a profile while it is verifying
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, teamProfiles[0]), nil, http.StatusConflict)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn’t resend. Configuration profiles with “pending” or “verifying” status can’t be resent.")

	// can't resend a profile from the wrong team
	// purposefully using deprecated path for backwards compatibility
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/resend/%s", host.ID, globalProfiles[0]), nil, http.StatusNotFound)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Unable to match profile to host.")

	// another sync shouldn't return profiles
	verifyProfiles(mdmDevice, 0, false)

	// set new team profiles (delete + addition)
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `DELETE FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, teamProfiles[1])
		return err
	})
	teamProfiles = []string{
		teamProfiles[0],
		mysql.InsertWindowsProfileForTest(t, s.ds, tm.ID),
	}
	// trigger a profile sync, this time fail the delivery
	verifyProfiles(mdmDevice, 1, true)

	// check that we deleted the old profile in the DB
	checkHostsProfilesMatch(host, teamProfiles)

	// a second sync gets the profile again, because of delivery retries.
	// Succeed that one
	verifyProfiles(mdmDevice, 1, false)

	// another sync shouldn't return profiles
	verifyProfiles(mdmDevice, 0, false)

	// make fleet add a Windows OS Updates profile
	tmResp := teamResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm.ID), json.RawMessage(`{"mdm": { "windows_updates": {"deadline_days": 1, "grace_period_days": 1} }}`), http.StatusOK, &tmResp)
	osUpdatesProf = getProfileUUID(t, servermdm.FleetWindowsOSUpdatesProfileName, &tm.ID)

	// os updates is sent via a profiles commands
	verifyProfiles(mdmDevice, 1, false)
	checkHostsProfilesMatch(host, append(teamProfiles, osUpdatesProf))
	// but is hidden from host details response
	checkHostDetails(t, host, teamProfiles, fleet.MDMDeliveryVerifying)

	// os updates profile status doesn't matter for filtered hosts results or summaries
	checkHostProfileStatus(t, host.UUID, osUpdatesProf, fleet.MDMDeliveryVerifying)
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryVerifying, &tm.ID, label)
	s.checkMDMProfilesSummaries(t, &tm.ID, fleet.MDMProfilesSummary{
		Verifying: 1,
	}, nil)
	// force os updates profile to failed, doesn't impact filtered hosts results or summaries
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_windows_profiles SET status = 'failed' WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, osUpdatesProf)
		return err
	})
	checkHostProfileStatus(t, host.UUID, osUpdatesProf, fleet.MDMDeliveryFailed)
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryVerifying, &tm.ID, label)
	s.checkMDMProfilesSummaries(t, &tm.ID, fleet.MDMProfilesSummary{
		Verifying: 1,
	}, nil)
	// force another profile to failed, does impact filtered hosts results and summaries
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_windows_profiles SET status = 'failed' WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, teamProfiles[0])
		return err
	})
	checkHostProfileStatus(t, host.UUID, teamProfiles[0], fleet.MDMDeliveryFailed)
	checkHostsFilteredByOSSettingsStatus(t, []string{}, fleet.MDMDeliveryVerifying, &tm.ID, label)           // expect no hosts
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryFailed, &tm.ID, label) // expect host
	s.checkMDMProfilesSummaries(t, &tm.ID, fleet.MDMProfilesSummary{
		Failed:    1,
		Verifying: 0,
	}, nil)

	// can resend a profile after it has failed
	_ = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, teamProfiles[0]), nil,
		http.StatusAccepted)
	verifyProfiles(mdmDevice, 1, false)                                               // trigger a profile sync, device gets the profile resent
	checkHostProfileStatus(t, host.UUID, teamProfiles[0], fleet.MDMDeliveryVerifying) // profile was resent, so back to verifying
	s.lastActivityMatches(
		fleet.ActivityTypeResentConfigurationProfile{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "profile_name": %q}`, host.ID, host.DisplayName(), "name-"+teamProfiles[0]),
		0)

	// add a macOS profile to the team
	mcUUID := "a" + uuid.NewString()
	prof := mcBytesForTest("name-"+mcUUID, "idenfifer-"+mcUUID, mcUUID)
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `INSERT INTO mdm_apple_configuration_profiles (profile_uuid, team_id, name, identifier, mobileconfig, checksum, uploaded_at) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);`
		_, err := q.ExecContext(context.Background(), stmt, mcUUID, tm.ID, "name-"+mcUUID, "identifier-"+mcUUID, prof, []byte("checksum-"+mcUUID))
		return err
	})

	// trigger a profile sync, device doesn't get the macOS profile
	verifyProfiles(mdmDevice, 0, false)

	// can't resend a macOS profile to a Windows host
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, mcUUID), nil, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Profile is not compatible with host platform")
}

func (s *integrationMDMTestSuite) TestApplyTeamsMDMWindowsProfiles() {
	t := s.T()

	// create a team through the service so it initializes the agent ops
	teamName := t.Name() + "team1"
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc team1",
	}
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	team = createTeamResp.Team

	rawTeamSpec := func(mdmValue string) json.RawMessage {
		return json.RawMessage(fmt.Sprintf(`{ "specs": [{ "name": %q, "mdm": %s }] }`, team.Name, mdmValue))
	}

	// set the windows custom settings fields
	var applyResp applyTeamSpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", rawTeamSpec(`
		{
			"windows_settings": {
				"custom_settings": [
					{"path": "foo", "labels": ["baz"]},
					{"path": "bar", "labels_exclude_any": ["x", "y"]}
				]
			}
		}
	`), http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	// check that they are returned by a GET /config
	var teamResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.ElementsMatch(t, []fleet.MDMProfileSpec{
		{Path: "foo", LabelsIncludeAll: []string{"baz"}},
		{Path: "bar", LabelsExcludeAny: []string{"x", "y"}},
	}, teamResp.Team.Config.MDM.WindowsSettings.CustomSettings.Value)

	// patch without specifying the windows custom settings fields and an unrelated
	// field, should not remove them
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", rawTeamSpec(`{ "enable_disk_encryption": true }`), http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	// check that they are returned by a GET /config
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.ElementsMatch(t, []fleet.MDMProfileSpec{
		{Path: "foo", LabelsIncludeAll: []string{"baz"}},
		{Path: "bar", LabelsExcludeAny: []string{"x", "y"}},
	}, teamResp.Team.Config.MDM.WindowsSettings.CustomSettings.Value)

	// patch with explicitly empty windows custom settings fields, would remove
	// them but this is a dry-run
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", rawTeamSpec(`
		{ "windows_settings": { "custom_settings": null } }
  `), http.StatusOK, &applyResp, "dry_run", "true")
	assert.Equal(t, map[string]uint{team.Name: team.ID}, applyResp.TeamIDsByName)

	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.ElementsMatch(t, []fleet.MDMProfileSpec{
		{Path: "foo", LabelsIncludeAll: []string{"baz"}},
		{Path: "bar", LabelsExcludeAny: []string{"x", "y"}},
	}, teamResp.Team.Config.MDM.WindowsSettings.CustomSettings.Value)

	// patch with explicitly empty windows custom settings fields, removes them
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", rawTeamSpec(`
		{ "windows_settings": { "custom_settings": null } }
  `), http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Empty(t, teamResp.Team.Config.MDM.WindowsSettings.CustomSettings.Value)

	// apply with invalid mix of labels fails
	res := s.Do("POST", "/api/latest/fleet/spec/teams", rawTeamSpec(`
		{
			"windows_settings": {
				"custom_settings": [
					{"path": "foo", "labels": ["a"], "labels_include_all": ["b"]}
				]
			}
		}
	`), http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, `For each profile, only one of "labels_exclude_any", "labels_include_all", "labels_include_any" or "labels" can be included.`)
}

func (s *integrationMDMTestSuite) TestBatchSetMDMProfiles() {
	t := s.T()
	ctx := context.Background()

	// create a new team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "batch_set_mdm_profiles"})
	require.NoError(t, err)

	bigString := strings.Repeat("a", 1024*1024+1)

	// Profile is too big
	resp := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{{Contents: []byte(bigString)}}},
		http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(resp.Body), "Validation Failed: maximum configuration profile file size is 1 MB")

	// apply an empty set to no-team
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: nil}, http.StatusNoContent)
	// Nothing changed, so no activity items
	s.lastActivityOfTypeDoesNotMatch(
		fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null}`,
		0,
	)
	s.lastActivityOfTypeDoesNotMatch(
		fleet.ActivityTypeEditedWindowsProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null}`,
		0,
	)
	s.lastActivityOfTypeDoesNotMatch(
		fleet.ActivityTypeEditedDeclarationProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null}`,
		0,
	)

	// apply to both team id and name
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: nil},
		http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID), "team_name", tm.Name)

	// invalid team name
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: nil},
		http.StatusNotFound, "team_name", uuid.New().String())

	// duplicate PayloadDisplayName
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: "N2", Contents: mobileconfigForTest("N1", "I2")},
		{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
		{Name: "N4", Contents: declarationForTest("D1")},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))

	// profiles with reserved macOS identifiers
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
			{Name: p, Contents: mobileconfigForTest(p, p)},
			{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
			{Name: "N4", Contents: declarationForTest("D1")},
		}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: payload identifier %s is not allowed", p))
	}

	// payloads with reserved types
	for p := range mobileconfig.FleetPayloadTypes() {
		res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "N1", Contents: mobileconfigForTestWithContent("N1", "I1", "II1", p, "")},
			{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
			{Name: "N4", Contents: declarationForTest("D1")},
		}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadType(s): %s", p))
	}

	// payloads with reserved identifiers
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "N1", Contents: mobileconfigForTestWithContent("N1", "I1", p, "random", "")},
			{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
			{Name: "N4", Contents: declarationForTest("D1")},
		}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadIdentifier(s): %s", p))
	}

	// profiles with forbidden declaration types
	for dt := range fleet.ForbiddenDeclTypes {
		res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
			{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
			{Name: "N4", Contents: declarationForTestWithType("D1", dt)},
		}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, "Only configuration declarations that don’t require an asset reference are supported", dt)
	}
	// and one more for the software update declaration
	res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
		{Name: "N4", Contents: declarationForTestWithType("D1", "com.apple.configuration.softwareupdate.enforcement.specific")},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Declaration profile can’t include OS updates settings. To control these settings, go to OS updates.")

	// invalid JSON
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
		{Name: "N4", Contents: []byte(`{"foo":}`)},
	}}, http.StatusBadRequest, "team_id", fmt.Sprint(tm.ID))
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "The file should include valid JSON")

	// profiles with reserved Windows location URIs
	// bitlocker
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: syncml.FleetBitLockerTargetLocURI, Contents: syncMLForTest(fmt.Sprintf("%s/Foo", syncml.FleetBitLockerTargetLocURI))},
		{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Custom configuration profiles can't include BitLocker settings. To control these settings, use the mdm.enable_disk_encryption option.")

	// os updates
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: syncml.FleetOSUpdateTargetLocURI, Contents: syncMLForTest(fmt.Sprintf("%s/Foo", syncml.FleetOSUpdateTargetLocURI))},
		{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Custom configuration profiles can't include Windows updates settings. To control these settings, use the mdm.windows_updates option.")

	// invalid windows tag
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N3", Contents: []byte(`<Exec></Exec>`)},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Windows configuration profiles can only have <Replace> or <Add> top level elements.")

	// invalid xml
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N3", Contents: []byte(`foo`)},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Windows configuration profiles can only have <Replace> or <Add> top level elements.")

	// successfully apply windows and macOS a profiles for the team, but it's a dry run
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: "N2", Contents: syncMLForTest("./Foo/Bar")},
		{Name: "N4", Contents: declarationForTest("D1")},
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID), "dry_run", "true")
	s.assertConfigProfilesByIdentifier(&tm.ID, "I1", false)
	s.assertWindowsConfigProfilesByName(&tm.ID, "N1", false)

	// successfully apply for a team and verify activities
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: "N2", Contents: syncMLForTest("./Foo/Bar")},
		{Name: "N4", Contents: declarationForTest("D1")},
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))
	s.assertConfigProfilesByIdentifier(&tm.ID, "I1", true)
	s.assertWindowsConfigProfilesByName(&tm.ID, "N2", true)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, tm.ID, tm.Name),
		0,
	)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedWindowsProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, tm.ID, tm.Name),
		0,
	)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedDeclarationProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, tm.ID, tm.Name),
		0,
	)

	// batch-apply profiles with labels
	lbl1, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "L1", Query: "select 1;"})
	require.NoError(t, err)
	lbl2, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "L2", Query: "select 1;"})
	require.NoError(t, err)
	lbl3, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "L3", Query: "select 1;"})
	require.NoError(t, err)

	// attempt with an invalid label name
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1"), Labels: []string{lbl1.Name, "no-such-label"}},
	}}, http.StatusBadRequest)
	msg := extractServerErrorText(res.Body)
	require.Contains(t, msg, "some or all the labels provided don't exist")

	// mix of labels fields
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1"), Labels: []string{lbl1.Name}, LabelsExcludeAny: []string{lbl2.Name}},
	}}, http.StatusUnprocessableEntity)
	msg = extractServerErrorText(res.Body)
	require.Contains(t, msg, `For each profile, only one of "labels_exclude_any", "labels_include_all", "labels_include_any" or "labels" can be included.`)

	// successful batch-set
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1"), Labels: []string{lbl1.Name, lbl2.Name}},
		{Name: "N2", Contents: syncMLForTest("./Foo/Bar"), LabelsIncludeAll: []string{lbl1.Name}},
		{Name: "N4", Contents: declarationForTest("D1"), LabelsExcludeAny: []string{lbl2.Name}},
	}}, http.StatusNoContent)

	// confirm expected results
	var listResp listMDMConfigProfilesResponse
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Profiles, 3)
	require.Equal(t, "N1", listResp.Profiles[0].Name)
	require.Equal(t, "N2", listResp.Profiles[1].Name)
	require.Equal(t, "N4", listResp.Profiles[2].Name)
	require.Equal(t, listResp.Profiles[0].LabelsIncludeAll, []fleet.ConfigurationProfileLabel{
		{LabelID: lbl1.ID, LabelName: lbl1.Name},
		{LabelID: lbl2.ID, LabelName: lbl2.Name},
	})
	require.Nil(t, listResp.Profiles[0].LabelsExcludeAny)
	require.Equal(t, listResp.Profiles[1].LabelsIncludeAll, []fleet.ConfigurationProfileLabel{
		{LabelID: lbl1.ID, LabelName: lbl1.Name},
	})
	require.Nil(t, listResp.Profiles[1].LabelsExcludeAny)
	require.Equal(t, listResp.Profiles[2].LabelsExcludeAny, []fleet.ConfigurationProfileLabel{
		{LabelID: lbl2.ID, LabelName: lbl2.Name},
	})
	require.Nil(t, listResp.Profiles[2].LabelsIncludeAll)

	// successful batch-set that updates some labels
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1"), LabelsExcludeAny: []string{lbl1.Name, lbl3.Name}},
		{Name: "N2", Contents: syncMLForTest("./Foo/Bar"), LabelsIncludeAll: []string{lbl2.Name}},
	}}, http.StatusNoContent)

	listResp = listMDMConfigProfilesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Profiles, 2)
	require.Equal(t, "N1", listResp.Profiles[0].Name)
	require.Equal(t, "N2", listResp.Profiles[1].Name)
	require.Equal(t, listResp.Profiles[0].LabelsExcludeAny, []fleet.ConfigurationProfileLabel{
		{LabelID: lbl1.ID, LabelName: lbl1.Name},
		{LabelID: lbl3.ID, LabelName: lbl3.Name},
	})
	require.Nil(t, listResp.Profiles[0].LabelsIncludeAll)
	require.Equal(t, listResp.Profiles[1].LabelsIncludeAll, []fleet.ConfigurationProfileLabel{
		{LabelID: lbl2.ID, LabelName: lbl2.Name},
	})
	require.Nil(t, listResp.Profiles[1].LabelsExcludeAny)

	// names cannot be duplicated across platforms
	declBytes := json.RawMessage(`{
	"Type": "com.apple.configuration.decl.foo",
	"Identifier": "com.fleet.config.foo",
	"Payload": {
		"ServiceType": "com.apple.bash",
		"DataAssetReference": "com.fleet.asset.bash"
	}}`)
	mcBytes := mobileconfigForTest("N1", "I1")
	winBytes := syncMLForTest("./Foo/Bar")

	for _, p := range []struct {
		payload   []fleet.MDMProfileBatchPayload
		expectErr string
	}{
		{
			payload:   []fleet.MDMProfileBatchPayload{{Name: "N1", Contents: mcBytes}, {Name: "N1", Contents: winBytes}},
			expectErr: "More than one configuration profile have the same name 'N1' (Windows .xml file name or macOS .mobileconfig PayloadDisplayName).",
		},
		{
			payload:   []fleet.MDMProfileBatchPayload{{Name: "N1", Contents: declBytes}, {Name: "N1", Contents: winBytes}},
			expectErr: "More than one configuration profile have the same name 'N1' (macOS .json file name or Windows .xml file name).",
		},
		{
			payload:   []fleet.MDMProfileBatchPayload{{Name: "N1", Contents: mcBytes}, {Name: "N1", Contents: declBytes}},
			expectErr: "More than one configuration profile have the same name 'N1' (macOS .json file name or macOS .mobileconfig PayloadDisplayName).",
		},
	} {
		// team profiles
		res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: p.payload},
			http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, p.expectErr)
		// no team profiles
		res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: p.payload}, http.StatusUnprocessableEntity)
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, p.expectErr)
	}
}

func (s *integrationMDMTestSuite) TestBatchSetMDMProfilesBackwardsCompat() {
	t := s.T()
	ctx := context.Background()

	// create a new team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "batch_set_mdm_profiles"})
	require.NoError(t, err)

	// apply an empty set to no-team
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": nil}, http.StatusNoContent)
	// Nothing changed, so no activity
	s.lastActivityOfTypeDoesNotMatch(
		fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null}`,
		0,
	)
	s.lastActivityOfTypeDoesNotMatch(
		fleet.ActivityTypeEditedWindowsProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null}`,
		0,
	)

	// apply to both team id and name
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": nil},
		http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID), "team_name", tm.Name)

	// invalid team name
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": nil},
		http.StatusNotFound, "team_name", uuid.New().String())

	// duplicate PayloadDisplayName
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
		"N1": mobileconfigForTest("N1", "I1"),
		"N2": mobileconfigForTest("N1", "I2"),
		"N3": syncMLForTest("./Foo/Bar"),
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))

	// profiles with reserved macOS identifiers
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
			"N1": mobileconfigForTest("N1", "I1"),
			p:    mobileconfigForTest(p, p),
			"N3": syncMLForTest("./Foo/Bar"),
		}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: payload identifier %s is not allowed", p))
	}

	// payloads with reserved types
	for p := range mobileconfig.FleetPayloadTypes() {
		res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
			"N1": mobileconfigForTestWithContent("N1", "I1", "II1", p, ""),
			"N3": syncMLForTest("./Foo/Bar"),
		}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadType(s): %s", p))
	}

	// payloads with reserved identifiers
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
			"N1": mobileconfigForTestWithContent("N1", "I1", p, "random", ""),
			"N3": syncMLForTest("./Foo/Bar"),
		}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadIdentifier(s): %s", p))
	}

	// profiles with reserved Windows location URIs
	// bitlocker
	res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
		"N1":                              mobileconfigForTest("N1", "I1"),
		syncml.FleetBitLockerTargetLocURI: syncMLForTest(fmt.Sprintf("%s/Foo", syncml.FleetBitLockerTargetLocURI)),
		"N3":                              syncMLForTest("./Foo/Bar"),
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Custom configuration profiles can't include BitLocker settings. To control these settings, use the mdm.enable_disk_encryption option.")

	// os updates
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
		"N1":                             mobileconfigForTest("N1", "I1"),
		syncml.FleetOSUpdateTargetLocURI: syncMLForTest(fmt.Sprintf("%s/Foo", syncml.FleetOSUpdateTargetLocURI)),
		"N3":                             syncMLForTest("./Foo/Bar"),
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Custom configuration profiles can't include Windows updates settings. To control these settings, use the mdm.windows_updates option.")

	// invalid windows tag
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
		"N3": []byte(`<Exec></Exec>`),
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Windows configuration profiles can only have <Replace> or <Add> top level elements.")

	// invalid xml
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
		"N3": []byte(`foo`),
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Windows configuration profiles can only have <Replace> or <Add> top level elements.")

	// successfully apply windows and macOS a profiles for the team, but it's a dry run
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
		"N1": mobileconfigForTest("N1", "I1"),
		"N2": syncMLForTest("./Foo/Bar"),
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID), "dry_run", "true")
	s.assertConfigProfilesByIdentifier(&tm.ID, "I1", false)
	s.assertWindowsConfigProfilesByName(&tm.ID, "N1", false)

	// successfully apply for a team and verify activities
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
		"N1": mobileconfigForTest("N1", "I1"),
		"N2": syncMLForTest("./Foo/Bar"),
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))
	s.assertConfigProfilesByIdentifier(&tm.ID, "I1", true)
	s.assertWindowsConfigProfilesByName(&tm.ID, "N2", true)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, tm.ID, tm.Name),
		0,
	)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedWindowsProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, tm.ID, tm.Name),
		0,
	)
}

func (s *integrationMDMTestSuite) TestMDMBatchSetProfilesKeepsReservedNames() {
	t := s.T()
	ctx := context.Background()

	checkMacProfs := func(teamID *uint, names ...string) {
		var count int
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			var tid uint
			if teamID != nil {
				tid = *teamID
			}
			return sqlx.GetContext(ctx, q, &count, `SELECT COUNT(*) FROM mdm_apple_configuration_profiles WHERE team_id = ?`, tid)
		})
		require.Equal(t, len(names), count)
		for _, n := range names {
			s.assertMacOSConfigProfilesByName(teamID, n, true)
		}
	}

	checkWinProfs := func(teamID *uint, names ...string) {
		var count int
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			var tid uint
			if teamID != nil {
				tid = *teamID
			}
			return sqlx.GetContext(ctx, q, &count, `SELECT COUNT(*) FROM mdm_windows_configuration_profiles WHERE team_id = ?`, tid)
		})
		for _, n := range names {
			s.assertWindowsConfigProfilesByName(teamID, n, true)
		}
	}

	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.True(t, acResp.MDM.EnabledAndConfigured)
	require.True(t, acResp.MDM.WindowsEnabledAndConfigured)

	// ensures that the fleetd profile is created
	secrets, err := s.ds.GetEnrollSecrets(ctx, nil)
	require.NoError(t, err)
	if len(secrets) == 0 {
		require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}}))
	}
	require.NoError(t, ReconcileAppleProfiles(ctx, s.ds, s.mdmCommander, s.logger))

	// turn on disk encryption and os updates
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"enable_disk_encryption": true,
			"windows_updates": {
				"deadline_days": 3,
				"grace_period_days": 1
			},
			"macos_updates": {
				"deadline": "2023-12-31",
				"minimum_version": "13.3.7"
			}
		}
	}`), http.StatusOK, &acResp)
	checkMacProfs(nil, servermdm.ListFleetReservedMacOSProfileNames()...)
	checkWinProfs(nil, servermdm.ListFleetReservedWindowsProfileNames()...)

	// batch set only windows profiles doesn't remove the reserved names
	newWinProfile := syncml.ForTestWithData(map[string]string{"l1": "d1"})
	var testProfiles []fleet.MDMProfileBatchPayload
	testProfiles = append(testProfiles, fleet.MDMProfileBatchPayload{
		Name:     "n1",
		Contents: newWinProfile,
	})
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
	checkMacProfs(nil, servermdm.ListFleetReservedMacOSProfileNames()...)
	checkWinProfs(nil, append(servermdm.ListFleetReservedWindowsProfileNames(), "n1")...)

	// batch set windows and mac profiles doesn't remove the reserved names
	newMacProfile := mcBytesForTest("n2", "i2", uuid.NewString())
	testProfiles = append(testProfiles, fleet.MDMProfileBatchPayload{
		Name:     "n2",
		Contents: newMacProfile,
	})
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
	checkMacProfs(nil, append(servermdm.ListFleetReservedMacOSProfileNames(), "n2")...)
	checkWinProfs(nil, append(servermdm.ListFleetReservedWindowsProfileNames(), "n1")...)

	// batch set only mac profiles doesn't remove the reserved names
	testProfiles = []fleet.MDMProfileBatchPayload{{
		Name:     "n2",
		Contents: newMacProfile,
	}}
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
	checkMacProfs(nil, append(servermdm.ListFleetReservedMacOSProfileNames(), "n2")...)
	checkWinProfs(nil, servermdm.ListFleetReservedWindowsProfileNames()...)

	// create a team
	var tmResp teamResponse
	s.DoJSON("POST", "/api/v1/fleet/teams", map[string]string{"Name": t.Name()}, http.StatusOK, &tmResp)

	// edit team mdm config to turn on disk encryption and os updates
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tmResp.Team.ID), modifyTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: ptr.String(t.Name()),
			MDM: &fleet.TeamPayloadMDM{
				EnableDiskEncryption: optjson.SetBool(true),
				WindowsUpdates: &fleet.WindowsUpdates{
					DeadlineDays:    optjson.SetInt(4),
					GracePeriodDays: optjson.SetInt(1),
				},
				MacOSUpdates: &fleet.AppleOSUpdateSettings{
					Deadline:       optjson.SetString("2023-12-31"),
					MinimumVersion: optjson.SetString("13.3.8"),
				},
			},
		},
	}, http.StatusOK, &teamResponse{})

	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/teams/%d", tmResp.Team.ID), nil, http.StatusOK, &tmResp)
	require.True(t, tmResp.Team.Config.MDM.EnableDiskEncryption)
	require.Equal(t, 4, tmResp.Team.Config.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 1, tmResp.Team.Config.MDM.WindowsUpdates.GracePeriodDays.Value)
	require.Equal(t, "2023-12-31", tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	require.Equal(t, "13.3.8", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)

	require.NoError(t, ReconcileAppleProfiles(ctx, s.ds, s.mdmCommander, s.logger))

	checkMacProfs(&tmResp.Team.ID, servermdm.ListFleetReservedMacOSProfileNames()...)
	checkWinProfs(&tmResp.Team.ID, servermdm.ListFleetReservedWindowsProfileNames()...)

	// batch set only windows profiles doesn't remove the reserved names
	var testTeamProfiles []fleet.MDMProfileBatchPayload
	testTeamProfiles = append(testTeamProfiles, fleet.MDMProfileBatchPayload{
		Name:     "n1",
		Contents: newWinProfile,
	})
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testTeamProfiles}, http.StatusNoContent,
		"team_id", fmt.Sprint(tmResp.Team.ID))
	checkMacProfs(&tmResp.Team.ID, servermdm.ListFleetReservedMacOSProfileNames()...)
	checkWinProfs(&tmResp.Team.ID, append(servermdm.ListFleetReservedWindowsProfileNames(), "n1")...)

	// batch set windows and mac profiles doesn't remove the reserved names
	testTeamProfiles = append(testTeamProfiles, fleet.MDMProfileBatchPayload{
		Name:     "n2",
		Contents: newMacProfile,
	})
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testTeamProfiles}, http.StatusNoContent,
		"team_id", fmt.Sprint(tmResp.Team.ID))
	checkMacProfs(&tmResp.Team.ID, append(servermdm.ListFleetReservedMacOSProfileNames(), "n2")...)
	checkWinProfs(&tmResp.Team.ID, append(servermdm.ListFleetReservedWindowsProfileNames(), "n1")...)

	// batch set only mac profiles doesn't remove the reserved names
	testTeamProfiles = []fleet.MDMProfileBatchPayload{{
		Name:     "n2",
		Contents: newMacProfile,
	}}
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testTeamProfiles}, http.StatusNoContent,
		"team_id", fmt.Sprint(tmResp.Team.ID))
	checkMacProfs(&tmResp.Team.ID, append(servermdm.ListFleetReservedMacOSProfileNames(), "n2")...)
	checkWinProfs(&tmResp.Team.ID, servermdm.ListFleetReservedWindowsProfileNames()...)
}

func (s *integrationMDMTestSuite) TestMDMAppleConfigProfileCRUD() {
	t := s.T()
	ctx := context.Background()

	testTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "TestTeam"})
	require.NoError(t, err)

	teamDelete, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "TeamDelete"})
	require.NoError(t, err)

	testProfiles := make(map[string]fleet.MDMAppleConfigProfile)
	generateTestProfile := func(name string, identifier string) {
		i := identifier
		if i == "" {
			i = fmt.Sprintf("%s.SomeIdentifier", name)
		}
		cp := fleet.MDMAppleConfigProfile{
			Name:       name,
			Identifier: i,
		}
		cp.Mobileconfig = mcBytesForTest(cp.Name, cp.Identifier, fmt.Sprintf("%s.UUID", name))
		testProfiles[name] = cp
	}
	setTestProfileID := func(name string, id uint) {
		tp := testProfiles[name]
		tp.ProfileID = id
		testProfiles[name] = tp
	}

	generateNewReq := func(name string, teamID *uint) (*bytes.Buffer, map[string]string) {
		args := map[string][]string{}
		if teamID != nil {
			args["team_id"] = []string{fmt.Sprintf("%d", *teamID)}
		}
		return generateNewProfileMultipartRequest(t, "some_filename", testProfiles[name].Mobileconfig, s.token, args)
	}

	checkGetResponse := func(resp *http.Response, expected fleet.MDMAppleConfigProfile) {
		// check expected headers
		require.Contains(t, resp.Header["Content-Type"], "application/x-apple-aspen-config")
		require.Contains(t, resp.Header["Content-Disposition"], fmt.Sprintf(`attachment;filename="%s_%s.%s"`, time.Now().Format("2006-01-02"), strings.ReplaceAll(expected.Name, " ", "_"), "mobileconfig"))
		// check expected body
		var bb bytes.Buffer
		_, err = io.Copy(&bb, resp.Body)
		require.NoError(t, err)
		require.Equal(t, []byte(expected.Mobileconfig), bb.Bytes())
	}

	checkConfigProfile := func(expected fleet.MDMAppleConfigProfile, actual fleet.MDMAppleConfigProfile) {
		require.Equal(t, expected.Name, actual.Name)
		require.Equal(t, expected.Identifier, actual.Identifier)
	}

	host, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.Do("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &teamDelete.ID,
		HostIDs: []uint{host.ID},
	}, http.StatusOK)

	// create new profile (no team)
	generateTestProfile("TestNoTeam", "")
	body, headers := generateNewReq("TestNoTeam", nil)
	newResp := s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusOK, headers)
	var newCP fleet.MDMAppleConfigProfile
	err = json.NewDecoder(newResp.Body).Decode(&newCP)
	require.NoError(t, err)
	require.NotEmpty(t, newCP.ProfileID)
	setTestProfileID("TestNoTeam", newCP.ProfileID)

	// create new profile (with team id)
	generateTestProfile("TestWithTeamID", "")
	body, headers = generateNewReq("TestWithTeamID", &testTeam.ID)
	newResp = s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(newResp.Body).Decode(&newCP)
	require.NoError(t, err)
	require.NotEmpty(t, newCP.ProfileID)
	setTestProfileID("TestWithTeamID", newCP.ProfileID)

	// Create a profile that we're going to remove immediately
	generateTestProfile("TestImmediateDelete", "")
	body, headers = generateNewReq("TestImmediateDelete", &teamDelete.ID)
	newResp = s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusOK, headers)
	newCP = fleet.MDMAppleConfigProfile{}
	err = json.NewDecoder(newResp.Body).Decode(&newCP)
	require.NoError(t, err)
	require.NotEmpty(t, newCP.ProfileID)
	setTestProfileID("TestImmediateDelete", newCP.ProfileID)

	// check that host_mdm_apple_profiles entry was created
	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.NotNil(t, hostResp.Host.MDM.Profiles)
	require.Len(t, *hostResp.Host.MDM.Profiles, 1)
	require.Equal(t, (*hostResp.Host.MDM.Profiles)[0].Name, "TestImmediateDelete")

	// now delete the profile before it's sent, we should see the host_mdm_apple_profiles entry go
	// away
	deletedCP := testProfiles["TestImmediateDelete"]
	deletePath := fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", deletedCP.ProfileID)
	var deleteResp deleteMDMAppleConfigProfileResponse
	s.DoJSON("DELETE", deletePath, nil, http.StatusOK, &deleteResp)
	// confirm deleted
	var listResp listMDMAppleConfigProfilesResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/profiles", listMDMAppleConfigProfilesRequest{TeamID: teamDelete.ID}, http.StatusOK, &listResp)
	require.Len(t, listResp.ConfigProfiles, 0)
	getPath := fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", deletedCP.ProfileID)
	_ = s.DoRawWithHeaders("GET", getPath, nil, http.StatusNotFound, map[string]string{"Authorization": fmt.Sprintf("Bearer %s", s.token)})
	// confirm no host profiles
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.Nil(t, hostResp.Host.MDM.Profiles)

	// list profiles (no team)
	expectedCP := testProfiles["TestNoTeam"]
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/profiles", nil, http.StatusOK, &listResp)
	require.Len(t, listResp.ConfigProfiles, 1)
	respCP := listResp.ConfigProfiles[0]
	require.Equal(t, expectedCP.Name, respCP.Name)
	checkConfigProfile(expectedCP, *respCP)
	require.Empty(t, respCP.Mobileconfig) // list profiles endpoint shouldn't include mobileconfig bytes
	require.Empty(t, respCP.TeamID)       // zero means no team

	// list profiles (team 1)
	expectedCP = testProfiles["TestWithTeamID"]
	listResp = listMDMAppleConfigProfilesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/profiles", listMDMAppleConfigProfilesRequest{TeamID: testTeam.ID}, http.StatusOK, &listResp)
	require.Len(t, listResp.ConfigProfiles, 1)
	respCP = listResp.ConfigProfiles[0]
	require.Equal(t, expectedCP.Name, respCP.Name)
	checkConfigProfile(expectedCP, *respCP)
	require.Empty(t, respCP.Mobileconfig)         // list profiles endpoint shouldn't include mobileconfig bytes
	require.Equal(t, testTeam.ID, *respCP.TeamID) // team 1

	// get profile (no team)
	expectedCP = testProfiles["TestNoTeam"]
	getPath = fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", expectedCP.ProfileID)
	getResp := s.DoRawWithHeaders("GET", getPath, nil, http.StatusOK, map[string]string{"Authorization": fmt.Sprintf("Bearer %s", s.token)})
	checkGetResponse(getResp, expectedCP)

	// get profile (team 1)
	expectedCP = testProfiles["TestWithTeamID"]
	getPath = fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", expectedCP.ProfileID)
	getResp = s.DoRawWithHeaders("GET", getPath, nil, http.StatusOK, map[string]string{"Authorization": fmt.Sprintf("Bearer %s", s.token)})
	checkGetResponse(getResp, expectedCP)

	// delete profile (no team)
	deletedCP = testProfiles["TestNoTeam"]
	deletePath = fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", deletedCP.ProfileID)
	s.DoJSON("DELETE", deletePath, nil, http.StatusOK, &deleteResp)
	// confirm deleted
	listResp = listMDMAppleConfigProfilesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/profiles", listMDMAppleConfigProfilesRequest{}, http.StatusOK, &listResp)
	require.Len(t, listResp.ConfigProfiles, 0)
	getPath = fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", deletedCP.ProfileID)
	_ = s.DoRawWithHeaders("GET", getPath, nil, http.StatusNotFound, map[string]string{"Authorization": fmt.Sprintf("Bearer %s", s.token)})

	// delete profile (team 1)
	deletedCP = testProfiles["TestWithTeamID"]
	deletePath = fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", deletedCP.ProfileID)
	deleteResp = deleteMDMAppleConfigProfileResponse{}
	s.DoJSON("DELETE", deletePath, nil, http.StatusOK, &deleteResp)
	// confirm deleted
	listResp = listMDMAppleConfigProfilesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/profiles", listMDMAppleConfigProfilesRequest{TeamID: testTeam.ID}, http.StatusOK, &listResp)
	require.Len(t, listResp.ConfigProfiles, 0)
	getPath = fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", deletedCP.ProfileID)
	_ = s.DoRawWithHeaders("GET", getPath, nil, http.StatusNotFound, map[string]string{"Authorization": fmt.Sprintf("Bearer %s", s.token)})

	// fail to create new profile (no team), invalid fleet secret
	testProfiles["badSecrets"] = fleet.MDMAppleConfigProfile{
		Name:       "badSecrets",
		Identifier: "badSecrets.One",
		Mobileconfig: mobileconfig.Mobileconfig(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array/>
	<key>PayloadDisplayName</key>
	<string>badSecrets</string>
	<key>PayloadIdentifier</key>
	<string>badSecrets.One</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>$FLEET_SECRET_INVALID.35E2029E-A0C2-4754-B709-4CAAB1B8D3CB</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
`),
	}

	body, headers = generateNewReq("badSecrets", nil)
	newResp = s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusUnprocessableEntity, headers)
	errMsg := extractServerErrorText(newResp.Body)
	require.Contains(t, errMsg, "$FLEET_SECRET_INVALID")

	// trying to add/delete profiles with identifiers managed by Fleet fails
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		generateTestProfile("TestNoTeam", p)
		body, headers := generateNewReq("TestNoTeam", nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)

		generateTestProfile("TestWithTeamID", p)
		body, headers = generateNewReq("TestWithTeamID", nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)
		cp, err := fleet.NewMDMAppleConfigProfile(mobileconfigForTestWithContent("N1", "I1", p, "random", ""), nil)
		require.NoError(t, err)
		testProfiles["WithContent"] = *cp
		body, headers = generateNewReq("WithContent", nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)
	}

	// trying to add profiles with identifiers managed by Fleet fails
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		generateTestProfile("TestNoTeam", p)
		body, headers := generateNewReq("TestNoTeam", nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)

		generateTestProfile("TestWithTeamID", p)
		body, headers = generateNewReq("TestWithTeamID", nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)
		cp, err := fleet.NewMDMAppleConfigProfile(mobileconfigForTestWithContent("N1", "I1", p, "random", ""), nil)
		require.NoError(t, err)
		testProfiles["WithContent"] = *cp
		body, headers = generateNewReq("WithContent", nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)
	}

	// trying to add profiles with names reserved by Fleet fails
	for name := range servermdm.FleetReservedProfileNames() {
		cp := &fleet.MDMAppleConfigProfile{
			Name:         name,
			Identifier:   "valid.identifier",
			Mobileconfig: mcBytesForTest(name, "valid.identifier", "some-uuid"),
		}
		body, headers := generateNewProfileMultipartRequest(t, "some_filename", cp.Mobileconfig, s.token, nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)

		body, headers = generateNewProfileMultipartRequest(t, "some_filename", cp.Mobileconfig, s.token, map[string][]string{
			"team_id": {fmt.Sprintf("%d", testTeam.ID)},
		})
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)

		cp, err := fleet.NewMDMAppleConfigProfile(mobileconfigForTestWithContent(
			"valid outer name",
			"valid.outer.identifier",
			"valid.inner.identifer",
			"some-uuid",
			name,
		), nil)
		require.NoError(t, err)
		body, headers = generateNewProfileMultipartRequest(t, "some_filename", cp.Mobileconfig, s.token, nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)

		cp.TeamID = &testTeam.ID
		body, headers = generateNewProfileMultipartRequest(t, "some_filename", cp.Mobileconfig, s.token, map[string][]string{
			"team_id": {fmt.Sprintf("%d", testTeam.ID)},
		})

		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)
	}

	// make fleet add a FileVault profile
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	profile := s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// try to delete the profile
	deletePath = fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", profile.ProfileID)
	deleteResp = deleteMDMAppleConfigProfileResponse{}
	s.DoJSON("DELETE", deletePath, nil, http.StatusBadRequest, &deleteResp)
}

func (s *integrationMDMTestSuite) TestHostMDMProfilesExcludeLabels() {
	t := s.T()
	ctx := context.Background()

	triggerReconcileProfiles := func() {
		s.awaitTriggerProfileSchedule(t)
		// this will only mark them as "pending", as the response to confirm
		// profile deployment is asynchronous, so we simulate it here by
		// updating any "pending" (not NULL) profiles to "verifying"
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			if _, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET status = ? WHERE status = ?`, fleet.OSSettingsVerifying, fleet.OSSettingsPending); err != nil {
				return err
			}
			if _, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_declarations SET status = ? WHERE status = ?`, fleet.OSSettingsVerifying, fleet.OSSettingsPending); err != nil {
				return err
			}
			if _, err := q.ExecContext(ctx, `UPDATE host_mdm_windows_profiles SET status = ? WHERE status = ?`, fleet.OSSettingsVerifying, fleet.OSSettingsPending); err != nil {
				return err
			}
			return nil
		})
	}

	// run the crons immediately, will create the Fleet-controlled profiles that
	// will then be expected to be applied (e.g. com.fleetdm.fleetd.config and
	// com.fleetdm.caroot)
	// first create the no-team enroll secret (required to create the fleet profiles)
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret",
		applyEnrollSecretSpecRequest{
			Spec: &fleet.EnrollSecretSpec{Secrets: []*fleet.EnrollSecret{{Secret: "super-global-secret"}}},
		}, http.StatusOK, &applyResp)
	s.awaitTriggerProfileSchedule(t)

	// create an Apple and a Windows host
	appleHost, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	windowsHost, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	// create a few labels
	labels := make([]*fleet.Label, 5)
	for i := 0; i < len(labels); i++ {
		label, err := s.ds.NewLabel(ctx, &fleet.Label{Name: fmt.Sprintf("label-%d", i), Query: "select 1;"})
		require.NoError(t, err)
		labels[i] = label
	}
	// simulate reporting label results for those hosts
	appleHost.LabelUpdatedAt = time.Now()
	windowsHost.LabelUpdatedAt = time.Now()
	err := s.ds.UpdateHost(ctx, appleHost)
	require.NoError(t, err)
	err = s.ds.UpdateHost(ctx, windowsHost)
	require.NoError(t, err)

	// set an Apple profile and declaration and a Windows profile
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "A1", Contents: mobileconfigForTest("A1", "A1"), LabelsExcludeAny: []string{labels[0].Name, labels[1].Name}},
		{Name: "W2", Contents: syncMLForTest("./Foo/W2"), LabelsExcludeAny: []string{labels[2].Name, labels[3].Name}},
		{Name: "D3", Contents: declarationForTest("D3"), LabelsExcludeAny: []string{labels[4].Name}},
	}}, http.StatusNoContent)

	// hosts are not members of any label yet, so running the cron applies the labels
	s.awaitTriggerProfileSchedule(t)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "A1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		windowsHost: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	// simulate the reconcile profiles deployment
	triggerReconcileProfiles()
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "A1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		windowsHost: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// mark some profiles as verified (despite accepting a HostMacOSProfile struct, it supports Windows too)
	err = apple_mdm.VerifyHostMDMProfiles(ctx, s.ds, appleHost, map[string]*fleet.HostMacOSProfile{
		"A1": {Identifier: "A1", DisplayName: "A1", InstallDate: time.Now()},
	})
	require.NoError(t, err)
	err = apple_mdm.VerifyHostMDMProfiles(ctx, s.ds, windowsHost, map[string]*fleet.HostMacOSProfile{
		"W2": {Identifier: "W2", DisplayName: "W2", InstallDate: time.Now()},
	})
	require.NoError(t, err)

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "A1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerified},
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		windowsHost: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerified},
		},
	})

	// make hosts members of labels [1], [2], [3] and [4], meaning that none of the profiles apply anymore
	err = s.ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{
		{labels[1].ID, appleHost.ID},
		{labels[2].ID, appleHost.ID},
		{labels[3].ID, appleHost.ID},
		{labels[4].ID, appleHost.ID},
		{labels[1].ID, windowsHost.ID},
		{labels[2].ID, windowsHost.ID},
		{labels[3].ID, windowsHost.ID},
		{labels[4].ID, windowsHost.ID},
	})
	require.NoError(t, err)

	s.awaitTriggerProfileSchedule(t)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "A1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
	// windows profiles go straight to removed without getting deleted on the host
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		windowsHost: {},
	})

	// remove membership of labels [2] for Windows, and [4] for Apple, meaning
	// that only D3 will be installed on Apple (as the Windows host is still
	// member of an excluded label)
	err = s.ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{
		{labels[4].ID, appleHost.ID},
		{labels[2].ID, windowsHost.ID},
	})
	require.NoError(t, err)

	s.awaitTriggerProfileSchedule(t)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "A1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		windowsHost: {},
	})

	// remove label [3] as an excluded label for the Windows profile, meaning
	// that the host now meets the requirement to install.
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "A1", Contents: mobileconfigForTest("A1", "A1"), LabelsExcludeAny: []string{labels[0].Name, labels[1].Name}},
		{Name: "W2", Contents: syncMLForTest("./Foo/W2"), LabelsExcludeAny: []string{labels[2].Name}},
		{Name: "D3", Contents: declarationForTest("D3"), LabelsExcludeAny: []string{labels[4].Name}},
	}}, http.StatusNoContent)

	s.awaitTriggerProfileSchedule(t)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "A1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		windowsHost: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	// simulate the reconcile profiles deployment and mark as verified
	triggerReconcileProfiles()
	err = apple_mdm.VerifyHostMDMProfiles(ctx, s.ds, windowsHost, map[string]*fleet.HostMacOSProfile{
		"W2": {Identifier: "W2", DisplayName: "W2", InstallDate: time.Now()},
	})
	require.NoError(t, err)

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		windowsHost: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerified},
		},
	})

	// break the A1 profile by deleting labels [1]
	err = s.ds.DeleteLabel(ctx, labels[1].Name)
	require.NoError(t, err)

	// it doesn't get installed to the Apple host, as it is broken
	triggerReconcileProfiles()
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		windowsHost: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerified},
		},
	})

	// it also doesn't get installed to a new host not a member of any labels
	appleHost2, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	triggerReconcileProfiles()
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		appleHost2: {
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		windowsHost: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerified},
		},
	})

	// delete labels [2] and [4], breaking D3 and W2, they don't get removed
	// since they are broken
	err = s.ds.DeleteLabel(ctx, labels[2].Name)
	require.NoError(t, err)
	err = s.ds.DeleteLabel(ctx, labels[4].Name)
	require.NoError(t, err)

	triggerReconcileProfiles()
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		appleHost2: {
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		windowsHost: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerified},
		},
	})
}

func (s *integrationMDMTestSuite) TestMDMProfilesIncludeAnyLabels() {
	t := s.T()
	ctx := context.Background()

	triggerReconcileProfiles := func() {
		s.awaitTriggerProfileSchedule(t)
		// this will only mark them as "pending", as the response to confirm
		// profile deployment is asynchronous, so we simulate it here by
		// updating any "pending" (not NULL) profiles to "verifying"
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			if _, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET status = ? WHERE status = ?`, fleet.OSSettingsVerifying, fleet.OSSettingsPending); err != nil {
				return err
			}
			if _, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_declarations SET status = ? WHERE status = ?`, fleet.OSSettingsVerifying, fleet.OSSettingsPending); err != nil {
				return err
			}
			if _, err := q.ExecContext(ctx, `UPDATE host_mdm_windows_profiles SET status = ? WHERE status = ?`, fleet.OSSettingsVerifying, fleet.OSSettingsPending); err != nil {
				return err
			}
			return nil
		})
	}

	// run the crons immediately, will create the Fleet-controlled profiles that
	// will then be expected to be applied (e.g. com.fleetdm.fleetd.config and
	// com.fleetdm.caroot)
	// first create the no-team enroll secret (required to create the fleet profiles)
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret",
		applyEnrollSecretSpecRequest{
			Spec: &fleet.EnrollSecretSpec{Secrets: []*fleet.EnrollSecret{{Secret: "super-global-secret"}}},
		}, http.StatusOK, &applyResp)
	s.awaitTriggerProfileSchedule(t)

	// create an Apple and a Windows host
	appleHost, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	windowsHost, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	// create a few labels, we'll use the first five for "exclude any" profiles and the remaining for "include any"
	labels := make([]*fleet.Label, 10)
	for i := 0; i < len(labels); i++ {
		label, err := s.ds.NewLabel(ctx, &fleet.Label{Name: fmt.Sprintf("label-%d", i), Query: "select 1;"})
		require.NoError(t, err)
		labels[i] = label
	}
	// simulate reporting label results for those hosts
	appleHost.LabelUpdatedAt = time.Now()
	windowsHost.LabelUpdatedAt = time.Now()
	err := s.ds.UpdateHost(ctx, appleHost)
	require.NoError(t, err)
	err = s.ds.UpdateHost(ctx, windowsHost)
	require.NoError(t, err)

	// set up some Apple profiles and declarations and Windows profiles
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "A1", Contents: mobileconfigForTest("A1", "A1"), LabelsIncludeAny: []string{labels[0].Name, labels[1].Name}},
		{Name: "W2", Contents: syncMLForTest("./Foo/W2"), LabelsIncludeAny: []string{labels[2].Name, labels[3].Name}},
		{Name: "D3", Contents: declarationForTest("D3"), LabelsIncludeAny: []string{labels[4].Name}},
	}}, http.StatusNoContent)

	// hosts are not members of any label yet, so running the cron applies no labels
	s.awaitTriggerProfileSchedule(t)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		windowsHost: {},
	})

	// make hosts members of labels [1], [2], [3] and [4], meaning that each of the "include any"
	// labels will now match at least one host
	err = s.ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{
		{labels[0].ID, appleHost.ID},
		{labels[1].ID, appleHost.ID},
		{labels[2].ID, appleHost.ID},
		{labels[3].ID, appleHost.ID},
		{labels[4].ID, appleHost.ID},
		{labels[1].ID, windowsHost.ID},
		{labels[2].ID, windowsHost.ID},
		{labels[3].ID, windowsHost.ID},
		{labels[4].ID, windowsHost.ID},
	})
	require.NoError(t, err)

	triggerReconcileProfiles()
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "A1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		windowsHost: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// remove membership of labels [2] for Windows, and [1] and [4] for Apple, meaning
	// that D3 will be removed on Apple, A1 will remain on Apple because the host is still a member
	// of [0], and W2 will remain on Windows because the host is still a member of [3]
	err = s.ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{
		{labels[1].ID, appleHost.ID},
		{labels[4].ID, appleHost.ID},
		{labels[2].ID, windowsHost.ID},
	})
	require.NoError(t, err)

	s.awaitTriggerProfileSchedule(t)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "A1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		windowsHost: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
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

	// Get profile with that enroll secret
	resp := s.Do("GET", "/api/latest/fleet/enrollment_profiles/ota", getOTAProfileRequest{}, http.StatusOK, "enroll_secret", globalEnrollSec)
	require.NotZero(t, resp.ContentLength)
	require.Contains(t, resp.Header.Get("Content-Disposition"), `attachment;filename="fleet-mdm-enrollment-profile.mobileconfig"`)
	require.Contains(t, resp.Header.Get("Content-Type"), "application/x-apple-aspen-config")
	require.Contains(t, resp.Header.Get("X-Content-Type-Options"), "nosniff")
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, resp.ContentLength, int64(len(b)))
	require.Contains(t, string(b), "com.fleetdm.fleet.mdm.apple.ota")
	require.Contains(t, string(b), fmt.Sprintf("%s/api/v1/fleet/ota_enrollment?enroll_secret=%s", cfg.ServerSettings.ServerURL, escSec))
	require.Contains(t, string(b), cfg.OrgInfo.OrgName)
}
