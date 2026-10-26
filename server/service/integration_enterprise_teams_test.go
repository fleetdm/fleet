package service

// Fleet (team) configuration tests for the enterprise (premium) suite.
//
// Belongs here: fleet specs (apply, body size, permissions), fleet CRUD and listing endpoints,
// available fleets, enroll secrets, fleet settings (historical data, software inventory, detail
// query overrides, host name templates), fleet secret obfuscation, and per-fleet and "No team"
// integrations/webhook settings.
//
// Does not belong here: fleet labels (integration_enterprise_labels_test.go), fleet policies
// (integration_enterprise_policies_test.go), fleet OS update settings
// (integration_enterprise_mdm_test.go), fleet scripts and software applied via specs
// (integration_enterprise_scripts_test.go, integration_enterprise_software_batch_test.go), and
// GitOps mode settings (integration_enterprise_gitops_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	mdmtest "github.com/fleetdm/fleet/v4/server/mdm/testing_utils"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestTeamSpecs() {
	t := s.T()

	// Mock Apple GDMF API (required for validating OS update minimum version settings)
	mdmtest.StartNewAppleGDMFTestServer(t)

	// create a team through the service so it initializes the agent ops
	teamName := t.Name() + "team1"
	teamNameDecomposed := teamName + "ᄀ" + "ᅡ" // Add a decomposed Unicode character
	team := &fleet.Team{
		Name:        teamNameDecomposed,
		Description: "desc team1",
	}
	teamName += "가"

	s.Do("POST", "/api/latest/fleet/teams", team, http.StatusOK)

	// Create global calendar integration
	calendarEmail := "service@example.com"
	calendarWebhookUrl := "https://example.com/webhook"
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q,
					"private_key": "testKey"
				},
				"domain": "example.com"
			}]
		}
	}`, calendarEmail,
		)), http.StatusOK,
	)

	// updates a team, no secret is provided so it will keep the one generated
	// automatically when the team was created.
	agentOpts := json.RawMessage(`{"config": {"views": {"foo": "bar"}}, "overrides": {"platforms": {"darwin": {"views": {"bar": "qux"}}}}}`)
	features := json.RawMessage(`{
    "enable_host_users": false,
    "enable_software_inventory": false,
    "additional_queries": {"foo": "bar"}
  }`)
	// must not use applyTeamSpecsRequest and marshal it as JSON, as it will set
	// all keys to their zerovalue, and some are only valid with mdm enabled.
	teamSpecs := map[string]any{
		"specs": []any{
			map[string]any{
				"name":          teamName,
				"agent_options": agentOpts,
				"features":      &features,
				"mdm": map[string]any{
					"macos_updates": map[string]any{
						"minimum_version":  "14.6.1",
						"deadline":         "2021-01-01",
						"update_new_hosts": true,
					},
					"ios_updates": map[string]any{
						"minimum_version": "17.6.1",
						"deadline":        "2024-07-23",
					},
					"ipados_updates": map[string]any{
						"minimum_version": "17.6.1",
						"deadline":        "2024-08-24",
					},
				},
			},
		},
	}
	var applyResp applyTeamSpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	team, err := s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Equal(t, applyResp.TeamIDsByName[teamName], team.ID)
	assert.Len(t, team.Secrets, 1)
	require.JSONEq(t, string(agentOpts), string(*team.Config.AgentOptions))
	require.Equal(t, fleet.Features{
		EnableHostUsers:         false,
		EnableSoftwareInventory: false,
		AdditionalQueries:       new(json.RawMessage(`{"foo": "bar"}`)),
		HistoricalData:          fleet.HistoricalDataSettings{Uptime: true, Vulnerabilities: true},
	}, team.Config.Features)
	require.Equal(t, withCreateDefaults(fleet.TeamMDM{
		MacOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("14.6.1"),
			Deadline:       optjson.SetString("2021-01-01"),
			DeadlineDays:   optjson.Int{Set: true},
			UpdateNewHosts: optjson.SetBool(true),
		},
		IOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("17.6.1"),
			Deadline:       optjson.SetString("2024-07-23"),
			DeadlineDays:   optjson.Int{Set: true},
			UpdateNewHosts: optjson.Bool{Set: true},
		},
		IPadOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("17.6.1"),
			Deadline:       optjson.SetString("2024-08-24"),
			DeadlineDays:   optjson.Int{Set: true},
			UpdateNewHosts: optjson.Bool{Set: true},
		},
		WindowsUpdates: fleet.WindowsUpdates{
			DeadlineDays:    optjson.Int{Set: true},
			GracePeriodDays: optjson.Int{Set: true},
		},
		MacOSSetup: fleet.MacOSSetup{
			// because the MacOSSetup was marshalled to JSON to be saved in the DB,
			// it did get marshalled, and then when unmarshalled it was set (but
			// null).
			MacOSSetupAssistant:         optjson.String{Set: true},
			BootstrapPackage:            optjson.String{Set: true},
			EnableReleaseDeviceManually: optjson.SetBool(false),
			Script:                      optjson.String{Set: true},
			Software:                    optjson.Slice[*fleet.MacOSSetupSoftware]{Set: true, Value: []*fleet.MacOSSetupSoftware{}},
			ManualAgentInstall:          optjson.Bool{Set: true},
			LockEndUserInfo:             optjson.SetBool(false),
			EnableManagedLocalAccount:   optjson.SetBool(false),
			EndUserLocalAccountType:     optjson.SetString("admin"),
		},
		// because the WindowsSettings was marshalled to JSON to be saved in the DB,
		// it did get marshalled, and then when unmarshalled it was set (but
		// empty).
		WindowsSettings: defaultExpectedWindowsSettings(),
		AndroidSettings: fleet.AndroidSettings{
			CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
			Certificates:   optjson.Slice[fleet.CertificateTemplateSpec]{Set: true, Value: []fleet.CertificateTemplateSpec{}},
		},
	}), team.Config.MDM)

	// an activity was created for team spec applied
	s.lastActivityMatches(fleet.ActivityTypeAppliedSpecTeam{}.ActivityName(), fmt.Sprintf(`{"teams": [{"id": %d, "name": %q}], "fleets": [{"id": %d, "name": %q}]}`, team.ID, team.Name, team.ID, team.Name), 0)

	// Create team policy
	teamPolicy, err := s.ds.NewTeamPolicy(
		context.Background(), team.ID, nil, fleet.PolicyPayload{Name: "TestSpecTeamPolicy", Query: "SELECT 1"},
	)
	require.NoError(t, err)
	defer func() {
		_, err = s.ds.DeleteTeamPolicies(context.Background(), team.ID, []uint{teamPolicy.ID})
		require.NoError(t, err)
	}()

	// Apply calendar integration
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"integrations": map[string]any{
					"google_calendar": map[string]any{
						"enable_calendar_events": true,
						"webhook_url":            calendarWebhookUrl,
					},
				},
			},
		},
	}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.NotNil(t, team.Config.Integrations.GoogleCalendar) // nolint:nilaway // cannot be nil due to previous require
	assert.Equal(t, calendarWebhookUrl, team.Config.Integrations.GoogleCalendar.WebhookURL)
	assert.True(t, team.Config.Integrations.GoogleCalendar.Enable)

	// dry-run with invalid EUA=disabled and lock_end_user_info=enabled
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamNameDecomposed,
				"mdm": map[string]any{
					"macos_setup": map[string]any{
						"lock_end_user_info": true,
					},
				},
			},
		},
	}
	res := s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity, "dry_run", "true")
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `"enable_end_user_authentication" must be set to "true"`)

	// dry-run with invalid windows updates
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamNameDecomposed,
				"mdm": map[string]any{
					"windows_updates": map[string]any{
						"deadline_days":     -1,
						"grace_period_days": 1,
					},
				},
			},
		},
	}
	res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity, "dry_run", "true")
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "deadline_days must be an integer between 0 and 30")

	// apply valid windows updates settings
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamNameDecomposed,
				"mdm": map[string]any{
					"windows_updates": map[string]any{
						"deadline_days":     1,
						"grace_period_days": 1,
					},
				},
			},
		},
	}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)
	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Equal(t, applyResp.TeamIDsByName[teamName], team.ID)
	require.Equal(t, withCreateDefaults(fleet.TeamMDM{
		MacOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("14.6.1"),
			Deadline:       optjson.SetString("2021-01-01"),
			DeadlineDays:   optjson.Int{Set: true},
			UpdateNewHosts: optjson.SetBool(true),
		},
		IOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("17.6.1"),
			Deadline:       optjson.SetString("2024-07-23"),
			DeadlineDays:   optjson.Int{Set: true},
			UpdateNewHosts: optjson.Bool{Set: true},
		},
		IPadOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("17.6.1"),
			Deadline:       optjson.SetString("2024-08-24"),
			DeadlineDays:   optjson.Int{Set: true},
			UpdateNewHosts: optjson.Bool{Set: true},
		},
		WindowsUpdates: fleet.WindowsUpdates{
			DeadlineDays:    optjson.SetInt(1),
			GracePeriodDays: optjson.SetInt(1),
		},
		MacOSSetup: fleet.MacOSSetup{
			MacOSSetupAssistant:         optjson.String{Set: true},
			BootstrapPackage:            optjson.String{Set: true},
			EnableReleaseDeviceManually: optjson.SetBool(false),
			Script:                      optjson.String{Set: true},
			Software:                    optjson.Slice[*fleet.MacOSSetupSoftware]{Set: true, Value: []*fleet.MacOSSetupSoftware{}},
			ManualAgentInstall:          optjson.Bool{Set: true},
			LockEndUserInfo:             optjson.SetBool(false),
			EnableManagedLocalAccount:   optjson.SetBool(false),
			EndUserLocalAccountType:     optjson.SetString("admin"),
		},
		WindowsSettings: defaultExpectedWindowsSettings(),
		AndroidSettings: fleet.AndroidSettings{
			CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
			Certificates:   optjson.Slice[fleet.CertificateTemplateSpec]{Set: true, Value: []fleet.CertificateTemplateSpec{}},
		},
	}), team.Config.MDM)

	// get the team via the GET endpoint, check that it properly returns the mdm settings
	var getTmResp getTeamResponse
	s.DoJSON("GET", "/api/latest/fleet/teams/"+fmt.Sprint(team.ID), nil, http.StatusOK, &getTmResp)
	require.Equal(t, withCreateDefaults(fleet.TeamMDM{
		MacOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("14.6.1"),
			Deadline:       optjson.SetString("2021-01-01"),
			DeadlineDays:   optjson.Int{Set: true},
			UpdateNewHosts: optjson.SetBool(true),
		},
		IOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("17.6.1"),
			Deadline:       optjson.SetString("2024-07-23"),
			DeadlineDays:   optjson.Int{Set: true},
			UpdateNewHosts: optjson.Bool{Set: true},
		},
		IPadOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("17.6.1"),
			Deadline:       optjson.SetString("2024-08-24"),
			DeadlineDays:   optjson.Int{Set: true},
			UpdateNewHosts: optjson.Bool{Set: true},
		},
		WindowsUpdates: fleet.WindowsUpdates{
			DeadlineDays:    optjson.SetInt(1),
			GracePeriodDays: optjson.SetInt(1),
		},
		MacOSSetup: fleet.MacOSSetup{
			MacOSSetupAssistant:         optjson.String{Set: true},
			BootstrapPackage:            optjson.String{Set: true},
			EnableReleaseDeviceManually: optjson.SetBool(false),
			Script:                      optjson.String{Set: true},
			Software:                    optjson.Slice[*fleet.MacOSSetupSoftware]{Set: true, Value: []*fleet.MacOSSetupSoftware{}},
			ManualAgentInstall:          optjson.Bool{Set: true},
			LockEndUserInfo:             optjson.SetBool(false),
			EnableManagedLocalAccount:   optjson.SetBool(false),
			EndUserLocalAccountType:     optjson.SetString("admin"),
		},
		WindowsSettings: defaultExpectedWindowsSettings(),
		AndroidSettings: fleet.AndroidSettings{
			CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
			Certificates:   optjson.Slice[fleet.CertificateTemplateSpec]{Set: true, Value: []fleet.CertificateTemplateSpec{}},
		},
	}), getTmResp.Team.Config.MDM)

	// get the team via the list teams endpoint, check that it properly returns the mdm settings
	var listTmResp listTeamsResponse
	s.DoJSON("GET", "/api/latest/fleet/teams", nil, http.StatusOK, &listTmResp, "query", teamName)
	require.NotEmpty(t, listTmResp.Teams)
	require.Equal(t, team.ID, listTmResp.Teams[0].ID)
	require.Equal(t, withCreateDefaults(fleet.TeamMDM{
		MacOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("14.6.1"),
			Deadline:       optjson.SetString("2021-01-01"),
			DeadlineDays:   optjson.Int{Set: true},
			UpdateNewHosts: optjson.SetBool(true),
		},
		IOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("17.6.1"),
			Deadline:       optjson.SetString("2024-07-23"),
			DeadlineDays:   optjson.Int{Set: true},
			UpdateNewHosts: optjson.Bool{Set: true},
		},
		IPadOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("17.6.1"),
			Deadline:       optjson.SetString("2024-08-24"),
			DeadlineDays:   optjson.Int{Set: true},
			UpdateNewHosts: optjson.Bool{Set: true},
		},
		WindowsUpdates: fleet.WindowsUpdates{
			DeadlineDays:    optjson.SetInt(1),
			GracePeriodDays: optjson.SetInt(1),
		},
		MacOSSetup: fleet.MacOSSetup{
			MacOSSetupAssistant:         optjson.String{Set: true},
			BootstrapPackage:            optjson.String{Set: true},
			EnableReleaseDeviceManually: optjson.SetBool(false),
			Script:                      optjson.String{Set: true},
			Software:                    optjson.Slice[*fleet.MacOSSetupSoftware]{Set: true, Value: []*fleet.MacOSSetupSoftware{}},
			ManualAgentInstall:          optjson.Bool{Set: true},
			LockEndUserInfo:             optjson.SetBool(false),
			EnableManagedLocalAccount:   optjson.SetBool(false),
			EndUserLocalAccountType:     optjson.SetString("admin"),
		},
		WindowsSettings: defaultExpectedWindowsSettings(),
		AndroidSettings: fleet.AndroidSettings{
			CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
			Certificates:   optjson.Slice[fleet.CertificateTemplateSpec]{Set: true, Value: []fleet.CertificateTemplateSpec{}},
		},
	}), listTmResp.Teams[0].Config.MDM)

	// dry-run with invalid agent options
	agentOpts = json.RawMessage(`{"config": {"nope": 1}}`)
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":          teamNameDecomposed,
				"agent_options": agentOpts,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusBadRequest, "dry_run", "true")

	// dry-run with empty body
	res = s.DoRaw("POST", "/api/latest/fleet/spec/teams", nil, http.StatusBadRequest, "force", "true")
	errBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Contains(t, string(errBody), `"Expected JSON Body"`)

	// dry-run with invalid top-level key
	s.Do("POST", "/api/latest/fleet/spec/teams", json.RawMessage(`{
		"specs": [
			{"name": "team_name_1", "unknown_key": true}
		]
	}`), http.StatusBadRequest, "dry_run", "true")

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Contains(t, string(*team.Config.AgentOptions), `"foo": "bar"`) // unchanged

	// dry-run with valid agent options and custom macos settings
	agentOpts = json.RawMessage(`{"config": {"views": {"foo": "qux"}}}`)
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":          teamNameDecomposed,
				"agent_options": agentOpts,
				"mdm": map[string]any{
					"macos_settings": map[string]any{
						"custom_settings": []string{"foo", "bar"},
					},
				},
			},
		},
	}
	res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity, "dry_run", "true")
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't update apple_settings because MDM features aren't turned on in Fleet.")

	// dry-run with macos disk encryption set to false, no error
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"mdm": map[string]any{
					"macos_settings": map[string]any{
						"enable_disk_encryption": false,
					},
				},
			},
		},
	}
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp, "dry_run", "true")
	assert.Equal(t, map[string]uint{teamName: team.ID}, applyResp.TeamIDsByName)

	// dry-run with macos disk encryption set to true, no error: like the disk
	// encryption API, it doesn't require Apple MDM
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"mdm": map[string]any{
					"macos_settings": map[string]any{
						"enable_disk_encryption": true,
					},
				},
			},
		},
	}
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp, "dry_run", "true")
	assert.Equal(t, map[string]uint{teamName: team.ID}, applyResp.TeamIDsByName)

	// dry-run with macos enable release device set to false, no error
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"mdm": map[string]any{
					"macos_setup": map[string]any{
						"enable_release_device_manually": false,
					},
				},
			},
		},
	}
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp, "dry_run", "true")
	assert.Equal(t, map[string]uint{teamName: team.ID}, applyResp.TeamIDsByName)

	// dry-run with macos enable release device manually set to true
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"mdm": map[string]any{
					"macos_setup": map[string]any{
						"enable_release_device_manually": true,
					},
				},
			},
		},
	}
	res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity, "dry_run", "true")
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't update setup_experience because MDM features aren't turned on in Fleet.")

	// dry-run with invalid host_expiry_settings.host_expiry_window
	teamSpecs = map[string]any{
		"specs": []map[string]any{
			{
				"name": teamName,
				"host_expiry_settings": map[string]any{
					"host_expiry_window":  0,
					"host_expiry_enabled": true,
				},
			},
		},
	}
	// Update team
	res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity, "dry_run", "true")
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "host expiry window")
	// Create team (coverage should show that this validation was covered for both update and create)
	teamSpecs["specs"].([]map[string]any)[0]["name"] = teamName + "invalid host expiry window"
	res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity, "dry_run", "true")
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "host expiry window")

	// dry-run with valid agent options only
	agentOpts = json.RawMessage(`{"config": {"views": {"foo": "qux"}}}`)
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":          teamName,
				"agent_options": agentOpts,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "dry_run", "true")

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Contains(t, string(*team.Config.AgentOptions), `"foo": "bar"`) // unchanged
	require.Empty(t, team.Config.MDM.MacOSSettings.CustomSettings)         // unchanged
	require.False(t, team.Config.MDM.EnableDiskEncryption)                 // unchanged

	// apply without agent options specified
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// agent options are unchanged, not cleared
	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Contains(t, string(*team.Config.AgentOptions), `"foo": "bar"`) // unchanged

	// apply with agent options specified but null
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":          teamName,
				"agent_options": json.RawMessage(`null`),
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// agent options are cleared
	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Nil(t, team.Config.AgentOptions)

	// force with invalid agent options
	agentOpts = json.RawMessage(`{"config": {"foo": "qux"}}`)
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":          teamName,
				"agent_options": agentOpts,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "force", "true")

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Contains(t, string(*team.Config.AgentOptions), `"foo": "qux"`)

	// force create new team with invalid top-level key
	s.Do("POST", "/api/latest/fleet/spec/teams", json.RawMessage(`{
		"specs": [
			{"name": "team_with_invalid_key", "unknown_key": true}
		]
	}`), http.StatusOK, "force", "true")

	_, err = s.ds.TeamByName(context.Background(), "team_with_invalid_key")
	require.NoError(t, err)

	// invalid agent options command-line flag
	agentOpts = json.RawMessage(`{"command_line_flags": {"nope": 1}}`)
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":          teamName,
				"agent_options": agentOpts,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusBadRequest)

	// valid agent options command-line flag
	agentOpts = json.RawMessage(`{"command_line_flags": {"enable_tables": "abcd"}}`)
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":          teamName,
				"agent_options": agentOpts,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Contains(t, string(*team.Config.AgentOptions), `"enable_tables": "abcd"`)

	// creates a team with default agent options
	user, err := s.ds.UserByEmail(context.Background(), "admin1@example.com")
	require.NoError(t, err)

	teams, err := s.ds.ListTeams(context.Background(), fleet.TeamFilter{User: user}, fleet.ListOptions{})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(teams), 1)

	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": "team2",
			},
		},
	}
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	teams, err = s.ds.ListTeams(context.Background(), fleet.TeamFilter{User: user}, fleet.ListOptions{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(teams), 2)

	team, err = s.ds.TeamByName(context.Background(), "team2")
	require.NoError(t, err)
	require.Equal(t, applyResp.TeamIDsByName["team2"], team.ID)

	appConfig, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	defaultOpts := `{"config": {"options": {"logger_plugin": "tls", "pack_delimiter": "/", "logger_tls_period": 10, "distributed_plugin": "tls", "disable_distributed": false, "logger_tls_endpoint": "/api/osquery/log", "distributed_interval": 10, "distributed_tls_max_attempts": 3}, "decorators": {"load": ["SELECT uuid AS host_uuid FROM system_info;", "SELECT hostname AS hostname FROM system_info;"]}}, "overrides": {}}`
	assert.Len(t, team.Secrets, 1) // secret gets created automatically for a new team when none is supplied.
	require.NotNil(t, team.Config.AgentOptions)
	require.JSONEq(t, defaultOpts, string(*team.Config.AgentOptions))
	require.Equal(t, appConfig.Features, team.Config.Features)

	// an activity was created for the newly created team via the applied spec
	s.lastActivityMatches(fleet.ActivityTypeAppliedSpecTeam{}.ActivityName(), fmt.Sprintf(`{"teams": [{"id": %d, "name": %q}], "fleets": [{"id": %d, "name": %q}]}`, team.ID, team.Name, team.ID, team.Name), 0)

	// updates
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":     "team2",
				"secrets":  []fleet.EnrollSecret{{Secret: "ABC"}},
				"features": nil,
			},
		},
	}
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	team, err = s.ds.TeamByName(context.Background(), "team2")
	require.NoError(t, err)
	require.Equal(t, applyResp.TeamIDsByName["team2"], team.ID)

	require.Len(t, team.Secrets, 1)
	assert.Equal(t, "ABC", team.Secrets[0].Secret)
}

func (s *integrationEnterpriseTestSuite) TestTeamSpecsMacOSDiskEncryptionWithoutAppleMDM() {
	t := s.T()
	ctx := t.Context()

	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	require.False(t, appCfg.MDM.EnabledAndConfigured, "this suite runs without Apple MDM")

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	spec := func(macOSSettings map[string]any) map[string]any {
		return map[string]any{"specs": []any{map[string]any{"name": team.Name, "mdm": map[string]any{"macos_settings": macOSSettings}}}}
	}

	var applyResp applyTeamSpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", spec(map[string]any{
		"enable_disk_encryption":            true,
		"enable_escrow_disk_encryption_key": true,
	}), http.StatusOK, &applyResp)

	var getResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)
	assert.True(t, getResp.Team.Config.MDM.MacOSSettings.EnableDiskEncryption.Value)
	assert.True(t, getResp.Team.Config.MDM.MacOSSettings.EnableEscrowDiskEncryptionKey.Value)

	// custom profiles still need Apple MDM
	res := s.Do("POST", "/api/latest/fleet/spec/teams", spec(map[string]any{
		"custom_settings": []string{"foo"},
	}), http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "Couldn't update apple_settings because MDM features aren't turned on in Fleet.")
}

// applyTeamSpecsRequest hands its decode error back as a UserMessageError, so an oversized body only
// reaches the caller as a 413 while that wrapper still unwraps to the size error underneath. Without
// it the request is answered with a 400 carrying the raw read error.
func (s *integrationEnterpriseTestSuite) TestTeamSpecsBodySizeLimit() {
	t := s.T()

	// The limit the /spec/fleets route is registered with.
	const limit = 5 * units.MiB

	// The padding sits inside a JSON string value so the body stays syntactically valid up to the
	// point where the reader is cut off.
	prefix := `{"specs":[{"name":"`
	suffix := `"}]}`
	padSize := limit + 1 - len(prefix) - len(suffix)
	require.Positive(t, padSize, "padding must be positive")

	res := s.DoRaw("POST", "/api/latest/fleet/spec/fleets",
		[]byte(prefix+strings.Repeat("x", padSize)+suffix), http.StatusRequestEntityTooLarge)
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	require.NoError(t, err)
	assert.Contains(t, string(body), "exceeds the max size limit")
	assert.NotContains(t, string(body), "request body too large")

	// A malformed body within the limit is still a 400, so the size error isn't reported for a
	// reader that simply ran out early.
	s.DoRaw("POST", "/api/latest/fleet/spec/fleets", []byte(prefix), http.StatusBadRequest)
}

func (s *integrationEnterpriseTestSuite) TestTeamSpecsPermissions() {
	t := s.T()

	//
	// Setup test
	//

	// Create two teams, team1 and team2.
	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          43,
		Name:        "team2",
		Description: "desc team2",
	})
	require.NoError(t, err)
	// Create a new admin for team1.
	password := test.GoodPassword
	email := "admin-team1@example.com"
	u := &fleet.User{
		Name:       "admin team1",
		Email:      email,
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team1,
				Role: fleet.RoleAdmin,
			},
		},
	}
	require.NoError(t, u.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)

	//
	// Start testing team specs with admin of team1.
	//

	s.setTokenForTest(t, "admin-team1@example.com", test.GoodPassword)

	// Should allow editing own team.
	agentOpts := json.RawMessage(`{"config": {"views": {"foo": "bar2"}}, "overrides": {"platforms": {"darwin": {"views": {"bar": "qux"}}}}}`)
	editTeam1Spec := map[string]any{
		"specs": []any{
			map[string]any{
				"name":          team1.Name,
				"agent_options": agentOpts,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", editTeam1Spec, http.StatusOK)
	team1b, err := s.ds.TeamLite(context.Background(), team1.ID)
	require.NoError(t, err)
	require.Equal(t, *team1b.Config.AgentOptions, agentOpts)

	// Should not allow editing other teams.
	editTeam2Spec := map[string]any{
		"specs": []any{
			map[string]any{
				"name":          team2.Name,
				"agent_options": agentOpts,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", editTeam2Spec, http.StatusForbidden)
}

func (s *integrationEnterpriseTestSuite) TestModifyTeamEnrollSecrets() {
	t := s.T()

	// Create new team and set initial secret
	teamName := t.Name() + "secretTeam"
	team := &fleet.Team{
		Name:        teamName,
		Description: "secretTeam description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "initialSecret"}},
	}

	s.Do("POST", "/api/latest/fleet/teams", team, http.StatusOK)

	team, err := s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	assert.Equal(t, "initialSecret", team.Secrets[0].Secret)

	// Test replace existing secrets
	req := json.RawMessage(`{"secrets": [{"secret": "testSecret1"},{"secret": "testSecret2"}]}`)
	var resp teamEnrollSecretsResponse

	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), req, http.StatusOK, &resp)
	require.Len(t, resp.Secrets, 2)

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	assert.Equal(t, "testSecret1", team.Secrets[0].Secret)
	assert.Equal(t, "testSecret2", team.Secrets[1].Secret)

	seenActivitiesIDs := map[uint]struct{}{}
	activityName := fleet.ActivityTypeEditedEnrollSecrets{}.ActivityName()
	activityDetails := fmt.Sprintf(`{"team_id": %d, "team_name": "%s", "fleet_id": %d, "fleet_name": "%s"}`, team.ID, team.Name, team.ID, team.Name)

	// Check that an activity was created for the new secrets
	seenActivitiesIDs[s.lastActivityMatches(activityName, activityDetails, 0)] = struct{}{}
	require.Len(t, seenActivitiesIDs, 1)

	// Posting the same secret shouldn't create a new activity, since we are only interested in mutations
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), req, http.StatusOK, &resp)
	seenActivitiesIDs[s.lastActivityMatches(activityName, activityDetails, 0)] = struct{}{}
	require.Len(t, seenActivitiesIDs, 1)

	// Test delete all enroll secrets
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{"secrets": []}`), http.StatusOK, &resp)
	require.Empty(t, resp.Secrets)

	// We should see a new activity for the deleted secrets
	seenActivitiesIDs[s.lastActivityMatches(activityName, activityDetails, 0)] = struct{}{}
	require.Len(t, seenActivitiesIDs, 2)

	// Test bad requests
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{"foo": [{"secret": "testSecret3"}]}`), http.StatusUnprocessableEntity, &resp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{}`), http.StatusUnprocessableEntity, &resp)

	// No new activities should be generated
	seenActivitiesIDs[s.lastActivityMatches(activityName, activityDetails, 0)] = struct{}{}
	require.Len(t, seenActivitiesIDs, 2)

	// too many secrets
	secrets := createEnrollSecrets(t, fleet.MaxEnrollSecretsCount+1)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{"secrets": `+string(jsonMustMarshal(t, secrets))+`}`), http.StatusUnprocessableEntity, &resp)

	// empty and whitespace-only secrets are rejected
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{"secrets": [{"secret": ""}]}`), http.StatusUnprocessableEntity, &resp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{"secrets": [{"secret": "   "}]}`), http.StatusUnprocessableEntity, &resp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{"secrets": [{"secret": "validSecret"},{"secret": ""}]}`), http.StatusUnprocessableEntity, &resp)

	// No new activities should be generated
	seenActivitiesIDs[s.lastActivityMatches(activityName, activityDetails, 0)] = struct{}{}
	require.Len(t, seenActivitiesIDs, 2)
}

func (s *integrationEnterpriseTestSuite) TestModifyTeamHistoricalData() {
	t := s.T()
	ctx := context.Background()

	// Create a fleet — features.historical_data should default to true/true.
	teamName := t.Name() + "historicalDataFleet"
	createPayload := &fleet.Team{
		Name:        teamName,
		Description: "historical data fleet",
		Secrets:     []*fleet.EnrollSecret{{Secret: t.Name() + "secret"}},
	}
	var createResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", createPayload, http.StatusOK, &createResp)
	require.True(t, createResp.Team.Config.Features.HistoricalData.Uptime, "new fleet defaults uptime=true")
	require.True(t, createResp.Team.Config.Features.HistoricalData.Vulnerabilities, "new fleet defaults vulnerabilities=true")
	teamID := createResp.Team.ID
	t.Cleanup(func() {
		require.NoError(t, s.ds.DeleteTeam(ctx, teamID))
	})

	// PATCH only `vulnerabilities=false` — uptime should remain true.
	var modResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", teamID),
		json.RawMessage(`{"features": {"historical_data": {"vulnerabilities": false}}}`),
		http.StatusOK, &modResp)
	require.True(t, modResp.Team.Config.Features.HistoricalData.Uptime, "uptime preserved when omitted")
	require.False(t, modResp.Team.Config.Features.HistoricalData.Vulnerabilities)

	// One disabled_historical_dataset activity scoped to this fleet.
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDisabledHistoricalDataset{}.ActivityName(),
		fmt.Sprintf(`{"dataset":"vulnerabilities","fleet_id":%d,"fleet_name":%q}`, teamID, teamName),
		0,
	)

	// PATCH the same value back — no new activity.
	priorActivityID := s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDisabledHistoricalDataset{}.ActivityName(), "", 0,
	)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", teamID),
		json.RawMessage(`{"features": {"historical_data": {"vulnerabilities": false}}}`),
		http.StatusOK, &modResp)
	require.Equal(t, priorActivityID, s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDisabledHistoricalDataset{}.ActivityName(), "", 0,
	), "no new activity for no-op fleet PATCH")

	// Flip both in one PATCH — re-enable vulnerabilities, disable uptime → 2 activities.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", teamID),
		json.RawMessage(`{"features": {"historical_data": {"uptime": false, "vulnerabilities": true}}}`),
		http.StatusOK, &modResp)
	require.False(t, modResp.Team.Config.Features.HistoricalData.Uptime)
	require.True(t, modResp.Team.Config.Features.HistoricalData.Vulnerabilities)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEnabledHistoricalDataset{}.ActivityName(),
		fmt.Sprintf(`{"dataset":"vulnerabilities","fleet_id":%d,"fleet_name":%q}`, teamID, teamName),
		0,
	)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDisabledHistoricalDataset{}.ActivityName(),
		fmt.Sprintf(`{"dataset":"uptime","fleet_id":%d,"fleet_name":%q}`, teamID, teamName),
		0,
	)

	// Unknown features sub-fields (e.g. enable_host_users — settable per-fleet
	// only via /spec/fleets) are silently ignored on the fleet PATCH endpoint;
	// the stored fleet state should be unchanged for the unknown field.
	priorFeatures := modResp.Team.Config.Features
	priorEnabledActivityID := s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEnabledHistoricalDataset{}.ActivityName(), "", 0,
	)
	priorDisabledActivityID := s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDisabledHistoricalDataset{}.ActivityName(), "", 0,
	)
	// Send the OPPOSITE of the current value so the PATCH would be observable
	// if the endpoint accidentally accepted the unknown sub-field.
	flippedEnableHostUsers := !priorFeatures.EnableHostUsers
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", teamID),
		json.RawMessage(fmt.Sprintf(`{"features": {"enable_host_users": %t}}`, flippedEnableHostUsers)),
		http.StatusOK, &modResp)
	require.Equal(t, priorFeatures.EnableHostUsers, modResp.Team.Config.Features.EnableHostUsers,
		"unknown features sub-field is ignored")
	require.Equal(t, priorFeatures.HistoricalData, modResp.Team.Config.Features.HistoricalData,
		"historical data unchanged when unknown field patched")
	require.Equal(t, priorEnabledActivityID, s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEnabledHistoricalDataset{}.ActivityName(), "", 0,
	), "no enable historical data activity for ignored patch")
	require.Equal(t, priorDisabledActivityID, s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDisabledHistoricalDataset{}.ActivityName(), "", 0,
	), "no disable historical data activity for ignored patch")
}

func (s *integrationEnterpriseTestSuite) TestModifyTeamEnableSoftwareInventory() {
	t := s.T()
	ctx := context.Background()

	// Create a fleet — features.enable_software_inventory snapshots the global
	// config. `features` in the POST payload is ignored (pre-existing
	// behavior, consistent with historical_data), so send the OPPOSITE of the
	// global value to make the ignore observable.
	globalCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	globalEnabled := globalCfg.Features.EnableSoftwareInventory
	teamName := t.Name() + "softwareInventoryFleet"
	var createResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", json.RawMessage(fmt.Sprintf(
		`{"name": %q, "features": {"enable_software_inventory": %t}}`, teamName, !globalEnabled,
	)), http.StatusOK, &createResp)
	require.Equal(t, globalEnabled, createResp.Team.Config.Features.EnableSoftwareInventory,
		"features is ignored on POST; new fleet snapshots the global config value")
	teamID := createResp.Team.ID
	t.Cleanup(func() {
		require.NoError(t, s.ds.DeleteTeam(ctx, teamID))
	})

	// PATCH enable_software_inventory=false.
	var modResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", teamID),
		json.RawMessage(`{"features": {"enable_software_inventory": false}}`),
		http.StatusOK, &modResp)
	require.False(t, modResp.Team.Config.Features.EnableSoftwareInventory)

	// PATCH other fields without the key — value retained (PATCH-merge semantics).
	newName := teamName + "renamed"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", teamID),
		json.RawMessage(fmt.Sprintf(`{"name": %q, "features": {"historical_data": {"uptime": false}}}`, newName)),
		http.StatusOK, &modResp)
	require.Equal(t, newName, modResp.Team.Name)
	require.False(t, modResp.Team.Config.Features.EnableSoftwareInventory,
		"enable_software_inventory preserved when omitted from PATCH")
	require.False(t, modResp.Team.Config.Features.HistoricalData.Uptime)

	// Invalid value returns a 4xx error and does not change the setting.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", teamID),
		json.RawMessage(`{"features": {"enable_software_inventory": "yes"}}`),
		http.StatusBadRequest, &modResp)
	teamFeatures, err := s.ds.TeamFeatures(ctx, teamID)
	require.NoError(t, err)
	require.False(t, teamFeatures.EnableSoftwareInventory,
		"setting unchanged after invalid value")

	// `null` is treated as omitted — value retained.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", teamID),
		json.RawMessage(`{"features": {"enable_software_inventory": null}}`),
		http.StatusOK, &modResp)
	require.False(t, modResp.Team.Config.Features.EnableSoftwareInventory)

	// Re-enable.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", teamID),
		json.RawMessage(`{"features": {"enable_software_inventory": true}}`),
		http.StatusOK, &modResp)
	require.True(t, modResp.Team.Config.Features.EnableSoftwareInventory)

	// PATCH /fleets/0 (Unassigned) ignores `features` (pre-existing behavior,
	// consistent with historical_data); Unassigned hosts follow the global
	// config setting, so send the OPPOSITE of the global value and verify it
	// is unchanged.
	s.DoJSON("PATCH", "/api/latest/fleet/fleets/0",
		json.RawMessage(fmt.Sprintf(`{"features": {"enable_software_inventory": %t}}`, !globalEnabled)),
		http.StatusOK, &modResp)
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	require.Equal(t, globalEnabled, appCfg.Features.EnableSoftwareInventory,
		"global setting unchanged by Unassigned fleet PATCH")
}

func (s *integrationEnterpriseTestSuite) TestAvailableTeams() {
	t := s.T()

	// create a new team
	team := &fleet.Team{
		Name:        "Available Team",
		Description: "Available Team description",
	}

	s.Do("POST", "/api/latest/fleet/teams", team, http.StatusOK)

	team, err := s.ds.TeamByName(context.Background(), "Available Team")
	require.NoError(t, err)

	// create a new user
	user := &fleet.User{
		Name:       "Available Teams User",
		Email:      "available@example.com",
		GlobalRole: new("observer"),
	}
	err = user.SetPassword(test.GoodPassword, 10, 10)
	require.NoError(t, err)
	user, err = s.ds.NewUser(context.Background(), user)
	require.NoError(t, err)

	// test available teams for user assigned to global role
	var getResp getUserResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", user.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, user.ID, getResp.User.ID)
	assert.Equal(t, new("observer"), getResp.User.GlobalRole)
	assert.Empty(t, getResp.User.Teams)      // teams is empty if user has a global role
	assert.Len(t, getResp.AvailableTeams, 1) // available teams includes all teams if user has a global role
	assert.Equal(t, "Available Team", getResp.AvailableTeams[0].Name)

	// assign user to a team
	user.GlobalRole = nil
	user.Teams = []fleet.UserTeam{{Team: *team, Role: "maintainer"}}
	err = s.ds.SaveUser(context.Background(), user)
	require.NoError(t, err)

	// test available teams for user assigned to team role
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", user.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, user.ID, getResp.User.ID)
	assert.Nil(t, getResp.User.GlobalRole)
	assert.Len(t, getResp.User.Teams, 1)
	assert.Equal(t, "Available Team", getResp.User.Teams[0].Name)
	assert.Len(t, getResp.AvailableTeams, 1)
	assert.Equal(t, "Available Team", getResp.AvailableTeams[0].Name)

	// test available teams returned by `/me` endpoint
	session, err := s.ds.NewSession(context.Background(), user.ID, 64)
	require.NoError(t, err)
	resp := s.DoRawWithHeaders("GET", "/api/latest/fleet/me", []byte(""), http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", session.Key),
	})
	err = json.NewDecoder(resp.Body).Decode(&getResp)
	require.NoError(t, err)
	assert.Equal(t, user.ID, getResp.User.ID)
	assert.Nil(t, getResp.User.GlobalRole)
	assert.Len(t, getResp.User.Teams, 1)
	assert.Equal(t, "Available Team", getResp.User.Teams[0].Name)
	assert.Len(t, getResp.AvailableTeams, 1)
	assert.Equal(t, "Available Team", getResp.AvailableTeams[0].Name)
}

func (s *integrationEnterpriseTestSuite) TestTeamEndpoints() {
	t := s.T()

	name := strings.ReplaceAll(t.Name(), "/", "_")
	// create a new team
	team := &fleet.Team{
		Name:        name,
		Description: "Team description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "DEF"}},
	}

	var tmResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &tmResp)
	assert.Equal(t, team.Name, tmResp.Team.Name)
	assert.False(t, tmResp.Team.CreatedAt.IsZero())
	assert.WithinDuration(t, time.Now(), tmResp.Team.CreatedAt, time.Minute)
	require.Len(t, tmResp.Team.Secrets, 1)
	assert.Equal(t, "DEF", tmResp.Team.Secrets[0].Secret)
	assert.False(t, tmResp.Team.Secrets[0].CreatedAt.IsZero())
	assert.WithinDuration(t, time.Now(), tmResp.Team.Secrets[0].CreatedAt, time.Minute)

	// create a duplicate team (same name)
	team2 := &fleet.Team{
		Name:        name,
		Description: "Team2 description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "GHI"}},
	}
	s.DoJSON("POST", "/api/latest/fleet/teams", team2, http.StatusConflict, &tmResp)

	// create a team whose name only differs by case — must also 409 with the
	// canonical "must differ" message.
	teamCaseVariant := &fleet.Team{
		Name:        strings.ToLower(name),
		Description: "case variant",
		Secrets:     []*fleet.EnrollSecret{{Secret: "CASEVAR"}},
	}
	r := s.Do("POST", "/api/latest/fleet/teams", teamCaseVariant, http.StatusConflict)
	require.Contains(t, extractServerErrorText(r.Body), "must differ by more than letter case")

	// create a team with reserved team names; should be case-insensitive
	teamReserved := &fleet.Team{
		Name:        "no TeAm",
		Description: "description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "foobar"}},
	}

	r = s.Do("POST", "/api/latest/fleet/teams", teamReserved, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), `is a reserved fleet name`)

	teamReserved.Name = "AlL TeaMS"
	r = s.Do("POST", "/api/latest/fleet/teams", teamReserved, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), `is a reserved fleet name`)

	// create a team with a whitespace-only name
	teamWhitespace := &fleet.Team{
		Name:        "     ",
		Description: "description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "whitespace-team"}},
	}
	r = s.Do("POST", "/api/latest/fleet/teams", teamWhitespace, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), `may not be empty`)

	// create a team with leading/trailing whitespace — name should be trimmed
	teamPadded := &fleet.Team{
		Name:        "  Padded Team  ",
		Description: "description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "padded-team"}},
	}
	s.DoJSON("POST", "/api/latest/fleet/teams", teamPadded, http.StatusOK, &tmResp)
	require.Equal(t, "Padded Team", tmResp.Team.Name)
	paddedTeamID := tmResp.Team.ID

	// rename with leading/trailing whitespace — name should be trimmed
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", paddedTeamID), fleet.TeamPayload{Name: new("  Renamed Padded  ")}, http.StatusOK, &tmResp)
	require.Equal(t, "Renamed Padded", tmResp.Team.Name)

	// case-only self-rename is allowed (the team's own id is excluded from
	// the conflict check).
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", paddedTeamID), fleet.TeamPayload{Name: new("RENAMED PADDED")}, http.StatusOK, &tmResp)
	require.Equal(t, "RENAMED PADDED", tmResp.Team.Name)

	// renaming into another team's name (the original team created above)
	// using only case differences must return 409 with the canonical message.
	r = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", paddedTeamID), fleet.TeamPayload{Name: new(strings.ToUpper(name))}, http.StatusConflict)
	require.Contains(t, extractServerErrorText(r.Body), "must differ by more than letter case")

	// clean up
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d", paddedTeamID), nil, http.StatusOK)

	// create a team with too many secrets
	team3 := &fleet.Team{
		Name:        name + "lots_of_secrets",
		Description: "Team3 description",
		Secrets:     createEnrollSecrets(t, fleet.MaxEnrollSecretsCount+1),
	}
	tmResp.Team = nil
	s.DoJSON("POST", "/api/latest/fleet/teams", team3, http.StatusUnprocessableEntity, &tmResp)

	// create a team with an empty enroll secret
	tmResp.Team = nil
	s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{
		Name:    name + "empty_secret",
		Secrets: []*fleet.EnrollSecret{{Secret: ""}},
	}, http.StatusUnprocessableEntity, &tmResp)

	// create a team with a whitespace-only enroll secret
	tmResp.Team = nil
	s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{
		Name:    name + "whitespace_secret",
		Secrets: []*fleet.EnrollSecret{{Secret: "   "}},
	}, http.StatusUnprocessableEntity, &tmResp)

	// create a team with invalid host expiry window
	team4 := &fleet.TeamPayload{
		Name:        new(name + "invalid host_expiry_window"),
		Description: new("Team4 description"),
		Secrets:     []*fleet.EnrollSecret{{Secret: "TEAM4"}},
		HostExpirySettings: &fleet.HostExpirySettings{
			HostExpiryEnabled: true,
			HostExpiryWindow:  -1,
		},
	}
	s.DoJSON("POST", "/api/latest/fleet/teams", team4, http.StatusUnprocessableEntity, &tmResp)

	// list teams matching name of one team
	var listResp listTeamsResponse
	s.DoJSON("GET", "/api/latest/fleet/teams", nil, http.StatusOK, &listResp, "query", name, "per_page", "2")
	require.Len(t, listResp.Teams, 1)
	assert.Equal(t, team.Name, listResp.Teams[0].Name)
	assert.NotNil(t, listResp.Teams[0].Config.AgentOptions)
	tm1ID := listResp.Teams[0].ID

	// same as above, with leading/trailing whitespace
	s.DoJSON("GET", "/api/latest/fleet/teams", nil, http.StatusOK, &listResp, "query", " "+name+" ", "per_page", "2")
	require.Len(t, listResp.Teams, 1)
	assert.Equal(t, team.Name, listResp.Teams[0].Name)
	assert.NotNil(t, listResp.Teams[0].Config.AgentOptions)

	// same as above, no match
	s.DoJSON("GET", "/api/latest/fleet/teams", nil, http.StatusOK, &listResp, "query", " nope ", "per_page", "2")
	require.Empty(t, listResp.Teams)

	// get team
	var getResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, team.Name, getResp.Team.Name)
	assert.NotNil(t, getResp.Team.Config.AgentOptions)

	// modify team
	team.Description = "Alt " + team.Description
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), team, http.StatusOK, &tmResp)
	assert.Contains(t, tmResp.Team.Description, "Alt ") // nolint:nilaway // DoJSON fails the test unless the response decodes into tmResp

	// modify team's disk encryption, impossible without either Apple or Windows MDM enabled
	res := s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			EnableDiskEncryption: optjson.SetBool(true),
		},
	}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, `Couldn't update mdm.enable_disk_encryption because neither Apple MDM nor Windows MDM is turned on in Fleet.`)

	// modify a team with a NULL config
	defaultFeatures := fleet.Features{}
	defaultFeatures.ApplyDefaultsForNewInstalls()
	mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `UPDATE teams SET config = NULL WHERE id = ? `, tm1ID)
		return err
	})
	tmResp.Team = nil
	// Use a minimal payload — sending the full team struct would marshal a
	// zero-value `features.historical_data`, which the modify-team endpoint
	// now interprets as an explicit disable of historical data.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID),
		json.RawMessage(`{"description":"`+team.Description+`"}`),
		http.StatusOK, &tmResp)
	assert.Equal(t, defaultFeatures, tmResp.Team.Config.Features) // nolint:nilaway // DoJSON fails the test unless the response decodes into tmResp

	// modify a team with an empty config
	mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `UPDATE teams SET config = '{}' WHERE id = ? `, tm1ID)
		return err
	})
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID),
		json.RawMessage(`{"description":"`+team.Description+`"}`),
		http.StatusOK, &tmResp)
	assert.Equal(t, defaultFeatures, tmResp.Team.Config.Features) // nolint:nilaway // DoJSON fails the test unless the response decodes into tmResp
	assert.False(t, tmResp.Team.Config.HostExpirySettings.HostExpiryEnabled)

	// modify non-existing team
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID+1), team, http.StatusNotFound, &tmResp)

	// modify team host expiry
	modifyExpiry := fleet.TeamPayload{
		HostExpirySettings: &fleet.HostExpirySettings{
			HostExpiryEnabled: true,
			HostExpiryWindow:  10,
		},
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), modifyExpiry, http.StatusOK, &tmResp)
	assert.Equal(t, *modifyExpiry.HostExpirySettings, tmResp.Team.Config.HostExpirySettings) // nolint:nilaway // DoJSON fails the test unless the response decodes into tmResp

	// invalid team host expiry (<= 0)
	modifyExpiry.HostExpirySettings.HostExpiryWindow = 0
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), modifyExpiry, http.StatusUnprocessableEntity, &tmResp)

	// try to rename to reserved names
	r = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), fleet.TeamPayload{Name: new("no TEAM")}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), `is a reserved fleet name`)

	r = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), fleet.TeamPayload{Name: new("ALL teAMs")}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), `is a reserved fleet name`)

	// try to rename to a whitespace-only name
	r = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), fleet.TeamPayload{Name: new("     ")}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), `may not be empty`)

	// Modify team's calendar config
	modifyCalendar := fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			GoogleCalendar: &fleet.TeamGoogleCalendarIntegration{
				WebhookURL: "https://example.com/modified",
			},
		},
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), modifyCalendar, http.StatusOK, &tmResp)
	assert.Equal(t, modifyCalendar.Integrations.GoogleCalendar, tmResp.Team.Config.Integrations.GoogleCalendar)

	// Illegal team calendar config
	modifyCalendar.Integrations.GoogleCalendar.Enable = true
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), modifyCalendar, http.StatusUnprocessableEntity, &tmResp)

	// list team users
	var usersResp listUsersResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), nil, http.StatusOK, &usersResp)
	assert.Empty(t, usersResp.Users)

	// list team users - non-existing team
	usersResp.Users = nil
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID+1), nil, http.StatusNotFound, &usersResp)

	// create a new user
	user := &fleet.User{
		Name:       "Team User",
		Email:      "user@example.com",
		GlobalRole: new("observer"),
	}
	require.NoError(t, user.SetPassword(test.GoodPassword, 10, 10))
	user, err := s.ds.NewUser(context.Background(), user)
	require.NoError(t, err)

	// add a team user
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: *user, Role: fleet.RoleObserver}}}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Users, 1) // nolint:nilaway // DoJSON fails the test unless the response decodes into tmResp
	assert.Equal(t, user.ID, tmResp.Team.Users[0].ID)

	// add a team user - non-existing team
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID+1), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: *user, Role: fleet.RoleObserver}}}, http.StatusNotFound, &tmResp)

	// add a team user - invalid user role
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: *user, Role: "foobar"}}}, http.StatusUnprocessableEntity, &tmResp)

	// search for that user
	usersResp.Users = nil
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), nil, http.StatusOK, &usersResp, "query", "user")
	require.Len(t, usersResp.Users, 1)
	assert.Equal(t, user.ID, usersResp.Users[0].ID)

	// search for unknown user
	usersResp.Users = nil
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), nil, http.StatusOK, &usersResp, "query", "notauser")
	require.Empty(t, usersResp.Users)

	// delete team user
	tmResp.Team = nil
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), modifyTeamUsersRequest{Users: []fleet.TeamUser{{ID: user.ID}}}, http.StatusOK, &tmResp)
	require.Empty(t, tmResp.Team.Users) // nolint:nilaway // DoJSON fails the test unless the response decodes into tmResp

	// delete team user - unknown user
	tmResp.Team = nil
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), modifyTeamUsersRequest{Users: []fleet.TeamUser{{ID: user.ID + 1}}}, http.StatusOK, &tmResp)
	require.Empty(t, tmResp.Team.Users) // nolint:nilaway // DoJSON fails the test unless the response decodes into tmResp

	// delete team user - unknown team
	tmResp.Team = nil
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID+1), modifyTeamUsersRequest{Users: []fleet.TeamUser{{ID: user.ID}}}, http.StatusNotFound, &tmResp)

	// modify team agent options with invalid options
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"x": "y"
	}`), http.StatusBadRequest, &tmResp)

	// modify team agent options with invalid key
	badRes := s.Do("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"bad_key": 1
	}`), http.StatusBadRequest)
	errText := extractServerErrorText(badRes.Body)
	require.Contains(t, errText, "unsupported key provided")

	// modify team agent options with correct options under the wrong key
	badRes = s.Do("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"distributed_tls_max_attempts": 3
	}`), http.StatusBadRequest)
	errText = extractServerErrorText(badRes.Body)
	require.Contains(t, errText, "\"distributed_tls_max_attempts\" should be part of the \"config.options\" object")

	badRes = s.Do("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"config": { "options": { "logger_plugin": 3 } }
	}`), http.StatusBadRequest)
	errText = extractServerErrorText(badRes.Body)
	require.Contains(t, errText, "\"logger_plugin\" should be part of the \"command_line_flags\" object")

	badRes = s.Do("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"update_channels": { "config": 1 }
	}`), http.StatusBadRequest)
	errText = extractServerErrorText(badRes.Body)
	require.Contains(t, errText, "\"config\" should be part of the top level object")

	// modify team agent options with invalid platform options
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(
		`{"overrides": {
			"platforms": {
				"linux": null
			}
		}}`,
	), http.StatusBadRequest, &tmResp)

	// modify team agent options with invalid options, but force-apply them
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"config": {
			"x": "y"
		}
	}`), http.StatusOK, &tmResp, "force", "true")
	require.Contains(t, string(*tmResp.Team.Config.AgentOptions), `"x": "y"`) // nolint:nilaway // DoJSON fails the test unless the response decodes into tmResp

	// modify team agent options with valid options
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": true
			}
		}
	}`), http.StatusOK, &tmResp)
	require.Contains(t, string(*tmResp.Team.Config.AgentOptions), `"aws_debug": true`) // nolint:nilaway // DoJSON fails the test unless the response decodes into tmResp

	// modify team agent using invalid options with dry-run
	tmResp.Team = nil
	resp := s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": "not-a-bool"
			}
		}
	}`), http.StatusBadRequest, "dry_run", "true")
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "invalid value type at 'options.aws_debug': expected bool but got string")

	// modify team agent using valid options with dry-run
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": false
			}
		}
	}`), http.StatusOK, &tmResp, "dry_run", "true")
	require.Contains(t, string(*tmResp.Team.Config.AgentOptions), `"aws_debug": true`) // nolint:nilaway // left unchanged; DoJSON fails the test unless the response decodes into tmResp

	// list activities, it should have created one for edited_agent_options
	s.lastActivityMatches(fleet.ActivityTypeEditedAgentOptions{}.ActivityName(), fmt.Sprintf(`{"global": false, "team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm1ID, team.Name, tm1ID, team.Name), 0)

	// modify team agent options - unknown team
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID+1), json.RawMessage(`{}`), http.StatusNotFound, &tmResp)

	// get team enroll secrets
	var secResp teamEnrollSecretsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", tm1ID), nil, http.StatusOK, &secResp)
	require.Len(t, secResp.Secrets, 1)
	assert.Equal(t, team.Secrets[0].Secret, secResp.Secrets[0].Secret)

	// get team enroll secrets- unknown team: does not return 404 because reads directly
	// the secrets table, does not load the team first (which would be unnecessary except
	// for checking that it exists)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", tm1ID+1), nil, http.StatusOK, &secResp)
	assert.Empty(t, secResp.Secrets)

	// delete team
	var delResp deleteTeamResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), nil, http.StatusOK, &delResp)

	// delete team again, now an unknown team
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), nil, http.StatusNotFound, &delResp)
}

