package service

// Batch configuration profile tests for the MDM suite.
//
// Belongs here: the batch-set, batch-modify, and batch-resend profile endpoints (including backwards
// compatibility and reserved names), and OS update settings deployed through fleet specs.
//
// Does not belong here: single-profile CRUD (integration_mdm_profiles_crud_test.go) and resends
// triggered by policy automations (integration_mdm_profiles_reconciler_test.go).

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	servermdm "github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/service/redis_key_value"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`,
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
		if p == mobileconfig.FleetCustomSettingsPayloadType {
			// FileVault options in the custom settings payload are checked in file_vault_options_test.go
			continue
		}
		res := s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
			mobileconfigForTestWithContent("N1", "I1", "II1", p, ""),
		}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg := extractServerErrorText(res.Body)
		switch p {
		case mobileconfig.FleetFileVaultPayloadType, mobileconfig.FleetRecoveryKeyEscrowPayloadType:
			assert.Contains(t, errMsg, mobileconfig.DiskEncryptionProfileRestrictionErrMsg)
		default:
			assert.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadType(s): %s", p))
		}
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
		fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name),
		0,
	)
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

	// invalid profile (bad mobileconfig)
	resp = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{
			Name: "Bad mobileconfig", Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array/>`),
		},
	}}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(resp.Body), "Validation Failed: new MDMAppleConfigProfile: XML syntax error")

	// apply an empty set to no-team
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: backwardsCompatProfilesParam{}}, http.StatusNoContent)
	// Nothing changed, so no activity items
	s.lastActivityOfTypeDoesNotMatch(
		fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`,
		0,
	)
	s.lastActivityOfTypeDoesNotMatch(
		fleet.ActivityTypeEditedWindowsProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`,
		0,
	)
	s.lastActivityOfTypeDoesNotMatch(
		fleet.ActivityTypeEditedDeclarationProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`,
		0,
	)

	// apply to both team id and name
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: backwardsCompatProfilesParam{}},
		http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID), "team_name", tm.Name)

	// invalid team name
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: backwardsCompatProfilesParam{}},
		http.StatusNotFound, "team_name", uuid.New().String())

	// duplicate PayloadDisplayName, with no name to override it
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Contents: mobileconfigForTest("N1", "I1")},
		{Contents: mobileconfigForTest("N1", "I2")},
		{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
		{Name: "N4", Contents: declarationForTest("D1")},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))

	// a given name replaces PayloadDisplayName, so two names that collide
	// are duplicates even when the payload names differ
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("P1", "I1")},
		{Name: "N1", Contents: mobileconfigForTest("P2", "I2")},
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
		if p == mobileconfig.FleetCustomSettingsPayloadType {
			// FileVault options in the custom settings payload are checked in file_vault_options_test.go
			continue
		}
		res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "N1", Contents: mobileconfigForTestWithContent("N1", "I1", "II1", p, "")},
			{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
			{Name: "N4", Contents: declarationForTest("D1")},
		}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg := extractServerErrorText(res.Body)
		switch p {
		case mobileconfig.FleetFileVaultPayloadType, mobileconfig.FleetRecoveryKeyEscrowPayloadType:
			assert.Contains(t, errMsg, mobileconfig.DiskEncryptionProfileRestrictionErrMsg)
		default:
			assert.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadType(s): %s", p))
		}
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
		require.Contains(t, errMsg, "is a forbidden declaration type", dt)
	}
	// and one more for the software update declaration
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
		{Name: "N4", Contents: declarationForTestWithType("D1", "com.apple.configuration.softwareupdate.enforcement.specific")},
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))
	// reset the profiles for later dry_run
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

	// invalid JSON
	res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
		{Name: "N4", Contents: []byte(`{"foo":}`)},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "N4 is not a valid macOS, Windows, or Android configuration profile")

	// profiles with reserved Windows location URIs
	// bitlocker
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: syncml.FleetBitLockerTargetLocURI, Contents: syncMLForTest(fmt.Sprintf("%s/Foo", syncml.FleetBitLockerTargetLocURI))},
		{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, syncml.DiskEncryptionProfileRestrictionErrMsg)

	// os updates
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: syncml.FleetOSUpdateTargetLocURI, Contents: syncMLForTest(fmt.Sprintf(".%s/Foo", syncml.FleetOSUpdateTargetLocURI))},
		{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))
	// reset the profiles for later dry_run
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

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
		fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name),
		0,
	)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedWindowsProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name),
		0,
	)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedDeclarationProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name),
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
	require.Contains(t, msg, `Couldn't update. Label "no-such-label" doesn't exist. Please remove the label from the configuration profile.`)

	// two include modes in the same profile is invalid (exclude may combine with either include mode)
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1"), LabelsIncludeAll: []string{lbl1.Name}, LabelsIncludeAny: []string{lbl2.Name}},
	}}, http.StatusUnprocessableEntity)
	msg = extractServerErrorText(res.Body)
	require.Contains(t, msg, `For each profile, only one of "labels_include_all", "labels_include_any" or "labels" can be included.`)

	// same label in both include and exclude is invalid
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1"), LabelsIncludeAll: []string{lbl1.Name}, LabelsExcludeAny: []string{lbl1.Name}},
	}}, http.StatusUnprocessableEntity)
	msg = extractServerErrorText(res.Body)
	require.Contains(t, msg, fmt.Sprintf(`Label %q cannot appear in both include and exclude lists.`, lbl1.Name))

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
	require.Equal(t, []fleet.ConfigurationProfileLabel{
		{LabelID: lbl1.ID, LabelName: lbl1.Name},
		{LabelID: lbl2.ID, LabelName: lbl2.Name},
	}, listResp.Profiles[0].LabelsIncludeAll)
	require.Nil(t, listResp.Profiles[0].LabelsExcludeAny)
	require.Equal(t, []fleet.ConfigurationProfileLabel{
		{LabelID: lbl1.ID, LabelName: lbl1.Name},
	}, listResp.Profiles[1].LabelsIncludeAll)
	require.Nil(t, listResp.Profiles[1].LabelsExcludeAny)
	require.Equal(t, []fleet.ConfigurationProfileLabel{
		{LabelID: lbl2.ID, LabelName: lbl2.Name},
	}, listResp.Profiles[2].LabelsExcludeAny)
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
	require.Equal(t, []fleet.ConfigurationProfileLabel{
		{LabelID: lbl1.ID, LabelName: lbl1.Name},
		{LabelID: lbl3.ID, LabelName: lbl3.Name},
	}, listResp.Profiles[0].LabelsExcludeAny)
	require.Nil(t, listResp.Profiles[0].LabelsIncludeAll)
	require.Equal(t, []fleet.ConfigurationProfileLabel{
		{LabelID: lbl2.ID, LabelName: lbl2.Name},
	}, listResp.Profiles[1].LabelsIncludeAll)
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
			expectErr: "More than one configuration profile have the same name 'N1'",
		},
		{
			payload:   []fleet.MDMProfileBatchPayload{{Name: "N1", Contents: declBytes}, {Name: "N1", Contents: winBytes}},
			expectErr: "More than one configuration profile have the same name 'N1'",
		},
		{
			payload:   []fleet.MDMProfileBatchPayload{{Name: "N1", Contents: mcBytes}, {Name: "N1", Contents: declBytes}},
			expectErr: "More than one configuration profile have the same name 'N1'",
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

// This tests the new public API endpoint for batch modifying MDM profiles
func (s *integrationMDMTestSuite) TestBatchModifyMDMProfiles() {
	t := s.T()
	ctx := context.Background()

	// create a new team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "batch_set_mdm_profiles"})
	require.NoError(t, err)

	bigString := strings.Repeat("a", 1024*1024+1)

	// Profile is too big
	resp := s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{{Profile: []byte(bigString)}}},
		http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(resp.Body), "Validation Failed: maximum configuration profile file size is 1 MB")

	// apply an empty set to no-team
	s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{}}, http.StatusNoContent)
	// Nothing changed, so no activity items
	s.lastActivityOfTypeDoesNotMatch(
		fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`,
		0,
	)
	s.lastActivityOfTypeDoesNotMatch(
		fleet.ActivityTypeEditedWindowsProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`,
		0,
	)
	s.lastActivityOfTypeDoesNotMatch(
		fleet.ActivityTypeEditedDeclarationProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`,
		0,
	)

	// apply to both team id and name
	s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{}},
		http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID), "team_name", tm.Name)

	// invalid team name
	s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{}},
		http.StatusNotFound, "team_name", uuid.New().String())

	// duplicate PayloadDisplayName
	s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
		{DisplayName: "N1", Profile: mobileconfigForTest("N1", "I1")},
		{DisplayName: "N2", Profile: mobileconfigForTest("N1", "I2")},
		{DisplayName: "N3", Profile: syncMLForTest("./Foo/Bar")},
		{DisplayName: "N4", Profile: declarationForTest("D1")},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))

	// profiles with reserved macOS identifiers
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		res := s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
			{DisplayName: "N1", Profile: mobileconfigForTest("N1", "I1")},
			{DisplayName: p, Profile: mobileconfigForTest(p, p)},
			{DisplayName: "N3", Profile: syncMLForTest("./Foo/Bar")},
			{DisplayName: "N4", Profile: declarationForTest("D1")},
		}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: payload identifier %s is not allowed", p))
	}

	// payloads with reserved types
	for p := range mobileconfig.FleetPayloadTypes() {
		if p == mobileconfig.FleetCustomSettingsPayloadType {
			// FileVault options in the custom settings payload are checked in file_vault_options_test.go
			continue
		}
		res := s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
			{DisplayName: "N1", Profile: mobileconfigForTestWithContent("N1", "I1", "II1", p, "")},
			{DisplayName: "N3", Profile: syncMLForTest("./Foo/Bar")},
			{DisplayName: "N4", Profile: declarationForTest("D1")},
		}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg := extractServerErrorText(res.Body)
		switch p {
		case mobileconfig.FleetFileVaultPayloadType, mobileconfig.FleetRecoveryKeyEscrowPayloadType:
			assert.Contains(t, errMsg, mobileconfig.DiskEncryptionProfileRestrictionErrMsg)
		default:
			assert.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadType(s): %s", p))
		}
	}

	// payloads with reserved identifiers
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		res := s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
			{DisplayName: "N1", Profile: mobileconfigForTestWithContent("N1", "I1", p, "random", "")},
			{DisplayName: "N3", Profile: syncMLForTest("./Foo/Bar")},
			{DisplayName: "N4", Profile: declarationForTest("D1")},
		}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadIdentifier(s): %s", p))
	}

	// profiles with forbidden declaration types
	for dt := range fleet.ForbiddenDeclTypes {
		res := s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
			{DisplayName: "N1", Profile: mobileconfigForTest("N1", "I1")},
			{DisplayName: "N3", Profile: syncMLForTest("./Foo/Bar")},
			{DisplayName: "N4", Profile: declarationForTestWithType("D1", dt)},
		}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, "is a forbidden declaration", dt)
	}
	// and one more for the software update declaration which should succeed.
	s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
		{DisplayName: "N1", Profile: mobileconfigForTest("N1", "I1")},
		{DisplayName: "N3", Profile: syncMLForTest("./Foo/Bar")},
		{DisplayName: "N4", Profile: declarationForTestWithType("D1", "com.apple.configuration.softwareupdate.enforcement.specific")},
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

	// reset the profiles for later dry_run
	s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

	// invalid JSON
	res := s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
		{DisplayName: "N1", Profile: mobileconfigForTest("N1", "I1")},
		{DisplayName: "N3", Profile: syncMLForTest("./Foo/Bar")},
		{DisplayName: "N4", Profile: []byte(`{"foo":}`)},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "N4 is not a valid macOS, Windows, or Android configuration profile.")

	// profiles with reserved Windows location URIs
	// bitlocker
	res = s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
		{DisplayName: "N1", Profile: mobileconfigForTest("N1", "I1")},
		{DisplayName: syncml.FleetBitLockerTargetLocURI, Profile: syncMLForTest(fmt.Sprintf("%s/Foo", syncml.FleetBitLockerTargetLocURI))},
		{DisplayName: "N3", Profile: syncMLForTest("./Foo/Bar")},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, syncml.DiskEncryptionProfileRestrictionErrMsg)

	// os updates
	s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
		{DisplayName: "N1", Profile: mobileconfigForTest("N1", "I1")},
		{DisplayName: syncml.FleetOSUpdateTargetLocURI, Profile: syncMLForTest(fmt.Sprintf(".%s/Foo", syncml.FleetOSUpdateTargetLocURI))},
		{DisplayName: "N3", Profile: syncMLForTest("./Foo/Bar")},
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

	// reset the profiles for later dry_run
	s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

	// invalid windows tag
	res = s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
		{DisplayName: "N3", Profile: []byte(`<Exec></Exec>`)},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Windows configuration profiles can only have <Replace> or <Add> top level elements.")

	// invalid xml
	res = s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
		{DisplayName: "N3", Profile: []byte(`foo`)},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Windows configuration profiles can only have <Replace> or <Add> top level elements.")

	// successfully apply windows and macOS a profiles for the team, but it's a dry run
	s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
		{DisplayName: "N1", Profile: mobileconfigForTest("N1", "I1")},
		{DisplayName: "N2", Profile: syncMLForTest("./Foo/Bar")},
		{DisplayName: "N4", Profile: declarationForTest("D1")},
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID), "dry_run", "true")
	s.assertConfigProfilesByIdentifier(&tm.ID, "I1", false)
	s.assertWindowsConfigProfilesByName(&tm.ID, "N1", false)

	// successfully apply for a team and verify activities
	s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
		{DisplayName: "NotRelevant", Profile: mobileconfigForTest("N1", "I1")}, // Check that we don't care about displayname for mobileconfig profiles
		{DisplayName: "N2", Profile: syncMLForTest("./Foo/Bar")},
		{DisplayName: "N4", Profile: declarationForTest("D1")},
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))
	s.assertConfigProfilesByIdentifier(&tm.ID, "I1", true)
	s.assertWindowsConfigProfilesByName(&tm.ID, "N2", true)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name),
		0,
	)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedWindowsProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name),
		0,
	)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedDeclarationProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name),
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
	res = s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
		{DisplayName: "N1", Profile: mobileconfigForTest("N1", "I1"), LabelsIncludeAll: []string{lbl1.Name, "no-such-label"}},
	}}, http.StatusBadRequest)
	msg := extractServerErrorText(res.Body)
	require.Contains(t, msg, `Couldn't update. Label "no-such-label" doesn't exist. Please remove the label from the configuration profile.`)

	// two include modes in the same profile is invalid (exclude may combine with either include mode)
	res = s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
		{DisplayName: "N1", Profile: mobileconfigForTest("N1", "I1"), LabelsIncludeAll: []string{lbl1.Name}, LabelsIncludeAny: []string{lbl2.Name}},
	}}, http.StatusUnprocessableEntity)
	msg = extractServerErrorText(res.Body)
	require.Contains(t, msg, `For each profile, only one of "labels_include_all", "labels_include_any" or "labels" can be included.`)

	// successful batch-set
	s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
		{DisplayName: "N1", Profile: mobileconfigForTest("N1", "I1"), LabelsIncludeAny: []string{lbl1.Name, lbl2.Name}},
		{DisplayName: "N2", Profile: syncMLForTest("./Foo/Bar"), LabelsIncludeAll: []string{lbl1.Name}},
		{DisplayName: "N4", Profile: declarationForTest("D1"), LabelsExcludeAny: []string{lbl2.Name}},
	}}, http.StatusNoContent)

	// confirm expected results
	var listResp listMDMConfigProfilesResponse
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Profiles, 3)
	require.Equal(t, "N1", listResp.Profiles[0].Name)
	require.Equal(t, "N2", listResp.Profiles[1].Name)
	require.Equal(t, "N4", listResp.Profiles[2].Name)
	require.Equal(t, []fleet.ConfigurationProfileLabel{
		{LabelID: lbl1.ID, LabelName: lbl1.Name},
		{LabelID: lbl2.ID, LabelName: lbl2.Name},
	}, listResp.Profiles[0].LabelsIncludeAny)
	require.Nil(t, listResp.Profiles[0].LabelsExcludeAny)
	require.Equal(t, []fleet.ConfigurationProfileLabel{
		{LabelID: lbl1.ID, LabelName: lbl1.Name},
	}, listResp.Profiles[1].LabelsIncludeAll)
	require.Nil(t, listResp.Profiles[1].LabelsExcludeAny)
	require.Equal(t, []fleet.ConfigurationProfileLabel{
		{LabelID: lbl2.ID, LabelName: lbl2.Name},
	}, listResp.Profiles[2].LabelsExcludeAny)
	require.Nil(t, listResp.Profiles[2].LabelsIncludeAll)

	// successful batch-set that updates some labels
	s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
		{DisplayName: "N1", Profile: mobileconfigForTest("N1", "I1"), LabelsExcludeAny: []string{lbl1.Name, lbl3.Name}},
		{DisplayName: "N2", Profile: syncMLForTest("./Foo/Bar"), LabelsIncludeAll: []string{lbl2.Name}},
	}}, http.StatusNoContent)

	listResp = listMDMConfigProfilesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Profiles, 2)
	require.Equal(t, "N1", listResp.Profiles[0].Name)
	require.Equal(t, "N2", listResp.Profiles[1].Name)
	require.Equal(t, []fleet.ConfigurationProfileLabel{
		{LabelID: lbl1.ID, LabelName: lbl1.Name},
		{LabelID: lbl3.ID, LabelName: lbl3.Name},
	}, listResp.Profiles[0].LabelsExcludeAny)
	require.Nil(t, listResp.Profiles[0].LabelsIncludeAll)
	require.Equal(t, []fleet.ConfigurationProfileLabel{
		{LabelID: lbl2.ID, LabelName: lbl2.Name},
	}, listResp.Profiles[1].LabelsIncludeAll)
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
		payload   []fleet.BatchModifyMDMConfigProfilePayload
		expectErr string
	}{
		{
			payload:   []fleet.BatchModifyMDMConfigProfilePayload{{DisplayName: "N1", Profile: mcBytes}, {DisplayName: "N1", Profile: winBytes}},
			expectErr: "More than one configuration profile have the same name 'N1'",
		},
		{
			payload:   []fleet.BatchModifyMDMConfigProfilePayload{{DisplayName: "N1", Profile: declBytes}, {DisplayName: "N1", Profile: winBytes}},
			expectErr: "More than one configuration profile have the same name 'N1'",
		},
		{
			payload:   []fleet.BatchModifyMDMConfigProfilePayload{{DisplayName: "N1", Profile: mcBytes}, {DisplayName: "N1", Profile: declBytes}},
			expectErr: "More than one configuration profile have the same name 'N1'",
		},
	} {
		// team profiles
		res = s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: p.payload},
			http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, p.expectErr)
		// no team profiles
		res = s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: p.payload}, http.StatusUnprocessableEntity)
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, p.expectErr)
	}

	// Get the current list of configuration profiles
	var currentProfiles listMDMConfigProfilesResponse
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", nil, http.StatusOK, &currentProfiles)
	require.NotEmpty(t, currentProfiles.Profiles)

	// Now we disable all three MDM's
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.MDM.EnabledAndConfigured = false
	appCfg.MDM.WindowsEnabledAndConfigured = false
	appCfg.MDM.AndroidEnabledAndConfigured = false
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)

	// Now do a batch with profiles in it, to see it fails trying to add.
	s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
		{DisplayName: "NEW", Profile: mobileconfigForTest("NEW", "INEW")},
	}}, http.StatusUnprocessableEntity)

	// Now do a batch without any profiles to ensure we can delete with all MDM's disabled.
	s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{}}, http.StatusNoContent)
}

