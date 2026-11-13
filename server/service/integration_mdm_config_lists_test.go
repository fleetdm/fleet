package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

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

	modifyResp = teamResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/agent_options", tm.ID),
		json.RawMessage(`{"config": {"options": {"aws_debug": true}}}`), http.StatusOK, &modifyResp)
	requireTeamLists(t, modifyResp.Team)

	member := &fleet.User{Name: t.Name(), Email: "lists-saved-member@example.com", Teams: []fleet.UserTeam{{Team: *tm, Role: fleet.RoleObserver}}}
	require.NoError(t, member.SetPassword(test.GoodPassword, 10, 10))
	member, err = s.ds.NewUser(ctx, member)
	require.NoError(t, err)
	usersReq := modifyTeamUsersRequest{Users: []fleet.TeamUser{{ID: member.ID, Role: fleet.RoleMaintainer}}}
	for _, method := range []string{"PATCH", "DELETE"} {
		modifyResp = teamResponse{}
		s.DoJSON(method, fmt.Sprintf("/api/latest/fleet/fleets/%d/users", tm.ID), usersReq, http.StatusOK, &modifyResp)
		requireTeamLists(t, modifyResp.Team)
	}

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
		ctx := context.WithoutCancel(t.Context())
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
	recordedGlobalWindows := []fleet.MDMProfileSpec{{Path: "lib/a.xml"}}
	recordedGlobalAndroid := []fleet.MDMProfileSpec{{Path: "lib/a.json"}}
	recordedTeam := []fleet.MDMProfileSpec{{Path: "lib/b.mobileconfig"}}
	original := storedAppConfig(t, s.ds).MDM
	t.Cleanup(func() {
		// t.Context() is already canceled when cleanups run.
		ctx := context.WithoutCancel(t.Context())
		ac, err := s.ds.AppConfig(ctx)
		require.NoError(t, err)
		ac.MDM.MacOSSettings.CustomSettings = original.MacOSSettings.CustomSettings
		ac.MDM.WindowsSettings.CustomSettings = original.WindowsSettings.CustomSettings
		ac.MDM.AndroidSettings.CustomSettings = original.AndroidSettings.CustomSettings
		require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
	})
	s.Do("PATCH", "/api/latest/fleet/config", map[string]any{"mdm": map[string]any{
		"macos_settings":   map[string]any{"custom_settings": recordedGlobal},
		"windows_settings": map[string]any{"custom_settings": recordedGlobalWindows},
		"android_settings": map[string]any{"custom_settings": recordedGlobalAndroid},
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
	for i := range 2 {
		body, headers := generateNewProfileMultipartRequest(t, fmt.Sprintf("w%d.xml", i),
			syncMLForTest(fmt.Sprintf("./Device/Vendor/MSFT/Policy/Config/SentBack%d", i)), s.token, nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusOK, headers)
		_, err := s.ds.NewMDMAndroidConfigProfile(ctx, fleet.MDMAndroidConfigProfile{Name: fmt.Sprintf("Android %d", i), RawJSON: []byte(`{}`)}, nil)
		require.NoError(t, err)
	}

	var acResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Len(t, acResp.MDM.MacOSSettings.CustomSettings, 2)
	require.Len(t, acResp.MDM.WindowsSettings.CustomSettings.Value, 2)
	require.Len(t, acResp.MDM.AndroidSettings.CustomSettings.Value, 2)
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
		"android_settings": map[string]any{"configuration_profiles": acResp.MDM.AndroidSettings.CustomSettings.Value},
	}}, http.StatusOK)
	stored := storedAppConfig(t, s.ds).MDM
	require.Equal(t, recordedGlobal, stored.MacOSSettings.CustomSettings)
	require.Equal(t, recordedGlobalWindows, stored.WindowsSettings.CustomSettings.Value)
	require.Equal(t, recordedGlobalAndroid, stored.AndroidSettings.CustomSettings.Value)

	s.Do("POST", "/api/latest/fleet/spec/fleets", map[string]any{"specs": []any{map[string]any{
		"name": tm.Name,
		"mdm": map[string]any{"macos_settings": map[string]any{
			"custom_settings": getResp.Team.Config.MDM.MacOSSettings.CustomSettings,
		}},
	}}}, http.StatusOK)
	require.Equal(t, recordedTeam, storedTeamConfig(t, s.ds, tm.ID).MDM.MacOSSettings.CustomSettings)

	// Validation rewrites the deprecated "labels" key, so it must run on the
	// entries that are kept, not on the spec's originals.
	newName := t.Name() + " new"
	s.Do("POST", "/api/latest/fleet/spec/fleets", map[string]any{"specs": []any{map[string]any{
		"name": newName,
		"mdm": map[string]any{"windows_settings": map[string]any{"custom_settings": []map[string]any{
			{"name": "echoed"}, {"path": "lib/w.xml", "labels": []string{"lbl"}},
		}}},
	}}}, http.StatusOK)
	newTm, err := s.ds.TeamByName(ctx, newName)
	require.NoError(t, err)
	require.Equal(t, []fleet.MDMProfileSpec{{Path: "lib/w.xml", LabelsIncludeAll: []string{"lbl"}}},
		storedTeamConfig(t, s.ds, newTm.ID).MDM.WindowsSettings.CustomSettings.Value)
}