func (s *integrationEnterpriseTestSuite) TestTeamSecretsAreObfuscated() {
	t := s.T()

	// -----------------
	// Set up test data
	// -----------------
	teams := []*fleet.Team{
		{
			Name:        "Team One",
			Description: "Team description",
			Secrets:     []*fleet.EnrollSecret{{Secret: "DEF"}},
		},
		{
			Name:        "Team Two",
			Description: "Team Two description",
			Secrets:     []*fleet.EnrollSecret{{Secret: "ABC"}},
		},
	}
	for _, team := range teams {
		_, err := s.ds.NewTeam(context.Background(), team)
		require.NoError(t, err)
	}

	global_obs := &fleet.User{
		Name:       "Global Obs",
		Email:      "global_obs@example.com",
		GlobalRole: new(fleet.RoleObserver),
	}
	global_obs_plus := &fleet.User{
		Name:       "Global Obs+",
		Email:      "global_obs_plus@example.com",
		GlobalRole: new(fleet.RoleObserverPlus),
	}
	team_obs := &fleet.User{
		Name:  "Team Obs",
		Email: "team_obs@example.com",
		Teams: []fleet.UserTeam{
			{
				Team: *teams[0],
				Role: fleet.RoleObserver,
			},
			{
				Team: *teams[1],
				Role: fleet.RoleAdmin,
			},
		},
	}
	team_obs_plus := &fleet.User{
		Name:  "Team Obs Plus",
		Email: "team_obs_plus@example.com",
		Teams: []fleet.UserTeam{
			{
				Team: *teams[0],
				Role: fleet.RoleAdmin,
			},
			{
				Team: *teams[1],
				Role: fleet.RoleObserverPlus,
			},
		},
	}
	// team_gitops can modify its team but must not be able to read the team's
	// enroll secrets (matching the enroll_secret authorization policy).
	team_gitops := &fleet.User{
		Name:  "Team GitOps",
		Email: "team_gitops@example.com",
		Teams: []fleet.UserTeam{
			{
				Team: *teams[0],
				Role: fleet.RoleGitOps,
			},
		},
	}
	// team_admin can modify its team and is allowed to read enroll secrets, so
	// it should still receive the plaintext secret in write responses.
	team_admin := &fleet.User{
		Name:  "Team Admin",
		Email: "team_admin@example.com",
		Teams: []fleet.UserTeam{
			{
				Team: *teams[0],
				Role: fleet.RoleAdmin,
			},
		},
	}
	// global_gitops can create and modify any team but must not be able to read
	// enroll secrets.
	global_gitops := &fleet.User{
		Name:       "Global GitOps",
		Email:      "global_gitops@example.com",
		GlobalRole: new(fleet.RoleGitOps),
	}
	users := []*fleet.User{global_obs, global_obs_plus, team_obs, team_obs_plus, team_gitops, team_admin, global_gitops}
	for _, u := range users {
		require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
		_, err := s.ds.NewUser(context.Background(), u)
		require.NoError(t, err)
	}

	// --------------------------------------------------------------------
	// Global obs/obs+ should not be able to see any team secrets
	// --------------------------------------------------------------------
	for _, u := range []*fleet.User{global_obs, global_obs_plus} {

		s.setTokenForTest(t, u.Email, test.GoodPassword)

		// list all teams
		var listResp listTeamsResponse
		s.DoJSON("GET", "/api/latest/fleet/teams", nil, http.StatusOK, &listResp)

		require.Len(t, listResp.Teams, len(teams))
		require.NoError(t, listResp.Err)

		for _, team := range listResp.Teams {
			for _, secret := range team.Secrets {
				require.Equal(t, fleet.MaskedPassword, secret.Secret)
			}
		}

		// listing a team / team secrets
		for _, team := range teams {
			var getResp getTeamResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)

			require.NoError(t, getResp.Err)
			for _, secret := range getResp.Team.Secrets {
				require.Equal(t, fleet.MaskedPassword, secret.Secret)
			}

			var secResp teamEnrollSecretsResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), nil, http.StatusOK, &secResp)

			require.Len(t, secResp.Secrets, 1)
			require.NoError(t, secResp.Err)
			for _, secret := range secResp.Secrets {
				require.Equal(t, fleet.MaskedPassword, secret.Secret)
			}
		}
	}

	// --------------------------------------------------------------------
	// Team obs/obs+ should not be able to see their team secrets
	// --------------------------------------------------------------------
	for _, u := range []*fleet.User{team_obs, team_obs_plus} {

		s.setTokenForTest(t, u.Email, test.GoodPassword)

		// list all teams
		var listResp listTeamsResponse
		s.DoJSON("GET", "/api/latest/fleet/teams", nil, http.StatusOK, &listResp)

		require.Len(t, listResp.Teams, len(u.Teams))
		require.NoError(t, listResp.Err)

		for _, team := range listResp.Teams {
			for _, secret := range team.Secrets {
				// team_obs has RoleObserver in Team 1, and an RoleAdmin in Team 2
				// so it should be able to see the secrets in Team 1
				if u.ID == team_obs.ID {
					require.Equal(t, fleet.MaskedPassword == secret.Secret, team.ID == teams[0].ID)
					require.Equal(t, fleet.MaskedPassword != secret.Secret, team.ID == teams[1].ID)
				}

				// team_obs_plus should not be able to see any Team Secret
				if u.ID == team_obs_plus.ID {
					require.Equal(t, fleet.MaskedPassword == secret.Secret, team.ID == teams[1].ID)
					require.Equal(t, fleet.MaskedPassword != secret.Secret, team.ID == teams[0].ID)
				}
			}
		}

		// listing a team / team secrets
		for _, team := range teams {
			var getResp getTeamResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)

			require.NoError(t, getResp.Err)
			// team_obs has RoleObserver in Team 1, and an RoleAdmin in Team 2
			// so it should be able to see the secrets in Team 1
			for _, secret := range getResp.Team.Secrets {
				if u.ID == team_obs.ID {
					require.Equal(t, fleet.MaskedPassword == secret.Secret, team.ID == teams[0].ID)
					require.Equal(t, fleet.MaskedPassword != secret.Secret, team.ID == teams[1].ID)
				}

				if u.ID == team_obs_plus.ID {
					require.Equal(t, fleet.MaskedPassword == secret.Secret, team.ID == teams[1].ID)
					require.Equal(t, fleet.MaskedPassword != secret.Secret, team.ID == teams[0].ID)
				}
			}

			var secResp teamEnrollSecretsResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), nil, http.StatusOK, &secResp)

			require.Len(t, secResp.Secrets, 1)
			require.NoError(t, secResp.Err)
			for _, secret := range secResp.Secrets {
				if u.ID == team_obs.ID {
					require.Equal(t, fleet.MaskedPassword == secret.Secret, team.ID == teams[0].ID)
					require.Equal(t, fleet.MaskedPassword != secret.Secret, team.ID == teams[1].ID)
				}

				if u.ID == team_obs_plus.ID {
					require.Equal(t, fleet.MaskedPassword == secret.Secret, team.ID == teams[1].ID)
					require.Equal(t, fleet.MaskedPassword != secret.Secret, team.ID == teams[0].ID)
				}
			}
		}
	}

	// --------------------------------------------------------------------
	// A team gitops user can modify a team but must not receive plaintext
	// enroll secrets in the PATCH response (regression test for the write
	// endpoint leaking secrets to a role that cannot read them).
	// --------------------------------------------------------------------
	s.setTokenForTest(t, team_gitops.Email, test.GoodPassword)

	// gitops is forbidden from reading the team and its secrets directly.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", teams[0].ID), nil, http.StatusForbidden, &getTeamResponse{})
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", teams[0].ID), nil, http.StatusForbidden, &teamEnrollSecretsResponse{})

	// ...but the PATCH write response must mask the secret.
	var gitopsPatchResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", teams[0].ID),
		fleet.TeamPayload{Description: new("patched by gitops")}, http.StatusOK, &gitopsPatchResp)
	require.NotEmpty(t, gitopsPatchResp.Team.Secrets)
	for _, secret := range gitopsPatchResp.Team.Secrets {
		require.Equal(t, fleet.MaskedPassword, secret.Secret)
	}

	// The modify agent options write endpoint must mask the secret too.
	var gitopsAgentResp teamResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", teams[0].ID),
		json.RawMessage(`{}`), http.StatusOK, &gitopsAgentResp)
	require.NotEmpty(t, gitopsAgentResp.Team.Secrets)
	for _, secret := range gitopsAgentResp.Team.Secrets {
		require.Equal(t, fleet.MaskedPassword, secret.Secret)
	}

	// --------------------------------------------------------------------
	// A global gitops user must not receive plaintext secrets from the
	// PATCH response nor when creating a team (the create response can leak
	// the server-generated enroll secret).
	// --------------------------------------------------------------------
	s.setTokenForTest(t, global_gitops.Email, test.GoodPassword)

	var globalGitopsPatchResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", teams[0].ID),
		fleet.TeamPayload{Description: new("patched by global gitops")}, http.StatusOK, &globalGitopsPatchResp)
	require.NotEmpty(t, globalGitopsPatchResp.Team.Secrets)
	for _, secret := range globalGitopsPatchResp.Team.Secrets {
		require.Equal(t, fleet.MaskedPassword, secret.Secret)
	}

	var gitopsCreateResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams",
		fleet.TeamPayload{Name: new("gitops-created-team")}, http.StatusOK, &gitopsCreateResp)
	require.NotEmpty(t, gitopsCreateResp.Team.Secrets)
	for _, secret := range gitopsCreateResp.Team.Secrets {
		require.Equal(t, fleet.MaskedPassword, secret.Secret)
	}

	// --------------------------------------------------------------------
	// A team admin can read enroll secrets, so the PATCH response must
	// still contain the plaintext secret.
	// --------------------------------------------------------------------
	s.setTokenForTest(t, team_admin.Email, test.GoodPassword)

	var adminPatchResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", teams[0].ID),
		fleet.TeamPayload{Description: new("patched by admin")}, http.StatusOK, &adminPatchResp)
	require.NotEmpty(t, adminPatchResp.Team.Secrets)
	for _, secret := range adminPatchResp.Team.Secrets {
		require.NotEqual(t, fleet.MaskedPassword, secret.Secret)
	}

	// A global admin creating a team can read the generated secret.
	s.token = s.getTestAdminToken()
	var adminCreateResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams",
		fleet.TeamPayload{Name: new("admin-created-team")}, http.StatusOK, &adminCreateResp)
	require.NotEmpty(t, adminCreateResp.Team.Secrets)
	for _, secret := range adminCreateResp.Team.Secrets {
		require.NotEqual(t, fleet.MaskedPassword, secret.Secret)
	}
}

