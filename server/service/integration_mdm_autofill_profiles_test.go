package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestAutofillConfigProfiles() {
	t := s.T()
	ctx := t.Context()

	s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"server_settings": {"ai_features_disabled": false}}`), http.StatusOK)

	// A stand-in for fleetdm.com that answers the profile route with whatever the subtest set.
	var mu sync.Mutex
	var contentBody string
	var calls int
	var lastRequest map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		_ = json.NewDecoder(r.Body).Decode(&lastRequest)
		if r.URL.Path == autofillConfigProfileNameDescriptionPath {
			_, _ = w.Write([]byte(`{"name": "Disable AirDrop (macOS)", "description": "Sets allowAirDrop to false."}`))
			return
		}
		_, _ = w.Write([]byte(contentBody))
	}))
	t.Cleanup(srv.Close)
	dev_mode.SetOverride(autofillConfigProfileBaseURLEnvVar, srv.URL, t)

	generate := func(profileType, profile string) {
		body, err := json.Marshal(map[string]any{"profileType": profileType, "profile": profile, "profileFilename": "generated", "deliveryNotes": "A note.", "items": []any{}, "additionalProfiles": []any{}})
		require.NoError(t, err)
		mu.Lock()
		contentBody = string(body)
		mu.Unlock()
	}
	callCount := func() int {
		mu.Lock()
		defer mu.Unlock()
		return calls
	}

	type autofillErrorBody struct {
		Message string              `json:"message"`
		Errors  []map[string]string `json:"errors"`
	}
	requireContentError := func(t *testing.T, params map[string]any, field, reasonContains string) {
		t.Helper()
		resp := s.Do("POST", "/api/latest/fleet/autofill/configuration_profile/content", params, http.StatusUnprocessableEntity)
		var body autofillErrorBody
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.Equal(t, fleet.AutofillConfigProfileContentErrorMessage, body.Message)
		require.Len(t, body.Errors, 1)
		assert.Equal(t, field, body.Errors[0]["name"])
		assert.Contains(t, body.Errors[0]["reason"], reasonContains)
	}

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	otherTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-other"})
	require.NoError(t, err)

	createdProfileActivities := map[string]struct{}{
		fleet.ActivityTypeCreatedMacosProfile{}.ActivityName():       {},
		fleet.ActivityTypeCreatedDeclarationProfile{}.ActivityName(): {},
		fleet.ActivityTypeCreatedWindowsProfile{}.ActivityName():     {},
	}
	countProfilesAndActivities := func() (int, int) {
		var profiles, activities int
		for _, teamID := range []uint{0, team.ID} {
			var listResp listMDMConfigProfilesResponse
			s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", nil, http.StatusOK, &listResp, "fleet_id", fmt.Sprint(teamID), "per_page", "1000")
			profiles += len(listResp.Profiles)
		}
		var actResp listActivitiesResponse
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &actResp, "per_page", "10000")
		for _, a := range actResp.Activities {
			if _, ok := createdProfileActivities[a.Type]; ok {
				activities++
			}
		}
		return profiles, activities
	}
	profilesBefore, activitiesBefore := countProfilesAndActivities()

	mobileconfigWithNestedName := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadDisplayName</key>
			<string>Restrictions</string>
			<key>PayloadIdentifier</key>
			<string>com.example.airdrop.restrictions</string>
			<key>PayloadType</key>
			<string>com.apple.applicationaccess</string>
			<key>PayloadUUID</key>
			<string>6A0E4F4C-0C3E-4E5C-9C1B-7A2B3C4D5E6F</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>allowAirDrop</key>
			<false/>
		</dict>
	</array>
	<key>PayloadDisplayName</key>
	<string>AirDrop Disabled</string>
	<key>PayloadIdentifier</key>
	<string>com.example.airdrop</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>1B2C3D4E-5F60-4718-8293-A4B5C6D7E8F9</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>`

	t.Run("mobileconfig named after its root PayloadDisplayName", func(t *testing.T) {
		generate("mobileconfig", mobileconfigWithNestedName)
		var resp autofillConfigProfileContentResponse
		s.DoJSON("POST", "/api/latest/fleet/autofill/configuration_profile/content", map[string]any{
			"natural_language_instructions": "Turn off AirDrop", "platform": "apple", "fleet_id": team.ID,
		}, http.StatusOK, &resp)
		require.NotNil(t, resp.AutofilledConfigProfile)
		assert.Equal(t, fleet.AutofillConfigProfileTypeMobileconfig, resp.Type)
		require.NotNil(t, resp.Name)
		assert.Equal(t, "AirDrop Disabled", *resp.Name)
		assert.Equal(t, mobileconfigWithNestedName, resp.Content)
		assert.Equal(t, "A note.", resp.DeliveryNotes)

		mu.Lock()
		assert.Equal(t, map[string]any{"naturalLanguageInstructions": "Turn off AirDrop", "platform": "apple", "forFleet": true}, lastRequest)
		mu.Unlock()
	})

	t.Run("declaration and Windows profiles have a null name", func(t *testing.T) {
		generate("ddm", `{"Type": "com.apple.configuration.passcode.settings", "Identifier": "com.example.passcode", "Payload": {"RequirePasscode": true}}`)
		resp := s.Do("POST", "/api/latest/fleet/autofill/configuration_profile/content", map[string]any{
			"natural_language_instructions": "Require a passcode", "platform": "apple",
		}, http.StatusOK)
		raw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"type": "declaration", "name": null, "content": "{\"Type\": \"com.apple.configuration.passcode.settings\", \"Identifier\": \"com.example.passcode\", \"Payload\": {\"RequirePasscode\": true}}", "delivery_notes": "A note."}`, string(raw))

		generate("csp", `<Replace><Item><Meta><Format xmlns="syncml:metinf">int</Format></Meta><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/DeviceLock/DevicePasswordEnabled</LocURI></Target><Data>0</Data></Item></Replace>`)
		var winResp autofillConfigProfileContentResponse
		s.DoJSON("POST", "/api/latest/fleet/autofill/configuration_profile/content", map[string]any{
			"natural_language_instructions": "Require a password", "platform": "windows", "fleet_id": team.ID,
		}, http.StatusOK, &winResp)
		assert.Equal(t, fleet.AutofillConfigProfileTypeWindows, winResp.Type)
		assert.Nil(t, winResp.Name)
	})

	autofillMobileconfig := func(payloadType, keys string) string {
		return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadIdentifier</key>
			<string>com.example.ai.payload</string>
			<key>PayloadType</key>
			<string>%s</string>
			<key>PayloadUUID</key>
			<string>3F2504E0-4F89-41D3-9A0C-0305E82C3301</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			%s
		</dict>
	</array>
	<key>PayloadDisplayName</key>
	<string>AI generated</string>
	<key>PayloadIdentifier</key>
	<string>com.example.ai</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>3F2504E0-4F89-41D3-9A0C-0305E82C3302</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>`, payloadType, keys)
	}
	setProfileValidation := func(on bool) {
		appCfg, err := s.ds.AppConfig(ctx)
		require.NoError(t, err)
		appCfg.MDM.EnableProfileValidation = on
		require.NoError(t, s.ds.SaveAppConfig(ctx, appCfg))
	}

	t.Run("strict schema findings fail under base regardless of the setting", func(t *testing.T) {
		cases := []struct {
			name, profileType, profile, platform, reason string
		}{
			{
				"wrong type", "mobileconfig",
				autofillMobileconfig("com.apple.applicationaccess", `<key>allowAirDrop</key><string>no</string><key>allowCamera</key><string>no</string>`),
				"apple", `"allowAirDrop" expects a boolean.`,
			},
			{
				"unknown key", "mobileconfig",
				autofillMobileconfig("com.apple.applicationaccess", `<key>allowSomethingMadeUp</key><true/>`),
				"apple", `"allowSomethingMadeUp" isn't valid for the "com.apple.applicationaccess" profile type.`,
			},
			{
				"unknown payload type", "mobileconfig",
				autofillMobileconfig("com.example.not.a.real.type", `<key>anything</key><string>x</string>`),
				"apple", `"com.example.not.a.real.type" isn't a valid profile type.`,
			},
			{
				"unknown LocURI", "csp",
				`<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/MadeUpArea/MadeUpSetting</LocURI></Target><Meta><Format xmlns="syncml:metinf">int</Format></Meta><Data>1</Data></Item></Replace>`,
				"windows", `"./Device/Vendor/MSFT/Policy/Config/MadeUpArea/MadeUpSetting" isn't a valid Windows setting.`,
			},
		}
		for _, on := range []bool{true, false} {
			setProfileValidation(on)
			for _, c := range cases {
				t.Run(fmt.Sprintf("%s setting %t", c.name, on), func(t *testing.T) {
					generate(c.profileType, c.profile)
					resp := s.Do("POST", "/api/latest/fleet/autofill/configuration_profile/content", map[string]any{
						"natural_language_instructions": "Make a profile", "platform": c.platform,
					}, http.StatusUnprocessableEntity)
					var body autofillErrorBody
					require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
					require.Equal(t, fleet.AutofillConfigProfileContentErrorMessage, body.Message)
					require.Len(t, body.Errors, 1)
					assert.Equal(t, "base", body.Errors[0]["name"])
					assert.Equal(t, c.reason, body.Errors[0]["reason"])
				})
			}
		}
		setProfileValidation(true)
	})

	t.Run("profiles that pass strict validation are returned", func(t *testing.T) {
		setProfileValidation(true)
		chrome := autofillMobileconfig("com.google.Chrome", `<key>HomepageLocation</key><string>https://fleetdm.com</string>`)
		generate("mobileconfig", chrome)
		var resp autofillConfigProfileContentResponse
		s.DoJSON("POST", "/api/latest/fleet/autofill/configuration_profile/content", map[string]any{
			"natural_language_instructions": "Set the Chrome homepage", "platform": "apple",
		}, http.StatusOK, &resp)
		assert.Equal(t, chrome, resp.Content)

		// built from Chrome's ADMX template, using the generator's AppName
		winChrome := `<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome~Startup/HomepageLocation</LocURI></Target><Meta><Format xmlns="syncml:metinf">chr</Format></Meta><Data><![CDATA[<enabled/><data id="HomepageLocation" value="https://fleetdm.com"/>]]></Data></Item></Replace>`
		generate("csp", winChrome)
		var winResp autofillConfigProfileContentResponse
		s.DoJSON("POST", "/api/latest/fleet/autofill/configuration_profile/content", map[string]any{
			"natural_language_instructions": "Set the Chrome homepage", "platform": "windows",
		}, http.StatusOK, &winResp)
		assert.Equal(t, winChrome, winResp.Content)
	})

	t.Run("name and description", func(t *testing.T) {
		var resp autofillConfigProfileNameDescriptionResponse
		s.DoJSON("POST", "/api/latest/fleet/autofill/configuration_profile/name_description", map[string]any{
			"natural_language_instructions": "Turn off AirDrop", "platform": "apple",
		}, http.StatusOK, &resp)
		assert.Equal(t, "Disable AirDrop (macOS)", resp.Name)
		assert.Equal(t, "Sets allowAirDrop to false.", resp.Description)
	})

	t.Run("checks run against the requested fleet", func(t *testing.T) {
		// FileVault is reserved while custom disk encryption is off.
		generate("mobileconfig", fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
	<key>PayloadContent</key><array><dict>
		<key>PayloadDisplayName</key><string>FileVault</string>
		<key>PayloadIdentifier</key><string>com.example.filevault.payload</string>
		<key>PayloadType</key><string>%s</string>
		<key>PayloadUUID</key><string>2B2C3D4E-5F60-4718-8293-A4B5C6D7E8F9</string>
		<key>PayloadVersion</key><integer>1</integer>
		<key>Enable</key><string>On</string>
	</dict></array>
	<key>PayloadDisplayName</key><string>FileVault</string>
	<key>PayloadIdentifier</key><string>com.example.filevault</string>
	<key>PayloadType</key><string>Configuration</string>
	<key>PayloadUUID</key><string>3B2C3D4E-5F60-4718-8293-A4B5C6D7E8F9</string>
	<key>PayloadVersion</key><integer>1</integer>
</dict></plist>`, mobileconfig.FleetFileVaultPayloadType))
		requireContentError(t, map[string]any{"natural_language_instructions": "Turn on FileVault", "platform": "apple"},
			"natural_language_instructions", mobileconfig.DiskEncryptionProfileRestrictionErrMsg)

		// OS updates configured on one fleet: the same declaration passes for the other fleet and fails for this one.
		// Compared across two fleets rather than against Unassigned, whose OS updates other tests in the suite set.
		team.Config.MDM.MacOSUpdates = fleet.AppleOSUpdateSettings{MinimumVersion: optjson.SetString("26.1"), Deadline: optjson.SetString("2026-12-31")}
		team.Config.MDM.WindowsUpdates = fleet.WindowsUpdates{DeadlineDays: optjson.SetInt(1), GracePeriodDays: optjson.SetInt(1)}
		_, err := s.ds.SaveTeam(ctx, team)
		require.NoError(t, err)

		generate("ddm", `{"Type": "com.apple.configuration.softwareupdate.enforcement.specific", "Identifier": "com.example.os-update", "Payload": {"TargetOSVersion": "26.1", "TargetLocalDateTime": "2026-12-15T18:00:00"}}`)
		s.Do("POST", "/api/latest/fleet/autofill/configuration_profile/content", map[string]any{
			"natural_language_instructions": "Update Macs to 26.1", "platform": "apple", "fleet_id": otherTeam.ID,
		}, http.StatusOK)
		requireContentError(t, map[string]any{"natural_language_instructions": "Update Macs to 26.1", "platform": "apple", "fleet_id": team.ID},
			"natural_language_instructions", fleet.OSUpdatesAlreadyConfiguredErrorMessage)

		generate("csp", `<Replace><Item><Meta><Format xmlns="syncml:metinf">int</Format></Meta><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Update/DeferFeatureUpdatesPeriodInDays</LocURI></Target><Data>90</Data></Item></Replace>`)
		requireContentError(t, map[string]any{"natural_language_instructions": "Defer feature updates by 90 days", "platform": "windows", "fleet_id": team.ID},
			"natural_language_instructions", fleet.OSUpdatesAlreadyConfiguredErrorMessage)

		// A declaration can't reference an asset that doesn't exist.
		generate("ddm", `{"Type": "com.apple.configuration.account.caldav", "Identifier": "com.example.caldav", "Payload": {"HostName": "cal.example.com", "AuthenticationCredentialsAssetReference": "com.example.missing-asset"}}`)
		requireContentError(t, map[string]any{"natural_language_instructions": "Add our calendar", "platform": "apple"},
			"natural_language_instructions", "com.example.missing-asset")
	})

	t.Run("generator declines", func(t *testing.T) {
		// The main stand-in always answers 200, so the decline comes from a second one.
		declineSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "couldNotGenerateProfile", "reason": "Platform SSO isn't supported yet."}`))
		}))
		t.Cleanup(declineSrv.Close)
		dev_mode.SetOverride(autofillConfigProfileBaseURLEnvVar, declineSrv.URL)
		t.Cleanup(func() { dev_mode.SetOverride(autofillConfigProfileBaseURLEnvVar, srv.URL) })

		requireContentError(t, map[string]any{"natural_language_instructions": "Set up Platform SSO", "platform": "apple"},
			"natural_language_instructions", "Platform SSO isn't supported yet.")
	})

	t.Run("AI service failure on both endpoints", func(t *testing.T) {
		failingSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(failingSrv.Close)
		dev_mode.SetOverride(autofillConfigProfileBaseURLEnvVar, failingSrv.URL)
		t.Cleanup(func() { dev_mode.SetOverride(autofillConfigProfileBaseURLEnvVar, srv.URL) })

		requireContentError(t, map[string]any{"natural_language_instructions": "Turn off AirDrop", "platform": "apple"}, "base", "unavailable")

		resp := s.Do("POST", "/api/latest/fleet/autofill/configuration_profile/name_description", map[string]any{
			"natural_language_instructions": "Turn off AirDrop", "platform": "apple",
		}, http.StatusUnprocessableEntity)
		var body autofillErrorBody
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, fleet.AutofillConfigProfileNameDescriptionErrorMessage, body.Message)
		assert.Equal(t, "base", body.Errors[0]["name"])
	})

	t.Run("requests stopped before calling fleetdm.com", func(t *testing.T) {
		before := callCount()

		s.Do("POST", "/api/latest/fleet/autofill/configuration_profile/content", map[string]any{
			"natural_language_instructions": "Turn off the camera", "platform": "android",
		}, http.StatusBadRequest)

		s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"mdm": {"windows_enabled_and_configured": false}}`), http.StatusOK)
		t.Cleanup(func() {
			s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"mdm": {"windows_enabled_and_configured": true}}`), http.StatusOK)
		})
		requireContentError(t, map[string]any{"natural_language_instructions": "Require a password", "platform": "windows"}, "base", fleet.WindowsMDMNotConfiguredMessage)

		s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"server_settings": {"ai_features_disabled": true}}`), http.StatusOK)
		t.Cleanup(func() {
			s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"server_settings": {"ai_features_disabled": false}}`), http.StatusOK)
		})
		s.Do("POST", "/api/latest/fleet/autofill/configuration_profile/content", map[string]any{
			"natural_language_instructions": "Turn off AirDrop", "platform": "apple",
		}, http.StatusBadRequest)
		s.Do("POST", "/api/latest/fleet/autofill/configuration_profile/name_description", map[string]any{
			"natural_language_instructions": "Turn off AirDrop", "platform": "apple",
		}, http.StatusBadRequest)

		assert.Equal(t, before, callCount())
	})

	t.Run("fleet maintainer is limited to their fleet", func(t *testing.T) {
		u := &fleet.User{
			Name:  "autofill team maintainer",
			Email: "autofill-maintainer@example.com",
			Teams: []fleet.UserTeam{{Team: *team, Role: fleet.RoleMaintainer}},
		}
		require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
		_, err := s.ds.NewUser(ctx, u)
		require.NoError(t, err)
		s.setTokenForTest(t, u.Email, test.GoodPassword)

		generate("mobileconfig", mobileconfigWithNestedName)
		s.Do("POST", "/api/latest/fleet/autofill/configuration_profile/content", map[string]any{
			"natural_language_instructions": "Turn off AirDrop", "platform": "apple", "fleet_id": team.ID,
		}, http.StatusOK)
		s.Do("POST", "/api/latest/fleet/autofill/configuration_profile/content", map[string]any{
			"natural_language_instructions": "Turn off AirDrop", "platform": "apple", "fleet_id": otherTeam.ID,
		}, http.StatusForbidden)
		s.Do("POST", "/api/latest/fleet/autofill/configuration_profile/name_description", map[string]any{
			"natural_language_instructions": "Turn off AirDrop", "platform": "apple",
		}, http.StatusOK)
	})

	t.Run("nothing is saved and no activity is created", func(t *testing.T) {
		profilesAfter, activitiesAfter := countProfilesAndActivities()
		assert.Equal(t, profilesBefore, profilesAfter)
		assert.Equal(t, activitiesBefore, activitiesAfter)
	})
}