func (s *integrationMDMTestSuite) TestBatchProfilesEndpointsRequireProfileList() {
	t := s.T()
	ctx := t.Context()

	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	teamID := fmt.Sprint(tm.ID)

	profileB64 := base64.StdEncoding.EncodeToString(mobileconfigForTest("N1", "I1"))
	assertProfileCount := func(want int) {
		var resp listMDMConfigProfilesResponse
		s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", nil, http.StatusOK, &resp, "team_id", teamID)
		require.Len(t, resp.Profiles, want)
	}

	cases := []struct {
		path        string
		field       string
		upload      string
		badBodies   []string
		emptyBodies []string
	}{
		{
			path:   "/api/latest/fleet/configuration_profiles/batch",
			field:  "configuration_profiles",
			upload: `{"configuration_profiles": [{"profile": "` + profileB64 + `"}]}`,
			badBodies: []string{
				`{}`,
				`{"configuration_profiles": null}`,
				// the body shape of /mdm/profiles/batch
				`{"profiles": [{"name": "N1", "contents": "` + profileB64 + `"}]}`,
			},
			emptyBodies: []string{`{"configuration_profiles": []}`},
		},
		{
			path:   "/api/latest/fleet/mdm/profiles/batch",
			field:  "profiles",
			upload: `{"profiles": [{"name": "N1", "contents": "` + profileB64 + `"}]}`,
			badBodies: []string{
				`{}`,
				`{"profiles": null}`,
				// the body shape of /configuration_profiles/batch
				`{"configuration_profiles": [{"profile": "` + profileB64 + `"}]}`,
			},
			// the deprecated name->contents map format clears with an empty object
			emptyBodies: []string{`{"profiles": []}`, `{"profiles": {}}`},
		},
	}
	for _, c := range cases {
		for _, empty := range c.emptyBodies {
			s.DoRaw("POST", c.path, []byte(c.upload), http.StatusNoContent, "team_id", teamID)
			assertProfileCount(1)

			for _, body := range c.badBodies {
				res := s.DoRaw("POST", c.path, []byte(body), http.StatusUnprocessableEntity, "team_id", teamID)
				require.Contains(t, extractServerErrorText(res.Body), c.field+" is required", body)
				assertProfileCount(1)

				s.DoRaw("POST", c.path, []byte(body), http.StatusUnprocessableEntity, "team_id", teamID, "dry_run", "true")
				assertProfileCount(1)
			}

			// An explicit empty list still clears the fleet.
			s.DoRaw("POST", c.path, []byte(empty), http.StatusNoContent, "team_id", teamID)
			assertProfileCount(0)
		}
	}
}

