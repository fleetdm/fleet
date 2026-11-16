package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Config and fleet responses leave the recorded lists out, so tests that
// check what a write recorded read it from the stored config.
func storedAppConfig(t *testing.T, ds fleet.Datastore) *fleet.AppConfig {
	t.Helper()
	appConfig, err := ds.AppConfig(t.Context())
	require.NoError(t, err)
	return appConfig
}

func storedTeam(t *testing.T, ds fleet.Datastore, teamID uint) *fleet.Team {
	t.Helper()
	team, err := ds.TeamWithExtras(t.Context(), teamID)
	require.NoError(t, err)
	return team
}

var recordedListKeysByObject = []struct {
	objectPath []string
	keys       []string
}{
	{objectPath: nil, keys: []string{"scripts"}},
	{objectPath: []string{"mdm"}, keys: []string{"volume_purchasing_program"}},
	{objectPath: []string{"mdm", "macos_settings"}, keys: []string{"custom_settings", "configuration_profiles"}},
	{objectPath: []string{"mdm", "apple_settings"}, keys: []string{"custom_settings", "configuration_profiles"}},
	{objectPath: []string{"mdm", "windows_settings"}, keys: []string{"custom_settings", "configuration_profiles"}},
	{objectPath: []string{"mdm", "android_settings"}, keys: []string{"custom_settings", "configuration_profiles"}},
	{objectPath: []string{"mdm", "setup_experience"}, keys: []string{"macos_bootstrap_package", "apple_setup_assistant", "macos_script"}},
}

func requireNoRecordedListKeys(t *testing.T, source string, config map[string]any) {
	t.Helper()

	walkedObjects := make(map[string]struct{})
	for _, recorded := range recordedListKeysByObject {
		object := config
		objectName := ""
		for _, segment := range recorded.objectPath {
			child, ok := object[segment].(map[string]any)
			if !ok {
				object = nil
				break
			}
			object = child
			objectName = segment
		}
		// Fleet responses only use the current object names, config responses use both.
		if object == nil {
			continue
		}
		walkedObjects[objectName] = struct{}{}
		for _, key := range recorded.keys {
			assert.NotContains(t, object, key, "%s: %v", source, recorded.objectPath)
		}
	}

	// Guards against the checks above passing because the objects were not found at all.
	require.Contains(t, walkedObjects, "apple_settings", source)
	require.Contains(t, walkedObjects, "windows_settings", source)
	require.Contains(t, walkedObjects, "android_settings", source)
	require.Contains(t, walkedObjects, "setup_experience", source)
	mdm := config["mdm"].(map[string]any)
	require.Contains(t, mdm["apple_settings"], "enable_disk_encryption", source)
	require.Contains(t, mdm["setup_experience"], "enable_end_user_authentication", source)
}

