package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestRemovedAppleSettingsKeys() {
	t := s.T()

	team, err := s.ds.NewTeam(t.Context(), &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	fleetPath := fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID)
	const configPath = "/api/latest/fleet/config"
	const specPath = "/api/latest/fleet/spec/fleets"

	removedMsg := func(key string) string {
		return `The "` + key + `" parameter is not available anymore. What to use instead: https://fleetdm.com/learn-more-about/fleet-5-migration`
	}
	spec := func(mdm string) string {
		return fmt.Sprintf(`{"specs": [{"name": %q, "mdm": %s}]}`, team.Name, mdm)
	}

	for _, tc := range []struct {
		name, method, path, body, removed string
	}{
		{"config macos_settings", "PATCH", configPath, `{"mdm": {"macos_settings": {"enable_disk_encryption": true}}}`, "macos_settings"},
		{"config macos_settings dry run", "PATCH", configPath + "?dry_run=true", `{"mdm": {"macos_settings": {}}}`, "macos_settings"},
		{"config macos_settings force", "PATCH", configPath + "?force=true", `{"mdm": {"macos_settings": {}}}`, "macos_settings"},
		{
			"config old and new together", "PATCH", configPath,
			`{"mdm": {"apple_settings": {"enable_disk_encryption": true}, "macos_settings": {"enable_disk_encryption": true}}}`, "macos_settings",
		},
		{"config apple custom_settings", "PATCH", configPath, `{"mdm": {"apple_settings": {"custom_settings": []}}}`, "custom_settings"},
		{"config windows custom_settings", "PATCH", configPath, `{"mdm": {"windows_settings": {"custom_settings": []}}}`, "custom_settings"},
		{"config android custom_settings", "PATCH", configPath, `{"mdm": {"android_settings": {"custom_settings": []}}}`, "custom_settings"},
		{"config differently cased", "PATCH", configPath, `{"mdm": {"Windows_Settings": {"Custom_Settings": []}}}`, "Custom_Settings"},
		{
			"config with another setting", "PATCH", configPath,
			`{"org_info": {"org_name": "set by a rejected request"}, "mdm": {"macos_settings": {}}}`, "macos_settings",
		},
		{"fleet macos_settings", "PATCH", fleetPath, `{"mdm": {"macos_settings": {"enable_disk_encryption": true}}}`, "macos_settings"},
		{"fleet spec macos_settings", "POST", specPath, spec(`{"macos_settings": {"enable_disk_encryption": true}}`), "macos_settings"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := s.DoRaw(tc.method, tc.path, []byte(tc.body), http.StatusUnprocessableEntity)
			require.Contains(t, extractServerErrorText(res.Body), removedMsg(tc.removed))
		})
	}

	for _, path := range []string{"/api/latest/fleet/hosts", "/api/latest/fleet/hosts/count"} {
		res := s.DoRaw("GET", path+"?macos_settings=failed", nil, http.StatusUnprocessableEntity)
		require.Contains(t, extractServerErrorText(res.Body), removedMsg("macos_settings"), path)
		s.DoRaw("GET", path+"?apple_settings=failed", nil, http.StatusOK)
	}

	// nothing was saved by the rejected requests
	acResp := appConfigResponse{}
	s.DoJSON("GET", configPath, nil, http.StatusOK, &acResp)
	require.NotEqual(t, "set by a rejected request", acResp.OrgInfo.OrgName)
	teamResp := getTeamResponse{}
	s.DoJSON("GET", fleetPath, nil, http.StatusOK, &teamResp)
	require.False(t, teamResp.Team.Config.MDM.MacOSSettings.EnableDiskEncryption.Value)

	// the same words are still accepted where they aren't the removed keys
	s.DoRaw("POST", "/api/latest/fleet/disk_encryption",
		[]byte(fmt.Sprintf(`{"fleet_id": %d, "macos_settings": {"enable_escrow_disk_encryption_key": true}}`, team.ID)), http.StatusNoContent)

	// the new names are returned alone
	oldKeys := []string{`"macos_settings"`, `"custom_settings"`}
	newKeys := []string{`"apple_settings"`}
	rawBody := func(path string) []byte {
		res := s.DoRaw("GET", path, nil, http.StatusOK)
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		return body
	}
	for _, path := range []string{configPath, fleetPath, "/api/latest/fleet/fleets"} {
		body := string(rawBody(path))
		for _, key := range newKeys {
			require.Contains(t, body, key+":", path)
		}
		for _, key := range oldKeys {
			require.NotContains(t, body, key+":", path)
		}
	}

	// a fleet's mdm object can be sent back as it was returned
	var got struct {
		Fleet struct {
			MDM json.RawMessage `json:"mdm"`
		} `json:"fleet"`
	}
	require.NoError(t, json.Unmarshal(rawBody(fleetPath), &got))
	require.NotEmpty(t, got.Fleet.MDM)
	s.DoRaw("PATCH", fleetPath, []byte(`{"mdm": `+string(got.Fleet.MDM)+`}`), http.StatusOK)
	var again struct {
		Fleet struct {
			MDM json.RawMessage `json:"mdm"`
		} `json:"fleet"`
	}
	require.NoError(t, json.Unmarshal(rawBody(fleetPath), &again))
	require.JSONEq(t, string(got.Fleet.MDM), string(again.Fleet.MDM))

	// host details name the object by its current name only
	host := createOrbitEnrolledHost(t, "darwin", "removed-apple-settings-keys", s.ds)
	var hostResp struct {
		Host struct {
			MDM map[string]json.RawMessage `json:"mdm"`
		} `json:"host"`
	}
	require.NoError(t, json.Unmarshal(rawBody(fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID)), &hostResp))
	require.Contains(t, hostResp.Host.MDM, "apple_settings")
	require.NotContains(t, hostResp.Host.MDM, "macos_settings")
}