func (s *integrationMDMTestSuite) TestBatchSetMDMProfilesBackwardsCompat() {
	t := s.T()
	ctx := context.Background()

	// create a new team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "batch_set_mdm_profiles"})
	require.NoError(t, err)

	// apply an empty set to no-team
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": []any{}}, http.StatusNoContent)
	// Nothing changed, so no activity
	s.lastActivityOfTypeDoesNotMatch(
		fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`,
		0,
	)
	s.lastActivityOfTypeDoesNotMatch(
		fleet.ActivityTypeEditedWindowsProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`,
		0,
	)

	// apply to both team id and name
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": []any{}},
		http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID), "team_name", tm.Name)

	// invalid team name
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": []any{}},
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
		if p == mobileconfig.FleetCustomSettingsPayloadType {
			// FileVault options in the custom settings payload are checked in file_vault_options_test.go
			continue
		}
		res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
			"N1": mobileconfigForTestWithContent("N1", "I1", "II1", p, ""),
			"N3": syncMLForTest("./Foo/Bar"),
		}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))
		errMsg := extractServerErrorText(res.Body)
		switch p {
		case mobileconfig.FleetFileVaultPayloadType, mobileconfig.FleetRecoveryKeyEscrowPayloadType:
			assert.Contains(t, errMsg, mobileconfig.DiskEncryptionProfileRestrictionErrMsg)
		default:
			assert.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadType(s): %s", p))
		}
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
	assert.Contains(t, errMsg, syncml.DiskEncryptionProfileRestrictionErrMsg)

	// os updates
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
		"N1":                             mobileconfigForTest("N1", "I1"),
		syncml.FleetOSUpdateTargetLocURI: syncMLForTest(fmt.Sprintf(".%s/Foo", syncml.FleetOSUpdateTargetLocURI)),
		"N3":                             syncMLForTest("./Foo/Bar"),
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

	// reset the profiles for later dry_run
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

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
		fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name),
		0,
	)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedWindowsProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name),
		0,
	)
}