func (s *integrationEnterpriseTestSuite) TestExternalIntegrationsTeamConfig() {
	t := s.T()

	// create a test http server to act as the Jira and Zendesk server
	srvURL := startExternalServiceWebServer(t)

	// create a new team
	team := &fleet.Team{
		Name:        t.Name(),
		Description: "Team description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "XYZ"}},
	}
	var tmResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &tmResp)
	require.Equal(t, team.Name, tmResp.Team.Name)
	require.Len(t, tmResp.Team.Secrets, 1)
	require.Equal(t, "XYZ", tmResp.Team.Secrets[0].Secret)
	team.ID = tmResp.Team.ID

	// modify the team's config - enable the webhook
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com",
		},
		HostStatusWebhook: &fleet.HostStatusWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com/host_status_webhook",
		},
	}}, http.StatusOK, &tmResp)
	require.True(t, tmResp.Team.Config.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Equal(t, "http://example.com", tmResp.Team.Config.WebhookSettings.FailingPoliciesWebhook.DestinationURL)
	require.True(t, tmResp.Team.Config.WebhookSettings.HostStatusWebhook.Enable)
	require.Equal(t, "http://example.com/host_status_webhook", tmResp.Team.Config.WebhookSettings.HostStatusWebhook.DestinationURL)

	// enable the host activities webhook
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com",
		},
		HostActivitiesWebhook: &fleet.HostActivitiesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com/host_activities_webhook",
		},
	}}, http.StatusOK, &tmResp)
	require.NotNil(t, tmResp.Team.Config.WebhookSettings.HostActivitiesWebhook)
	require.True(t, tmResp.Team.Config.WebhookSettings.HostActivitiesWebhook.Enable)
	require.Equal(t, "http://example.com/host_activities_webhook", tmResp.Team.Config.WebhookSettings.HostActivitiesWebhook.DestinationURL)

	// a webhook_settings PATCH that omits host_activities_webhook preserves the stored value
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com",
		},
	}}, http.StatusOK, &tmResp)
	require.NotNil(t, tmResp.Team.Config.WebhookSettings.HostActivitiesWebhook)
	require.True(t, tmResp.Team.Config.WebhookSettings.HostActivitiesWebhook.Enable)
	require.Equal(t, "http://example.com/host_activities_webhook", tmResp.Team.Config.WebhookSettings.HostActivitiesWebhook.DestinationURL)
	require.NotNil(t, tmResp.Team.Config.WebhookSettings.HostStatusWebhook)
	require.True(t, tmResp.Team.Config.WebhookSettings.HostStatusWebhook.Enable)

	// a PATCH with only host_status_webhook (the fleet settings page) keeps the other webhooks
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		HostStatusWebhook: &fleet.HostStatusWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com/host_status_webhook2",
		},
	}}, http.StatusOK, &tmResp)
	require.Equal(t, "http://example.com/host_status_webhook2", tmResp.Team.Config.WebhookSettings.HostStatusWebhook.DestinationURL)
	require.True(t, tmResp.Team.Config.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Equal(t, "http://example.com", tmResp.Team.Config.WebhookSettings.FailingPoliciesWebhook.DestinationURL)
	require.True(t, tmResp.Team.Config.WebhookSettings.HostActivitiesWebhook.Enable)

	// enabling with an empty destination URL fails validation
	res := s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		HostActivitiesWebhook: &fleet.HostActivitiesWebhookSettings{
			Enable: true,
		},
	}}, http.StatusUnprocessableEntity)
	errText := extractServerErrorText(res.Body)
	require.Contains(t, errText, "destination_url is required to enable the host activities webhook")

	// enabling with a non-http(s) destination URL fails validation
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		HostActivitiesWebhook: &fleet.HostActivitiesWebhookSettings{
			Enable:         true,
			DestinationURL: "ftp://example.com",
		},
	}}, http.StatusUnprocessableEntity)
	errText = extractServerErrorText(res.Body)
	require.Contains(t, errText, "destination_url must be https or http")

	// explicitly disabling works
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com",
		},
		HostActivitiesWebhook: &fleet.HostActivitiesWebhookSettings{
			Enable: false,
		},
	}}, http.StatusOK, &tmResp)
	require.NotNil(t, tmResp.Team.Config.WebhookSettings.HostActivitiesWebhook)
	require.False(t, tmResp.Team.Config.WebhookSettings.HostActivitiesWebhook.Enable)

	// an empty host_activities_webhook object passes validation (enable defaults
	// to false) and replaces the stored value: disabled with the URL cleared
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com",
		},
		HostActivitiesWebhook: &fleet.HostActivitiesWebhookSettings{},
	}}, http.StatusOK, &tmResp)
	require.NotNil(t, tmResp.Team.Config.WebhookSettings.HostActivitiesWebhook)
	require.False(t, tmResp.Team.Config.WebhookSettings.HostActivitiesWebhook.Enable)
	require.Empty(t, tmResp.Team.Config.WebhookSettings.HostActivitiesWebhook.DestinationURL)
	require.True(t, tmResp.Team.Config.WebhookSettings.FailingPoliciesWebhook.Enable)

	// add an unknown automation - does not exist at the global level
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{Integrations: &fleet.TeamIntegrations{
		Jira: []*fleet.TeamJiraIntegration{
			{
				URL:                   srvURL,
				ProjectKey:            "qux",
				EnableFailingPolicies: false,
			},
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// add a couple Jira integrations at the global level (qux and qux2)
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux"
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux2"
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	// enable an automation - should fail as the webhook is enabled too
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{Integrations: &fleet.TeamIntegrations{
		Jira: []*fleet.TeamJiraIntegration{
			{
				URL:                   srvURL,
				ProjectKey:            "qux",
				EnableFailingPolicies: true,
			},
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// get the team, no integration was saved
	var getResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)
	require.Empty(t, getResp.Team.Config.Integrations.Jira)
	require.Empty(t, getResp.Team.Config.Integrations.Zendesk)

	// disable the webhook and enable the automation
	tmResp = teamResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
			},
		},
		WebhookSettings: &fleet.TeamSpecWebhookSettings{
			FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
				Enable:         false,
				DestinationURL: "http://example.com",
			},
		},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Jira, 1)
	require.Equal(t, "qux", tmResp.Team.Config.Integrations.Jira[0].ProjectKey)

	// update the team with an unrelated field, should not change integrations
	tmResp = teamResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Description: new("team-desc"),
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Jira, 1)
	require.Equal(t, "team-desc", tmResp.Team.Description)

	// make an unrelated appconfig change, should not remove the global integrations nor the teams'
	var appCfgResp appConfigResponse
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"org_info": {
			"org_name": "test-integrations"
		}
	}`), http.StatusOK, &appCfgResp)
	require.Equal(t, "test-integrations", appCfgResp.OrgInfo.OrgName)
	require.Len(t, appCfgResp.Integrations.Jira, 2)

	// enable the webhook without changing the integration should fail (an integration is already enabled)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com",
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// add a second, disabled Jira integration
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					ProjectKey:            "qux2",
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Jira, 2)
	require.Equal(t, "qux", tmResp.Team.Config.Integrations.Jira[0].ProjectKey)
	require.Equal(t, "qux2", tmResp.Team.Config.Integrations.Jira[1].ProjectKey)

	// enabling the second without disabling the first fails
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					ProjectKey:            "qux2",
					EnableFailingPolicies: true,
				},
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// updating to use the same project key fails (must be unique)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// remove second integration, disable first so that nothing is enabled now
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusOK, &tmResp)

	// enable the webhook now works
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com",
		},
	}}, http.StatusOK, &tmResp)

	// set environmental varible to use Zendesk test client
	t.Setenv("TEST_ZENDESK_CLIENT", "true")

	// add an unknown automation - does not exist at the global level
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{Integrations: &fleet.TeamIntegrations{
		Zendesk: []*fleet.TeamZendeskIntegration{
			{
				URL:                   srvURL,
				GroupID:               122,
				EnableFailingPolicies: false,
			},
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// add a couple Zendesk integrations at the global level (122 and 123), keep the jira ones too
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "a@b.c",
					"api_token": "ok",
					"group_id": 122
				},
				{
					"url": %[1]q,
					"email": "b@b.c",
					"api_token": "ok",
					"group_id": 123
				}
			],
			"jira": [
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux"
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux2"
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	// enable a Zendesk automation - should fail as the webhook is enabled too
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{Integrations: &fleet.TeamIntegrations{
		Zendesk: []*fleet.TeamZendeskIntegration{
			{
				URL:                   srvURL,
				GroupID:               122,
				EnableFailingPolicies: true,
			},
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// disable the webhook and enable the automation
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
			},
		},
		WebhookSettings: &fleet.TeamSpecWebhookSettings{
			FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
				Enable:         false,
				DestinationURL: "http://example.com",
			},
		},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), tmResp.Team.Config.Integrations.Zendesk[0].GroupID)

	// enable the webhook without changing the integration should fail (an integration is already enabled)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com",
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// add a second, disabled Zendesk integration
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Zendesk, 2)
	require.Equal(t, int64(122), tmResp.Team.Config.Integrations.Zendesk[0].GroupID)
	require.Equal(t, int64(123), tmResp.Team.Config.Integrations.Zendesk[1].GroupID)

	// update the team with an unrelated field, should not change integrations
	tmResp = teamResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Description: new("team-desc-2"),
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Zendesk, 2)
	require.Equal(t, "team-desc-2", tmResp.Team.Description)

	// make an unrelated appconfig change, should not remove the global integrations nor the teams'
	appCfgResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"org_info": {
			"org_name": "test-integrations-2"
		}
	}`), http.StatusOK, &appCfgResp)
	require.Equal(t, "test-integrations-2", appCfgResp.OrgInfo.OrgName)
	require.Len(t, appCfgResp.Integrations.Zendesk, 2)
	require.Len(t, appCfgResp.Integrations.Jira, 2)

	// enabling the second without disabling the first fails
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: true,
				},
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// updating to use the same group ID fails (must be unique per group ID)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// remove second Zendesk integration, add disabled Jira integration
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
			},
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Jira, 1)
	require.Equal(t, "qux", tmResp.Team.Config.Integrations.Jira[0].ProjectKey)
	require.Len(t, tmResp.Team.Config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), tmResp.Team.Config.Integrations.Zendesk[0].GroupID)

	// enabling a Jira integration when a Zendesk one is enabled fails
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
			},
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// set additional integrations on the team
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: false,
				},
			},
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusOK, &tmResp)

	// removing Zendesk 122 from the global config removes it from the team too
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %[1]q,
					"email": "b@b.c",
					"api_token": "ok",
					"group_id": 123
				}
			],
			"jira": [
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux"
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux2"
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	// get the team, only one Zendesk integration remains, none are enabled
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)
	require.Len(t, getResp.Team.Config.Integrations.Jira, 1)
	require.Equal(t, "qux", getResp.Team.Config.Integrations.Jira[0].ProjectKey)
	require.False(t, getResp.Team.Config.Integrations.Jira[0].EnableFailingPolicies)
	require.Len(t, getResp.Team.Config.Integrations.Zendesk, 1)
	require.Equal(t, int64(123), getResp.Team.Config.Integrations.Zendesk[0].GroupID)
	require.False(t, getResp.Team.Config.Integrations.Zendesk[0].EnableFailingPolicies)

	// removing Jira qux2 from the global config does not impact the team as it is unused.
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %[1]q,
					"email": "b@b.c",
					"api_token": "ok",
					"group_id": 123
				}
			],
			"jira": [
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux"
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	// get the team, integrations are unchanged
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)
	require.Len(t, getResp.Team.Config.Integrations.Jira, 1)
	require.Equal(t, "qux", getResp.Team.Config.Integrations.Jira[0].ProjectKey)
	require.False(t, getResp.Team.Config.Integrations.Jira[0].EnableFailingPolicies)
	require.Len(t, getResp.Team.Config.Integrations.Zendesk, 1)
	require.Equal(t, int64(123), getResp.Team.Config.Integrations.Zendesk[0].GroupID)
	require.False(t, getResp.Team.Config.Integrations.Zendesk[0].EnableFailingPolicies)

	// enable Jira qux for the team
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: false,
				},
			},
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
			},
		},
	}, http.StatusOK, &tmResp)

	// removing Zendesk 123 from the global config removes it from the team but
	// leaves the Jira integration enabled.
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux"
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)
	require.Len(t, getResp.Team.Config.Integrations.Jira, 1)
	require.Equal(t, "qux", getResp.Team.Config.Integrations.Jira[0].ProjectKey)
	require.True(t, getResp.Team.Config.Integrations.Jira[0].EnableFailingPolicies)
	require.Empty(t, getResp.Team.Config.Integrations.Zendesk)

	// remove all integrations on exit, so that other tests can enable the
	// webhook as needed
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{},
			Jira:    []*fleet.TeamJiraIntegration{},
		},
		WebhookSettings: &fleet.TeamSpecWebhookSettings{FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{}},
	}, http.StatusOK, &tmResp)
	require.Empty(t, tmResp.Team.Config.Integrations.Jira)
	require.Empty(t, tmResp.Team.Config.Integrations.Zendesk)
	require.False(t, tmResp.Team.Config.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Empty(t, tmResp.Team.Config.WebhookSettings.FailingPoliciesWebhook.DestinationURL)

	appCfgResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"integrations": {
			"jira": [],
			"zendesk": []
		}
	}`), http.StatusOK, &appCfgResp)
	require.Empty(t, appCfgResp.Integrations.Jira)
	require.Empty(t, appCfgResp.Integrations.Zendesk)
}

func (s *integrationEnterpriseTestSuite) TestNoTeamWebhookConfig() {
	t := s.T()

	// Test that we can configure webhooks for "No Team" (team ID 0)
	// Use a generic response that will work with DefaultTeam
	var defaultTeamResp struct {
		Team *fleet.DefaultTeam `json:"team"` //nolint:apiparamcheck // test helper; matches server response shape
	}

	// First clear any existing webhook configuration for "No Team"
	s.DoJSON("PATCH", "/api/latest/fleet/teams/0", fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
			Enable: false,
		},
	}}, http.StatusOK, &defaultTeamResp)

	// Get the default team config (team ID 0) - should be disabled now
	s.DoJSON("GET", "/api/latest/fleet/teams/0", nil, http.StatusOK, &defaultTeamResp)
	require.Equal(t, uint(0), defaultTeamResp.Team.ID)
	require.Equal(t, fleet.ReservedNameNoTeam, defaultTeamResp.Team.Name)
	require.False(t, defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.Enable)

	// Configure webhook settings for "No Team"
	s.DoJSON("PATCH", "/api/latest/fleet/teams/0", fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "https://example.com/no-team-webhook",
			PolicyIDs:      []uint{1, 2, 3},
			HostBatchSize:  100,
		},
	}}, http.StatusOK, &defaultTeamResp)
	require.Equal(t, uint(0), defaultTeamResp.Team.ID)
	require.Equal(t, fleet.ReservedNameNoTeam, defaultTeamResp.Team.Name)
	require.True(t, defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Equal(t, "https://example.com/no-team-webhook", defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.DestinationURL)
	require.Equal(t, []uint{1, 2, 3}, defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)
	require.Equal(t, 100, defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.HostBatchSize)

	// Get the config again to verify it persisted
	defaultTeamResp = struct {
		Team *fleet.DefaultTeam `json:"team"` //nolint:apiparamcheck // test helper; matches server response shape
	}{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0", nil, http.StatusOK, &defaultTeamResp)
	require.Equal(t, uint(0), defaultTeamResp.Team.ID)
	require.Equal(t, fleet.ReservedNameNoTeam, defaultTeamResp.Team.Name)
	require.True(t, defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Equal(t, "https://example.com/no-team-webhook", defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.DestinationURL)
	require.Equal(t, []uint{1, 2, 3}, defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)
	require.Equal(t, 100, defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.HostBatchSize)

	// Update the webhook settings
	s.DoJSON("PATCH", "/api/latest/fleet/teams/0", fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
			Enable:         false,
			DestinationURL: "https://example.com/updated",
			PolicyIDs:      []uint{4, 5},
			HostBatchSize:  200,
		},
	}}, http.StatusOK, &defaultTeamResp)
	require.False(t, defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Equal(t, "https://example.com/updated", defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.DestinationURL)
	require.Equal(t, []uint{4, 5}, defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)
	require.Equal(t, 200, defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.HostBatchSize)

	// Clear the webhook settings
	s.DoJSON("PATCH", "/api/latest/fleet/teams/0", fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
			Enable: false,
		},
	}}, http.StatusOK, &defaultTeamResp)
	require.False(t, defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.Enable)

	// Configure the host activities webhook for "No Team"
	s.DoJSON("PATCH", "/api/latest/fleet/teams/0", fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		HostActivitiesWebhook: &fleet.HostActivitiesWebhookSettings{
			Enable:         true,
			DestinationURL: "https://example.com/no-team-activities-webhook",
		},
	}}, http.StatusOK, &defaultTeamResp)
	require.NotNil(t, defaultTeamResp.Team.WebhookSettings.HostActivitiesWebhook)
	require.True(t, defaultTeamResp.Team.WebhookSettings.HostActivitiesWebhook.Enable)
	require.Equal(t, "https://example.com/no-team-activities-webhook", defaultTeamResp.Team.WebhookSettings.HostActivitiesWebhook.DestinationURL)

	// Verify it persisted
	defaultTeamResp = struct {
		Team *fleet.DefaultTeam `json:"team"` //nolint:apiparamcheck // test helper; matches server response shape
	}{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0", nil, http.StatusOK, &defaultTeamResp)
	require.NotNil(t, defaultTeamResp.Team.WebhookSettings.HostActivitiesWebhook)
	require.True(t, defaultTeamResp.Team.WebhookSettings.HostActivitiesWebhook.Enable)
	require.Equal(t, "https://example.com/no-team-activities-webhook", defaultTeamResp.Team.WebhookSettings.HostActivitiesWebhook.DestinationURL)

	// A webhook_settings PATCH that omits host_activities_webhook preserves the stored value
	s.DoJSON("PATCH", "/api/latest/fleet/teams/0", fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
			Enable: false,
		},
	}}, http.StatusOK, &defaultTeamResp)
	require.NotNil(t, defaultTeamResp.Team.WebhookSettings.HostActivitiesWebhook)
	require.True(t, defaultTeamResp.Team.WebhookSettings.HostActivitiesWebhook.Enable)

	// Enabling with an invalid destination URL fails validation
	res := s.Do("PATCH", "/api/latest/fleet/teams/0", fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		HostActivitiesWebhook: &fleet.HostActivitiesWebhookSettings{
			Enable: true,
		},
	}}, http.StatusUnprocessableEntity)
	errText := extractServerErrorText(res.Body)
	require.Contains(t, errText, "destination_url is required to enable the host activities webhook")

	// Explicitly disabling works
	s.DoJSON("PATCH", "/api/latest/fleet/teams/0", fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		HostActivitiesWebhook: &fleet.HostActivitiesWebhookSettings{
			Enable: false,
		},
	}}, http.StatusOK, &defaultTeamResp)
	require.NotNil(t, defaultTeamResp.Team.WebhookSettings.HostActivitiesWebhook)
	require.False(t, defaultTeamResp.Team.WebhookSettings.HostActivitiesWebhook.Enable)

	// A PATCH with only host_activities_webhook (the hosts page) keeps the failing policies webhook
	s.DoJSON("PATCH", "/api/latest/fleet/teams/0", fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "https://example.com/no-team-webhook",
		},
	}}, http.StatusOK, &defaultTeamResp)
	s.DoJSON("PATCH", "/api/latest/fleet/teams/0", fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		HostActivitiesWebhook: &fleet.HostActivitiesWebhookSettings{
			Enable: false,
		},
	}}, http.StatusOK, &defaultTeamResp)
	require.True(t, defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Equal(t, "https://example.com/no-team-webhook", defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.DestinationURL)

	s.DoJSON("PATCH", "/api/latest/fleet/teams/0", fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
			Enable: false,
		},
	}}, http.StatusOK, &defaultTeamResp)
}

func (s *integrationEnterpriseTestSuite) TestTeamConfigDetailQueriesOverrides() {
	t := s.T()
	ctx := context.Background()

	teamName := t.Name() + "team1"
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc team1",
	}
	s.Do("POST", "/api/latest/fleet/teams", team, http.StatusOK)

	spec := []byte(fmt.Sprintf(`
  name: %s
  features:
    additional_queries:
      time: SELECT * FROM time
    enable_host_users: true
    detail_query_overrides:
      users: null
      software_linux: "select * from blah;"
      disk_encryption_linux: null