// GitOps users can write config, fleets, bootstrap packages and setup
// assistants but can't read the last two, so responses leave them out. Sending
// a response back, or null, must keep them.
func (s *integrationMDMTestSuite) TestConfigAndFleetResponsesSentBackAsGitOps() {
	t := s.T()
	ctx := t.Context()

	var createResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/fleets", &fleet.Team{Name: t.Name()}, http.StatusOK, &createResp)
	tm := createResp.Team

	// Overwrite mode below also resets settings such as features, so restore the whole config.
	original := storedAppConfig(t, s.ds).Copy()
	t.Cleanup(func() {
		// t.Context() is already canceled when cleanups run.
		ctx := context.WithoutCancel(t.Context())
		if err := s.ds.DeleteMDMAppleBootstrapPackage(ctx, 0); err != nil && !fleet.IsNotFound(err) {
			require.NoError(t, err)
		}
		if err := s.ds.DeleteMDMAppleSetupAssistant(ctx, nil); err != nil && !fleet.IsNotFound(err) {
			require.NoError(t, err)
		}
		require.NoError(t, s.ds.SaveAppConfig(ctx, original))
	})

	pkg, err := os.ReadFile(filepath.Join("testdata", "bootstrap-packages", "signed.pkg"))
	require.NoError(t, err)
	for _, teamID := range []uint{0, tm.ID} {
		s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: pkg, Name: "bootstrap.pkg", TeamID: teamID}, http.StatusOK, "", false)
		_, err = s.ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{
			TeamID: ptr.UintOrNilIfZero(teamID), Name: "assistant.json", Profile: json.RawMessage(`{"x": 1}`),
		})
		require.NoError(t, err)
	}

	applySetup := func(teamID uint, setup any, configQuery ...string) {
		t.Helper()
		if teamID == 0 {
			s.Do("PATCH", "/api/latest/fleet/config", map[string]any{"mdm": map[string]any{"macos_setup": setup}}, http.StatusOK, configQuery...)
			return
		}
		s.Do("POST", "/api/latest/fleet/spec/fleets", map[string]any{"specs": []any{map[string]any{
			"name": tm.Name,
			"mdm":  map[string]any{"macos_setup": setup},
		}}}, http.StatusOK)
	}
	recordedSetup := func(teamID uint) fleet.MacOSSetup {
		if teamID == 0 {
			return storedAppConfig(t, s.ds).MDM.MacOSSetup
		}
		return storedTeamConfig(t, s.ds, teamID).MDM.MacOSSetup
	}
	requireSaved := func(teamID uint, want bool) {
		t.Helper()
		_, err := s.ds.GetMDMAppleBootstrapPackageMeta(ctx, teamID)
		if want {
			require.NoError(t, err, "bootstrap package, fleet %d", teamID)
		} else {
			require.True(t, fleet.IsNotFound(err), "bootstrap package, fleet %d: %v", teamID, err)
		}
		_, err = s.ds.GetMDMAppleSetupAssistant(ctx, ptr.UintOrNilIfZero(teamID))
		if want {
			require.NoError(t, err, "setup assistant, fleet %d", teamID)
		} else {
			require.True(t, fleet.IsNotFound(err), "setup assistant, fleet %d: %v", teamID, err)
		}
	}

	// What a GitOps run records in the config.
	recorded := map[string]any{
		"bootstrap_package":     "https://example.com/bootstrap.pkg",
		"macos_setup_assistant": "lib/assistant.json",
	}
	for _, teamID := range []uint{0, tm.ID} {
		applySetup(teamID, recorded)
		requireSaved(teamID, true)
	}

	email := "lists-sent-back-gitops@example.com"
	u := &fleet.User{Name: email, Email: email, GlobalRole: new(fleet.RoleGitOps)}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(ctx, u)
	require.NoError(t, err)
	s.setTokenForTest(t, email, test.GoodPassword)

	// setupFrom checks that both names of each key are left out, and returns
	// the object under the deprecated name, which is what older clients send.
	setupFrom := func(mdm map[string]any) map[string]any {
		t.Helper()
		setup, ok := mdm["macos_setup"].(map[string]any)
		require.True(t, ok)
		require.NotContains(t, setup, "bootstrap_package")
		require.NotContains(t, setup, "macos_setup_assistant")
		renamed, ok := mdm["setup_experience"].(map[string]any)
		require.True(t, ok)
		require.NotContains(t, renamed, "macos_bootstrap_package")
		require.NotContains(t, renamed, "apple_setup_assistant")
		return setup
	}
	requireKept := func(teamID uint, setup map[string]any, configQuery ...string) {
		t.Helper()
		applySetup(teamID, setup, configQuery...)
		requireSaved(teamID, true)
		stored := recordedSetup(teamID)
		require.Equal(t, recorded["bootstrap_package"], stored.BootstrapPackage.Value)
		require.Equal(t, recorded["macos_setup_assistant"], stored.MacOSSetupAssistant.Value)
	}
	nulls := map[string]any{"bootstrap_package": nil, "macos_setup_assistant": nil}

	var cfg map[string]any
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &cfg)
	configSetup := setupFrom(cfg["mdm"].(map[string]any))
	for _, setup := range []map[string]any{configSetup, nulls} {
		requireKept(0, setup)
		requireKept(0, setup, "overwrite", "true")
	}

	var list map[string]any
	s.DoJSON("GET", "/api/latest/fleet/fleets", nil, http.StatusOK, &list)
	var fleetSetup map[string]any
	for _, f := range list["fleets"].([]any) {
		if fm := f.(map[string]any); fm["name"] == tm.Name {
			fleetSetup = setupFrom(fm["mdm"].(map[string]any))
		}
	}
	require.NotNil(t, fleetSetup)
	requireKept(tm.ID, fleetSetup)
	requireKept(tm.ID, nulls)

	// "" still deletes them, including in overwrite mode, which GitOps uses.
	cleared := map[string]any{"bootstrap_package": "", "macos_setup_assistant": ""}
	applySetup(0, cleared, "overwrite", "true")
	requireSaved(0, false)
	applySetup(tm.ID, cleared)
	requireSaved(tm.ID, false)
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
		ctx := context.WithoutCancel(t.Context())
		require.NoError(t, s.ds.DeleteSetupExperienceScript(ctx, nil))
		require.NoError(t, s.ds.DeleteMDMAppleSetupAssistant(ctx, nil))
	})
	s.setVPPTokenForTeam(tm.ID)

	// Each list's JSON value by its path in a config or fleet response, under
	// both the current and the deprecated key names. A hidden list has no entry.
	type view map[string]any
	listGroup := map[string]string{
		"custom_settings": "profiles", "configuration_profiles": "profiles",
		"scripts": "scripts", "script": "scripts", "macos_script": "scripts",
		"bootstrap_package": "bootstrap", "macos_bootstrap_package": "bootstrap",
		"macos_setup_assistant": "assistant", "apple_setup_assistant": "assistant",
		"volume_purchasing_program": "vpp",
	}
	viewOf := func(t *testing.T, obj map[string]any) view {
		t.Helper()
		v := view{}
		if scripts, ok := obj["scripts"]; ok {
			v["scripts"] = scripts
		}
		mdm, ok := obj["mdm"].(map[string]any)
		require.True(t, ok)
		if vpp, ok := mdm["volume_purchasing_program"]; ok {
			v["mdm.volume_purchasing_program"] = vpp
		}
		for _, parent := range []string{"macos_settings", "apple_settings", "windows_settings", "android_settings", "macos_setup", "setup_experience"} {
			sub, _ := mdm[parent].(map[string]any)
			for k, val := range sub {
				if _, isList := listGroup[k]; isList {
					v["mdm."+parent+"."+k] = val
				}
			}
		}
		return v
	}
	fleetIn := func(t *testing.T, obj map[string]any) map[string]any {
		t.Helper()
		fl, ok := obj["fleet"].(map[string]any)
		require.True(t, ok)
		return fl
	}

	type lists map[string]struct{}
	visible := func(v view, can lists) view {
		shown := view{}
		for path, val := range v {
			if _, ok := can[listGroup[path[strings.LastIndex(path, ".")+1:]]]; ok {
				shown[path] = val
			}
		}
		return shown
	}

	var adminConfig map[string]any
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &adminConfig)
	noFleet := viewOf(t, adminConfig)
	var adminFleet map[string]any
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", tm.ID), nil, http.StatusOK, &adminFleet)
	inFleet := viewOf(t, fleetIn(t, adminFleet))
	// Admins see every list filled in, under both key names, so the checks
	// below can't pass on empty or missing lists.
	require.Len(t, noFleet, 14)
	require.Len(t, inFleet, 13)
	for _, v := range []view{noFleet, inFleet} {
		require.NotEmpty(t, v["mdm.apple_settings.configuration_profiles"])
		require.Contains(t, v["scripts"], "visible.sh")
		require.Equal(t, "setup.sh", v["mdm.setup_experience.macos_script"])
		require.Equal(t, "bootstrap.pkg", v["mdm.setup_experience.macos_bootstrap_package"])
		require.Equal(t, "assistant", v["mdm.setup_experience.apple_setup_assistant"])
	}
	require.NotEmpty(t, noFleet["mdm.volume_purchasing_program"])

	// The lists each role can read on the features' own endpoints, per
	// policy.rego. Fleet roles get the same on their fleets, and nothing for
	// "No fleet", which includes VPP assignments.
	all := lists{"profiles": {}, "scripts": {}, "bootstrap": {}, "assistant": {}, "vpp": {}}
	reads := map[string]lists{
		fleet.RoleObserver:     {"scripts": {}},
		fleet.RoleObserverPlus: {"scripts": {}},
		fleet.RoleTechnician:   {"profiles": {}, "scripts": {}, "vpp": {}},
		fleet.RoleMaintainer:   all,
		fleet.RoleGitOps:       {"profiles": {}, "vpp": {}},
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

				var cfg map[string]any
				s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &cfg)
				require.Equal(t, wantConfig, viewOf(t, cfg))
				if global && canWrite {
					cfg = nil
					s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{}`), http.StatusOK, &cfg)
					require.Equal(t, wantConfig, viewOf(t, cfg))
				}

				var list map[string]any
				s.DoJSON("GET", "/api/latest/fleet/fleets", nil, http.StatusOK, &list)
				fleets, _ := list["fleets"].([]any)
				i := slices.IndexFunc(fleets, func(f any) bool { return f.(map[string]any)["id"] == float64(tm.ID) })
				require.GreaterOrEqual(t, i, 0)
				require.Equal(t, wantFleet, viewOf(t, fleets[i].(map[string]any)))

				// GitOps users can list and write fleets but not read one.
				if role != fleet.RoleGitOps {
					var getResp map[string]any
					s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", tm.ID), nil, http.StatusOK, &getResp)
					require.Equal(t, wantFleet, viewOf(t, fleetIn(t, getResp)))
				}
				if canWrite {
					var modifyResp map[string]any
					s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", tm.ID), fleet.TeamPayload{Description: new(email)}, http.StatusOK, &modifyResp)
					require.Equal(t, wantFleet, viewOf(t, fleetIn(t, modifyResp)))
				}
			})
		}
	}
}