func (s *integrationMDMTestSuite) TestConfigAndFleetResponsesOmitRecordedLists() {
	t := s.T()
	ctx := t.Context()

	originalAppConfig, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	originalAppConfig = originalAppConfig.Copy()
	// The test context is already canceled when cleanups run.
	cleanupCtx := context.WithoutCancel(ctx)
	t.Cleanup(func() {
		require.NoError(t, s.ds.SaveAppConfig(cleanupCtx, originalAppConfig))
	})

	decodeObject := func(res *http.Response) map[string]any {
		defer res.Body.Close()
		var object map[string]any
		require.NoError(t, json.NewDecoder(res.Body).Decode(&object))
		return object
	}

	recordedLists := map[string]any{
		"scripts": []string{"script.sh"},
		"mdm": map[string]any{
			"macos_settings":   map[string]any{"custom_settings": []map[string]any{{"path": "apple.mobileconfig"}}},
			"windows_settings": map[string]any{"custom_settings": []map[string]any{{"path": "windows.xml"}}},
			"android_settings": map[string]any{"custom_settings": []map[string]any{{"path": "android.json"}}},
			"setup_experience": map[string]any{
				"apple_setup_assistant":   "assistant.json",
				"macos_bootstrap_package": "https://example.com/bootstrap.pkg",
			},
		},
	}
	// The server still records them, to compare the next GitOps run against.
	requireRecorded := func(scripts []string, mdm fleet.TeamMDM) {
		t.Helper()
		require.Equal(t, []string{"script.sh"}, scripts)
		require.Equal(t, []fleet.MDMProfileSpec{{Path: "apple.mobileconfig"}}, mdm.MacOSSettings.CustomSettings)
		require.Equal(t, []fleet.MDMProfileSpec{{Path: "windows.xml"}}, mdm.WindowsSettings.CustomSettings.Value)
		require.Equal(t, []fleet.MDMProfileSpec{{Path: "android.json"}}, mdm.AndroidSettings.CustomSettings.Value)
		require.Equal(t, "assistant.json", mdm.MacOSSetup.MacOSSetupAssistant.Value)
		require.Equal(t, "https://example.com/bootstrap.pkg", mdm.MacOSSetup.BootstrapPackage.Value)
	}

	patchConfig := decodeObject(s.Do("PATCH", "/api/latest/fleet/config", recordedLists, http.StatusOK))
	requireNoRecordedListKeys(t, "PATCH /config", patchConfig)
	requireNoRecordedListKeys(t, "GET /config", decodeObject(s.Do("GET", "/api/latest/fleet/config", nil, http.StatusOK)))

	appConfig := storedAppConfig(t, s.ds)
	requireRecorded(appConfig.Scripts.Value, fleet.TeamMDM{
		MacOSSettings:   appConfig.MDM.MacOSSettings,
		WindowsSettings: appConfig.MDM.WindowsSettings,
		AndroidSettings: appConfig.MDM.AndroidSettings,
		MacOSSetup:      appConfig.MDM.MacOSSetup,
	})

	teamName := t.Name() + "fleet"
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/fleets", &fleet.Team{Name: teamName}, http.StatusOK, &createTeamResp)
	teamID := createTeamResp.Team.ID
	require.NotZero(t, teamID)
	t.Cleanup(func() {
		require.NoError(t, s.ds.DeleteTeam(cleanupCtx, teamID))
	})

	teamSpec := maps.Clone(recordedLists)
	teamSpec["name"] = teamName
	s.Do("POST", "/api/latest/fleet/spec/fleets", map[string]any{"specs": []any{teamSpec}}, http.StatusOK)

	fleetPath := fmt.Sprintf("/api/latest/fleet/fleets/%d", teamID)
	getFleet := decodeObject(s.Do("GET", fleetPath, nil, http.StatusOK))
	requireNoRecordedListKeys(t, "GET /fleets/:id", getFleet["fleet"].(map[string]any))

	patchFleet := decodeObject(s.Do("PATCH", fleetPath, map[string]any{"description": "updated"}, http.StatusOK))
	requireNoRecordedListKeys(t, "PATCH /fleets/:id", patchFleet["fleet"].(map[string]any))

	listFleets := decodeObject(s.Do("GET", "/api/latest/fleet/fleets", nil, http.StatusOK))
	var listedFleet map[string]any
	for _, listed := range listFleets["fleets"].([]any) {
		if candidate := listed.(map[string]any); candidate["name"] == teamName {
			listedFleet = candidate
		}
	}
	require.NotNil(t, listedFleet)
	requireNoRecordedListKeys(t, "GET /fleets", listedFleet)

	// Target search returns each fleet's whole config.
	targets, err := io.ReadAll(s.Do("POST", "/api/latest/fleet/targets", map[string]any{"query": teamName}, http.StatusOK).Body)
	require.NoError(t, err)
	require.Contains(t, string(targets), teamName)
	for _, sent := range []string{"script.sh", "apple.mobileconfig", "windows.xml", "android.json", "assistant.json", "bootstrap.pkg"} {
		require.NotContains(t, string(targets), sent)
	}

	team := storedTeam(t, s.ds, teamID)
	requireRecorded(team.Config.Scripts.Value, team.Config.MDM)
}