`, teamName))

	s.applyTeamSpec(spec)
	team, err := s.ds.TeamByName(ctx, teamName)
	require.NoError(t, err)
	require.NotNil(t, team.Config.Features.DetailQueryOverrides)
	require.Nil(t, team.Config.Features.DetailQueryOverrides["users"])
	require.Nil(t, team.Config.Features.DetailQueryOverrides["disk_encryption_linux"])
	require.NotNil(t, team.Config.Features.DetailQueryOverrides["software_linux"])
	require.Equal(t, "select * from blah;", *team.Config.Features.DetailQueryOverrides["software_linux"])

	// create a linux host
	linuxHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now().Add(-10 * time.Hour),
		LabelUpdatedAt:  time.Now().Add(-10 * time.Hour),
		PolicyUpdatedAt: time.Now().Add(-10 * time.Hour),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "linux",
	})
	require.NoError(t, err)

	// add the host to team1
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team.ID, []uint{linuxHost.ID}))
	require.NoError(t, err)

	// get distributed queries for the host
	s.lq.On("QueriesForHost", linuxHost.ID).Return(map[string]string{t.Name(): "select 1 from osquery;"}, nil)
	req := getDistributedQueriesRequest{NodeKey: *linuxHost.NodeKey}
	var dqResp getDistributedQueriesResponse
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_users")
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_disk_encryption_linux")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_linux")
	require.Contains(t, dqResp.Queries, fmt.Sprintf("fleet_distributed_query_%s", t.Name()))

	spec = []byte(fmt.Sprintf(`
  name: %s
  features:
    additional_queries:
      time: SELECT * FROM time
    enable_host_users: true
    detail_query_overrides:
      software_linux: "select * from blah;"