func (s *integrationMDMTestSuite) TestMDMBatchSetProfilesKeepsReservedNames() {
	t := s.T()
	ctx := context.Background()
	kv := redis_key_value.New(s.redisPool)
	// The Fleetd enroll secret profile only exists when mdm.windows_one_time_enroll_secrets is on, which this suite doesn't set.
	reservedWindowsNames := []string{servermdm.FleetWindowsOSUpdatesProfileName}

	checkMacProfs := func(teamID *uint, names ...string) {
		var count int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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
	require.NoError(t, ReconcileAppleProfilesBatched(ctx, s.ds, s.mdmCommander, kv, s.logger, 0, false))

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
				"minimum_version": "14.6.1"
			}
		}
	}`), http.StatusOK, &acResp)
	checkMacProfs(nil, servermdm.ListFleetReservedMacOSProfileNames()...)
	checkWinProfs(nil, reservedWindowsNames...)

	// batch set only windows profiles doesn't remove the reserved names
	newWinProfile := syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Replace", LocURI: "l1", Data: "d1"}})
	var testProfiles []fleet.MDMProfileBatchPayload
	testProfiles = append(testProfiles, fleet.MDMProfileBatchPayload{
		Name:     "n1",
		Contents: newWinProfile,
	})
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
	checkMacProfs(nil, servermdm.ListFleetReservedMacOSProfileNames()...)
	checkWinProfs(nil, append(reservedWindowsNames, "n1")...)

	// batch set windows and mac profiles doesn't remove the reserved names
	newMacProfile := mcBytesForTest("n2", "i2", uuid.NewString())
	testProfiles = append(testProfiles, fleet.MDMProfileBatchPayload{
		Name:     "n2",
		Contents: newMacProfile,
	})
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
	checkMacProfs(nil, append(servermdm.ListFleetReservedMacOSProfileNames(), "n2")...)
	checkWinProfs(nil, append(reservedWindowsNames, "n1")...)

	// batch set only mac profiles doesn't remove the reserved names
	testProfiles = []fleet.MDMProfileBatchPayload{{
		Name:     "n2",
		Contents: newMacProfile,
	}}
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
	checkMacProfs(nil, append(servermdm.ListFleetReservedMacOSProfileNames(), "n2")...)
	checkWinProfs(nil, reservedWindowsNames...)

	// create a team
	var tmResp teamResponse
	s.DoJSON("POST", "/api/v1/fleet/teams", map[string]string{"Name": t.Name()}, http.StatusOK, &tmResp)

	// edit team mdm config to turn on disk encryption and os updates
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tmResp.Team.ID), modifyTeamRequest{
		Name: new(t.Name()),
		MDM: &fleet.TeamPayloadMDM{
			EnableDiskEncryption: optjson.SetBool(true),
			WindowsUpdates: &fleet.WindowsUpdates{
				DeadlineDays:    optjson.SetInt(4),
				GracePeriodDays: optjson.SetInt(1),
			},
			MacOSUpdates: &fleet.AppleOSUpdateSettings{
				Deadline:       optjson.SetString("2023-12-31"),
				MinimumVersion: optjson.SetString("14.6.1"),
				UpdateNewHosts: optjson.SetBool(true),
			},
		},
	}, http.StatusOK, &teamResponse{})

	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/teams/%d", tmResp.Team.ID), nil, http.StatusOK, &tmResp)
	require.True(t, tmResp.Team.Config.MDM.EnableDiskEncryption)
	require.Equal(t, 4, tmResp.Team.Config.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 1, tmResp.Team.Config.MDM.WindowsUpdates.GracePeriodDays.Value)
	require.Equal(t, "2023-12-31", tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	require.Equal(t, "14.6.1", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.True(t, tmResp.Team.Config.MDM.MacOSUpdates.UpdateNewHosts.Value)

	require.NoError(t, ReconcileAppleProfilesBatched(ctx, s.ds, s.mdmCommander, kv, s.logger, 0, false))

	checkMacProfs(&tmResp.Team.ID, servermdm.ListFleetReservedMacOSProfileNames()...)
	checkWinProfs(&tmResp.Team.ID, reservedWindowsNames...)

	// batch set only windows profiles doesn't remove the reserved names
	var testTeamProfiles []fleet.MDMProfileBatchPayload
	testTeamProfiles = append(testTeamProfiles, fleet.MDMProfileBatchPayload{
		Name:     "n1",
		Contents: newWinProfile,
	})
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testTeamProfiles}, http.StatusNoContent,
		"team_id", fmt.Sprint(tmResp.Team.ID))
	checkMacProfs(&tmResp.Team.ID, servermdm.ListFleetReservedMacOSProfileNames()...)
	checkWinProfs(&tmResp.Team.ID, append(reservedWindowsNames, "n1")...)

	// batch set windows and mac profiles doesn't remove the reserved names
	testTeamProfiles = append(testTeamProfiles, fleet.MDMProfileBatchPayload{
		Name:     "n2",
		Contents: newMacProfile,
	})
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testTeamProfiles}, http.StatusNoContent,
		"team_id", fmt.Sprint(tmResp.Team.ID))
	checkMacProfs(&tmResp.Team.ID, append(servermdm.ListFleetReservedMacOSProfileNames(), "n2")...)
	checkWinProfs(&tmResp.Team.ID, append(reservedWindowsNames, "n1")...)

	// batch set only mac profiles doesn't remove the reserved names
	testTeamProfiles = []fleet.MDMProfileBatchPayload{{
		Name:     "n2",
		Contents: newMacProfile,
	}}
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testTeamProfiles}, http.StatusNoContent,
		"team_id", fmt.Sprint(tmResp.Team.ID))
	checkMacProfs(&tmResp.Team.ID, append(servermdm.ListFleetReservedMacOSProfileNames(), "n2")...)
	checkWinProfs(&tmResp.Team.ID, reservedWindowsNames...)
}

func (s *integrationMDMTestSuite) TestBatchResendMDMProfiles() {
	t := s.T()
	ctx := t.Context()
	s.setSkipWorkerJobs(t)

	// create a few hosts
	host1, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	host2, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	host3, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	// register a couple profiles for Apple and one for Windows
	profN1 := mobileconfigForTest("N1", "I1")
	profN2 := mobileconfigForTest("N2", "I2")
	profN3 := syncMLForTest("./Foo/N3")
	declN4 := declarationForTest("N4")
	batchRequest := batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: profN1},
		{Name: "N2", Contents: profN2},
		{Name: "N3", Contents: profN3},
		{Name: "N4", Contents: declN4},
	}}
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchRequest, http.StatusNoContent)

	// list the profiles to get the UUIDs
	var listResp listMDMConfigProfilesResponse
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", nil, http.StatusOK, &listResp)
	profNameToPayload := make(map[string]*fleet.MDMConfigProfilePayload)
	for _, prof := range listResp.Profiles {
		if len(prof.Checksum) == 0 {
			// not important, but must not be empty or it causes issues when forcing a status
			prof.Checksum = []byte("checksum")
		}
		profNameToPayload[prof.Name] = prof
	}

	// get status for non-existing profile
	s.Do("GET", fmt.Sprintf("/api/v1/fleet/configuration_profiles/%s/status", "ano-such-profile"), nil, http.StatusNotFound)
	s.Do("GET", fmt.Sprintf("/api/v1/fleet/configuration_profiles/%s/status", "wno-such-profile"), nil, http.StatusNotFound)
	s.Do("GET", fmt.Sprintf("/api/v1/fleet/configuration_profiles/%s/status", "dno-such-profile"), nil, http.StatusNotFound)
	s.Do("GET", fmt.Sprintf("/api/v1/fleet/configuration_profiles/%s/status", "zno-such-profile"), nil, http.StatusNotFound)

	// get status for existing profiles, all 0 counts
	for _, uuid := range []string{profNameToPayload["N1"].ProfileUUID, profNameToPayload["N2"].ProfileUUID, profNameToPayload["N3"].ProfileUUID, profNameToPayload["N4"].ProfileUUID} { // nolint:nilaway // every name looked up here was inserted from the list response above
		var statusResp getMDMConfigProfileStatusResponse
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/configuration_profiles/%s/status", uuid), getMDMConfigProfileStatusRequest{}, http.StatusOK, &statusResp)
		require.Equal(t, fleet.MDMConfigProfileStatus{}, statusResp.MDMConfigProfileStatus)
	}

	// try to batch-resend a non-existing profile
	batchReq := batchResendMDMProfileToHostsRequest{ProfileUUID: "zzzz"} // not a known prefix
	batchReq.Filters.ProfileStatus = string(fleet.MDMDeliveryFailed)
	s.Do("POST", "/api/v1/fleet/configuration_profiles/resend/batch", batchReq, http.StatusNotFound)
	batchReq = batchResendMDMProfileToHostsRequest{ProfileUUID: "azzzz"} // unknown Apple profile
	batchReq.Filters.ProfileStatus = string(fleet.MDMDeliveryFailed)
	s.Do("POST", "/api/v1/fleet/configuration_profiles/resend/batch", batchReq, http.StatusNotFound)
	batchReq = batchResendMDMProfileToHostsRequest{ProfileUUID: "wzzzz"} // unknown Windows profile
	batchReq.Filters.ProfileStatus = string(fleet.MDMDeliveryFailed)
	s.Do("POST", "/api/v1/fleet/configuration_profiles/resend/batch", batchReq, http.StatusNotFound)

	// batch-resend with an invalid filter
	batchReq = batchResendMDMProfileToHostsRequest{ProfileUUID: profNameToPayload["N1"].ProfileUUID}
	batchReq.Filters.ProfileStatus = string(fleet.MDMDeliveryPending)
	res := s.Do("POST", "/api/v1/fleet/configuration_profiles/resend/batch", batchReq, http.StatusBadRequest)
	msg := extractServerErrorText(res.Body)
	require.Contains(t, msg, "Invalid profile_status filter value, only 'failed' is currently supported.")

	// batch-resend with an Apple DDM
	batchReq = batchResendMDMProfileToHostsRequest{ProfileUUID: profNameToPayload["N4"].ProfileUUID}
	batchReq.Filters.ProfileStatus = string(fleet.MDMDeliveryFailed)
	res = s.Do("POST", "/api/v1/fleet/configuration_profiles/resend/batch", batchReq, http.StatusBadRequest)
	msg = extractServerErrorText(res.Body)
	require.Contains(t, msg, "Can't resend declaration (DDM) profiles.")

	// batch-resend an Apple and a Windows profile, does nothing as it is not delivered yet
	batchReq = batchResendMDMProfileToHostsRequest{ProfileUUID: profNameToPayload["N1"].ProfileUUID}
	batchReq.Filters.ProfileStatus = string(fleet.MDMDeliveryFailed)
	s.Do("POST", "/api/v1/fleet/configuration_profiles/resend/batch", batchReq, http.StatusAccepted)
	batchReq = batchResendMDMProfileToHostsRequest{ProfileUUID: profNameToPayload["N3"].ProfileUUID}
	batchReq.Filters.ProfileStatus = string(fleet.MDMDeliveryFailed)
	s.Do("POST", "/api/v1/fleet/configuration_profiles/resend/batch", batchReq, http.StatusAccepted)

	forceSetAppleHostProfileStatus(t, s.ds, host1.UUID, test.ToMDMAppleConfigProfile(profNameToPayload["N1"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryPending)
	forceSetAppleHostProfileStatus(t, s.ds, host1.UUID, test.ToMDMAppleConfigProfile(profNameToPayload["N2"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryPending)
	forceSetAppleHostProfileStatus(t, s.ds, host2.UUID, test.ToMDMAppleConfigProfile(profNameToPayload["N1"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryPending)
	forceSetAppleHostProfileStatus(t, s.ds, host2.UUID, test.ToMDMAppleConfigProfile(profNameToPayload["N2"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryPending)
	forceSetAppleHostDeclarationStatus(t, s.ds, host1.UUID, test.ToMDMAppleDecl(profNameToPayload["N4"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryPending)
	forceSetAppleHostDeclarationStatus(t, s.ds, host2.UUID, test.ToMDMAppleDecl(profNameToPayload["N4"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryPending)
	forceSetWindowsHostProfileStatus(t, s.ds, host3.UUID, test.ToMDMWindowsConfigProfile(profNameToPayload["N3"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryPending)

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		host1: {
			{Identifier: "I1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "I2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "N4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		host2: {
			{Identifier: "I1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "I2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "N4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		host3: {
			{Name: "N3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	var statusResp getMDMConfigProfileStatusResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/configuration_profiles/%s/status", profNameToPayload["N1"].ProfileUUID), getMDMConfigProfileStatusRequest{}, http.StatusOK, &statusResp)
	require.Equal(t, fleet.MDMConfigProfileStatus{Pending: 2}, statusResp.MDMConfigProfileStatus)
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/configuration_profiles/%s/status", profNameToPayload["N2"].ProfileUUID), getMDMConfigProfileStatusRequest{}, http.StatusOK, &statusResp)
	require.Equal(t, fleet.MDMConfigProfileStatus{Pending: 2}, statusResp.MDMConfigProfileStatus)
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/configuration_profiles/%s/status", profNameToPayload["N3"].ProfileUUID), getMDMConfigProfileStatusRequest{}, http.StatusOK, &statusResp)
	require.Equal(t, fleet.MDMConfigProfileStatus{Pending: 1}, statusResp.MDMConfigProfileStatus)
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/configuration_profiles/%s/status", profNameToPayload["N4"].ProfileUUID), getMDMConfigProfileStatusRequest{}, http.StatusOK, &statusResp)
	require.Equal(t, fleet.MDMConfigProfileStatus{Pending: 2}, statusResp.MDMConfigProfileStatus)

	// acknowledge the Apple profiles, failing I2 on both hosts, and fail the Windows one
	forceSetAppleHostProfileStatus(t, s.ds, host1.UUID, test.ToMDMAppleConfigProfile(profNameToPayload["N1"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, s.ds, host1.UUID, test.ToMDMAppleConfigProfile(profNameToPayload["N2"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryFailed)
	forceSetAppleHostProfileStatus(t, s.ds, host2.UUID, test.ToMDMAppleConfigProfile(profNameToPayload["N1"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, s.ds, host2.UUID, test.ToMDMAppleConfigProfile(profNameToPayload["N2"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryFailed)
	forceSetWindowsHostProfileStatus(t, s.ds, host3.UUID, test.ToMDMWindowsConfigProfile(profNameToPayload["N3"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryFailed)

	// batch-resend N2 profile
	batchReq = batchResendMDMProfileToHostsRequest{ProfileUUID: profNameToPayload["N2"].ProfileUUID}
	batchReq.Filters.ProfileStatus = string(fleet.MDMDeliveryFailed)
	s.Do("POST", "/api/v1/fleet/configuration_profiles/resend/batch", batchReq, http.StatusAccepted)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeResentConfigurationProfileBatch{}.ActivityName(),
		fmt.Sprintf(`{"profile_name": %q, "profile_uuid": %q, "host_count": %d}`, "N2", profNameToPayload["N2"].ProfileUUID, 2),
		0,
	)

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		host1: {
			{Identifier: "I1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "I2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "N4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		host2: {
			{Identifier: "I1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "I2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "N4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		host3: {
			{Name: "N3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryFailed},
		},
	})

	// set I2/N2 as verifying
	forceSetAppleHostProfileStatus(t, s.ds, host1.UUID, test.ToMDMAppleConfigProfile(profNameToPayload["N2"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, s.ds, host2.UUID, test.ToMDMAppleConfigProfile(profNameToPayload["N2"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)

	// batch-resend N3 profile
	batchReq = batchResendMDMProfileToHostsRequest{ProfileUUID: profNameToPayload["N3"].ProfileUUID}
	batchReq.Filters.ProfileStatus = string(fleet.MDMDeliveryFailed)
	s.Do("POST", "/api/v1/fleet/configuration_profiles/resend/batch", batchReq, http.StatusAccepted)

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		host1: {
			{Identifier: "I1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "I2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "N4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		host2: {
			{Identifier: "I1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "I2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "N4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		host3: {
			{Name: "N3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeResentConfigurationProfileBatch{}.ActivityName(),
		fmt.Sprintf(`{"profile_name": %q, "profile_uuid": %q, "host_count": %d}`, "N3", profNameToPayload["N3"].ProfileUUID, 1),
		0,
	)

	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/configuration_profiles/%s/status", profNameToPayload["N1"].ProfileUUID), getMDMConfigProfileStatusRequest{}, http.StatusOK, &statusResp)
	require.Equal(t, fleet.MDMConfigProfileStatus{Verifying: 2}, statusResp.MDMConfigProfileStatus)
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/configuration_profiles/%s/status", profNameToPayload["N2"].ProfileUUID), getMDMConfigProfileStatusRequest{}, http.StatusOK, &statusResp)
	require.Equal(t, fleet.MDMConfigProfileStatus{Verifying: 2}, statusResp.MDMConfigProfileStatus)
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/configuration_profiles/%s/status", profNameToPayload["N3"].ProfileUUID), getMDMConfigProfileStatusRequest{}, http.StatusOK, &statusResp)
	require.Equal(t, fleet.MDMConfigProfileStatus{Pending: 1}, statusResp.MDMConfigProfileStatus)
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/configuration_profiles/%s/status", profNameToPayload["N4"].ProfileUUID), getMDMConfigProfileStatusRequest{}, http.StatusOK, &statusResp)
	require.Equal(t, fleet.MDMConfigProfileStatus{Pending: 2}, statusResp.MDMConfigProfileStatus)

	// trigger profile schedule to get the fleet-controlled profiles
	s.awaitTriggerProfileSchedule(t)

	// list the profiles to get a fleet-controlled profile UUID
	gotProfs, err := s.ds.GetHostMDMAppleProfiles(ctx, host1.UUID)
	require.NoError(t, err)
	var fleetReservedProfile string
	for _, prof := range gotProfs {
		// find the fleetd config one
		if prof.Identifier == mobileconfig.FleetdConfigPayloadIdentifier {
			fleetReservedProfile = prof.ProfileUUID
		}
	}
	require.NotEmpty(t, fleetReservedProfile)
	// fleet-reserved profiles are not returned by the API, only custom profiles
	s.Do("GET", fmt.Sprintf("/api/v1/fleet/configuration_profiles/%s/status", fleetReservedProfile), getMDMConfigProfileStatusRequest{}, http.StatusNotFound)
}

// TestSpecTeamsOSUpdatesDeployToHosts verifies that POST /api/latest/fleet/spec/teams
// deploys OS updates declarations/profiles to enrolled Windows, iOS, iPadOS, and macOS
// hosts when OS update settings are initially set or when deadline/minimum version changes.
func (s *integrationMDMTestSuite) TestSpecTeamsOSUpdatesDeployToHosts() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := context.Background()

	// Create a team via the API so agent ops are initialized.
	teamName := t.Name() + "team1"
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{Name: teamName}, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	teamID := createTeamResp.Team.ID

	// Enroll one host per platform.
	macOSHost, macOSDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	windowsHost, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	iOSHost, iOSDevice := s.createAppleMobileHostThenEnrollMDM("ios")
	iPadOSHost, iPadOSDevice := s.createAppleMobileHostThenEnrollMDM("ipados")

	// Add all hosts to the team.
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &teamID, HostIDs: []uint{macOSHost.ID, windowsHost.ID, iOSHost.ID, iPadOSHost.ID}},
		http.StatusOK)

	// OS update declarations are scoped to platform-specific built-in labels:
	// - macOS: "macOS 14+ (Sonoma+)" (dynamic label)
	// - iOS:   "iOS" (manual label)
	// - iPadOS: "iPadOS" (manual label)
	// Add each host to the appropriate label so the declarations/profiles are queued.
	lblIDs, err := s.ds.LabelIDsByName(ctx,
		[]string{fleet.BuiltinLabelMacOS14Plus, fleet.BuiltinLabelIOS, fleet.BuiltinLabelIPadOS},
		fleet.TeamFilter{})
	require.NoError(t, err)

	// macOS 14+ is a dynamic label; use RecordLabelQueryExecutions to simulate osquery results.
	require.Contains(t, lblIDs, fleet.BuiltinLabelMacOS14Plus)
	require.NoError(t, s.ds.RecordLabelQueryExecutions(ctx, macOSHost,
		map[uint]*bool{lblIDs[fleet.BuiltinLabelMacOS14Plus]: new(true)}, time.Now(), false))

	// iOS and iPadOS are manual labels; insert label membership directly.
	require.Contains(t, lblIDs, fleet.BuiltinLabelIOS)
	require.Contains(t, lblIDs, fleet.BuiltinLabelIPadOS)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT IGNORE INTO label_membership (host_id, label_id) VALUES (?, ?), (?, ?)`,
			iOSHost.ID, lblIDs[fleet.BuiltinLabelIOS],
			iPadOSHost.ID, lblIDs[fleet.BuiltinLabelIPadOS])
		return err
	})

	// assertAppleDeclarationQueued verifies that the declaration row for a host
	// has been created with NULL status, meaning it has been queued for sync by
	// BulkSetPendingMDMHostProfiles but not yet processed by ReconcileAppleDeclarationsBatched.
	assertAppleDeclarationQueued := func(hostUUID, identifier string) {
		t.Helper()
		var count int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &count,
				`SELECT COUNT(*) FROM host_mdm_apple_declarations WHERE host_uuid = ? AND declaration_identifier = ?`,
				hostUUID, identifier)
		})
		require.Equal(t, 1, count, "expected declaration row to exist for host %s identifier %s", hostUUID, identifier)
	}

	// assertAppleDeclarationGone verifies that the declaration row for a host
	// has been removed from host_mdm_apple_declarations (i.e. the removal was
	// processed by MDMAppleStoreDDMStatusReport).
	assertAppleDeclarationGone := func(hostUUID, identifier string) {
		t.Helper()
		var count int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &count,
				`SELECT COUNT(*) FROM host_mdm_apple_declarations WHERE host_uuid = ? AND declaration_identifier = ?`,
				hostUUID, identifier)
		})
		require.Equal(t, 0, count, "expected declaration row to be gone for host %s identifier %s", hostUUID, identifier)
	}

	// assertAppleDeclarationVerifying verifies that the declaration for a host
	// is in the verifying state (device has acknowledged the DDM sync command).
	assertAppleDeclarationVerifying := func(hostUUID, identifier string) {
		t.Helper()
		var gotStatus *fleet.MDMDeliveryStatus
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &gotStatus,
				`SELECT status FROM host_mdm_apple_declarations WHERE host_uuid = ? AND declaration_identifier = ?`,
				hostUUID, identifier)
		})
		require.NotNil(t, gotStatus)
		require.Equal(t, fleet.MDMDeliveryVerifying, *gotStatus)
	}

	// checkWindowsProfileQueued verifies that the Windows OS updates profile
	// has been queued for the host (inserted into host_mdm_windows_profiles).
	checkWindowsProfileQueued := func(profUUID string) {
		t.Helper()
		var count int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &count,
				`SELECT COUNT(*) FROM host_mdm_windows_profiles WHERE host_uuid = ? AND profile_uuid = ?`,
				windowsHost.UUID, profUUID)
		})
		require.Equal(t, 1, count)
	}

	// ackAllCmds is used to ack commands sent during enrollment (like fleetd configuration).
	ackAllCmds := func(device *mdmtest.TestAppleMDMClient) {
		t.Helper()

		cmd, err := device.Idle()
		require.NoError(t, err)

		for cmd != nil {
			cmd, err = device.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
	}

	// checkDDMSync verifies that a DDM (DeclarativeManagement) sync command
	// is pending for the Apple device and acknowledges it.
	checkDDMSync := func(device *mdmtest.TestAppleMDMClient) {
		t.Helper()

		cmd, err := device.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd, "expected a DDM sync command to be pending (DeclarativeManagement)")
		require.Equal(t, "DeclarativeManagement", cmd.Command.RequestType, "%s", cmd.Raw)
		cmd, err = device.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
		require.Nil(t, cmd)
	}

	s.awaitTriggerProfileSchedule(t)

	// Acknowledge all commands to clear the command queue.
	ackAllCmds(macOSDevice)
	ackAllCmds(iOSDevice)
	ackAllCmds(iPadOSDevice)

	// OS update declaration identifiers used by Fleet (suffix is fixed in the service code).
	const (
		macOSUpdateIdent  = "macos-software-update-94f4bbdf-f439-4fb1-8d27-ae1bb793e105"
		iOSUpdateIdent    = "ios-software-update-94f4bbdf-f439-4fb1-8d27-ae1bb793e105"
		iPadOSUpdateIdent = "ipados-software-update-94f4bbdf-f439-4fb1-8d27-ae1bb793e105"
	)

	// -------------------------------------------------------------------------
	// Step 1: Set initial OS updates for all platforms via POST /spec/teams.
	// -------------------------------------------------------------------------
	var applyResp applyTeamSpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("14.6.1"),
				Deadline:       optjson.SetString("2024-03-01"),
			},
			IOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("17.6.1"),
				Deadline:       optjson.SetString("2024-03-01"),
			},
			IPadOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("17.6.1"),
				Deadline:       optjson.SetString("2024-03-01"),
			},
			WindowsUpdates: fleet.WindowsUpdates{
				DeadlineDays:    optjson.SetInt(5),
				GracePeriodDays: optjson.SetInt(1),
			},
		},
	}}}, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	// Verify that OS update declarations/profiles were created for the team.
	s.assertMacOSDeclarationsByName(&teamID, servermdm.FleetMacOSUpdatesProfileName, true)
	s.assertMacOSDeclarationsByName(&teamID, servermdm.FleetIOSUpdatesProfileName, true)
	s.assertMacOSDeclarationsByName(&teamID, servermdm.FleetIPadOSUpdatesProfileName, true)
	windowsOSUpdatesProfileUUID := checkWindowsOSUpdatesProfile(t, s.ds, &teamID,
		&fleet.WindowsUpdates{DeadlineDays: optjson.SetInt(5), GracePeriodDays: optjson.SetInt(1)})

	// Trigger the profile schedule so that:
	//   - ReconcileAppleDeclarationsBatched sends DDM sync commands to Apple hosts.
	//   - ReconcileWindowsProfiles queues the OS updates profile for the Windows host.
	s.awaitTriggerProfileSchedule(t)

	// Each Apple device should now have a pending DeclarativeManagement command.
	// Acknowledging the command transitions the declaration to "verifying".
	checkDDMSync(macOSDevice)
	assertAppleDeclarationVerifying(macOSHost.UUID, macOSUpdateIdent)

	checkDDMSync(iOSDevice)
	assertAppleDeclarationVerifying(iOSHost.UUID, iOSUpdateIdent)

	checkDDMSync(iPadOSDevice)
	assertAppleDeclarationVerifying(iPadOSHost.UUID, iPadOSUpdateIdent)

	// The Windows OS updates profile should be queued for the Windows host.
	checkWindowsProfileQueued(windowsOSUpdatesProfileUUID)

	// -------------------------------------------------------------------------
	// Step 2: Update deadline and minimum version via POST /spec/teams.
	// -------------------------------------------------------------------------
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("14.3.0"),
				Deadline:       optjson.SetString("2025-06-01"),
			},
			IOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("16.7.2"),
				Deadline:       optjson.SetString("2025-06-01"),
			},
			IPadOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("16.7.2"),
				Deadline:       optjson.SetString("2025-06-01"),
			},
			WindowsUpdates: fleet.WindowsUpdates{
				DeadlineDays:    optjson.SetInt(7),
				GracePeriodDays: optjson.SetInt(2),
			},
		},
	}}}, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	// Verify that the OS update declarations/profiles were updated for the team.
	s.assertMacOSDeclarationsByName(&teamID, servermdm.FleetMacOSUpdatesProfileName, true)
	s.assertMacOSDeclarationsByName(&teamID, servermdm.FleetIOSUpdatesProfileName, true)
	s.assertMacOSDeclarationsByName(&teamID, servermdm.FleetIPadOSUpdatesProfileName, true)
	checkWindowsOSUpdatesProfile(t, s.ds, &teamID,
		&fleet.WindowsUpdates{DeadlineDays: optjson.SetInt(7), GracePeriodDays: optjson.SetInt(2)})

	// The updated OS update settings should cause the Apple declarations to be
	// re-queued (status reset to NULL) for all enrolled hosts.
	assertAppleDeclarationQueued(macOSHost.UUID, macOSUpdateIdent)
	assertAppleDeclarationQueued(iOSHost.UUID, iOSUpdateIdent)
	assertAppleDeclarationQueued(iPadOSHost.UUID, iPadOSUpdateIdent)

	// Trigger the profile schedule again to reconcile the updated declarations
	// and the updated Windows profile.
	s.awaitTriggerProfileSchedule(t)

	// Each Apple device should receive another DDM sync command for the updated
	// declaration content, and transition back to "verifying" after acknowledging.
	checkDDMSync(macOSDevice)
	assertAppleDeclarationVerifying(macOSHost.UUID, macOSUpdateIdent)

	checkDDMSync(iOSDevice)
	assertAppleDeclarationVerifying(iOSHost.UUID, iOSUpdateIdent)

	checkDDMSync(iPadOSDevice)
	assertAppleDeclarationVerifying(iPadOSHost.UUID, iPadOSUpdateIdent)

	// The Windows OS updates profile (same UUID, updated content) should be
	// re-queued for the Windows host with the new deadline/grace period.
	checkWindowsProfileQueued(windowsOSUpdatesProfileUUID)

	// -------------------------------------------------------------------------
	// Step 3: Clear deadline and minimum version via POST /spec/teams.
	// -------------------------------------------------------------------------
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString(""),
				Deadline:       optjson.SetString(""),
			},
			IOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString(""),
				Deadline:       optjson.SetString(""),
			},
			IPadOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString(""),
				Deadline:       optjson.SetString(""),
			},
			WindowsUpdates: fleet.WindowsUpdates{
				DeadlineDays:    optjson.Int{},
				GracePeriodDays: optjson.Int{},
			},
		},
	}}}, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	// Verify that the OS update declarations/profiles were removed for the team.
	s.assertMacOSDeclarationsByName(&teamID, servermdm.FleetMacOSUpdatesProfileName, false)
	s.assertMacOSDeclarationsByName(&teamID, servermdm.FleetIOSUpdatesProfileName, false)
	s.assertMacOSDeclarationsByName(&teamID, servermdm.FleetIPadOSUpdatesProfileName, false)
	checkWindowsOSUpdatesProfile(t, s.ds, &teamID, nil)

	// The updated OS update settings should cause the Apple declarations to be
	// re-queued (status reset to NULL) for all enrolled hosts.
	assertAppleDeclarationQueued(macOSHost.UUID, macOSUpdateIdent)
	assertAppleDeclarationQueued(iOSHost.UUID, iOSUpdateIdent)
	assertAppleDeclarationQueued(iPadOSHost.UUID, iPadOSUpdateIdent)

	// Trigger the profile schedule again to reconcile the removed declarations.
	s.awaitTriggerProfileSchedule(t)

	// Each Apple device should receive a DDM sync command to remove the OS update
	// declaration. After acknowledging, the device sends an empty status report,
	// which causes Fleet to delete the pending remove rows via deletePendingRemovesStmt
	// in MDMAppleStoreDDMStatusReport.
	checkDDMSync(macOSDevice)
	resp, err := macOSDevice.DeclarativeManagement("status", fleet.MDMAppleDDMStatusReport{})
	require.NoError(t, err)
	resp.Body.Close()
	assertAppleDeclarationGone(macOSHost.UUID, macOSUpdateIdent)

	checkDDMSync(iOSDevice)
	resp, err = iOSDevice.DeclarativeManagement("status", fleet.MDMAppleDDMStatusReport{})
	require.NoError(t, err)
	resp.Body.Close()
	assertAppleDeclarationGone(iOSHost.UUID, iOSUpdateIdent)

	checkDDMSync(iPadOSDevice)
	resp, err = iPadOSDevice.DeclarativeManagement("status", fleet.MDMAppleDDMStatusReport{})
	require.NoError(t, err)
	resp.Body.Close()
	assertAppleDeclarationGone(iPadOSHost.UUID, iPadOSUpdateIdent)
}

