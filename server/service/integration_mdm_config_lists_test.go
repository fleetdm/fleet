package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The config and fleet settings keys that versions before Fleet 5 saved.
var formerlySavedKeyPaths = []string{
	"$.scripts",
	"$.mdm.volume_purchasing_program",
	"$.mdm.macos_settings.custom_settings",
	"$.mdm.windows_settings.custom_settings",
	"$.mdm.android_settings.custom_settings",
	"$.mdm.setup_experience.macos_bootstrap_package",
	"$.mdm.setup_experience.apple_setup_assistant",
	"$.mdm.setup_experience.macos_script",
}

var formerlySavedKeysByObject = []struct {
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

func requireNoFormerlySavedKeys(t *testing.T, source string, config map[string]any) {
	t.Helper()

	walkedObjects := make(map[string]struct{})
	for _, applied := range formerlySavedKeysByObject {
		object := config
		objectName := ""
		for _, segment := range applied.objectPath {
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
		for _, key := range applied.keys {
			assert.NotContains(t, object, key, "%s: %v", source, applied.objectPath)
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

func (s *integrationMDMTestSuite) TestConfigAndFleetSettingsLeaveOutFormerlySavedKeys() {
	t := s.T()
	ctx := t.Context()
	// The test context is already canceled when cleanups run.
	cleanupCtx := context.WithoutCancel(ctx)

	decodeObject := func(res *http.Response) map[string]any {
		defer res.Body.Close()
		var object map[string]any
		require.NoError(t, json.NewDecoder(res.Body).Decode(&object))
		return object
	}
	// What earlier versions saved after a GitOps run.
	seedSavedCopies := func(table, column, where string, args ...any) {
		t.Helper()
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET %s = JSON_SET(%s,
				'$.scripts', JSON_ARRAY('old-script.sh'),
				'$.mdm.volume_purchasing_program', JSON_ARRAY(),
				'$.mdm.macos_settings.custom_settings', JSON_ARRAY(JSON_OBJECT('path', 'old-apple.mobileconfig')),
				'$.mdm.windows_settings.custom_settings', JSON_ARRAY(JSON_OBJECT('path', 'old-windows.xml')),
				'$.mdm.android_settings.custom_settings', JSON_ARRAY(JSON_OBJECT('path', 'old-android.json')),
				'$.mdm.setup_experience.macos_bootstrap_package', 'https://example.com/old-bootstrap.pkg',
				'$.mdm.setup_experience.apple_setup_assistant', 'old-assistant.json',
				'$.mdm.setup_experience.macos_script', 'old-script.sh'
			) %s`, table, column, column, where), args...)
			return err
		})
	}
	savedCopies := func(table, column, where string, args ...any) []string {
		t.Helper()
		var saved []string
		for _, path := range formerlySavedKeyPaths {
			var found bool
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				return sqlx.GetContext(ctx, q, &found,
					fmt.Sprintf(`SELECT JSON_CONTAINS_PATH(%s, 'one', ?) FROM %s %s`, column, table, where), append([]any{path}, args...)...)
			})
			if found {
				saved = append(saved, path)
			}
		}
		return saved
	}
	requireRejected := func(method, path, body, key string) {
		t.Helper()
		res := s.Do(method, path, json.RawMessage(body), http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(res.Body), fmt.Sprintf("unsupported key provided: %q", key))
	}

	var originalConfig string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &originalConfig, "SELECT json_value FROM app_config_json")
	})
	t.Cleanup(func() {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(cleanupCtx, "UPDATE app_config_json SET json_value = ?", originalConfig)
			return err
		})
	})

	seedSavedCopies("app_config_json", "json_value", "")
	require.Len(t, savedCopies("app_config_json", "json_value", ""), len(formerlySavedKeyPaths))
	requireNoFormerlySavedKeys(t, "GET /config", decodeObject(s.Do("GET", "/api/latest/fleet/config", nil, http.StatusOK)))

	// The config doesn't keep the bootstrap package URL, and saving drops the earlier copies.
	patchConfig := decodeObject(s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {"setup_experience": {"macos_bootstrap_package": "https://example.com/bootstrap.pkg"}}
	}`), http.StatusOK))
	requireNoFormerlySavedKeys(t, "PATCH /config", patchConfig)
	require.Empty(t, savedCopies("app_config_json", "json_value", ""))

	for _, c := range []struct{ body, key string }{
		{`{"scripts": ["script.sh"]}`, "scripts"},
		{`{"mdm": {"macos_settings": {"custom_settings": [{"path": "apple.mobileconfig"}]}}}`, "custom_settings"},
		{`{"mdm": {"windows_settings": {"custom_settings": [{"path": "windows.xml"}]}}}`, "custom_settings"},
		{`{"mdm": {"android_settings": {"custom_settings": [{"path": "android.json"}]}}}`, "custom_settings"},
		{`{"mdm": {"setup_experience": {"apple_setup_assistant": "assistant.json"}}}`, "apple_setup_assistant"},
		{`{"mdm": {"setup_experience": {"macos_script": "script.sh"}}}`, "macos_script"},
	} {
		requireRejected("PATCH", "/api/latest/fleet/config", c.body, c.key)
	}

	teamName := t.Name() + "fleet"
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/fleets", &fleet.Team{Name: teamName}, http.StatusOK, &createTeamResp)
	teamID := createTeamResp.Team.ID
	require.NotZero(t, teamID)
	t.Cleanup(func() {
		require.NoError(t, s.ds.DeleteTeam(cleanupCtx, teamID))
	})

	seedSavedCopies("teams", "config", "WHERE id = ?", teamID)
	require.Len(t, savedCopies("teams", "config", "WHERE id = ?", teamID), len(formerlySavedKeyPaths))

	fleetPath := fmt.Sprintf("/api/latest/fleet/fleets/%d", teamID)
	getFleet := decodeObject(s.Do("GET", fleetPath, nil, http.StatusOK))
	requireNoFormerlySavedKeys(t, "GET /fleets/:id", getFleet["fleet"].(map[string]any))

	s.Do("POST", "/api/latest/fleet/spec/fleets", json.RawMessage(fmt.Sprintf(`{"specs": [{
		"name": %q,
		"mdm": {"setup_experience": {"macos_bootstrap_package": "https://example.com/bootstrap.pkg"}}
	}]}`, teamName)), http.StatusOK)
	require.Empty(t, savedCopies("teams", "config", "WHERE id = ?", teamID))

	for _, c := range []struct{ spec, key string }{
		{`"scripts": ["script.sh"]`, "scripts"},
		{`"mdm": {"macos_settings": {"custom_settings": [{"path": "apple.mobileconfig"}]}}`, "custom_settings"},
		{`"mdm": {"windows_settings": {"custom_settings": [{"path": "windows.xml"}]}}`, "custom_settings"},
		{`"mdm": {"android_settings": {"custom_settings": [{"path": "android.json"}]}}`, "custom_settings"},
		{`"mdm": {"setup_experience": {"apple_setup_assistant": "assistant.json"}}`, "apple_setup_assistant"},
		{`"mdm": {"setup_experience": {"macos_script": "script.sh"}}`, "macos_script"},
	} {
		requireRejected("POST", "/api/latest/fleet/spec/fleets", fmt.Sprintf(`{"specs": [{"name": %q, %s}]}`, teamName, c.spec), c.key)
	}
	// The fleet PATCH endpoint ignores unknown keys, as it already did for the bootstrap package.
	s.Do("PATCH", fleetPath, json.RawMessage(`{"mdm": {"setup_experience": {"macos_bootstrap_package": "https://example.com/bootstrap.pkg"}}}`),
		http.StatusOK)
	require.Empty(t, savedCopies("teams", "config", "WHERE id = ?", teamID))

	patchFleet := decodeObject(s.Do("PATCH", fleetPath, map[string]any{"description": "updated"}, http.StatusOK))
	requireNoFormerlySavedKeys(t, "PATCH /fleets/:id", patchFleet["fleet"].(map[string]any))

	listFleets := decodeObject(s.Do("GET", "/api/latest/fleet/fleets", nil, http.StatusOK))
	var listedFleet map[string]any
	for _, listed := range listFleets["fleets"].([]any) {
		if candidate := listed.(map[string]any); candidate["name"] == teamName {
			listedFleet = candidate
		}
	}
	require.NotNil(t, listedFleet)
	requireNoFormerlySavedKeys(t, "GET /fleets", listedFleet)

	// Target search returns each fleet's whole config.
	targets, err := io.ReadAll(s.Do("POST", "/api/latest/fleet/targets", map[string]any{"query": teamName}, http.StatusOK).Body)
	require.NoError(t, err)
	require.Contains(t, string(targets), teamName)
	require.NotContains(t, string(targets), "bootstrap.pkg")
}

func (s *integrationMDMTestSuite) TestManualAgentInstallChecksUploadedBootstrapPackage() {
	t := s.T()
	ctx := t.Context()

	originalAppConfig, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	originalAppConfig = originalAppConfig.Copy()
	cleanupCtx := context.WithoutCancel(ctx)
	t.Cleanup(func() {
		require.NoError(t, s.ds.SaveAppConfig(cleanupCtx, originalAppConfig))
		_ = s.ds.DeleteMDMAppleBootstrapPackage(cleanupCtx, 0)
	})
	_ = s.ds.DeleteMDMAppleBootstrapPackage(ctx, 0)

	enable := json.RawMessage(`{"mdm": {"setup_experience": {"macos_manual_agent_install": true}}}`)
	res := s.Do("PATCH", "/api/latest/fleet/config", enable, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "first specify a bootstrap package")

	require.NoError(t, s.ds.InsertMDMAppleBootstrapPackage(ctx, &fleet.MDMAppleBootstrapPackage{
		Name:   "bootstrap.pkg",
		Token:  uuid.NewString(),
		Bytes:  []byte("bootstrap"),
		Sha256: []byte("bootstrap-sha256"),
	}, nil))
	s.Do("PATCH", "/api/latest/fleet/config", enable, http.StatusOK)
	// The URL isn't saved, so later changes to other settings rely on the uploaded package too.
	s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"org_info": {"org_name": "manual agent install"}}`), http.StatusOK)
	appConfig, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	require.True(t, appConfig.MDM.MacOSSetup.ManualAgentInstall.Value)

	// GitOps sends an empty URL when it's deleting the package, so the uploaded one doesn't count.
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"mdm": {"setup_experience": {
		"macos_manual_agent_install": true, "macos_bootstrap_package": ""
	}}}`), http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "first specify a bootstrap package")

	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/fleets", &fleet.Team{Name: t.Name()}, http.StatusOK, &createTeamResp)
	teamID := createTeamResp.Team.ID
	t.Cleanup(func() {
		_ = s.ds.DeleteMDMAppleBootstrapPackage(cleanupCtx, teamID)
		require.NoError(t, s.ds.DeleteTeam(cleanupCtx, teamID))
	})
	// Update fleet doesn't have the URL, so it checks the fleet's uploaded package.
	fleetPath := fmt.Sprintf("/api/latest/fleet/fleets/%d", teamID)
	res = s.Do("PATCH", fleetPath, enable, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "first specify a bootstrap package")
	require.NoError(t, s.ds.InsertMDMAppleBootstrapPackage(ctx, &fleet.MDMAppleBootstrapPackage{
		Name:   "bootstrap.pkg",
		TeamID: teamID,
		Token:  uuid.NewString(),
		Bytes:  []byte("fleet bootstrap"),
		Sha256: []byte("fleet-bootstrap-sha256"),
	}, nil))
	s.Do("PATCH", fleetPath, enable, http.StatusOK)
}