`, teamName))

	s.applyTeamSpec(spec)
	team, err = s.ds.TeamByName(ctx, teamName)
	require.NoError(t, err)
	require.NotNil(t, team.Config.Features.DetailQueryOverrides)
	require.Nil(t, team.Config.Features.DetailQueryOverrides["users"])
	require.Nil(t, team.Config.Features.DetailQueryOverrides["disk_encryption_linux"])
	require.NotNil(t, team.Config.Features.DetailQueryOverrides["software_linux"])
	require.Equal(t, "select * from blah;", *team.Config.Features.DetailQueryOverrides["software_linux"])

	// get distributed queries for the host
	req = getDistributedQueriesRequest{NodeKey: *linuxHost.NodeKey}
	dqResp = getDistributedQueriesResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.Contains(t, dqResp.Queries, "fleet_detail_query_users")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_disk_encryption_linux")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_linux")
	require.Contains(t, dqResp.Queries, fmt.Sprintf("fleet_distributed_query_%s", t.Name()))
}

func (s *integrationEnterpriseTestSuite) TestTeamConfigHistoricalDataGitOps() {
	t := s.T()
	ctx := context.Background()

	// Create the fleet via the team-spec apply path so it goes through the
	// same plumbing GitOps uses.
	teamName := t.Name() + "fleet"
	initialSpec := fmt.Appendf(nil, `
  name: %s