func (s *integrationMDMTestSuite) TestBatchModifyConfigProfilesSelfServiceAndHidden() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	teamID := fmt.Sprint(team.ID)

	// raw JSON so the request isn't limited to the fields the Go struct knows about
	batch := func(profiles []map[string]any, wantStatus int) {
		s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", map[string]any{"configuration_profiles": profiles}, wantStatus, "team_id", teamID)
	}
	batch([]map[string]any{
		{"name": "SS", "profile": mobileconfigForTest("SS", "com.test.batchmodify.ss"), "self_service": true},
		{"name": "W", "profile": syncMLForTest("./Device/Vendor/MSFT/Policy/Config/BatchModifyW/Test"), "hidden": true},
		{"name": "D", "profile": declBytesForTest("com.test.batchmodify.decl", "d"), "hidden": true},
		{"name": "Plain", "profile": mobileconfigForTest("Plain", "com.test.batchmodify.plain")},
	}, http.StatusNoContent)

	var listResp listMDMConfigProfilesResponse
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", listMDMConfigProfilesRequest{}, http.StatusOK, &listResp, "team_id", teamID)
	require.Len(t, listResp.Profiles, 4)
	for _, p := range listResp.Profiles {
		require.Equal(t, p.Name == "SS", p.SelfService, p.Name)
		require.Equal(t, p.Name == "W" || p.Name == "D", p.Hidden, p.Name)
	}

	// validation applies here too
	batch([]map[string]any{
		{"name": "W", "profile": syncMLForTest("./Device/Vendor/MSFT/Policy/Config/BatchModifyW/Test"), "self_service": true},
	}, http.StatusUnprocessableEntity)
}
