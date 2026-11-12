package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func storedAppConfig(t *testing.T, ds fleet.Datastore) *fleet.AppConfig {
	ac, err := ds.AppConfig(t.Context())
	require.NoError(t, err)
	return ac
}

func storedTeamConfig(t *testing.T, ds fleet.Datastore, teamID uint) fleet.TeamConfig {
	tm, err := ds.TeamWithExtras(t.Context(), teamID)
	require.NoError(t, err)
	return tm.Config
}

func (s *integrationMDMTestSuite) TestConfigAndFleetResponsesListSavedItems() {
	t := s.T()
	ctx := t.Context()

	var createResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/fleets", &fleet.Team{Name: t.Name()}, http.StatusOK, &createResp)
	tm := createResp.Team
	require.Equal(t, fleet.NewConfigLists().AppleProfiles, tm.Config.MDM.MacOSSettings.CustomSettings)
	require.Equal(t, []string{}, tm.Config.Scripts.Value)

	// YAML values that don't match anything saved: responses must not echo them.
	s.Do("POST", "/api/latest/fleet/spec/fleets", map[string]any{"specs": []any{map[string]any{
		"name":    tm.Name,
		"scripts": []string{"stale.sh"},
		"mdm": map[string]any{"macos_settings": map[string]any{
			"custom_settings": []map[string]any{{"path": "stale.mobileconfig"}},
		}},
	}}}, http.StatusOK)

	lbl, err := s.ds.NewLabel(ctx, &fleet.Label{Name: t.Name(), Query: "select 1"})
	require.NoError(t, err)
	teamField := map[string][]string{"team_id": {fmt.Sprint(tm.ID)}}

	body, headers := generateNewProfileMultipartRequest(t, "wifi.mobileconfig", mobileconfigForTest("Wi-Fi", t.Name()), s.token,
		map[string][]string{"team_id": {fmt.Sprint(tm.ID)}, "labels_include_all": {lbl.Name}})
	s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusOK, headers)
	body, headers = generateNewProfileMultipartRequest(t, "firewall.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Firewall"), s.token, teamField)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusOK, headers)

	body, headers = generateNewScriptMultipartRequest(t, "hello.sh", []byte(`echo "hello"`), s.token, teamField)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusOK, headers)
	body, headers = generateNewScriptMultipartRequest(t, "setup.sh", []byte(`echo "setup"`), s.token, map[string][]string{"fleet_id": {fmt.Sprint(tm.ID)}})
	s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusOK, headers)

	// The setup assistant endpoint needs Apple Business Manager, which this suite doesn't set up.
	_, err = s.ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{
		TeamID: &tm.ID, Name: "assistant", Profile: json.RawMessage(`{"x": 1}`),
	})
	require.NoError(t, err)
	pkg, err := os.ReadFile(filepath.Join("testdata", "bootstrap-packages", "signed.pkg"))
	require.NoError(t, err)
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: pkg, Name: "bootstrap.pkg", TeamID: tm.ID}, http.StatusOK, "", false)

	requireTeamLists := func(t *testing.T, got *fleet.Team) {
		t.Helper()
		require.Equal(t, []fleet.MDMProfileSpec{{Name: "Wi-Fi", LabelsIncludeAll: []string{lbl.Name}}}, got.Config.MDM.MacOSSettings.CustomSettings)
		require.Equal(t, []fleet.MDMProfileSpec{{Name: "firewall"}}, got.Config.MDM.WindowsSettings.CustomSettings.Value)
		require.Equal(t, []fleet.MDMProfileSpec{}, got.Config.MDM.AndroidSettings.CustomSettings.Value)
		require.Equal(t, []string{"hello.sh"}, got.Config.Scripts.Value)
		require.Equal(t, "bootstrap.pkg", got.Config.MDM.MacOSSetup.BootstrapPackage.Value)
		require.Equal(t, "assistant", got.Config.MDM.MacOSSetup.MacOSSetupAssistant.Value)
		require.Equal(t, "setup.sh", got.Config.MDM.MacOSSetup.Script.Value)
	}

	var getResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", tm.ID), nil, http.StatusOK, &getResp)
	requireTeamLists(t, getResp.Team)

	var listResp listTeamsResponse
	s.DoJSON("GET", "/api/latest/fleet/fleets", nil, http.StatusOK, &listResp)
	var listed *fleet.Team
	for i := range listResp.Teams {
		if listResp.Teams[i].ID == tm.ID {
			listed = &listResp.Teams[i]
		}
	}
	require.NotNil(t, listed)
	requireTeamLists(t, listed)

	var modifyResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", tm.ID), fleet.TeamPayload{Description: new("updated")}, http.StatusOK, &modifyResp)
	requireTeamLists(t, modifyResp.Team)

	// The stored YAML values are left untouched.
	stored := storedTeamConfig(t, s.ds, tm.ID)
	require.Equal(t, []string{"stale.sh"}, stored.Scripts.Value)
	require.Equal(t, []fleet.MDMProfileSpec{{Path: "stale.mobileconfig"}}, stored.MDM.MacOSSettings.CustomSettings)

	// "No fleet" lists, including VPP assignments, are in the config responses.
	body, headers = generateNewScriptMultipartRequest(t, "no-fleet.sh", []byte(`echo "no fleet"`), s.token, nil)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusOK, headers)

	tok := s.setVPPTokenForTeam(tm.ID)
	wantVPP := fleet.MDMAppleVolumePurchasingProgramInfo{Location: tok.Location, Teams: []string{tm.Name}}

	originalScripts := storedAppConfig(t, s.ds).Scripts
	t.Cleanup(func() {
		// t.Context() is already canceled when cleanups run.
		ctx := context.Background()
		ac, err := s.ds.AppConfig(ctx)
		require.NoError(t, err)
		ac.Scripts = originalScripts
		require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
	})
	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"scripts": ["stale.sh"]}`), http.StatusOK, &acResp)
	require.Contains(t, acResp.Scripts.Value, "no-fleet.sh")
	require.NotContains(t, acResp.Scripts.Value, "stale.sh")
	require.Contains(t, acResp.MDM.VolumePurchasingProgram.Value, wantVPP)

	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, acResp.Scripts.Value, "no-fleet.sh")
	require.NotContains(t, acResp.Scripts.Value, "stale.sh")
	require.Contains(t, acResp.MDM.VolumePurchasingProgram.Value, wantVPP)
	require.Equal(t, []string{"stale.sh"}, storedAppConfig(t, s.ds).Scripts.Value)
}

func (s *integrationMDMTestSuite) TestConfigAndFleetResponsesSentBack() {
	t := s.T()
	ctx := t.Context()

	var createResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/fleets", &fleet.Team{Name: t.Name()}, http.StatusOK, &createResp)
	tm := createResp.Team

	recordedGlobal := []fleet.MDMProfileSpec{{Path: "lib/a.mobileconfig"}}
	recordedTeam := []fleet.MDMProfileSpec{{Path: "lib/b.mobileconfig"}}
	s.Do("PATCH", "/api/latest/fleet/config", map[string]any{"mdm": map[string]any{
		"macos_settings": map[string]any{"custom_settings": recordedGlobal},
	}}, http.StatusOK)
	s.Do("POST", "/api/latest/fleet/spec/fleets", map[string]any{"specs": []any{map[string]any{
		"name": tm.Name,
		"mdm":  map[string]any{"macos_settings": map[string]any{"custom_settings": recordedTeam}},
	}}}, http.StatusOK)

	for i, teamID := range []uint{0, 0, tm.ID, tm.ID} {
		fields := map[string][]string{}
		if teamID != 0 {
			fields["team_id"] = []string{fmt.Sprint(teamID)}
		}
		name := fmt.Sprintf("%s %d", t.Name(), i)
		body, headers := generateNewProfileMultipartRequest(t, fmt.Sprintf("p%d.mobileconfig", i), mobileconfigForTest(name, name), s.token, fields)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusOK, headers)
	}

	var acResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Len(t, acResp.MDM.MacOSSettings.CustomSettings, 2)
	var getResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", tm.ID), nil, http.StatusOK, &getResp)
	require.Len(t, getResp.Team.Config.MDM.MacOSSettings.CustomSettings, 2)

	// With Apple MDM off, the server compares incoming profiles to the recorded
	// ones, which panicked on the name-only entries the responses return.
	ac := storedAppConfig(t, s.ds)
	ac.MDM.EnabledAndConfigured = false
	require.NoError(t, s.ds.SaveAppConfig(ctx, ac))

	s.Do("PATCH", "/api/latest/fleet/config", map[string]any{"mdm": map[string]any{
		"apple_settings":   map[string]any{"configuration_profiles": acResp.MDM.MacOSSettings.CustomSettings},
		"windows_settings": map[string]any{"configuration_profiles": acResp.MDM.WindowsSettings.CustomSettings.Value},
	}}, http.StatusOK)
	require.Equal(t, recordedGlobal, storedAppConfig(t, s.ds).MDM.MacOSSettings.CustomSettings)

	s.Do("POST", "/api/latest/fleet/spec/fleets", map[string]any{"specs": []any{map[string]any{
		"name": tm.Name,
		"mdm": map[string]any{"macos_settings": map[string]any{
			"custom_settings": getResp.Team.Config.MDM.MacOSSettings.CustomSettings,
		}},
	}}}, http.StatusOK)
	require.Equal(t, recordedTeam, storedTeamConfig(t, s.ds, tm.ID).MDM.MacOSSettings.CustomSettings)
}

func (s *integrationMDMTestSuite) TestConfigAndFleetResponsesListVisibility() {
	t := s.T()
	ctx := t.Context()

	var createResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/fleets", &fleet.Team{Name: t.Name()}, http.StatusOK, &createResp)
	tm := createResp.Team

	for _, teamID := range []*uint{nil, &tm.ID} {
		cp, err := fleet.NewMDMAppleConfigProfile(mobileconfigForTest(fmt.Sprintf("%s %v", t.Name(), teamID != nil), uuid.NewString()), teamID)
		require.NoError(t, err)
		_, err = s.ds.NewMDMAppleConfigProfile(ctx, *cp, nil)
		require.NoError(t, err)
		_, err = s.ds.NewScript(ctx, &fleet.Script{TeamID: teamID, Name: "visible.sh", ScriptContents: "echo"})
		require.NoError(t, err)
		_, err = s.ds.SetSetupExperienceScript(ctx, &fleet.Script{TeamID: teamID, Name: "setup.sh", ScriptContents: "echo"})
		require.NoError(t, err)
		_, err = s.ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{TeamID: teamID, Name: "assistant", Profile: json.RawMessage(`{}`)})
		require.NoError(t, err)
		require.NoError(t, s.ds.InsertMDMAppleBootstrapPackage(ctx, &fleet.MDMAppleBootstrapPackage{
			TeamID: ptr.ValOrZero(teamID), Name: "bootstrap.pkg", Token: uuid.NewString(), Bytes: []byte("pkg"), Sha256: []byte("sha"),
		}, nil))
	}
	t.Cleanup(func() {
		// t.Context() is already canceled when cleanups run.
		ctx := context.Background()
		require.NoError(t, s.ds.DeleteSetupExperienceScript(ctx, nil))
		require.NoError(t, s.ds.DeleteMDMAppleSetupAssistant(ctx, nil))
	})
	s.setVPPTokenForTeam(tm.ID)

	type view struct {
		AppleProfiles   []fleet.MDMProfileSpec                                    `json:"apple_profiles"`
		WindowsProfiles optjson.Slice[fleet.MDMProfileSpec]                       `json:"windows_profiles"`
		AndroidProfiles optjson.Slice[fleet.MDMProfileSpec]                       `json:"android_profiles"`
		Scripts         optjson.Slice[string]                                     `json:"scripts"`
		SetupScript     optjson.String                                            `json:"setup_script"`
		Bootstrap       optjson.String                                            `json:"bootstrap_package"`
		Assistant       optjson.String                                            `json:"setup_assistant"`
		VPP             *optjson.Slice[fleet.MDMAppleVolumePurchasingProgramInfo] `json:"vpp,omitempty"`
	}
	configView := func(ac fleet.AppConfig) view {
		return view{
			ac.MDM.MacOSSettings.CustomSettings, ac.MDM.WindowsSettings.CustomSettings, ac.MDM.AndroidSettings.CustomSettings,
			ac.Scripts, ac.MDM.MacOSSetup.Script, ac.MDM.MacOSSetup.BootstrapPackage, ac.MDM.MacOSSetup.MacOSSetupAssistant,
			&ac.MDM.VolumePurchasingProgram,
		}
	}
	fleetView := func(tc fleet.TeamConfig) view {
		return view{
			tc.MDM.MacOSSettings.CustomSettings, tc.MDM.WindowsSettings.CustomSettings, tc.MDM.AndroidSettings.CustomSettings,
			tc.Scripts, tc.MDM.MacOSSetup.Script, tc.MDM.MacOSSetup.BootstrapPackage, tc.MDM.MacOSSetup.MacOSSetupAssistant,
			nil,
		}
	}
	// Compared as JSON, where a hidden list is null.
	requireView := func(t *testing.T, want, got view) {
		t.Helper()
		wantJSON, err := json.Marshal(want)
		require.NoError(t, err)
		gotJSON, err := json.Marshal(got)
		require.NoError(t, err)
		require.JSONEq(t, string(wantJSON), string(gotJSON))
	}

	type lists struct{ profiles, scripts, bootstrap, assistant, vpp bool }
	visible := func(v view, can lists) view {
		if !can.profiles {
			v.AppleProfiles = nil
			v.WindowsProfiles = optjson.Slice[fleet.MDMProfileSpec]{Set: true}
			v.AndroidProfiles = optjson.Slice[fleet.MDMProfileSpec]{Set: true}
		}
		if !can.scripts {
			v.Scripts = optjson.Slice[string]{Set: true}
			v.SetupScript = optjson.String{Set: true}
		}
		if !can.bootstrap {
			v.Bootstrap = optjson.String{Set: true}
		}
		if !can.assistant {
			v.Assistant = optjson.String{Set: true}
		}
		if !can.vpp && v.VPP != nil {
			v.VPP = &optjson.Slice[fleet.MDMAppleVolumePurchasingProgramInfo]{Set: true}
		}
		return v
	}

	var adminConfig appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &adminConfig)
	noFleet := configView(adminConfig.AppConfig)
	var adminFleet getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", tm.ID), nil, http.StatusOK, &adminFleet)
	inFleet := fleetView(adminFleet.Team.Config)
	// Admins see every list filled in, so the checks below can't pass on empty lists.
	for _, v := range []view{noFleet, inFleet} {
		require.NotEmpty(t, v.AppleProfiles)
		require.Contains(t, v.Scripts.Value, "visible.sh")
		require.Equal(t, "setup.sh", v.SetupScript.Value)
		require.Equal(t, "bootstrap.pkg", v.Bootstrap.Value)
		require.Equal(t, "assistant", v.Assistant.Value)
	}
	require.NotEmpty(t, noFleet.VPP.Value)

	// The lists each role can read on the features' own endpoints, per
	// policy.rego. Fleet roles get the same on their fleets, and nothing for
	// "No fleet", which includes VPP assignments.
	all := lists{profiles: true, scripts: true, bootstrap: true, assistant: true, vpp: true}
	reads := map[string]lists{
		fleet.RoleObserver:     {scripts: true},
		fleet.RoleObserverPlus: {scripts: true},
		fleet.RoleTechnician:   {profiles: true, scripts: true, vpp: true},
		fleet.RoleMaintainer:   all,
		fleet.RoleGitOps:       {profiles: true, vpp: true},
		fleet.RoleAdmin:        all,
	}
	for role, can := range reads {
		for _, global := range []bool{true, false} {
			scope := "fleet"
			if global {
				scope = "global"
			}
			t.Run(scope+" "+role, func(t *testing.T) {
				email := fmt.Sprintf("lists-%s-%s@example.com", scope, role)
				u := &fleet.User{Name: email, Email: email}
				wantConfig, wantFleet := visible(noFleet, can), visible(inFleet, can)
				if global {
					u.GlobalRole = new(role)
				} else {
					u.Teams = []fleet.UserTeam{{Team: *tm, Role: role}}
					wantConfig = visible(noFleet, lists{})
				}
				require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
				_, err := s.ds.NewUser(ctx, u)
				require.NoError(t, err)
				s.setTokenForTest(t, email, test.GoodPassword)
				canWrite := role == fleet.RoleAdmin || role == fleet.RoleGitOps

				var acResp appConfigResponse
				s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
				requireView(t, wantConfig, configView(acResp.AppConfig))
				if global && canWrite {
					acResp = appConfigResponse{}
					s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{}`), http.StatusOK, &acResp)
					requireView(t, wantConfig, configView(acResp.AppConfig))
				}

				var listResp listTeamsResponse
				s.DoJSON("GET", "/api/latest/fleet/fleets", nil, http.StatusOK, &listResp)
				i := slices.IndexFunc(listResp.Teams, func(lt fleet.Team) bool { return lt.ID == tm.ID })
				require.GreaterOrEqual(t, i, 0)
				requireView(t, wantFleet, fleetView(listResp.Teams[i].Config))

				// GitOps users can list and write fleets but not read one.
				if role != fleet.RoleGitOps {
					var getResp getTeamResponse
					s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", tm.ID), nil, http.StatusOK, &getResp)
					requireView(t, wantFleet, fleetView(getResp.Team.Config))
				}
				if canWrite {
					var modifyResp teamResponse
					s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", tm.ID), fleet.TeamPayload{Description: new(email)}, http.StatusOK, &modifyResp)
					requireView(t, wantFleet, fleetView(modifyResp.Team.Config))
				}
			})
		}
	}
}