`, teamName)
	s.applyTeamSpec(initialSpec)
	team, err := s.ds.TeamByName(ctx, teamName)
	require.NoError(t, err)
	require.True(t, team.Config.Features.HistoricalData.Uptime, "new fleet defaults uptime=true")
	require.True(t, team.Config.Features.HistoricalData.Vulnerabilities, "new fleet defaults vulnerabilities=true")
	t.Cleanup(func() {
		require.NoError(t, s.ds.DeleteTeam(ctx, team.ID))
	})

	// Apply explicit historical_data values via team spec.
	specWithHistoricalData := fmt.Appendf(nil, `
  name: %s
  features:
    historical_data:
      uptime: true
      vulnerabilities: false
`, teamName)
	s.applyTeamSpec(specWithHistoricalData)
	team, err = s.ds.TeamByName(ctx, teamName)
	require.NoError(t, err)
	require.True(t, team.Config.Features.HistoricalData.Uptime)
	require.False(t, team.Config.Features.HistoricalData.Vulnerabilities)

	// applyTeamSpec uses the spec/teams endpoint, not gitops, so the
	// client-side default-injection only happens via fleetctl's gitops
	// pathway (exercised in cmd/fleetctl/fleetctl/gitops_test.go's
	// TestGitOpsBasicTeam-style flows). Here we just confirm explicit
	// values round-trip via the spec/teams path; the gitops-specific
	// default-injection contract is tested in fleetctl's test suite.
}

func (s *integrationEnterpriseTestSuite) TestTeamHostNameTemplate() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	const tmpl = "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL"
	activityName := fleet.ActivityTypeEditedHostNameTemplate{}.ActivityName()
	activityDetails := func(nameTemplate string) string {
		jsonTemplate := "null"
		if nameTemplate != "" {
			jsonTemplate = fmt.Sprintf("%q", nameTemplate)
		}
		return fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "name_template": %s}`, team.ID, team.Name, jsonTemplate)
	}
	getTemplate := func() string {
		var getResp getTeamResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)
		return getResp.Team.Config.MDM.HostNameTemplate
	}

	// PATCH /teams/{id} with an invalid template is rejected
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID),
		json.RawMessage(`{"mdm": {"name_template": "X-$FLEET_SECRET_Y"}}`), http.StatusUnprocessableEntity)
	require.Empty(t, getTemplate())

	// PATCH sets the template and logs the activity
	var tmResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID),
		json.RawMessage(fmt.Sprintf(`{"mdm": {"name_template": %q}}`, tmpl)), http.StatusOK, &tmResp)
	require.Equal(t, tmpl, tmResp.Team.Config.MDM.HostNameTemplate)
	require.Equal(t, tmpl, getTemplate())
	activityID := s.lastActivityMatches(activityName, activityDetails(tmpl), 0)

	// re-PATCHing the identical template logs no duplicate activity
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID),
		json.RawMessage(fmt.Sprintf(`{"mdm": {"name_template": %q}}`, tmpl)), http.StatusOK, &tmResp)
	s.lastActivityMatches("", "", activityID)

	// a PATCH without the key leaves the template untouched
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID),
		json.RawMessage(`{"mdm": {"windows_require_bitlocker_pin": false}}`), http.StatusOK, &tmResp)
	require.Equal(t, tmpl, getTemplate())

	// team-spec apply: an invalid template is rejected, even in dry-run
	specWith := func(mdm map[string]any) map[string]any {
		return map[string]any{"specs": []any{map[string]any{"name": team.Name, "mdm": mdm}}}
	}
	s.Do("POST", "/api/latest/fleet/spec/fleets",
		specWith(map[string]any{"name_template": "X-$FLEET_VAR_NOPE"}), http.StatusUnprocessableEntity)
	require.Equal(t, tmpl, getTemplate())
	s.Do("POST", "/api/latest/fleet/spec/fleets",
		specWith(map[string]any{"name_template": "X-$FLEET_VAR_NOPE"}), http.StatusUnprocessableEntity, "dry_run", "true")
	require.Equal(t, tmpl, getTemplate())

	// a valid dry-run does not persist the change
	s.Do("POST", "/api/latest/fleet/spec/fleets",
		specWith(map[string]any{"name_template": "DR-$FLEET_VAR_HOST_UUID"}), http.StatusOK, "dry_run", "true")
	require.Equal(t, tmpl, getTemplate())

	// spec apply sets a new template and logs the activity
	const specTmpl = "iPad $FLEET_VAR_HOST_UUID"
	var applyResp applyTeamSpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/fleets",
		specWith(map[string]any{"name_template": specTmpl}), http.StatusOK, &applyResp)
	require.Equal(t, specTmpl, getTemplate())
	activityID = s.lastActivityOfTypeMatches(activityName, activityDetails(specTmpl), 0)

	// re-applying the same spec logs no duplicate activity
	s.DoJSON("POST", "/api/latest/fleet/spec/fleets",
		specWith(map[string]any{"name_template": specTmpl}), http.StatusOK, &applyResp)
	s.lastActivityOfTypeMatches(activityName, "", activityID)

	// a spec without the key leaves the template untouched
	s.DoJSON("POST", "/api/latest/fleet/spec/fleets",
		specWith(map[string]any{}), http.StatusOK, &applyResp)
	require.Equal(t, specTmpl, getTemplate())

	// an empty string clears the template and logs a null-template activity
	s.DoJSON("POST", "/api/latest/fleet/spec/fleets",
		specWith(map[string]any{"name_template": ""}), http.StatusOK, &applyResp)
	require.Empty(t, getTemplate())
	s.lastActivityOfTypeMatches(activityName, activityDetails(""), 0)

	// creating a new team from a spec with a template applies it and logs the activity
	newTeamName := t.Name() + "-new"
	s.DoJSON("POST", "/api/latest/fleet/spec/fleets",
		map[string]any{"specs": []any{map[string]any{"name": newTeamName, "mdm": map[string]any{"name_template": tmpl}}}},
		http.StatusOK, &applyResp)
	newTeam, err := s.ds.TeamByName(ctx, newTeamName)
	require.NoError(t, err)
	require.Equal(t, tmpl, newTeam.Config.MDM.HostNameTemplate)
	s.lastActivityOfTypeMatches(activityName,
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "name_template": %q}`, newTeam.ID, newTeam.Name, tmpl), 0)
}
