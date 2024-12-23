package service

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/calendar"
	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/cron"
	"github.com/fleetdm/fleet/v4/server/datastore/filesystem"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query/live_query_mock"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	commonCalendar "github.com/fleetdm/fleet/v4/server/service/calendar"
	"github.com/fleetdm/fleet/v4/server/service/redis_lock"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/go-kit/log"
	kitlog "github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	googleCalendar "google.golang.org/api/calendar/v3"
	"gopkg.in/guregu/null.v3"
)

func TestIntegrationsEnterprise(t *testing.T) {
	testingSuite := new(integrationEnterpriseTestSuite)
	testingSuite.withServer.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type integrationEnterpriseTestSuite struct {
	withServer
	suite.Suite
	redisPool            fleet.RedisPool
	calendarSchedule     *schedule.Schedule
	softwareInstallStore fleet.SoftwareInstallerStore

	lq *live_query_mock.MockLiveQuery
}

func (s *integrationEnterpriseTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationEnterpriseTestSuite")

	s.redisPool = redistest.SetupRedis(s.T(), "integration_enterprise", false, false, false)
	s.lq = live_query_mock.New(s.T())
	var calendarSchedule *schedule.Schedule

	// Create a software install store
	dir := s.T().TempDir()
	softwareInstallStore, err := filesystem.NewSoftwareInstallerStore(dir)
	require.NoError(s.T(), err)
	s.softwareInstallStore = softwareInstallStore

	config := TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierPremium,
		},
		Pool:           s.redisPool,
		Rs:             pubsub.NewInmemQueryResults(),
		Lq:             s.lq,
		Logger:         log.NewLogfmtLogger(os.Stdout),
		EnableCachedDS: true,
		StartCronSchedules: []TestNewScheduleFunc{
			func(ctx context.Context, ds fleet.Datastore) fleet.NewCronScheduleFunc {
				return func() (fleet.CronSchedule, error) {
					// We set 24-hour interval so that it only runs when triggered.
					var err error
					cronLog := log.NewJSONLogger(os.Stdout)
					if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
						cronLog = kitlog.NewNopLogger()
					}
					calendarSchedule, err = cron.NewCalendarSchedule(
						ctx, s.T().Name(), s.ds, redis_lock.NewLock(s.redisPool), config.CalendarConfig{Periodicity: 24 * time.Hour},
						cronLog,
					)
					return calendarSchedule, err
				}
			},
		},
		SoftwareInstallStore: softwareInstallStore,
	}
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		config.Logger = kitlog.NewNopLogger()
	}
	users, server := RunServerForTestsWithDS(s.T(), s.ds, &config)
	s.server = server
	s.users = users
	s.token = s.getTestAdminToken()
	s.cachedTokens = make(map[string]string)
	s.calendarSchedule = calendarSchedule
}

func (s *integrationEnterpriseTestSuite) TearDownTest() {
	// reset the mock
	s.lq.Mock = mock.Mock{}
	s.withServer.commonTearDownTest(s.T())
}

func (s *integrationEnterpriseTestSuite) TestTeamSpecs() {
	t := s.T()

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
						"minimum_version": "10.15.0",
						"deadline":        "2021-01-01",
					},
					"ios_updates": map[string]any{
						"minimum_version": "17.5.1",
						"deadline":        "2024-07-23",
					},
					"ipados_updates": map[string]any{
						"minimum_version": "18.0",
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
		AdditionalQueries:       ptr.RawMessage(json.RawMessage(`{"foo": "bar"}`)),
	}, team.Config.Features)
	require.Equal(t, fleet.TeamMDM{
		MacOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("10.15.0"),
			Deadline:       optjson.SetString("2021-01-01"),
		},
		IOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("17.5.1"),
			Deadline:       optjson.SetString("2024-07-23"),
		},
		IPadOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("18.0"),
			Deadline:       optjson.SetString("2024-08-24"),
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
		},
		// because the WindowsSettings was marshalled to JSON to be saved in the DB,
		// it did get marshalled, and then when unmarshalled it was set (but
		// empty).
		WindowsSettings: fleet.WindowsSettings{
			CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
		},
	}, team.Config.MDM)

	// an activity was created for team spec applied
	s.lastActivityMatches(fleet.ActivityTypeAppliedSpecTeam{}.ActivityName(), fmt.Sprintf(`{"teams": [{"id": %d, "name": %q}]}`, team.ID, team.Name), 0)

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
	require.NotNil(t, team.Config.Integrations.GoogleCalendar)
	assert.Equal(t, calendarWebhookUrl, team.Config.Integrations.GoogleCalendar.WebhookURL)
	assert.True(t, team.Config.Integrations.GoogleCalendar.Enable)

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
	res := s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity, "dry_run", "true")
	errMsg := extractServerErrorText(res.Body)
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
	require.Equal(t, fleet.TeamMDM{
		MacOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("10.15.0"),
			Deadline:       optjson.SetString("2021-01-01"),
		},
		IOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("17.5.1"),
			Deadline:       optjson.SetString("2024-07-23"),
		},
		IPadOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("18.0"),
			Deadline:       optjson.SetString("2024-08-24"),
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
		},
		WindowsSettings: fleet.WindowsSettings{
			CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
		},
	}, team.Config.MDM)

	// get the team via the GET endpoint, check that it properly returns the mdm settings
	var getTmResp getTeamResponse
	s.DoJSON("GET", "/api/latest/fleet/teams/"+fmt.Sprint(team.ID), nil, http.StatusOK, &getTmResp)
	require.Equal(t, fleet.TeamMDM{
		MacOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("10.15.0"),
			Deadline:       optjson.SetString("2021-01-01"),
		},
		IOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("17.5.1"),
			Deadline:       optjson.SetString("2024-07-23"),
		},
		IPadOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("18.0"),
			Deadline:       optjson.SetString("2024-08-24"),
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
		},
		WindowsSettings: fleet.WindowsSettings{
			CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
		},
	}, getTmResp.Team.Config.MDM)

	// get the team via the list teams endpoint, check that it properly returns the mdm settings
	var listTmResp listTeamsResponse
	s.DoJSON("GET", "/api/latest/fleet/teams", nil, http.StatusOK, &listTmResp, "query", teamName)
	require.True(t, len(listTmResp.Teams) > 0)
	require.Equal(t, team.ID, listTmResp.Teams[0].ID)
	require.Equal(t, fleet.TeamMDM{
		MacOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("10.15.0"),
			Deadline:       optjson.SetString("2021-01-01"),
		},
		IOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("17.5.1"),
			Deadline:       optjson.SetString("2024-07-23"),
		},
		IPadOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString("18.0"),
			Deadline:       optjson.SetString("2024-08-24"),
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
		},
		WindowsSettings: fleet.WindowsSettings{
			CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
		},
	}, listTmResp.Teams[0].Config.MDM)

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
	require.Contains(t, errMsg, "Couldn't update macos_settings because MDM features aren't turned on in Fleet.")

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

	// dry-run with macos disk encryption set to true
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
	res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity, "dry_run", "true")
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't update macos_settings because MDM features aren't turned on in Fleet.")

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
	require.Contains(t, errMsg, "Couldn't update macos_setup because MDM features aren't turned on in Fleet.")

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
	require.True(t, len(teams) >= 1)

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
	assert.True(t, len(teams) >= 2)

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
	s.lastActivityMatches(fleet.ActivityTypeAppliedSpecTeam{}.ActivityName(), fmt.Sprintf(`{"teams": [{"id": %d, "name": %q}]}`, team.ID, team.Name), 0)

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
	team1b, err := s.ds.Team(context.Background(), team1.ID)
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

func (s *integrationEnterpriseTestSuite) TestTeamSchedule() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)

	ts := getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 0)

	qr, err := s.ds.NewQuery(
		context.Background(),
		&fleet.Query{
			Name:           "TestQueryTeamPolicy",
			Description:    "Some description",
			Query:          "select * from osquery;",
			ObserverCanRun: true,
			Saved:          true,
			Logging:        fleet.LoggingSnapshot,
		},
	)
	require.NoError(t, err)

	gsParams := teamScheduleQueryRequest{ScheduledQueryPayload: fleet.ScheduledQueryPayload{
		QueryID:  &qr.ID,
		Interval: ptr.Uint(42),
	}}
	r := teamScheduleQueryResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", team1.ID), gsParams, http.StatusOK, &r)

	ts = getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 1)
	assert.Equal(t, uint(42), ts.Scheduled[0].Interval)
	assert.Contains(t, ts.Scheduled[0].Name, "Copy of TestQueryTeamPolicy")
	assert.NotEqual(t, qr.ID, ts.Scheduled[0].QueryID) // it creates a new query (copy)
	id := ts.Scheduled[0].ID

	modifyResp := modifyTeamScheduleResponse{}
	modifyParams := modifyTeamScheduleRequest{ScheduledQueryPayload: fleet.ScheduledQueryPayload{Interval: ptr.Uint(55)}}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule/%d", team1.ID, id), modifyParams, http.StatusOK, &modifyResp)

	// just to satisfy my paranoia, wanted to make sure the contents of the json would work
	s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule/%d", team1.ID, id), []byte(`{"interval": 77}`), http.StatusOK)

	ts = getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 1)
	assert.Equal(t, uint(77), ts.Scheduled[0].Interval)

	deleteResp := deleteTeamScheduleResponse{}
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule/%d", team1.ID, id), nil, http.StatusOK, &deleteResp)

	ts = getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 0)
}

func (s *integrationEnterpriseTestSuite) TestTeamPolicies() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1" + t.Name(),
		Description: "desc team1",
	})
	require.NoError(t, err)

	oldToken := s.token
	t.Cleanup(func() {
		s.token = oldToken
	})

	password := test.GoodPassword
	email := "testteam@user.com"

	u := &fleet.User{
		Name:       "test team user",
		Email:      email,
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team1,
				Role: fleet.RoleMaintainer,
			},
		},
	}
	require.NoError(t, u.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)

	s.token = s.getTestToken(email, password)

	ts := listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 0)
	require.Len(t, ts.InheritedPolicies, 0)

	// create a global policy
	gpol, err := s.ds.NewGlobalPolicy(context.Background(), nil, fleet.PolicyPayload{Name: "TestGlobalPolicy", Query: "SELECT 1"})
	require.NoError(t, err)
	defer func() {
		_, err := s.ds.DeleteGlobalPolicies(context.Background(), []uint{gpol.ID})
		require.NoError(t, err)
	}()

	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQuery2",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
		Logging:        fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	tpParams := teamPolicyRequest{
		QueryID:    &qr.ID,
		Resolution: "some team resolution",
	}
	r := teamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &r)

	ts = listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 1)
	assert.Equal(t, "TestQuery2", ts.Policies[0].Name)
	assert.Equal(t, "select * from osquery;", ts.Policies[0].Query)
	assert.Equal(t, "Some description", ts.Policies[0].Description)
	require.NotNil(t, ts.Policies[0].Resolution)
	assert.Equal(t, "some team resolution", *ts.Policies[0].Resolution)
	require.Len(t, ts.InheritedPolicies, 1)
	assert.Equal(t, gpol.Name, ts.InheritedPolicies[0].Name)
	assert.Equal(t, gpol.ID, ts.InheritedPolicies[0].ID)

	tc := countTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/count", team1.ID), nil, http.StatusOK, &tc)
	require.Nil(t, tc.Err)
	require.Equal(t, 1, tc.Count)

	gc := countGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &gc)
	require.Nil(t, gc.Err)
	require.Equal(t, 1, gc.Count)

	// Test merge inherited
	ts = listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts, "merge_inherited", "true", "order_key", "team_id", "order_direction", "desc")
	require.Len(t, ts.Policies, 2)
	require.Nil(t, ts.InheritedPolicies)
	assert.Equal(t, "TestQuery2", ts.Policies[0].Name)
	assert.Equal(t, "select * from osquery;", ts.Policies[0].Query)
	assert.Equal(t, "Some description", ts.Policies[0].Description)
	require.NotNil(t, ts.Policies[0].Resolution)
	assert.Equal(t, "some team resolution", *ts.Policies[0].Resolution)
	assert.Equal(t, gpol.Name, ts.Policies[1].Name)
	assert.Equal(t, gpol.ID, ts.Policies[1].ID)

	countResp := countTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/count", team1.ID), nil, http.StatusOK, &countResp, "merge_inherited", "true")
	require.Nil(t, countResp.Err)
	require.Equal(t, 2, countResp.Count)

	// Test delete
	deletePolicyParams := deleteTeamPoliciesRequest{IDs: []uint{ts.Policies[0].ID}}
	deletePolicyResp := deleteTeamPoliciesResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", team1.ID), deletePolicyParams, http.StatusOK, &deletePolicyResp)

	ts = listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 0)
}

func (s *integrationEnterpriseTestSuite) TestNoTeamPolicies() {
	t := s.T()
	ctx := context.Background()

	//
	// Test a global admin can read and write "No team" policies.
	//

	// List "No team" policies.
	ts := listTeamPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0/policies", nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 0)
	require.Len(t, ts.InheritedPolicies, 0)
	// Create a placeholder global policy.
	_, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{
		Name:  "globalPolicy1",
		Query: "SELECT 0;",
	})
	require.NoError(t, err)
	// Create a "No team" policy.
	tpParams := teamPolicyRequest{
		Name:  "noTeamPolicy1",
		Query: "SELECT 1;",
	}
	r := teamPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/teams/0/policies", tpParams, http.StatusOK, &r)
	require.NotNil(t, r.Policy.TeamID)
	require.Zero(t, *r.Policy.TeamID)
	// Test that we can't create a policy with the same name under "No team" domain.
	s.DoJSON("POST", "/api/latest/fleet/teams/0/policies", tpParams, http.StatusConflict, &r)
	// Create a second "No team" policy.
	tpParams = teamPolicyRequest{
		Name:  "noTeamPolicy2",
		Query: "SELECT 2;",
	}
	r = teamPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/teams/0/policies", tpParams, http.StatusOK, &r)
	require.NotNil(t, r.Policy.TeamID)
	require.Zero(t, *r.Policy.TeamID)
	// List "No team" policies.
	ts = listTeamPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0/policies", nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 2)
	assert.Equal(t, "noTeamPolicy1", ts.Policies[0].Name)
	assert.Equal(t, "SELECT 1;", ts.Policies[0].Query)
	require.NotNil(t, ts.Policies[0].TeamID)
	require.Zero(t, *ts.Policies[0].TeamID)
	assert.Equal(t, "noTeamPolicy2", ts.Policies[1].Name)
	assert.Equal(t, "SELECT 2;", ts.Policies[1].Query)
	require.NotNil(t, ts.Policies[1].TeamID)
	require.Zero(t, *ts.Policies[1].TeamID)
	require.Len(t, ts.InheritedPolicies, 1)
	assert.Equal(t, "globalPolicy1", ts.InheritedPolicies[0].Name)
	assert.Equal(t, "SELECT 0;", ts.InheritedPolicies[0].Query)
	assert.Nil(t, ts.InheritedPolicies[0].TeamID)
	// Test policy count for "No team" policies.
	tc := countTeamPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0/policies/count", nil, http.StatusOK, &tc)
	require.Equal(t, 2, tc.Count)
	// Test merge inherited for "No team" policies.
	ts = listTeamPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0/policies", nil, http.StatusOK, &ts, "merge_inherited", "true", "order_key", "team_id", "order_direction", "desc")
	require.Len(t, ts.Policies, 3)
	require.Nil(t, ts.InheritedPolicies)
	assert.Equal(t, "noTeamPolicy1", ts.Policies[0].Name)
	assert.Equal(t, "SELECT 1;", ts.Policies[0].Query)
	assert.Equal(t, "noTeamPolicy2", ts.Policies[1].Name)
	assert.Equal(t, "SELECT 2;", ts.Policies[1].Query)
	assert.Equal(t, "globalPolicy1", ts.Policies[2].Name)
	assert.Equal(t, "SELECT 0;", ts.Policies[2].Query)
	// Test merge inherited count for "No team" policies.
	countResp := countTeamPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0/policies/count", nil, http.StatusOK, &countResp, "merge_inherited", "true")
	require.Nil(t, countResp.Err)
	require.Equal(t, 3, countResp.Count)
	// Test deleting "No team" policies.
	deletePolicyParams := deleteTeamPoliciesRequest{
		IDs: []uint{ts.Policies[0].ID},
	}
	deletePolicyResp := deleteTeamPoliciesResponse{}
	s.DoJSON("POST", "/api/latest/fleet/teams/0/policies/delete", deletePolicyParams, http.StatusOK, &deletePolicyResp)
	ts = listTeamPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0/policies", nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 1)
	assert.Equal(t, "noTeamPolicy2", ts.Policies[0].Name)
	assert.Equal(t, "SELECT 2;", ts.Policies[0].Query)
	noTeamPolicy2 := ts.Policies[0]

	//
	// Test that a team admin is not allowed to access "No team" policies.
	//

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)
	oldToken := s.token
	t.Cleanup(func() {
		s.token = oldToken
	})
	password := test.GoodPassword
	email := "testteam@user.com"
	team1Admin := &fleet.User{
		Name:       "test team user",
		Email:      email,
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team1,
				Role: fleet.RoleAdmin,
			},
		},
	}
	require.NoError(t, team1Admin.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(context.Background(), team1Admin)
	require.NoError(t, err)

	s.token = s.getTestToken(email, password)

	ts = listTeamPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0/policies", nil, http.StatusForbidden, &ts)
	tpParams = teamPolicyRequest{
		Name:  "noTeamPolicy1",
		Query: "SELECT 1;",
	}
	r = teamPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/teams/0/policies", tpParams, http.StatusForbidden, &r)
	tc = countTeamPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0/policies/count", nil, http.StatusForbidden, &tc)
	deletePolicyParams = deleteTeamPoliciesRequest{
		IDs: []uint{noTeamPolicy2.ID},
	}
	s.DoJSON("POST", "/api/latest/fleet/teams/0/policies/delete", deletePolicyParams, http.StatusForbidden, &deleteTeamPoliciesResponse{})
}

func (s *integrationEnterpriseTestSuite) TestTeamQueries() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1" + t.Name(),
		Description: "desc team1",
	})
	require.NoError(t, err)

	oldToken := s.token
	t.Cleanup(func() {
		s.token = oldToken
	})

	// create global query
	params := fleet.QueryPayload{
		Name:  ptr.String("global1"),
		Query: ptr.String("select * from time;"),
	}
	var createQueryResp createQueryResponse
	s.DoJSON("POST", "/api/latest/fleet/queries", &params, http.StatusOK, &createQueryResp)
	defer s.cleanupQuery(createQueryResp.Query.ID)

	// create team query
	params = fleet.QueryPayload{
		Name:   ptr.String("team1"),
		Query:  ptr.String("select * from time;"),
		TeamID: ptr.Uint(team1.ID),
	}
	createQueryResp = createQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", &params, http.StatusOK, &createQueryResp)
	defer s.cleanupQuery(createQueryResp.Query.ID)

	// list team queries
	var listQueriesResp listQueriesResponse
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQueriesResp, "team_id", fmt.Sprint(team1.ID))
	require.Len(t, listQueriesResp.Queries, 1)
	assert.Equal(t, "team1", listQueriesResp.Queries[0].Name)

	// list merged team queries
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQueriesResp, "team_id", fmt.Sprint(team1.ID), "merge_inherited", "true", "order_key", "team_id", "order_direction", "desc")
	require.Len(t, listQueriesResp.Queries, 2)
	assert.Equal(t, "team1", listQueriesResp.Queries[0].Name)
	assert.Equal(t, "global1", listQueriesResp.Queries[1].Name)
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
	assert.Equal(t, team.Secrets[0].Secret, "initialSecret")

	// Test replace existing secrets
	req := json.RawMessage(`{"secrets": [{"secret": "testSecret1"},{"secret": "testSecret2"}]}`)
	var resp teamEnrollSecretsResponse

	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), req, http.StatusOK, &resp)
	require.Len(t, resp.Secrets, 2)

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	assert.Equal(t, "testSecret1", team.Secrets[0].Secret)
	assert.Equal(t, "testSecret2", team.Secrets[1].Secret)

	// Test delete all enroll secrets
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{"secrets": []}`), http.StatusOK, &resp)
	require.Len(t, resp.Secrets, 0)

	// Test bad requests
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{"foo": [{"secret": "testSecret3"}]}`), http.StatusUnprocessableEntity, &resp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{}`), http.StatusUnprocessableEntity, &resp)

	// too many secrets
	secrets := createEnrollSecrets(t, fleet.MaxEnrollSecretsCount+1)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{"secrets": `+string(jsonMustMarshal(t, secrets))+`}`), http.StatusUnprocessableEntity, &resp)
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
		GlobalRole: ptr.String("observer"),
	}
	err = user.SetPassword(test.GoodPassword, 10, 10)
	require.Nil(t, err)
	user, err = s.ds.NewUser(context.Background(), user)
	require.Nil(t, err)

	// test available teams for user assigned to global role
	var getResp getUserResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", user.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, user.ID, getResp.User.ID)
	assert.Equal(t, ptr.String("observer"), getResp.User.GlobalRole)
	assert.Len(t, getResp.User.Teams, 0)     // teams is empty if user has a global role
	assert.Len(t, getResp.AvailableTeams, 1) // available teams includes all teams if user has a global role
	assert.Equal(t, getResp.AvailableTeams[0].Name, "Available Team")

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
	assert.Equal(t, getResp.User.Teams[0].Name, "Available Team")
	assert.Len(t, getResp.AvailableTeams, 1)
	assert.Equal(t, getResp.AvailableTeams[0].Name, "Available Team")

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
	assert.Equal(t, getResp.User.Teams[0].Name, "Available Team")
	assert.Len(t, getResp.AvailableTeams, 1)
	assert.Equal(t, getResp.AvailableTeams[0].Name, "Available Team")
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
	require.Len(t, tmResp.Team.Secrets, 1)
	assert.Equal(t, "DEF", tmResp.Team.Secrets[0].Secret)

	// create a duplicate team (same name)
	team2 := &fleet.Team{
		Name:        name,
		Description: "Team2 description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "GHI"}},
	}
	tmResp.Team = nil
	s.DoJSON("POST", "/api/latest/fleet/teams", team2, http.StatusConflict, &tmResp)

	// create a team with reserved team names; should be case-insensitive
	teamReserved := &fleet.Team{
		Name:        "no TeAm",
		Description: "description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "foobar"}},
	}

	r := s.Do("POST", "/api/latest/fleet/teams", teamReserved, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), `"No team" is a reserved team name`)

	teamReserved.Name = "AlL TeaMS"
	r = s.Do("POST", "/api/latest/fleet/teams", teamReserved, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), `"All teams" is a reserved team name`)

	// create a team with too many secrets
	team3 := &fleet.Team{
		Name:        name + "lots_of_secrets",
		Description: "Team3 description",
		Secrets:     createEnrollSecrets(t, fleet.MaxEnrollSecretsCount+1),
	}
	tmResp.Team = nil
	s.DoJSON("POST", "/api/latest/fleet/teams", team3, http.StatusUnprocessableEntity, &tmResp)

	// create a team with invalid host expiry window
	team4 := &fleet.TeamPayload{
		Name:        ptr.String(name + "invalid host_expiry_window"),
		Description: ptr.String("Team4 description"),
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
	require.Len(t, listResp.Teams, 0)

	// get team
	var getResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, team.Name, getResp.Team.Name)
	assert.NotNil(t, getResp.Team.Config.AgentOptions)

	// modify team
	team.Description = "Alt " + team.Description
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), team, http.StatusOK, &tmResp)
	assert.Contains(t, tmResp.Team.Description, "Alt ")

	// modify team's disk encryption, impossible without mdm enabled
	res := s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			EnableDiskEncryption: optjson.SetBool(true),
		},
	}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, `Couldn't update macos_settings because MDM features aren't turned on in Fleet.`)

	// modify a team with a NULL config
	defaultFeatures := fleet.Features{}
	defaultFeatures.ApplyDefaultsForNewInstalls()
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `UPDATE teams SET config = NULL WHERE id = ? `, tm1ID)
		return err
	})
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), team, http.StatusOK, &tmResp)
	assert.Equal(t, defaultFeatures, tmResp.Team.Config.Features)

	// modify a team with an empty config
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `UPDATE teams SET config = '{}' WHERE id = ? `, tm1ID)
		return err
	})
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), team, http.StatusOK, &tmResp)
	assert.Equal(t, defaultFeatures, tmResp.Team.Config.Features)
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
	assert.Equal(t, *modifyExpiry.HostExpirySettings, tmResp.Team.Config.HostExpirySettings)

	// invalid team host expiry (<= 0)
	modifyExpiry.HostExpirySettings.HostExpiryWindow = 0
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), modifyExpiry, http.StatusUnprocessableEntity, &tmResp)

	// try to rename to reserved names
	r = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), fleet.TeamPayload{Name: ptr.String("no TEAM")}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), `"No team" is a reserved team name`)

	r = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), fleet.TeamPayload{Name: ptr.String("ALL teAMs")}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), `"All teams" is a reserved team name`)

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
	assert.Len(t, usersResp.Users, 0)

	// list team users - non-existing team
	usersResp.Users = nil
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID+1), nil, http.StatusNotFound, &usersResp)

	// create a new user
	user := &fleet.User{
		Name:       "Team User",
		Email:      "user@example.com",
		GlobalRole: ptr.String("observer"),
	}
	require.NoError(t, user.SetPassword(test.GoodPassword, 10, 10))
	user, err := s.ds.NewUser(context.Background(), user)
	require.NoError(t, err)

	// add a team user
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: *user, Role: fleet.RoleObserver}}}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Users, 1)
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
	require.Len(t, usersResp.Users, 0)

	// delete team user
	tmResp.Team = nil
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: fleet.User{ID: user.ID}}}}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Users, 0)

	// delete team user - unknown user
	tmResp.Team = nil
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: fleet.User{ID: user.ID + 1}}}}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Users, 0)

	// delete team user - unknown team
	tmResp.Team = nil
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID+1), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: fleet.User{ID: user.ID}}}}, http.StatusNotFound, &tmResp)

	// modify team agent options with invalid options
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"x": "y"
	}`), http.StatusBadRequest, &tmResp)

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
	require.Contains(t, string(*tmResp.Team.Config.AgentOptions), `"x": "y"`)

	// modify team agent options with valid options
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": true
			}
		}
	}`), http.StatusOK, &tmResp)
	require.Contains(t, string(*tmResp.Team.Config.AgentOptions), `"aws_debug": true`)

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
	require.Contains(t, string(*tmResp.Team.Config.AgentOptions), `"aws_debug": true`) // left unchanged

	// list activities, it should have created one for edited_agent_options
	s.lastActivityMatches(fleet.ActivityTypeEditedAgentOptions{}.ActivityName(), fmt.Sprintf(`{"global": false, "team_id": %d, "team_name": %q}`, tm1ID, team.Name), 0)

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
	assert.Len(t, secResp.Secrets, 0)

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
		GlobalRole: ptr.String(fleet.RoleObserver),
	}
	global_obs_plus := &fleet.User{
		Name:       "Global Obs+",
		Email:      "global_obs_plus@example.com",
		GlobalRole: ptr.String(fleet.RoleObserverPlus),
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
	users := []*fleet.User{global_obs, global_obs_plus, team_obs, team_obs_plus}
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
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamWebhookSettings{
		FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
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
	require.Len(t, getResp.Team.Config.Integrations.Jira, 0)
	require.Len(t, getResp.Team.Config.Integrations.Zendesk, 0)

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
		WebhookSettings: &fleet.TeamWebhookSettings{
			FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
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
		Description: ptr.String("team-desc"),
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
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamWebhookSettings{
		FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
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
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamWebhookSettings{
		FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
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
		WebhookSettings: &fleet.TeamWebhookSettings{
			FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
				Enable:         false,
				DestinationURL: "http://example.com",
			},
		},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), tmResp.Team.Config.Integrations.Zendesk[0].GroupID)

	// enable the webhook without changing the integration should fail (an integration is already enabled)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamWebhookSettings{
		FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
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
		Description: ptr.String("team-desc-2"),
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
	require.Len(t, getResp.Team.Config.Integrations.Zendesk, 0)

	// remove all integrations on exit, so that other tests can enable the
	// webhook as needed
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{},
			Jira:    []*fleet.TeamJiraIntegration{},
		},
		WebhookSettings: &fleet.TeamWebhookSettings{},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Jira, 0)
	require.Len(t, tmResp.Team.Config.Integrations.Zendesk, 0)
	require.False(t, tmResp.Team.Config.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Empty(t, tmResp.Team.Config.WebhookSettings.FailingPoliciesWebhook.DestinationURL)

	appCfgResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"integrations": {
			"jira": [],
			"zendesk": []
		}
	}`), http.StatusOK, &appCfgResp)
	require.Len(t, appCfgResp.Integrations.Jira, 0)
	require.Len(t, appCfgResp.Integrations.Zendesk, 0)
}

func (s *integrationEnterpriseTestSuite) TestWindowsUpdatesTeamConfig() {
	t := s.T()

	// Create a team
	team := &fleet.Team{
		Name:        t.Name(),
		Description: "Team description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "XYZ"}},
	}
	var tmResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &tmResp)
	require.Equal(t, team.Name, tmResp.Team.Name)
	team.ID = tmResp.Team.ID

	checkWindowsOSUpdatesProfile(t, s.ds, &team.ID, nil)

	// modify the team's config
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": &fleet.WindowsUpdates{
				DeadlineDays:    optjson.SetInt(5),
				GracePeriodDays: optjson.SetInt(2),
			},
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, 5, tmResp.Team.Config.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 2, tmResp.Team.Config.MDM.WindowsUpdates.GracePeriodDays.Value)
	s.lastActivityMatches(fleet.ActivityTypeEditedWindowsUpdates{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "deadline_days": 5, "grace_period_days": 2}`, team.ID, team.Name), 0)

	checkWindowsOSUpdatesProfile(t, s.ds, &team.ID, &fleet.WindowsUpdates{
		DeadlineDays:    optjson.SetInt(5),
		GracePeriodDays: optjson.SetInt(2),
	})

	// get the team via the GET endpoint, check that it properly returns the mdm
	// settings.
	var getTmResp getTeamResponse
	s.DoJSON("GET", "/api/latest/fleet/teams/"+fmt.Sprint(team.ID), nil, http.StatusOK, &getTmResp)
	require.Equal(t, fleet.TeamMDM{
		MacOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.String{Set: true},
			Deadline:       optjson.String{Set: true},
		},
		IOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.String{Set: true},
			Deadline:       optjson.String{Set: true},
		},
		IPadOSUpdates: fleet.AppleOSUpdateSettings{
			MinimumVersion: optjson.String{Set: true},
			Deadline:       optjson.String{Set: true},
		},
		WindowsUpdates: fleet.WindowsUpdates{
			DeadlineDays:    optjson.SetInt(5),
			GracePeriodDays: optjson.SetInt(2),
		},
		MacOSSetup: fleet.MacOSSetup{
			MacOSSetupAssistant:         optjson.String{Set: true},
			BootstrapPackage:            optjson.String{Set: true},
			EnableReleaseDeviceManually: optjson.SetBool(false),
			Script:                      optjson.String{Set: true},
			Software:                    optjson.Slice[*fleet.MacOSSetupSoftware]{Set: true, Value: []*fleet.MacOSSetupSoftware{}},
		},
		WindowsSettings: fleet.WindowsSettings{
			CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
		},
	}, getTmResp.Team.Config.MDM)

	// only update the deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": &fleet.WindowsUpdates{
				DeadlineDays:    optjson.SetInt(6),
				GracePeriodDays: optjson.SetInt(2),
			},
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, 6, tmResp.Team.Config.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 2, tmResp.Team.Config.MDM.WindowsUpdates.GracePeriodDays.Value)
	lastActivity := s.lastActivityMatches(fleet.ActivityTypeEditedWindowsUpdates{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "deadline_days": 6, "grace_period_days": 2}`, team.ID, team.Name), 0)

	checkWindowsOSUpdatesProfile(t, s.ds, &team.ID, &fleet.WindowsUpdates{
		DeadlineDays:    optjson.SetInt(6),
		GracePeriodDays: optjson.SetInt(2),
	})

	// setting the macos updates doesn't alter the windows updates
	tmResp = teamResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": &fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("10.15.0"),
				Deadline:       optjson.SetString("2021-01-01"),
			},
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, "10.15.0", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2021-01-01", tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	require.Equal(t, 6, tmResp.Team.Config.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 2, tmResp.Team.Config.MDM.WindowsUpdates.GracePeriodDays.Value)
	// did not create a new activity for windows updates
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedWindowsUpdates{}.ActivityName(), "", lastActivity)
	lastActivity = s.lastActivityMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), ``, 0)

	checkWindowsOSUpdatesProfile(t, s.ds, &team.ID, &fleet.WindowsUpdates{
		DeadlineDays:    optjson.SetInt(6),
		GracePeriodDays: optjson.SetInt(2),
	})

	// sending a nil MDM or WindowsUpdates config doesn't modify anything
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": nil,
	}, http.StatusOK, &tmResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": nil,
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, "10.15.0", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2021-01-01", tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	require.Equal(t, 6, tmResp.Team.Config.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 2, tmResp.Team.Config.MDM.WindowsUpdates.GracePeriodDays.Value)
	// no new activity is created
	s.lastActivityMatches("", "", lastActivity)

	checkWindowsOSUpdatesProfile(t, s.ds, &team.ID, &fleet.WindowsUpdates{
		DeadlineDays:    optjson.SetInt(6),
		GracePeriodDays: optjson.SetInt(2),
	})

	// sending empty WindowsUpdates fields empties both fields
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": map[string]any{
				"deadline_days":     nil,
				"grace_period_days": nil,
			},
		},
	}, http.StatusOK, &tmResp)
	require.False(t, tmResp.Team.Config.MDM.WindowsUpdates.DeadlineDays.Valid)
	require.False(t, tmResp.Team.Config.MDM.WindowsUpdates.GracePeriodDays.Valid)
	s.lastActivityMatches(fleet.ActivityTypeEditedWindowsUpdates{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "deadline_days": null, "grace_period_days": null}`, team.ID, team.Name), 0)

	checkWindowsOSUpdatesProfile(t, s.ds, &team.ID, nil)

	// error checks:

	// try to set an invalid deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": map[string]any{
				"deadline_days":     1000,
				"grace_period_days": 1,
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set an invalid grace period
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": map[string]any{
				"deadline_days":     1,
				"grace_period_days": 1000,
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set a deadline but not a grace period
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": map[string]any{
				"deadline_days": 1,
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set a grace period but no deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": map[string]any{
				"grace_period_days": 1,
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set an empty grace period but a non-empty deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": map[string]any{
				"deadline_days":     1,
				"grace_period_days": nil,
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)
}

func (s *integrationEnterpriseTestSuite) assertAppleOSUpdatesDeclaration(teamID *uint, profileName string, expected *fleet.AppleOSUpdateSettings) {
	t := s.T()
	if teamID == nil {
		teamID = ptr.Uint(0)
	}

	var declUUID string
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		err := sqlx.GetContext(context.Background(), q, &declUUID,
			`SELECT declaration_uuid FROM mdm_apple_declarations WHERE team_id = ? AND name = ?`, teamID, profileName)
		if expected == nil {
			require.Error(t, err)
			return nil
		}
		return err
	})

	if expected == nil {
		// we already validated that the declaration did not exist
		return
	}
	decl, err := s.ds.GetMDMAppleDeclaration(context.Background(), declUUID)
	require.NoError(t, err)

	require.Contains(t, string(decl.RawJSON), fmt.Sprintf(`"TargetOSVersion": "%s"`, expected.MinimumVersion.Value))
	require.Contains(t, string(decl.RawJSON), fmt.Sprintf(`"TargetLocalDateTime": "%sT12:00:00"`, expected.Deadline.Value))
}

func (s *integrationEnterpriseTestSuite) TestAppleOSUpdatesTeamConfig() {
	t := s.T()

	team := &fleet.Team{
		Name:        t.Name(),
		Description: "Team description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "XYZ"}},
	}
	var tmResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &tmResp)
	require.Equal(t, team.Name, tmResp.Team.Name)
	team.ID = tmResp.Team.ID

	// no OS updates settings at the moment
	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetMacOSUpdatesProfileName, nil)
	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetIOSUpdatesProfileName, nil)
	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetIPadOSUpdatesProfileName, nil)

	// modify the team's config (macOS first)
	macOSUpdates := &fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("10.15.0"),
		Deadline:       optjson.SetString("2021-01-01"),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": macOSUpdates,
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, "10.15.0", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2021-01-01", tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	s.lastActivityMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "minimum_version": "10.15.0", "deadline": "2021-01-01"}`, team.ID, team.Name), 0)

	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetMacOSUpdatesProfileName, macOSUpdates)
	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetIOSUpdatesProfileName, nil)
	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetIPadOSUpdatesProfileName, nil)

	// modify the team's config (now iOS and iPadOS)
	iOSUpdates := &fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("11.11.11"),
		Deadline:       optjson.SetString("2022-02-02"),
	}
	iPadOSUpdates := &fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("12.12.12"),
		Deadline:       optjson.SetString("2023-03-03"),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"ios_updates":    iOSUpdates,
			"ipados_updates": iPadOSUpdates,
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, "10.15.0", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2021-01-01", tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	require.Equal(t, "11.11.11", tmResp.Team.Config.MDM.IOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2022-02-02", tmResp.Team.Config.MDM.IOSUpdates.Deadline.Value)
	require.Equal(t, "12.12.12", tmResp.Team.Config.MDM.IPadOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2023-03-03", tmResp.Team.Config.MDM.IPadOSUpdates.Deadline.Value)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "minimum_version": "10.15.0", "deadline": "2021-01-01"}`, team.ID, team.Name), 0)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedIOSMinVersion{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "minimum_version": "11.11.11", "deadline": "2022-02-02"}`, team.ID, team.Name), 0)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedIPadOSMinVersion{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "minimum_version": "12.12.12", "deadline": "2023-03-03"}`, team.ID, team.Name), 0)

	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetMacOSUpdatesProfileName, macOSUpdates)
	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetIOSUpdatesProfileName, iOSUpdates)
	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetIPadOSUpdatesProfileName, iPadOSUpdates)

	// only update the deadlines
	macOSUpdates = &fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("10.15.0"),
		Deadline:       optjson.SetString("2025-10-01"),
	}
	iOSUpdates = &fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("11.11.11"),
		Deadline:       optjson.SetString("2024-02-02"),
	}
	iPadOSUpdates = &fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("12.12.12"),
		Deadline:       optjson.SetString("2024-03-03"),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates":  macOSUpdates,
			"ios_updates":    iOSUpdates,
			"ipados_updates": iPadOSUpdates,
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, "10.15.0", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2025-10-01", tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	require.Equal(t, "11.11.11", tmResp.Team.Config.MDM.IOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2024-02-02", tmResp.Team.Config.MDM.IOSUpdates.Deadline.Value)
	require.Equal(t, "12.12.12", tmResp.Team.Config.MDM.IPadOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2024-03-03", tmResp.Team.Config.MDM.IPadOSUpdates.Deadline.Value)
	macOSLastActivity := s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "minimum_version": "10.15.0", "deadline": "2025-10-01"}`, team.ID, team.Name), 0)
	iOSLastActivity := s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedIOSMinVersion{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "minimum_version": "11.11.11", "deadline": "2024-02-02"}`, team.ID, team.Name), 0)
	iPadOSLastActivity := s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedIPadOSMinVersion{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "minimum_version": "12.12.12", "deadline": "2024-03-03"}`, team.ID, team.Name), 0)

	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetMacOSUpdatesProfileName, macOSUpdates)
	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetIOSUpdatesProfileName, iOSUpdates)
	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetIPadOSUpdatesProfileName, iPadOSUpdates)

	// setting the windows updates doesn't alter the apple updates
	tmResp = teamResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": &fleet.WindowsUpdates{
				DeadlineDays:    optjson.SetInt(10),
				GracePeriodDays: optjson.SetInt(2),
			},
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, "10.15.0", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2025-10-01", tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	require.Equal(t, "11.11.11", tmResp.Team.Config.MDM.IOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2024-02-02", tmResp.Team.Config.MDM.IOSUpdates.Deadline.Value)
	require.Equal(t, "12.12.12", tmResp.Team.Config.MDM.IPadOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2024-03-03", tmResp.Team.Config.MDM.IPadOSUpdates.Deadline.Value)
	require.Equal(t, 10, tmResp.Team.Config.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 2, tmResp.Team.Config.MDM.WindowsUpdates.GracePeriodDays.Value)
	// did not create a new activity for os updates
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), "", macOSLastActivity)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedIOSMinVersion{}.ActivityName(), "", iOSLastActivity)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedIPadOSMinVersion{}.ActivityName(), "", iPadOSLastActivity)
	lastActivity := s.lastActivityMatches(fleet.ActivityTypeEditedWindowsUpdates{}.ActivityName(), ``, 0)

	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetMacOSUpdatesProfileName, macOSUpdates)
	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetIOSUpdatesProfileName, iOSUpdates)
	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetIPadOSUpdatesProfileName, iPadOSUpdates)

	// sending a nil MDM or MacOSUpdate config doesn't modify anything
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": nil,
	}, http.StatusOK, &tmResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": nil,
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, "10.15.0", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2025-10-01", tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	require.Equal(t, "11.11.11", tmResp.Team.Config.MDM.IOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2024-02-02", tmResp.Team.Config.MDM.IOSUpdates.Deadline.Value)
	require.Equal(t, "12.12.12", tmResp.Team.Config.MDM.IPadOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2024-03-03", tmResp.Team.Config.MDM.IPadOSUpdates.Deadline.Value)
	// no new activity is created
	s.lastActivityMatches("", "", lastActivity)

	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetMacOSUpdatesProfileName, macOSUpdates)
	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetIOSUpdatesProfileName, iOSUpdates)
	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetIPadOSUpdatesProfileName, iPadOSUpdates)

	// sending macos settings but no macos_updates does not change the macos updates
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_settings": map[string]any{
				"custom_settings": nil,
			},
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, "10.15.0", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2025-10-01", tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	// no new activity is created
	s.lastActivityMatches("", "", lastActivity)

	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetMacOSUpdatesProfileName, macOSUpdates)
	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetIOSUpdatesProfileName, iOSUpdates)
	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetIPadOSUpdatesProfileName, iPadOSUpdates)

	// sending empty apple os updates fields empties both fields and removes the DDM profiles
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": map[string]any{
				"minimum_version": "",
				"deadline":        nil,
			},
			"ios_updates": map[string]any{
				"minimum_version": "",
				"deadline":        nil,
			},
			"ipados_updates": map[string]any{
				"minimum_version": "",
				"deadline":        nil,
			},
		},
	}, http.StatusOK, &tmResp)
	require.Empty(t, tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Empty(t, tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	require.Empty(t, tmResp.Team.Config.MDM.IOSUpdates.MinimumVersion.Value)
	require.Empty(t, tmResp.Team.Config.MDM.IOSUpdates.Deadline.Value)
	require.Empty(t, tmResp.Team.Config.MDM.IPadOSUpdates.MinimumVersion.Value)
	require.Empty(t, tmResp.Team.Config.MDM.IPadOSUpdates.Deadline.Value)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "minimum_version": "", "deadline": ""}`, team.ID, team.Name), 0)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedIOSMinVersion{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "minimum_version": "", "deadline": ""}`, team.ID, team.Name), 0)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedIPadOSMinVersion{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "minimum_version": "", "deadline": ""}`, team.ID, team.Name), 0)

	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetMacOSUpdatesProfileName, nil)
	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetIOSUpdatesProfileName, nil)
	s.assertAppleOSUpdatesDeclaration(&team.ID, mdm.FleetIPadOSUpdatesProfileName, nil)

	// error checks:

	// try to set an invalid deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": map[string]any{
				"minimum_version": "10.15.0",
				"deadline":        "2021-01-01T00:00:00Z",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"ios_updates": map[string]any{
				"minimum_version": "10.15.0",
				"deadline":        "2021-01-01T00:00:00Z",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"ipados_updates": map[string]any{
				"minimum_version": "10.15.0",
				"deadline":        "2021-01-01T00:00:00Z",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set an invalid minimum version
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": map[string]any{
				"minimum_version": "10.15.0 (19A583)",
				"deadline":        "2021-01-01T00:00:00Z",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"ios_updates": map[string]any{
				"minimum_version": "10.15.0 (19A583)",
				"deadline":        "2021-01-01T00:00:00Z",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"ipados_updates": map[string]any{
				"minimum_version": "10.15.0 (19A583)",
				"deadline":        "2021-01-01T00:00:00Z",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set a deadline but not a minimum version
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": map[string]any{
				"deadline": "2021-01-01T00:00:00Z",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"ios_updates": map[string]any{
				"deadline": "2021-01-01T00:00:00Z",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"ipados_updates": map[string]any{
				"deadline": "2021-01-01T00:00:00Z",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set an empty deadline but not a minimum version
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": map[string]any{
				"deadline": "",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"ios_updates": map[string]any{
				"deadline": "",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"ipados_updates": map[string]any{
				"deadline": "",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set a minimum version but not a deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": map[string]any{
				"minimum_version": "10.15.0 (19A583)",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"ios_updates": map[string]any{
				"minimum_version": "10.15.0 (19A583)",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"ipados_updates": map[string]any{
				"minimum_version": "10.15.0 (19A583)",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set an empty minimum version but not a deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": map[string]any{
				"minimum_version": "",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"ios_updates": map[string]any{
				"minimum_version": "",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"ipados_updates": map[string]any{
				"minimum_version": "",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)
}

func (s *integrationEnterpriseTestSuite) TestLinuxDiskEncryption() {
	t := s.T()

	// create a Linux host
	noTeamHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "3"),
		OsqueryHostID:   ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "3"),
		UUID:            t.Name() + "3",
		Hostname:        t.Name() + "foo3.local",
		PrimaryIP:       "192.168.1.3",
		PrimaryMac:      "30-65-EC-6F-C4-60",
		Platform:        "ubuntu",
		OSVersion:       "Ubuntu 22.04",
	})
	require.NoError(t, err)
	orbitKey := setOrbitEnrollment(t, noTeamHost, s.ds)
	noTeamHost.OrbitNodeKey = &orbitKey

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "A team"})
	require.NoError(t, err)
	teamID := ptr.Uint(team.ID)
	teamHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "2"),
		OsqueryHostID:   ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "2"),
		UUID:            t.Name() + "2",
		Hostname:        t.Name() + "foo2.local",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		Platform:        "rhel",
		OSVersion:       "Fedora 38.0", // this check is why HostLite now includes os_version in the data it's selecting
		TeamID:          teamID,
	})
	require.NoError(t, err)
	teamOrbitKey := setOrbitEnrollment(t, teamHost, s.ds)
	teamHost.OrbitNodeKey = &teamOrbitKey

	// NO TEAM //

	// config profiles endpoint should work but show all zeroes
	var profileSummary getMDMProfilesSummaryResponse
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles/summary", getMDMProfilesSummaryRequest{}, http.StatusOK, &profileSummary)
	require.Equal(t, fleet.MDMProfilesSummary{}, profileSummary.MDMProfilesSummary)

	// set encrypted for host
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), noTeamHost.ID, true))

	// should still show zeroes
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles/summary", getMDMProfilesSummaryRequest{}, http.StatusOK, &profileSummary)
	require.Equal(t, fleet.MDMProfilesSummary{}, profileSummary.MDMProfilesSummary)

	// should be nil before disk encryption is turned on
	// from host details
	getHostResp := getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", noTeamHost.ID), nil, http.StatusOK, &getHostResp)
	require.Nil(t, getHostResp.Host.MDM.OSSettings)

	// and my device
	deviceToken := "for_sure_secure"
	createDeviceTokenForHost(t, s.ds, noTeamHost.ID, deviceToken)

	getDeviceHostResp := getDeviceHostResponse{}
	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+deviceToken, nil, http.StatusOK)
	err = json.NewDecoder(res.Body).Decode(&getDeviceHostResp)
	require.NoError(t, err)
	require.Nil(t, getHostResp.Host.MDM.OSSettings)

	// turn on disk encryption enforcement
	s.Do("POST", "/api/latest/fleet/disk_encryption", updateDiskEncryptionRequest{EnableDiskEncryption: true}, http.StatusNoContent)

	// should be populated after disk encryption is turned on
	// from host details
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", noTeamHost.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings)

	// and my device
	getDeviceHostResp = getDeviceHostResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+deviceToken, nil, http.StatusOK)
	err = json.NewDecoder(res.Body).Decode(&getDeviceHostResp)
	require.NoError(t, err)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings)

	// should show the Linux host as pending
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles/summary", getMDMProfilesSummaryRequest{}, http.StatusOK, &profileSummary)
	require.Equal(t, fleet.MDMProfilesSummary{Pending: 1}, profileSummary.MDMProfilesSummary)

	// encryption summary should succeed (Linux encryption doesn't require MDM)
	var summary getMDMDiskEncryptionSummaryResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/disk_encryption/summary", getMDMDiskEncryptionSummaryRequest{}, http.StatusOK, &summary)
	s.DoJSON("GET", "/api/latest/fleet/disk_encryption", getMDMDiskEncryptionSummaryRequest{}, http.StatusOK, &summary)
	// disk is encrypted but key hasn't been escrowed yet
	require.Equal(t, fleet.MDMDiskEncryptionSummary{ActionRequired: fleet.MDMPlatformsCounts{Linux: 1}}, *summary.MDMDiskEncryptionSummary)

	// trigger escrow process from device
	// should fail because default Orbit version is too old
	res = s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/mdm/linux/trigger_escrow", deviceToken), nil, http.StatusBadRequest)
	res.Body.Close()

	// should succeed now that Orbit version isn't too old
	require.NoError(t, s.ds.SetOrUpdateHostOrbitInfo(context.Background(), noTeamHost.ID, fleet.MinOrbitLUKSVersion, sql.NullString{}, sql.NullBool{}))
	res = s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/mdm/linux/trigger_escrow", deviceToken), nil, http.StatusNoContent)
	res.Body.Close()

	// confirm that Orbit endpoint shows notification flag
	var orbitResponse orbitGetConfigResponse
	s.DoJSON("POST", "/api/fleet/orbit/config", orbitGetConfigRequest{OrbitNodeKey: orbitKey}, http.StatusOK, &orbitResponse)
	require.True(t, orbitResponse.Notifications.RunDiskEncryptionEscrow)

	// confirm that second Orbit pull doesn't show notification flag
	var secondOrbitResponse orbitGetConfigResponse
	s.DoJSON("POST", "/api/fleet/orbit/config", orbitGetConfigRequest{OrbitNodeKey: orbitKey}, http.StatusOK, &secondOrbitResponse)
	require.False(t, secondOrbitResponse.Notifications.RunDiskEncryptionEscrow)

	// set an error first; the successful write should overwrite that
	s.Do("POST", "/api/fleet/orbit/luks_data", orbitPostLUKSRequest{
		OrbitNodeKey: *noTeamHost.OrbitNodeKey,
		ClientError:  "Houston, we had a problem",
	}, http.StatusNoContent)

	// upload LUKS data
	keySlot := ptr.Uint(1)
	s.Do("POST", "/api/fleet/orbit/luks_data", orbitPostLUKSRequest{
		OrbitNodeKey: *noTeamHost.OrbitNodeKey,
		Passphrase:   "whale makes pail rise",
		Salt:         "the team i like lost",
		KeySlot:      keySlot,
	}, http.StatusNoContent)

	// confirm verified
	s.DoJSON("GET", "/api/latest/fleet/disk_encryption", getMDMDiskEncryptionSummaryRequest{}, http.StatusOK, &summary)
	require.Equal(t, fleet.MDMDiskEncryptionSummary{Verified: fleet.MDMPlatformsCounts{Linux: 1}}, *summary.MDMDiskEncryptionSummary)

	// get passphrase back
	var keyResponse getHostEncryptionKeyResponse
	s.DoJSON("GET", fmt.Sprintf(`/api/latest/fleet/mdm/hosts/%d/encryption_key`, noTeamHost.ID), getHostEncryptionKeyRequest{}, http.StatusOK, &keyResponse)
	s.DoJSON("GET", fmt.Sprintf(`/api/latest/fleet/hosts/%d/encryption_key`, noTeamHost.ID), getHostEncryptionKeyRequest{}, http.StatusOK, &keyResponse)
	require.Equal(t, "whale makes pail rise", keyResponse.EncryptionKey.DecryptedValue)

	// TEAM //
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles/summary", getMDMProfilesSummaryRequest{TeamID: teamID}, http.StatusOK, &profileSummary)
	require.Equal(t, fleet.MDMProfilesSummary{}, profileSummary.MDMProfilesSummary)

	// set encrypted for host
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), teamHost.ID, true))

	// should still show zeroes
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles/summary", getMDMProfilesSummaryRequest{TeamID: teamID}, http.StatusOK, &profileSummary)
	require.Equal(t, fleet.MDMProfilesSummary{}, profileSummary.MDMProfilesSummary)

	// turn on disk encryption enforcement for team
	s.Do("POST", "/api/latest/fleet/disk_encryption", updateDiskEncryptionRequest{TeamID: teamID, EnableDiskEncryption: true}, http.StatusNoContent)

	// should show the Linux host as pending
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles/summary", getMDMProfilesSummaryRequest{TeamID: teamID}, http.StatusOK, &profileSummary)
	require.Equal(t, fleet.MDMProfilesSummary{Pending: 1}, profileSummary.MDMProfilesSummary)

	// encryption summary should show host as action required
	s.DoJSON("GET", "/api/latest/fleet/disk_encryption", getMDMDiskEncryptionSummaryRequest{TeamID: teamID}, http.StatusOK, &summary)
	require.Equal(t, fleet.MDMDiskEncryptionSummary{ActionRequired: fleet.MDMPlatformsCounts{Linux: 1}}, *summary.MDMDiskEncryptionSummary)

	// upload LUKS data (no error, and no trigger, first this time)
	keySlot = ptr.Uint(3)
	s.Do("POST", "/api/fleet/orbit/luks_data", orbitPostLUKSRequest{
		OrbitNodeKey: *teamHost.OrbitNodeKey,
		Passphrase:   "the mome raths outgrabe",
		Salt:         "jabberwocky, but salty",
		KeySlot:      keySlot,
	}, http.StatusNoContent)

	// confirm verified
	s.DoJSON("GET", "/api/latest/fleet/disk_encryption", getMDMDiskEncryptionSummaryRequest{TeamID: teamID}, http.StatusOK, &summary)
	require.Equal(t, fleet.MDMDiskEncryptionSummary{Verified: fleet.MDMPlatformsCounts{Linux: 1}}, *summary.MDMDiskEncryptionSummary)

	// get passphrase back
	s.DoJSON("GET", fmt.Sprintf(`/api/latest/fleet/hosts/%d/encryption_key`, teamHost.ID), getHostEncryptionKeyRequest{}, http.StatusOK, &keyResponse)
	require.Equal(t, "the mome raths outgrabe", keyResponse.EncryptionKey.DecryptedValue)
}

func (s *integrationEnterpriseTestSuite) TestListDevicePolicies() {
	t := s.T()
	ctx := context.Background()

	// set the logo via the modify appconfig endpoint, so that the cache is
	// properly updated.
	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/latest/fleet/config",
		json.RawMessage(`{
		"org_info":{
			"org_logo_url": "http://example.com/logo",
			"contact_url": "http://example.com/contact"
		}
	}`), http.StatusOK, &acResp)
	require.Equal(t, "http://example.com/logo", acResp.OrgInfo.OrgLogoURL)
	require.Equal(t, "http://example.com/contact", acResp.OrgInfo.ContactURL)

	team, err := s.ds.NewTeam(ctx, &fleet.Team{
		ID:          51,
		Name:        "team1-policies",
		Description: "desc team1",
	})
	require.NoError(t, err)

	token := "much_valid"
	host := createHostAndDeviceToken(t, s.ds, token)
	err = s.ds.AddHostsToTeam(ctx, &team.ID, []uint{host.ID})
	require.NoError(t, err)

	qr, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name:           "TestQueryEnterpriseGlobalPolicy",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
		Logging:        fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	// add a global policy
	gpParams := globalPolicyRequest{
		QueryID:    &qr.ID,
		Resolution: "some global resolution",
	}
	gpResp := globalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)

	// add a policy execution
	require.NoError(t, s.ds.RecordPolicyQueryExecutions(ctx, host,
		map[uint]*bool{gpResp.Policy.ID: ptr.Bool(false)}, time.Now(), false))

	// add a policy to team
	oldToken := s.token
	t.Cleanup(func() {
		s.token = oldToken
	})

	password := test.GoodPassword
	email := "test_enterprise_policies@user.com"

	u := &fleet.User{
		Name:       "test team user",
		Email:      email,
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team,
				Role: fleet.RoleMaintainer,
			},
		},
	}

	require.NoError(t, u.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(ctx, u)
	require.NoError(t, err)

	s.token = s.getTestToken(email, password)
	tpParams := teamPolicyRequest{
		Name:        "TestQueryEnterpriseTeamPolicy",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some team resolution",
		Platform:    "darwin",
	}
	tpResp := teamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team.ID), tpParams, http.StatusOK, &tpResp)

	// try with invalid token
	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/invalid_token/policies", nil, http.StatusUnauthorized)
	err = res.Body.Close()
	require.NoError(t, err)

	// GET `/api/_version_/fleet/device/{token}/policies`
	listDevicePoliciesResp := listDevicePoliciesResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/policies", nil, http.StatusOK)
	err = json.NewDecoder(res.Body).Decode(&listDevicePoliciesResp)
	require.NoError(t, err)
	err = res.Body.Close()
	require.NoError(t, err)
	require.Len(t, listDevicePoliciesResp.Policies, 2)
	require.NoError(t, listDevicePoliciesResp.Err)

	// GET `/api/_version_/fleet/device/{token}`
	getDeviceHostResp := getDeviceHostResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token, nil, http.StatusOK)
	err = json.NewDecoder(res.Body).Decode(&getDeviceHostResp)
	require.NoError(t, err)
	err = res.Body.Close()
	require.NoError(t, err)
	require.NoError(t, getDeviceHostResp.Err)
	require.Equal(t, host.ID, getDeviceHostResp.Host.ID)
	require.False(t, getDeviceHostResp.Host.RefetchRequested)
	require.Equal(t, "http://example.com/logo", getDeviceHostResp.OrgLogoURL)
	require.Equal(t, "http://example.com/contact", getDeviceHostResp.OrgContactURL)
	require.Len(t, *getDeviceHostResp.Host.Policies, 2)
	require.False(t, getDeviceHostResp.GlobalConfig.Features.EnableSoftwareInventory)

	// GET `/api/_version_/fleet/device/{token}/desktop`
	getDesktopResp := fleetDesktopResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
	err = json.NewDecoder(res.Body).Decode(&getDesktopResp)
	require.NoError(t, err)
	err = res.Body.Close()
	require.NoError(t, err)
	require.NoError(t, getDesktopResp.Err)
	require.Equal(t, *getDesktopResp.FailingPolicies, uint(1))
	require.False(t, getDesktopResp.Notifications.NeedsMDMMigration)

	// update the team to enable software inventory
	team.Config.Features.EnableSoftwareInventory = true
	_, err = s.ds.SaveTeam(ctx, team)
	require.NoError(t, err)

	getDeviceHostResp = getDeviceHostResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token, nil, http.StatusOK)
	err = json.NewDecoder(res.Body).Decode(&getDeviceHostResp)
	require.NoError(t, err)
	require.True(t, getDeviceHostResp.GlobalConfig.Features.EnableSoftwareInventory)
}

// TestCustomTransparencyURL tests that Fleet Premium licensees can use custom transparency urls.
func (s *integrationEnterpriseTestSuite) TestCustomTransparencyURL() {
	t := s.T()

	token := "token_test_custom_transparency_url"
	createHostAndDeviceToken(t, s.ds, token)

	// confirm intitial default url
	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.Equal(t, fleet.DefaultTransparencyURL, acResp.FleetDesktop.TransparencyURL)

	// confirm device endpoint returns initial default url
	deviceResp := &transparencyURLResponse{}
	rawResp := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/transparency", nil, http.StatusTemporaryRedirect)
	json.NewDecoder(rawResp.Body).Decode(deviceResp) //nolint:errcheck
	rawResp.Body.Close()                             //nolint:errcheck
	require.NoError(t, deviceResp.Err)
	require.Equal(t, fleet.DefaultTransparencyURL, rawResp.Header.Get("Location"))

	// set custom url
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"fleet_desktop":{"transparency_url": "customURL"}}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.Equal(t, "customURL", acResp.FleetDesktop.TransparencyURL)

	// device endpoint returns custom url
	deviceResp = &transparencyURLResponse{}
	rawResp = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/transparency", nil, http.StatusTemporaryRedirect)
	json.NewDecoder(rawResp.Body).Decode(deviceResp) //nolint:errcheck
	rawResp.Body.Close()                             //nolint:errcheck
	require.NoError(t, deviceResp.Err)
	require.Equal(t, "customURL", rawResp.Header.Get("Location"))

	// empty string applies default url
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"fleet_desktop":{"transparency_url": ""}}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.Equal(t, fleet.DefaultTransparencyURL, acResp.FleetDesktop.TransparencyURL)

	// device endpoint returns default url
	deviceResp = &transparencyURLResponse{}
	rawResp = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/transparency", nil, http.StatusTemporaryRedirect)
	json.NewDecoder(rawResp.Body).Decode(deviceResp) //nolint:errcheck
	rawResp.Body.Close()                             //nolint:errcheck
	require.NoError(t, deviceResp.Err)
	require.Equal(t, fleet.DefaultTransparencyURL, rawResp.Header.Get("Location"))
}

func (s *integrationEnterpriseTestSuite) TestMDMWindowsUpdates() {
	t := s.T()

	// keep the last activity, to detect newly created ones
	var activitiesResp listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp, "order_key", "a.id", "order_direction", "desc")
	var lastActivity uint
	if len(activitiesResp.Activities) > 0 {
		lastActivity = activitiesResp.Activities[0].ID
	}

	checkInvalidConfig := func(config string) {
		// try to set an invalid config
		acResp := appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(config), http.StatusUnprocessableEntity, &acResp)

		// get the appconfig, nothing changed
		acResp = appConfigResponse{}
		s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
		require.Equal(t, fleet.WindowsUpdates{DeadlineDays: optjson.Int{Set: true}, GracePeriodDays: optjson.Int{Set: true}}, acResp.MDM.WindowsUpdates)

		// no activity got created
		activitiesResp = listActivitiesResponse{}
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp, "order_key", "a.id", "order_direction", "desc")
		require.Condition(t, func() bool {
			return (lastActivity == 0 && len(activitiesResp.Activities) == 0) ||
				(len(activitiesResp.Activities) > 0 && activitiesResp.Activities[0].ID == lastActivity)
		})
	}

	// missing grace period
	checkInvalidConfig(`{"mdm": {
		"windows_updates": {
			"deadline_days": 1
		}
	}}`)

	// missing deadline
	checkInvalidConfig(`{"mdm": {
		"windows_updates": {
			"grace_period_days": 1
		}
	}}`)

	// invalid deadline
	checkInvalidConfig(`{"mdm": {
		"windows_updates": {
			"grace_period_days": 1,
			"deadline_days": -1
		}
	}}`)

	// invalid grace period
	checkInvalidConfig(`{"mdm": {
		"windows_updates": {
			"grace_period_days": -1,
			"deadline_days": 1
		}
	}}`)

	checkWindowsOSUpdatesProfile(t, s.ds, nil, nil)

	// valid config
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"windows_updates": {
					"deadline_days": 5,
					"grace_period_days": 1
				}
			}
		}`), http.StatusOK, &acResp)
	require.Equal(t, 5, acResp.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 1, acResp.MDM.WindowsUpdates.GracePeriodDays.Value)

	checkWindowsOSUpdatesProfile(t, s.ds, nil, &fleet.WindowsUpdates{
		DeadlineDays:    optjson.SetInt(5),
		GracePeriodDays: optjson.SetInt(1),
	})

	// edited windows updates activity got created
	s.lastActivityMatches(fleet.ActivityTypeEditedWindowsUpdates{}.ActivityName(), `{"deadline_days":5, "grace_period_days":1, "team_id": null, "team_name": null}`, 0)

	// get the appconfig
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Equal(t, 5, acResp.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 1, acResp.MDM.WindowsUpdates.GracePeriodDays.Value)

	// update the deadline
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"windows_updates": {
					"deadline_days": 6,
					"grace_period_days": 1
				}
			}
		}`), http.StatusOK, &acResp)
	require.Equal(t, 6, acResp.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 1, acResp.MDM.WindowsUpdates.GracePeriodDays.Value)

	checkWindowsOSUpdatesProfile(t, s.ds, nil, &fleet.WindowsUpdates{
		DeadlineDays:    optjson.SetInt(6),
		GracePeriodDays: optjson.SetInt(1),
	})

	// another edited windows updates activity got created
	lastActivity = s.lastActivityMatches(fleet.ActivityTypeEditedWindowsUpdates{}.ActivityName(), `{"deadline_days":6, "grace_period_days":1, "team_id": null, "team_name": null}`, 0)

	// update something unrelated - the transparency url
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"fleet_desktop":{"transparency_url": "customURL"}}`), http.StatusOK, &acResp)
	require.Equal(t, 6, acResp.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 1, acResp.MDM.WindowsUpdates.GracePeriodDays.Value)

	// no activity got created
	s.lastActivityMatches("", ``, lastActivity)

	checkWindowsOSUpdatesProfile(t, s.ds, nil, &fleet.WindowsUpdates{
		DeadlineDays:    optjson.SetInt(6),
		GracePeriodDays: optjson.SetInt(1),
	})

	// clear the Windows requirement
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"windows_updates": {
					"deadline_days": null,
					"grace_period_days": null
				}
			}
		}`), http.StatusOK, &acResp)
	require.False(t, acResp.MDM.WindowsUpdates.DeadlineDays.Valid)
	require.False(t, acResp.MDM.WindowsUpdates.GracePeriodDays.Valid)

	// edited windows updates activity got created with empty requirement
	lastActivity = s.lastActivityMatches(fleet.ActivityTypeEditedWindowsUpdates{}.ActivityName(), `{"deadline_days":null, "grace_period_days":null, "team_id": null, "team_name": null}`, 0)

	checkWindowsOSUpdatesProfile(t, s.ds, nil, nil)

	// update again with empty windows requirement
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"windows_updates": {
					"deadline_days": null,
					"grace_period_days": null
				}
			}
		}`), http.StatusOK, &acResp)
	require.False(t, acResp.MDM.WindowsUpdates.DeadlineDays.Valid)
	require.False(t, acResp.MDM.WindowsUpdates.GracePeriodDays.Valid)

	// no activity got created
	s.lastActivityMatches("", ``, lastActivity)
}

func (s *integrationEnterpriseTestSuite) TestMDMAppleOSUpdates() {
	t := s.T()

	// keep the last activity, to detect newly created ones
	var activitiesResp listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp, "order_key", "a.id", "order_direction", "desc")
	var lastActivity uint
	if len(activitiesResp.Activities) > 0 {
		lastActivity = activitiesResp.Activities[0].ID
	}

	checkInvalidConfig := func(config string) {
		// try to set an invalid config
		acResp := appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(config), http.StatusUnprocessableEntity, &acResp)

		// get the appconfig, nothing changed
		acResp = appConfigResponse{}
		s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
		require.Equal(t, fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}}, acResp.MDM.MacOSUpdates)

		// no activity got created
		activitiesResp = listActivitiesResponse{}
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp, "order_key", "a.id", "order_direction", "desc")
		require.Condition(t, func() bool {
			return (lastActivity == 0 && len(activitiesResp.Activities) == 0) ||
				(len(activitiesResp.Activities) > 0 && activitiesResp.Activities[0].ID == lastActivity)
		})
	}

	// missing minimum_version
	checkInvalidConfig(`{"mdm": {
		"macos_updates": {
			"deadline": "2022-01-01"
		}
	}}`)
	checkInvalidConfig(`{"mdm": {
		"ios_updates": {
			"deadline": "2022-01-01"
		}
	}}`)
	checkInvalidConfig(`{"mdm": {
		"ipados_updates": {
			"deadline": "2022-01-01"
		}
	}}`)

	// missing deadline
	checkInvalidConfig(`{"mdm": {
		"macos_updates": {
			"minimum_version": "12.1.1"
		}
	}}`)
	checkInvalidConfig(`{"mdm": {
		"ios_updates": {
			"minimum_version": "12.1.1"
		}
	}}`)
	checkInvalidConfig(`{"mdm": {
		"ipados_updates": {
			"minimum_version": "12.1.1"
		}
	}}`)

	// invalid deadline
	checkInvalidConfig(`{"mdm": {
		"macos_updates": {
			"minimum_version": "12.1.1",
			"deadline": "2022"
		}
	}}`)
	checkInvalidConfig(`{"mdm": {
		"ios_updates": {
			"minimum_version": "12.1.1",
			"deadline": "2022"
		}
	}}`)
	checkInvalidConfig(`{"mdm": {
		"ipados_updates": {
			"minimum_version": "12.1.1",
			"deadline": "2022"
		}
	}}`)

	// deadline includes timestamp
	checkInvalidConfig(`{"mdm": {
		"macos_updates": {
			"minimum_version": "12.1.1",
			"deadline": "2022-01-01T00:00:00Z"
		}
	}}`)
	checkInvalidConfig(`{"mdm": {
		"ios_updates": {
			"minimum_version": "12.1.1",
			"deadline": "2022-01-01T00:00:00Z"
		}
	}}`)
	checkInvalidConfig(`{"mdm": {
		"ipados_updates": {
			"minimum_version": "12.1.1",
			"deadline": "2022-01-01T00:00:00Z"
		}
	}}`)

	// minimum_version includes build info
	checkInvalidConfig(`{"mdm": {
		"macos_updates": {
			"minimum_version": "12.1.1 (ABCD)",
			"deadline": "2022-01-01"
		}
	}}`)
	checkInvalidConfig(`{"mdm": {
		"ios_updates": {
			"minimum_version": "12.1.1 (ABCD)",
			"deadline": "2022-01-01"
		}
	}}`)
	checkInvalidConfig(`{"mdm": {
		"ipados_updates": {
			"minimum_version": "12.1.1 (ABCD)",
			"deadline": "2022-01-01"
		}
	}}`)

	// valid config
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"macos_updates": {
					"minimum_version": "12.3.1",
					"deadline": "2022-01-01"
				},
				"ios_updates": {
					"minimum_version": "13.13.13",
					"deadline": "2023-03-03"
				},
				"ipados_updates": {
					"minimum_version": "14.14.14",
					"deadline": "2024-04-04"
				}
			}
		}`), http.StatusOK, &acResp)
	require.Equal(t, "12.3.1", acResp.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2022-01-01", acResp.MDM.MacOSUpdates.Deadline.Value)
	require.Equal(t, "13.13.13", acResp.MDM.IOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2023-03-03", acResp.MDM.IOSUpdates.Deadline.Value)
	require.Equal(t, "14.14.14", acResp.MDM.IPadOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2024-04-04", acResp.MDM.IPadOSUpdates.Deadline.Value)

	// edited macos min version activity got created
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), `{"deadline":"2022-01-01", "minimum_version":"12.3.1", "team_id": null, "team_name": null}`, 0)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedIOSMinVersion{}.ActivityName(), `{"deadline":"2023-03-03", "minimum_version":"13.13.13", "team_id": null, "team_name": null}`, 0)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedIPadOSMinVersion{}.ActivityName(), `{"deadline":"2024-04-04", "minimum_version":"14.14.14", "team_id": null, "team_name": null}`, 0)
	s.assertAppleOSUpdatesDeclaration(nil, mdm.FleetMacOSUpdatesProfileName, &fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("12.3.1"), Deadline: optjson.SetString("2022-01-01"),
	})
	s.assertAppleOSUpdatesDeclaration(nil, mdm.FleetIOSUpdatesProfileName, &fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("13.13.13"), Deadline: optjson.SetString("2023-03-03"),
	})
	s.assertAppleOSUpdatesDeclaration(nil, mdm.FleetIPadOSUpdatesProfileName, &fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("14.14.14"), Deadline: optjson.SetString("2024-04-04"),
	})

	// get the appconfig
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Equal(t, "12.3.1", acResp.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2022-01-01", acResp.MDM.MacOSUpdates.Deadline.Value)
	require.Equal(t, "13.13.13", acResp.MDM.IOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2023-03-03", acResp.MDM.IOSUpdates.Deadline.Value)
	require.Equal(t, "14.14.14", acResp.MDM.IPadOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2024-04-04", acResp.MDM.IPadOSUpdates.Deadline.Value)

	// update the deadline
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"macos_updates": {
					"minimum_version": "12.3.1",
					"deadline": "2024-01-01"
				},
				"ios_updates": {
					"minimum_version": "13.13.13",
					"deadline": "2025-05-05"
				},
				"ipados_updates": {
					"minimum_version": "14.14.14",
					"deadline": "2026-06-06"
				}
			}
		}`), http.StatusOK, &acResp)
	require.Equal(t, "12.3.1", acResp.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2024-01-01", acResp.MDM.MacOSUpdates.Deadline.Value)
	require.Equal(t, "13.13.13", acResp.MDM.IOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2025-05-05", acResp.MDM.IOSUpdates.Deadline.Value)
	require.Equal(t, "14.14.14", acResp.MDM.IPadOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2026-06-06", acResp.MDM.IPadOSUpdates.Deadline.Value)

	// another edited macos min version activity got created
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), `{"deadline":"2024-01-01", "minimum_version":"12.3.1", "team_id": null, "team_name": null}`, 0)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedIOSMinVersion{}.ActivityName(), `{"deadline":"2025-05-05", "minimum_version":"13.13.13", "team_id": null, "team_name": null}`, 0)
	lastActivity = s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedIPadOSMinVersion{}.ActivityName(), `{"deadline":"2026-06-06", "minimum_version":"14.14.14", "team_id": null, "team_name": null}`, 0)
	s.assertAppleOSUpdatesDeclaration(nil, mdm.FleetMacOSUpdatesProfileName, &fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("12.3.1"), Deadline: optjson.SetString("2024-01-01"),
	})
	s.assertAppleOSUpdatesDeclaration(nil, mdm.FleetIOSUpdatesProfileName, &fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("13.13.13"), Deadline: optjson.SetString("2025-05-05"),
	})
	s.assertAppleOSUpdatesDeclaration(nil, mdm.FleetIPadOSUpdatesProfileName, &fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("14.14.14"), Deadline: optjson.SetString("2026-06-06"),
	})

	// update something unrelated - the transparency url
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"fleet_desktop":{"transparency_url": "customURL"}}`), http.StatusOK, &acResp)
	require.Equal(t, "12.3.1", acResp.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2024-01-01", acResp.MDM.MacOSUpdates.Deadline.Value)

	// no activity got created
	s.lastActivityMatches("", ``, lastActivity)
	s.assertAppleOSUpdatesDeclaration(nil, mdm.FleetMacOSUpdatesProfileName, &fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("12.3.1"), Deadline: optjson.SetString("2024-01-01"),
	})
	s.assertAppleOSUpdatesDeclaration(nil, mdm.FleetIOSUpdatesProfileName, &fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("13.13.13"), Deadline: optjson.SetString("2025-05-05"),
	})
	s.assertAppleOSUpdatesDeclaration(nil, mdm.FleetIPadOSUpdatesProfileName, &fleet.AppleOSUpdateSettings{
		MinimumVersion: optjson.SetString("14.14.14"), Deadline: optjson.SetString("2026-06-06"),
	})

	// clear the apple OS requirements
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"macos_updates": {
					"minimum_version": "",
					"deadline": ""
				},
				"ios_updates": {
					"minimum_version": "",
					"deadline": ""
				},
				"ipados_updates": {
					"minimum_version": "",
					"deadline": ""
				}
			}
		}`), http.StatusOK, &acResp)
	require.Empty(t, acResp.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Empty(t, acResp.MDM.MacOSUpdates.Deadline.Value)
	require.Empty(t, acResp.MDM.IOSUpdates.MinimumVersion.Value)
	require.Empty(t, acResp.MDM.IOSUpdates.Deadline.Value)
	require.Empty(t, acResp.MDM.IPadOSUpdates.MinimumVersion.Value)
	require.Empty(t, acResp.MDM.IPadOSUpdates.Deadline.Value)

	// edited macos min version activity got created with empty requirement
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), `{"deadline":"", "minimum_version":"", "team_id": null, "team_name": null}`, 0)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedIOSMinVersion{}.ActivityName(), `{"deadline":"", "minimum_version":"", "team_id": null, "team_name": null}`, 0)
	lastActivity = s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedIPadOSMinVersion{}.ActivityName(), `{"deadline":"", "minimum_version":"", "team_id": null, "team_name": null}`, 0)

	// check DDM profiles were removed
	s.assertAppleOSUpdatesDeclaration(nil, mdm.FleetMacOSUpdatesProfileName, nil)
	s.assertAppleOSUpdatesDeclaration(nil, mdm.FleetIOSUpdatesProfileName, nil)
	s.assertAppleOSUpdatesDeclaration(nil, mdm.FleetIPadOSUpdatesProfileName, nil)

	// update again with empty apple OS requirements
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"macos_updates": {
					"minimum_version": "",
					"deadline": ""
				},
				"ios_updates": {
					"minimum_version": "",
					"deadline": ""
				},
				"ipados_updates": {
					"minimum_version": "",
					"deadline": ""
				}
			}
		}`), http.StatusOK, &acResp)
	require.Empty(t, acResp.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Empty(t, acResp.MDM.MacOSUpdates.Deadline.Value)
	require.Empty(t, acResp.MDM.IOSUpdates.MinimumVersion.Value)
	require.Empty(t, acResp.MDM.IOSUpdates.Deadline.Value)
	require.Empty(t, acResp.MDM.IPadOSUpdates.MinimumVersion.Value)
	require.Empty(t, acResp.MDM.IPadOSUpdates.Deadline.Value)

	// no activity or DDM profiles were created
	s.lastActivityMatches("", ``, lastActivity)
	s.assertAppleOSUpdatesDeclaration(nil, mdm.FleetMacOSUpdatesProfileName, nil)
	s.assertAppleOSUpdatesDeclaration(nil, mdm.FleetIOSUpdatesProfileName, nil)
	s.assertAppleOSUpdatesDeclaration(nil, mdm.FleetIPadOSUpdatesProfileName, nil)
}

// Skipping admin-created users because we don't have email fully set up in integration tests
func (s *integrationEnterpriseTestSuite) TestInvitedUserMFA() {
	t := s.T()

	// create valid invite
	createInviteReq := createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      ptr.String("some email"),
		Name:       ptr.String("some name"),
		GlobalRole: null.StringFrom(fleet.RoleAdmin),
		MFAEnabled: ptr.Bool(true),
		SSOEnabled: ptr.Bool(true),
	}}
	createInviteResp := createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusConflict, &createInviteResp)
	createInviteReq.SSOEnabled = nil
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusOK, &createInviteResp)
	require.NotNil(t, createInviteResp.Invite)
	require.NotZero(t, createInviteResp.Invite.ID)
	validInvite := *createInviteResp.Invite

	// create user from valid invite - the token was not returned via the
	// response's json, must get it from the db
	inv, err := s.ds.Invite(context.Background(), validInvite.ID)
	require.NoError(t, err)
	validInviteToken := inv.Token

	// verify the token with valid invite
	var verifyInvResp verifyInviteResponse
	s.DoJSON("GET", "/api/latest/fleet/invites/"+validInviteToken, nil, http.StatusOK, &verifyInvResp)
	require.Equal(t, validInvite.ID, verifyInvResp.Invite.ID)

	var createFromInviteResp createUserResponse
	s.DoJSON("POST", "/api/latest/fleet/users", fleet.UserPayload{
		Name:        ptr.String("Full Name"),
		Password:    ptr.String(test.GoodPassword),
		Email:       ptr.String("a@b.c"),
		InviteToken: ptr.String(validInviteToken),
	}, http.StatusOK, &createFromInviteResp)
	require.True(t, createFromInviteResp.User.MFAEnabled)

	// create an invite with SSO, swap to MFA
	createInviteReq = createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      ptr.String("a@b.d"),
		Name:       ptr.String("some other name"),
		GlobalRole: null.StringFrom(fleet.RoleAdmin),
		SSOEnabled: ptr.Bool(true),
	}}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusOK, &createInviteResp)
	validInvite = *createInviteResp.Invite
	var updateInviteResp updateInviteResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID), updateInviteRequest{
		InvitePayload: fleet.InvitePayload{MFAEnabled: ptr.Bool(true), SSOEnabled: ptr.Bool(false)},
	}, http.StatusOK, &updateInviteResp)
	require.True(t, updateInviteResp.Invite.MFAEnabled)
}

func (s *integrationEnterpriseTestSuite) TestSSOJITProvisioning() {
	t := s.T()

	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.False(t, acResp.SSOSettings.EnableJITProvisioning)

	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
			"idp_name": "SimpleSAML",
			"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php",
			"enable_jit_provisioning": false
		}
	}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.False(t, acResp.SSOSettings.EnableJITProvisioning)

	// users can't be created if SSO is disabled
	auth, body := s.LoginSSOUser("sso_user", "user123#")
	require.Contains(t, body, "/login?status=account_invalid")
	// ensure theresn't a user in the DB
	_, err := s.ds.UserByEmail(context.Background(), auth.UserID())
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	// If enable_jit_provisioning is enabled Roles won't be updated for existing SSO users.

	// enable JIT provisioning
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
			"idp_name": "SimpleSAML",
			"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php",
			"enable_jit_provisioning": true
		}
	}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.True(t, acResp.SSOSettings.EnableJITProvisioning)

	// a new user is created and redirected accordingly
	auth, body = s.LoginSSOUser("sso_user", "user123#")
	// a successful redirect has this content
	require.Contains(t, body, "Redirecting to Fleet at  ...")
	user, err := s.ds.UserByEmail(context.Background(), auth.UserID())
	require.NoError(t, err)
	require.Equal(t, auth.UserID(), user.Email)

	// a new activity item is created
	activitiesResp := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp)
	require.NoError(t, activitiesResp.Err)
	require.NotEmpty(t, activitiesResp.Activities)
	require.Condition(t, func() bool {
		for _, a := range activitiesResp.Activities {
			if (a.Type == fleet.ActivityTypeUserAddedBySSO{}.ActivityName()) && *a.ActorEmail == auth.UserID() {
				return true
			}
		}
		return false
	})

	// Test that roles are not updated for an existing user when SSO attributes are not set.

	// Change role to global admin first.
	user.GlobalRole = ptr.String("admin")
	err = s.ds.SaveUser(context.Background(), user)
	require.NoError(t, err)
	// Login should NOT change the role to the default (global observer) because SSO attributes
	// are not set for this user (see ../../tools/saml/users.php).
	auth, body = s.LoginSSOUser("sso_user", "user123#")
	assert.Equal(t, "sso_user@example.com", auth.UserID())
	assert.Equal(t, "SSO User 1", auth.UserDisplayName())
	require.Contains(t, body, "Redirecting to Fleet at  ...")
	user, err = s.ds.UserByEmail(context.Background(), "sso_user@example.com")
	require.NoError(t, err)
	require.NotNil(t, user.GlobalRole)
	require.Equal(t, *user.GlobalRole, "admin")

	// A user with pre-configured roles can be created
	// see `tools/saml/users.php` for details.
	auth, body = s.LoginSSOUser("sso_user_3_global_admin", "user123#")
	assert.Equal(t, "sso_user_3_global_admin@example.com", auth.UserID())
	assert.Equal(t, "SSO User 3", auth.UserDisplayName())
	assert.Contains(t, auth.AssertionAttributes(), fleet.SAMLAttribute{
		Name: "FLEET_JIT_USER_ROLE_GLOBAL",
		Values: []fleet.SAMLAttributeValue{{
			Value: "admin",
		}},
	})
	require.Contains(t, body, "Redirecting to Fleet at  ...")

	// Test that roles are updated for an existing user when SSO attributes are set.

	// Change role to global maintainer first.
	user3, err := s.ds.UserByEmail(context.Background(), auth.UserID())
	require.NoError(t, err)
	require.Equal(t, auth.UserID(), user3.Email)
	user3.GlobalRole = ptr.String("maintainer")
	err = s.ds.SaveUser(context.Background(), user3)
	require.NoError(t, err)

	// Login should change the role to the configured role in the SSO attributes (global admin).
	auth, body = s.LoginSSOUser("sso_user_3_global_admin", "user123#")
	assert.Equal(t, "sso_user_3_global_admin@example.com", auth.UserID())
	assert.Equal(t, "SSO User 3", auth.UserDisplayName())
	require.Contains(t, body, "Redirecting to Fleet at  ...")
	user3, err = s.ds.UserByEmail(context.Background(), "sso_user_3_global_admin@example.com")
	require.NoError(t, err)
	require.NotNil(t, user3.GlobalRole)
	require.Equal(t, *user3.GlobalRole, "admin")

	// We cannot use NewTeam and must use adhoc SQL because the teams.id is
	// auto-incremented and other tests cause it to be different than what we need (ID=1).
	var execErr error
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, execErr = db.ExecContext(context.Background(), `INSERT INTO teams (id, name) VALUES (1, 'Foobar') ON DUPLICATE KEY UPDATE name = VALUES(name);`)
		return execErr
	})
	require.NoError(t, execErr)

	// Create a team for the test below.
	_, err = s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "team_" + t.Name(),
		Description: "desc team_" + t.Name(),
	})
	require.NoError(t, err)

	// A user with pre-configured roles can be created,
	// see `tools/saml/users.php` for details.
	auth, body = s.LoginSSOUser("sso_user_4_team_maintainer", "user123#")
	assert.Equal(t, "sso_user_4_team_maintainer@example.com", auth.UserID())
	assert.Equal(t, "SSO User 4", auth.UserDisplayName())
	assert.Contains(t, auth.AssertionAttributes(), fleet.SAMLAttribute{
		Name: "FLEET_JIT_USER_ROLE_TEAM_1",
		Values: []fleet.SAMLAttributeValue{{
			Value: "maintainer",
		}},
	})
	require.Contains(t, body, "Redirecting to Fleet at  ...")

	// A user with pre-configured roles can be created,
	// see `tools/saml/users.php` for details.
	auth, body = s.LoginSSOUser("sso_user_5_team_admin", "user123#")
	assert.Equal(t, "sso_user_5_team_admin@example.com", auth.UserID())
	assert.Equal(t, "SSO User 5", auth.UserDisplayName())
	assert.Contains(t, auth.AssertionAttributes(), fleet.SAMLAttribute{
		Name: "FLEET_JIT_USER_ROLE_TEAM_1",
		Values: []fleet.SAMLAttributeValue{{
			Value: "admin",
		}},
	})
	// FLEET_JIT_USER_ROLE_* attributes with value `null` are ignored by Fleet.
	assert.Contains(t, auth.AssertionAttributes(), fleet.SAMLAttribute{
		Name: "FLEET_JIT_USER_ROLE_GLOBAL",
		Values: []fleet.SAMLAttributeValue{{
			Value: "null",
		}},
	})
	// FLEET_JIT_USER_ROLE_* attributes with value `null` are ignored by Fleet.
	assert.Contains(t, auth.AssertionAttributes(), fleet.SAMLAttribute{
		Name: "FLEET_JIT_USER_ROLE_TEAM_2",
		Values: []fleet.SAMLAttributeValue{{
			Value: "null",
		}},
	})
	require.Contains(t, body, "Redirecting to Fleet at  ...")

	// A user with pre-configured roles can be created,
	// see `tools/saml/users.php` for details.
	auth, body = s.LoginSSOUser("sso_user_6_global_observer", "user123#")
	assert.Equal(t, "sso_user_6_global_observer@example.com", auth.UserID())
	assert.Equal(t, "SSO User 6", auth.UserDisplayName())
	// FLEET_JIT_USER_ROLE_* attributes with value `null` are ignored by Fleet.
	assert.Contains(t, auth.AssertionAttributes(), fleet.SAMLAttribute{
		Name: "FLEET_JIT_USER_ROLE_GLOBAL",
		Values: []fleet.SAMLAttributeValue{{
			Value: "null",
		}},
	})
	// FLEET_JIT_USER_ROLE_* attributes with value `null` are ignored by Fleet.
	assert.Contains(t, auth.AssertionAttributes(), fleet.SAMLAttribute{
		Name: "FLEET_JIT_USER_ROLE_TEAM_1",
		Values: []fleet.SAMLAttributeValue{{
			Value: "null",
		}},
	})
	require.Contains(t, body, "Redirecting to Fleet at  ...")
}

func (s *integrationEnterpriseTestSuite) TestDistributedReadWithFeatures() {
	t := s.T()

	// Global config has both features enabled
	spec := []byte(`
  features:
    additional_queries: null
    enable_host_users: true
    enable_software_inventory: true
`)
	s.applyConfig(spec)

	// Team config has only additional queries enabled
	a := json.RawMessage(`{"time": "SELECT * FROM time"}`)
	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          8324,
		Name:        "team1_" + t.Name(),
		Description: "desc team1_" + t.Name(),
		Config: fleet.TeamConfig{
			Features: fleet.Features{
				EnableHostUsers:         false,
				EnableSoftwareInventory: false,
				AdditionalQueries:       &a,
			},
		},
	})
	require.NoError(t, err)

	// Create a host without a team
	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	s.lq.On("QueriesForHost", host.ID).Return(map[string]string{fmt.Sprintf("%d", host.ID): "select 1 from osquery;"}, nil)

	// ensure we can read distributed queries for the host
	err = s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)

	// get distributed queries for the host
	req := getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	var dqResp getDistributedQueriesResponse
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.Contains(t, dqResp.Queries, "fleet_detail_query_users")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_macos")
	require.NotContains(t, dqResp.Queries, "fleet_additional_query_time")

	// add the host to team1
	err = s.ds.AddHostsToTeam(context.Background(), &team.ID, []uint{host.ID})
	require.NoError(t, err)

	err = s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)
	req = getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	dqResp = getDistributedQueriesResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_users")
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_software_macos")
	require.Contains(t, dqResp.Queries, "fleet_additional_query_time")
}

func (s *integrationEnterpriseTestSuite) TestListHosts() {
	t := s.T()

	// create a couple of hosts
	host1, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	host2, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + "2"),
		NodeKey:         ptr.String(t.Name() + "2"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sbar.local", t.Name()),
		Platform:        "linux",
	})
	require.NoError(t, err)
	host3, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + "3"),
		NodeKey:         ptr.String(t.Name() + "3"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sbaz.local", t.Name()),
		Platform:        "windows",
	})
	require.NoError(t, err)
	require.NotNil(t, host3)

	// set disk space information for some hosts (none provided for host3)
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), host1.ID, 10.0, 2.0, 500.0))
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), host2.ID, 32.0, 4.0, 1000.0))

	var resp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp)
	require.Len(t, resp.Hosts, 3)

	allHostsLabel, err := s.ds.GetLabelSpec(context.Background(), "All hosts")
	require.NoError(t, err)
	for _, h := range resp.Hosts {
		err = s.ds.RecordLabelQueryExecutions(
			context.Background(), h.Host, map[uint]*bool{allHostsLabel.ID: ptr.Bool(true)}, time.Now(), false,
		)
		require.NoError(t, err)
	}

	resp = listHostsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", allHostsLabel.ID), nil, http.StatusOK, &resp, "low_disk_space", "32")
	require.Len(t, resp.Hosts, 1)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "low_disk_space", "32")
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, host1.ID, resp.Hosts[0].ID)
	assert.Equal(t, 10.0, resp.Hosts[0].GigsDiskSpaceAvailable)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "low_disk_space", "100")
	require.Len(t, resp.Hosts, 2)

	// returns an error when the low_disk_space value is invalid (outside 1-100)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusBadRequest, &resp, "low_disk_space", "101")
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusBadRequest, &resp, "low_disk_space", "0")

	// counting hosts works with and without the filter too
	var countResp countHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp)
	require.Equal(t, 3, countResp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "low_disk_space", "32")
	require.Equal(t, 1, countResp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "low_disk_space", "100")
	require.Equal(t, 2, countResp.Count)

	// host summary returns counts for low disk space
	var summaryResp getHostSummaryResponse
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &summaryResp, "low_disk_space", "32")
	require.Equal(t, uint(3), summaryResp.TotalsHostsCount)
	require.NotNil(t, summaryResp.LowDiskSpaceCount)
	require.Equal(t, uint(1), *summaryResp.LowDiskSpaceCount)

	summaryResp = getHostSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &summaryResp, "platform", "windows", "low_disk_space", "32")
	require.Equal(t, uint(1), summaryResp.TotalsHostsCount)
	require.NotNil(t, summaryResp.LowDiskSpaceCount)
	require.Equal(t, uint(0), *summaryResp.LowDiskSpaceCount)

	// all possible filters
	summaryResp = getHostSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &summaryResp, "team_id", "1", "platform", "linux", "low_disk_space", "32")
	require.Equal(t, uint(0), summaryResp.TotalsHostsCount)
	require.NotNil(t, summaryResp.LowDiskSpaceCount)
	require.Equal(t, uint(0), *summaryResp.LowDiskSpaceCount)

	// without low_disk_space, does not return the count
	summaryResp = getHostSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &summaryResp, "team_id", "1", "platform", "linux")
	require.Equal(t, uint(0), summaryResp.TotalsHostsCount)
	require.Nil(t, summaryResp.LowDiskSpaceCount)

	// Add a failing policy
	ctx := context.Background()
	qr, err := s.ds.NewQuery(
		ctx, &fleet.Query{
			Name:           "TestQueryEnterpriseTestListHosts",
			Description:    "Some description",
			Query:          "select * from osquery;",
			ObserverCanRun: true,
			Logging:        fleet.LoggingSnapshot,
		},
	)
	require.NoError(t, err)

	// add a global policy
	gpParams := globalPolicyRequest{
		QueryID:    &qr.ID,
		Resolution: "some global resolution",
	}
	gpResp := globalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)

	// add a failing policy execution
	require.NoError(
		t, s.ds.RecordPolicyQueryExecutions(
			ctx, host1,
			map[uint]*bool{gpResp.Policy.ID: ptr.Bool(false)}, time.Now(), false,
		),
	)

	// populate software for hosts
	now := time.Now()
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host1.ID, software)
	require.NoError(t, err)

	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host1, false))

	inserted, err := s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: host1.Software[0].ID,
		CVE:        "cve-123-123-123",
	}, fleet.NVDSource)
	require.NoError(t, err)
	require.True(t, inserted)

	vulnMeta := []fleet.CVEMeta{{
		CVE:              "cve-123-123-123",
		CVSSScore:        ptr.Float64(9.8),
		EPSSProbability:  ptr.Float64(0.5),
		CISAKnownExploit: ptr.Bool(true),
		Published:        &now,
		Description:      "a long description of the cve",
	}}

	require.NoError(t, s.ds.InsertCVEMeta(context.Background(), vulnMeta))
	ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	require.NoError(t, s.ds.UpdateHostIssuesVulnerabilities(ctx))

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "populate_software", "true")
	require.Len(t, resp.Hosts, 3)
	for _, h := range resp.Hosts {
		if h.ID == host1.ID {
			require.NotEmpty(t, h.Software)
			require.Len(t, h.Software, 1)
			require.NotEmpty(t, h.Software[0].Vulnerabilities)

			s := &vulnMeta[0].Description
			require.Equal(t, &vulnMeta[0].CVSSScore, h.Software[0].Vulnerabilities[0].CVSSScore)
			require.Equal(t, &vulnMeta[0].EPSSProbability, h.Software[0].Vulnerabilities[0].EPSSProbability)
			require.Equal(t, &vulnMeta[0].CISAKnownExploit, h.Software[0].Vulnerabilities[0].CISAKnownExploit)
			require.Equal(t, &s, h.Software[0].Vulnerabilities[0].Description)
			assert.Equal(t, uint64(1), h.HostIssues.FailingPoliciesCount)
			assert.Equal(t, uint64(1), *h.HostIssues.CriticalVulnerabilitiesCount)
			assert.Equal(t, uint64(2), h.HostIssues.TotalIssuesCount)
		} else {
			assert.Zero(t, h.HostIssues.FailingPoliciesCount)
			assert.Zero(t, *h.HostIssues.CriticalVulnerabilitiesCount)
			assert.Zero(t, h.HostIssues.TotalIssuesCount)
		}
	}

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "populate_software", "without_vulnerability_details")
	require.Len(t, resp.Hosts, 3)
	for _, h := range resp.Hosts {
		if h.ID == host1.ID {
			require.NotEmpty(t, h.Software)
			require.Len(t, h.Software, 1)
			require.NotEmpty(t, h.Software[0].Vulnerabilities)

			require.Nil(t, h.Software[0].Vulnerabilities[0].CVSSScore)
			require.Nil(t, h.Software[0].Vulnerabilities[0].EPSSProbability)
			require.Nil(t, h.Software[0].Vulnerabilities[0].CISAKnownExploit)
			require.Nil(t, h.Software[0].Vulnerabilities[0].Description)
			assert.Equal(t, uint64(1), h.HostIssues.FailingPoliciesCount)
			assert.Equal(t, uint64(1), *h.HostIssues.CriticalVulnerabilitiesCount)
			assert.Equal(t, uint64(2), h.HostIssues.TotalIssuesCount)
		} else {
			assert.Zero(t, h.HostIssues.FailingPoliciesCount)
			assert.Zero(t, *h.HostIssues.CriticalVulnerabilitiesCount)
			assert.Zero(t, h.HostIssues.TotalIssuesCount)
		}
	}

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "populate_software", "false")
	require.Len(t, resp.Hosts, 3)
	for _, h := range resp.Hosts {
		require.Empty(t, h.Software)
	}

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", allHostsLabel.ID), nil, http.StatusOK, &resp)
	assert.Len(t, resp.Hosts, 3)
	for _, h := range resp.Hosts {
		if h.ID == host1.ID {
			assert.Equal(t, uint64(1), h.HostIssues.FailingPoliciesCount)
			assert.Equal(t, uint64(1), *h.HostIssues.CriticalVulnerabilitiesCount)
			assert.Equal(t, uint64(2), h.HostIssues.TotalIssuesCount)
		} else {
			assert.Zero(t, h.HostIssues.FailingPoliciesCount)
			assert.Zero(t, *h.HostIssues.CriticalVulnerabilitiesCount)
			assert.Zero(t, h.HostIssues.TotalIssuesCount)
		}
	}

	// Test ordering by issues
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "order_key", "issues",
	) // defaults to ascending order (lowest issues to most issues)
	require.Len(t, resp.Hosts, 3)
	assert.Equal(t, host1.ID, resp.Hosts[2].ID)
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", allHostsLabel.ID), nil, http.StatusOK, &resp, "order_key", "issues",
		"order_direction", "desc",
	)
	require.Len(t, resp.Hosts, 3)
	assert.Equal(t, host1.ID, resp.Hosts[0].ID)
}

func (s *integrationEnterpriseTestSuite) TestHostHealth() {
	t := s.T()

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		OsqueryHostID:   ptr.String(t.Name() + "hostid1"),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(t.Name() + "nodekey1"),
		UUID:            t.Name() + "uuid1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OSVersion:       "Mac OS X 10.14.6",
		Platform:        "darwin",
		CPUType:         "cpuType",
		TeamID:          ptr.Uint(team.ID),
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	passingTeamPolicy, err := s.ds.NewTeamPolicy(context.Background(), team.ID, nil, fleet.PolicyPayload{
		Name:       "Passing Global Policy",
		Query:      "select 1",
		Resolution: "Run this command to fix it",
	})
	require.NoError(t, err)

	failingTeamPolicy, err := s.ds.NewTeamPolicy(context.Background(), team.ID, nil, fleet.PolicyPayload{
		Name:       "Failing Global Policy",
		Query:      "select 1",
		Resolution: "Run this command to fix it",
		Critical:   true,
	})
	require.NoError(t, err)

	passingGlobalPolicy, err := s.ds.NewGlobalPolicy(context.Background(), nil, fleet.PolicyPayload{
		Name:       "Passing Global Policy",
		Query:      "select 1",
		Resolution: "Run this command to fix it",
	})
	require.NoError(t, err)

	failingGlobalPolicy, err := s.ds.NewGlobalPolicy(context.Background(), nil, fleet.PolicyPayload{
		Name:       "Failing Global Policy",
		Query:      "select 1",
		Resolution: "Run this command to fix it",
		Critical:   false,
	})
	require.NoError(t, err)

	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{failingGlobalPolicy.ID: ptr.Bool(false)}, time.Now(), false))
	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{passingGlobalPolicy.ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{failingTeamPolicy.ID: ptr.Bool(false)}, time.Now(), false))
	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{passingTeamPolicy.ID: ptr.Bool(true)}, time.Now(), false))

	hh := getHostHealthResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/health", host.ID), nil, http.StatusOK, &hh)
	require.Equal(t, host.ID, hh.HostID)
	assert.NotNil(t, hh.HostHealth)
	assert.Equal(t, host.OSVersion, hh.HostHealth.OsVersion)
	assert.Equal(t, 2, hh.HostHealth.FailingPoliciesCount)
	assert.Equal(t, ptr.Int(1), hh.HostHealth.FailingCriticalPoliciesCount)
	assert.Contains(t, hh.HostHealth.FailingPolicies, &fleet.HostHealthFailingPolicy{
		ID:         failingTeamPolicy.ID,
		Name:       failingTeamPolicy.Name,
		Resolution: failingTeamPolicy.Resolution,
		Critical:   ptr.Bool(true),
	})
	assert.Contains(t, hh.HostHealth.FailingPolicies, &fleet.HostHealthFailingPolicy{
		ID:         failingGlobalPolicy.ID,
		Name:       failingGlobalPolicy.Name,
		Resolution: failingGlobalPolicy.Resolution,
		Critical:   ptr.Bool(false),
	})
}

func (s *integrationEnterpriseTestSuite) TestListVulnerabilities() {
	t := s.T()
	var resp listVulnerabilitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)

	// Invalid Order Key
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusBadRequest, &resp, "order_key", "foo", "order_direction", "asc")

	// EE Only Order Key
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "order_key", "cvss_score", "order_direction", "asc")

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)
	require.Len(s.T(), resp.Vulnerabilities, 0)

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "2"),
		OsqueryHostID:   ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "2"),
		UUID:            t.Name() + "2",
		Hostname:        t.Name() + "foo2.local",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		Platform:        "windows",
	})
	require.NoError(t, err)

	err = s.ds.UpdateHostOperatingSystem(context.Background(), host.ID, fleet.OperatingSystem{
		Name:     "Windows 11 Enterprise 22H2",
		Version:  "10.0.19042.1234",
		Platform: "windows",
	})
	require.NoError(t, err)
	allos, err := s.ds.ListOperatingSystems(context.Background())
	require.NoError(t, err)
	var os fleet.OperatingSystem
	for _, o := range allos {
		if o.ID > os.ID {
			os = o
		}
	}

	err = s.ds.UpdateOSVersions(context.Background())
	require.NoError(t, err)

	_, err = s.ds.InsertOSVulnerability(context.Background(), fleet.OSVulnerability{
		OSID:              os.ID,
		CVE:               "CVE-2021-1234",
		ResolvedInVersion: ptr.String("10.0.19043.2013"),
	}, fleet.MSRCSource)
	require.NoError(t, err)

	res, err := s.ds.UpdateHostSoftware(context.Background(), host.ID, []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	})
	require.NoError(t, err)
	sw := res.Inserted[0]

	_, err = s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: sw.ID,
		CVE:        "CVE-2021-1235",
	}, fleet.NVDSource)
	require.NoError(t, err)

	// insert CVEMeta
	mockTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	err = s.ds.InsertCVEMeta(context.Background(), []fleet.CVEMeta{
		{
			CVE:              "CVE-2021-1234",
			CVSSScore:        ptr.Float64(7.5),
			EPSSProbability:  ptr.Float64(0.5),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2021-1234",
		},
		{
			CVE:              "CVE-2021-1235",
			CVSSScore:        ptr.Float64(5.4),
			EPSSProbability:  ptr.Float64(0.6),
			CISAKnownExploit: ptr.Bool(false),
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2021-1235",
		},
	})
	require.NoError(t, err)

	err = s.ds.UpdateVulnerabilityHostCounts(context.Background())
	require.NoError(t, err)

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)
	require.Len(s.T(), resp.Vulnerabilities, 2)
	require.Equal(t, resp.Count, uint(2))
	require.False(t, resp.Meta.HasPreviousResults)
	require.False(t, resp.Meta.HasNextResults)
	require.Empty(t, resp.Err)

	expected := map[string]struct {
		fleet.CVE
		HostCount   uint
		DetailsLink string
		Source      fleet.VulnerabilitySource
	}{
		"CVE-2021-1234": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-1234",
			CVE: fleet.CVE{
				CVE:              "CVE-2021-1234",
				CVSSScore:        ptr.Float64Ptr(7.5),
				EPSSProbability:  ptr.Float64Ptr(0.5),
				CISAKnownExploit: ptr.BoolPtr(true),
				CVEPublished:     ptr.TimePtr(mockTime),
				Description:      ptr.StringPtr("Test CVE 2021-1234"),
			},
		},
		"CVE-2021-1235": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-1235",
			CVE: fleet.CVE{
				CVE:              "CVE-2021-1235",
				CVSSScore:        ptr.Float64Ptr(5.4),
				EPSSProbability:  ptr.Float64Ptr(0.6),
				CISAKnownExploit: ptr.BoolPtr(false),
				CVEPublished:     ptr.TimePtr(mockTime),
				Description:      ptr.StringPtr("Test CVE 2021-1235"),
			},
		},
	}

	for _, vuln := range resp.Vulnerabilities {
		expectedVuln, ok := expected[vuln.CVE.CVE]
		require.True(t, ok)
		require.Equal(t, expectedVuln.HostCount, vuln.HostsCount)
		require.Equal(t, expectedVuln.DetailsLink, vuln.DetailsLink)
		require.Equal(t, expectedVuln.CVE.CVE, vuln.CVE.CVE)
	}

	// EE Exploit Filter
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "exploit", "true")
	require.Len(t, resp.Vulnerabilities, 1)
	require.Equal(t, "CVE-2021-1234", resp.Vulnerabilities[0].CVE.CVE)

	// Test Team Filter
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "team_id", "1")
	require.Len(s.T(), resp.Vulnerabilities, 0)

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(context.Background(), &team.ID, []uint{host.ID})
	require.NoError(t, err)

	err = s.ds.UpdateVulnerabilityHostCounts(context.Background())
	require.NoError(t, err)

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "team_id", fmt.Sprintf("%d", team.ID))
	require.Len(t, resp.Vulnerabilities, 2)
	require.Equal(t, uint(2), resp.Count)
	require.False(t, resp.Meta.HasPreviousResults)
	require.False(t, resp.Meta.HasNextResults)
	require.Empty(t, resp.Err)

	for _, vuln := range resp.Vulnerabilities {
		expectedVuln, ok := expected[vuln.CVE.CVE]
		require.True(t, ok)
		require.Equal(t, expectedVuln.HostCount, vuln.HostsCount)
		require.Equal(t, expectedVuln.DetailsLink, vuln.DetailsLink)
		require.Equal(t, expectedVuln.CVE.CVE, vuln.CVE.CVE)
	}

	var gResp getVulnerabilityResponse
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-1234", nil, http.StatusOK, &gResp)
	require.Empty(t, gResp.Err)
	require.Equal(t, "CVE-2021-1234", gResp.Vulnerability.CVE.CVE)
	require.Equal(t, uint(1), gResp.Vulnerability.HostsCount)
	require.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2021-1234", gResp.Vulnerability.DetailsLink)
	require.Equal(t, ptr.StringPtr("Test CVE 2021-1234"), gResp.Vulnerability.Description)
	require.Equal(t, ptr.Float64Ptr(7.5), gResp.Vulnerability.CVSSScore)
	require.Equal(t, ptr.BoolPtr(true), gResp.Vulnerability.CISAKnownExploit)
	require.Equal(t, ptr.Float64Ptr(0.5), gResp.Vulnerability.EPSSProbability)
	require.Equal(t, ptr.TimePtr(mockTime), gResp.Vulnerability.CVEPublished)
	require.Len(t, gResp.OSVersions, 1)
	require.Equal(t, "Windows 11 Enterprise 22H2 10.0.19042.1234", gResp.OSVersions[0].Name)
	require.Equal(t, "Windows 11 Enterprise 22H2", gResp.OSVersions[0].NameOnly)
	require.Equal(t, "windows", gResp.OSVersions[0].Platform)
	require.Equal(t, "10.0.19042.1234", gResp.OSVersions[0].Version)
	require.Equal(t, 1, gResp.OSVersions[0].HostsCount)
	require.Equal(t, "10.0.19043.2013", *gResp.OSVersions[0].ResolvedInVersion)
}

func (s *integrationEnterpriseTestSuite) TestOSVersions() {
	t := s.T()

	testOS := fleet.OperatingSystem{Name: "Windows 11 Pro", Version: "10.0.22621.2861", Arch: "x86_64", KernelVersion: "10.0.22621.2861", Platform: "windows"}

	hosts := s.createHosts(t)

	var resp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp)
	require.Len(t, resp.Hosts, len(hosts))

	// set operating system information on a host
	require.NoError(t, s.ds.UpdateHostOperatingSystem(context.Background(), hosts[0].ID, testOS))
	var osinfo struct {
		ID          uint `db:"id"`
		OSVersionID uint `db:"os_version_id"`
	}
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &osinfo,
			`SELECT id, os_version_id FROM operating_systems WHERE name = ? AND version = ? AND arch = ? AND kernel_version = ? AND platform = ?`,
			testOS.Name, testOS.Version, testOS.Arch, testOS.KernelVersion, testOS.Platform)
	})
	require.Greater(t, osinfo.ID, uint(0))

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_name", testOS.Name, "os_version", testOS.Version)
	require.Len(t, resp.Hosts, 1)

	expected := resp.Hosts[0]
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_id", fmt.Sprintf("%d", osinfo.ID))
	require.Len(t, resp.Hosts, 1)
	require.Equal(t, expected, resp.Hosts[0])

	// generate aggregated stats
	require.NoError(t, s.ds.UpdateOSVersions(context.Background()))

	// insert OS Vulns
	_, err := s.ds.InsertOSVulnerability(context.Background(), fleet.OSVulnerability{
		OSID: osinfo.ID,
		CVE:  "CVE-2021-1234",
	}, fleet.MSRCSource)
	require.NoError(t, err)

	// insert CVE MEta
	vulnMeta := []fleet.CVEMeta{
		{
			CVE:              "CVE-2021-1234",
			CVSSScore:        ptr.Float64(5.4),
			EPSSProbability:  ptr.Float64(0.5),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
			Description:      "a long description of the cve",
		},
	}
	require.NoError(t, s.ds.InsertCVEMeta(context.Background(), vulnMeta))

	var osVersionsResp osVersionsResponse
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp)
	require.Len(t, osVersionsResp.OSVersions, 1)
	require.Equal(t, 1, osVersionsResp.OSVersions[0].HostsCount)
	require.Equal(t, fmt.Sprintf("%s %s", testOS.Name, testOS.Version), osVersionsResp.OSVersions[0].Name)
	require.Equal(t, testOS.Name, osVersionsResp.OSVersions[0].NameOnly)
	require.Equal(t, testOS.Version, osVersionsResp.OSVersions[0].Version)
	require.Equal(t, testOS.Platform, osVersionsResp.OSVersions[0].Platform)
	require.Len(t, osVersionsResp.OSVersions[0].Vulnerabilities, 1)
	require.Equal(t, "CVE-2021-1234", osVersionsResp.OSVersions[0].Vulnerabilities[0].CVE)
	require.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2021-1234", osVersionsResp.OSVersions[0].Vulnerabilities[0].DetailsLink)
	require.Equal(t, *vulnMeta[0].CVSSScore, **osVersionsResp.OSVersions[0].Vulnerabilities[0].CVSSScore)
	require.Equal(t, *vulnMeta[0].EPSSProbability, **osVersionsResp.OSVersions[0].Vulnerabilities[0].EPSSProbability)
	require.Equal(t, *vulnMeta[0].CISAKnownExploit, **osVersionsResp.OSVersions[0].Vulnerabilities[0].CISAKnownExploit)
	require.Equal(t, *vulnMeta[0].Published, **osVersionsResp.OSVersions[0].Vulnerabilities[0].CVEPublished)
	require.Equal(t, vulnMeta[0].Description, **osVersionsResp.OSVersions[0].Vulnerabilities[0].Description)
	expectedOSVersion := osVersionsResp.OSVersions[0]

	var osVersionResp getOSVersionResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusOK, &osVersionResp)
	require.Equal(t, &expectedOSVersion, osVersionResp.OSVersion)

	// OS versions with invalid team
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusNotFound, &osVersionResp, "team_id",
		"99999",
	)

	// Create team and ask for the OS versions from the team (with no hosts) -- should get 404.
	tr := teamResponse{}
	s.DoJSON(
		"POST", "/api/latest/fleet/teams", createTeamRequest{
			TeamPayload: fleet.TeamPayload{
				Name: ptr.String("os_versions_team"),
			},
		}, http.StatusOK, &tr,
	)
	osVersionResp = getOSVersionResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusOK, &osVersionResp, "team_id",
		fmt.Sprintf("%d", tr.Team.ID),
	)
	assert.Zero(t, osVersionResp.OSVersion.HostsCount)

	// return empty json if UpdateOSVersions cron hasn't run yet for new team
	team0, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "new team"})
	require.NoError(t, err)
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), &team0.ID, []uint{hosts[0].ID}))
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp, "team_id", fmt.Sprintf("%d", team0.ID))
	require.Len(t, osVersionsResp.OSVersions, 0)

	// return err if team_id is invalid
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusBadRequest, &osVersionsResp, "team_id", "invalid")

	// Create another team and a team user
	team1, err := s.ds.NewTeam(
		context.Background(), &fleet.Team{
			ID:          42,
			Name:        "team1-os_version",
			Description: "desc team1",
		},
	)
	require.NoError(t, err)
	// Create a new admin for team1.
	password := test.GoodPassword
	email := "admin-team1-os_version@example.com"
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

	s.setTokenForTest(t, email, test.GoodPassword)

	// generate aggregated stats
	require.NoError(t, s.ds.UpdateOSVersions(context.Background()))
	// team1 user does not have access to team0 host
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp)
	assert.Empty(t, osVersionsResp.OSVersions)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusOK, &osVersionResp)
	assert.Zero(t, osVersionResp.OSVersion.HostsCount)

	// Move host from team0 to team1
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{hosts[0].ID}))
	require.NoError(t, s.ds.UpdateOSVersions(context.Background()))
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp)
	require.Len(t, osVersionsResp.OSVersions, 1)
	assert.Equal(t, expectedOSVersion, osVersionsResp.OSVersions[0])
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusOK, &osVersionResp)
	require.Equal(t, &expectedOSVersion, osVersionResp.OSVersion)

	// Team user is forbidden to access invalid team
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusForbidden, &osVersionResp, "team_id",
		"99999",
	)

	// team user doesn't have acess to "no team"
	osVersionsResp = osVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusForbidden, &osVersionsResp, "team_id", "0")
	require.Len(t, osVersionsResp.OSVersions, 0)

	// team_id=0 is supported and returns results for hosts in "no team"
	s.token = getTestAdminToken(t, s.server)
	// no hosts, the results are empty
	osVersionsResp = osVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp, "team_id", "0")
	require.Len(t, osVersionsResp.OSVersions, 0)
	osVersionsResp = osVersionsResponse{}
	// move the host to "no team" and update the stats
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), nil, []uint{hosts[0].ID}))
	require.NoError(t, s.ds.UpdateOSVersions(context.Background()))
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp, "team_id", "0")
	require.Len(t, osVersionsResp.OSVersions, 1)
	osVersionResp = getOSVersionResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusOK, &osVersionResp, "team_id", "0")
	require.Equal(t, &expectedOSVersion, osVersionResp.OSVersion)
	require.Equal(t, 1, osVersionResp.OSVersion.HostsCount)
}

func (s *integrationEnterpriseTestSuite) TestMDMNotConfiguredEndpoints() {
	t := s.T()

	// create a host with device token to test device authenticated routes
	tkn := "D3V1C370K3N"
	createHostAndDeviceToken(t, s.ds, tkn)

	for _, route := range mdmConfigurationRequiredEndpoints() {
		var expectedErr fleet.ErrWithStatusCode = fleet.ErrMDMNotConfigured
		path := route.path
		if route.deviceAuthenticated {
			path = fmt.Sprintf(route.path, tkn)
		}

		res := s.Do(route.method, path, nil, expectedErr.StatusCode())
		errMsg := extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, expectedErr.Error())
	}

	fleetdmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Setenv("TEST_FLEETDM_API_URL", fleetdmSrv.URL)
	t.Cleanup(fleetdmSrv.Close)

	// Always accessible
	var reqCSRResp requestMDMAppleCSRResponse
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/request_csr", requestMDMAppleCSRRequest{EmailAddress: "a@b.c", Organization: "test"}, http.StatusOK, &reqCSRResp)
	s.Do("POST", "/api/latest/fleet/mdm/apple/dep/key_pair", nil, http.StatusOK)

	// setting enable release device manually requires MDM
	res := s.Do("PATCH", "/api/v1/fleet/setup_experience", fleet.MDMAppleSetupPayload{EnableReleaseDeviceManually: ptr.Bool(true)}, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, fleet.ErrMDMNotConfigured.Error())

	res = s.Do("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_setup": { "enable_release_device_manually": true } }
	}`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Couldn't update macos_setup because MDM features aren't turned on in Fleet.`)
}

func (s *integrationEnterpriseTestSuite) TestGlobalPolicyCreateReadPatch() {
	t := s.T()
	fields := []string{"Query", "Name", "Description", "Resolution", "Platform", "Critical"}

	createPol1 := &globalPolicyResponse{}
	createPol1Req := &globalPolicyRequest{
		Query:       "query",
		Name:        "name1",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    true,
	}
	s.DoJSON("POST", "/api/latest/fleet/policies", createPol1Req, http.StatusOK, &createPol1)
	allEqual(t, createPol1Req, createPol1.Policy, fields...)

	createPol2 := &globalPolicyResponse{}
	createPol2Req := &globalPolicyRequest{
		Query:       "query",
		Name:        "name2",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    false,
	}
	s.DoJSON("POST", "/api/latest/fleet/policies", createPol2Req, http.StatusOK, &createPol2)
	allEqual(t, createPol2Req, createPol2.Policy, fields...)

	listPol := &listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, listPol)
	require.Len(t, listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	require.Equal(t, createPol1.Policy, listPol.Policies[0])
	require.Equal(t, createPol2.Policy, listPol.Policies[1])

	// match policy by name with leading/trailing whitespace
	listPolByName := &listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, listPolByName, "query", " name1 ")
	require.Len(t, listPolByName.Policies, 1)
	require.Equal(t, listPolByName.Policies[0].Name, "name1")

	patchPol1Req := &modifyGlobalPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:        ptr.String("newName1"),
			Query:       ptr.String("newQuery"),
			Description: ptr.String("newDescription"),
			Resolution:  ptr.String("newResolution"),
			Platform:    ptr.String("windows"),
			Critical:    ptr.Bool(false),
		},
	}
	patchPol1 := &modifyGlobalPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol1.Policy.ID), patchPol1Req, http.StatusOK, patchPol1)
	allEqual(t, patchPol1Req, patchPol1.Policy, fields...)

	patchPol2Req := &modifyGlobalPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:        ptr.String("newName2"),
			Query:       ptr.String("newQuery"),
			Description: ptr.String("newDescription"),
			Resolution:  ptr.String("newResolution"),
			Platform:    ptr.String("windows"),
			Critical:    ptr.Bool(true),
		},
	}
	patchPol2 := &modifyGlobalPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol2.Policy.ID), patchPol2Req, http.StatusOK, patchPol2)
	allEqual(t, patchPol2Req, patchPol2.Policy, fields...)

	listPol = &listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, listPol)
	require.Len(t, listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	// not using require.Equal because "PATCH policies" returns the wrong updated timestamp.
	allEqual(t, patchPol1.Policy, listPol.Policies[0], fields...)
	allEqual(t, patchPol2.Policy, listPol.Policies[1], fields...)

	getPol2 := &getPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol2.Policy.ID), nil, http.StatusOK, getPol2)
	require.Equal(t, listPol.Policies[1], getPol2.Policy)
}

func (s *integrationEnterpriseTestSuite) TestTeamPolicyCreateReadPatch() {
	fields := []string{"Query", "Name", "Description", "Resolution", "Platform", "Critical", "CalendarEventsEnabled"}

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(s.T(), err)

	createPol1 := &teamPolicyResponse{}
	createPol1Req := &teamPolicyRequest{
		Query:                 "query",
		Name:                  "name1",
		Description:           "description",
		Resolution:            "resolution",
		Platform:              "linux",
		Critical:              true,
		CalendarEventsEnabled: true,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol1Req, http.StatusOK, &createPol1)
	allEqual(s.T(), createPol1Req, createPol1.Policy, fields...)

	createPol2 := &teamPolicyResponse{}
	createPol2Req := &teamPolicyRequest{
		Query:                 "query",
		Name:                  "name2",
		Description:           "description",
		Resolution:            "resolution",
		Platform:              "linux",
		Critical:              false,
		CalendarEventsEnabled: false,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol2Req, http.StatusOK, &createPol2)
	allEqual(s.T(), createPol2Req, createPol2.Policy, fields...)

	listPol := &listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, listPol)
	require.Len(s.T(), listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	require.Equal(s.T(), createPol1.Policy, listPol.Policies[0])
	require.Equal(s.T(), createPol2.Policy, listPol.Policies[1])

	patchPol1Req := &modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:                  ptr.String("newName1"),
			Query:                 ptr.String("newQuery"),
			Description:           ptr.String("newDescription"),
			Resolution:            ptr.String("newResolution"),
			Platform:              ptr.String("windows"),
			Critical:              ptr.Bool(false),
			CalendarEventsEnabled: ptr.Bool(false),
		},
	}
	patchPol1 := &modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, createPol1.Policy.ID), patchPol1Req, http.StatusOK, patchPol1)
	allEqual(s.T(), patchPol1Req, patchPol1.Policy, fields...)

	patchPol2Req := &modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:                  ptr.String("newName2"),
			Query:                 ptr.String("newQuery"),
			Description:           ptr.String("newDescription"),
			Resolution:            ptr.String("newResolution"),
			Platform:              ptr.String("windows"),
			Critical:              ptr.Bool(true),
			CalendarEventsEnabled: ptr.Bool(true),
		},
	}
	patchPol2 := &modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, createPol2.Policy.ID), patchPol2Req, http.StatusOK, patchPol2)
	allEqual(s.T(), patchPol2Req, patchPol2.Policy, fields...)

	listPol = &listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, listPol)
	require.Len(s.T(), listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	// not using require.Equal because "PATCH policies" returns the wrong updated timestamp.
	allEqual(s.T(), patchPol1.Policy, listPol.Policies[0], fields...)
	allEqual(s.T(), patchPol2.Policy, listPol.Policies[1], fields...)

	getPol2 := &getPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, createPol2.Policy.ID), nil, http.StatusOK, getPol2)
	require.Equal(s.T(), listPol.Policies[1], getPol2.Policy)
}

func (s *integrationEnterpriseTestSuite) TestResetAutomation() {
	ctx := context.Background()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(s.T(), err)

	createPol1 := &teamPolicyResponse{}
	createPol1Req := &teamPolicyRequest{
		Query:       "query",
		Name:        "name1",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    true,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol1Req, http.StatusOK, &createPol1)

	createPol2 := &teamPolicyResponse{}
	createPol2Req := &teamPolicyRequest{
		Query:       "query",
		Name:        "name2",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    false,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol2Req, http.StatusOK, &createPol2)

	createPol3 := &teamPolicyResponse{}
	createPol3Req := &teamPolicyRequest{
		Query:       "query",
		Name:        "name3",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    false,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol3Req, http.StatusOK, &createPol3)

	var tmResp teamResponse
	// modify the team's config - enable the webhook
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team1.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamWebhookSettings{
		FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://127/",
			PolicyIDs:      []uint{createPol1.Policy.ID, createPol2.Policy.ID},
			HostBatchSize:  12345,
		},
	}}, http.StatusOK, &tmResp)

	h1, err := s.ds.NewHost(ctx, &fleet.Host{})
	require.NoError(s.T(), err)

	err = s.ds.RecordPolicyQueryExecutions(ctx, h1, map[uint]*bool{
		createPol1.Policy.ID: ptr.Bool(false),
		createPol2.Policy.ID: ptr.Bool(false),
		createPol3.Policy.ID: ptr.Bool(false), // This policy is not activated for automation in config.
	}, time.Now(), false)
	require.NoError(s.T(), err)

	pfs, err := s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(s.T(), err)
	require.Empty(s.T(), pfs)

	s.DoJSON("POST", "/api/latest/fleet/automations/reset", resetAutomationRequest{
		TeamIDs:   nil,
		PolicyIDs: []uint{},
	}, http.StatusOK, &tmResp)

	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(s.T(), err)
	require.Empty(s.T(), pfs)

	s.DoJSON("POST", "/api/latest/fleet/automations/reset", resetAutomationRequest{
		TeamIDs:   nil,
		PolicyIDs: []uint{createPol1.Policy.ID, createPol2.Policy.ID, createPol3.Policy.ID},
	}, http.StatusOK, &tmResp)

	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(s.T(), err)
	require.Len(s.T(), pfs, 2)

	s.DoJSON("POST", "/api/latest/fleet/automations/reset", resetAutomationRequest{
		TeamIDs:   []uint{team1.ID},
		PolicyIDs: nil,
	}, http.StatusOK, &tmResp)

	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(s.T(), err)
	require.Len(s.T(), pfs, 2)

	s.DoJSON("POST", "/api/latest/fleet/automations/reset", resetAutomationRequest{
		TeamIDs:   nil,
		PolicyIDs: []uint{createPol2.Policy.ID},
	}, http.StatusOK, &tmResp)

	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(s.T(), err)
	require.Len(s.T(), pfs, 1)
}

// allEqual compares all fields of a struct.
// If a field is a pointer on one side but not on the other, then it follows that pointer. This is useful for optional
// arguments.
func allEqual(t *testing.T, expect, actual interface{}, fields ...string) {
	require.NotEmpty(t, fields)
	t.Helper()
	expV := reflect.Indirect(reflect.ValueOf(expect))
	actV := reflect.Indirect(reflect.ValueOf(actual))
	for _, f := range fields {
		e, a := expV.FieldByName(f), actV.FieldByName(f)
		switch {
		case e.Kind() == reflect.Ptr && a.Kind() != reflect.Ptr && !e.IsZero():
			e = e.Elem()
		case a.Kind() == reflect.Ptr && e.Kind() != reflect.Ptr && !a.IsZero():
			a = a.Elem()
		}
		require.Equal(t, e.Interface(), a.Interface(), "%s", f)
	}
}

func createHostAndDeviceToken(t *testing.T, ds *mysql.Datastore, token string) *fleet.Host {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		HardwareSerial:  uuid.New().String(),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	createDeviceTokenForHost(t, ds, host.ID, token)

	return host
}

func updateDeviceTokenForHost(t *testing.T, ds *mysql.Datastore, hostID uint, token string) {
	mysql.ExecAdhocSQL(t, ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `UPDATE host_device_auth SET token = ? WHERE host_id = ?`, token, hostID)
		return err
	})
}

func createDeviceTokenForHost(t *testing.T, ds *mysql.Datastore, hostID uint, token string) {
	mysql.ExecAdhocSQL(t, ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `INSERT INTO host_device_auth (host_id, token) VALUES (?, ?)`, hostID, token)
		return err
	})
}

func (s *integrationEnterpriseTestSuite) TestListSoftware() {
	t := s.T()
	now := time.Now().UTC().Truncate(time.Second)
	ctx := context.Background()

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(t.Name() + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "apps"},
	}
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))

	bar := host.Software[0]
	if bar.Name != "bar" {
		bar = host.Software[1]
	}

	inserted, err := s.ds.InsertSoftwareVulnerability(
		ctx, fleet.SoftwareVulnerability{
			SoftwareID:        bar.ID,
			CVE:               "cve-123",
			ResolvedInVersion: ptr.String("1.2.3"),
		}, fleet.NVDSource,
	)
	require.NoError(t, err)
	require.True(t, inserted)

	require.NoError(t, s.ds.InsertCVEMeta(ctx, []fleet.CVEMeta{{
		CVE:              "cve-123",
		CVSSScore:        ptr.Float64(5.4),
		EPSSProbability:  ptr.Float64(0.5),
		CISAKnownExploit: ptr.Bool(true),
		Published:        &now,
		Description:      "a long description of the cve",
	}}))

	require.NoError(t, s.ds.SyncHostsSoftware(ctx, time.Now().UTC()))

	var resp listSoftwareResponse
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &resp)
	require.NotNil(t, resp)

	var fooPayload, barPayload fleet.Software
	for _, s := range resp.Software {
		switch s.Name {
		case "foo":
			fooPayload = s
		case "bar":
			barPayload = s
		default:
			require.Failf(t, "unrecognized software %s", s.Name)

		}
	}

	require.Empty(t, fooPayload.Vulnerabilities)
	require.Len(t, barPayload.Vulnerabilities, 1)
	require.Equal(t, barPayload.Vulnerabilities[0].CVE, "cve-123")
	require.NotNil(t, barPayload.Vulnerabilities[0].CVSSScore, ptr.Float64Ptr(5.4))
	require.NotNil(t, barPayload.Vulnerabilities[0].EPSSProbability, ptr.Float64Ptr(0.5))
	require.NotNil(t, barPayload.Vulnerabilities[0].CISAKnownExploit, ptr.BoolPtr(true))
	require.Equal(t, barPayload.Vulnerabilities[0].CVEPublished, ptr.TimePtr(now))
	require.Equal(t, barPayload.Vulnerabilities[0].Description, ptr.StringPtr("a long description of the cve"))
	require.Equal(t, barPayload.Vulnerabilities[0].ResolvedInVersion, ptr.StringPtr("1.2.3"))

	var respVersions listSoftwareVersionsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &respVersions)
	require.NotNil(t, resp)

	for _, s := range resp.Software {
		switch s.Name {
		case "foo":
			fooPayload = s
		case "bar":
			barPayload = s
		default:
			require.Failf(t, "unrecognized software %s", s.Name)

		}
	}

	require.Empty(t, fooPayload.Vulnerabilities)
	require.Len(t, barPayload.Vulnerabilities, 1)
	require.Equal(t, barPayload.Vulnerabilities[0].CVE, "cve-123")
	require.NotNil(t, barPayload.Vulnerabilities[0].CVSSScore, ptr.Float64Ptr(5.4))
	require.NotNil(t, barPayload.Vulnerabilities[0].EPSSProbability, ptr.Float64Ptr(0.5))
	require.NotNil(t, barPayload.Vulnerabilities[0].CISAKnownExploit, ptr.BoolPtr(true))
	require.Equal(t, barPayload.Vulnerabilities[0].CVEPublished, ptr.TimePtr(now))
	require.Equal(t, barPayload.Vulnerabilities[0].Description, ptr.StringPtr("a long description of the cve"))
	require.Equal(t, barPayload.Vulnerabilities[0].ResolvedInVersion, ptr.StringPtr("1.2.3"))

	// vulnerable param required when using vulnerability filters
	respVersions = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"without_vulnerability_details", "true",
	)
	for _, s := range respVersions.Software {
		for _, cve := range s.Vulnerabilities {
			require.Nil(t, cve.CVSSScore)
			require.Nil(t, cve.EPSSProbability)
			require.Nil(t, cve.CISAKnownExploit)
			require.Nil(t, cve.CVEPublished)
			require.Nil(t, cve.Description)
			require.Nil(t, cve.ResolvedInVersion)
		}
	}
	// without_vulnerability_details with vulnerability filter
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"exploit", "true",
		"vulnerable", "true",
		"without_vulnerability_details", "true",
	)
	for _, s := range respVersions.Software {
		for _, cve := range s.Vulnerabilities {
			require.Nil(t, cve.CVSSScore)
			require.Nil(t, cve.EPSSProbability)
			require.Nil(t, cve.CISAKnownExploit)
			require.Nil(t, cve.CVEPublished)
			require.Nil(t, cve.Description)
			require.Nil(t, cve.ResolvedInVersion)
		}
	}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusUnprocessableEntity, &respVersions,
		"exploit", "true",
	)
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusUnprocessableEntity, &respVersions,
		"min_cvss_score", "1.1",
	)
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusUnprocessableEntity, &respVersions,
		"max_cvss_score", "10.0",
	)

	// vulnerability filters
	respVersions = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"exploit", "true",
		"vulnerable", "true",
	)
	require.Len(t, respVersions.Software, 1)
	require.NotEmpty(t, respVersions.CountsUpdatedAt)

	respVersions = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"min_cvss_score", "1",
		"vulnerable", "true",
	)
	require.Len(t, respVersions.Software, 1)
	require.NotEmpty(t, respVersions.CountsUpdatedAt)

	respVersions = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"min_cvss_score", "10",
		"vulnerable", "true",
	)
	require.Len(t, respVersions.Software, 0)
	require.Nil(t, respVersions.CountsUpdatedAt)

	respVersions = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"max_cvss_score", "10",
		"vulnerable", "true",
	)
	require.Len(t, respVersions.Software, 1)
	require.NotEmpty(t, respVersions.CountsUpdatedAt)

	respVersions = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"max_cvss_score", "1",
		"vulnerable", "true",
	)
	require.Len(t, respVersions.Software, 0)
	require.Nil(t, respVersions.CountsUpdatedAt)

	respVersions = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"min_cvss_score", "1",
		"max_cvss_score", "10",
		"vulnerable", "true",
	)
	require.Len(t, respVersions.Software, 1)
	require.NotEmpty(t, respVersions.CountsUpdatedAt)

	respVersions = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"min_cvss_score", "1",
		"max_cvss_score", "10",
		"exploit", "true",
		"vulnerable", "true",
	)
	require.Len(t, respVersions.Software, 1)
	require.NotEmpty(t, respVersions.CountsUpdatedAt)
}

// TestGitOpsUserActions tests the MDM permissions listed in ../../docs/Using\ Fleet/manage-access.md
func (s *integrationEnterpriseTestSuite) TestGitOpsUserActions() {
	t := s.T()
	ctx := context.Background()

	//
	// Setup test data.
	// All setup actions are authored by a global admin.
	//

	admin, err := s.ds.UserByEmail(ctx, "admin1@example.com")
	require.NoError(t, err)
	h1, err := s.ds.NewHost(ctx, &fleet.Host{
		NodeKey:  ptr.String(t.Name() + "1"),
		UUID:     t.Name() + "1",
		Hostname: strings.ReplaceAll(t.Name()+"foo.local", "/", "_"),
	})
	require.NoError(t, err)
	t1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Foo",
	})
	require.NoError(t, err)
	t2, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Bar",
	})
	require.NoError(t, err)
	t3, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Zoo",
	})
	require.NoError(t, err)
	team1Host, err := s.ds.NewHost(ctx, &fleet.Host{
		NodeKey:  ptr.String(t.Name() + "2"),
		UUID:     t.Name() + "2",
		Hostname: strings.ReplaceAll(t.Name()+"zoo.local", "/", "_"),
		TeamID:   &t1.ID,
	})
	require.NoError(t, err)
	globalHost, err := s.ds.NewHost(ctx, &fleet.Host{
		NodeKey:  ptr.String(t.Name() + "3"),
		UUID:     t.Name() + "3",
		Hostname: strings.ReplaceAll(t.Name()+"global.local", "/", "_"),
	})
	require.NoError(t, err)
	acr := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": false
			}
		}
	}`), http.StatusOK, &acr)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acr)
	require.False(t, acr.WebhookSettings.VulnerabilitiesWebhook.Enable)
	q1, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name:    "Foo",
		Query:   "SELECT * from time;",
		Logging: fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	ggsr := getGlobalScheduleResponse{}
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &ggsr)
	require.NoError(t, ggsr.Err)
	cpar := createPackResponse{}
	var userPackID uint
	s.DoJSON("POST", "/api/latest/fleet/packs", createPackRequest{
		PackPayload: fleet.PackPayload{
			Name:     ptr.String("Foobar"),
			Disabled: ptr.Bool(false),
		},
	}, http.StatusOK, &cpar)
	userPackID = cpar.Pack.Pack.ID
	require.NotZero(t, userPackID)
	cur := createUserResponse{}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", createUserRequest{
		UserPayload: fleet.UserPayload{
			Email:      ptr.String("foo42@example.com"),
			Password:   ptr.String("p4ssw0rd.123"),
			Name:       ptr.String("foo42"),
			GlobalRole: ptr.String("maintainer"),
		},
	}, http.StatusOK, &cur)
	maintainer := cur.User
	var carveBeginResp carveBeginResponse
	s.DoJSON("POST", "/api/osquery/carve/begin", carveBeginRequest{
		NodeKey:    *h1.NodeKey,
		BlockCount: 3,
		BlockSize:  3,
		CarveSize:  8,
		CarveId:    "c1",
		RequestId:  "r1",
	}, http.StatusOK, &carveBeginResp)
	require.NotEmpty(t, carveBeginResp.SessionId)
	lcr := listCarvesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/carves", listCarvesRequest{}, http.StatusOK, &lcr)
	require.NotEmpty(t, lcr.Carves)
	carveID := lcr.Carves[0].ID
	// Create the global GitOps user we'll use in tests.
	u := &fleet.User{
		Name:       "GitOps",
		Email:      "gitops1@example.com",
		GlobalRole: ptr.String(fleet.RoleGitOps),
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)
	// Create a GitOps user for team t1 we'll use in tests.
	u2 := &fleet.User{
		Name:       "GitOps 2",
		Email:      "gitops2@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *t1,
				Role: fleet.RoleGitOps,
			},
			{
				Team: *t3,
				Role: fleet.RoleGitOps,
			},
		},
	}
	require.NoError(t, u2.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u2)
	require.NoError(t, err)
	gp2, err := s.ds.NewGlobalPolicy(ctx, &admin.ID, fleet.PolicyPayload{
		Name:  "Zoo",
		Query: "SELECT 0;",
	})
	require.NoError(t, err)
	t2p, err := s.ds.NewTeamPolicy(ctx, t2.ID, &admin.ID, fleet.PolicyPayload{
		Name:  "Zoo2",
		Query: "SELECT 2;",
	})
	require.NoError(t, err)
	// Create some test user to test moving from/to teams.
	u3 := &fleet.User{
		Name:       "Test Foo Observer",
		Email:      "test-foo-observer@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *t1,
				Role: fleet.RoleObserver,
			},
		},
	}
	require.NoError(t, u3.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u3)
	require.NoError(t, err)
	manualLabel1, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:                "manualLabel1",
		Query:               "SELECT 2;",
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)

	//
	// Start running permission tests with user gitops1.
	//
	s.setTokenForTest(t, "gitops1@example.com", test.GoodPassword)

	// Attempt to retrieve activities, should fail.
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusForbidden, &listActivitiesResponse{})

	// Attempt to retrieve hosts, should fail.
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusForbidden, &listHostsResponse{})

	// Attempt to retrieve a host by identifier should succeed
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", h1.Hostname), hostByIdentifierRequest{}, http.StatusOK, &getHostResponse{})

	// Attempt to filter hosts using labels, should fail (label ID 6 is the builtin label "All Hosts")
	s.DoJSON("GET", "/api/latest/fleet/labels/6/hosts", nil, http.StatusForbidden, &listHostsResponse{})

	// Attempt to delete hosts, should fail.
	s.DoJSON("DELETE", "/api/latest/fleet/hosts/1", nil, http.StatusForbidden, &deleteHostResponse{})

	// Attempt to transfer host from global to a team, should allow.
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &t1.ID,
		HostIDs: []uint{h1.ID},
	}, http.StatusOK, &addHostsToTeamResponse{})

	// Attempt to create a label, should allow.
	clr := createLabelResponse{}
	s.DoJSON("POST", "/api/latest/fleet/labels", createLabelRequest{
		LabelPayload: fleet.LabelPayload{
			Name:  "foo",
			Query: "SELECT 1;",
		},
	}, http.StatusOK, &clr)

	// Attempt to modify a label, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", clr.Label.ID), modifyLabelRequest{
		ModifyLabelPayload: fleet.ModifyLabelPayload{
			Name: ptr.String("foo2"),
		},
	}, http.StatusOK, &modifyLabelResponse{})

	// Attempt to get a label, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", clr.Label.ID), getLabelRequest{}, http.StatusForbidden, &getLabelResponse{})

	// Attempt to list all labels, should fail.
	s.DoJSON("GET", "/api/latest/fleet/labels", listLabelsRequest{}, http.StatusForbidden, &listLabelsResponse{})

	// Attempt to delete a label, should allow.
	s.DoJSON("DELETE", "/api/latest/fleet/labels/foo2", deleteLabelRequest{}, http.StatusOK, &deleteLabelResponse{})

	// Attempt to list all software, should fail.
	s.DoJSON("GET", "/api/latest/fleet/software/versions", listSoftwareRequest{}, http.StatusForbidden, &listSoftwareVersionsResponse{})
	s.DoJSON("GET", "/api/latest/fleet/software", listSoftwareRequest{}, http.StatusForbidden, &listSoftwareResponse{})
	s.DoJSON("GET", "/api/latest/fleet/software/count", countSoftwareRequest{}, http.StatusForbidden, &countSoftwareResponse{})
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusForbidden, &listSoftwareTitlesResponse{})
	s.DoJSON("GET", "/api/latest/fleet/software/titles/1", getSoftwareTitleRequest{}, http.StatusForbidden, &getSoftwareTitleResponse{})

	// Attempt to list a software, should fail.
	s.DoJSON("GET", "/api/latest/fleet/software/1", getSoftwareRequest{}, http.StatusForbidden, &getSoftwareResponse{})
	s.DoJSON("GET", "/api/latest/fleet/software/versions/1", getSoftwareRequest{}, http.StatusForbidden, &getSoftwareResponse{})

	// Attempt to read app config, should pass.
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &appConfigResponse{})

	// Attempt to write app config, should allow.
	acr = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": true,
				"destination_url": "https://foobar.example.com"
			}
		}
	}`), http.StatusOK, &acr)
	require.True(t, acr.AppConfig.WebhookSettings.VulnerabilitiesWebhook.Enable)
	require.Equal(t, "https://foobar.example.com", acr.AppConfig.WebhookSettings.VulnerabilitiesWebhook.DestinationURL)

	// Attempt to add/remove manual labels to/from a host.
	var addLabelsToHostResp addLabelsToHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", h1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &addLabelsToHostResp)
	var removeLabelsFromHostResp removeLabelsFromHostResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", h1.ID), removeLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)

	// Attempt to run live queries synchronously, should fail.
	s.DoJSON("GET", "/api/latest/fleet/queries/run", runLiveQueryRequest{
		HostIDs:  []uint{h1.ID},
		QueryIDs: []uint{q1.ID},
	}, http.StatusForbidden, &runLiveQueryResponse{},
	)

	// Attempt to run live queries asynchronously (new unsaved query), should fail.
	s.DoJSON("POST", "/api/latest/fleet/queries/run", createDistributedQueryCampaignRequest{
		QuerySQL: "SELECT * FROM time;",
		Selected: fleet.HostTargets{
			HostIDs: []uint{h1.ID},
		},
	}, http.StatusForbidden, &runLiveQueryResponse{})

	// Attempt to run live queries asynchronously (saved query), should fail.
	s.DoJSON("POST", "/api/latest/fleet/queries/run", createDistributedQueryCampaignRequest{
		QueryID: ptr.Uint(q1.ID),
		Selected: fleet.HostTargets{
			HostIDs: []uint{h1.ID},
		},
	}, http.StatusForbidden, &runLiveQueryResponse{})

	// Attempt to create queries, should allow.
	cqr := createQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", createQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  ptr.String("foo4"),
			Query: ptr.String("SELECT * from osquery_info;"),
		},
	}, http.StatusOK, &cqr)
	cqr2 := createQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", createQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  ptr.String("foo5"),
			Query: ptr.String("SELECT * from os_version;"),
		},
	}, http.StatusOK, &cqr2)
	cqr3 := createQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", createQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  ptr.String("foo6"),
			Query: ptr.String("SELECT * from processes;"),
		},
	}, http.StatusOK, &cqr3)
	cqr4 := createQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", createQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  ptr.String("foo7"),
			Query: ptr.String("SELECT * from managed_policies;"),
		},
	}, http.StatusOK, &cqr4)

	// Attempt to edit queries, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", cqr.Query.ID), modifyQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  ptr.String("foo4"),
			Query: ptr.String("SELECT * FROM system_info;"),
		},
	}, http.StatusOK, &modifyQueryResponse{})

	// Attempt to view a query, should work.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d", cqr.Query.ID), getQueryRequest{}, http.StatusOK, &getQueryResponse{})

	// Attempt to list all queries, should work.
	s.DoJSON("GET", "/api/latest/fleet/queries", listQueriesRequest{}, http.StatusOK, &listQueriesResponse{})

	// Attempt to delete queries, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", cqr.Query.ID), deleteQueryByIDRequest{}, http.StatusOK, &deleteQueryByIDResponse{})
	s.DoJSON("POST", "/api/latest/fleet/queries/delete", deleteQueriesRequest{IDs: []uint{cqr2.Query.ID}}, http.StatusOK, &deleteQueriesResponse{})
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/%s", cqr3.Query.Name), deleteQueryRequest{}, http.StatusOK, &deleteQueryResponse{})

	// Attempt to add a query to a user pack, should allow.
	sqr := scheduleQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", scheduleQueryRequest{
		PackID:   userPackID,
		QueryID:  cqr4.Query.ID,
		Interval: 60,
	}, http.StatusOK, &sqr)

	// Attempt to edit a scheduled query in the global schedule, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sqr.Scheduled.ID), modifyScheduledQueryRequest{
		ScheduledQueryPayload: fleet.ScheduledQueryPayload{
			Interval: ptr.Uint(30),
		},
	}, http.StatusOK, &scheduleQueryResponse{})

	// Attempt to remove a query from the global schedule, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sqr.Scheduled.ID), deleteScheduledQueryRequest{}, http.StatusOK, &scheduleQueryResponse{})

	// Attempt to read the global schedule, should allow.
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &getGlobalScheduleResponse{})

	// Attempt to create a pack, should allow.
	cpr := createPackResponse{}
	s.DoJSON("POST", "/api/latest/fleet/packs", createPackRequest{
		PackPayload: fleet.PackPayload{
			Name: ptr.String("foo8"),
		},
	}, http.StatusOK, &cpr)

	// Attempt to edit a pack, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/%d", cpr.Pack.ID), modifyPackRequest{
		PackPayload: fleet.PackPayload{
			Name: ptr.String("foo9"),
		},
	}, http.StatusOK, &modifyPackResponse{})

	// Attempt to read a pack, should allow.
	// This is an exception to the "write only" nature of gitops (packs can be viewed by gitops).
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d", cpr.Pack.ID), nil, http.StatusOK, &getPackResponse{})

	// Attempt to delete a pack, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/id/%d", cpr.Pack.ID), deletePackRequest{}, http.StatusOK, &deletePackResponse{})

	// Attempt to create a global policy, should allow.
	gplr := globalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", globalPolicyRequest{
		Name:  "foo9",
		Query: "SELECT * from plist;",
	}, http.StatusOK, &gplr)

	// Attempt to edit a global policy, should allow.
	mgplr := modifyGlobalPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", gplr.Policy.ID), modifyGlobalPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Query: ptr.String("SELECT * from plist WHERE path = 'foo';"),
		},
	}, http.StatusOK, &mgplr)

	// Attempt to read a global policy, should allow.
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/policies/%d", gplr.Policy.ID), getPolicyByIDRequest{}, http.StatusOK,
		&getPolicyByIDResponse{},
	)

	// Attempt to delete a global policy, should allow.
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deleteGlobalPoliciesRequest{
		IDs: []uint{gplr.Policy.ID},
	}, http.StatusOK, &deleteGlobalPoliciesResponse{})

	// Attempt to create a team policy, should allow.
	tplr := teamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/team/%d/policies", t1.ID), teamPolicyRequest{
		Name:  "foo10",
		Query: "SELECT * from file;",
	}, http.StatusOK, &tplr)

	// Attempt to edit a team policy, should allow.
	mtplr := modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", t1.ID, tplr.Policy.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Query: ptr.String("SELECT * from file WHERE path = 'foo';"),
		},
	}, http.StatusOK, &mtplr)

	// Attempt to view a team policy, should allow.
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/team/%d/policies/%d", t1.ID, tplr.Policy.ID), getTeamPolicyByIDRequest{}, http.StatusOK,
		&getTeamPolicyByIDResponse{},
	)

	// Attempt to delete a team policy, should allow.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", t1.ID), deleteTeamPoliciesRequest{
		IDs: []uint{tplr.Policy.ID},
	}, http.StatusOK, &deleteTeamPoliciesResponse{})

	// Attempt to create a user, should fail.
	s.DoJSON("POST", "/api/latest/fleet/users/admin", createUserRequest{
		UserPayload: fleet.UserPayload{
			Email:      ptr.String("foo10@example.com"),
			Name:       ptr.String("foo10"),
			GlobalRole: ptr.String("admin"),
		},
	}, http.StatusForbidden, &createUserResponse{})

	// Attempt to modify a user, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", admin.ID), modifyUserRequest{
		UserPayload: fleet.UserPayload{
			GlobalRole: ptr.String("observer"),
		},
	}, http.StatusForbidden, &modifyUserResponse{})

	// Attempt to view a user, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", admin.ID), getUserRequest{}, http.StatusForbidden, &getUserResponse{})

	// Attempt to delete a user, should fail.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/users/%d", admin.ID), deleteUserRequest{}, http.StatusForbidden, &deleteUserResponse{})

	// Attempt to add users to team, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t1.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *maintainer,
				Role: "admin",
			},
		},
	}, http.StatusOK, &teamResponse{})

	// Attempt to delete users from team, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t1.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *maintainer,
				Role: "admin",
			},
		},
	}, http.StatusOK, &teamResponse{})

	// Attempt to create a team, should allow.
	tr := teamResponse{}
	s.DoJSON("POST", "/api/latest/fleet/teams", createTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: ptr.String("foo11"),
		},
	}, http.StatusOK, &tr)

	// Attempt to edit a team, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tr.Team.ID), modifyTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: ptr.String("foo12"),
		},
	}, http.StatusOK, &teamResponse{})

	// Attempt to edit a team's agent options, should allow.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tr.Team.ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": true
			}
		}
	}`), http.StatusOK, &teamResponse{})

	// Attempt to view a team, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", tr.Team.ID), getTeamRequest{}, http.StatusForbidden, &teamResponse{})

	// Attempt to delete a team, should allow.
	dtr := deleteTeamResponse{}
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d", tr.Team.ID), deleteTeamRequest{}, http.StatusOK, &dtr)

	// Attempt to create/edit enroll secrets, should allow.
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{
				{
					Secret: "foo400",
					TeamID: nil,
				},
				{
					Secret: "foo500",
					TeamID: ptr.Uint(t1.ID),
				},
			},
		},
	}, http.StatusOK, &applyEnrollSecretSpecResponse{})

	// Attempt to get enroll secrets, should fail.
	s.DoJSON("GET", "/api/latest/fleet/spec/enroll_secret", nil, http.StatusForbidden, &getEnrollSecretSpecResponse{})

	// Attempt to get team enroll secret, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", t1.ID), teamEnrollSecretsRequest{}, http.StatusForbidden, &teamEnrollSecretsResponse{})

	// Attempt to list carved files, should fail.
	s.DoJSON("GET", "/api/latest/fleet/carves", listCarvesRequest{}, http.StatusForbidden, &listCarvesResponse{})

	// Attempt to get a carved file, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d", carveID), listCarvesRequest{}, http.StatusForbidden, &listCarvesResponse{})

	// Attempt to search hosts, should fail.
	s.DoJSON("POST", "/api/latest/fleet/targets", searchTargetsRequest{
		MatchQuery: "foo",
		QueryID:    &q1.ID,
	}, http.StatusForbidden, &searchTargetsResponse{})

	// Attempt to count target hosts, should fail.
	s.DoJSON("POST", "/api/latest/fleet/targets/count", countTargetsRequest{
		Selected: fleet.HostTargets{
			HostIDs:  []uint{h1.ID},
			LabelIDs: []uint{clr.Label.ID},
			TeamIDs:  []uint{t1.ID},
		},
		QueryID: &q1.ID,
	}, http.StatusForbidden, &countTargetsResponse{})

	//
	// Start running permission tests with user gitops2 (which is a GitOps use for team t1).
	//

	s.setTokenForTest(t, "gitops2@example.com", test.GoodPassword)

	// Attempt to create queries in global domain, should fail.
	tcqr := createQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", createQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  ptr.String("foo600"),
			Query: ptr.String("SELECT * from orbit_info;"),
		},
	}, http.StatusForbidden, &tcqr)

	// Attempt to create queries in its team, should allow.
	tcqr = createQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", createQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:   ptr.String("foo600"),
			Query:  ptr.String("SELECT * from orbit_info;"),
			TeamID: &t1.ID,
		},
	}, http.StatusOK, &tcqr)

	// Attempt to edit own query, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", tcqr.Query.ID), modifyQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  ptr.String("foo4"),
			Query: ptr.String("SELECT * FROM system_info;"),
		},
	}, http.StatusOK, &modifyQueryResponse{})

	// Attempt to delete own query, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", tcqr.Query.ID), deleteQueryByIDRequest{}, http.StatusOK, &deleteQueryByIDResponse{})

	// Attempt to edit query created by somebody else, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", cqr4.Query.ID), modifyQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  ptr.String("foo4"),
			Query: ptr.String("SELECT * FROM system_info;"),
		},
	}, http.StatusForbidden, &modifyQueryResponse{})

	// Attempt to delete query created by somebody else, should fail.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", cqr4.Query.ID), deleteQueryByIDRequest{}, http.StatusForbidden, &deleteQueryByIDResponse{})

	// Attempt to read the global schedule, should fail.
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusForbidden, &getGlobalScheduleResponse{})

	// Attempt to read the team's schedule, should allow.
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", t1.ID), getTeamScheduleRequest{}, http.StatusOK,
		&getTeamScheduleResponse{},
	)

	// Attempt to read other team's schedule, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", t2.ID), getTeamScheduleRequest{}, http.StatusForbidden, &getTeamScheduleResponse{})

	// Attempt to add a query to a user pack, should fail.
	tsqr := scheduleQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", scheduleQueryRequest{
		PackID:   userPackID,
		QueryID:  cqr4.Query.ID,
		Interval: 60,
	}, http.StatusForbidden, &tsqr)

	// Attempt to add a query to the team's schedule, should allow.
	cqrt1 := createQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", createQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:   ptr.String("foo8"),
			Query:  ptr.String("SELECT * from managed_policies;"),
			TeamID: &t1.ID,
		},
	}, http.StatusOK, &cqrt1)
	ttsqr := teamScheduleQueryResponse{}
	// Add a schedule with the deprecated APIs (by referencing a global query).
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", t1.ID), teamScheduleQueryRequest{
		ScheduledQueryPayload: fleet.ScheduledQueryPayload{
			QueryID:  ptr.Uint(q1.ID),
			Interval: ptr.Uint(60),
		},
	}, http.StatusOK, &ttsqr)

	// Attempt to remove a query from the team's schedule, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule/%d", t1.ID, ttsqr.Scheduled.ID), deleteTeamScheduleRequest{}, http.StatusOK, &deleteTeamScheduleResponse{})

	// Attempt to add/remove a manual label from a team host, should allow.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", team1Host.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &addLabelsToHostResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", team1Host.ID), removeLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)

	// Attempt to add/remove a manual label from a global host, should not allow.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", globalHost.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &addLabelsToHostResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", globalHost.ID), removeLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &removeLabelsFromHostResp)

	// Attempt to read the global schedule, should fail.
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusForbidden, &getGlobalScheduleResponse{})

	// Attempt to read a global policy, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", gp2.ID), getPolicyByIDRequest{}, http.StatusForbidden, &getPolicyByIDResponse{})

	// Attempt to delete a global policy, should fail.
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deleteGlobalPoliciesRequest{
		IDs: []uint{gp2.ID},
	}, http.StatusForbidden, &deleteGlobalPoliciesResponse{})

	// Attempt to create a team policy, should allow.
	ttplr := teamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/team/%d/policies", t1.ID), teamPolicyRequest{
		Name:  "foo1000",
		Query: "SELECT * from file;",
	}, http.StatusOK, &ttplr)

	// Attempt to edit a team policy, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", t1.ID, ttplr.Policy.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Query: ptr.String("SELECT * from file WHERE path = 'foobar';"),
		},
	}, http.StatusOK, &modifyTeamPolicyResponse{})

	// Attempt to edit another team's policy, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", t2.ID, t2p.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Query: ptr.String("SELECT * from file WHERE path = 'foobar';"),
		},
	}, http.StatusForbidden, &modifyTeamPolicyResponse{})

	// Attempt to view a team policy, should allow.
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/team/%d/policies/%d", t1.ID, ttplr.Policy.ID), getTeamPolicyByIDRequest{}, http.StatusOK,
		&getTeamPolicyByIDResponse{},
	)

	// Attempt to view another team's policy, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/team/%d/policies/%d", t2.ID, t2p.ID), getTeamPolicyByIDRequest{}, http.StatusForbidden, &getTeamPolicyByIDResponse{})

	// Attempt to delete a team policy, should allow.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", t1.ID), deleteTeamPoliciesRequest{
		IDs: []uint{ttplr.Policy.ID},
	}, http.StatusOK, &deleteTeamPoliciesResponse{})

	// Attempt to edit own team, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", t1.ID), modifyTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: ptr.String("foo123456"),
		},
	}, http.StatusOK, &teamResponse{})

	// Attempt to edit another team, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", t2.ID), modifyTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: ptr.String("foo123456"),
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to edit own team's agent options, should allow.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", t1.ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": true
			}
		}
	}`), http.StatusOK, &teamResponse{})

	// Attempt to edit another team's agent options, should fail.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", t2.ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": true
			}
		}
	}`), http.StatusForbidden, &teamResponse{})

	// Attempt to add users from team it owns to another team it owns, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t3.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *u3,
				Role: "maintainer",
			},
		},
	}, http.StatusOK, &teamResponse{})

	// Attempt to delete users from team it owns, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t3.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *u3,
			},
		},
	}, http.StatusOK, &teamResponse{})

	// Attempt to add users to another team it doesn't own, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t2.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *u3,
				Role: "maintainer",
			},
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to delete users from team it doesn't own, should fail.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t2.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *u2,
			},
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to search hosts, should fail.
	s.DoJSON("POST", "/api/latest/fleet/targets", searchTargetsRequest{
		MatchQuery: "foo",
		QueryID:    &q1.ID,
	}, http.StatusForbidden, &searchTargetsResponse{})

	// Attempt to count target hosts, should fail.
	s.DoJSON("POST", "/api/latest/fleet/targets/count", countTargetsRequest{
		Selected: fleet.HostTargets{
			HostIDs:  []uint{h1.ID},
			LabelIDs: []uint{clr.Label.ID},
			TeamIDs:  []uint{t1.ID},
		},
		QueryID: &q1.ID,
	}, http.StatusForbidden, &countTargetsResponse{})
}

func (s *integrationEnterpriseTestSuite) TestDesktopEndpointWithInvalidPolicy() {
	t := s.T()

	token := "abcd123"
	host := createHostAndDeviceToken(t, s.ds, token)

	// Create an 'invalid' global policy for host
	admin := s.users["admin1@example.com"]
	err := s.ds.SaveUser(context.Background(), &admin)
	require.NoError(t, err)

	policy, err := s.ds.NewGlobalPolicy(context.Background(), &admin.ID, fleet.PolicyPayload{
		Query:       "SELECT 1 FROM table",
		Name:        "test",
		Description: "Some invalid Query",
		Resolution:  "",
		Platform:    host.Platform,
		Critical:    false,
	})
	require.NoError(t, err)
	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{policy.ID: nil}, time.Now(), false))

	// Any 'invalid' policies should be ignored.
	desktopRes := fleetDesktopResponse{}
	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
	require.NoError(t, json.NewDecoder(res.Body).Decode(&desktopRes))
	require.NoError(t, res.Body.Close())
	require.NoError(t, desktopRes.Err)
	require.Equal(t, uint(0), *desktopRes.FailingPolicies)
}

func (s *integrationEnterpriseTestSuite) TestRunHostScript() {
	t := s.T()

	testRunScriptWaitForResult = 2 * time.Second
	defer func() { testRunScriptWaitForResult = 0 }()

	ctx := context.Background()

	host := createOrbitEnrolledHost(t, "linux", "", s.ds)
	otherHost := createOrbitEnrolledHost(t, "linux", "other", s.ds)

	// attempt to run a script on a non-existing host
	var runResp runScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID + 100, ScriptContents: "echo"}, http.StatusNotFound, &runResp)

	// attempt to run an empty script
	res := s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: ""}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Validation Failed: One of 'script_id', 'script_contents', or 'script_name' is required.")

	// attempt to run an overly long script
	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: strings.Repeat("a", fleet.UnsavedScriptMaxRuneLen+1)}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Script is too large. It's limited to 10,000 characters")

	// make sure the host is still seen as "online"
	err := s.ds.MarkHostsSeen(ctx, []uint{host.ID}, time.Now())
	require.NoError(t, err)

	// make sure invalid secrets aren't allowed
	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo $FLEET_SECRET_INVALID"}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `$FLEET_SECRET_INVALID`)

	// Upload a valid secret
	secretValue := "abc123"
	req := secretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  "FLEET_SECRET_TestRunHostScript",
				Value: secretValue,
			},
		},
	}
	secretResp := secretVariablesResponse{}
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &secretResp)

	// create a valid script execution request
	expectedScriptContents := "echo ${FLEET_SECRET_TestRunHostScript}"
	expectedScriptContentsWithSecret := fmt.Sprintf("echo %s", secretValue)
	s.DoJSON("POST", "/api/latest/fleet/scripts/run",
		fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: expectedScriptContents}, http.StatusAccepted, &runResp)
	require.Equal(t, host.ID, runResp.HostID)
	require.NotEmpty(t, runResp.ExecutionID)

	result, err := s.ds.GetHostScriptExecutionResult(ctx, runResp.ExecutionID)
	require.NoError(t, err)
	require.Equal(t, host.ID, result.HostID)
	require.Equal(t, expectedScriptContents, result.ScriptContents)
	require.Nil(t, result.ExitCode)

	// get script result
	var scriptResultResp getScriptResultResponse
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+runResp.ExecutionID, nil, http.StatusOK, &scriptResultResp)
	require.Equal(t, host.ID, scriptResultResp.HostID)
	require.Equal(t, expectedScriptContents, scriptResultResp.ScriptContents)
	require.Nil(t, scriptResultResp.ExitCode)
	require.False(t, scriptResultResp.HostTimeout)
	require.Contains(t, scriptResultResp.Message, fleet.RunScriptAsyncScriptEnqueuedMsg)

	// an async script doesn't care about timeouts
	now := time.Now()
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, `UPDATE host_script_results SET created_at = ? WHERE execution_id = ?`,
			now.Add(-1*time.Hour),
			runResp.ExecutionID,
		)
		return err
	})
	scriptResultResp = getScriptResultResponse{}
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+runResp.ExecutionID, nil, http.StatusOK, &scriptResultResp)
	require.Equal(t, host.ID, scriptResultResp.HostID)
	require.Equal(t, expectedScriptContents, scriptResultResp.ScriptContents)
	require.Nil(t, scriptResultResp.ExitCode)
	require.False(t, scriptResultResp.HostTimeout)
	require.Contains(t, scriptResultResp.Message, fleet.RunScriptAsyncScriptEnqueuedMsg)

	// Disable scripts and verify that there are no Orbit notifs
	acr := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"server_settings": {
			"scripts_disabled": true
		}
	}`), http.StatusOK, &acr)
	require.True(t, acr.AppConfig.ServerSettings.ScriptsDisabled)

	var orbitResp orbitGetConfigResponse
	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)),
		http.StatusOK, &orbitResp)
	require.Empty(t, orbitResp.Notifications.PendingScriptExecutionIDs)

	// Verify that endpoints related to scripts are disabled
	srResp := s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusForbidden)
	assertBodyContains(t, srResp, fleet.RunScriptScriptsDisabledGloballyErrMsg)

	srResp = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusForbidden)
	assertBodyContains(t, srResp, fleet.RunScriptScriptsDisabledGloballyErrMsg)

	acr = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"server_settings": {
			"scripts_disabled": false
		}
	}`), http.StatusOK, &acr)
	require.False(t, acr.AppConfig.ServerSettings.ScriptsDisabled)

	// verify that orbit would get the notification that it has a script to run
	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)),
		http.StatusOK, &orbitResp)
	require.Equal(t, []string{scriptResultResp.ExecutionID}, orbitResp.Notifications.PendingScriptExecutionIDs)

	// the orbit endpoint to get a pending script to execute returns it
	var orbitGetScriptResp orbitGetScriptResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/request",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q}`, *host.OrbitNodeKey, scriptResultResp.ExecutionID)),
		http.StatusOK, &orbitGetScriptResp)
	require.Equal(t, host.ID, orbitGetScriptResp.HostID)
	require.Equal(t, scriptResultResp.ExecutionID, orbitGetScriptResp.ExecutionID)
	require.Equal(t, expectedScriptContentsWithSecret, orbitGetScriptResp.ScriptContents)

	// trying to get that script via its execution ID but a different host returns not found
	s.DoJSON("POST", "/api/fleet/orbit/scripts/request",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q}`, *otherHost.OrbitNodeKey, scriptResultResp.ExecutionID)),
		http.StatusNotFound, &orbitGetScriptResp)

	// trying to get an unknown execution id returns not found
	s.DoJSON("POST", "/api/fleet/orbit/scripts/request",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q}`, *host.OrbitNodeKey, scriptResultResp.ExecutionID+"no-such")),
		http.StatusNotFound, &orbitGetScriptResp)

	// attempt to run a sync script on a non-existing host
	var runSyncResp runScriptSyncResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID + 100, ScriptContents: "echo"}, http.StatusNotFound, &runSyncResp)

	// attempt to sync run an empty script
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: ""}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "One of 'script_id', 'script_contents', or 'script_name' is required.")

	// attempt to sync run an overly long script
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: strings.Repeat("a", fleet.UnsavedScriptMaxRuneLen+1)}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Script is too large. It's limited to 10,000 characters")

	// make sure the host is still seen as "online"
	err = s.ds.MarkHostsSeen(ctx, []uint{host.ID}, time.Now())
	require.NoError(t, err)

	// attempt to create a valid sync script execution request, fails because the
	// host has a pending script execution
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusConflict)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, fleet.RunScriptAlreadyRunningErrMsg)

	// save a result via the orbit endpoint
	var orbitPostScriptResp orbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, scriptResultResp.ExecutionID)),
		http.StatusOK, &orbitPostScriptResp)

	// verify that orbit does not receive any pending script anymore
	orbitResp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)),
		http.StatusOK, &orbitResp)
	require.Empty(t, orbitResp.Notifications.PendingScriptExecutionIDs)

	// create a valid sync script execution request, fails because the
	// request will time-out waiting for a result.
	runSyncResp = runScriptSyncResponse{}
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusRequestTimeout, &runSyncResp)
	require.Equal(t, host.ID, runSyncResp.HostID)
	require.NotEmpty(t, runSyncResp.ExecutionID)
	require.True(t, runSyncResp.HostTimeout)
	require.Contains(t, runSyncResp.Message, fleet.RunScriptHostTimeoutErrMsg)

	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, runSyncResp.ExecutionID)),
		http.StatusOK, &orbitPostScriptResp)

	// create a valid sync script execution request, and simulate a result
	// arriving before timeout.
	testRunScriptWaitForResult = 5 * time.Second
	ctx, cancel := context.WithTimeout(ctx, testRunScriptWaitForResult)
	defer cancel()

	resultsCh := make(chan *fleet.HostScriptResultPayload, 1)
	go func() {
		for range time.Tick(300 * time.Millisecond) {
			pending, err := s.ds.ListPendingHostScriptExecutions(ctx, host.ID, false)
			if err != nil {
				t.Log(err)
				return
			}
			if len(pending) > 0 {
				select {
				case <-ctx.Done():
					return
				case r := <-resultsCh:
					r.ExecutionID = pending[0].ExecutionID
					// ignoring errors in this goroutine, the HTTP request below will fail if this fails
					_, _, err = s.ds.SetHostScriptExecutionResult(ctx, r)
					if err != nil {
						t.Log(err)
					}
				}
			}
		}
	}()

	// simulate a successful script result
	resultsCh <- &fleet.HostScriptResultPayload{
		HostID:   host.ID,
		Output:   "ok",
		Runtime:  1,
		ExitCode: 0,
	}
	runSyncResp = runScriptSyncResponse{}
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusOK, &runSyncResp)
	require.Equal(t, host.ID, runSyncResp.HostID)
	require.NotEmpty(t, runSyncResp.ExecutionID)
	require.Equal(t, "ok", runSyncResp.Output)
	require.NotNil(t, runSyncResp.ExitCode)
	require.Equal(t, int64(0), *runSyncResp.ExitCode)
	require.False(t, runSyncResp.HostTimeout)

	// simulate a scripts disabled result
	resultsCh <- &fleet.HostScriptResultPayload{
		HostID:   host.ID,
		Output:   "",
		Runtime:  0,
		ExitCode: -2,
	}
	runSyncResp = runScriptSyncResponse{}
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusOK, &runSyncResp)
	require.Equal(t, host.ID, runSyncResp.HostID)
	require.NotEmpty(t, runSyncResp.ExecutionID)
	require.Empty(t, runSyncResp.Output)
	require.NotNil(t, runSyncResp.ExitCode)
	require.Equal(t, int64(-2), *runSyncResp.ExitCode)
	require.False(t, runSyncResp.HostTimeout)
	require.Contains(t, runSyncResp.Message, "Scripts are disabled")

	// create a sync execution request.
	runSyncResp = runScriptSyncResponse{}
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusRequestTimeout, &runSyncResp)

	// modify the timestamp of the script to simulate an script that has
	// been pending for a long time
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(context.Background(), "UPDATE host_script_results SET created_at = ? WHERE execution_id = ?", time.Now().Add(-24*time.Hour), runSyncResp.ExecutionID)
		return err
	})

	// fetch the results for the timed-out script
	scriptResultResp = getScriptResultResponse{}
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+runSyncResp.ExecutionID, nil, http.StatusOK, &scriptResultResp)
	require.Equal(t, host.ID, scriptResultResp.HostID)
	require.Equal(t, "echo", scriptResultResp.ScriptContents)
	require.Nil(t, scriptResultResp.ExitCode)
	require.True(t, scriptResultResp.HostTimeout)
	require.Contains(t, scriptResultResp.Message, fleet.RunScriptHostTimeoutErrMsg)

	// Disable scripts on the host
	scriptsEnabled := false
	err = s.ds.SetOrUpdateHostOrbitInfo(
		context.Background(), host.ID, "1.22.0", sql.NullString{}, sql.NullBool{Bool: scriptsEnabled, Valid: true},
	)
	require.NoError(t, err)
	s.DoJSON(
		"POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"},
		http.StatusUnprocessableEntity, &runResp,
	)
	s.DoJSON(
		"POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"},
		http.StatusUnprocessableEntity, &runResp,
	)
	// Re-enable scripts on the host
	scriptsEnabled = true
	err = s.ds.SetOrUpdateHostOrbitInfo(
		context.Background(), host.ID, "1.22.0", sql.NullString{}, sql.NullBool{Bool: scriptsEnabled, Valid: true},
	)
	require.NoError(t, err)

	// make the host "offline"
	err = s.ds.MarkHostsSeen(context.Background(), []uint{host.ID}, time.Now().Add(-time.Hour))
	require.NoError(t, err)

	// attempt to create a sync script execution request, fails because the host
	// is offline.
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, fleet.RunScriptHostOfflineErrMsg)

	// attempt to create an async script execution request, succeeds because script is added to queue.
	s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusAccepted)

	// attempt to run a script on a plain osquery host
	plainOsqueryHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-time.Minute),
		OsqueryHostID:   ptr.String("plain-osquery-host"),
		NodeKey:         ptr.String("plain-osquery-host"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", "plain-osquery-host"),
		HardwareSerial:  uuid.New().String(),
		Platform:        "linux",
	})
	require.NoError(t, err)
	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: plainOsqueryHost.ID, ScriptContents: "echo"}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), fleet.RunScriptDisabledErrMsg)
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: plainOsqueryHost.ID, ScriptContents: "echo"}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), fleet.RunScriptDisabledErrMsg)

	// create a execution request that will return a timeout
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusAccepted, &runResp)

	// simulate a host response
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": -1, "output": "script execution error: signal: killed", "timeout": 900}`, *host.OrbitNodeKey, runSyncResp.ExecutionID)),
		http.StatusOK, &orbitPostScriptResp)

	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+runSyncResp.ExecutionID, nil, http.StatusOK, &scriptResultResp)
	require.Equal(t, host.ID, scriptResultResp.HostID)
	require.Equal(t, "echo", scriptResultResp.ScriptContents)
	require.Equal(t, int64(-1), *scriptResultResp.ExitCode)
	require.Equal(t, "Timeout. Fleet stopped the script after 900 seconds to protect host performance.", scriptResultResp.Message)
	require.Equal(t, "script execution error: signal: killed", scriptResultResp.Output)
}

func (s *integrationEnterpriseTestSuite) TestRunHostSavedScript() {
	t := s.T()

	testRunScriptWaitForResult = 2 * time.Second
	defer func() { testRunScriptWaitForResult = 0 }()

	ctx := context.Background()

	host := createOrbitEnrolledHost(t, "linux", "", s.ds)
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)
	savedNoTmScript, err := s.ds.NewScript(ctx, &fleet.Script{
		TeamID:         nil,
		Name:           "no_team_script.sh",
		ScriptContents: "echo 'no team'",
	})
	require.NoError(t, err)
	savedTmScript, err := s.ds.NewScript(ctx, &fleet.Script{
		TeamID:         &tm.ID,
		Name:           "team_script.sh",
		ScriptContents: "echo 'team'",
	})
	require.NoError(t, err)

	// attempt to run a script on a non-existing host
	var runResp runScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID + 100, ScriptID: &savedNoTmScript.ID}, http.StatusNotFound, &runResp)

	// attempt to run with both script contents and id
	res := s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo", ScriptID: ptr.Uint(savedTmScript.ID + 999)}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Only one of 'script_id' or 'script_contents' is allowed.`)

	// attempt to run with unknown script id
	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: ptr.Uint(savedTmScript.ID + 999)}, http.StatusNotFound)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `No script exists for the provided "script_id".`)

	// make sure the host is still seen as "online"
	err = s.ds.MarkHostsSeen(ctx, []uint{host.ID}, time.Now())
	require.NoError(t, err)

	// attempt to run a team script on a non-team host
	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: &savedTmScript.ID}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `The script does not belong to the same team`)

	// make sure the host is still seen as "online"
	err = s.ds.MarkHostsSeen(ctx, []uint{host.ID}, time.Now())
	require.NoError(t, err)

	// create a valid script execution request
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: &savedNoTmScript.ID}, http.StatusAccepted, &runResp)
	require.Equal(t, host.ID, runResp.HostID)
	require.NotEmpty(t, runResp.ExecutionID)

	var scriptResultResp getScriptResultResponse
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+runResp.ExecutionID, nil, http.StatusOK, &scriptResultResp)
	require.Equal(t, host.ID, scriptResultResp.HostID)
	require.Equal(t, "echo 'no team'", scriptResultResp.ScriptContents)
	require.Nil(t, scriptResultResp.ExitCode)
	require.False(t, scriptResultResp.HostTimeout)
	require.Contains(t, scriptResultResp.Message, fleet.RunScriptAsyncScriptEnqueuedMsg)
	require.NotNil(t, scriptResultResp.ScriptID)
	require.Equal(t, savedNoTmScript.ID, *scriptResultResp.ScriptID)

	// verify that orbit would get the notification that it has a script to run
	var orbitResp orbitGetConfigResponse
	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)),
		http.StatusOK, &orbitResp)
	require.Equal(t, []string{scriptResultResp.ExecutionID}, orbitResp.Notifications.PendingScriptExecutionIDs)

	// the orbit endpoint to get a pending script to execute returns it
	var orbitGetScriptResp orbitGetScriptResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/request",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q}`, *host.OrbitNodeKey, scriptResultResp.ExecutionID)),
		http.StatusOK, &orbitGetScriptResp)
	require.Equal(t, host.ID, orbitGetScriptResp.HostID)
	require.Equal(t, scriptResultResp.ExecutionID, orbitGetScriptResp.ExecutionID)
	require.Equal(t, "echo 'no team'", orbitGetScriptResp.ScriptContents)

	// make sure the host is still seen as "online"
	err = s.ds.MarkHostsSeen(ctx, []uint{host.ID}, time.Now())
	require.NoError(t, err)

	// save a result via the orbit endpoint
	var orbitPostScriptResp orbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, scriptResultResp.ExecutionID)),
		http.StatusOK, &orbitPostScriptResp)

	// an activity was created for the script execution
	s.lastActivityMatches(
		fleet.ActivityTypeRanScript{}.ActivityName(),
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": %q, "script_name": %q, "script_execution_id": %q, "async": true, "policy_id": null, "policy_name": null}`,
			host.ID, host.DisplayName(), savedNoTmScript.Name, scriptResultResp.ExecutionID,
		),
		0,
	)

	// verify that orbit does not receive any pending script anymore
	orbitResp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)),
		http.StatusOK, &orbitResp)
	require.Empty(t, orbitResp.Notifications.PendingScriptExecutionIDs)

	// create a valid sync script execution request, fails because the
	// request will time-out waiting for a result.
	var runSyncResp runScriptSyncResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: &savedNoTmScript.ID}, http.StatusRequestTimeout, &runSyncResp)
	require.Equal(t, host.ID, runSyncResp.HostID)
	require.NotEmpty(t, runSyncResp.ExecutionID)
	require.NotNil(t, runSyncResp.ScriptID)
	require.Equal(t, savedNoTmScript.ID, *runSyncResp.ScriptID)
	require.Equal(t, "echo 'no team'", runSyncResp.ScriptContents)
	require.True(t, runSyncResp.HostTimeout)
	require.Contains(t, runSyncResp.Message, fleet.RunScriptHostTimeoutErrMsg)

	// attempt to run sync with both script contents and script id
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo", ScriptID: ptr.Uint(savedTmScript.ID + 999)}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Only one of 'script_id' or 'script_contents' is allowed.`)

	// attempt to run sync with both script contents and script name
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo", ScriptName: savedTmScript.Name}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Only one of 'script_contents' or 'script_name' is allowed.`)

	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo", ScriptName: savedTmScript.Name}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Only one of 'script_contents' or 'script_name' is allowed.`)

	// attempt to run sync with both script id and script name
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: ptr.Uint(savedTmScript.ID + 999), ScriptName: savedTmScript.Name}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Only one of 'script_id' or 'script_name' is allowed.`)

	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: ptr.Uint(savedTmScript.ID + 999), ScriptName: savedTmScript.Name}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Only one of 'script_id' or 'script_name' is allowed.`)

	// attempt to run sync with both script contents and team id
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo", TeamID: 1}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Only one of 'script_contents' or 'team_id' is allowed.`)

	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo", TeamID: 1}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Only one of 'script_contents' or 'team_id' is allowed.`)

	// attempt to run sync with both script id and team id
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: ptr.Uint(savedTmScript.ID + 999), TeamID: 1}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Only one of 'script_id' or 'team_id' is allowed.`)

	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: ptr.Uint(savedTmScript.ID + 999), TeamID: 1}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Only one of 'script_id' or 'team_id' is allowed.`)

	// attempt to run sync without script contents, script id, or script name
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `One of 'script_id', 'script_contents', or 'script_name' is required.`)

	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `One of 'script_id', 'script_contents', or 'script_name' is required.`)

	// deleting the saved script should delete the pending script
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/scripts/%d", savedNoTmScript.ID), nil, http.StatusNoContent)
	scriptResultResp = getScriptResultResponse{}
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+runSyncResp.ExecutionID, nil, http.StatusNotFound, &scriptResultResp)

	// Verify that we can't enqueue more than 1k scripts

	// Make the host offline so that scripts enqueue
	err = s.ds.MarkHostsSeen(ctx, []uint{host.ID}, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	for i := 1; i <= 1000; i++ {
		script, err := s.ds.NewScript(ctx, &fleet.Script{
			TeamID:         nil,
			Name:           fmt.Sprintf("script_1k_%d.sh", i),
			ScriptContents: fmt.Sprintf("echo %d", i),
		})
		require.NoError(t, err)

		_, err = s.ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: &script.ID})
		require.NoError(t, err)
	}

	script, err := s.ds.NewScript(ctx, &fleet.Script{
		TeamID:         nil,
		Name:           "script_1k_1001.sh",
		ScriptContents: "echo 1001",
	})
	require.NoError(t, err)

	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: &script.ID}, http.StatusConflict, &runResp)

	// set up a new host, new team, and some new scripts
	host2 := createOrbitEnrolledHost(t, "linux", "f1337", s.ds)
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)
	savedNoTmScript2, err := s.ds.NewScript(ctx, &fleet.Script{
		TeamID:         nil,
		Name:           "f1337.sh",
		ScriptContents: "echo 'ALL YOUR BASE ARE BELONG TO US'",
	})
	require.NoError(t, err)
	savedTmScript2, err := s.ds.NewScript(ctx, &fleet.Script{
		TeamID:         &tm2.ID,
		Name:           "f1337.sh",
		ScriptContents: "echo 'ALL YOUR BASE ARE BELONG TO US'",
	})
	require.NoError(t, err)
	require.NotEqual(t, savedNoTmScript2.ID, savedTmScript2.ID)

	_, err = s.ds.NewScript(ctx, &fleet.Script{
		TeamID:         nil,
		Name:           "f13372.sh",
		ScriptContents: "echo 'ALL YOUR BASE ARE BELONG TO US'",
	})
	require.NoError(t, err)

	// make sure the new host is seen as "online"
	err = s.ds.MarkHostsSeen(ctx, []uint{host2.ID}, time.Now())
	require.NoError(t, err)

	// attempt to run sync with a script that does not exist on the specified team
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host2.ID, ScriptName: "f1337.sh", TeamID: tm.ID}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Script 'f1337.sh' doesn’t exist.`)

	// attempt to run sync with an existing team script that belongs to a team different from the host's team
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host2.ID, ScriptName: "f1337.sh", TeamID: tm2.ID}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `The script does not belong to the same team`)

	// create a valid sync script execution request by script name, fails because the
	// request will time-out waiting for a result.
	var runSyncResp2 runScriptSyncResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host2.ID, ScriptName: "f1337.sh"}, http.StatusRequestTimeout, &runSyncResp2)
	require.Equal(t, host2.ID, runSyncResp2.HostID)
	require.NotEmpty(t, runSyncResp2.ExecutionID)
	require.NotNil(t, runSyncResp2.ScriptID)
	require.Equal(t, savedNoTmScript2.ID, *runSyncResp2.ScriptID)
	require.Equal(t, "echo 'ALL YOUR BASE ARE BELONG TO US'", runSyncResp2.ScriptContents)
	require.True(t, runSyncResp2.HostTimeout)
	require.Contains(t, runSyncResp2.Message, fleet.RunScriptHostTimeoutErrMsg)

	// attempt to run a script on a plain osquery host
	plainOsqueryHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-time.Minute),
		OsqueryHostID:   ptr.String("plain-osquery-host-2"),
		NodeKey:         ptr.String("plain-osquery-host-2"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", "plain-osquery-host-2"),
		HardwareSerial:  uuid.New().String(),
		Platform:        "linux",
	})
	require.NoError(t, err)
	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: plainOsqueryHost.ID, ScriptID: &script.ID}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), fleet.RunScriptDisabledErrMsg)
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: plainOsqueryHost.ID, ScriptID: &script.ID}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), fleet.RunScriptDisabledErrMsg)

	// Async Run Script by Name

	// attempt to run async with a script that does not exist on the specified team
	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host2.ID, ScriptName: "f1337.sh", TeamID: tm.ID}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Script 'f1337.sh' doesn’t exist.`)

	// attempt to run async with an existing team script that belongs to a team different from the host's team
	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host2.ID, ScriptName: "f1337.sh", TeamID: tm2.ID}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `The script does not belong to the same team`)

	var runSyncResp3 runScriptSyncResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host2.ID, ScriptName: "f13372.sh"}, http.StatusAccepted, &runSyncResp3)
	require.Equal(t, host2.ID, runSyncResp3.HostID)
	require.NotEmpty(t, runSyncResp3.ExecutionID)

	// verify pending result
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+runSyncResp3.ExecutionID, nil, http.StatusOK, &scriptResultResp)
	require.Equal(t, host2.ID, scriptResultResp.HostID)
	require.Equal(t, "echo 'ALL YOUR BASE ARE BELONG TO US'", scriptResultResp.ScriptContents)
	require.Nil(t, scriptResultResp.ExitCode)
	require.False(t, scriptResultResp.HostTimeout)
	require.Contains(t, scriptResultResp.Message, fleet.RunScriptAsyncScriptEnqueuedMsg)
}

func (s *integrationEnterpriseTestSuite) TestEnqueueSameScriptTwice() {
	t := s.T()
	ctx := context.Background()

	host := createOrbitEnrolledHost(t, "linux", "", s.ds)
	script, err := s.ds.NewScript(ctx, &fleet.Script{
		TeamID:         nil,
		Name:           "script.sh",
		ScriptContents: "echo 'hi from script'",
	})
	require.NoError(t, err)

	// Make the host offline so that scripts enqueue
	err = s.ds.MarkHostsSeen(ctx, []uint{host.ID}, time.Now().Add(-time.Hour))
	require.NoError(t, err)

	var runResp runScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: &script.ID}, http.StatusAccepted, &runResp)
	require.Equal(t, host.ID, runResp.HostID)
	require.NotEmpty(t, runResp.ExecutionID)

	// Should fail because the same script is already enqueued for this host.
	resp := s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: &script.ID}, http.StatusConflict)
	errorMsg := extractServerErrorText(resp.Body)
	require.Contains(t, errorMsg, "The script is already queued on the given host")
}

func (s *integrationEnterpriseTestSuite) TestOrbitConfigExtensions() {
	t := s.T()
	ctx := context.Background()

	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	defer func() {
		err = s.ds.SaveAppConfig(ctx, appCfg)
		require.NoError(t, err)
	}()

	foobarLabel, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:  "Foobar",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)
	zoobarLabel, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:  "Zoobar",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)
	allHostsLabel, err := s.ds.GetLabelSpec(ctx, "All hosts")
	require.NoError(t, err)

	orbitDarwinClient := createOrbitEnrolledHost(t, "darwin", "foobar1", s.ds)
	orbitLinuxClient := createOrbitEnrolledHost(t, "linux", "foobar2", s.ds)
	orbitWindowsClient := createOrbitEnrolledHost(t, "windows", "foobar3", s.ds)

	// orbitDarwinClient is member of 'All hosts' and 'Zoobar' labels.
	err = s.ds.RecordLabelQueryExecutions(ctx, orbitDarwinClient, map[uint]*bool{
		allHostsLabel.ID: ptr.Bool(true),
		zoobarLabel.ID:   ptr.Bool(true),
	}, time.Now(), false)
	require.NoError(t, err)
	// orbitLinuxClient is member of 'All hosts' and 'Foobar' labels.
	err = s.ds.RecordLabelQueryExecutions(ctx, orbitLinuxClient, map[uint]*bool{
		allHostsLabel.ID: ptr.Bool(true),
		foobarLabel.ID:   ptr.Bool(true),
	}, time.Now(), false)
	require.NoError(t, err)
	// orbitWindowsClient is member of the 'All hosts' label only.
	err = s.ds.RecordLabelQueryExecutions(ctx, orbitWindowsClient, map[uint]*bool{
		allHostsLabel.ID: ptr.Bool(true),
	}, time.Now(), false)
	require.NoError(t, err)

	// Attempt to add labels to extensions.
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
  "agent_options": {
	"config": {
		"options": {
		"pack_delimiter": "/",
		"logger_tls_period": 10,
		"distributed_plugin": "tls",
		"disable_distributed": false,
		"logger_tls_endpoint": "/api/osquery/log",
		"distributed_interval": 10,
		"distributed_tls_max_attempts": 3
		}
	},
	"extensions": {
		"hello_world_linux": {
			"labels": [
				"All hosts",
				"Foobar"
			],
			"channel": "stable",
			"platform": "linux"
		},
		"hello_world_macos": {
			"labels": [
				"All hosts",
				"Foobar"
			],
			"channel": "stable",
			"platform": "macos"
		},
		"hello_mars_macos": {
			"labels": [
				"All hosts",
				"Zoobar"
			],
			"channel": "stable",
			"platform": "macos"
		},
		"hello_world_windows": {
			"labels": [
				"Zoobar"
			],
			"channel": "stable",
			"platform": "windows"
		},
		"hello_mars_windows": {
			"labels": [
				"Foobar"
			],
			"channel": "stable",
			"platform": "windows"
		}
	}
  }
}`), http.StatusOK)

	resp := orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitDarwinClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.JSONEq(t, `{
	"hello_mars_macos": {
		"channel": "stable",
		"platform": "macos"
	}
  }`, string(resp.Extensions))

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitLinuxClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.JSONEq(t, `{
	"hello_world_linux": {
		"channel": "stable",
		"platform": "linux"
	}
  }`, string(resp.Extensions))

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitWindowsClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, string(resp.Extensions))

	// orbitDarwinClient is now also a member of the 'Foobar' label.
	err = s.ds.RecordLabelQueryExecutions(ctx, orbitDarwinClient, map[uint]*bool{
		foobarLabel.ID: ptr.Bool(true),
	}, time.Now(), false)
	require.NoError(t, err)

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitDarwinClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.JSONEq(t, `{
	"hello_world_macos": {
		"channel": "stable",
		"platform": "macos"
	},
	"hello_mars_macos": {
		"channel": "stable",
		"platform": "macos"
	}
  }`, string(resp.Extensions))

	// orbitLinuxClient is no longer a member of the 'Foobar' label.
	err = s.ds.RecordLabelQueryExecutions(ctx, orbitLinuxClient, map[uint]*bool{
		foobarLabel.ID: nil,
	}, time.Now(), false)
	require.NoError(t, err)

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitLinuxClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, string(resp.Extensions))

	// Attempt to set non-existent labels in the config.
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
  "agent_options": {
	"config": {
		"options": {
		"pack_delimiter": "/",
		"logger_tls_period": 10,
		"distributed_plugin": "tls",
		"disable_distributed": false,
		"logger_tls_endpoint": "/api/osquery/log",
		"distributed_interval": 10,
		"distributed_tls_max_attempts": 3
		}
	},
	"extensions": {
		"hello_world_linux": {
			"labels": [
				"All hosts",
				"Doesn't exist"
			],
			"channel": "stable",
			"platform": "linux"
		}
	}
  }
}`), http.StatusBadRequest)
}

func (s *integrationEnterpriseTestSuite) TestSavedScripts() {
	t := s.T()
	ctx := context.Background()

	// create a saved script for no team
	var newScriptResp createScriptResponse
	body, headers := generateNewScriptMultipartRequest(t,
		"script1.sh", []byte(`echo "hello"`), s.token, nil)
	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusOK, headers)
	err := json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)
	require.NotZero(t, newScriptResp.ScriptID)
	noTeamScriptID := newScriptResp.ScriptID
	s.lastActivityMatches("added_script", fmt.Sprintf(`{"script_name": %q, "team_name": null, "team_id": null}`, "script1.sh"), 0)

	// get the script
	var getScriptResp getScriptResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/scripts/%d", noTeamScriptID), nil, http.StatusOK, &getScriptResp)
	require.Equal(t, noTeamScriptID, getScriptResp.ID)
	require.Nil(t, getScriptResp.TeamID)
	require.Equal(t, "script1.sh", getScriptResp.Name)
	require.NotZero(t, getScriptResp.CreatedAt)
	require.NotZero(t, getScriptResp.UpdatedAt)
	require.Empty(t, getScriptResp.ScriptContents)

	// download the script's content
	res = s.Do("GET", fmt.Sprintf("/api/latest/fleet/scripts/%d", noTeamScriptID), nil, http.StatusOK, "alt", "media")
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, `echo "hello"`, string(b))
	require.Equal(t, int64(len(`echo "hello"`)), res.ContentLength)
	require.Equal(t, fmt.Sprintf("attachment;filename=\"%s %s\"", time.Now().Format(time.DateOnly), "script1.sh"), res.Header.Get("Content-Disposition"))

	// get a non-existing script
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/scripts/%d", noTeamScriptID+999), nil, http.StatusNotFound, &getScriptResp)
	// download a non-existing script
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/scripts/%d", noTeamScriptID+999), nil, http.StatusNotFound, &getScriptResp, "alt", "media")

	// file name is empty
	body, headers = generateNewScriptMultipartRequest(t,
		"", []byte(`echo "hello"`), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusBadRequest, headers)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "no file headers for script")

	// contains invalid fleet secret
	body, headers = generateNewScriptMultipartRequest(t,
		"secrets.sh", []byte(`echo "$FLEET_SECRET_INVALID"`), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusUnprocessableEntity, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "$FLEET_SECRET_INVALID")

	// file name is not .sh
	body, headers = generateNewScriptMultipartRequest(t,
		"not_sh.txt", []byte(`echo "hello"`), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusUnprocessableEntity, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Validation Failed: File type not supported. Only .sh and .ps1 file type is allowed.")

	// file content is empty
	body, headers = generateNewScriptMultipartRequest(t,
		"script2.sh", []byte(``), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusUnprocessableEntity, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Script contents must not be empty")

	// file content is too large
	body, headers = generateNewScriptMultipartRequest(t,
		"script2.sh", []byte(strings.Repeat("a", fleet.SavedScriptMaxRuneLen+1)), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusUnprocessableEntity, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Script is too large. It's limited to 500,000 characters")

	// invalid hashbang
	body, headers = generateNewScriptMultipartRequest(t,
		"script2.sh", []byte(`#!/bin/python`), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusUnprocessableEntity, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Interpreter not supported.")

	// script already exists with this name for this no-team
	body, headers = generateNewScriptMultipartRequest(t,
		"script1.sh", []byte(`echo "hello"`), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusConflict, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "A script with this name already exists")

	// team id does not exist
	body, headers = generateNewScriptMultipartRequest(t,
		"script1.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {"123"}})
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusNotFound, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "The team does not exist.")

	// create a team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	// create with existing name for this time for a team
	body, headers = generateNewScriptMultipartRequest(t,
		"script1.sh", []byte(`echo "team"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)
	require.NotZero(t, newScriptResp.ScriptID)
	require.NotEqual(t, noTeamScriptID, newScriptResp.ScriptID)
	tmScriptID := newScriptResp.ScriptID
	s.lastActivityMatches("added_script", fmt.Sprintf(`{"script_name": %q, "team_name": %q, "team_id": %d}`, "script1.sh", tm.Name, tm.ID), 0)

	// create a windows script
	body, headers = generateNewScriptMultipartRequest(t,
		"script2.ps1", []byte(`Write-Host "Hello, World!"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})

	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)
	require.NotZero(t, newScriptResp.ScriptID)
	require.NotEqual(t, noTeamScriptID, newScriptResp.ScriptID)
	s.lastActivityMatches("added_script", fmt.Sprintf(`{"script_name": %q, "team_name": %q, "team_id": %d}`, "script2.ps1", tm.Name, tm.ID), 0)

	// get team's script
	getScriptResp = getScriptResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/scripts/%d", tmScriptID), nil, http.StatusOK, &getScriptResp)
	require.Equal(t, tmScriptID, getScriptResp.ID)
	require.NotNil(t, getScriptResp.TeamID)
	require.Equal(t, tm.ID, *getScriptResp.TeamID)
	require.Equal(t, "script1.sh", getScriptResp.Name)
	require.NotZero(t, getScriptResp.CreatedAt)
	require.NotZero(t, getScriptResp.UpdatedAt)
	require.Empty(t, getScriptResp.ScriptContents)

	// download the team's script's content
	res = s.Do("GET", fmt.Sprintf("/api/latest/fleet/scripts/%d", tmScriptID), nil, http.StatusOK, "alt", "media")
	b, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, `echo "team"`, string(b))
	require.Equal(t, int64(len(`echo "team"`)), res.ContentLength)
	require.Equal(t, fmt.Sprintf("attachment;filename=\"%s %s\"", time.Now().Format(time.DateOnly), "script1.sh"), res.Header.Get("Content-Disposition"))

	// script already exists with this name for this team
	body, headers = generateNewScriptMultipartRequest(t,
		"script1.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})

	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusConflict, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "A script with this name already exists")

	// create with a different name for this team
	body, headers = generateNewScriptMultipartRequest(t,
		"script2.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})

	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)
	require.NotZero(t, newScriptResp.ScriptID)
	require.NotEqual(t, noTeamScriptID, newScriptResp.ScriptID)
	require.NotEqual(t, tmScriptID, newScriptResp.ScriptID)
	s.lastActivityMatches("added_script", fmt.Sprintf(`{"script_name": %q, "team_name": %q, "team_id": %d}`, "script2.sh", tm.Name, tm.ID), 0)

	// delete the no-team script
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/scripts/%d", noTeamScriptID), nil, http.StatusNoContent)
	s.lastActivityMatches("deleted_script", fmt.Sprintf(`{"script_name": %q, "team_name": null, "team_id": null}`, "script1.sh"), 0)

	// delete the initial team script
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/scripts/%d", tmScriptID), nil, http.StatusNoContent)
	s.lastActivityMatches("deleted_script", fmt.Sprintf(`{"script_name": %q, "team_name": %q, "team_id": %d}`, "script1.sh", tm.Name, tm.ID), 0)

	// delete a non-existing script
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/scripts/%d", noTeamScriptID), nil, http.StatusNotFound)
}

func (s *integrationEnterpriseTestSuite) TestListSavedScripts() {
	t := s.T()
	ctx := context.Background()

	// create some teams
	tm1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	tm3, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team3"})
	require.NoError(t, err)

	// create 5 scripts for no team and team 1
	for i := 0; i < 5; i++ {
		_, err = s.ds.NewScript(ctx, &fleet.Script{
			Name:           string('a' + byte(i)), // i.e. "a", "b", "c", ...
			ScriptContents: "echo",
		})
		require.NoError(t, err)
		_, err = s.ds.NewScript(ctx, &fleet.Script{Name: string('a' + byte(i)), TeamID: &tm1.ID, ScriptContents: "echo"})
		require.NoError(t, err)
	}

	// create a single script for team 2
	_, err = s.ds.NewScript(ctx, &fleet.Script{Name: "a", TeamID: &tm2.ID, ScriptContents: "echo"})
	require.NoError(t, err)

	cases := []struct {
		queries   []string // alternate query name and value
		teamID    *uint
		wantNames []string
		wantMeta  *fleet.PaginationMetadata
	}{
		{
			wantNames: []string{"a", "b", "c", "d", "e"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2"},
			wantNames: []string{"a", "b"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2", "page", "1"},
			wantNames: []string{"c", "d"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "2", "page", "2"},
			wantNames: []string{"e"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "3"},
			teamID:    &tm1.ID,
			wantNames: []string{"a", "b", "c"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "3", "page", "1"},
			teamID:    &tm1.ID,
			wantNames: []string{"d", "e"},
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
			wantNames: []string{"a"},
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
			var listResp listScriptsResponse
			queryArgs := c.queries
			if c.teamID != nil {
				queryArgs = append(queryArgs, "team_id", fmt.Sprint(*c.teamID))
			}
			s.DoJSON("GET", "/api/latest/fleet/scripts", nil, http.StatusOK, &listResp, queryArgs...)

			require.Equal(t, len(c.wantNames), len(listResp.Scripts))
			require.Equal(t, c.wantMeta, listResp.Meta)

			var gotNames []string
			if len(listResp.Scripts) > 0 {
				gotNames = make([]string, len(listResp.Scripts))
				for i, s := range listResp.Scripts {
					gotNames[i] = s.Name
					if c.teamID == nil {
						require.Nil(t, s.TeamID)
					} else {
						require.NotNil(t, s.TeamID)
						require.Equal(t, *c.teamID, *s.TeamID)
					}
				}
			}
			require.Equal(t, c.wantNames, gotNames)
		})
	}
}

func (s *integrationEnterpriseTestSuite) TestHostScriptDetails() {
	t := s.T()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// create some teams
	tm1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "test-script-details-team1"})
	require.NoError(t, err)
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "test-script-details-team2"})
	require.NoError(t, err)
	tm3, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "test-script-details-team3"})
	require.NoError(t, err)
	tm4, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "test-script-details-team4-windows"})
	require.NoError(t, err)

	// create 5 scripts for no team and team 1
	for i := 0; i < 5; i++ {
		_, err = s.ds.NewScript(ctx, &fleet.Script{Name: fmt.Sprintf("test-script-details-%d.sh", i), ScriptContents: "echo"})
		require.NoError(t, err)
		_, err = s.ds.NewScript(ctx, &fleet.Script{Name: fmt.Sprintf("test-script-details-%d.sh", i), TeamID: &tm1.ID, ScriptContents: "echo"})
		require.NoError(t, err)
	}

	// add a windows script to team 4
	_, err = s.ds.NewScript(ctx, &fleet.Script{Name: "test-script-details-windows.ps1", TeamID: &tm4.ID, ScriptContents: `Write-Host "Hello, World!"`})
	require.NoError(t, err)

	// create a single script for team 2
	_, err = s.ds.NewScript(ctx, &fleet.Script{Name: "test-script-details-team-2.sh", TeamID: &tm2.ID, ScriptContents: "echo"})
	require.NoError(t, err)

	// create a host without a team
	host0, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String("host0"),
		NodeKey:         ptr.String("host0"),
		UUID:            uuid.New().String(),
		Hostname:        "host0",
		Platform:        "darwin",
	})
	require.NoError(t, err)

	// create a host for team 1
	host1, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String("host1"),
		NodeKey:         ptr.String("host1"),
		UUID:            uuid.New().String(),
		Hostname:        "host1",
		Platform:        "darwin",
		TeamID:          &tm1.ID,
	})
	require.NoError(t, err)

	// create a host for team 3
	host2, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String("host2"),
		NodeKey:         ptr.String("host2"),
		UUID:            uuid.New().String(),
		Hostname:        "host2",
		Platform:        "darwin",
		TeamID:          &tm3.ID,
	})
	require.NoError(t, err)

	// create a Windows host
	host3, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String("host3"),
		NodeKey:         ptr.String("host3"),
		UUID:            uuid.New().String(),
		Hostname:        "host3",
		Platform:        "windows",
		TeamID:          &tm4.ID,
	})
	require.NoError(t, err)

	// create a Linux host
	host4, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String("host4"),
		NodeKey:         ptr.String("host4"),
		UUID:            uuid.New().String(),
		Hostname:        "host4",
		Platform:        "ubuntu",
		TeamID:          nil,
	})
	require.NoError(t, err)

	// create a chrome host
	host5, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String("host5"),
		NodeKey:         ptr.String("host5"),
		UUID:            uuid.New().String(),
		Hostname:        "host5",
		Platform:        "chrome",
		TeamID:          nil,
	})
	require.NoError(t, err)

	insertResults := func(t *testing.T, hostID uint, script *fleet.Script, createdAt time.Time, execID string, exitCode *int64) {
		stmt := `
INSERT INTO
	host_script_results (%s host_id, created_at, execution_id, exit_code, script_content_id, output, sync_request)
VALUES
	(%s ?,?,?,?,?,?, 1)`

		args := []interface{}{}
		var scID uint
		mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
			res, err := tx.ExecContext(ctx, `
INSERT INTO
	script_contents (md5_checksum, contents, created_at)
VALUES
	(?,?,?)`,
				uuid.NewString(),
				"echo test-script-details-timeout",
				now.Add(-1*time.Hour),
			)
			if err != nil {
				return err
			}
			id, err := res.LastInsertId()
			if err != nil {
				return err
			}

			scID = uint(id) //nolint:gosec // dismiss G115
			return nil
		})
		if script.ID == 0 {
			stmt = fmt.Sprintf(stmt, "", "")
		} else {
			stmt = fmt.Sprintf(stmt, "script_id,", "?,")
			args = append(args, script.ID)
		}
		args = append(args, hostID, createdAt, execID, exitCode, scID, "")

		mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, stmt, args...)
			return err
		})
	}

	// insert some ad hoc script results, these are never included in the host script details
	insertResults(t, host0.ID, &fleet.Script{Name: "ad hoc script", ScriptContents: "echo foo"}, now, "ad-hoc-0", ptr.Int64(0))
	insertResults(t, host1.ID, &fleet.Script{Name: "ad hoc script", ScriptContents: "echo foo"}, now.Add(-1*time.Hour), "ad-hoc-1", ptr.Int64(1))

	t.Run("no team", func(t *testing.T) {
		noTeamScripts, _, err := s.ds.ListScripts(ctx, nil, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, noTeamScripts, 5)

		// insert saved script results for host0
		insertResults(t, host0.ID, noTeamScripts[0], now, "exec0-0", ptr.Int64(0))                   // expect status ran
		insertResults(t, host0.ID, noTeamScripts[1], now.Add(-1*time.Hour), "exec0-1", ptr.Int64(1)) // expect status error
		insertResults(t, host0.ID, noTeamScripts[2], now.Add(-2*time.Hour), "exec0-2", nil)          // expect status pending

		// insert some ad hoc script results, these are never included in the host script details
		insertResults(t, host0.ID, &fleet.Script{Name: "ad hoc script", ScriptContents: "echo foo"}, now.Add(-3*time.Hour), "exec0-3", ptr.Int64(0))

		// check host script details, should include all no team scripts
		var resp getHostScriptDetailsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", host0.ID), nil, http.StatusOK, &resp)
		require.Len(t, resp.Scripts, len(noTeamScripts))
		byScriptID := make(map[uint]*fleet.HostScriptDetail, len(resp.Scripts))
		for _, s := range resp.Scripts {
			byScriptID[s.ScriptID] = s
		}
		for i, s := range noTeamScripts {
			gotScript, ok := byScriptID[s.ID]
			require.True(t, ok)
			require.Equal(t, s.Name, gotScript.Name)
			switch i {
			case 0:
				require.NotNil(t, gotScript.LastExecution)
				require.Equal(t, "exec0-0", gotScript.LastExecution.ExecutionID)
				require.Equal(t, now, gotScript.LastExecution.ExecutedAt)
				require.Equal(t, "ran", gotScript.LastExecution.Status)
			case 1:
				require.NotNil(t, gotScript.LastExecution)
				require.Equal(t, "exec0-1", gotScript.LastExecution.ExecutionID)
				require.Equal(t, now.Add(-1*time.Hour), gotScript.LastExecution.ExecutedAt)
				require.Equal(t, "error", gotScript.LastExecution.Status)
			case 2:
				require.NotNil(t, gotScript.LastExecution)
				require.Equal(t, "exec0-2", gotScript.LastExecution.ExecutionID)
				require.Equal(t, now.Add(-2*time.Hour), gotScript.LastExecution.ExecutedAt)
				require.Equal(t, "pending", gotScript.LastExecution.Status)
			default:
				require.Nil(t, gotScript.LastExecution)
			}
		}
	})

	t.Run("team 1", func(t *testing.T) {
		tm1Scripts, _, err := s.ds.ListScripts(ctx, &tm1.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, tm1Scripts, 5)

		// insert results for host1
		insertResults(t, host1.ID, tm1Scripts[0], now, "exec1-0", ptr.Int64(0)) // expect status ran

		// check host script details, should match team 1
		var resp getHostScriptDetailsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", host1.ID), nil, http.StatusOK, &resp)
		require.Len(t, resp.Scripts, len(tm1Scripts))
		byScriptID := make(map[uint]*fleet.HostScriptDetail, len(resp.Scripts))
		for _, s := range resp.Scripts {
			byScriptID[s.ScriptID] = s
		}
		for i, s := range tm1Scripts {
			gotScript, ok := byScriptID[s.ID]
			require.True(t, ok)
			require.Equal(t, s.Name, gotScript.Name)
			switch i {
			case 0:
				require.NotNil(t, gotScript.LastExecution)
				require.Equal(t, "exec1-0", gotScript.LastExecution.ExecutionID)
				require.Equal(t, now, gotScript.LastExecution.ExecutedAt)
				require.Equal(t, "ran", gotScript.LastExecution.Status)
			default:
				require.Nil(t, gotScript.LastExecution)
			}
		}
	})

	t.Run("deleted script", func(t *testing.T) {
		noTeamScripts, _, err := s.ds.ListScripts(ctx, nil, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, noTeamScripts, 5)

		// delete a script
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/scripts/%d", noTeamScripts[0].ID), nil, http.StatusNoContent)

		// check host script details, should not include deleted script
		var resp getHostScriptDetailsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", host0.ID), nil, http.StatusOK, &resp)
		require.Len(t, resp.Scripts, len(noTeamScripts)-1)
		byScriptID := make(map[uint]*fleet.HostScriptDetail, len(resp.Scripts))
		for _, s := range resp.Scripts {
			require.NotEqual(t, noTeamScripts[0].ID, s.ScriptID)
			byScriptID[s.ScriptID] = s
		}
		for i, s := range noTeamScripts {
			gotScript, ok := byScriptID[s.ID]
			if i == 0 {
				require.False(t, ok)
			} else {
				require.True(t, ok)
				require.Equal(t, s.Name, gotScript.Name)
				switch i {
				case 1:
					require.NotNil(t, gotScript.LastExecution)
					require.Equal(t, "exec0-1", gotScript.LastExecution.ExecutionID)
					require.Equal(t, now.Add(-1*time.Hour), gotScript.LastExecution.ExecutedAt)
					require.Equal(t, "error", gotScript.LastExecution.Status)
				case 2:
					require.NotNil(t, gotScript.LastExecution)
					require.Equal(t, "exec0-2", gotScript.LastExecution.ExecutionID)
					require.Equal(t, now.Add(-2*time.Hour), gotScript.LastExecution.ExecutedAt)
					require.Equal(t, "pending", gotScript.LastExecution.Status)
				case 3, 4:
					require.Nil(t, gotScript.LastExecution)
				default:
					require.Fail(t, "unexpected script")
				}
			}
		}
	})

	t.Run("transfer team", func(t *testing.T) {
		s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
			TeamID:  &tm2.ID,
			HostIDs: []uint{host1.ID},
		}, http.StatusOK, &addHostsToTeamResponse{})

		tm2Scripts, _, err := s.ds.ListScripts(ctx, &tm2.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, tm2Scripts, 1)

		// check host script details, should not include prior team's scripts
		var resp getHostScriptDetailsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", host1.ID), nil, http.StatusOK, &resp)
		require.Len(t, resp.Scripts, len(tm2Scripts))
		byScriptID := make(map[uint]*fleet.HostScriptDetail, len(resp.Scripts))
		for _, s := range resp.Scripts {
			byScriptID[s.ScriptID] = s
		}
		for _, s := range tm2Scripts {
			gotScript, ok := byScriptID[s.ID]
			require.True(t, ok)
			require.Equal(t, s.Name, gotScript.Name)
			require.Nil(t, gotScript.LastExecution)
		}
	})

	t.Run("no scripts", func(t *testing.T) {
		var resp getHostScriptDetailsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", host2.ID), nil, http.StatusOK, &resp)
		require.NotNil(t, resp.Scripts)
		require.Len(t, resp.Scripts, 0)
	})

	t.Run("windows", func(t *testing.T) {
		team4Scripts, _, err := s.ds.ListScripts(ctx, &tm4.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, team4Scripts, 1)

		var resp getHostScriptDetailsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", host3.ID), nil, http.StatusOK, &resp)
		require.NotNil(t, resp.Scripts)
		require.Len(t, resp.Scripts, 1)
	})

	t.Run("linux", func(t *testing.T) {
		require.Nil(t, host4.TeamID)
		noTeamScripts, _, err := s.ds.ListScripts(ctx, nil, fleet.ListOptions{})
		require.NoError(t, err)
		require.True(t, len(noTeamScripts) > 0)

		var resp getHostScriptDetailsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", host4.ID), nil, http.StatusOK, &resp)
		require.NotNil(t, resp.Scripts)
		require.Len(t, resp.Scripts, 4)

		for _, s := range resp.Scripts {
			require.Nil(t, s.LastExecution)
			require.Contains(t, s.Name, ".sh")
		}
	})

	// NOTE: Scripts are specified only for platforms other than macOS, Linux,
	// and Windows; however, we default to listing all scripts for unspecified platforms.
	// Separately, the UI restricts scripts related functionality to only macOS,
	// Linux, and Windows.
	t.Run("unspecified platform", func(t *testing.T) {
		var resp getHostScriptDetailsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", host5.ID), nil, http.StatusOK, &resp)
		require.NotNil(t, resp.Scripts)
		require.Len(t, resp.Scripts, 4)
	})

	t.Run("get script results user message", func(t *testing.T) {
		// add a script with an older created_at timestamp
		var oldScriptID, oldScriptContentsID uint
		mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
			// create script_contents first
			res, err := tx.ExecContext(ctx, `
INSERT INTO
	script_contents (md5_checksum, contents, created_at)
VALUES
	(?,?,?)`,

				uuid.NewString(),
				"echo test-script-details-timeout",
				now.Add(-1*time.Hour),
			)
			if err != nil {
				return err
			}
			id, err := res.LastInsertId()
			if err != nil {
				return err
			}

			oldScriptContentsID = uint(id) //nolint:gosec // dismiss G115

			res, err = tx.ExecContext(ctx, `
INSERT INTO
	scripts (name, script_content_id, created_at, updated_at)
VALUES
	(?,?,?,?)`,
				"test-script-details-timeout.sh",
				oldScriptContentsID,
				now.Add(-1*time.Hour),
				now.Add(-1*time.Hour),
			)
			if err != nil {
				return err
			}
			id, err = res.LastInsertId()
			if err != nil {
				return err
			}
			oldScriptID = uint(id) //nolint:gosec // dismiss G115
			return nil
		})

		for _, c := range []struct {
			name       string
			exitCode   *int64
			executedAt time.Time
			expected   string
		}{
			{
				name:       "host-timeout",
				exitCode:   nil,
				executedAt: now.Add(-1 * time.Hour),
				expected:   fleet.RunScriptHostTimeoutErrMsg,
			},
			{
				name:       "script-timeout",
				exitCode:   ptr.Int64(-1),
				executedAt: now.Add(-1 * time.Hour),
				expected:   fleet.HostScriptTimeoutMessage(ptr.Int(int(scripts.MaxHostExecutionTime.Seconds()))),
			},
			{
				name:       "pending",
				exitCode:   nil,
				executedAt: now.Add(-1 * time.Minute),
				expected:   fleet.RunScriptAlreadyRunningErrMsg,
			},
			{
				name:       "success",
				exitCode:   ptr.Int64(0),
				executedAt: now.Add(-1 * time.Hour),
				expected:   "",
			},
			{
				name:       "error",
				exitCode:   ptr.Int64(1),
				executedAt: now.Add(-1 * time.Hour),
				expected:   "",
			},
			{
				name:       "disabled",
				exitCode:   ptr.Int64(-2),
				executedAt: now.Add(-1 * time.Hour),
				expected:   fleet.RunScriptDisabledErrMsg,
			},
		} {
			t.Run(c.name, func(t *testing.T) {
				insertResults(t, host0.ID, &fleet.Script{ID: oldScriptID, ScriptContentID: oldScriptContentsID, Name: "test-script-details-timeout.sh"}, c.executedAt, "test-user-message_"+c.name, c.exitCode)

				var resp getScriptResultResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/scripts/results/%s", "test-user-message_"+c.name), nil, http.StatusOK, &resp)
				require.Equal(t, c.expected, resp.Message)
			})
		}
	})
}

// generates the body and headers part of a multipart request ready to be
// used via s.DoRawWithHeaders to POST /api/_version_/fleet/scripts.
func generateNewScriptMultipartRequest(t *testing.T,
	fileName string, fileContent []byte, token string, extraFields map[string][]string,
) (*bytes.Buffer, map[string]string) {
	return generateMultipartRequest(t, "script", fileName, fileContent, token, extraFields)
}

func (s *integrationEnterpriseTestSuite) TestAppConfigScripts() {
	t := s.T()

	// set the script fields
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "scripts": ["foo", "bar"] }`), http.StatusOK, &acResp)
	assert.ElementsMatch(t, []string{"foo", "bar"}, acResp.Scripts.Value)

	// check that they are returned by a GET /config
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.ElementsMatch(t, []string{"foo", "bar"}, acResp.Scripts.Value)

	// patch without specifying the scripts fields, should not remove them
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{}`), http.StatusOK, &acResp)
	assert.ElementsMatch(t, []string{"foo", "bar"}, acResp.Scripts.Value)

	// patch with explicitly empty scripts fields, would remove
	// them but this is a dry-run
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "scripts": null }`), http.StatusOK, &acResp, "dry_run", "true")
	assert.ElementsMatch(t, []string{"foo", "bar"}, acResp.Scripts.Value)

	// patch with explicitly empty scripts fields, removes them
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "scripts": null }`), http.StatusOK, &acResp)
	assert.Empty(t, acResp.Scripts.Value)

	// set the script fields again
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "scripts": ["foo", "bar"] }`), http.StatusOK, &acResp)
	assert.ElementsMatch(t, []string{"foo", "bar"}, acResp.Scripts.Value)

	// patch with an empty array sets the scripts to an empty array as well
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "scripts": [] }`), http.StatusOK, &acResp)
	assert.Empty(t, acResp.Scripts.Value)

	// patch with an invalid array returns an error
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "scripts": ["foo", 1] }`), http.StatusBadRequest, &acResp)
	assert.Empty(t, acResp.Scripts.Value)
}

func (s *integrationEnterpriseTestSuite) TestApplyTeamsScriptsConfig() {
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

	// apply with scripts
	// must not use applyTeamSpecsRequest and marshal it as JSON, as it will set
	// all keys to their zerovalue, and some are only valid with mdm enabled.
	teamSpecs := map[string]any{
		"specs": []any{
			map[string]any{
				"name":    teamName,
				"scripts": []string{"foo", "bar"},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// retrieving the team returns the scripts
	var teamResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []string{"foo", "bar"}, teamResp.Team.Config.Scripts.Value)

	// apply without custom scripts specified, should not replace existing scripts
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []string{"foo", "bar"}, teamResp.Team.Config.Scripts.Value)

	// apply with explicitly empty custom scripts would clear the existing
	// scripts, but dry-run
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":    teamName,
				"scripts": nil,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "dry_run", "true")
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []string{"foo", "bar"}, teamResp.Team.Config.Scripts.Value)

	// apply with explicitly empty scripts clears the existing scripts
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":    teamName,
				"scripts": nil,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Empty(t, teamResp.Team.Config.Scripts.Value)

	// patch with an invalid array returns an error
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":    teamName,
				"scripts": []any{"foo", 1},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusBadRequest)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Empty(t, teamResp.Team.Config.Scripts.Value)
}

func (s *integrationEnterpriseTestSuite) TestBatchApplyScriptsEndpoints() {
	t := s.T()
	ctx := context.Background()

	saveAndCheckScripts := func(team *fleet.Team, scripts []fleet.ScriptPayload) {
		var teamID *uint
		teamIDStr := ""
		teamActivity := `{"team_id": null, "team_name": null}`
		if team != nil {
			teamID = &team.ID
			teamIDStr = fmt.Sprint(team.ID)
			teamActivity = fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, team.ID, team.Name)
		}

		// create, check activities, and check scripts response
		var scriptsBatchResponse batchSetScriptsResponse
		s.DoJSON("POST", "/api/v1/fleet/scripts/batch", batchSetScriptsRequest{Scripts: scripts}, http.StatusOK, &scriptsBatchResponse, "team_id", teamIDStr)
		s.lastActivityMatches(
			fleet.ActivityTypeEditedScript{}.ActivityName(),
			teamActivity,
			0,
		)
		require.Len(t, scriptsBatchResponse.Scripts, len(scripts))

		// check that the right values got stored in the db
		var listResp listScriptsResponse
		s.DoJSON("GET", "/api/latest/fleet/scripts", nil, http.StatusOK, &listResp, "team_id", teamIDStr)
		require.Len(t, listResp.Scripts, len(scripts))

		got := make([]fleet.ScriptPayload, len(scripts))
		for i, gotScript := range listResp.Scripts {
			// add the script contents
			res := s.Do("GET", fmt.Sprintf("/api/latest/fleet/scripts/%d", gotScript.ID), nil, http.StatusOK, "alt", "media")
			b, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			got[i] = fleet.ScriptPayload{
				Name:           gotScript.Name,
				ScriptContents: b,
			}
			// check that it belongs to the right team
			require.Equal(t, teamID, gotScript.TeamID)
		}

		require.ElementsMatch(t, scripts, got)
	}

	// create a new team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "batch_set_scripts"})
	require.NoError(t, err)

	// apply an empty set to no-team
	saveAndCheckScripts(nil, nil)

	// apply to both team id and name
	s.Do("POST", "/api/v1/fleet/scripts/batch", batchSetScriptsRequest{Scripts: nil},
		http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID), "team_name", tm.Name)

	// invalid team name
	s.Do("POST", "/api/v1/fleet/scripts/batch", batchSetScriptsRequest{Scripts: nil},
		http.StatusNotFound, "team_name", uuid.New().String())

	// duplicate script names
	s.Do("POST", "/api/v1/fleet/scripts/batch", batchSetScriptsRequest{Scripts: []fleet.ScriptPayload{
		{Name: "N1.sh", ScriptContents: []byte("foo")},
		{Name: "N1.sh", ScriptContents: []byte("bar")},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))

	// invalid script name
	s.Do("POST", "/api/v1/fleet/scripts/batch", batchSetScriptsRequest{Scripts: []fleet.ScriptPayload{
		{Name: "N1", ScriptContents: []byte("foo")},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))

	// empty script name
	s.Do("POST", "/api/v1/fleet/scripts/batch", batchSetScriptsRequest{Scripts: []fleet.ScriptPayload{
		{Name: "", ScriptContents: []byte("foo")},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))

	// invalid secret
	s.Do("POST", "/api/v1/fleet/scripts/batch", batchSetScriptsRequest{Scripts: []fleet.ScriptPayload{
		{Name: "N2.sh", ScriptContents: []byte("echo $FLEET_SECRET_INVALID")},
	}}, http.StatusUnprocessableEntity, "team_id", fmt.Sprint(tm.ID))

	// successfully apply a scripts for the team
	saveAndCheckScripts(tm, []fleet.ScriptPayload{
		{Name: "N1.sh", ScriptContents: []byte("foo")},
		{Name: "N2.sh", ScriptContents: []byte("bar")},
	})

	// successfully apply scripts for "no team"
	saveAndCheckScripts(nil, []fleet.ScriptPayload{
		{Name: "N1.sh", ScriptContents: []byte("foo")},
		{Name: "N2.sh", ScriptContents: []byte("bar")},
	})

	// edit, delete and add a new one for "no team"
	saveAndCheckScripts(nil, []fleet.ScriptPayload{
		{Name: "N2.sh", ScriptContents: []byte("bar-edited")},
		{Name: "N3.sh", ScriptContents: []byte("baz")},
	})

	// edit, delete and add a new one for the team
	saveAndCheckScripts(tm, []fleet.ScriptPayload{
		{Name: "N2.sh", ScriptContents: []byte("bar-edited")},
		{Name: "N3.sh", ScriptContents: []byte("baz")},
	})

	// remove all scripts for a team
	saveAndCheckScripts(tm, nil)

	// remove all scripts for "no team"
	saveAndCheckScripts(nil, nil)
}

func (s *integrationEnterpriseTestSuite) TestTeamConfigDetailQueriesOverrides() {
	ctx := context.Background()
	t := s.T()

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
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "linux",
	})
	require.NoError(t, err)

	// add the host to team1
	err = s.ds.AddHostsToTeam(context.Background(), &team.ID, []uint{linuxHost.ID})
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

func (s *integrationEnterpriseTestSuite) TestAllSoftwareTitles() {
	ctx := context.Background()
	t := s.T()

	softwareTitleListResultsMatch := func(want, got []fleet.SoftwareTitleListResult) {
		// compare only the fields we care about
		for i := range got {
			require.NotZero(t, got[i].ID)
			got[i].ID = 0

			for j := range got[i].Versions {
				require.NotZero(t, got[i].Versions[j].ID)
				got[i].Versions[j].ID = 0
			}
			// Sort versions by version
			sort.Slice(
				got[i].Versions, func(a, b int) bool {
					return got[i].Versions[a].Version < got[i].Versions[b].Version
				},
			)
		}

		// sort and use EqualValues instead of ElementsMatch in order
		// to do a deep comparison of nested structures
		sort.Slice(got, func(i, j int) bool {
			return got[i].Name < got[j].Name
		})
		sort.Slice(want, func(i, j int) bool {
			return want[i].Name < want[j].Name
		})
		for _, v := range got {
			sort.Slice(v.Versions, func(i, j int) bool {
				return v.Versions[i].Version < v.Versions[j].Version
			})
		}
		for _, v := range want {
			sort.Slice(v.Versions, func(i, j int) bool {
				return v.Versions[i].Version < v.Versions[j].Version
			})
		}

		require.EqualValues(t, want, got)
	}

	softwareTitlesMatch := func(want, got []fleet.SoftwareTitle) {
		// compare only the fields we care about
		for i := range got {
			require.NotZero(t, got[i].ID)
			got[i].CountsUpdatedAt = nil
			got[i].ID = 0

			for j := range got[i].Versions {
				require.NotZero(t, got[i].Versions[j].ID)
				got[i].Versions[j].ID = 0
			}
			// Sort versions by version
			sort.Slice(
				got[i].Versions, func(a, b int) bool {
					return got[i].Versions[a].Version < got[i].Versions[b].Version
				},
			)
		}

		// sort and use EqualValues instead of ElementsMatch in order
		// to do a deep comparison of nested structures
		sort.Slice(got, func(i, j int) bool {
			return got[i].Name < got[j].Name
		})
		sort.Slice(want, func(i, j int) bool {
			return want[i].Name < want[j].Name
		})

		require.EqualValues(t, want, got)
	}

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	tmHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + "tm"),
		NodeKey:         ptr.String(t.Name() + "tm"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()+"tm"),
		Platform:        "linux",
	})
	require.NoError(t, err)

	// create a couple of teams and add tmHost to one
	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team2"})
	require.NoError(t, err)
	require.NoError(t, s.ds.AddHostsToTeam(ctx, &team1.ID, []uint{tmHost.ID}))

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "homebrew"},
		{Name: "foo", Version: "0.0.3", Source: "homebrew"},
		{Name: "bar", Version: "0.0.4", Source: "apps"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host, false))

	soft1 := host.Software[0]
	for _, item := range host.Software {
		if item.Name == "bar" {
			soft1 = item
			break
		}
	}

	cpes := []fleet.SoftwareCPE{{SoftwareID: soft1.ID, CPE: "somecpe"}}
	_, err = s.ds.UpsertSoftwareCPEs(context.Background(), cpes)
	require.NoError(t, err)

	// Reload software so that 'GeneratedCPEID is set.
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host, false))
	soft1 = host.Software[0]
	for _, item := range host.Software {
		if item.Name == "bar" {
			soft1 = item
			break
		}
	}

	inserted, err := s.ds.InsertSoftwareVulnerability(
		context.Background(), fleet.SoftwareVulnerability{
			SoftwareID: soft1.ID,
			CVE:        "cve-123-123-132",
		}, fleet.NVDSource,
	)
	require.NoError(t, err)
	require.True(t, inserted)

	err = s.ds.InsertCVEMeta(context.Background(), []fleet.CVEMeta{
		{
			CVE:              "cve-123-123-132",
			CVSSScore:        ptr.Float64(7.8),
			CISAKnownExploit: ptr.Bool(true),
		},
	})
	require.NoError(t, err)

	// calculate hosts counts
	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, hostsCountTs))
	require.NoError(t, s.ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))

	var resp listSoftwareTitlesResponse
	// no self-service software yet
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp, "self_service", "1")
	require.Empty(t, resp.SoftwareTitles)
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "foo",
			Source:        "homebrew",
			VersionsCount: 2,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.1", Vulnerabilities: nil},
				{Version: "0.0.3", Vulnerabilities: nil},
			},
		},
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
	}, resp.SoftwareTitles)

	// per_page equals 1, so we get only one item, but the total count is
	// still 2
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"per_page", "1",
		"order_key", "name",
		"order_direction", "desc",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "foo",
			Source:        "homebrew",
			VersionsCount: 2,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.1", Vulnerabilities: nil},
				{Version: "0.0.3", Vulnerabilities: nil},
			},
		},
	}, resp.SoftwareTitles)

	// get the second item
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"per_page", "1",
		"page", "1",
		"order_key", "name",
		"order_direction", "desc",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
	}, resp.SoftwareTitles)

	// asking for a non-existent page returns an empty list
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"per_page", "1",
		"page", "4",
		"order_key", "name",
		"order_direction", "desc",
	)
	require.Equal(t, 2, resp.Count)
	require.Empty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{}, resp.SoftwareTitles)

	// asking for vulnerable only software returns the expected values
	expectedVulnSoftware := []fleet.SoftwareTitleListResult{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
	}

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"vulnerable", "true",
	)
	require.Equal(t, 1, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch(expectedVulnSoftware, resp.SoftwareTitles)

	// vulnerable param required when using vulnerability filters
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusUnprocessableEntity, &resp,
		"exploit", "true",
	)
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusUnprocessableEntity, &resp,
		"min_cvss_score", "1",
	)
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusUnprocessableEntity, &resp,
		"max_cvss_score", "10",
	)

	// vulnerability filters
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"exploit", "true",
		"vulnerable", "true",
	)
	require.Equal(t, 1, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch(expectedVulnSoftware, resp.SoftwareTitles)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"min_cvss_score", "1",
		"vulnerable", "true",
	)
	require.Equal(t, 1, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch(expectedVulnSoftware, resp.SoftwareTitles)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"min_cvss_score", "10",
		"vulnerable", "true",
	)
	require.Zero(t, resp.Count)
	require.Nil(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{}, resp.SoftwareTitles)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"max_cvss_score", "10",
		"vulnerable", "true",
	)
	require.Equal(t, 1, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch(expectedVulnSoftware, resp.SoftwareTitles)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"max_cvss_score", "1",
		"vulnerable", "true",
	)
	require.Zero(t, resp.Count)
	require.Nil(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{}, resp.SoftwareTitles)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"min_cvss_score", "1",
		"max_cvss_score", "10",
		"vulnerable", "true",
	)
	require.Equal(t, 1, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch(expectedVulnSoftware, resp.SoftwareTitles)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"min_cvss_score", "1",
		"max_cvss_score", "10",
		"exploit", "true",
		"vulnerable", "true",
	)
	require.Equal(t, 1, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch(expectedVulnSoftware, resp.SoftwareTitles)

	// request titles for team1, nothing there yet
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	require.Equal(t, 0, resp.Count)
	require.Empty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{}, resp.SoftwareTitles)

	// add new software for tmHost
	software = []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "homebrew"},
		{Name: "baz", Version: "0.0.5", Source: "deb_packages"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), tmHost.ID, software)
	require.NoError(t, err)

	// calculate hosts counts
	hostsCountTs = time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(context.Background(), hostsCountTs))
	require.NoError(t, s.ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))

	// request software for the team, this time we get results
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team1.ID),
		"order_key", "name",
		"order_direction", "desc",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "baz",
			Source:        "deb_packages",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.5", Vulnerabilities: nil},
			},
		},
		{
			Name:          "foo",
			Source:        "homebrew",
			VersionsCount: 1, // NOTE: this value is 1 because the team has only 1 matching host in the team
			HostsCount:    1, // NOTE: this value is 1 because the team has only 1 matching host in the team
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.1", Vulnerabilities: nil}, // NOTE: this only includes versions present in the team
			},
		},
	}, resp.SoftwareTitles)

	// request software for no-team, we get all results and 2 hosts for
	// `"foo"`
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"order_key", "name",
		"order_direction", "desc",
	)
	require.Equal(t, 3, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "baz",
			Source:        "deb_packages",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.5", Vulnerabilities: nil},
			},
		},
		{
			Name:          "foo",
			Source:        "homebrew",
			VersionsCount: 2, // NOTE: this value is 2, important because no team filter was applied
			HostsCount:    2, // NOTE: this value is 2, important because no team filter was applied
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.1", Vulnerabilities: nil},
				{Version: "0.0.3", Vulnerabilities: nil},
			},
		},
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
	}, resp.SoftwareTitles)

	// match cve by name
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "123",
	)
	require.Equal(t, 1, resp.Count)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
	}, resp.SoftwareTitles)

	// match software title by name
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "ba",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
		{
			Name:          "baz",
			Source:        "deb_packages",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.5", Vulnerabilities: nil},
			},
		},
	}, resp.SoftwareTitles)

	// match software title by name with leading whitespace
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "  ba",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
		{
			Name:          "baz",
			Source:        "deb_packages",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.5", Vulnerabilities: nil},
			},
		},
	}, resp.SoftwareTitles)

	// match software title by name with trailing whitespace
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "ba  ",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
		{
			Name:          "baz",
			Source:        "deb_packages",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.5", Vulnerabilities: nil},
			},
		},
	}, resp.SoftwareTitles)

	// match software title by name with leading and trailing whitespace
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "  ba  ",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
		{
			Name:          "baz",
			Source:        "deb_packages",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.5", Vulnerabilities: nil},
			},
		},
	}, resp.SoftwareTitles)

	// find the ID of "foo"
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "foo",
	)
	require.Equal(t, 1, resp.Count)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	fooTitle := resp.SoftwareTitles[0]
	require.Equal(t, "foo", fooTitle.Name)

	// find the ID of "baz" (team 1)
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "baz",
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	require.Equal(t, 1, resp.Count)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	bazTitle := resp.SoftwareTitles[0]
	require.Equal(t, "baz", bazTitle.Name)

	// non-existent id is a 404
	var stResp getSoftwareTitleResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles/999", getSoftwareTitleRequest{}, http.StatusNotFound, &stResp)

	// valid title
	stResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", fooTitle.ID), getSoftwareTitleRequest{}, http.StatusOK, &stResp)
	s.NotZero(*stResp.SoftwareTitle.CountsUpdatedAt)
	softwareTitlesMatch([]fleet.SoftwareTitle{
		{
			Name:          "foo",
			Source:        "homebrew",
			VersionsCount: 2,
			HostsCount:    2,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.1", Vulnerabilities: nil, HostsCount: ptr.Uint(2)},
				{Version: "0.0.3", Vulnerabilities: nil, HostsCount: ptr.Uint(1)},
			},
		},
	}, []fleet.SoftwareTitle{*stResp.SoftwareTitle})

	// valid title for team
	stResp = getSoftwareTitleResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", fooTitle.ID), getSoftwareTitleRequest{}, http.StatusOK, &stResp,
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	softwareTitlesMatch(
		[]fleet.SoftwareTitle{
			{
				Name:          "foo",
				Source:        "homebrew",
				VersionsCount: 1,
				HostsCount:    1,
				Versions: []fleet.SoftwareVersion{
					{Version: "0.0.1", Vulnerabilities: nil, HostsCount: ptr.Uint(1)},
				},
			},
		}, []fleet.SoftwareTitle{*stResp.SoftwareTitle},
	)

	// find the ID of "bar"
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "bar",
	)
	require.Equal(t, 1, resp.Count)
	require.Len(t, resp.SoftwareTitles, 1)
	barTitle := resp.SoftwareTitles[0]
	require.Equal(t, "bar", barTitle.Name)

	// valid title with vulnerabilities
	expected := []fleet.SoftwareTitle{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{
					Version:         "0.0.4",
					Vulnerabilities: &fleet.SliceString{"cve-123-123-132"},
					HostsCount:      ptr.Uint(1),
				},
			},
		},
	}
	// Global
	stResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", barTitle.ID), getSoftwareTitleRequest{}, http.StatusOK, &stResp)
	softwareTitlesMatch(expected, []fleet.SoftwareTitle{*stResp.SoftwareTitle})

	// No Team
	stResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", barTitle.ID), getSoftwareTitleRequest{}, http.StatusOK, &stResp,
		"team_id", "0")
	softwareTitlesMatch(expected, []fleet.SoftwareTitle{*stResp.SoftwareTitle})

	// invalid title for team
	stResp = getSoftwareTitleResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", barTitle.ID), getSoftwareTitleRequest{}, http.StatusNotFound, &stResp,
		"team_id", fmt.Sprintf("%d", team1.ID),
	)

	// invalid title for no team
	stResp = getSoftwareTitleResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", bazTitle.ID), getSoftwareTitleRequest{}, http.StatusNotFound, &stResp,
		"team_id", "0")

	// add bar tmHost
	software = []fleet.Software{
		{Name: "bar", Version: "0.0.4", Source: "apps"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), tmHost.ID, software)
	require.NoError(t, err)

	// calculate hosts counts
	hostsCountTs = time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(context.Background(), hostsCountTs))
	require.NoError(t, s.ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))

	// valid title with vulnerabilities
	stResp = getSoftwareTitleResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", barTitle.ID), getSoftwareTitleRequest{}, http.StatusOK, &stResp,
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	softwareTitlesMatch(
		[]fleet.SoftwareTitle{
			{
				Name:          "bar",
				Source:        "apps",
				VersionsCount: 1,
				HostsCount:    1,
				Versions: []fleet.SoftwareVersion{
					{
						Version:         "0.0.4",
						Vulnerabilities: &fleet.SliceString{"cve-123-123-132"},
						HostsCount:      ptr.Uint(1),
					},
				},
			},
		}, []fleet.SoftwareTitle{*stResp.SoftwareTitle},
	)

	// Team without hosts
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", barTitle.ID), getSoftwareTitleRequest{}, http.StatusNotFound, &stResp,
		"team_id", fmt.Sprintf("%d", team2.ID),
	)

	// Non-existent team
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", barTitle.ID), getSoftwareTitleRequest{}, http.StatusNotFound, &stResp,
		"team_id", "99999",
	)

	// verify that software installers contain SoftwarePackage field
	payloadRubyTm1 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "ruby.deb",
		SelfService:   false,
		TeamID:        &team1.ID,
	}
	s.uploadSoftwareInstaller(t, payloadRubyTm1, http.StatusOK, "")

	payloadEmacsMissingSecret := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "install $FLEET_SECRET_INVALID",
		Filename:          "emacs.deb",
		PostInstallScript: "d",
		SelfService:       true,
	}
	s.uploadSoftwareInstaller(t, payloadEmacsMissingSecret, http.StatusUnprocessableEntity, "$FLEET_SECRET_INVALID")

	payloadEmacsMissingPostSecret := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "install",
		Filename:          "emacs.deb",
		PostInstallScript: "d $FLEET_SECRET_INVALID",
		SelfService:       true,
	}
	s.uploadSoftwareInstaller(t, payloadEmacsMissingPostSecret, http.StatusUnprocessableEntity, "$FLEET_SECRET_INVALID")

	payloadEmacsMissingUnSecret := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "install",
		Filename:          "emacs.deb",
		PostInstallScript: "d",
		UninstallScript:   "delet $FLEET_SECRET_INVALID",
		SelfService:       true,
	}
	s.uploadSoftwareInstaller(t, payloadEmacsMissingUnSecret, http.StatusUnprocessableEntity, "$FLEET_SECRET_INVALID")

	payloadEmacs := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "emacs.deb",
		SelfService:   true,
	}
	s.uploadSoftwareInstaller(t, payloadEmacs, http.StatusOK, "")

	payloadVim := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "vim.deb",
		SelfService:   true,
		TeamID:        ptr.Uint(0),
	}
	s.uploadSoftwareInstaller(t, payloadVim, http.StatusOK, "")

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "ruby",
		"team_id", fmt.Sprintf("%d", team1.ID),
	)

	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)

	// Upload an installer for the same software but different arch to a different team
	payloadRubyTm2 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "ruby_arm64.deb",
		TeamID:        &team2.ID,
	}
	s.uploadSoftwareInstaller(t, payloadRubyTm2, http.StatusOK, "")

	// We should only see the one we uploaded to team 1
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "ruby",
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)

	// software installer not returned with self-service only (not marked as such)
	resp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp,
		"self_service", "1", "query", "ruby", "team_id", fmt.Sprint(team1.ID))
	require.Len(t, resp.SoftwareTitles, 0)

	// update it to be self-service, check that it gets returned
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "UPDATE software_installers SET self_service = 1 WHERE filename = ?", payloadRubyTm1.Filename)
		return err
	})
	resp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp,
		"self_service", "1", "query", "ruby", "team_id", fmt.Sprint(team1.ID))
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage.SelfService)
	require.True(t, *resp.SoftwareTitles[0].SoftwarePackage.SelfService)

	// "All teams" returns no software because the self-service software it's not installed (host_counts == 0).
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"self_service", "true",
	)

	require.Empty(t, resp.SoftwareTitles, 0)

	// "No team" returns the emacs software
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"self_service", "true",
		"team_id", "0",
	)

	require.Len(t, resp.SoftwareTitles, 2)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage.SelfService)
	require.True(t, *resp.SoftwareTitles[0].SoftwarePackage.SelfService)
	require.NotNil(t, resp.SoftwareTitles[1].SoftwarePackage)
	require.NotNil(t, resp.SoftwareTitles[1].SoftwarePackage.SelfService)
	require.True(t, *resp.SoftwareTitles[1].SoftwarePackage.SelfService)

	emacsPath := fmt.Sprintf("/api/latest/fleet/software/titles/%d", resp.SoftwareTitles[0].ID)
	respTitle := getSoftwareTitleResponse{}
	s.DoJSON("GET", emacsPath, listSoftwareTitlesRequest{}, http.StatusOK, &respTitle)

	require.NotNil(t, respTitle.SoftwareTitle)
	require.Equal(t, "emacs.deb", respTitle.SoftwareTitle.SoftwarePackage.Name)
	require.True(t, respTitle.SoftwareTitle.SoftwarePackage.SelfService)
}

func (s *integrationEnterpriseTestSuite) TestLockUnlockWipeWindowsLinux() {
	ctx := context.Background()
	t := s.T()

	// create a Windows and a Linux hosts
	winHost := createOrbitEnrolledHost(t, "windows", "win_lock_unlock", s.ds)
	linuxHost := createOrbitEnrolledHost(t, "linux", "linux_lock_unlock", s.ds)

	// get the host's information
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", winHost.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", linuxHost.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

	// try to lock/unlock/wipe the Windows host, fails because Windows MDM must be enabled
	res := s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", winHost.ID), nil, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Windows MDM isn't turned on.")
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", winHost.ID), nil, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Windows MDM isn't turned on.")
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", winHost.ID), nil, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Windows MDM isn't turned on.")

	// Disable scripts on Linux host
	err := s.ds.SetOrUpdateHostOrbitInfo(
		context.Background(), linuxHost.ID, "1.22.0", sql.NullString{}, sql.NullBool{Bool: false, Valid: true},
	)
	require.NoError(t, err)
	// try to lock/unlock/wipe the Linux host. Fails because scripts are not enabled.
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", linuxHost.ID), nil, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't lock host. To lock, deploy the fleetd agent with --enable-scripts")
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", linuxHost.ID), nil, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't unlock host. To unlock, deploy the fleetd agent with --enable-scripts")
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", linuxHost.ID), nil, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't wipe host. To wipe, deploy the fleetd agent with --enable-scripts")

	// Enable scripts on Linux host
	err = s.ds.SetOrUpdateHostOrbitInfo(
		context.Background(), linuxHost.ID, "1.22.0", sql.NullString{}, sql.NullBool{Bool: true, Valid: true},
	)
	require.NoError(t, err)

	// try to lock/unlock/wipe the Linux host succeeds, no MDM constraints
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", linuxHost.ID), nil, http.StatusNoContent)

	// simulate a successful script result for the lock command
	status, err := s.ds.GetHostLockWipeStatus(ctx, linuxHost)
	require.NoError(t, err)

	var orbitScriptResp orbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *linuxHost.OrbitNodeKey, status.LockScript.ExecutionID)),
		http.StatusOK, &orbitScriptResp)

	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", linuxHost.ID), nil, http.StatusNoContent)

	// windows host status is unchanged, linux is locked pending unlock
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", winHost.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", linuxHost.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "locked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "unlock", *getHostResp.Host.MDM.PendingAction)

	// attempting to Wipe the linux host fails due to pending unlock, not because
	// of MDM not enabled
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", linuxHost.ID), nil, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Host cannot be wiped until unlock is complete.")
}

// checks that the specified team/no-team has the Windows OS Updates profile with
// the specified deadline/grace settings (or checks that it doesn't have the
// profile if wantSettings is nil). It returns the profile_uuid if it exists,
// empty string otherwise.
func checkWindowsOSUpdatesProfile(t *testing.T, ds *mysql.Datastore, teamID *uint, wantSettings *fleet.WindowsUpdates) string {
	ctx := context.Background()

	var prof fleet.MDMWindowsConfigProfile
	mysql.ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		var globalOrTeamID uint
		if teamID != nil {
			globalOrTeamID = *teamID
		}
		err := sqlx.GetContext(ctx, tx, &prof, `SELECT profile_uuid, syncml FROM mdm_windows_configuration_profiles WHERE team_id = ? AND name = ?`, globalOrTeamID, mdm.FleetWindowsOSUpdatesProfileName)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	})
	if wantSettings == nil {
		require.Empty(t, prof.ProfileUUID)
	} else {
		require.NotEmpty(t, prof.ProfileUUID)
		require.Contains(t, string(prof.SyncML), fmt.Sprintf(`<Data>%d</Data>`, wantSettings.DeadlineDays.Value))
		require.Contains(t, string(prof.SyncML), fmt.Sprintf(`<Data>%d</Data>`, wantSettings.GracePeriodDays.Value))
	}

	if len(prof.ProfileUUID) > 0 {
		require.Equal(t, byte('w'), prof.ProfileUUID[0])
	}

	return prof.ProfileUUID
}

func (s *integrationEnterpriseTestSuite) createHosts(t *testing.T, platforms ...string) []*fleet.Host {
	var hosts []*fleet.Host
	if len(platforms) == 0 {
		platforms = []string{"debian", "rhel", "linux", "windows", "darwin"}
	}
	for i, platform := range platforms {
		host, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(fmt.Sprintf("%s%d", t.Name(), i)),
			NodeKey:         ptr.String(fmt.Sprintf("%s%d", t.Name(), i)),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
			Platform:        platform,
		})
		require.NoError(t, err)
		hosts = append(hosts, host)
	}
	return hosts
}

func (s *integrationEnterpriseTestSuite) TestSoftwareAuth() {
	t := s.T()
	ctx := context.Background()
	// create two hosts, one belongs to team1 and one has no team
	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	tmHost, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + "tm"),
		NodeKey:         ptr.String(t.Name() + "tm"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()+"tm"),
		Platform:        "linux",
	})
	require.NoError(t, err)

	// Create two teams, team1 and team2.
	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(ctx, &team1.ID, []uint{tmHost.ID})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{
		ID:          43,
		Name:        "team2",
		Description: "desc team2",
	})
	require.NoError(t, err)

	allSoftware := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "homebrew"},
		{Name: "foo", Version: "0.0.3", Source: "homebrew"},
		{Name: "bar", Version: "0.0.4", Source: "apps"},
	}
	// add all the software entries to the "no team host"
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, allSoftware)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))

	// add only one version of "foo" to the team host
	_, err = s.ds.UpdateHostSoftware(ctx, tmHost.ID, []fleet.Software{allSoftware[0]})
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, tmHost, false))

	// calculate hosts counts
	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, hostsCountTs))
	require.NoError(t, s.ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))

	// add variations of user roles to different teams
	extraTestUsers := make(map[string]fleet.User)
	for k, u := range map[string]struct {
		Email      string
		GlobalRole *string
		Teams      *[]fleet.UserTeam
	}{
		"team-1-admin": {
			Email: "team-1-admin@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team1,
				Role: fleet.RoleAdmin,
			}}),
		},
		"team-1-maintainer": {
			Email: "team-1-maintainer@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team1,
				Role: fleet.RoleMaintainer,
			}}),
		},
		"team-1-observer": {
			Email: "team-1-observer@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team1,
				Role: fleet.RoleObserver,
			}}),
		},
		"team-2-admin": {
			Email: "team-2-admin@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team2,
				Role: fleet.RoleAdmin,
			}}),
		},
		"team-2-maintainer": {
			Email: "team-2-maintainer@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team2,
				Role: fleet.RoleMaintainer,
			}}),
		},
		"team-2-observer": {
			Email: "team-2-observer@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team2,
				Role: fleet.RoleObserver,
			}}),
		},
	} {
		uu := u
		cur := createUserResponse{}
		s.DoJSON("POST", "/api/latest/fleet/users/admin", createUserRequest{
			UserPayload: fleet.UserPayload{
				Email:                    &uu.Email,
				Password:                 &test.GoodPassword,
				Name:                     &uu.Email,
				Teams:                    uu.Teams,
				AdminForcedPasswordReset: ptr.Bool(false),
			},
		}, http.StatusOK, &cur)
		extraTestUsers[k] = *cur.User
	}

	// List all software titles with an admin
	var listSoftwareTitlesResp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &listSoftwareTitlesResp)

	var softwareFoo, softwareBar *fleet.SoftwareTitleListResult
	for _, s := range listSoftwareTitlesResp.SoftwareTitles {
		s := s
		switch s.Name {
		case "foo":
			softwareFoo = &s
		case "bar":
			softwareBar = &s
		}
	}
	require.NotNil(t, softwareFoo)
	require.NotNil(t, softwareBar)

	var teamFooVersion *fleet.SoftwareVersion
	for _, sv := range softwareFoo.Versions {
		sv := sv
		if sv.Version == "0.0.1" {
			teamFooVersion = &sv
		}
	}
	require.NotNil(t, teamFooVersion)

	for _, tc := range []struct {
		name                 string
		user                 fleet.User
		shouldFailGlobalRead bool
		shouldFailTeamRead   bool
	}{
		{
			name:                 "global-admin",
			user:                 s.users["admin1@example.com"],
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "global-maintainer",
			user:                 s.users["user1@example.com"],
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "global-observer",
			user:                 s.users["user2@example.com"],
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "team-admin-belongs-to-team",
			user:                 extraTestUsers["team-1-admin"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "team-maintainer-belongs-to-team",
			user:                 extraTestUsers["team-1-maintainer"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "team-observer-belongs-to-team",
			user:                 extraTestUsers["team-1-observer"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "team-admin-does-not-belong-to-team",
			user:                 extraTestUsers["team-2-admin"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
		},
		{
			name:                 "team-maintainer-does-not-belong-to-team",
			user:                 extraTestUsers["team-2-maintainer"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
		},
		{
			name:                 "team-observer-does-not-belong-to-team",
			user:                 extraTestUsers["team-2-observer"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// to make the request as the user
			s.token = s.getTestToken(tc.user.Email, test.GoodPassword)

			if tc.shouldFailGlobalRead {
				// List all software titles
				var listSoftwareTitlesResp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusForbidden, &listSoftwareTitlesResp)

				// List all software versions
				var resp listSoftwareVersionsResponse
				s.DoJSON("GET", "/api/latest/fleet/software/versions", listSoftwareTitlesRequest{}, http.StatusForbidden, &resp)

				// Get a global software title
				var getSoftwareTitleResp getSoftwareTitleResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", softwareBar.ID), getSoftwareTitleRequest{}, http.StatusForbidden, &getSoftwareTitleResp)

				// Get a global software version
				var getSoftwareResp getSoftwareResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", softwareBar.Versions[0].ID), getSoftwareRequest{}, http.StatusForbidden, &getSoftwareResp)

				// Get a global software vesion using the deprecated endpoint
				getSoftwareResp = getSoftwareResponse{}
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/%d", softwareBar.Versions[0].ID), getSoftwareRequest{}, http.StatusForbidden, &getSoftwareResp)

				// Get a count of software vesions using the deprecated endpoint
				var countSoftwareResp countSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software/count", getSoftwareRequest{}, http.StatusForbidden, &countSoftwareResp)

				// List all software versions using the deprecated endpoint
				var softwareListResp listSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software", listSoftwareRequest{}, http.StatusForbidden, &softwareListResp)
			} else {
				// List all software titles
				var listSoftwareTitlesResp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &listSoftwareTitlesResp)
				require.Equal(t, 2, listSoftwareTitlesResp.Count)
				require.NotEmpty(t, listSoftwareTitlesResp.CountsUpdatedAt)

				// List all software versions
				var resp listSoftwareVersionsResponse
				s.DoJSON("GET", "/api/latest/fleet/software/versions", listSoftwareRequest{}, http.StatusOK, &resp)
				require.Equal(t, 3, resp.Count)
				require.NotEmpty(t, resp.CountsUpdatedAt)

				// Get a global software title
				var getSoftwareTitleResp getSoftwareTitleResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", softwareBar.ID), getSoftwareTitleRequest{}, http.StatusOK, &getSoftwareTitleResp)

				// Get a global software version
				var getSoftwareResp getSoftwareResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", softwareBar.Versions[0].ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResp)

				// Get a global software vesion using the deprecated endpoint
				getSoftwareResp = getSoftwareResponse{}
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/%d", softwareBar.Versions[0].ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResp)

				// Get a global count of software vesions using the deprecated endpoint
				var countSoftwareResp countSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software/count", countSoftwareRequest{}, http.StatusOK, &countSoftwareResp)
				require.Equal(t, 3, countSoftwareResp.Count)

				// List all software versions using the deprecated endpoint
				var softwareListResp listSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software", listSoftwareRequest{}, http.StatusOK, &softwareListResp)
				require.Equal(t, countSoftwareResp.Count, 3)
			}

			if tc.shouldFailTeamRead {
				// List all software titles for a team
				var listSoftwareTitlesResp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{SoftwareTitleListOptions: fleet.SoftwareTitleListOptions{TeamID: &team1.ID}}, http.StatusForbidden, &listSoftwareTitlesResp)

				// List software versions for a team.
				var resp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/versions", listSoftwareRequest{SoftwareListOptions: fleet.SoftwareListOptions{TeamID: &team1.ID}}, http.StatusForbidden, &resp)

				// Get a team software title
				var getSoftwareTitleResp getSoftwareTitleResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", softwareFoo.ID), getSoftwareTitleRequest{}, http.StatusForbidden, &getSoftwareTitleResp)

				// Get a team software version
				var getSoftwareResp getSoftwareResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", teamFooVersion.ID), getSoftwareRequest{}, http.StatusForbidden, &getSoftwareResp)

				// Get a team software vesion using the deprecated endpoint
				getSoftwareResp = getSoftwareResponse{}
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/%d", teamFooVersion.ID), getSoftwareRequest{}, http.StatusForbidden, &getSoftwareResp)

				// Get a count of team software vesions using the deprecated endpoint
				var countSoftwareResp countSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software/count", getSoftwareRequest{}, http.StatusForbidden, &countSoftwareResp)

				// List all software versions using the deprecated endpoint for a team
				var softwareListResp listSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software", listSoftwareRequest{}, http.StatusForbidden, &softwareListResp)
			} else {
				// List all software titles for a team
				var listSoftwareTitlesResp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{SoftwareTitleListOptions: fleet.SoftwareTitleListOptions{TeamID: &team1.ID}}, http.StatusOK, &listSoftwareTitlesResp)
				require.Equal(t, 1, listSoftwareTitlesResp.Count)
				require.NotEmpty(t, listSoftwareTitlesResp.CountsUpdatedAt)

				// List software versions for a team.
				var resp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/versions", listSoftwareRequest{SoftwareListOptions: fleet.SoftwareListOptions{TeamID: &team1.ID}}, http.StatusOK, &resp)
				require.Equal(t, 1, resp.Count)
				require.NotEmpty(t, resp.CountsUpdatedAt)

				// Get a team software title
				var getSoftwareTitleResp getSoftwareTitleResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", softwareFoo.ID), getSoftwareTitleRequest{}, http.StatusOK, &getSoftwareTitleResp)

				// Get a team software version
				var getSoftwareResp getSoftwareResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", teamFooVersion.ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResp)

				// Get a team software vesion using the deprecated endpoint
				getSoftwareResp = getSoftwareResponse{}
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/%d", teamFooVersion.ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResp)

				// Get a team count of software vesions using the deprecated endpoint
				var countSoftwareResp countSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software/count", countSoftwareRequest{SoftwareListOptions: fleet.SoftwareListOptions{TeamID: &team1.ID}}, http.StatusOK, &countSoftwareResp)
				require.Equal(t, 1, countSoftwareResp.Count)

				// List all software versions using the deprecated endpoint for a team
				var softwareListResp listSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software", listSoftwareRequest{SoftwareListOptions: fleet.SoftwareListOptions{TeamID: &team1.ID}}, http.StatusOK, &softwareListResp)
				require.Equal(t, countSoftwareResp.Count, 1)
			}
		})
	}

	// set the admin token again to avoid breaking other tests
	s.token = s.getTestAdminToken()
}

func (s *integrationEnterpriseTestSuite) TestCalendarEvents() {
	ctx := context.Background()
	t := s.T()
	t.Cleanup(func() {
		calendar.ClearMockEvents()
		calendar.ClearMockChannels()
	})
	currentAppCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = s.ds.SaveAppConfig(ctx, currentAppCfg)
		require.NoError(t, err)
	})

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "team2",
	})
	require.NoError(t, err)

	newHost := func(name string, teamID *uint) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   ptr.String(t.Name() + name),
			NodeKey:         ptr.String(t.Name() + name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:        "darwin",
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}

	host1Team1 := newHost("host1", &team1.ID)
	host2Team1 := newHost("host2", &team1.ID)
	host3Team2 := newHost("host3", &team2.ID)
	host4Team2 := newHost("host4", &team2.ID)
	_ = newHost("host5", nil) // global host

	team1Policy1Calendar, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team1Policy1Calendar",
			Query:                 "SELECT 1;",
			CalendarEventsEnabled: true,
		},
	)
	require.NoError(t, err)
	team1Policy2, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team1Policy2",
			Query:                 "SELECT 2;",
			CalendarEventsEnabled: true,
		},
	)
	require.NoError(t, err)
	team2Policy1Calendar, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team2Policy1Calendar",
			Query:                 "SELECT 3;",
			CalendarEventsEnabled: true,
		},
	)
	require.NoError(t, err)
	team2Policy2, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team2Policy2",
			Query:                 "SELECT 4;",
			CalendarEventsEnabled: false,
		},
	)
	require.NoError(t, err)
	globalPolicy, err := s.ds.NewGlobalPolicy(
		ctx, nil, fleet.PolicyPayload{
			Name:                  "globalPolicy",
			Query:                 "SELECT 5;",
			CalendarEventsEnabled: false,
		},
	)
	require.NoError(t, err)

	genDistributedReqWithPolicyResults := func(host *fleet.Host, policyResults map[uint]*bool) submitDistributedQueryResultsRequestShim {
		var (
			results  = make(map[string]json.RawMessage)
			statuses = make(map[string]interface{})
			messages = make(map[string]string)
		)
		for policyID, policyResult := range policyResults {
			distributedQueryName := hostPolicyQueryPrefix + fmt.Sprint(policyID)
			switch {
			case policyResult == nil:
				results[distributedQueryName] = json.RawMessage(`[]`)
				statuses[distributedQueryName] = 1
				messages[distributedQueryName] = "policy failed execution"
			case *policyResult:
				results[distributedQueryName] = json.RawMessage(`[{"1": "1"}]`)
				statuses[distributedQueryName] = 0
			case !*policyResult:
				results[distributedQueryName] = json.RawMessage(`[]`)
				statuses[distributedQueryName] = 0
			}
		}
		return submitDistributedQueryResultsRequestShim{
			NodeKey:  *host.NodeKey,
			Results:  results,
			Statuses: statuses,
			Messages: messages,
			Stats:    map[string]*fleet.Stats{},
		}
	}

	// host1Team1 is failing a calendar policy and not a non-calendar policy (no results for global).
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: ptr.Bool(false),
			team1Policy2.ID:         ptr.Bool(true),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// host2Team1 is passing the calendar policy but not the non-calendar policy (no results for global).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host2Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: ptr.Bool(true),
			team1Policy2.ID:         ptr.Bool(false),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// host3Team2 is passing team2Policy1Calendar and failing the global policy
	// (not results for team2Policy2).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3Team2,
		map[uint]*bool{
			team2Policy1Calendar.ID: ptr.Bool(true),
			team2Policy2.ID:         nil,
			globalPolicy.ID:         ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	// host4Team2 is not returning results for the calendar policy, failing the non-calendar
	// policy and passing the global policy.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host4Team2,
		map[uint]*bool{
			team2Policy1Calendar.ID: nil,
			team2Policy2.ID:         ptr.Bool(false),
			globalPolicy.ID:         ptr.Bool(true),
		},
	), http.StatusOK, &distributedResp)

	// Trigger the calendar cron with the global feature is disabled.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// No calendar events were created.
	allCalendarEvents, err := s.ds.ListCalendarEvents(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, allCalendarEvents)

	// Set global configuration for the calendar feature.
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.Integrations.GoogleCalendar = []*fleet.GoogleCalendarIntegration{
		{
			Domain: "example.com",
			ApiKey: map[string]string{
				fleet.GoogleCalendarEmail: "calendar-mock@example.com",
			},
		},
	}
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	time.Sleep(2 * time.Second) // Wait 2 seconds for the app config cache to clear.

	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// No calendar events were created because we are missing enabling it on the teams.
	allCalendarEvents, err = s.ds.ListCalendarEvents(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, allCalendarEvents)

	// Run distributed/write for host4Team2 again, it should not attempt to trigger the webhook because
	// it's disabled for the teams.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host4Team2,
		map[uint]*bool{
			team2Policy1Calendar.ID: nil,
			team2Policy2.ID:         ptr.Bool(false),
			globalPolicy.ID:         ptr.Bool(true),
		},
	), http.StatusOK, &distributedResp)

	var (
		team1Fired   int
		team1FiredMu sync.Mutex
	)

	team1WebhookFired := make(chan struct{})
	team1WebhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		requestBodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		t.Logf("team1 webhook request: %s\n", requestBodyBytes)
		team1FiredMu.Lock()
		team1Fired++
		team1WebhookFired <- struct{}{}
		team1FiredMu.Unlock()
	}))
	t.Cleanup(func() {
		team1WebhookServer.Close()
	})

	team1.Config.Integrations.GoogleCalendar = &fleet.TeamGoogleCalendarIntegration{
		Enable:     true,
		WebhookURL: team1WebhookServer.URL,
	}
	team1, err = s.ds.SaveTeam(ctx, team1)
	require.NoError(t, err)

	var (
		team2Fired   int
		team2FiredMu sync.Mutex
	)

	team2WebhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		requestBodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		t.Logf("team2 webhook request: %s\n", requestBodyBytes)
		team2FiredMu.Lock()
		team2Fired++
		team2FiredMu.Unlock()
	}))
	t.Cleanup(func() {
		team2WebhookServer.Close()
	})

	team2.Config.Integrations.GoogleCalendar = &fleet.TeamGoogleCalendarIntegration{
		Enable:     true,
		WebhookURL: team2WebhookServer.URL,
	}
	_, err = s.ds.SaveTeam(ctx, team2)
	require.NoError(t, err)

	//
	// Same distributed/write as before but they should not fire yet.
	//

	// host1Team1 is failing a calendar policy and not a non-calendar policy (no results for global).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: ptr.Bool(false),
			team1Policy2.ID:         ptr.Bool(true),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// host2Team1 is passing the calendar policy but not the non-calendar policy (no results for global).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host2Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: ptr.Bool(true),
			team1Policy2.ID:         ptr.Bool(false),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// host3Team2 is passing team2Policy1Calendar and failing the global policy
	// (not results for team2Policy2).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3Team2,
		map[uint]*bool{
			team2Policy1Calendar.ID: ptr.Bool(true),
			team2Policy2.ID:         nil,
			globalPolicy.ID:         ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	// host4Team2 is not returning results for the calendar policy, failing the non-calendar
	// policy and passing the global policy.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host4Team2,
		map[uint]*bool{
			team2Policy1Calendar.ID: nil,
			team2Policy2.ID:         ptr.Bool(false),
			globalPolicy.ID:         ptr.Bool(true),
		},
	), http.StatusOK, &distributedResp)

	team1FiredMu.Lock()
	require.Zero(t, team1Fired)
	team1FiredMu.Unlock()

	team2FiredMu.Lock()
	require.Zero(t, team2Fired)
	team2FiredMu.Unlock()

	// Trigger the calendar cron, global feature enabled, team1 enabled, team2 not yet enabled
	// and hosts do not have an associated email yet.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	team1CalendarEvents, err := s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Empty(t, team1CalendarEvents)

	// Add an email but of another domain.
	err = s.ds.ReplaceHostDeviceMapping(ctx, host1Team1.ID, []*fleet.HostDeviceMapping{
		{
			HostID: host1Team1.ID,
			Email:  "user@other.com",
			Source: "google_chrome_profiles",
		},
	}, "google_chrome_profiles")
	require.NoError(t, err)

	// Trigger the calendar cron, global feature enabled, team1 enabled, team2 not yet enabled
	// and hosts do not have an associated email for the domain yet.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Empty(t, team1CalendarEvents)

	err = s.ds.ReplaceHostDeviceMapping(ctx, host1Team1.ID, []*fleet.HostDeviceMapping{
		{
			HostID: host1Team1.ID,
			Email:  "user1@example.com",
			Source: "google_chrome_profiles",
		},
	}, "google_chrome_profiles")
	require.NoError(t, err)

	// Trigger the calendar cron, global feature enabled, team1 enabled, team2 not yet enabled
	// and host1Team1 has a domain email associated.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// An event should be generated for host1Team1
	team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, team1CalendarEvents, 1)
	require.NotZero(t, team1CalendarEvents[0].ID)
	require.Equal(t, "user1@example.com", team1CalendarEvents[0].Email)
	require.NotZero(t, team1CalendarEvents[0].StartTime)
	require.NotZero(t, team1CalendarEvents[0].EndTime)

	calendar.SetMockEventsToNow()

	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		// Update updated_at so the event gets updated (the event is updated regularly)
		_, err := db.ExecContext(ctx,
			`UPDATE calendar_events SET updated_at = DATE_SUB(CURRENT_TIMESTAMP, INTERVAL 25 HOUR) WHERE id = ?`, team1CalendarEvents[0].ID)
		if err != nil {
			return err
		}
		// Set host1Team1 as online.
		if _, err := db.ExecContext(ctx,
			`UPDATE host_seen_times SET seen_time = CURRENT_TIMESTAMP WHERE host_id = ?`, host1Team1.ID); err != nil {
			return err
		}
		return nil
	})

	// Trigger the calendar cron, global feature enabled, team1 enabled, team2 not yet enabled
	// and host1Team1 has a domain email associated.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// Check that refetch on the host was set.
	host, err := s.ds.Host(ctx, host1Team1.ID)
	require.NoError(t, err)
	require.True(t, host.RefetchRequested)

	// host1Team1 is failing a calendar policy and not a non-calendar policy (no results for global).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: ptr.Bool(false),
			team1Policy2.ID:         ptr.Bool(true),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// host2Team1 is passing the calendar policy but not the non-calendar policy (no results for global).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host2Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: ptr.Bool(true),
			team1Policy2.ID:         ptr.Bool(false),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	select {
	case <-team1WebhookFired:
	case <-time.After(5 * time.Second):
		t.Error("timeout waiting for team1 webhook to fire")
	}

	// Trigger again, nothing should fire as webhook has already fired.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	team1FiredMu.Lock()
	require.Equal(t, 1, team1Fired)
	team1FiredMu.Unlock()
	team2FiredMu.Lock()
	require.Equal(t, 0, team2Fired)
	team2FiredMu.Unlock()

	// Make host1Team1 pass all policies.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: ptr.Bool(true),
			team1Policy2.ID:         ptr.Bool(true),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// Trigger calendar should cleanup the events.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// Events in the user calendar should not be cleaned up because they are not in the future.
	mockEvents := calendar.ListGoogleMockEvents()
	require.NotEmpty(t, mockEvents)

	// Event should be cleaned up from our database.
	team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Empty(t, team1CalendarEvents)
}

func (s *integrationEnterpriseTestSuite) TestCalendarEventsTransferringHosts() {
	ctx := context.Background()
	t := s.T()
	t.Cleanup(func() {
		calendar.ClearMockEvents()
		calendar.ClearMockChannels()
	})
	currentAppCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = s.ds.SaveAppConfig(ctx, currentAppCfg)
		require.NoError(t, err)
	})

	// Set global configuration for the calendar feature.
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.Integrations.GoogleCalendar = []*fleet.GoogleCalendarIntegration{
		{
			Domain: "example.com",
			ApiKey: map[string]string{
				fleet.GoogleCalendarEmail: "calendar-mock@example.com",
			},
		},
	}
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	time.Sleep(2 * time.Second) // Wait 2 seconds for the app config cache to clear.

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "team2",
	})
	require.NoError(t, err)

	team1.Config.Integrations.GoogleCalendar = &fleet.TeamGoogleCalendarIntegration{
		Enable:     true,
		WebhookURL: "https://foo.example.com",
	}
	team1, err = s.ds.SaveTeam(ctx, team1)
	require.NoError(t, err)
	team2.Config.Integrations.GoogleCalendar = &fleet.TeamGoogleCalendarIntegration{
		Enable:     true,
		WebhookURL: "https://foo.example.com",
	}
	team2, err = s.ds.SaveTeam(ctx, team2)
	require.NoError(t, err)

	newHost := func(name string, teamID *uint) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   ptr.String(t.Name() + name),
			NodeKey:         ptr.String(t.Name() + name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:        "darwin",
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}

	host1 := newHost("host1", &team1.ID)
	err = s.ds.ReplaceHostDeviceMapping(ctx, host1.ID, []*fleet.HostDeviceMapping{
		{
			HostID: host1.ID,
			Email:  "user1@example.com",
			Source: "google_chrome_profiles",
		},
	}, "google_chrome_profiles")
	require.NoError(t, err)

	team1Policy1, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team1Policy1",
			Query:                 "SELECT 1;",
			CalendarEventsEnabled: true,
		},
	)
	require.NoError(t, err)
	team2Policy1, err := s.ds.NewTeamPolicy(
		ctx, team2.ID, nil, fleet.PolicyPayload{
			Name:                  "team2Policy1",
			Query:                 "SELECT 2;",
			CalendarEventsEnabled: true,
		},
	)
	require.NoError(t, err)

	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1,
		map[uint]*bool{
			team1Policy1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	team1CalendarEvents, err := s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, team1CalendarEvents, 1)

	// Check the calendar was created on the DB.
	hostCalendarEvent, calendarEvent, err := s.ds.GetHostCalendarEventByEmail(ctx, "user1@example.com")
	require.NoError(t, err)

	// Transfer host to team2.
	err = s.ds.AddHostsToTeam(ctx, &team2.ID, []uint{host1.ID})
	require.NoError(t, err)

	// host1 is failing team2's policy too.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1,
		map[uint]*bool{
			team2Policy1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// Check the calendar event entry was reused.
	hostCalendarEvent2, calendarEvent2, err := s.ds.GetHostCalendarEventByEmail(ctx, "user1@example.com")
	require.NoError(t, err)
	require.Equal(t, calendarEvent2.ID, calendarEvent.ID)
	require.Equal(t, hostCalendarEvent2.CalendarEventID, hostCalendarEvent.CalendarEventID)

	// Transfer host to global.
	err = s.ds.AddHostsToTeam(ctx, nil, []uint{host1.ID})
	require.NoError(t, err)

	// Move event to two days ago (to clean up the calendar event)
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(ctx,
			`UPDATE calendar_events SET updated_at = DATE_SUB(CURRENT_TIMESTAMP, INTERVAL 49 HOUR) WHERE id = ?`, team1CalendarEvents[0].ID)
		if err != nil {
			return err
		}
		return nil
	})

	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// Calendar event is cleaned up.
	_, _, err = s.ds.GetHostCalendarEventByEmail(ctx, "user1@example.com")
	require.True(t, fleet.IsNotFound(err))
}

func (s *integrationEnterpriseTestSuite) TestLabelsHostsCounts() {
	// ensure that on exit, the admin token is used
	defer func() { s.token = s.getTestAdminToken() }()

	t := s.T()
	ctx := context.Background()

	hosts := s.createHosts(t, "debian", "linux", "fedora", "darwin", "darwin")
	tm1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	// move a couple hosts to tm1, one to tm2
	err = s.ds.AddHostsToTeam(ctx, &tm1.ID, []uint{hosts[0].ID, hosts[1].ID})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(ctx, &tm2.ID, []uint{hosts[2].ID})
	require.NoError(t, err)

	// create new users for tm1, tm2 and one with both tm1 and tm2
	users := []fleet.UserPayload{
		{
			Name:                     ptr.String("team1 user"),
			Email:                    ptr.String("tm1user@example.com"),
			Password:                 ptr.String(test.GoodPassword),
			AdminForcedPasswordReset: ptr.Bool(false),
			Teams: &[]fleet.UserTeam{
				{Team: fleet.Team{ID: tm1.ID}, Role: fleet.RoleMaintainer},
			},
		},
		{
			Name:                     ptr.String("team2 user"),
			Email:                    ptr.String("tm2user@example.com"),
			Password:                 ptr.String(test.GoodPassword),
			AdminForcedPasswordReset: ptr.Bool(false),
			Teams: &[]fleet.UserTeam{
				{Team: fleet.Team{ID: tm2.ID}, Role: fleet.RoleAdmin},
			},
		},
		{
			Name:                     ptr.String("team1and2 user"),
			Email:                    ptr.String("tm1and2user@example.com"),
			Password:                 ptr.String(test.GoodPassword),
			AdminForcedPasswordReset: ptr.Bool(false),
			Teams: &[]fleet.UserTeam{
				{Team: fleet.Team{ID: tm1.ID}, Role: fleet.RoleObserver},
				{Team: fleet.Team{ID: tm2.ID}, Role: fleet.RoleObserverPlus},
			},
		},
	}
	for _, u := range users {
		var createResp createUserResponse
		s.DoJSON("POST", "/api/latest/fleet/users/admin", u, http.StatusOK, &createResp)
	}

	// create a manual label with hosts across no team, team1 and team2
	var createLbl createLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", createLabelRequest{
		LabelPayload: fleet.LabelPayload{
			Name:  "manual1",
			Hosts: []string{hosts[0].UUID, hosts[1].UUID, hosts[2].UUID, hosts[3].UUID},
		},
	}, http.StatusOK, &createLbl)
	// user is admin, count contains all hosts
	require.Equal(t, 4, createLbl.Label.Count)
	lblM1 := createLbl.Label.ID
	require.NotZero(t, lblM1)

	// create a dynamic label always returns a count of 0 (no members yet)
	s.DoJSON("POST", "/api/latest/fleet/labels", createLabelRequest{
		LabelPayload: fleet.LabelPayload{
			Name:  "dynamic1",
			Query: "select 1",
		},
	}, http.StatusOK, &createLbl)
	require.Equal(t, 0, createLbl.Label.Count)
	lblD1 := createLbl.Label.ID
	require.NotZero(t, lblD1)

	// record membership for hosts across no team, team1 and team2
	err = s.ds.RecordLabelQueryExecutions(ctx, hosts[4], map[uint]*bool{lblD1: ptr.Bool(true)}, time.Now(), false)
	require.NoError(t, err)
	err = s.ds.RecordLabelQueryExecutions(ctx, hosts[2], map[uint]*bool{lblD1: ptr.Bool(true)}, time.Now(), false)
	require.NoError(t, err)
	err = s.ds.RecordLabelQueryExecutions(ctx, hosts[1], map[uint]*bool{lblD1: ptr.Bool(true)}, time.Now(), false)
	require.NoError(t, err)
	err = s.ds.RecordLabelQueryExecutions(ctx, hosts[0], map[uint]*bool{lblD1: ptr.Bool(true)}, time.Now(), false)
	require.NoError(t, err)

	// create another dynamic label which will stay empty
	s.DoJSON("POST", "/api/latest/fleet/labels", createLabelRequest{
		LabelPayload: fleet.LabelPayload{
			Name:  "dynamic2",
			Query: "select 2",
		},
	}, http.StatusOK, &createLbl)
	require.Equal(t, 0, createLbl.Label.Count)
	lblD2 := createLbl.Label.ID
	require.NotZero(t, lblD2)

	// test access with each team user
	adminUserPayload := fleet.UserPayload{
		Name:     ptr.String("admin1"),
		Email:    ptr.String(testUsers["admin1"].Email),
		Password: ptr.String(testUsers["admin1"].PlaintextPassword),
	}
	cases := []struct {
		desc  string
		u     fleet.UserPayload
		lblID uint
		want  int
	}{
		{"team1 user, manual1", users[0], lblM1, 2},
		{"team1 user, dynamic1", users[0], lblD1, 2},
		{"team1 user, dynamic2", users[0], lblD2, 0},
		{"team2 user, manual1", users[1], lblM1, 1},
		{"team2 user, dynamic1", users[1], lblD1, 1},
		{"team2 user, dynamic2", users[1], lblD2, 0},
		{"team1 and 2 user, manual1", users[2], lblM1, 3},
		{"team1 and 2 user, dynamic1", users[2], lblD1, 3},
		{"team1 and 2 user, dynamic2", users[2], lblD2, 0},
		{"admin user, manual1", adminUserPayload, lblM1, 4},
		{"admin user, dynamic1", adminUserPayload, lblD1, 4},
		{"admin user, dynamic2", adminUserPayload, lblD2, 0},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			s.setTokenForTest(t, *c.u.Email, *c.u.Password)

			var getLbl getLabelResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", c.lblID), nil, http.StatusOK, &getLbl)
			require.Equal(t, c.want, getLbl.Label.Count)

			var listLbls listLabelsResponse
			s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listLbls)
			var found bool
			for _, lbl := range listLbls.Labels {
				if lbl.ID == c.lblID {
					found = true
					require.Equal(t, c.want, lbl.Count)
					break
				}
			}
			require.True(t, found)

			// create and update label and just not possible for non-global users
			if c.u != adminUserPayload {
				s.DoJSON("POST", "/api/latest/fleet/labels", createLabelRequest{
					LabelPayload: fleet.LabelPayload{
						Name:  "will fail",
						Query: "select 3",
					},
				}, http.StatusForbidden, &createLbl)

				s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", c.lblID), modifyLabelRequest{
					ModifyLabelPayload: fleet.ModifyLabelPayload{
						Name: ptr.String("will fail"),
					},
				}, http.StatusForbidden, &modifyLabelResponse{})
			}
		})
	}
}

func (s *integrationEnterpriseTestSuite) TestListHostSoftware() {
	ctx := context.Background()
	t := s.T()

	token := "good_token"
	host := createOrbitEnrolledHost(t, "linux", "host1", s.ds)
	createDeviceTokenForHost(t, s.ds, host.ID, token)

	// no software yet
	var getHostSw getHostSoftwareResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw)
	require.Len(t, getHostSw.Software, 0)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "self_service", "true")
	require.Len(t, getHostSw.Software, 0)

	var getDeviceSw getDeviceSoftwareResponse
	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software", nil, http.StatusOK)
	err := json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 0)
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?self_service=1", nil, http.StatusOK)
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 0)

	// create some software for that host
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.1", Source: "apps"},
	}
	us, err := s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	err = s.ds.ReconcileSoftwareTitles(ctx)
	require.NoError(t, err)

	// Note: the ID returned by ListHostSoftware is the title ID, not the software ID. We need the
	// software ID to assign the vulnerabilities correctly below.
	var barSoftwareID uint
	for _, s := range us.Inserted {
		if s.Name == "bar" {
			barSoftwareID = s.ID
		}
	}

	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw)
	require.Len(t, getHostSw.Software, 2) // foo and bar
	require.Equal(t, getHostSw.Software[0].Name, "bar")
	require.Equal(t, getHostSw.Software[1].Name, "foo")
	require.Len(t, getHostSw.Software[1].InstalledVersions, 2)
	// no package information as there is no installer
	require.Nil(t, getHostSw.Software[0].SoftwarePackage)
	require.Nil(t, getHostSw.Software[0].AppStoreApp)
	require.Nil(t, getHostSw.Software[1].SoftwarePackage)
	require.Nil(t, getHostSw.Software[1].AppStoreApp)

	// Add vulnerabilities to software to check query param filtering
	_, err = s.ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{SoftwareID: barSoftwareID, CVE: "CVE-bar-1234"}, fleet.NVDSource)
	require.NoError(t, err)
	_, err = s.ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{SoftwareID: barSoftwareID, CVE: "CVE-bar-5678"}, fleet.NVDSource)
	require.NoError(t, err)

	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "vulnerable", "true")
	require.Len(t, getHostSw.Software, 1)
	require.NoError(t, err)
	require.Equal(t, "bar", getHostSw.Software[0].Name)
	require.Nil(t, getHostSw.Software[0].SoftwarePackage)
	require.Nil(t, getHostSw.Software[0].AppStoreApp)
	require.Len(t, getHostSw.Software[0].InstalledVersions, 1)
	require.Len(t, getHostSw.Software[0].InstalledVersions[0].Vulnerabilities, 2)
	require.Equal(t, getHostSw.Software[0].InstalledVersions[0].Vulnerabilities, []string{"CVE-bar-1234", "CVE-bar-5678"})

	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 2) // foo and bar
	require.Equal(t, getDeviceSw.Software[0].Name, "bar")
	require.Equal(t, getDeviceSw.Software[1].Name, "foo")
	require.Len(t, getDeviceSw.Software[1].InstalledVersions, 2)
	// no package information as there is no installer
	require.Nil(t, getDeviceSw.Software[0].SoftwarePackage)
	require.Nil(t, getDeviceSw.Software[0].AppStoreApp)
	require.Nil(t, getDeviceSw.Software[1].SoftwarePackage)
	require.Nil(t, getDeviceSw.Software[1].AppStoreApp)

	// create a software installer, not installed on the host
	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "ruby.deb",
		Version:       "1:2.5.1",
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
	titleID := getSoftwareTitleID(t, s.ds, "ruby", "deb_packages")

	// update it to be self-service
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "UPDATE software_installers SET self_service = 1 WHERE filename = ?", payload.Filename)
		return err
	})

	// available installer is returned by user-authenticated endpoint
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw)
	require.Len(t, getHostSw.Software, 3) // foo, bar and ruby.deb
	require.Equal(t, getHostSw.Software[0].Name, "bar")
	require.Equal(t, getHostSw.Software[1].Name, "foo")
	require.Equal(t, getHostSw.Software[2].Name, "ruby")
	require.Len(t, getHostSw.Software[1].InstalledVersions, 2)
	require.Nil(t, getHostSw.Software[2].AppStoreApp)
	require.NotNil(t, getHostSw.Software[2].SoftwarePackage)
	require.Equal(t, "ruby.deb", getHostSw.Software[2].SoftwarePackage.Name)
	require.Equal(t, payload.Version, getHostSw.Software[2].SoftwarePackage.Version)
	require.NotNil(t, getHostSw.Software[2].SoftwarePackage.SelfService)
	require.True(t, *getHostSw.Software[2].SoftwarePackage.SelfService)
	require.Nil(t, getHostSw.Software[2].Status)

	// only the installer is returned for self-service only
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "self_service", "true")
	require.Len(t, getHostSw.Software, 1)
	require.Equal(t, getHostSw.Software[0].Name, "ruby")

	// available installer is not returned by device-authenticated endpoint
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 2) // foo and bar
	require.Equal(t, getDeviceSw.Software[0].Name, "bar")
	require.Equal(t, getDeviceSw.Software[1].Name, "foo")
	require.Len(t, getDeviceSw.Software[1].InstalledVersions, 2)

	// but it gets returned for self-service only
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?self_service=1", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 1)
	require.Equal(t, getDeviceSw.Software[0].Name, "ruby")
	require.Nil(t, getDeviceSw.Software[0].AppStoreApp)
	require.NotNil(t, getDeviceSw.Software[0].SoftwarePackage)
	require.NotNil(t, getDeviceSw.Software[0].SoftwarePackage.SelfService)
	require.True(t, *getDeviceSw.Software[0].SoftwarePackage.SelfService)
	require.Equal(t, payload.Filename, getDeviceSw.Software[0].SoftwarePackage.Name)
	require.Equal(t, payload.Version, getDeviceSw.Software[0].SoftwarePackage.Version)

	// request installation on the host
	var installResp installSoftwareResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install",
		host.ID, titleID), nil, http.StatusAccepted, &installResp)

	// still returned by user-authenticated endpoint, now pending
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw)
	require.Len(t, getHostSw.Software, 3) // foo, bar and ruby.deb
	require.Equal(t, getHostSw.Software[0].Name, "bar")
	require.Equal(t, getHostSw.Software[1].Name, "foo")
	require.Equal(t, getHostSw.Software[2].Name, "ruby")
	require.Len(t, getHostSw.Software[1].InstalledVersions, 2)
	require.NotNil(t, getHostSw.Software[2].SoftwarePackage)
	require.Equal(t, "ruby.deb", getHostSw.Software[2].SoftwarePackage.Name)
	require.NotNil(t, getHostSw.Software[2].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[2].Status)
	require.NotNil(t, getHostSw.Software[2].SoftwarePackage.SelfService)
	require.True(t, *getHostSw.Software[2].SoftwarePackage.SelfService)

	// still returned with self-service filter
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "self_service", "true")
	require.Len(t, getHostSw.Software, 1)
	require.Equal(t, getHostSw.Software[0].Name, "ruby")

	// now returned by device-authenticated endpoint
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 3) // foo, bar and ruby
	require.Equal(t, getDeviceSw.Software[0].Name, "bar")
	require.Equal(t, getDeviceSw.Software[1].Name, "foo")
	require.Equal(t, getDeviceSw.Software[2].Name, "ruby")
	require.Len(t, getDeviceSw.Software[1].InstalledVersions, 2)
	require.NotNil(t, getDeviceSw.Software[2].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getDeviceSw.Software[2].Status)
	require.NotNil(t, getDeviceSw.Software[2].SoftwarePackage)
	require.Nil(t, getDeviceSw.Software[2].AppStoreApp)
	require.NotNil(t, getDeviceSw.Software[2].SoftwarePackage.SelfService)
	require.True(t, *getDeviceSw.Software[2].SoftwarePackage.SelfService)

	// still returned for self-service only too
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?self_service=1", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 1)
	require.Equal(t, getDeviceSw.Software[0].Name, "ruby")
	require.NotNil(t, getDeviceSw.Software[0].SoftwarePackage)
	require.NotNil(t, getDeviceSw.Software[0].SoftwarePackage.SelfService)
	require.True(t, *getDeviceSw.Software[0].SoftwarePackage.SelfService)
	require.Nil(t, getDeviceSw.Software[0].AppStoreApp)

	// test with a query
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "query", "foo")
	require.Len(t, getHostSw.Software, 1) // foo only
	require.Equal(t, getHostSw.Software[0].Name, "foo")
	require.Len(t, getHostSw.Software[0].InstalledVersions, 2)

	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?query=bar", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 1) // bar only
	require.Equal(t, getDeviceSw.Software[0].Name, "bar")
	require.Len(t, getDeviceSw.Software[0].InstalledVersions, 1)

	// Add new software to host -- installed on host, but not by Fleet
	installedVersion := "1.0.1"
	softwareAlreadyInstalled := fleet.Software{
		Name: "DummyApp.app", Version: installedVersion, Source: "apps",
		BundleIdentifier: "com.example.dummy",
	}
	software = append(software, softwareAlreadyInstalled)
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	err = s.ds.ReconcileSoftwareTitles(ctx)
	require.NoError(t, err)
	// Add installer for software that is already installed on host
	payload = &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "dummy_installer.pkg",
		Version:       "0.0.2", // The version can be anything -- we match on title
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

	// Get software available for install
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "available_for_install",
		"true", "order_key", "name", "order_direction", "asc")
	require.Len(t, getHostSw.Software, 2) // DummyApp.app and ruby
	assert.Equal(t, softwareAlreadyInstalled.Name, getHostSw.Software[0].Name)
	require.Len(t, getHostSw.Software[0].InstalledVersions, 1)
	assert.Equal(t, installedVersion, getHostSw.Software[0].InstalledVersions[0].Version)
	assert.NotNil(t, getHostSw.Software[0].SoftwarePackage)
	assert.Nil(t, getHostSw.Software[0].Status)
}

func (s *integrationEnterpriseTestSuite) TestSoftwareInstallerUploadDownloadAndDelete() {
	t := s.T()

	openFile := func(name string) *os.File {
		f, err := os.Open(filepath.Join("testdata", "software-installers", name))
		require.NoError(t, err)
		return f
	}

	var expectBytes []byte
	var expectLen int
	f := openFile("ruby.deb")
	st, err := f.Stat()
	require.NoError(t, err)
	expectLen = int(st.Size())
	require.Equal(t, expectLen, 11340)
	expectBytes = make([]byte, expectLen)
	n, err := f.Read(expectBytes)
	require.NoError(t, err)
	require.Equal(t, n, expectLen)
	f.Close()

	checkDownloadResponse := func(t *testing.T, r *http.Response, expectedFilename string) {
		require.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))
		require.Equal(t, fmt.Sprintf(`attachment;filename="%s"`, expectedFilename), r.Header.Get("Content-Disposition"))
		require.NotZero(t, r.ContentLength)
		require.Equal(t, expectLen, int(r.ContentLength))
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, expectLen, len(b))
		require.Equal(t, expectBytes, b)
	}

	checkSoftwareTitle := func(t *testing.T, title string, source string) uint {
		var id uint
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(context.Background(), q, &id, `SELECT id FROM software_titles WHERE name = ? AND source = ? AND browser = ''`, title, source)
		})
		return id
	}

	checkScriptContentsID := func(t *testing.T, id uint, expectedContents string) {
		var contents string
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(context.Background(), q, &contents, `SELECT contents FROM script_contents WHERE id = ?`, id)
		})
		require.Equal(t, expectedContents, contents)
	}

	checkSoftwareInstaller := func(t *testing.T, payload *fleet.UploadSoftwareInstallerPayload) (installerID uint, titleID uint) {
		var tid uint
		if payload.TeamID != nil {
			tid = *payload.TeamID
		}
		var id uint
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(context.Background(), q, &id, `SELECT id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, tid, payload.Filename)
		})
		require.NotZero(t, id)

		var platform string
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(context.Background(), q, &platform, `SELECT platform FROM software_installers WHERE id = ?`, id)
		})
		require.Equal(t, payload.Platform, "linux")

		meta, err := s.ds.GetSoftwareInstallerMetadataByID(context.Background(), id)
		require.NoError(t, err)

		if payload.TeamID != nil && *payload.TeamID > 0 {
			require.Equal(t, *payload.TeamID, *meta.TeamID)
		} else {
			require.Nil(t, meta.TeamID)
		}

		checkScriptContentsID(t, meta.InstallScriptContentID, payload.InstallScript)

		if payload.PostInstallScript != "" {
			require.NotNil(t, meta.PostInstallScriptContentID)
			checkScriptContentsID(t, *meta.PostInstallScriptContentID, payload.PostInstallScript)
		} else {
			require.Nil(t, meta.PostInstallScriptContentID)
		}

		require.Equal(t, payload.PreInstallQuery, meta.PreInstallQuery)
		require.Equal(t, payload.StorageID, meta.StorageID)
		require.Equal(t, payload.Filename, meta.Name)
		require.Equal(t, payload.Version, meta.Version)
		require.Equal(t, checkSoftwareTitle(t, payload.Title, "deb_packages"), *meta.TitleID)
		require.NotZero(t, meta.UploadedAt)

		return meta.InstallerID, *meta.TitleID
	}

	t.Run("upload no team software installer", func(t *testing.T) {
		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallScript:     "some install script",
			PreInstallQuery:   "some pre install query",
			PostInstallScript: "some post install script",
			Filename:          "ruby.deb",
			// additional fields below are pre-populated so we can re-use the payload later for the test assertions
			Title:     "ruby",
			Version:   "1:2.5.1",
			Source:    "deb_packages",
			StorageID: "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628",
			Platform:  "linux",
		}

		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

		// check the software installer
		_, titleID := checkSoftwareInstaller(t, payload)

		// check activity
		activityData := fmt.Sprintf(`{"software_title": "ruby", "software_package": "ruby.deb", "team_name": null, "team_id": null, "self_service": false, "software_title_id": %d}`, titleID)
		s.lastActivityOfTypeMatches(fleet.ActivityTypeAddedSoftware{}.ActivityName(), activityData, 0)

		// upload again fails
		s.uploadSoftwareInstaller(t, payload, http.StatusConflict, "already exists")

		// orbit-downloading fails with invalid orbit node key
		s.Do("POST", "/api/fleet/orbit/software_install/package?alt=media", orbitDownloadSoftwareInstallerRequest{
			InstallerID:  123,
			OrbitNodeKey: uuid.NewString(),
		}, http.StatusUnauthorized)

		// download the installer
		s.Do("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package?alt=media", titleID), nil, http.StatusBadRequest)

		// delete the installer from nil team fails
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusBadRequest)

		// delete from team 0 succeeds
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent, "team_id", "0")
	})

	t.Run("create team software installer", func(t *testing.T) {
		var createTeamResp teamResponse
		s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{
			Name: t.Name(),
		}, http.StatusOK, &createTeamResp)
		require.NotZero(t, createTeamResp.Team.ID)

		payload := &fleet.UploadSoftwareInstallerPayload{
			TeamID:            &createTeamResp.Team.ID,
			InstallScript:     "another install script",
			PreInstallQuery:   "another pre install query",
			PostInstallScript: "another post install script",
			Filename:          "ruby.deb",
			// additional fields below are pre-populated so we can re-use the payload later for the test assertions
			Title:       "ruby",
			Version:     "1:2.5.1",
			Source:      "deb_packages",
			StorageID:   "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628",
			Platform:    "linux",
			SelfService: true,
		}
		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

		// check the software installer
		installerID, titleID := checkSoftwareInstaller(t, payload)

		// check activity
		activityData := fmt.Sprintf(
			`{"software_title": "ruby", "software_package": "ruby.deb", "team_name": "%s", "team_id": %d, "self_service": true, "software_title_id": %d}`,
			createTeamResp.Team.Name,
			createTeamResp.Team.ID,
			titleID,
		)
		s.lastActivityOfTypeMatches(fleet.ActivityTypeAddedSoftware{}.ActivityName(), activityData, 0)

		// upload again fails
		s.uploadSoftwareInstaller(t, payload, http.StatusConflict, "already exists")

		// download the installer
		r := s.Do("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package?alt=media", titleID), nil, http.StatusOK, "team_id", fmt.Sprintf("%d", *payload.TeamID))
		checkDownloadResponse(t, r, payload.Filename)

		// download the installer by getting token first
		tokenResp := getSoftwareInstallerTokenResponse{}
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package/token?alt=media", titleID), nil, http.StatusOK,
			&tokenResp, "team_id", fmt.Sprintf("%d", *payload.TeamID))
		require.NotEmpty(t, tokenResp.Token)
		r = s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package/token/%s", titleID, tokenResp.Token), nil,
			http.StatusOK)
		checkDownloadResponse(t, r, payload.Filename)

		// downloading a second time using the same token should fail
		_ = s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package/token/%s", titleID, tokenResp.Token), nil,
			http.StatusForbidden)

		// alt != media should fail
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package/token?alt=bozo", titleID), nil,
			http.StatusUnprocessableEntity,
			&tokenResp, "team_id", fmt.Sprintf("%d", *payload.TeamID))

		// missing team_id should fail
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package/token?alt=media", titleID), nil,
			http.StatusBadRequest,
			&tokenResp)

		// create an orbit host that is not in the team
		hostNotInTeam := createOrbitEnrolledHost(t, "windows", "orbit-host-no-team", s.ds)
		// downloading installer doesn't work if the host doesn't have a pending install request
		s.Do("POST", "/api/fleet/orbit/software_install/package?alt=media", orbitDownloadSoftwareInstallerRequest{
			InstallerID:  installerID,
			OrbitNodeKey: *hostNotInTeam.OrbitNodeKey,
		}, http.StatusForbidden)

		// create an orbit host, assign to team
		hostInTeam := createOrbitEnrolledHost(t, "linux", "orbit-host-team", s.ds)
		require.NoError(t, s.ds.AddHostsToTeam(context.Background(), &createTeamResp.Team.ID, []uint{hostInTeam.ID}))

		// Create a software installation request
		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", hostInTeam.ID, titleID), installSoftwareRequest{}, http.StatusAccepted)

		// requesting download with alt != media fails
		r = s.Do("POST", "/api/fleet/orbit/software_install/package?alt=FOOBAR", orbitDownloadSoftwareInstallerRequest{
			InstallerID:  installerID,
			OrbitNodeKey: *hostInTeam.OrbitNodeKey,
		}, http.StatusBadRequest)
		errMsg := extractServerErrorText(r.Body)
		require.Contains(t, errMsg, "only alt=media is supported")

		// valid download
		r = s.Do("POST", "/api/fleet/orbit/software_install/package?alt=media", orbitDownloadSoftwareInstallerRequest{
			InstallerID:  installerID,
			OrbitNodeKey: *hostInTeam.OrbitNodeKey,
		}, http.StatusOK)
		checkDownloadResponse(t, r, payload.Filename)

		// Get execution ID, normally comes from orbit config
		var installUUID string
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(context.Background(), q, &installUUID, "SELECT execution_id FROM host_software_installs WHERE host_id = ? AND install_script_exit_code IS NULL", hostInTeam.ID)
		})

		// Installation complete, host no longer has access to software
		s.Do("POST", "/api/fleet/orbit/software_install/result", orbitPostSoftwareInstallResultRequest{
			OrbitNodeKey: *hostInTeam.OrbitNodeKey,
			HostSoftwareInstallResultPayload: &fleet.HostSoftwareInstallResultPayload{
				HostID:                hostInTeam.ID,
				InstallUUID:           installUUID,
				InstallScriptExitCode: ptr.Int(0),
				InstallScriptOutput:   ptr.String("done"),
			},
		}, http.StatusNoContent)

		_ = s.Do("POST", "/api/fleet/orbit/software_install/package?alt=media", orbitDownloadSoftwareInstallerRequest{
			InstallerID:  installerID,
			OrbitNodeKey: *hostInTeam.OrbitNodeKey,
		}, http.StatusForbidden)

		// delete the installer
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent, "team_id", fmt.Sprintf("%d", *payload.TeamID))

		// check activity
		s.lastActivityOfTypeMatches(fleet.ActivityTypeDeletedSoftware{}.ActivityName(), fmt.Sprintf(`{"software_title": "ruby", "software_package": "ruby.deb", "team_name": "%s", "team_id": %d, "self_service": true}`, createTeamResp.Team.Name, createTeamResp.Team.ID), 0)

		// download the installer, not found anymore
		s.Do("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package?alt=media", titleID), nil, http.StatusNotFound, "team_id", fmt.Sprintf("%d", *payload.TeamID))
	})

	t.Run("create team 0 software installer", func(t *testing.T) {
		payload := &fleet.UploadSoftwareInstallerPayload{
			TeamID:            ptr.Uint(0),
			InstallScript:     "another install script",
			PreInstallQuery:   "another pre install query",
			PostInstallScript: "another post install script",
			Filename:          "ruby.deb",
			// additional fields below are pre-populated so we can re-use the payload later for the test assertions
			Title:       "ruby",
			Version:     "1:2.5.1",
			Source:      "deb_packages",
			StorageID:   "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628",
			Platform:    "linux",
			SelfService: true,
		}
		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

		// check the software installer
		installerID, titleID := checkSoftwareInstaller(t, payload)

		// check activity
		s.lastActivityOfTypeMatches(fleet.ActivityTypeAddedSoftware{}.ActivityName(),
			fmt.Sprintf(`{"software_title": "ruby", "software_package": "ruby.deb", "team_name": null, "team_id": 0, "self_service": true, "software_title_id": %d}`, titleID), 0)

		// upload again fails
		s.uploadSoftwareInstaller(t, payload, http.StatusConflict, "already exists")

		// download the installer
		r := s.Do("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package?alt=media", titleID), nil, http.StatusOK, "team_id", fmt.Sprintf("%d", 0))
		checkDownloadResponse(t, r, payload.Filename)

		// create an orbit host that is not in the team
		hostNotInTeam := createOrbitEnrolledHost(t, "windows", "orbit-host-no-team", s.ds)
		// downloading installer fails because there's no install request
		s.Do("POST", "/api/fleet/orbit/software_install/package?alt=media", orbitDownloadSoftwareInstallerRequest{
			InstallerID:  installerID,
			OrbitNodeKey: *hostNotInTeam.OrbitNodeKey,
		}, http.StatusForbidden)

		// create an orbit host, assign to team
		hostInTeam := createOrbitEnrolledHost(t, "linux", "orbit-host-team", s.ds)

		// requesting download with alt != media fails
		r = s.Do("POST", "/api/fleet/orbit/software_install/package?alt=FOOBAR", orbitDownloadSoftwareInstallerRequest{
			InstallerID:  installerID,
			OrbitNodeKey: *hostInTeam.OrbitNodeKey,
		}, http.StatusBadRequest)
		errMsg := extractServerErrorText(r.Body)
		require.Contains(t, errMsg, "only alt=media is supported")

		// Create a software installation request
		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", hostInTeam.ID, titleID), installSoftwareRequest{}, http.StatusAccepted)

		// valid download
		r = s.Do("POST", "/api/fleet/orbit/software_install/package?alt=media", orbitDownloadSoftwareInstallerRequest{
			InstallerID:  installerID,
			OrbitNodeKey: *hostInTeam.OrbitNodeKey,
		}, http.StatusOK)
		checkDownloadResponse(t, r, payload.Filename)

		// Get execution ID, normally comes from orbit config
		var installUUID string
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(context.Background(), q, &installUUID, "SELECT execution_id FROM host_software_installs WHERE host_id = ? AND install_script_exit_code IS NULL", hostInTeam.ID)
		})

		// Installation complete, host no longer has access to software
		s.Do("POST", "/api/fleet/orbit/software_install/result", orbitPostSoftwareInstallResultRequest{
			OrbitNodeKey: *hostInTeam.OrbitNodeKey,
			HostSoftwareInstallResultPayload: &fleet.HostSoftwareInstallResultPayload{
				HostID:                hostInTeam.ID,
				InstallUUID:           installUUID,
				InstallScriptExitCode: ptr.Int(0),
				InstallScriptOutput:   ptr.String("done"),
			},
		}, http.StatusNoContent)

		_ = s.Do("POST", "/api/fleet/orbit/software_install/package?alt=media", orbitDownloadSoftwareInstallerRequest{
			InstallerID:  installerID,
			OrbitNodeKey: *hostInTeam.OrbitNodeKey,
		}, http.StatusForbidden)

		// delete the installer
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent, "team_id", "0")

		// check activity
		s.lastActivityOfTypeMatches(fleet.ActivityTypeDeletedSoftware{}.ActivityName(),
			`{"software_title": "ruby", "software_package": "ruby.deb", "team_name": null, "team_id": 0, "self_service": true}`, 0)

		// download the installer, not found anymore
		s.Do("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package?alt=media", titleID), nil, http.StatusNotFound, "team_id", fmt.Sprintf("%d", 0))
	})

	t.Run("uninstall migration for software installer", func(t *testing.T) {
		var createTeamResp teamResponse
		s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{
			Name: t.Name(),
		}, http.StatusOK, &createTeamResp)
		require.NotZero(t, createTeamResp.Team.ID)

		payload := &fleet.UploadSoftwareInstallerPayload{
			TeamID:          &createTeamResp.Team.ID,
			InstallScript:   "another install script",
			UninstallScript: "exit 1",
			Filename:        "ruby.deb",
			// additional fields below are pre-populated so we can re-use the payload later for the test assertions
			Title:     "ruby",
			Version:   "1:2.5.1",
			Source:    "deb_packages",
			StorageID: "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628",
			Platform:  "linux",
		}
		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

		logger := kitlog.NewLogfmtLogger(os.Stderr)

		// Run the migration when nothing is to be done
		err = eeservice.UninstallSoftwareMigration(context.Background(), s.ds, s.softwareInstallStore, logger)
		require.NoError(t, err)

		// check the software installer
		installerID, titleID := checkSoftwareInstaller(t, payload)

		var origPackageIDs string
		var origExtension string
		// Update DB by clearing package id and tweaking extension
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			if err := sqlx.GetContext(context.Background(), q, &origPackageIDs, `SELECT package_ids FROM software_installers WHERE id = ?`,
				installerID); err != nil {
				return err
			}
			require.NotEmpty(t, origPackageIDs)

			if err := sqlx.GetContext(context.Background(), q, &origExtension, `SELECT extension FROM software_installers WHERE id = ?`,
				installerID); err != nil {
				return err
			}
			require.NotEmpty(t, origExtension)

			if _, err = q.ExecContext(context.Background(), `UPDATE software_installers SET package_ids = '', extension = 'rb' WHERE id = ?`,
				installerID); err != nil {
				return err
			}
			return nil
		})

		// Check title to make it works without package id
		respTitle := getSoftwareTitleResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &respTitle, "team_id",
			fmt.Sprintf("%d", createTeamResp.Team.ID))
		require.NotNil(t, respTitle.SoftwareTitle.SoftwarePackage)
		assert.Equal(t, "another install script", respTitle.SoftwareTitle.SoftwarePackage.InstallScript)
		assert.Equal(t, "exit 1", respTitle.SoftwareTitle.SoftwarePackage.UninstallScript)

		// Run the migration
		err = eeservice.UninstallSoftwareMigration(context.Background(), s.ds, s.softwareInstallStore, logger)
		require.NoError(t, err)

		// Check package ID and extension
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			var packageIDs string
			if err := sqlx.GetContext(context.Background(), q, &packageIDs, `SELECT package_ids FROM software_installers WHERE id = ?`,
				installerID); err != nil {
				return err
			}
			assert.Equal(t, origPackageIDs, packageIDs)

			var extension string
			if err := sqlx.GetContext(context.Background(), q, &extension, `SELECT extension FROM software_installers WHERE id = ?`,
				installerID); err != nil {
				return err
			}
			assert.Equal(t, origExtension, extension)

			return nil
		})

		// Check uninstall script
		uninstallScript := file.GetUninstallScript("deb")
		uninstallScript = strings.ReplaceAll(uninstallScript, "$PACKAGE_ID", "\"ruby\"")
		respTitle = getSoftwareTitleResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &respTitle, "team_id",
			fmt.Sprintf("%d", createTeamResp.Team.ID))
		require.NotNil(t, respTitle.SoftwareTitle.SoftwarePackage)
		assert.Equal(t, "another install script", respTitle.SoftwareTitle.SoftwarePackage.InstallScript)
		assert.Equal(t, uninstallScript, respTitle.SoftwareTitle.SoftwarePackage.UninstallScript)

		// Running the migration again causes no issues.
		err = eeservice.UninstallSoftwareMigration(context.Background(), s.ds, s.softwareInstallStore, logger)
		require.NoError(t, err)

		// delete the installer
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent,
			"team_id", fmt.Sprintf("%d", *payload.TeamID))
	})
}

func (s *integrationEnterpriseTestSuite) TestApplyTeamsSoftwareConfig() {
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

	// apply with software
	// must not use applyTeamSpecsRequest and marshal it as JSON, as it will set
	// all keys to their zerovalue, and some are only valid with mdm enabled.
	teamSpecs := map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"software": map[string]any{
					"packages": []map[string]any{
						{
							"url":          "http://foo.com",
							"self_service": true,
							"install_script": map[string]string{
								"path": "./foo/install-script.sh",
							},
							"post_install_script": map[string]string{
								"path": "./foo/post-install-script.sh",
							},
							"pre_install_query": map[string]string{
								"path": "./foo/query.yaml",
							},
						},
						{
							"url": "http://bar.com",
							"install_script": map[string]string{
								"path": "./bar/install-script.sh",
							},
							"post_install_script": map[string]string{
								"path": "./bar/post-install-script.sh",
							},
							"pre_install_query": map[string]string{
								"path": "./bar/query.yaml",
							},
						},
					},
					"app_store_apps": []map[string]any{
						{
							"app_store_id": "1234",
						},
						{
							"app_store_id": "5678",
						},
					},
				},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	wantSoftwarePackages := []fleet.SoftwarePackageSpec{
		{
			URL:               "http://foo.com",
			SelfService:       true,
			InstallScript:     fleet.TeamSpecSoftwareAsset{Path: "./foo/install-script.sh"},
			PostInstallScript: fleet.TeamSpecSoftwareAsset{Path: "./foo/post-install-script.sh"},
			PreInstallQuery:   fleet.TeamSpecSoftwareAsset{Path: "./foo/query.yaml"},
		},
		{
			URL:               "http://bar.com",
			SelfService:       false,
			InstallScript:     fleet.TeamSpecSoftwareAsset{Path: "./bar/install-script.sh"},
			PostInstallScript: fleet.TeamSpecSoftwareAsset{Path: "./bar/post-install-script.sh"},
			PreInstallQuery:   fleet.TeamSpecSoftwareAsset{Path: "./bar/query.yaml"},
		},
	}
	wantAppStoreApps := []fleet.TeamSpecAppStoreApp{
		{
			AppStoreID: "1234",
		},
		{
			AppStoreID: "5678",
		},
	}

	// retrieving the team returns the software
	var teamResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, wantSoftwarePackages, teamResp.Team.Config.Software.Packages.Value)

	// apply without custom software specified, should not replace existing software
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, wantSoftwarePackages, teamResp.Team.Config.Software.Packages.Value)
	require.Equal(t, wantAppStoreApps, teamResp.Team.Config.Software.AppStoreApps.Value)

	// apply with explicitly empty custom software would clear the existing
	// software, but dry-run
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"software": map[string]any{
					"packages": nil,
				},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "dry_run", "true")
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, wantSoftwarePackages, teamResp.Team.Config.Software.Packages.Value)

	// apply with explicitly empty custom app store apps would clear the existing
	// software, but dry-run
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"software": map[string]any{
					"app_store_apps": nil,
				},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "dry_run", "true")
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, wantAppStoreApps, teamResp.Team.Config.Software.AppStoreApps.Value)

	// apply with empty top-level software field, should not clear packages
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":     teamName,
				"software": nil,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, wantSoftwarePackages, teamResp.Team.Config.Software.Packages.Value)
	require.Equal(t, wantAppStoreApps, teamResp.Team.Config.Software.AppStoreApps.Value)

	// apply with explicitly empty software packages clears the existing software, but not apps
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"software": map[string]any{
					"packages": nil,
				},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Empty(t, teamResp.Team.Config.Software.Packages.Value)
	require.Equal(t, wantAppStoreApps, teamResp.Team.Config.Software.AppStoreApps.Value)

	// apply with explicitly empty software apps clears the existing apps
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"software": map[string]any{
					"app_store_apps": nil,
				},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Empty(t, teamResp.Team.Config.Software.Packages.Value)
	require.Empty(t, teamResp.Team.Config.Software.AppStoreApps.Value)

	// patch with an invalid array returns an error
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":     teamName,
				"software": []any{"foo", 1},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusBadRequest)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Empty(t, teamResp.Team.Config.Software.Packages.Value)

	// patch with an invalid array returns an error
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":           teamName,
				"app_store_apps": []any{"foo", 1},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusBadRequest)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Empty(t, teamResp.Team.Config.Software.AppStoreApps.Value)
}

func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallers() {
	t := s.T()

	// non-existent team
	s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{}, http.StatusNotFound, "team_name", "foo")

	// create a team
	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)

	// software with a bad URL
	softwareToInstall := []fleet.SoftwareInstallerPayload{
		{URL: "."},
	}
	s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusUnprocessableEntity, "team_name", tm.Name)

	// software with a too big URL
	softwareToInstall = []fleet.SoftwareInstallerPayload{
		{URL: "https://ftp.mozilla.org/" + strings.Repeat("a", 4000-23)},
	}
	s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusUnprocessableEntity, "team_name", tm.Name)

	// create an HTTP server to host the software installer
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ruby.deb" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		file, err := os.Open(filepath.Join("testdata", "software-installers", "ruby.deb"))
		require.NoError(t, err)
		defer file.Close()
		w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
		_, err = io.Copy(w, file)
		require.NoError(t, err)
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	// do a request with a URL that returns a 404.
	softwareToInstall = []fleet.SoftwareInstallerPayload{
		{URL: srv.URL + "/not_found.pkg"},
	}
	var batchResponse batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	message := waitBatchSetSoftwareInstallersFailed(t, s, tm.Name, batchResponse.RequestUUID)
	require.NotEmpty(t, message)
	require.Contains(t, message, fmt.Sprintf("validation failed: software.url Couldn't edit software. URL (\"%s/not_found.pkg\") returned \"Not Found\". Please make sure that URLs are reachable from your Fleet server.", srv.URL))

	// do a request with a valid URL
	rubyURL := srv.URL + "/ruby.deb"
	softwareToInstall = []fleet.SoftwareInstallerPayload{
		{URL: rubyURL},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages := waitBatchSetSoftwareInstallersCompleted(t, s, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, rubyURL, packages[0].URL)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, tm.ID, *packages[0].TeamID)

	softwareToInstallBadSecret := []fleet.SoftwareInstallerPayload{
		{
			URL:           rubyURL,
			InstallScript: "echo $FLEET_SECRET_INVALID",
		},
	}
	resp := s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstallBadSecret}, http.StatusUnprocessableEntity, "team_name", tm.Name)
	errMsg := extractServerErrorText(resp.Body)
	require.Contains(t, errMsg, "$FLEET_SECRET_INVALID")

	softwareToInstallBadSecret[0].InstallScript = ""
	softwareToInstallBadSecret[0].PostInstallScript = "echo $FLEET_SECRET_ALSO_INVALID"
	resp = s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstallBadSecret}, http.StatusUnprocessableEntity, "team_name", tm.Name)
	errMsg = extractServerErrorText(resp.Body)
	require.Contains(t, errMsg, "$FLEET_SECRET_ALSO_INVALID")

	softwareToInstallBadSecret[0].PostInstallScript = ""
	softwareToInstallBadSecret[0].UninstallScript = "echo $FLEET_SECRET_THIRD_INVALID"
	resp = s.Do("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstallBadSecret}, http.StatusUnprocessableEntity, "team_name", tm.Name)
	errMsg = extractServerErrorText(resp.Body)
	require.Contains(t, errMsg, "$FLEET_SECRET_THIRD_INVALID")

	// TODO(roberto): test with a variety of response codes

	// check the application status
	titlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &titlesResp, "available_for_install", "true", "team_id",
		fmt.Sprint(tm.ID))
	require.Equal(t, 1, titlesResp.Count)
	require.Len(t, titlesResp.SoftwareTitles, 1)
	// Check that the URL is set to software installers uploaded via batch.
	require.NotNil(t, titlesResp.SoftwareTitles[0].SoftwarePackage.PackageURL)
	require.Equal(t, rubyURL, *titlesResp.SoftwareTitles[0].SoftwarePackage.PackageURL)

	// check that platform is set when the installer is created
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var platform string
		if err := sqlx.GetContext(context.Background(), q, &platform, `SELECT platform FROM software_installers WHERE title_id= ? AND team_id = ?`, titlesResp.SoftwareTitles[0].ID, tm.ID); err != nil {
			return err
		}
		require.Equal(t, "linux", platform)
		return nil
	})

	// same payload doesn't modify anything
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, s, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, rubyURL, packages[0].URL)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, tm.ID, *packages[0].TeamID)
	newTitlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &newTitlesResp, "available_for_install", "true", "team_id",
		fmt.Sprint(tm.ID))
	require.Equal(t, titlesResp, newTitlesResp)

	// setting self-service to true updates the software title metadata
	softwareToInstall[0].SelfService = true
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, s, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, rubyURL, packages[0].URL)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, tm.ID, *packages[0].TeamID)
	newTitlesResp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &newTitlesResp, "available_for_install", "true", "team_id",
		fmt.Sprint(tm.ID))
	titlesResp.SoftwareTitles[0].SoftwarePackage.SelfService = ptr.Bool(true)
	require.Equal(t, titlesResp, newTitlesResp)

	// empty payload cleans the software items
	softwareToInstall = []fleet.SoftwareInstallerPayload{}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, s, tm.Name, batchResponse.RequestUUID)
	require.Empty(t, packages)
	titlesResp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &titlesResp, "available_for_install", "true", "team_id",
		fmt.Sprint(tm.ID))
	require.Equal(t, 0, titlesResp.Count)
	require.Len(t, titlesResp.SoftwareTitles, 0)

	//////////////////////////
	// Do a request with a valid URL with no team
	//////////////////////////
	softwareToInstall = []fleet.SoftwareInstallerPayload{
		{URL: rubyURL},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse)
	packages = waitBatchSetSoftwareInstallersCompleted(t, s, "", batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, rubyURL, packages[0].URL)
	require.Nil(t, packages[0].TeamID)

	// check the application status on team 0
	titlesResp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &titlesResp, "available_for_install", "true", "team_id", strconv.Itoa(int(0)))
	require.Equal(t, 1, titlesResp.Count)
	require.Len(t, titlesResp.SoftwareTitles, 1)

	// same payload doesn't modify anything
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse)
	packages = waitBatchSetSoftwareInstallersCompleted(t, s, "", batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, rubyURL, packages[0].URL)
	require.Nil(t, packages[0].TeamID)
	newTitlesResp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &newTitlesResp, "available_for_install", "true", "team_id", strconv.Itoa(int(0)))
	require.Equal(t, titlesResp, newTitlesResp)

	// setting self-service to true updates the software title metadata
	softwareToInstall[0].SelfService = true
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse)
	packages = waitBatchSetSoftwareInstallersCompleted(t, s, "", batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.Equal(t, rubyURL, packages[0].URL)
	require.Nil(t, packages[0].TeamID)
	newTitlesResp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &newTitlesResp, "available_for_install", "true", "team_id", strconv.Itoa(int(0)))
	titlesResp.SoftwareTitles[0].SoftwarePackage.SelfService = ptr.Bool(true)
	require.Equal(t, titlesResp, newTitlesResp)

	// empty payload cleans the software items
	softwareToInstall = []fleet.SoftwareInstallerPayload{}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse)
	packages = waitBatchSetSoftwareInstallersCompleted(t, s, "", batchResponse.RequestUUID)
	require.Empty(t, packages)
	titlesResp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &titlesResp, "available_for_install", "true", "team_id", strconv.Itoa(int(0)))
	require.Equal(t, 0, titlesResp.Count)
	require.Len(t, titlesResp.SoftwareTitles, 0)
}

func waitBatchSetSoftwareInstallersCompleted(t *testing.T, s *integrationEnterpriseTestSuite, teamName string, requestUUID string) []fleet.SoftwarePackageResponse {
	timeout := time.After(1 * time.Minute)
	for {
		var batchResultResponse batchSetSoftwareInstallersResultResponse
		s.DoJSON("GET", "/api/latest/fleet/software/batch/"+requestUUID, nil, http.StatusOK, &batchResultResponse, "team_name", teamName)
		if batchResultResponse.Status == fleet.BatchSetSoftwareInstallersStatusCompleted {
			return batchResultResponse.Packages
		}
		select {
		case <-timeout:
			t.Fatalf("timeout: %s, %s", teamName, requestUUID)
		case <-time.After(500 * time.Millisecond):
			// OK, continue
		}
	}
}

func waitBatchSetSoftwareInstallersFailed(t *testing.T, s *integrationEnterpriseTestSuite, teamName string, requestUUID string) string {
	timeout := time.After(1 * time.Minute)
	for {
		var batchResultResponse batchSetSoftwareInstallersResultResponse
		s.DoJSON("GET", "/api/latest/fleet/software/batch/"+requestUUID, nil, http.StatusOK, &batchResultResponse, "team_name", teamName)
		if batchResultResponse.Status == fleet.BatchSetSoftwareInstallersStatusFailed {
			require.Empty(t, batchResultResponse.Packages)
			return batchResultResponse.Message
		}
		select {
		case <-timeout:
			t.Fatalf("timeout: %s, %s", teamName, requestUUID)
		case <-time.After(500 * time.Millisecond):
			// OK, continue
		}
	}
}

func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallersSideEffects() {
	t := s.T()

	// create a team
	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)

	// create an HTTP server to host the software installer
	trailer := ""
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open(filepath.Join("testdata", "software-installers", "ruby.deb"))
		require.NoError(t, err)
		defer file.Close()
		w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
		_, err = io.Copy(w, file)
		require.NoError(t, err)
		_, err = w.Write([]byte(trailer))
		require.NoError(t, err)
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	// set up software to install
	softwareToInstall := []fleet.SoftwareInstallerPayload{
		{URL: srv.URL},
	}
	var batchResponse batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages := waitBatchSetSoftwareInstallersCompleted(t, s, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, tm.ID, *packages[0].TeamID)
	require.Equal(t, srv.URL, packages[0].URL)
	titlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &titlesResp, "available_for_install", "true", "team_id",
		fmt.Sprint(tm.ID))
	titleResponse := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", titlesResp.SoftwareTitles[0].ID), nil, http.StatusOK, &titleResponse,
		"team_id", fmt.Sprint(tm.ID))
	uploadedAt := titleResponse.SoftwareTitle.SoftwarePackage.UploadedAt

	// create a host that doesn't have fleetd installed
	h, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + uuid.New().String()),
		NodeKey:         ptr.String(t.Name() + uuid.New().String()),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "linux",
	})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(context.Background(), &tm.ID, []uint{h.ID})
	require.NoError(t, err)
	h.TeamID = &tm.ID

	// host installs fleetd
	orbitKey := setOrbitEnrollment(t, h, s.ds)
	h.OrbitNodeKey = &orbitKey

	// create another host that doesn't have fleetd installed
	h2, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + uuid.New().String()),
		NodeKey:         ptr.String(t.Name() + uuid.New().String()),
		Hostname:        fmt.Sprintf("%sbar.local", t.Name()),
		Platform:        "linux",
	})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(context.Background(), &tm.ID, []uint{h2.ID})
	require.NoError(t, err)
	h2.TeamID = &tm.ID

	// host installs fleetd
	orbitKey2 := setOrbitEnrollment(t, h2, s.ds)
	h2.OrbitNodeKey = &orbitKey2

	// install software
	installResp := installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h.ID, titlesResp.SoftwareTitles[0].ID), nil, http.StatusAccepted, &installResp)

	// Get the install response, should be pending
	getHostSoftwareResp := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &getHostSoftwareResp)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSoftwareResp.Software[0].Status)

	// Switch self-service flag
	softwareToInstall[0].SelfService = true
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, s, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, tm.ID, *packages[0].TeamID)
	require.Equal(t, srv.URL, packages[0].URL)
	newTitlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &newTitlesResp, "available_for_install", "true", "team_id",
		fmt.Sprint(tm.ID))
	require.Equal(t, true, *newTitlesResp.SoftwareTitles[0].SoftwarePackage.SelfService)

	// Install should still be pending
	afterSelfServiceHostResp := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &afterSelfServiceHostResp)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSoftwareResp.Software[0].Status)

	// update pre-install query
	withUpdatedPreinstallQuery := []fleet.SoftwareInstallerPayload{
		{URL: srv.URL, PreInstallQuery: "SELECT * FROM os_version"},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: withUpdatedPreinstallQuery}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, s, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, tm.ID, *packages[0].TeamID)
	require.Equal(t, srv.URL, packages[0].URL)
	titleResponse = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", newTitlesResp.SoftwareTitles[0].ID), nil, http.StatusOK, &titleResponse,
		"team_id", fmt.Sprint(tm.ID))
	require.Equal(t, "SELECT * FROM os_version", titleResponse.SoftwareTitle.SoftwarePackage.PreInstallQuery)
	require.Equal(t, uint(0), titleResponse.SoftwareTitle.SoftwarePackage.Status.PendingInstall)

	// install should no longer be pending
	afterPreinstallHostResp := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &afterPreinstallHostResp)
	require.Nil(t, afterPreinstallHostResp.Software[0].Status)

	// install software fully
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h.ID, titlesResp.SoftwareTitles[0].ID), nil, http.StatusAccepted, &installResp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &getHostSoftwareResp)
	installUUID := getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall.InstallUUID
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "ok",
			"install_script_exit_code": 0,
			"install_script_output": "ok"
		}`, *h.OrbitNodeKey, installUUID)), http.StatusNoContent)

	// ensure install count is updated
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", newTitlesResp.SoftwareTitles[0].ID), nil, http.StatusOK, &titleResponse,
		"team_id", fmt.Sprint(tm.ID))
	require.Equal(t, uint(1), titleResponse.SoftwareTitle.SoftwarePackage.Status.Installed)
	require.Equal(t, uint(0), titleResponse.SoftwareTitle.SoftwarePackage.Status.PendingInstall)

	// install should show as complete
	hostResp := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &hostResp)
	require.Equal(t, fleet.SoftwareInstalled, *hostResp.Software[0].Status)

	// update install script
	withUpdatedInstallScript := []fleet.SoftwareInstallerPayload{
		{URL: srv.URL, InstallScript: "apt install ruby"},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: withUpdatedInstallScript}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, s, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, tm.ID, *packages[0].TeamID)
	require.Equal(t, srv.URL, packages[0].URL)

	// ensure install count is the same, and uploaded_at hasn't changed
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", newTitlesResp.SoftwareTitles[0].ID), nil, http.StatusOK, &titleResponse,
		"team_id", fmt.Sprint(tm.ID))
	require.Equal(t, uint(1), titleResponse.SoftwareTitle.SoftwarePackage.Status.Installed)
	require.Equal(t, uint(0), titleResponse.SoftwareTitle.SoftwarePackage.Status.PendingInstall)
	require.Equal(t, uploadedAt, titleResponse.SoftwareTitle.SoftwarePackage.UploadedAt)

	// install should still show as complete
	hostResp = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &hostResp)
	require.Equal(t, fleet.SoftwareInstalled, *hostResp.Software[0].Status)

	trailer = " " // add a character to the response for the installer HTTP call to ensure the file hashes differently
	// update package
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: withUpdatedInstallScript}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, s, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, tm.ID, *packages[0].TeamID)
	require.Equal(t, srv.URL, packages[0].URL)

	// ensure install count is zeroed and uploaded_at HAS changed
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", newTitlesResp.SoftwareTitles[0].ID), nil, http.StatusOK, &titleResponse,
		"team_id", fmt.Sprint(tm.ID))
	require.Equal(t, uint(0), titleResponse.SoftwareTitle.SoftwarePackage.Status.Installed)
	require.Equal(t, uint(0), titleResponse.SoftwareTitle.SoftwarePackage.Status.PendingInstall)
	require.NotEqual(t, uploadedAt, titleResponse.SoftwareTitle.SoftwarePackage.UploadedAt)

	// install should be nulled out
	hostResp = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &hostResp)
	require.Nil(t, hostResp.Software[0].Status)

	// install details record should still show as installed
	installDetailsResp := getSoftwareInstallResultsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/install/%s/results", installUUID), nil, http.StatusOK, &installDetailsResp)
	require.Equal(t, fleet.SoftwareInstalled, installDetailsResp.Results.Status)

	// queue another install before we delete the installer via batch
	pendingResp := installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h.ID, titlesResp.SoftwareTitles[0].ID), nil, http.StatusAccepted, &pendingResp)

	// install should show as pending
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &afterPreinstallHostResp)
	require.Equal(t, fleet.SoftwareInstallPending, *afterPreinstallHostResp.Software[0].Status)
	installUUID = afterPreinstallHostResp.Software[0].SoftwarePackage.LastInstall.InstallUUID

	// queue an uninstall on another host
	uninstallResp := installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", h2.ID, titlesResp.SoftwareTitles[0].ID), nil, http.StatusAccepted, &uninstallResp)

	// uninstall should show as pending
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h2.ID), nil, http.StatusOK, &afterPreinstallHostResp)
	require.Equal(t, fleet.SoftwareUninstallPending, *afterPreinstallHostResp.Software[0].Status)

	// delete all installers
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: []fleet.SoftwareInstallerPayload{}}, http.StatusAccepted, &batchResponse, "team_name", tm.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, s, tm.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 0)

	// software should no longer exist on either host
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &afterPreinstallHostResp)
	require.Len(t, afterPreinstallHostResp.Software, 0)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h2.ID), nil, http.StatusOK, &afterPreinstallHostResp)
	require.Len(t, afterPreinstallHostResp.Software, 0)

	// pending install record should not exist
	installDetailsResp = getSoftwareInstallResultsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/install/%s/results", installUUID), nil, http.StatusNotFound, &installDetailsResp)
}

func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallersWithPoliciesAssociated() {
	ctx := context.Background()
	t := s.T()

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	policy1Team1, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:  "team1Policy1",
			Query: "SELECT 1;",
		},
	)
	require.NoError(t, err)
	policy2Team2, err := s.ds.NewTeamPolicy(
		ctx, team2.ID, nil, fleet.PolicyPayload{
			Name:  "team2Policy2",
			Query: "SELECT 2;",
		},
	)
	require.NoError(t, err)

	// create an HTTP server to host software installers
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var fileName string
		switch r.URL.Path {
		case "/ruby.deb", "/dummy_installer.pkg":
			fileName = strings.TrimPrefix(r.URL.Path, "/")
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		file, err := os.Open(filepath.Join("testdata", "software-installers", fileName))
		require.NoError(t, err)
		defer file.Close()
		w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
		_, err = io.Copy(w, file)
		require.NoError(t, err)
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	// team1 has ruby.deb
	softwareToInstall := []fleet.SoftwareInstallerPayload{
		{
			URL: srv.URL + "/ruby.deb",
		},
	}
	var batchResponse batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team1.Name)
	packages := waitBatchSetSoftwareInstallersCompleted(t, s, team1.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, team1.ID, *packages[0].TeamID)
	require.Equal(t, srv.URL+"/ruby.deb", packages[0].URL)

	// team2 has dummy_installer.pkg and ruby.deb.
	softwareToInstall = []fleet.SoftwareInstallerPayload{
		{
			URL: srv.URL + "/dummy_installer.pkg",
		},
		{
			URL: srv.URL + "/ruby.deb",
		},
	}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team2.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, s, team2.Name, batchResponse.RequestUUID)
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].URL < packages[j].URL
	})
	require.Len(t, packages, 2)
	require.NotNil(t, packages[0].TitleID)
	require.NotNil(t, packages[0].TeamID)
	require.Equal(t, team2.ID, *packages[0].TeamID)
	require.Equal(t, srv.URL+"/dummy_installer.pkg", packages[0].URL)
	require.NotNil(t, packages[1].TitleID)
	require.NotNil(t, packages[1].TeamID)
	require.Equal(t, team2.ID, *packages[1].TeamID)
	require.Equal(t, srv.URL+"/ruby.deb", packages[1].URL)

	// Associate ruby.deb to policy1Team1.
	resp := listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "ruby",
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	rubyDebTitleID := resp.SoftwareTitles[0].ID
	mtplr := modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: rubyDebTitleID},
		},
	}, http.StatusOK, &mtplr)

	// Associate ruby.deb in team2 to policy2Team2.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team2.ID, policy2Team2.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: rubyDebTitleID},
		},
	}, http.StatusOK, &mtplr)

	// Get rid of all installers in team1.
	softwareToInstall = []fleet.SoftwareInstallerPayload{}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{Software: softwareToInstall}, http.StatusAccepted, &batchResponse, "team_name", team1.Name)
	packages = waitBatchSetSoftwareInstallersCompleted(t, s, team1.Name, batchResponse.RequestUUID)
	require.Len(t, packages, 0)

	// policy1Team1 should not be associated to any installer.
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Nil(t, policy1Team1.SoftwareInstallerID)
	// team1 should be empty.
	titlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &titlesResp, "available_for_install", "true", "team_id",
		fmt.Sprint(team1.ID))
	require.Equal(t, 0, titlesResp.Count)

	// team2 should be untouched.
	titlesResp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/titles", nil, http.StatusOK, &titlesResp, "available_for_install", "true", "team_id",
		fmt.Sprint(team2.ID))
	require.Equal(t, 2, titlesResp.Count)
	require.Len(t, titlesResp.SoftwareTitles, 2)
	require.NotNil(t, titlesResp.SoftwareTitles[0].SoftwarePackage.PackageURL)
	require.Equal(t, srv.URL+"/dummy_installer.pkg", *titlesResp.SoftwareTitles[0].SoftwarePackage.PackageURL)
	require.NotNil(t, titlesResp.SoftwareTitles[1].SoftwarePackage.PackageURL)
	require.Equal(t, srv.URL+"/ruby.deb", *titlesResp.SoftwareTitles[1].SoftwarePackage.PackageURL)

	// policy2Team2 should still be associated to ruby.deb of team2.
	policy2Team2, err = s.ds.Policy(ctx, policy2Team2.ID)
	require.NoError(t, err)
	require.NotNil(t, policy2Team2.SoftwareInstallerID)
}

func (s *integrationEnterpriseTestSuite) TestSoftwareInstallerNewInstallRequestPlatformValidation() {
	t := s.T()

	hostsByPlatform := map[string]*fleet.Host{
		"linux": nil, "darwin": nil, "windows": nil,
	}

	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)

	for platform := range hostsByPlatform {
		h, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   ptr.String(t.Name() + uuid.New().String()),
			NodeKey:         ptr.String(t.Name() + uuid.New().String()),
			Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
			Platform:        platform,
		})
		require.NoError(t, err)
		setOrbitEnrollment(t, h, s.ds)

		err = s.ds.AddHostsToTeam(context.Background(), &tm.ID, []uint{h.ID})
		require.NoError(t, err)

		hostsByPlatform[platform] = h
	}

	softwareTitles := map[string]uint{
		"deb": 0, "msi": 0, "exe": 0, "pkg": 0,
	}

	for kind := range softwareTitles {
		// TODO(roberto): we need real binaries for exe, msi and pkg to
		// perform the API calls.
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			ctx := context.Background()
			installScript := fmt.Sprintf(`echo '%s'`, kind)
			res, err := q.ExecContext(ctx, `INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(md5(?)), ?)`, installScript, installScript)
			if err != nil {
				return err
			}
			scriptContentID, _ := res.LastInsertId()

			uninstallScript := fmt.Sprintf(`echo uninstall '%s'`, kind)
			resUninstall, err := q.ExecContext(ctx, `INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(md5(?)), ?)`,
				uninstallScript, uninstallScript)
			if err != nil {
				return err
			}
			uninstallScriptContentID, _ := resUninstall.LastInsertId()

			res, err = q.ExecContext(ctx, `INSERT INTO software_titles (name, source) VALUES ('foo', ?)`, kind)
			if err != nil {
				return err
			}
			titleID, _ := res.LastInsertId()
			softwareTitles[kind] = uint(titleID) //nolint:gosec // dismiss G115

			_, err = q.ExecContext(ctx, `
			INSERT INTO software_installers
				(title_id, filename, extension, version, install_script_content_id, uninstall_script_content_id, storage_id, team_id, global_or_team_id, pre_install_query)
			VALUES
				(?, ?, ?, ?, ?, ?, unhex(?), ?, ?, ?)`,
				titleID, fmt.Sprintf("installer.%s", kind), kind, "v1.0.0", scriptContentID, uninstallScriptContentID,
				hex.EncodeToString([]byte("test")), tm.ID, tm.ID, "foo")
			return err
		})
	}

	testCases := []struct {
		platform            string
		supportedInstallers []string
	}{
		{"windows", []string{"exe", "msi"}},
		{"darwin", []string{"pkg"}},
		{"linux", []string{"deb"}},
	}

	for _, tc := range testCases {
		for platform, host := range hostsByPlatform {
			for _, kind := range tc.supportedInstallers {
				wantStatus := http.StatusAccepted
				if tc.platform != platform {
					wantStatus = http.StatusBadRequest
				}

				var resp installSoftwareResponse
				s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, softwareTitles[kind]), nil,
					wantStatus, &resp)
			}
		}
	}
}

func (s *integrationEnterpriseTestSuite) TestSoftwareInstallerHostRequests() {
	t := s.T()

	// Enabling software inventory globally, which will be inherited by the team
	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.Features.EnableSoftwareInventory = true
	appConf.ServerSettings.ScriptsDisabled = true // shouldn't stop installs/uninstalls
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)
	time.Sleep(2 * time.Second) // Wait for the app config cache to clear

	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{
		Name: t.Name(),
	}, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	teamID := &createTeamResp.Team.ID

	var resp installSoftwareResponse
	// non-existent host
	s.DoJSON("POST", "/api/latest/fleet/hosts/1/software/1/install", nil, http.StatusNotFound, &resp)
	s.DoJSON("POST", "/api/latest/fleet/hosts/1/software/1/uninstall", nil, http.StatusNotFound, &resp)

	// create a host that doesn't have fleetd installed
	h, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + uuid.New().String()),
		NodeKey:         ptr.String(t.Name() + uuid.New().String()),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "linux",
	})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(context.Background(), teamID, []uint{h.ID})
	require.NoError(t, err)
	h.TeamID = teamID

	// request fails
	resp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/1/install", h.ID), nil, http.StatusUnprocessableEntity, &resp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/1/uninstall", h.ID), nil, http.StatusUnprocessableEntity, &resp)

	// host installs fleetd
	orbitKey := setOrbitEnrollment(t, h, s.ds)
	h.OrbitNodeKey = &orbitKey

	// request fails because of non-existent title
	resp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/1/install", h.ID), nil, http.StatusBadRequest, &resp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/1/uninstall", h.ID), nil, http.StatusBadRequest, &resp)

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "another install script",
		PreInstallQuery:   "another pre install query",
		PostInstallScript: "another post install script",
		UninstallScript:   "another uninstall script with $PACKAGE_ID",
		Filename:          "ruby.deb",
		Title:             "ruby",
		TeamID:            teamID,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
	titleID := getSoftwareTitleID(t, s.ds, payload.Title, "deb_packages")

	// Get title with software installer
	respTitle := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &respTitle, "team_id",
		fmt.Sprintf("%d", *teamID))
	require.NotNil(t, respTitle.SoftwareTitle.SoftwarePackage)
	assert.Equal(t, "another install script", respTitle.SoftwareTitle.SoftwarePackage.InstallScript)
	assert.Equal(t, `another uninstall script with "ruby"`, respTitle.SoftwareTitle.SoftwarePackage.UninstallScript)

	// Upload another package for another platform
	payloadDummy := &fleet.UploadSoftwareInstallerPayload{
		Filename: "dummy_installer.pkg",
		Title:    "DummyApp.app",
		TeamID:   teamID,
	}
	s.uploadSoftwareInstaller(t, payloadDummy, http.StatusOK, "")
	pkgTitleID := getSoftwareTitleID(t, s.ds, payloadDummy.Title, "apps")
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", pkgTitleID), nil, http.StatusOK, &respTitle, "team_id",
		fmt.Sprintf("%d", *teamID))
	require.NotNil(t, respTitle.SoftwareTitle.SoftwarePackage)
	assert.NotEmpty(t, respTitle.SoftwareTitle.SoftwarePackage.InstallScript)
	assert.NotEmpty(t, respTitle.SoftwareTitle.SoftwarePackage.UninstallScript)
	assert.NotContains(t, respTitle.SoftwareTitle.SoftwarePackage.UninstallScript, "$PACKAGE_ID")
	assert.Contains(t, respTitle.SoftwareTitle.SoftwarePackage.UninstallScript, "com.example.dummy")

	// install/uninstall request fails for the wrong platform
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h.ID, pkgTitleID), nil, http.StatusBadRequest, &resp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", h.ID, pkgTitleID), nil, http.StatusBadRequest, &resp)

	// delete software installer which we will not use
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", pkgTitleID), nil, http.StatusNoContent,
		"team_id", fmt.Sprintf("%d", *teamID))

	// install script request succeeds
	resp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h.ID, titleID), nil, http.StatusAccepted, &resp)

	// Get the results, should be pending
	getHostSoftwareResp := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &getHostSoftwareResp)
	require.Len(t, getHostSoftwareResp.Software, 1)
	require.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage)
	require.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall)
	require.NotNil(t, getHostSoftwareResp.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSoftwareResp.Software[0].Status)
	assert.Nil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastUninstall)
	installUUID := getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall.InstallUUID

	gsirr := getSoftwareInstallResultsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/install/%s/results", installUUID), nil, http.StatusOK, &gsirr)
	require.NoError(t, gsirr.Err)
	require.NotNil(t, gsirr.Results)
	results := gsirr.Results
	require.Equal(t, installUUID, results.InstallUUID)
	require.Equal(t, fleet.SoftwareInstallPending, results.Status)

	// Can't install/uninstall if software install is pending
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h.ID, titleID), nil, http.StatusBadRequest, &resp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", h.ID, titleID), nil, http.StatusBadRequest, &resp)

	// create 3 more hosts, will have statuses installed, failed and one with two
	// install requests - one failed and the latest install pending
	h2 := createOrbitEnrolledHost(t, "linux", "host2", s.ds)
	h3 := createOrbitEnrolledHost(t, "linux", "host3", s.ds)
	h4 := createOrbitEnrolledHost(t, "linux", "host4", s.ds)
	err = s.ds.AddHostsToTeam(context.Background(), teamID, []uint{h2.ID, h3.ID, h4.ID})
	require.NoError(t, err)

	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h2.ID, titleID), nil, http.StatusAccepted, &resp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h2.ID), nil, http.StatusOK, &getHostSoftwareResp)
	require.Len(t, getHostSoftwareResp.Software, 1)
	installUUID2 := getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall.InstallUUID
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "ok",
			"install_script_exit_code": 0,
			"install_script_output": "ok"
		}`, *h2.OrbitNodeKey, installUUID2)), http.StatusNoContent)

	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h3.ID, titleID), nil, http.StatusAccepted, &resp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h3.ID), nil, http.StatusOK, &getHostSoftwareResp)
	require.Len(t, getHostSoftwareResp.Software, 1)
	installUUID3 := getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall.InstallUUID
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "ok",
			"install_script_exit_code": 1,
			"install_script_output": "failed"
		}`, *h3.OrbitNodeKey, installUUID3)), http.StatusNoContent)

	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h4.ID, titleID), nil, http.StatusAccepted, &resp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h4.ID), nil, http.StatusOK, &getHostSoftwareResp)
	require.Len(t, getHostSoftwareResp.Software, 1)
	installUUID4a := getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall.InstallUUID
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": ""
		}`, *h4.OrbitNodeKey, installUUID4a)), http.StatusNoContent)

	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h4.ID, titleID), nil, http.StatusAccepted, &resp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h4.ID), nil, http.StatusOK, &getHostSoftwareResp)
	require.Len(t, getHostSoftwareResp.Software, 1)
	installUUID4b := getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall.InstallUUID
	_ = installUUID4b

	// status is reflected in software title response
	titleResp := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp, "team_id",
		fmt.Sprint(*teamID))
	// TODO: confirm expected behavior of the title response host counts (unspecified)
	require.Zero(t, titleResp.SoftwareTitle.HostsCount)
	require.Nil(t, titleResp.SoftwareTitle.CountsUpdatedAt)
	require.NotNil(t, titleResp.SoftwareTitle.SoftwarePackage)
	require.Equal(t, "ruby.deb", titleResp.SoftwareTitle.SoftwarePackage.Name)
	require.NotNil(t, titleResp.SoftwareTitle.SoftwarePackage.Status)
	require.Equal(t, fleet.SoftwareInstallerStatusSummary{
		Installed:      1,
		PendingInstall: 2,
		FailedInstall:  1,
	}, *titleResp.SoftwareTitle.SoftwarePackage.Status)

	// status is reflected in list hosts responses and counts when filtering by software title and status
	// create a label to test also the counts per label with the software install status filter
	var labelResp createLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", &createLabelRequest{fleet.LabelPayload{
		Name:  "test",
		Hosts: []string{h.Hostname, h2.Hostname, h3.Hostname, h4.Hostname},
	}}, http.StatusOK, &labelResp)
	require.NotZero(t, labelResp.Label.ID)

	cases := []struct {
		status  string
		count   int
		hostIDs []uint
	}{
		{"pending", 2, []uint{h.ID, h4.ID}},
		{"failed", 1, []uint{h3.ID}},
		{"installed", 1, []uint{h2.ID}},
	}
	for _, c := range cases {
		t.Run(c.status, func(t *testing.T) {
			var listResp listHostsResponse
			s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", c.status, "team_id",
				fmt.Sprint(*teamID), "software_title_id", fmt.Sprint(titleID))
			require.Len(t, listResp.Hosts, c.count)
			gotIDs := make([]uint, 0, c.count)
			for _, h := range listResp.Hosts {
				gotIDs = append(gotIDs, h.ID)
			}
			require.ElementsMatch(t, c.hostIDs, gotIDs)

			var countResp countHostsResponse
			s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", c.status, "team_id",
				fmt.Sprint(*teamID), "software_title_id", fmt.Sprint(titleID))
			require.Equal(t, c.count, countResp.Count)

			// count with label filter
			countResp = countHostsResponse{}
			s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", c.status, "team_id",
				fmt.Sprint(*teamID), "software_title_id", fmt.Sprint(titleID), "label_id", fmt.Sprint(labelResp.Label.ID))
			require.Equal(t, c.count, countResp.Count)

			listResp = listHostsResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", labelResp.Label.ID), nil, http.StatusOK, &listResp,
				"software_status", c.status, "team_id", fmt.Sprint(*teamID), "software_title_id", fmt.Sprint(titleID))
			require.Len(t, listResp.Hosts, c.count)
			gotIDs = make([]uint, 0, c.count)
			for _, h := range listResp.Hosts {
				gotIDs = append(gotIDs, h.ID)
			}
			require.ElementsMatch(t, c.hostIDs, gotIDs)
		})
	}

	// filter validations
	r := s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, "software_status", "uninstalled")
	require.Contains(t, extractServerErrorText(r.Body), "Invalid software_status")
	r = s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, "software_status", "installed")
	require.Contains(t, extractServerErrorText(r.Body), "Missing software_title_id")
	r = s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, "software_status", "installed", "software_title_id", "1")
	require.Contains(t, extractServerErrorText(r.Body), "Missing team_id")
	r = s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, "software_status", "installed", "team_id", "1")
	require.Contains(t, extractServerErrorText(r.Body), "Missing software_title_id")
	r = s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, "software_status", "installed", "team_id", "1", "software_title_id", "1", "software_version_id", "1")
	require.Contains(t, extractServerErrorText(r.Body), "Invalid parameters. The combination of software_version_id and software_title_id is not allowed.")
	r = s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, "software_status", "installed", "team_id", "1", "software_title_id", "1", "software_id", "1")
	require.Contains(t, extractServerErrorText(r.Body), "Invalid parameters. The combination of software_id and software_title_id is not allowed.")

	// Return installed app with software detail query
	distributedReq := submitDistributedQueryResultsRequestShim{
		NodeKey: *h2.NodeKey,
		Results: map[string]json.RawMessage{
			hostDetailQueryPrefix + "software_linux": json.RawMessage(fmt.Sprintf(
				`[{"name": "%s", "version": "1.0", "type": "Package (deb)",
					"source": "deb_packages", "last_opened_at": "",
					"installed_path": "/bin/ruby"}]`, payload.Title)),
		},
		Statuses: map[string]interface{}{
			hostDistributedQueryPrefix + "software_linux": 0,
		},
		Messages: map[string]string{},
		Stats:    map[string]*fleet.Stats{},
	}
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h2.ID), nil, http.StatusOK, &getHostSoftwareResp)
	require.Len(t, getHostSoftwareResp.Software, 1)
	assert.NotNil(t, getHostSoftwareResp.Software[0].Status)
	assert.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall)
	assert.NotEmpty(t, getHostSoftwareResp.Software[0].InstalledVersions, "Installed versions should exist")

	// Remove the installed app by not returning it
	distributedReq = submitDistributedQueryResultsRequestShim{
		NodeKey: *h2.NodeKey,
		Results: map[string]json.RawMessage{
			hostDetailQueryPrefix + "software_linux": json.RawMessage(`[]`),
		},
		Statuses: map[string]interface{}{
			hostDistributedQueryPrefix + "software_linux": 0,
		},
		Messages: map[string]string{},
		Stats:    map[string]*fleet.Stats{},
	}
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h2.ID), nil, http.StatusOK, &getHostSoftwareResp)
	require.Len(t, getHostSoftwareResp.Software, 1)
	assert.Nil(t, getHostSoftwareResp.Software[0].Status)
	assert.Nil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall)
	assert.Empty(t, getHostSoftwareResp.Software[0].InstalledVersions, "Installed versions should now not exist")

	// Mark original install successful
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "ok",
			"install_script_exit_code": 0,
			"install_script_output": "ok"
		}`, *h.OrbitNodeKey, installUUID)), http.StatusNoContent)

	// Do uninstall
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", h.ID, titleID), nil, http.StatusAccepted, &resp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &getHostSoftwareResp)
	require.Len(t, getHostSoftwareResp.Software, 1)
	assert.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall)
	assert.Equal(t, fleet.SoftwareUninstallPending, *getHostSoftwareResp.Software[0].Status)
	require.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastUninstall)
	uninstallExecutionID := getHostSoftwareResp.Software[0].SoftwarePackage.LastUninstall.ExecutionID

	// Uninstall should show up as a pending activity
	var listUpcomingAct listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", h.ID), nil, http.StatusOK, &listUpcomingAct)
	require.Len(t, listUpcomingAct.Activities, 1)
	assert.Equal(t, fleet.ActivityTypeUninstalledSoftware{}.ActivityName(), listUpcomingAct.Activities[0].Type)
	details := make(map[string]interface{}, 5)
	require.NoError(t, json.Unmarshal(*listUpcomingAct.Activities[0].Details, &details))
	assert.EqualValues(t, fleet.SoftwareUninstallPending, details["status"])

	// Check that status is reflected in software title response
	titleResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp, "team_id",
		fmt.Sprint(*teamID))
	require.NotNil(t, titleResp.SoftwareTitle.SoftwarePackage)
	assert.Equal(t, "ruby.deb", titleResp.SoftwareTitle.SoftwarePackage.Name)
	require.NotNil(t, titleResp.SoftwareTitle.SoftwarePackage.Status)
	assert.Equal(t, fleet.SoftwareInstallerStatusSummary{
		PendingInstall:   1,
		FailedInstall:    1,
		PendingUninstall: 1,
	}, *titleResp.SoftwareTitle.SoftwarePackage.Status)

	// Another install/uninstall cannot be sent once an uninstall is pending
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", h.ID, titleID), nil, http.StatusBadRequest, &resp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", h.ID, titleID), nil, http.StatusBadRequest, &resp)

	// expect uninstall script to be pending
	var orbitResp orbitGetConfigResponse
	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)),
		http.StatusOK, &orbitResp)
	require.Len(t, orbitResp.Notifications.PendingScriptExecutionIDs, 1)

	// Host sends successful uninstall result
	var orbitPostScriptResp orbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *h.OrbitNodeKey,
			uninstallExecutionID)),
		http.StatusOK, &orbitPostScriptResp)

	// Check activity feed
	var activitiesResp listActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", h.ID), nil, http.StatusOK, &activitiesResp, "order_key", "a.id",
		"order_direction", "desc")
	require.NotEmpty(t, activitiesResp.Activities)
	assert.Equal(t, fleet.ActivityTypeUninstalledSoftware{}.ActivityName(), activitiesResp.Activities[0].Type)
	details = make(map[string]interface{}, 5)
	require.NoError(t, json.Unmarshal(*activitiesResp.Activities[0].Details, &details))
	assert.Equal(t, "uninstalled", details["status"])

	// Software should be available for install again
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &getHostSoftwareResp)
	require.Len(t, getHostSoftwareResp.Software, 1)
	assert.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall)
	require.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastUninstall)
	assert.Nil(t, getHostSoftwareResp.Software[0].Status)

	// Uninstall again, but this time with a failed result
	beforeUninstall := time.Now()
	// Since host_script_results does not use fine-grained timestamps yet, we adjust
	beforeUninstall = beforeUninstall.Add(-time.Second)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", h.ID, titleID), nil, http.StatusAccepted, &resp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), nil, http.StatusOK, &getHostSoftwareResp)
	require.Len(t, getHostSoftwareResp.Software, 1)
	assert.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastInstall)
	assert.Equal(t, fleet.SoftwareUninstallPending, *getHostSoftwareResp.Software[0].Status)
	require.NotNil(t, getHostSoftwareResp.Software[0].SoftwarePackage.LastUninstall)
	uninstallExecutionID = getHostSoftwareResp.Software[0].SoftwarePackage.LastUninstall.ExecutionID
	// Host sends failed uninstall result
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 1, "output": "not ok"}`, *h.OrbitNodeKey,
			uninstallExecutionID)),
		http.StatusOK, &orbitPostScriptResp)

	// Check activity feed
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", h.ID), nil, http.StatusOK, &activitiesResp, "order_key", "a.id",
		"order_direction", "desc")
	require.NotEmpty(t, activitiesResp.Activities)
	assert.Equal(t, fleet.ActivityTypeUninstalledSoftware{}.ActivityName(), activitiesResp.Activities[0].Type)
	details = make(map[string]interface{}, 5)
	require.NoError(t, json.Unmarshal(*activitiesResp.Activities[0].Details, &details))
	assert.Equal(t, "failed", details["status"])

	// Access software install/uninstall result after host is deleted
	err = s.ds.DeleteHost(context.Background(), h.ID)
	require.NoError(t, err)

	instResult, err := s.ds.GetSoftwareInstallResults(context.Background(), installUUID)
	require.NoError(t, err)
	require.NotNil(t, instResult.HostDeletedAt)

	gsirr = getSoftwareInstallResultsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/install/%s/results", installUUID), nil, http.StatusOK, &gsirr)
	require.NoError(t, gsirr.Err)
	require.NotNil(t, gsirr.Results)
	results = gsirr.Results
	require.Equal(t, installUUID, results.InstallUUID)
	require.Equal(t, fleet.SoftwareInstalled, results.Status)

	var scriptResultResp getScriptResultResponse
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+uninstallExecutionID, nil, http.StatusOK, &scriptResultResp)
	assert.Equal(t, h.ID, scriptResultResp.HostID)
	assert.NotEmpty(t, scriptResultResp.ScriptContents)
	require.NotNil(t, scriptResultResp.ExitCode)
	assert.EqualValues(t, 1, *scriptResultResp.ExitCode)
	assert.Equal(t, "not ok", scriptResultResp.Output)
	assert.Less(t, beforeUninstall, scriptResultResp.CreatedAt)

	// Enabling software inventory globally, which will be inherited by the team
	appConf.ServerSettings.ScriptsDisabled = false // set back to normal
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)
}

func (s *integrationEnterpriseTestSuite) TestSelfServiceSoftwareInstall() {
	t := s.T()

	host1 := createOrbitEnrolledHost(t, "linux", "", s.ds)
	token := "secret_token"
	createDeviceTokenForHost(t, s.ds, host1.ID, token)

	payloadNoSS := &fleet.UploadSoftwareInstallerPayload{
		PreInstallQuery:   "SELECT 1",
		InstallScript:     "install",
		PostInstallScript: "echo hi",
		Filename:          "ruby.deb",
		Title:             "ruby",
		SelfService:       false,
	}
	s.uploadSoftwareInstaller(t, payloadNoSS, http.StatusOK, "")
	titleIDNoSS := getSoftwareTitleID(t, s.ds, payloadNoSS.Title, "deb_packages")

	payloadSS := &fleet.UploadSoftwareInstallerPayload{
		PreInstallQuery:   "SELECT 2",
		InstallScript:     "install again",
		PostInstallScript: "echo bye",
		Filename:          "emacs.deb",
		Title:             "emacs",
		SelfService:       true,
	}
	s.uploadSoftwareInstaller(t, payloadSS, http.StatusOK, "")
	titleIDSS := getSoftwareTitleID(t, s.ds, payloadSS.Title, "deb_packages")

	// cannot self-install if software installer does not allow it
	res := s.DoRawNoAuth("POST", fmt.Sprintf("/api/v1/fleet/device/%s/software/install/%d", token, titleIDNoSS), nil, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Software title is not available through self-service")

	// request self-install of software that allows it
	s.DoRawNoAuth("POST", fmt.Sprintf("/api/v1/fleet/device/%s/software/install/%d", token, titleIDSS), nil, http.StatusAccepted)

	// it shows up as "self-installed" in the upcoming activities of the host
	var listUpcomingAct listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host1.ID), nil, http.StatusOK, &listUpcomingAct)
	require.Len(t, listUpcomingAct.Activities, 1)
	require.Nil(t, listUpcomingAct.Activities[0].ActorID)

	var details fleet.ActivityTypeInstalledSoftware
	err := json.Unmarshal([]byte(*listUpcomingAct.Activities[0].Details), &details)
	require.NoError(t, err)
	require.Equal(t, host1.ID, details.HostID)
	require.Equal(t, details.SoftwareTitle, payloadSS.Title)
	require.True(t, details.SelfService)
	require.EqualValues(t, fleet.SoftwareInstallPending, details.Status)
	installID := details.InstallUUID

	// record the installation results
	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "1",
			"install_script_exit_code": 0,
			"install_script_output": "ok"
		}`, *host1.OrbitNodeKey, installID)),
		http.StatusNoContent)

	// nothing in upcoming activities anymore
	listUpcomingAct = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host1.ID), nil, http.StatusOK, &listUpcomingAct)
	require.Len(t, listUpcomingAct.Activities, 0)

	// installation shows up in past activities
	var listPastAct listActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host1.ID), nil, http.StatusOK, &listPastAct)
	require.Len(t, listPastAct.Activities, 1)
	require.Nil(t, listPastAct.Activities[0].ActorID)

	err = json.Unmarshal([]byte(*listPastAct.Activities[0].Details), &details)
	require.NoError(t, err)
	require.Equal(t, host1.ID, details.HostID)
	require.Equal(t, details.SoftwareTitle, payloadSS.Title)
	require.True(t, details.SelfService)
	require.EqualValues(t, fleet.SoftwareInstalled, details.Status)
}

func (s *integrationEnterpriseTestSuite) TestHostSoftwareInstallResult() {
	ctx := context.Background()
	t := s.T()

	host := createOrbitEnrolledHost(t, "linux", "", s.ds)

	// Create software installers and corresponding host install requests.
	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "install script",
		PreInstallQuery:   "pre install query",
		PostInstallScript: "post install script",
		Filename:          "ruby.deb",
		Title:             "ruby",
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
	titleID := getSoftwareTitleID(t, s.ds, payload.Title, "deb_packages")
	payload2 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "install script 2",
		PreInstallQuery:   "pre install query 2",
		PostInstallScript: "post install script 2",
		Filename:          "vim.deb",
		Title:             "vim",
	}
	s.uploadSoftwareInstaller(t, payload2, http.StatusOK, "")
	titleID2 := getSoftwareTitleID(t, s.ds, payload2.Title, "deb_packages")
	payload3 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "install script 3",
		PreInstallQuery:   "pre install query 3",
		PostInstallScript: "post install script 3",
		Filename:          "emacs.deb",
		Title:             "emacs",
	}
	s.uploadSoftwareInstaller(t, payload3, http.StatusOK, "")
	titleID3 := getSoftwareTitleID(t, s.ds, payload3.Title, "deb_packages")

	latestInstallUUID := func() string {
		var id string
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id, `SELECT execution_id FROM host_software_installs ORDER BY id DESC LIMIT 1`)
		})
		return id
	}

	// create some install requests for the host
	beforeInstall := time.Now()
	installUUIDs := make([]string, 3)
	titleIDs := []uint{titleID, titleID2, titleID3}
	for i := 0; i < len(installUUIDs); i++ {
		resp := installSoftwareResponse{}
		s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/hosts/%d/software/%d/install", host.ID, titleIDs[i]), nil, http.StatusAccepted, &resp)
		installUUIDs[i] = latestInstallUUID()
	}

	type result struct {
		HostID                  uint
		InstallUUID             string
		Status                  fleet.SoftwareInstallerStatus
		Output                  *string
		PostInstallScriptOutput *string
		PreInstallQueryOutput   *string
	}
	checkResults := func(want result) {
		var resp getSoftwareInstallResultsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/install/%s/results", want.InstallUUID), nil, http.StatusOK, &resp)

		assert.Equal(t, want.HostID, resp.Results.HostID)
		assert.Equal(t, want.InstallUUID, resp.Results.InstallUUID)
		assert.Equal(t, want.Status, resp.Results.Status)
		assert.Equal(t, want.PreInstallQueryOutput, resp.Results.PreInstallQueryOutput)
		assert.Equal(t, want.Output, resp.Results.Output)
		assert.Equal(t, want.PostInstallScriptOutput, resp.Results.PostInstallScriptOutput)
		assert.Less(t, beforeInstall, resp.Results.CreatedAt)
		assert.Greater(t, time.Now(), resp.Results.CreatedAt)
	}

	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "1",
			"install_script_exit_code": 1,
			"install_script_output": "failed"
		}`, *host.OrbitNodeKey, installUUIDs[0])),
		http.StatusNoContent)
	checkResults(result{
		HostID:                host.ID,
		InstallUUID:           installUUIDs[0],
		Status:                fleet.SoftwareInstallFailed,
		PreInstallQueryOutput: ptr.String(fleet.SoftwareInstallerQuerySuccessCopy),
		Output:                ptr.String(fmt.Sprintf(fleet.SoftwareInstallerInstallFailCopy, "failed")),
	})
	wantAct := fleet.ActivityTypeInstalledSoftware{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
		SoftwareTitle:   payload.Title,
		SoftwarePackage: payload.Filename,
		InstallUUID:     installUUIDs[0],
		Status:          string(fleet.SoftwareInstallFailed),
	}
	s.lastActivityMatches(wantAct.ActivityName(), string(jsonMustMarshal(t, wantAct)), 0)

	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": ""
		}`, *host.OrbitNodeKey, installUUIDs[1])),
		http.StatusNoContent)
	checkResults(result{
		HostID:                host.ID,
		InstallUUID:           installUUIDs[1],
		Status:                fleet.SoftwareInstallFailed,
		PreInstallQueryOutput: ptr.String(fleet.SoftwareInstallerQueryFailCopy),
	})
	wantAct = fleet.ActivityTypeInstalledSoftware{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
		SoftwareTitle:   payload2.Title,
		SoftwarePackage: payload2.Filename,
		InstallUUID:     installUUIDs[1],
		Status:          string(fleet.SoftwareInstallFailed),
	}
	s.lastActivityOfTypeMatches(wantAct.ActivityName(), string(jsonMustMarshal(t, wantAct)), 0)

	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "1",
			"install_script_exit_code": 0,
			"install_script_output": "success",
			"post_install_script_exit_code": 0,
			"post_install_script_output": "ok"
		}`, *host.OrbitNodeKey, installUUIDs[2])),
		http.StatusNoContent)
	checkResults(result{
		HostID:                  host.ID,
		InstallUUID:             installUUIDs[2],
		Status:                  fleet.SoftwareInstalled,
		PreInstallQueryOutput:   ptr.String(fleet.SoftwareInstallerQuerySuccessCopy),
		Output:                  ptr.String(fmt.Sprintf(fleet.SoftwareInstallerInstallSuccessCopy, "success")),
		PostInstallScriptOutput: ptr.String(fmt.Sprintf(fleet.SoftwareInstallerPostInstallSuccessCopy, "ok")),
	})
	wantAct = fleet.ActivityTypeInstalledSoftware{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
		SoftwareTitle:   payload3.Title,
		SoftwarePackage: payload3.Filename,
		InstallUUID:     installUUIDs[2],
		Status:          string(fleet.SoftwareInstalled),
	}
	lastActID := s.lastActivityOfTypeMatches(wantAct.ActivityName(), string(jsonMustMarshal(t, wantAct)), 0)

	// non-existing installation uuid
	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": "uuid-no-such",
			"pre_install_condition_output": ""
		}`, *host.OrbitNodeKey)),
		http.StatusNotFound)
	// no new activity created
	s.lastActivityOfTypeMatches(wantAct.ActivityName(), string(jsonMustMarshal(t, wantAct)), lastActID)
}

func (s *integrationEnterpriseTestSuite) TestHostScriptSoftDelete() {
	t := s.T()
	ctx := context.Background()

	// create a host and request a script execution
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "host_soft_delete_team"})
	require.NoError(t, err)
	host := createOrbitEnrolledHost(t, "linux", "", s.ds)
	err = s.ds.AddHostsToTeam(ctx, &tm.ID, []uint{host.ID})
	require.NoError(t, err)

	// create an anonymous script execution request
	var runResp runScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusAccepted, &runResp)
	scriptExecID := runResp.ExecutionID

	// post a script result so that the (past) activity is created
	s.Do("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, scriptExecID)),
		http.StatusOK)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeRanScript{}.ActivityName(),
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": %q, "script_name": "", "script_execution_id": %q, "async": true, "policy_id": null, "policy_name": null}`,
			host.ID, host.DisplayName(), scriptExecID), 0)

	// create a saved script execution request
	var newScriptResp createScriptResponse
	body, headers := generateNewScriptMultipartRequest(t,
		"script1.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprint(tm.ID)}})
	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)
	require.NotZero(t, newScriptResp.ScriptID)
	savedScriptID := newScriptResp.ScriptID

	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: &savedScriptID}, http.StatusAccepted, &runResp)
	savedScriptExecID := runResp.ExecutionID

	// post a script result so that the (past) activity is created
	s.Do("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "saved"}`, *host.OrbitNodeKey, savedScriptExecID)),
		http.StatusOK)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeRanScript{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "script_name": "script1.sh", "script_execution_id": %q, "async": true, "policy_id": null, "policy_name": null}`,
			host.ID, host.DisplayName(), savedScriptExecID), 0)

	// get the anoymous script result details
	var scriptRes getScriptResultResponse
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+scriptExecID, nil, http.StatusOK, &scriptRes)
	require.Equal(t, scriptExecID, scriptRes.ExecutionID)
	require.Equal(t, host.ID, scriptRes.HostID)
	require.Equal(t, "ok", scriptRes.Output)
	require.NotNil(t, scriptRes.ExitCode)
	require.EqualValues(t, 0, *scriptRes.ExitCode)

	// get the saved script result details
	scriptRes = getScriptResultResponse{}
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+savedScriptExecID, nil, http.StatusOK, &scriptRes)
	require.Equal(t, savedScriptExecID, scriptRes.ExecutionID)
	require.Equal(t, host.ID, scriptRes.HostID)
	require.Equal(t, "saved", scriptRes.Output)
	require.NotNil(t, scriptRes.ExitCode)
	require.EqualValues(t, 0, *scriptRes.ExitCode)

	// delete the host
	var deleteResp deleteHostResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &deleteResp)

	// get the anonymous script result details, still works
	scriptRes = getScriptResultResponse{}
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+scriptExecID, nil, http.StatusOK, &scriptRes)
	require.Equal(t, scriptExecID, scriptRes.ExecutionID)
	require.Equal(t, host.ID, scriptRes.HostID)
	require.Equal(t, "ok", scriptRes.Output)
	require.NotNil(t, scriptRes.ExitCode)
	require.EqualValues(t, 0, *scriptRes.ExitCode)

	// get the saved script result details, still works
	scriptRes = getScriptResultResponse{}
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+savedScriptExecID, nil, http.StatusOK, &scriptRes)
	require.Equal(t, savedScriptExecID, scriptRes.ExecutionID)
	require.Equal(t, host.ID, scriptRes.HostID)
	require.Equal(t, "saved", scriptRes.Output)
	require.NotNil(t, scriptRes.ExitCode)
	require.EqualValues(t, 0, *scriptRes.ExitCode)

	// delete the named script
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/scripts/%d", savedScriptID), nil, http.StatusNoContent)

	// get the saved script result details, still works because the saved script
	// is a "soft-reference", when deleted the results become essentially results
	// for an anonymous script (i.e. the script_id FK is "ON DELETE SET NULL").
	scriptRes = getScriptResultResponse{}
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+savedScriptExecID, nil, http.StatusOK, &scriptRes)
	require.Equal(t, savedScriptExecID, scriptRes.ExecutionID)
	require.Equal(t, host.ID, scriptRes.HostID)
	require.Equal(t, "saved", scriptRes.Output)
	require.NotNil(t, scriptRes.ExitCode)
	require.EqualValues(t, 0, *scriptRes.ExitCode)
}

func getSoftwareTitleID(t *testing.T, ds *mysql.Datastore, title, source string) uint {
	var id uint
	mysql.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &id, `SELECT id FROM software_titles WHERE name = ? AND source = ? AND browser = ''`, title, source)
	})
	return id
}

func genDistributedReqWithPolicyResults(host *fleet.Host, policyResults map[uint]*bool) submitDistributedQueryResultsRequestShim {
	var (
		results  = make(map[string]json.RawMessage)
		statuses = make(map[string]interface{})
		messages = make(map[string]string)
	)
	for policyID, policyResult := range policyResults {
		distributedQueryName := hostPolicyQueryPrefix + fmt.Sprint(policyID)
		switch {
		case policyResult == nil:
			results[distributedQueryName] = json.RawMessage(`[]`)
			statuses[distributedQueryName] = 1
			messages[distributedQueryName] = "policy failed execution"
		case *policyResult:
			results[distributedQueryName] = json.RawMessage(`[{"1": "1"}]`)
			statuses[distributedQueryName] = 0
		case !*policyResult:
			results[distributedQueryName] = json.RawMessage(`[]`)
			statuses[distributedQueryName] = 0
		}
	}
	return submitDistributedQueryResultsRequestShim{
		NodeKey:  *host.NodeKey,
		Results:  results,
		Statuses: statuses,
		Messages: messages,
		Stats:    map[string]*fleet.Stats{},
	}
}

func triggerAndWait(ctx context.Context, t *testing.T, ds fleet.Datastore, s *schedule.Schedule, timeout time.Duration) {
	// Following code assumes (for simplicity) only triggered runs.
	stats, err := ds.GetLatestCronStats(ctx, s.Name())
	require.NoError(t, err)
	var previousRunID int
	if len(stats) > 0 {
		previousRunID = stats[0].ID
	}

	_, err = s.Trigger()
	require.NoError(t, err)

	timeoutCh := time.After(timeout)
	for {
		stats, err := ds.GetLatestCronStats(ctx, s.Name())
		require.NoError(t, err)
		if len(stats) > 0 && stats[0].ID > previousRunID && stats[0].Status == fleet.CronStatsStatusCompleted {
			t.Logf("cron %s:%d done", s.Name(), stats[0].ID)
			return
		}
		select {
		case <-timeoutCh:
			t.Logf("timeout waiting for schedule %s to complete", s.Name())
			t.Fail()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func (s *integrationEnterpriseTestSuite) cleanupQuery(queryID uint) {
	var delResp deleteQueryByIDResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", queryID), nil, http.StatusOK, &delResp)
}

func (s *integrationEnterpriseTestSuite) TestAutofillPoliciesAuthTeamUser() {
	t := s.T()
	startMockServer := func(t *testing.T) string {
		// create a test http server
		srv := httptest.NewServer(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "POST" {
						w.WriteHeader(http.StatusMethodNotAllowed)
						return
					}
					switch r.URL.Path {
					case "/ok":
						var body map[string]interface{}
						err := json.NewDecoder(r.Body).Decode(&body)
						if err != nil {
							t.Log(err)
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						_, _ = w.Write([]byte(`{"risks":"description", "whatWillProbablyHappenDuringMaintenance":"resolution"}`))
					default:
						w.WriteHeader(http.StatusNotFound)
					}
				},
			),
		)
		t.Cleanup(srv.Close)
		return srv.URL
	}
	mockUrl := startMockServer(t)
	originalUrl := getHumanInterpretationFromOsquerySqlUrl
	originalTimeout := getHumanInterpretationFromOsquerySqlTimeout
	t.Cleanup(
		func() {
			getHumanInterpretationFromOsquerySqlUrl = originalUrl
			getHumanInterpretationFromOsquerySqlTimeout = originalTimeout
		},
	)

	// Create teams
	team1, err := s.ds.NewTeam(
		context.Background(), &fleet.Team{
			ID:          42,
			Name:        "team1" + t.Name(),
			Description: "desc team1",
		},
	)
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(
		context.Background(), &fleet.Team{
			ID:          43,
			Name:        "team2" + t.Name(),
			Description: "desc team2",
		},
	)
	require.NoError(t, err)

	oldToken := s.token
	t.Cleanup(
		func() {
			s.token = oldToken
		},
	)

	switchUser := func(t *testing.T, role string) {
		password := test.GoodPassword
		email := role + "-testteam@user.com"
		u := &fleet.User{
			Name:       "test team user",
			Email:      email,
			GlobalRole: nil,
			Teams: []fleet.UserTeam{
				{
					Team: *team2,
					Role: fleet.RoleObserver,
				},
				{
					Team: *team1,
					Role: role,
				},
			},
		}
		require.NoError(t, u.SetPassword(password, 10, 10))
		_, err = s.ds.NewUser(context.Background(), u)
		require.NoError(t, err)

		s.token = s.getTestToken(email, password)
	}

	req := autofillPoliciesRequest{
		SQL: "select 1",
	}
	getHumanInterpretationFromOsquerySqlUrl = mockUrl + "/ok"

	tests := []struct {
		role string
		pass bool
	}{
		{role: fleet.RoleAdmin, pass: true},
		{role: fleet.RoleMaintainer, pass: true},
		{role: fleet.RoleGitOps, pass: true},
		{role: fleet.RoleObserver, pass: false},
		{role: fleet.RoleObserverPlus, pass: false},
	}

	for _, tt := range tests {
		t.Run(
			tt.role, func(t *testing.T) {
				switchUser(t, tt.role)
				if tt.pass {
					var res autofillPoliciesResponse
					s.DoJSON("POST", "/api/latest/fleet/autofill/policy", req, http.StatusOK, &res)
					assert.Equal(t, "description", res.Description)
					assert.Equal(t, "resolution", res.Resolution)
				} else {
					_ = s.Do("POST", "/api/latest/fleet/autofill/policy", req, http.StatusForbidden)
				}
			},
		)
	}
}

// 1. software title uploaded doesn't match existing title
// 2. host reports software with the same bundle identifier
// 3. reconciler runs, doesn't create a new title
func (s *integrationEnterpriseTestSuite) TestPKGNewSoftwareTitleFlow() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	err = s.ds.AddHostsToTeam(ctx, &team.ID, []uint{host.ID})
	require.NoError(t, err)

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some install script",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

	resp := listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "homebrew"},
		{Name: "foo", Version: "0.0.3", Source: "homebrew"},
		{Name: "bar", Version: "0.0.4", Source: "apps"},
		{Name: "DummyApp.app", Version: "1.0.0", Source: "apps", BundleIdentifier: "com.example.dummy"},
	}
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))
	require.Len(t, host.Software, 4)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	// still one because the counts didn't update yet
	require.Len(t, resp.SoftwareTitles, 1)

	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, hostsCountTs))
	require.NoError(t, s.ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	require.Len(t, resp.SoftwareTitles, 3)
	require.ElementsMatch(
		t,
		[]string{"foo", "bar", "DummyApp.app"},
		[]string{
			resp.SoftwareTitles[0].Name,
			resp.SoftwareTitles[1].Name,
			resp.SoftwareTitles[2].Name,
		},
	)

	// host reports another version of dummy, but this one has a
	// different name, but same identifier
	software = []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "homebrew"},
		{Name: "foo", Version: "0.0.3", Source: "homebrew"},
		{Name: "bar", Version: "0.0.4", Source: "apps"},
		{Name: "DummyApp.app", Version: "1.0.0", Source: "apps", BundleIdentifier: "com.example.dummy"},
		{Name: "AppDummy.app", Version: "2.0.0", Source: "apps", BundleIdentifier: "com.example.dummy"},
	}
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))
	require.Len(t, host.Software, 5)
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, hostsCountTs))
	require.NoError(t, s.ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))

	// titles are the same
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	require.Len(t, resp.SoftwareTitles, 3)
	require.ElementsMatch(
		t,
		[]string{"foo", "bar", "DummyApp.app"},
		[]string{
			resp.SoftwareTitles[0].Name,
			resp.SoftwareTitles[1].Name,
			resp.SoftwareTitles[2].Name,
		},
	)
}

func (s *integrationEnterpriseTestSuite) TestPKGNoVersion() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some installer script",
		Filename:      "no_version.pkg",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusBadRequest, "Couldn't add. Fleet couldn't read the version from no_version.pkg.")
}

// 1. host reports software
// 2. reconciler runs, creates title
// 3. installer is uploaded, matches existing software title
func (s *integrationEnterpriseTestSuite) TestPKGSoftwareAlreadyReported() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	err = s.ds.AddHostsToTeam(ctx, &team.ID, []uint{host.ID})
	require.NoError(t, err)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "homebrew"},
		{Name: "foo", Version: "0.0.3", Source: "homebrew"},
		{Name: "bar", Version: "0.0.4", Source: "apps"},
		// note: the source is not "apps"
		{Name: "DummyApp.app", Version: "1.0.0", Source: "homebrew", BundleIdentifier: "com.example.dummy"},
	}
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))
	require.Len(t, host.Software, 4)

	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, hostsCountTs))
	require.NoError(t, s.ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))
	resp := listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	require.Len(t, resp.SoftwareTitles, 3)
	require.ElementsMatch(
		t,
		[]string{"foo", "bar", "DummyApp.app"},
		[]string{
			resp.SoftwareTitles[0].Name,
			resp.SoftwareTitles[1].Name,
			resp.SoftwareTitles[2].Name,
		},
	)

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some install script",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	require.Len(t, resp.SoftwareTitles, 3)
	require.ElementsMatch(
		t,
		[]string{"foo", "bar", "DummyApp.app"},
		[]string{
			resp.SoftwareTitles[0].Name,
			resp.SoftwareTitles[1].Name,
			resp.SoftwareTitles[2].Name,
		},
	)
}

// 1. host reports software
// 2. installer is uploaded, matches existing software
// 2. reconciler runs, matches existing software title
func (s *integrationEnterpriseTestSuite) TestPKGSoftwareReconciliation() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	err = s.ds.AddHostsToTeam(ctx, &team.ID, []uint{host.ID})
	require.NoError(t, err)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "homebrew"},
		{Name: "foo", Version: "0.0.3", Source: "homebrew"},
		{Name: "bar", Version: "0.0.4", Source: "apps"},
		// note: the source is not "apps"
		{Name: "DummyApp.app", Version: "1.0.0", Source: "homebrew", BundleIdentifier: "com.example.dummy"},
	}
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))
	require.Len(t, host.Software, 4)

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some install script",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

	resp := listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	// only one title (the uploaded software) because the cron didn't run yet
	require.Len(t, resp.SoftwareTitles, 1)
	require.ElementsMatch(
		t,
		[]string{"DummyApp.app"},
		[]string{resp.SoftwareTitles[0].Name},
	)

	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, hostsCountTs))
	require.NoError(t, s.ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	require.Len(t, resp.SoftwareTitles, 3)
	require.ElementsMatch(
		t,
		[]string{"foo", "bar", "DummyApp.app"},
		[]string{
			resp.SoftwareTitles[0].Name,
			resp.SoftwareTitles[1].Name,
			resp.SoftwareTitles[2].Name,
		},
	)
}

func (s *integrationEnterpriseTestSuite) TestCalendarCallback() {
	ctx := context.Background()
	t := s.T()
	t.Cleanup(func() {
		calendar.ClearMockEvents()
		calendar.ClearMockChannels()
	})
	currentAppCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = s.ds.SaveAppConfig(ctx, currentAppCfg)
		require.NoError(t, err)
	})

	origRecentUpdateDuration := commonCalendar.RecentCalendarUpdateDuration
	commonCalendar.RecentCalendarUpdateDuration = 1 * time.Millisecond
	t.Cleanup(func() {
		commonCalendar.RecentCalendarUpdateDuration = origRecentUpdateDuration
	})

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)

	newHost := func(name string, teamID *uint) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   ptr.String(t.Name() + name),
			NodeKey:         ptr.String(t.Name() + name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:        "darwin",
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}

	host1Team1 := newHost("host1", &team1.ID)
	host2Team1 := newHost("host2", &team1.ID)
	_ = newHost("host5", nil) // global host

	team1Policy1Calendar, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team1Policy1Calendar",
			Query:                 "SELECT 1;",
			CalendarEventsEnabled: true,
		},
	)
	require.NoError(t, err)
	team1Policy2Calendar, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team1Policy2Calendar",
			Query:                 "SELECT 2;",
			CalendarEventsEnabled: true,
		},
	)
	require.NoError(t, err)
	globalPolicy, err := s.ds.NewGlobalPolicy(
		ctx, nil, fleet.PolicyPayload{
			Name:                  "globalPolicy",
			Query:                 "SELECT 5;",
			CalendarEventsEnabled: false,
		},
	)
	require.NoError(t, err)

	genDistributedReqWithPolicyResults := func(host *fleet.Host, policyResults map[uint]*bool) submitDistributedQueryResultsRequestShim {
		var (
			results  = make(map[string]json.RawMessage)
			statuses = make(map[string]interface{})
			messages = make(map[string]string)
		)
		for policyID, policyResult := range policyResults {
			distributedQueryName := hostPolicyQueryPrefix + fmt.Sprint(policyID)
			switch {
			case policyResult == nil:
				results[distributedQueryName] = json.RawMessage(`[]`)
				statuses[distributedQueryName] = 1
				messages[distributedQueryName] = "policy failed execution"
			case *policyResult:
				results[distributedQueryName] = json.RawMessage(`[{"1": "1"}]`)
				statuses[distributedQueryName] = 0
			case !*policyResult:
				results[distributedQueryName] = json.RawMessage(`[]`)
				statuses[distributedQueryName] = 0
			}
		}
		return submitDistributedQueryResultsRequestShim{
			NodeKey:  *host.NodeKey,
			Results:  results,
			Statuses: statuses,
			Messages: messages,
			Stats:    map[string]*fleet.Stats{},
		}
	}

	// host1Team1 is failing a calendar policy and not a non-calendar policy (no results for global).
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: ptr.Bool(false),
			team1Policy2Calendar.ID: ptr.Bool(true),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// host2Team1 is passing the calendar policy but not the non-calendar policy (no results for global).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host2Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: ptr.Bool(true),
			team1Policy2Calendar.ID: ptr.Bool(false),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// Set global configuration for the calendar feature.
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.Integrations.GoogleCalendar = []*fleet.GoogleCalendarIntegration{
		{
			Domain: "example.com",
			ApiKey: map[string]string{
				fleet.GoogleCalendarEmail: calendar.MockEmail,
			},
		},
	}
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	time.Sleep(2 * time.Second) // Wait 2 seconds for the app config cache to clear.

	team1.Config.Integrations.GoogleCalendar = &fleet.TeamGoogleCalendarIntegration{
		Enable:     true,
		WebhookURL: "https://example.com",
	}
	team1, err = s.ds.SaveTeam(ctx, team1)
	require.NoError(t, err)

	// Add email mapping for host1Team1
	const user1Email = "user1@example.com"
	err = s.ds.ReplaceHostDeviceMapping(ctx, host1Team1.ID, []*fleet.HostDeviceMapping{
		{
			HostID: host1Team1.ID,
			Email:  user1Email,
			Source: "google_chrome_profiles",
		},
	}, "google_chrome_profiles")
	require.NoError(t, err)
	assert.Equal(t, 0, calendar.MockChannelsCount())

	// Trigger the calendar cron, global feature enabled, team1 enabled
	// and host1Team1 has a domain email associated.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// An event should be generated for host1Team1
	team1CalendarEvents, err := s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, team1CalendarEvents, 1)
	event := team1CalendarEvents[0]
	require.NotZero(t, event.ID)
	require.Equal(t, user1Email, event.Email)
	require.NotZero(t, event.StartTime)
	require.NotZero(t, event.EndTime)
	require.NotEmpty(t, event.UUID)
	bodyTag := event.GetBodyTag()
	assert.NotEmpty(t, bodyTag)
	assert.Equal(t, 1, calendar.MockChannelsCount())

	// Get channel ID
	type eventDetails struct {
		ChannelID string `json:"channel_id"`
		BodyTag   string `json:"body_tag"`
		ETag      string `json:"etag"`
	}
	var details eventDetails
	err = json.Unmarshal(event.Data, &details)
	require.NoError(t, err)

	// Send a sync command
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+event.UUID, []byte(""), http.StatusOK, map[string]string{
		"X-Goog-Channel-Id":     details.ChannelID,
		"X-Goog-Resource-State": "sync",
	})

	// Send a regular callback with bad channel ID
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+event.UUID, []byte(""), http.StatusForbidden, map[string]string{
		"X-Goog-Channel-Id":     "bad",
		"X-Goog-Resource-State": "exists",
	})

	// Send a regular callback
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+event.UUID, []byte(""), http.StatusOK, map[string]string{
		"X-Goog-Channel-Id":     details.ChannelID,
		"X-Goog-Resource-State": "exists",
	})

	// Delete the event on the calendar
	calendar.ClearMockEvents()

	// Grab the distributed lock for this event
	distributedLock := redis_lock.NewLock(s.redisPool)
	lockValue := uuid.New().String()
	result, err := distributedLock.SetIfNotExist(ctx, commonCalendar.LockKeyPrefix+event.UUID, lockValue, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// This callback should put the event processing in a queue for async processing. It does not start async
	// processing because it assumes another server is handling this webhook, and that server will start
	// async processing.
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+event.UUID, []byte(""), http.StatusOK, map[string]string{
		"X-Goog-Channel-Id":     details.ChannelID,
		"X-Goog-Resource-State": "exists",
	})

	uuids, err := distributedLock.GetSet(ctx, commonCalendar.QueueKey)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{event.UUID}, uuids)
	// The calendar should still be empty since event hasn't processed yet
	assert.Zero(t, len(calendar.ListGoogleMockEvents()))
	// We clear the queue
	assert.NoError(t, distributedLock.RemoveFromSet(ctx, commonCalendar.QueueKey, event.UUID))

	// We release the normal lock, but grab the reserve lock instead
	ok, err := distributedLock.ReleaseLock(ctx, commonCalendar.LockKeyPrefix+event.UUID, lockValue)
	require.NoError(t, err)
	assert.True(t, ok)
	result, err = distributedLock.SetIfNotExist(ctx, commonCalendar.ReservedLockKeyPrefix+event.UUID, lockValue, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// This callback should put the event processing in a queue for async processing, AND start the async processing
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+event.UUID, []byte(""), http.StatusOK, map[string]string{
		"X-Goog-Channel-Id":     details.ChannelID,
		"X-Goog-Resource-State": "exists",
	})

	uuids, err = distributedLock.GetSet(ctx, commonCalendar.QueueKey)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{event.UUID}, uuids)
	// The calendar should still be empty since event hasn't processed yet
	assert.Zero(t, len(calendar.ListGoogleMockEvents()))

	// We grab the normal lock again.
	lockValue2 := uuid.New().String()
	result, err = distributedLock.SetIfNotExist(ctx, commonCalendar.LockKeyPrefix+event.UUID, lockValue2, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	// We release the reserve lock.
	ok, err = distributedLock.ReleaseLock(ctx, commonCalendar.ReservedLockKeyPrefix+event.UUID, lockValue)
	require.NoError(t, err)
	assert.True(t, ok)
	// We release the normal lock.
	ok, err = distributedLock.ReleaseLock(ctx, commonCalendar.LockKeyPrefix+event.UUID, lockValue2)
	require.NoError(t, err)
	assert.True(t, ok)

	done := make(chan struct{})
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
			require.NoError(t, err)
			// Event should be rescheduled on a future date/time
			if len(team1CalendarEvents) == 1 && team1CalendarEvents[0].UUID == event.UUID &&
				team1CalendarEvents[0].StartTime.After(event.StartTime) {
				done <- struct{}{}
				return
			}
		}
	}()
	select {
	case <-done: // All good
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for calendar event processing")
	}

	eventRecreated := team1CalendarEvents[0]
	assert.NotZero(t, eventRecreated.ID)
	assert.Equal(t, user1Email, eventRecreated.Email)
	assert.NotZero(t, eventRecreated.StartTime)
	assert.NotZero(t, eventRecreated.EndTime)
	assert.NotEmpty(t, eventRecreated.UUID)
	assert.NotEqual(t, event.StartTime, eventRecreated.StartTime)
	assert.NotEqual(t, event.EndTime, eventRecreated.EndTime)
	assert.Equal(t, 1, calendar.MockChannelsCount())
	assert.Equal(t, 1, len(calendar.ListGoogleMockEvents()))

	// The previous event UUID should not work anymore, but API returns OK because this is a common occurrence.
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+event.UUID, []byte(""), http.StatusOK, map[string]string{
		"X-Goog-Channel-Id":     details.ChannelID,
		"X-Goog-Resource-State": "exists",
	})

	err = json.Unmarshal(eventRecreated.Data, &details)
	require.NoError(t, err)
	assert.NotEmpty(t, details.BodyTag)
	bodyTag = details.BodyTag

	// New event callback should work
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+eventRecreated.UUID, []byte(""), http.StatusOK,
		map[string]string{
			"X-Goog-Channel-Id":     details.ChannelID,
			"X-Goog-Resource-State": "exists",
		})

	// Update the time of the event
	events := calendar.ListGoogleMockEvents()
	require.Len(t, events, 1)
	for _, e := range events {
		st, err := time.Parse(time.RFC3339, e.Start.DateTime)
		require.NoError(t, err)
		newStartTime := st.Add(5 * time.Minute).Format(time.RFC3339)
		e.Start.DateTime = newStartTime
	}

	// New event callback should cause the time to be updated in the DB
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+eventRecreated.UUID, []byte(""), http.StatusOK,
		map[string]string{
			"X-Goog-Channel-Id":     details.ChannelID,
			"X-Goog-Resource-State": "exists",
		})

	// Check that the time was updated in the DB
	team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, team1CalendarEvents, 1)
	eventUpdated := team1CalendarEvents[0]
	assert.NotZero(t, eventUpdated.ID)
	assert.Equal(t, user1Email, eventUpdated.Email)
	assert.Equal(t, eventRecreated.UUID, eventUpdated.UUID)
	assert.Greater(t, eventUpdated.StartTime, eventRecreated.StartTime)
	assert.Equal(t, eventRecreated.EndTime, eventUpdated.EndTime)
	assert.Equal(t, 1, calendar.MockChannelsCount())
	assert.Equal(t, bodyTag, eventRecreated.GetBodyTag())

	// Change the body contents of event.
	events = calendar.ListGoogleMockEvents()
	require.Len(t, events, 1)
	eTag := "description change etag"
	for _, e := range events {
		e.Etag = eTag
		e.Description = "new description"
	}
	// New event callback should cause Etag to update but Body tag to remain the same
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+eventRecreated.UUID, []byte(""), http.StatusOK,
		map[string]string{
			"X-Goog-Channel-Id":     details.ChannelID,
			"X-Goog-Resource-State": "exists",
		})
	team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, team1CalendarEvents, 1)
	eventDescUpdated := team1CalendarEvents[0]
	err = json.Unmarshal(eventDescUpdated.Data, &details)
	require.NoError(t, err)
	assert.Equal(t, bodyTag, details.BodyTag)
	assert.Equal(t, eTag, details.ETag)

	// Update the time of the event again
	events = calendar.ListGoogleMockEvents()
	require.Len(t, events, 1)
	for _, e := range events {
		st, err := time.Parse(time.RFC3339, e.Start.DateTime)
		require.NoError(t, err)
		newStartTime := st.Add(5 * time.Minute).Format(time.RFC3339)
		e.Start.DateTime = newStartTime
		e.Etag += "1"
	}

	// Grab the lock
	event = eventUpdated
	lockValue = uuid.New().String()
	result, err = distributedLock.SetIfNotExist(ctx, commonCalendar.LockKeyPrefix+event.UUID, lockValue, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		// Update updated_at so the event gets updated (the event is updated regularly)
		_, err := db.ExecContext(ctx,
			`UPDATE calendar_events SET updated_at = DATE_SUB(CURRENT_TIMESTAMP, INTERVAL 25 HOUR) WHERE id = ?`, event.ID)
		return err
	})

	// Trigger the calendar cron async. It should wait for the lock and set reserve lock.
	go triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 10*time.Second)
	done = make(chan struct{})
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			reserveLock, err := distributedLock.Get(ctx, commonCalendar.ReservedLockKeyPrefix+event.UUID)
			require.NoError(t, err)
			if reserveLock != nil {
				done <- struct{}{}
				return
			}
		}
	}()
	select {
	case <-done: // All good
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for cron to set reserve lock")
	}

	// Release the normal lock
	ok, err = distributedLock.ReleaseLock(ctx, commonCalendar.LockKeyPrefix+event.UUID, lockValue)
	require.NoError(t, err)
	assert.True(t, ok)

	// Wait for the event to update
	done = make(chan struct{})
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
			require.NoError(t, err)
			if len(team1CalendarEvents) == 1 && team1CalendarEvents[0].UUID == event.UUID &&
				team1CalendarEvents[0].StartTime.After(event.StartTime) {
				err = json.Unmarshal(team1CalendarEvents[0].Data, &details)
				require.NoError(t, err)
				assert.NotEqual(t, eTag, details.ETag, "ETag should have updated")
				done <- struct{}{}
				return
			}
		}
	}()
	select {
	case <-done: // All good
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for event to update during cron")
	}

	// Delete the event on the calendar
	calendar.ClearMockEvents()

	// Make host1Team1 pass all policies.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: ptr.Bool(true),
			team1Policy2Calendar.ID: ptr.Bool(true),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// We set a flag that event was updated recently. Callback shouldn't do anything since event was updated recently
	_, err = distributedLock.SetIfNotExist(ctx, commonCalendar.RecentUpdateKeyPrefix+event.UUID, commonCalendar.RecentCalendarUpdateValue,
		1000)
	require.NoError(t, err)
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+eventRecreated.UUID, []byte(""), http.StatusOK,
		map[string]string{
			"X-Goog-Channel-Id":     details.ChannelID,
			"X-Goog-Resource-State": "exists",
		})
	assert.Equal(t, 1, calendar.MockChannelsCount())

	// Callback should work, but only clear the callback channel. Event in DB will be deleted on the next cron run.
	_, err = distributedLock.ReleaseLock(ctx, commonCalendar.RecentUpdateKeyPrefix+event.UUID, commonCalendar.RecentCalendarUpdateValue)
	require.NoError(t, err)
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+eventRecreated.UUID, []byte(""), http.StatusOK,
		map[string]string{
			"X-Goog-Channel-Id":     details.ChannelID,
			"X-Goog-Resource-State": "exists",
		})
	assert.Equal(t, 0, calendar.MockChannelsCount())

	previousEvent := team1CalendarEvents[0]
	team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, team1CalendarEvents, 1)
	assert.Equal(t, previousEvent, team1CalendarEvents[0])

	// Trigger calendar should cleanup the events
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)
	assert.Equal(t, 0, calendar.MockChannelsCount())

	// Event should be cleaned up from our database.
	team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	assert.Empty(t, team1CalendarEvents)
}

func (s *integrationEnterpriseTestSuite) TestCalendarEventBodyUpdate() {
	ctx := context.Background()
	t := s.T()
	t.Cleanup(func() {
		calendar.ClearMockEvents()
		calendar.ClearMockChannels()
	})
	currentAppCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = s.ds.SaveAppConfig(ctx, currentAppCfg)
		require.NoError(t, err)
	})
	origRecentUpdateDuration := commonCalendar.RecentCalendarUpdateDuration
	commonCalendar.RecentCalendarUpdateDuration = 1 * time.Millisecond
	t.Cleanup(func() {
		commonCalendar.RecentCalendarUpdateDuration = origRecentUpdateDuration
	})

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)

	newHost := func(name string, teamID *uint) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   ptr.String(t.Name() + name),
			NodeKey:         ptr.String(t.Name() + name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:        "darwin",
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}

	host1Team1 := newHost("host1", &team1.ID)
	host2Team1 := newHost("host2", &team1.ID)
	_ = newHost("host5", nil) // global host

	team1Policy1Calendar, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team1Policy1Calendar",
			Query:                 "SELECT 1;",
			CalendarEventsEnabled: true,
			Description:           "team1Policy1CalendarDescription",
			Resolution:            "team1Policy1CalendarResolution",
		},
	)
	require.NoError(t, err)
	team1Policy2Calendar, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team1Policy2Calendar",
			Query:                 "SELECT 2;",
			CalendarEventsEnabled: true,
			Description:           "team1Policy2CalendarDescription",
			Resolution:            "team1Policy2CalendarResolution",
		},
	)
	require.NoError(t, err)
	globalPolicy, err := s.ds.NewGlobalPolicy(
		ctx, nil, fleet.PolicyPayload{
			Name:                  "globalPolicy",
			Query:                 "SELECT 5;",
			CalendarEventsEnabled: false,
			Description:           "globalPolicyDescription",
			Resolution:            "globalPolicyResolution",
		},
	)
	require.NoError(t, err)

	genDistributedReqWithPolicyResults := func(host *fleet.Host, policyResults map[uint]*bool) submitDistributedQueryResultsRequestShim {
		var (
			results  = make(map[string]json.RawMessage)
			statuses = make(map[string]interface{})
			messages = make(map[string]string)
		)
		for policyID, policyResult := range policyResults {
			distributedQueryName := hostPolicyQueryPrefix + fmt.Sprint(policyID)
			switch {
			case policyResult == nil:
				results[distributedQueryName] = json.RawMessage(`[]`)
				statuses[distributedQueryName] = 1
				messages[distributedQueryName] = "policy failed execution"
			case *policyResult:
				results[distributedQueryName] = json.RawMessage(`[{"1": "1"}]`)
				statuses[distributedQueryName] = 0
			case !*policyResult:
				results[distributedQueryName] = json.RawMessage(`[]`)
				statuses[distributedQueryName] = 0
			}
		}
		return submitDistributedQueryResultsRequestShim{
			NodeKey:  *host.NodeKey,
			Results:  results,
			Statuses: statuses,
			Messages: messages,
			Stats:    map[string]*fleet.Stats{},
		}
	}

	// host1Team1 is failing a calendar policy and not a non-calendar policy (no results for global).
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: ptr.Bool(false),
			team1Policy2Calendar.ID: ptr.Bool(true),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// host2Team1 is passing the calendar policy but not the non-calendar policy (no results for global).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host2Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: ptr.Bool(true),
			team1Policy2Calendar.ID: ptr.Bool(false),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// Set global configuration for the calendar feature.
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.Integrations.GoogleCalendar = []*fleet.GoogleCalendarIntegration{
		{
			Domain: "example.com",
			ApiKey: map[string]string{
				fleet.GoogleCalendarEmail: calendar.MockEmail,
			},
		},
	}
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	time.Sleep(2 * time.Second) // Wait 2 seconds for the app config cache to clear.

	team1.Config.Integrations.GoogleCalendar = &fleet.TeamGoogleCalendarIntegration{
		Enable:     true,
		WebhookURL: "https://example.com",
	}
	team1, err = s.ds.SaveTeam(ctx, team1)
	require.NoError(t, err)

	// Add email mapping for host1Team1
	const user1Email = "user1@example.com"
	err = s.ds.ReplaceHostDeviceMapping(ctx, host1Team1.ID, []*fleet.HostDeviceMapping{
		{
			HostID: host1Team1.ID,
			Email:  user1Email,
			Source: "google_chrome_profiles",
		},
	}, "google_chrome_profiles")
	require.NoError(t, err)
	assert.Equal(t, 0, calendar.MockChannelsCount())

	// Trigger the calendar cron, global feature enabled, team1 enabled
	// and host1Team1 has a domain email associated.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// An event should be generated for host1Team1
	team1CalendarEvents, err := s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, team1CalendarEvents, 1)
	event := team1CalendarEvents[0]
	require.NotZero(t, event.ID)
	require.Equal(t, user1Email, event.Email)
	require.NotZero(t, event.StartTime)
	require.NotZero(t, event.EndTime)
	require.NotEmpty(t, event.UUID)
	assert.Equal(t, 1, calendar.MockChannelsCount())

	getEvents := func() []*googleCalendar.Event {
		calEvents := calendar.ListGoogleMockEvents()
		calEventValues := make([]*googleCalendar.Event, 0, len(calEvents))
		for _, v := range calEvents {
			calEventValues = append(calEventValues, v)
		}
		return calEventValues
	}

	calEvents := getEvents()
	require.Len(t, calEvents, 1)
	assert.Contains(t, calEvents[0].Description, team1Policy1Calendar.Description)
	assert.Contains(t, calEvents[0].Description, *team1Policy1Calendar.Resolution)

	// Remove resolution from policy
	team1Policy1Calendar.Resolution = nil
	require.NoError(t, s.ds.SavePolicy(ctx, team1Policy1Calendar, false, false))
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	calEvents = getEvents()
	require.Len(t, calEvents, 1)
	assert.Contains(t, calEvents[0].Description, fleet.CalendarDefaultDescription)
	assert.Contains(t, calEvents[0].Description, fleet.CalendarDefaultResolution)

	// Put resolution back
	team1Policy1Calendar.Resolution = ptr.String("putResolutionBack")
	require.NoError(t, s.ds.SavePolicy(ctx, team1Policy1Calendar, false, false))
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	calEvents = getEvents()
	require.Len(t, calEvents, 1)
	assert.Contains(t, calEvents[0].Description, team1Policy1Calendar.Description)
	assert.Contains(t, calEvents[0].Description, *team1Policy1Calendar.Resolution)

	// Change resolution
	team1Policy1Calendar.Resolution = ptr.String("changeResolution")
	require.NoError(t, s.ds.SavePolicy(ctx, team1Policy1Calendar, false, false))
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	calEvents = getEvents()
	require.Len(t, calEvents, 1)
	assert.Contains(t, calEvents[0].Description, team1Policy1Calendar.Description)
	assert.Contains(t, calEvents[0].Description, *team1Policy1Calendar.Resolution)

	// Cause another policy to fail
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: ptr.Bool(false),
			team1Policy2Calendar.ID: ptr.Bool(false),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	calEvents = getEvents()
	require.Len(t, calEvents, 1)
	assert.Contains(t, calEvents[0].Description, fleet.CalendarDefaultDescription)
	assert.Contains(t, calEvents[0].Description, fleet.CalendarDefaultResolution)

	// Cause the other policy to pass
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: ptr.Bool(false),
			team1Policy2Calendar.ID: ptr.Bool(true),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	calEvents = getEvents()
	require.Len(t, calEvents, 1)
	assert.Contains(t, calEvents[0].Description, team1Policy1Calendar.Description)
	assert.Contains(t, calEvents[0].Description, *team1Policy1Calendar.Resolution)

	// Get channel ID
	type eventDetails struct {
		ChannelID string `json:"channel_id"`
	}
	var details eventDetails
	err = json.Unmarshal(event.Data, &details)
	require.NoError(t, err)

	// Delete the event on the calendar
	calendar.ClearMockEvents()

	// Send a regular callback
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+event.UUID, []byte(""), http.StatusOK, map[string]string{
		"X-Goog-Channel-Id":     details.ChannelID,
		"X-Goog-Resource-State": "exists",
	})

	// Make sure event body is correct after event was recreated
	calEvents = getEvents()
	require.Len(t, calEvents, 1)
	assert.Contains(t, calEvents[0].Description, team1Policy1Calendar.Description)
	assert.Contains(t, calEvents[0].Description, *team1Policy1Calendar.Resolution)

	// Get the new event
	team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, team1CalendarEvents, 1)
	event = team1CalendarEvents[0]
	err = json.Unmarshal(event.Data, &details)
	require.NoError(t, err)

	// Remove description from policy
	team1Policy1Calendar.Description = " "
	require.NoError(t, s.ds.SavePolicy(ctx, team1Policy1Calendar, false, false))
	// Delete the event on the calendar
	calendar.ClearMockEvents()

	// Send a regular callback
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+event.UUID, []byte(""), http.StatusOK, map[string]string{
		"X-Goog-Channel-Id":     details.ChannelID,
		"X-Goog-Resource-State": "exists",
	})

	// Make sure event body is correct after event was recreated
	calEvents = getEvents()
	require.Len(t, calEvents, 1)
	assert.Contains(t, calEvents[0].Description, fleet.CalendarDefaultDescription)
	assert.Contains(t, calEvents[0].Description, fleet.CalendarDefaultResolution)
}

func (s *integrationEnterpriseTestSuite) TestVPPAppsWithoutMDM() {
	t := s.T()
	ctx := context.Background()

	// Create host
	orbitHost := createOrbitEnrolledHost(t, "darwin", "nonmdm", s.ds)

	test.CreateInsertGlobalVPPToken(t, s.ds)

	// Create team and add host to team
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("Team 1")}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	s.Do("POST", "/api/latest/fleet/hosts/transfer", &addHostsToTeamRequest{HostIDs: []uint{orbitHost.ID}, TeamID: &team.ID}, http.StatusOK)

	// Add an app so that we don't get a not found error
	app, err := s.ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name:             "App " + t.Name(),
		BundleIdentifier: "bid_" + t.Name(),
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "adam_" + t.Name(),
				Platform: fleet.MacOSPlatform,
			},
		},
	}, &team.ID)
	require.NoError(t, err)

	pkgPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some pkg install script",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, pkgPayload, http.StatusOK, "")

	// We don't see VPP, but we do still see the installers
	resp := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", orbitHost.ID), getHostSoftwareRequest{}, http.StatusOK, &resp)
	assert.Len(t, resp.Software, 1)
	assert.NotNil(t, resp.Software[0].SoftwarePackage)
	assert.Nil(t, resp.Software[0].AppStoreApp)

	r := s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", orbitHost.ID, app.TitleID), &installSoftwareRequest{},
		http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "Couldn't install. MDM is turned off. Please make sure that MDM is turned on to install App Store apps.")
}

func (s *integrationEnterpriseTestSuite) TestPolicyAutomationsSoftwareInstallers() {
	t := s.T()
	ctx := context.Background()
	test.CreateInsertGlobalVPPToken(t, s.ds)

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team2"})
	require.NoError(t, err)

	newHost := func(name string, teamID *uint, platform string) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   ptr.String(t.Name() + name),
			NodeKey:         ptr.String(t.Name() + name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:        platform,
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}
	newFleetdHost := func(name string, teamID *uint, platform string) *fleet.Host {
		h := newHost(name, teamID, platform)
		orbitKey := setOrbitEnrollment(t, h, s.ds)
		h.OrbitNodeKey = &orbitKey
		return h
	}

	host0NoTeam := newFleetdHost("host1NoTeam", nil, "darwin")
	host1Team1 := newFleetdHost("host1Team1", &team1.ID, "darwin")
	host2Team1 := newFleetdHost("host2Team1", &team1.ID, "ubuntu")
	host3Team2 := newFleetdHost("host3Team2", &team2.ID, "windows")
	hostVanillaOsquery5Team1 := newHost("hostVanillaOsquery5Team2", &team1.ID, "darwin")

	// Upload dummy_installer.pkg to team1.
	pkgPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some pkg install script",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team1.ID,
	}
	s.uploadSoftwareInstaller(t, pkgPayload, http.StatusOK, "")
	// Get software title ID of the uploaded installer.
	resp := listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "DummyApp.app",
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	dummyInstallerPkgTitleID := resp.SoftwareTitles[0].ID
	var dummyInstallerPkg struct {
		ID        uint   `db:"id"`
		UserID    *uint  `db:"user_id"`
		UserName  string `db:"user_name"`
		UserEmail string `db:"user_email"`
	}
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&dummyInstallerPkg,
			`SELECT id, user_id, user_name, user_email FROM software_installers WHERE global_or_team_id = ? AND filename = ?`,
			team1.ID, "dummy_installer.pkg",
		)
	})
	dummyInstallerPkgInstallerID := dummyInstallerPkg.ID
	require.NotZero(t, dummyInstallerPkgInstallerID)
	require.NotNil(t, dummyInstallerPkg.UserID)
	globalAdmin, err := s.ds.UserByEmail(ctx, "admin1@example.com")
	require.NoError(t, err)
	require.Equal(t, globalAdmin.ID, *dummyInstallerPkg.UserID)
	require.Equal(t, "Test Name admin1@example.com", dummyInstallerPkg.UserName)
	require.Equal(t, "admin1@example.com", dummyInstallerPkg.UserEmail)

	// Upload ruby.deb to team1 by a user who will be deleted.
	u2 := &fleet.User{
		Name:       "admin team1",
		Email:      "admin_team1@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team1,
				Role: fleet.RoleAdmin,
			},
		},
	}
	require.NoError(t, u2.SetPassword(test.GoodPassword, 10, 10))
	adminTeam1, err := s.ds.NewUser(context.Background(), u2)
	require.NoError(t, err)
	rubyPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some deb install script",
		Filename:      "ruby.deb",
		TeamID:        &team1.ID,
	}
	adminTeam1Session, err := s.ds.NewSession(ctx, adminTeam1.ID, 64)
	require.NoError(t, err)
	adminToken := s.token
	t.Cleanup(func() {
		s.token = adminToken
	})
	s.token = adminTeam1Session.Key
	s.uploadSoftwareInstaller(t, rubyPayload, http.StatusOK, "")
	s.token = adminToken
	err = s.ds.DeleteUser(ctx, adminTeam1.ID)
	require.NoError(t, err)
	// Get software title ID of the uploaded installer.
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "ruby",
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	rubyDebTitleID := resp.SoftwareTitles[0].ID
	var rubyDeb struct {
		ID        uint   `db:"id"`
		UserID    *uint  `db:"user_id"`
		UserName  string `db:"user_name"`
		UserEmail string `db:"user_email"`
	}
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&rubyDeb,
			`SELECT id, user_id, user_name, user_email FROM software_installers WHERE global_or_team_id = ? AND filename = ?`,
			team1.ID, "ruby.deb",
		)
	})
	rubyDebInstallerID := rubyDeb.ID
	require.NotZero(t, rubyDebInstallerID)
	require.Nil(t, rubyDeb.UserID)
	require.Equal(t, "admin team1", rubyDeb.UserName)
	require.Equal(t, "admin_team1@example.com", rubyDeb.UserEmail)

	// Upload fleet-osquery.msi to team2.
	fleetOsqueryPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some msi install script",
		Filename:      "fleet-osquery.msi",
		TeamID:        &team2.ID,
		// Set as Self-service to check that the generated host_software_installs
		// is generated with self_service=false and the activity has the correct
		// author (the admin that uploaded the installer).
		SelfService: true,
	}
	s.uploadSoftwareInstaller(t, fleetOsqueryPayload, http.StatusOK, "")
	// Get software title ID of the uploaded installer.
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "Fleet osquery",
		"team_id", fmt.Sprintf("%d", team2.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	fleetOsqueryMSITitleID := resp.SoftwareTitles[0].ID
	var fleetOsqueryMSIInstallerID uint
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&fleetOsqueryMSIInstallerID,
			`SELECT id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`,
			team2.ID, "fleet-osquery.msi",
		)
	})
	require.NotZero(t, fleetOsqueryMSIInstallerID)

	// Create a VPP app to test that policies cannot be assigned to them.
	_, err = s.ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name:             "App123 " + t.Name(),
		BundleIdentifier: "bid_" + t.Name(),
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "adam_" + t.Name(),
				Platform: fleet.MacOSPlatform,
			},
		},
	}, &team1.ID)
	require.NoError(t, err)
	// Get software title ID of the uploaded VPP app.
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "App123",
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].AppStoreApp)
	vppAppTitleID := resp.SoftwareTitles[0].ID

	// Populate software for host1Team1 (to have a software title
	// that doesn't have an associated installer)
	software := []fleet.Software{
		{Name: "Foobar.app", Version: "0.0.1", Source: "apps"},
	}
	_, err = s.ds.UpdateHostSoftware(ctx, host1Team1.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, s.ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, time.Now()))
	// Get software title ID of the software.
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "Foobar.app",
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.Nil(t, resp.SoftwareTitles[0].SoftwarePackage)
	foobarAppTitleID := resp.SoftwareTitles[0].ID

	// policy0AllTeams is a global policy that runs on all devices.
	policy0AllTeams, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{
		Name:     "policy0AllTeams",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)
	// policy1Team1 runs on macOS devices.
	policy1Team1, err := s.ds.NewTeamPolicy(ctx, team1.ID, nil, fleet.PolicyPayload{
		Name:     "policy1Team1",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)
	// policy2Team1 runs on macOS and Linux devices.
	policy2Team1, err := s.ds.NewTeamPolicy(ctx, team1.ID, nil, fleet.PolicyPayload{
		Name:     "policy2Team1",
		Query:    "SELECT 2;",
		Platform: "linux,darwin",
	})
	require.NoError(t, err)
	// policy3Team1 runs on all devices in team1 (will have no associated installers).
	policy3Team1, err := s.ds.NewTeamPolicy(ctx, team1.ID, nil, fleet.PolicyPayload{
		Name:  "policy3Team1",
		Query: "SELECT 3;",
	})
	require.NoError(t, err)
	// policy4Team2 runs on Windows devices.
	policy4Team2, err := s.ds.NewTeamPolicy(ctx, team2.ID, nil, fleet.PolicyPayload{
		Name:     "policy4Team2",
		Query:    "SELECT 4;",
		Platform: "windows",
	})
	require.NoError(t, err)

	// Attempt to associate to an unknown software title.
	mtplr := modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: 999_999},
		},
	}, http.StatusBadRequest, &mtplr)
	// Attempt to associate to a software title without associated installer.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: foobarAppTitleID},
		},
	}, http.StatusBadRequest, &mtplr)
	// Attempt to associate vppApp to policy1Team1 which should fail because we only allow associating software installers.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: vppAppTitleID},
		},
	}, http.StatusBadRequest, &mtplr)
	// Associate dummy_installer.pkg to policy1Team1.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: dummyInstallerPkgTitleID},
		},
	}, http.StatusOK, &mtplr)
	// Change name only (to test not setting a software_title_id).
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID),
		json.RawMessage(`{"name": "policy1Team1_Renamed"}`), http.StatusOK, &mtplr,
	)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.NotNil(t, policy1Team1.SoftwareInstallerID)
	require.Equal(t, dummyInstallerPkgInstallerID, *policy1Team1.SoftwareInstallerID)
	require.Equal(t, "policy1Team1_Renamed", policy1Team1.Name)
	// Explicit set to 0 to disable.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: 0},
		},
	}, http.StatusOK, &mtplr)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Nil(t, policy1Team1.SoftwareInstallerID)

	// re-add software installer to policy1Team1
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: dummyInstallerPkgTitleID},
		},
	}, http.StatusOK, &mtplr)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.NotNil(t, policy1Team1.SoftwareInstallerID)
	require.Equal(t, dummyInstallerPkgInstallerID, *policy1Team1.SoftwareInstallerID)
	// Set to null to disable
	mtplr = modifyTeamPolicyResponse{}
	s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), []byte(`{
			"software_title_id": null
	}`), http.StatusOK)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Nil(t, policy1Team1.SoftwareInstallerID)

	host1LastInstall, err := s.ds.GetHostLastInstallData(ctx, host1Team1.ID, dummyInstallerPkgInstallerID)
	require.NoError(t, err)
	require.Nil(t, host1LastInstall)

	// Add some results and stats that should be cleared after setting an installer again.
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Equal(t, uint(0), policy1Team1.PassingHostCount)
	require.Equal(t, uint(1), policy1Team1.FailingHostCount)
	passes := true
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&passes,
			`SELECT passes FROM policy_membership WHERE policy_id = ? AND host_id = ?`,
			policy1Team1.ID, host1Team1.ID,
		)
	})
	require.False(t, passes)

	// Back to associating dummy_installer.pkg to policy1Team1.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: dummyInstallerPkgTitleID},
		},
	}, http.StatusOK, &mtplr)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.NotNil(t, policy1Team1.SoftwareInstallerID)
	require.Equal(t, dummyInstallerPkgInstallerID, *policy1Team1.SoftwareInstallerID)
	// Policy stats and membership should be cleared from policy1Team1.
	require.Equal(t, uint(0), policy1Team1.PassingHostCount)
	require.Equal(t, uint(0), policy1Team1.FailingHostCount)
	countBiggerThanZero := true
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&countBiggerThanZero,
			`SELECT COUNT(*) > 0 FROM policy_membership WHERE policy_id = ?`,
			policy1Team1.ID,
		)
	})
	require.False(t, countBiggerThanZero)

	// Add (again) some results and stats that should be cleared after changing an existing installer.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Equal(t, uint(0), policy1Team1.PassingHostCount)
	require.Equal(t, uint(1), policy1Team1.FailingHostCount)
	passes = true
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&passes,
			`SELECT passes FROM policy_membership WHERE policy_id = ? AND host_id = ?`,
			policy1Team1.ID, host1Team1.ID,
		)
	})
	require.False(t, passes)

	// Change the installer (temporarily to test that changing an installer will clear results)
	// Associate ruby.deb to policy1Team1.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: rubyDebTitleID},
		},
	}, http.StatusOK, &mtplr)

	// After changing the installer, membership and stats should be cleared.
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.NotNil(t, policy1Team1.SoftwareInstallerID)
	require.Equal(t, rubyDebInstallerID, *policy1Team1.SoftwareInstallerID)
	// Policy stats and membership should be cleared from policy1Team1.
	require.Equal(t, uint(0), policy1Team1.PassingHostCount)
	require.Equal(t, uint(0), policy1Team1.FailingHostCount)
	countBiggerThanZero = true
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&countBiggerThanZero,
			`SELECT COUNT(*) > 0 FROM policy_membership WHERE policy_id = ?`,
			policy1Team1.ID,
		)
	})
	require.False(t, countBiggerThanZero)

	// Back to (again) associating dummy_installer.pkg to policy1Team1.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: dummyInstallerPkgTitleID},
		},
	}, http.StatusOK, &mtplr)

	// Associate ruby.deb to policy2Team1.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy2Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: rubyDebTitleID},
		},
	}, http.StatusOK, &mtplr)

	// We use DoJSONWithoutAuth for distributed/write because we want the requests to not have the
	// current user's "Authorization: Bearer <API_TOKEN>" header.

	// host1Team1 fails all policies on the first report.
	// Failing policy1Team1 means an install request must be generated.
	// Failing policy2Team1 should not trigger a install request because it has a .deb attached to it (does not apply to macOS hosts).
	// Failing policy3Team1 should do nothing because it doesn't have any installers associated to it.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
			policy2Team1.ID: ptr.Bool(false),
			policy3Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host1Team1.ID, dummyInstallerPkgInstallerID)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)
	require.NotEmpty(t, host1LastInstall.ExecutionID)
	require.NotNil(t, host1LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstallPending, *host1LastInstall.Status)
	prevExecutionID := host1LastInstall.ExecutionID

	// Request a manual installation on the host for the same installer, which should fail.
	var installResp installSoftwareResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install",
		host1Team1.ID, dummyInstallerPkgTitleID), nil, http.StatusBadRequest, &installResp)

	// Submit same results as before, which should not trigger a installation because the policy is already failing.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
			policy2Team1.ID: ptr.Bool(false),
			policy3Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host1Team1.ID, dummyInstallerPkgInstallerID)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)
	require.Equal(t, prevExecutionID, host1LastInstall.ExecutionID)
	require.NotNil(t, host1LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstallPending, *host1LastInstall.Status)

	// Submit same results but policy1Team1 now passes,
	// and then submit again but policy1Team1 fails.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(true),
			policy2Team1.ID: ptr.Bool(false),
			policy3Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
			policy2Team1.ID: ptr.Bool(false),
			policy3Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	// Another installation should not be triggered because the last installation is pending.
	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host1Team1.ID, dummyInstallerPkgInstallerID)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)
	require.Equal(t, prevExecutionID, host1LastInstall.ExecutionID)
	require.NotNil(t, host1LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstallPending, *host1LastInstall.Status)

	// host2Team1 is failing policy2Team1 and policy3Team1 policies.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host2Team1,
		map[uint]*bool{
			policy2Team1.ID: ptr.Bool(false),
			policy3Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	host2LastInstall, err := s.ds.GetHostLastInstallData(ctx, host2Team1.ID, rubyDebInstallerID)
	require.NoError(t, err)
	require.NotNil(t, host2LastInstall)
	require.NotEmpty(t, host2LastInstall.ExecutionID)
	require.NotNil(t, host2LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstallPending, *host2LastInstall.Status)

	// Associate fleet-osquery.msi to policy4Team2.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team2.ID, policy4Team2.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: fleetOsqueryMSITitleID},
		},
	}, http.StatusOK, &mtplr)

	// host3Team2 reports a failing result for policy4Team2, which should trigger an installation.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3Team2,
		map[uint]*bool{
			policy4Team2.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	host3LastInstall, err := s.ds.GetHostLastInstallData(ctx, host3Team2.ID, fleetOsqueryMSIInstallerID)
	require.NoError(t, err)
	require.NotNil(t, host3LastInstall)
	require.NotEmpty(t, host3LastInstall.ExecutionID)
	require.NotNil(t, host3LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstallPending, *host3LastInstall.Status)
	host3LastInstallDetails, err := s.ds.GetSoftwareInstallDetails(ctx, host3LastInstall.ExecutionID)
	require.NoError(t, err)
	// Even if fleet-osquery.msi was uploaded as Self-service, it was installed by Fleet, so
	// host3LastInstallDetails.SelfService should be false.
	require.False(t, host3LastInstallDetails.SelfService)

	//
	// The following increase coverage of policies result processing in distributed/write.
	//

	// host3Team2 reports a passing result for policy0AllTeams which is a global policy.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3Team2,
		map[uint]*bool{
			policy0AllTeams.ID: ptr.Bool(true),
		},
	), http.StatusOK, &distributedResp)

	// host0NoTeam reports a failing result for policy0AllTeams which is a global policy.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host0NoTeam,
		map[uint]*bool{
			policy0AllTeams.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	// host3Team2 reports a failing result for policy0AllTeams which is a global policy.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3Team2,
		map[uint]*bool{
			policy0AllTeams.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	// Unassociate policy4Team2 from installer.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team2.ID, policy4Team2.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: 0},
		},
	}, http.StatusOK, &mtplr)

	// host3Team2 reports a failing result for policy4Team2.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3Team2,
		map[uint]*bool{
			policy4Team2.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	// Upcoming activities for host1Team1 should show the automatic installation of dummy_installer.pkg.
	// Check the author should be the admin that uploaded the installer.
	var listUpcomingAct listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host1Team1.ID), nil, http.StatusOK, &listUpcomingAct)
	require.Len(t, listUpcomingAct.Activities, 1)
	require.Nil(t, listUpcomingAct.Activities[0].ActorID)
	require.Equal(t, "Fleet", *listUpcomingAct.Activities[0].ActorFullName)
	require.Nil(t, listUpcomingAct.Activities[0].ActorEmail)

	//
	// Finally have orbit install the packages and check activities.
	//

	// host1Team1 posts the installation result for dummy_installer.pkg.
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "ok",
			"install_script_exit_code": 0,
			"install_script_output": "ok"
		}`, *host1Team1.OrbitNodeKey, host1LastInstall.ExecutionID)), http.StatusNoContent)
	s.lastActivityMatches(fleet.ActivityTypeInstalledSoftware{}.ActivityName(), fmt.Sprintf(`{
		"host_id": %d,
		"host_display_name": "%s",
		"software_title": "%s",
		"software_package": "%s",
		"self_service": false,
		"install_uuid": "%s",
		"status": "installed",
		"policy_id": %d,
		"policy_name": "%s"
	}`, host1Team1.ID, host1Team1.DisplayName(), "DummyApp.app", "dummy_installer.pkg", host1LastInstall.ExecutionID, policy1Team1.ID, policy1Team1.Name), 0)

	// host2Team1 posts the installation result for ruby.deb.
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "ok",
			"install_script_exit_code": 1,
			"install_script_output": "failed"
		}`, *host2Team1.OrbitNodeKey, host2LastInstall.ExecutionID)), http.StatusNoContent)
	activityID := s.lastActivityMatches(fleet.ActivityTypeInstalledSoftware{}.ActivityName(), fmt.Sprintf(`{
		"host_id": %d,
		"host_display_name": "%s",
		"software_title": "%s",
		"software_package": "%s",
		"self_service": false,
		"install_uuid": "%s",
		"status": "%s",
		"policy_id": %d,
		"policy_name": "%s"
	}`, host2Team1.ID, host2Team1.DisplayName(), "ruby", "ruby.deb", host2LastInstall.ExecutionID, fleet.SoftwareInstallFailed, policy2Team1.ID, policy2Team1.Name), 0)

	// Check that the activity item generated for ruby.deb installation is shown as coming from Fleet
	var actor struct {
		UserID     *uint   `db:"user_id"`
		UserName   *string `db:"user_name"`
		UserEmail  string  `db:"user_email"`
		PolicyID   *uint   `db:"policy_id"`
		PolicyName *string `db:"policy_name"`
	}
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&actor,
			`SELECT user_id, user_name, user_email, details->>'$.policy_id' policy_id, details->>'$.policy_name' policy_name FROM activities WHERE id = ?`,
			activityID,
		)
	})
	require.Nil(t, actor.UserID)
	require.NotNil(t, actor.UserName)
	require.Equal(t, "Fleet", *actor.UserName)
	require.Equal(t, "", actor.UserEmail)
	require.Equal(t, policy2Team1.ID, *actor.PolicyID)
	require.Equal(t, policy2Team1.Name, *actor.PolicyName)

	// host3Team2 posts the installation result for fleet-osquery.msi.
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "ok",
			"install_script_exit_code": 1,
			"install_script_output": "failed"
		}`, *host3Team2.OrbitNodeKey, host3LastInstall.ExecutionID)), http.StatusNoContent)
	activityID = s.lastActivityMatches(fleet.ActivityTypeInstalledSoftware{}.ActivityName(), fmt.Sprintf(`{
		"host_id": %d,
		"host_display_name": "%s",
		"software_title": "%s",
		"software_package": "%s",
		"self_service": false,
		"install_uuid": "%s",
		"status": "%s",
		"policy_id": %f,
		"policy_name": "%s"
	}`, host3Team2.ID, host3Team2.DisplayName(), "Fleet osquery", "fleet-osquery.msi", host3LastInstall.ExecutionID,
		fleet.SoftwareInstallFailed, float64(policy4Team2.ID), policy4Team2.Name), 0)

	// Check that the activity item generated for fleet-osquery.msi installation has Fleet set as author.
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&actor,
			`SELECT user_id, user_name, user_email, details->>'$.policy_id' policy_id, details->>'$.policy_name' policy_name FROM activities WHERE id = ?`,
			activityID,
		)
	})
	require.Nil(t, actor.UserID)
	require.NotNil(t, actor.UserName)
	require.Equal(t, "Fleet", *actor.UserName)
	require.Equal(t, "", actor.UserEmail)
	require.Equal(t, policy4Team2.ID, *actor.PolicyID)
	require.Equal(t, policy4Team2.Name, *actor.PolicyName)

	// hostVanillaOsquery5Team1 sends policy results with failed policies with associated installers.
	// Fleet should not queue an install for vanilla osquery hosts.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		hostVanillaOsquery5Team1,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	hostVanillaOsquery5Team1LastInstall, err := s.ds.GetHostLastInstallData(ctx, hostVanillaOsquery5Team1.ID, dummyInstallerPkgInstallerID)
	require.NoError(t, err)
	require.Nil(t, hostVanillaOsquery5Team1LastInstall)
}

func (s *integrationEnterpriseTestSuite) TestPolicyAutomationsScripts() {
	t := s.T()
	ctx := context.Background()

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team2"})
	require.NoError(t, err)

	newHost := func(name string, teamID *uint, platform string) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   ptr.String(t.Name() + name),
			NodeKey:         ptr.String(t.Name() + name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:        platform,
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}
	newFleetdHost := func(name string, teamID *uint, platform string) *fleet.Host {
		h := newHost(name, teamID, platform)
		orbitKey := setOrbitEnrollment(t, h, s.ds)
		h.OrbitNodeKey = &orbitKey
		return h
	}

	host1Team1 := newFleetdHost("host1Team1", &team1.ID, "darwin")
	host2Team1 := newFleetdHost("host2Team1", &team1.ID, "ubuntu")
	host3Team2 := newFleetdHost("host3Team2", &team2.ID, "windows")
	hostVanillaOsquery5Team1 := newHost("hostVanillaOsquery5Team2", &team1.ID, "darwin")

	// Upload script to team1.
	script, err := s.ds.NewScript(ctx, &fleet.Script{
		Name:           "unix-script.sh",
		ScriptContents: "echo 'Hello World'",
		TeamID:         &team1.ID,
	})
	require.NoError(t, err)
	require.NotZero(t, script.ID)

	// Upload winScript to team1.
	winScript, err := s.ds.NewScript(ctx, &fleet.Script{
		Name:           "windows-script.ps1",
		ScriptContents: "beep boop I am a windoge",
		TeamID:         &team1.ID,
	})
	require.NoError(t, err)
	require.NotZero(t, winScript.ID)

	// Upload script to team2.
	psScript, err := s.ds.NewScript(ctx, &fleet.Script{
		Name:           "windows-script.ps1",
		ScriptContents: "beep boop I am a window",
		TeamID:         &team2.ID,
	})
	require.NoError(t, err)
	require.NotZero(t, psScript.ID)

	// craete a global policy that runs on all devices.
	_, err = s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{
		Name:     "policy0AllTeams",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)
	// policy1Team1 runs on macOS devices.
	policy1Team1, err := s.ds.NewTeamPolicy(ctx, team1.ID, nil, fleet.PolicyPayload{
		Name:     "policy1Team1",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)
	// policy2Team1 runs on macOS and Linux devices.
	policy2Team1, err := s.ds.NewTeamPolicy(ctx, team1.ID, nil, fleet.PolicyPayload{
		Name:     "policy2Team1",
		Query:    "SELECT 2;",
		Platform: "linux,darwin",
	})
	require.NoError(t, err)
	// policy3Team1 runs on all devices in team1 (will have no associated scripts).
	policy3Team1, err := s.ds.NewTeamPolicy(ctx, team1.ID, nil, fleet.PolicyPayload{
		Name:  "policy3Team1",
		Query: "SELECT 3;",
	})
	require.NoError(t, err)
	// policy4Team2 runs on Windows devices.
	policy4Team2, err := s.ds.NewTeamPolicy(ctx, team2.ID, nil, fleet.PolicyPayload{
		Name:     "policy4Team2",
		Query:    "SELECT 4;",
		Platform: "windows",
	})
	require.NoError(t, err)

	// Attempt to associate to an unknown script.
	mtplr := modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: 999_999},
		},
	}, http.StatusBadRequest, &mtplr)
	// Associate first script to policy1Team1.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: script.ID},
		},
	}, http.StatusOK, &mtplr)
	// Change name only (to test not setting a script_id).
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID),
		json.RawMessage(`{"name": "policy1Team1_Renamed"}`), http.StatusOK, &mtplr,
	)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.NotNil(t, policy1Team1.ScriptID)
	require.Equal(t, script.ID, *policy1Team1.ScriptID)
	require.Equal(t, "policy1Team1_Renamed", policy1Team1.Name)
	// Explicit set to 0 to disable.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: 0},
		},
	}, http.StatusOK, &mtplr)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Nil(t, policy1Team1.ScriptID)

	// re-add script to policy1Team1.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: script.ID},
		},
	}, http.StatusOK, &mtplr)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.NotNil(t, policy1Team1.ScriptID)
	require.Equal(t, script.ID, *policy1Team1.ScriptID)
	// set to null to disable
	mtplr = modifyTeamPolicyResponse{}
	s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), []byte(`{
		"script_id": null
	}`), http.StatusOK)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Nil(t, policy1Team1.ScriptID)

	// Add some results and stats that should be cleared after updating the script
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Equal(t, uint(0), policy1Team1.PassingHostCount)
	require.Equal(t, uint(1), policy1Team1.FailingHostCount)
	passes := true
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&passes,
			`SELECT passes FROM policy_membership WHERE policy_id = ? AND host_id = ?`,
			policy1Team1.ID, host1Team1.ID,
		)
	})
	require.False(t, passes)

	// Back to associating the script with policy1Team1.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: script.ID},
		},
	}, http.StatusOK, &mtplr)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.NotNil(t, policy1Team1.ScriptID)
	require.Equal(t, script.ID, *policy1Team1.ScriptID)
	// Policy stats and membership should be cleared from policy1Team1.
	require.Equal(t, uint(0), policy1Team1.PassingHostCount)
	require.Equal(t, uint(0), policy1Team1.FailingHostCount)
	countBiggerThanZero := true
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&countBiggerThanZero,
			`SELECT COUNT(*) > 0 FROM policy_membership WHERE policy_id = ?`,
			policy1Team1.ID,
		)
	})
	require.False(t, countBiggerThanZero)

	// Add (again) some results and stats that should be cleared after changing an existing script.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Equal(t, uint(0), policy1Team1.PassingHostCount)
	require.Equal(t, uint(1), policy1Team1.FailingHostCount)
	passes = true
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&passes,
			`SELECT passes FROM policy_membership WHERE policy_id = ? AND host_id = ?`,
			policy1Team1.ID, host1Team1.ID,
		)
	})
	require.False(t, passes)

	// Change the script (temporarily to test that changing a script will clear results)
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: winScript.ID},
		},
	}, http.StatusOK, &mtplr)

	// After changing the script, membership and stats should be cleared.
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.NotNil(t, policy1Team1.ScriptID)
	require.Equal(t, winScript.ID, *policy1Team1.ScriptID)
	// Policy stats and membership should be cleared from policy1Team1.
	require.Equal(t, uint(0), policy1Team1.PassingHostCount)
	require.Equal(t, uint(0), policy1Team1.FailingHostCount)
	countBiggerThanZero = true
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&countBiggerThanZero,
			`SELECT COUNT(*) > 0 FROM policy_membership WHERE policy_id = ?`,
			policy1Team1.ID,
		)
	})
	require.False(t, countBiggerThanZero)

	// Back to (again) associating first script to policy1Team1.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: script.ID},
		},
	}, http.StatusOK, &mtplr)

	// Associate winScript to policy2Team1.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy2Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: winScript.ID},
		},
	}, http.StatusOK, &mtplr)

	// We use DoJSONWithoutAuth for distributed/write because we want the requests to not have the
	// current user's "Authorization: Bearer <API_TOKEN>" header.

	// host1Team1 fails all policies on the first report.
	// Failing policy1Team1 means a script run must be generated.
	// Failing policy2Team1 should not trigger a script run because it has a PowerShell script attached to it (doesn't apply to macOS).
	// Failing policy3Team1 should do nothing because it doesn't have any scripts associated to it.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
			policy2Team1.ID: ptr.Bool(false),
			policy3Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	hostPendingScript, err := s.ds.IsExecutionPendingForHost(ctx, host1Team1.ID, script.ID)
	require.NoError(t, err)
	require.True(t, hostPendingScript)

	// Request a manual script execution on the host for the same script, which should fail.
	var scriptRunResp runScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host1Team1.ID, ScriptID: &script.ID}, http.StatusConflict, &scriptRunResp)

	// Submit same results as before, which should not trigger a script run because the policy is already failing.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
			policy2Team1.ID: ptr.Bool(false),
			policy3Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	hostPendingScript, err = s.ds.IsExecutionPendingForHost(ctx, host1Team1.ID, script.ID)
	require.NoError(t, err)
	require.True(t, hostPendingScript)

	// Submit same results but policy1Team1 now passes,
	// and then submit again but policy1Team1 fails.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(true),
			policy2Team1.ID: ptr.Bool(false),
			policy3Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
			policy2Team1.ID: ptr.Bool(false),
			policy3Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	hostPendingScript, err = s.ds.IsExecutionPendingForHost(ctx, host1Team1.ID, script.ID)
	require.NoError(t, err)
	require.True(t, hostPendingScript)

	// host2Team1 is failing policy2Team1 (incompatible) and policy3Team1 (no script) policies; no scripts should be queued
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host2Team1,
		map[uint]*bool{
			policy2Team1.ID: ptr.Bool(false),
			policy3Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	hostPendingScript, err = s.ds.IsExecutionPendingForHost(ctx, host2Team1.ID, script.ID)
	require.NoError(t, err)
	require.False(t, hostPendingScript)

	// Associate psScript to policy4Team2.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team2.ID, policy4Team2.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: psScript.ID},
		},
	}, http.StatusOK, &mtplr)

	// host3Team2 reports a failing result for policy4Team2, which should trigger a script run.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3Team2,
		map[uint]*bool{
			policy4Team2.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	host3PendingScripts, err := s.ds.ListPendingHostScriptExecutions(ctx, host3Team2.ID, false)
	require.NoError(t, err)
	require.Len(t, host3PendingScripts, 1)
	host3executionID := host3PendingScripts[0].ExecutionID

	// Dissociate policy4Team2 from script.
	mtplr = modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team2.ID, policy4Team2.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: 0},
		},
	}, http.StatusOK, &mtplr)

	// host3Team2 reports a failing result for policy4Team2.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3Team2,
		map[uint]*bool{
			policy4Team2.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)

	// hostVanillaOsquery5Team1 sends policy results with failed policies with associated scripts.
	// Fleet should not queue scripts for vanilla osquery hosts.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		hostVanillaOsquery5Team1,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	hostPendingScripts, err := s.ds.ListPendingHostScriptExecutions(ctx, hostVanillaOsquery5Team1.ID, false)
	require.NoError(t, err)
	require.Len(t, hostPendingScripts, 0)

	// activity feed should show script run as pending, with "Fleet" as author, policy ID and name set in body
	var listResp listActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host3Team2.ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Activities, 1)
	require.Nil(t, listResp.Activities[0].ActorEmail)
	require.Equal(t, "Fleet", *listResp.Activities[0].ActorFullName)
	require.Nil(t, listResp.Activities[0].ActorGravatar)
	require.Equal(t, "ran_script", listResp.Activities[0].Type)
	var activityJson map[string]interface{}
	err = json.Unmarshal(*listResp.Activities[0].Details, &activityJson)
	require.NoError(t, err)
	require.Equal(t, float64(policy4Team2.ID), activityJson["policy_id"])
	require.Equal(t, "policy4Team2", activityJson["policy_name"])

	// post script result response
	var orbitPostScriptResp orbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host3Team2.OrbitNodeKey, host3executionID)),
		http.StatusOK, &orbitPostScriptResp)

	// activity feed should show script run as completed
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host3Team2.ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Activities, 1)
	require.Equal(t, "", *listResp.Activities[0].ActorEmail) // actor email is blank rather than nil here 👀
	require.Equal(t, "Fleet", *listResp.Activities[0].ActorFullName)
	require.Nil(t, listResp.Activities[0].ActorGravatar)
	require.Equal(t, "ran_script", listResp.Activities[0].Type)
	err = json.Unmarshal(*listResp.Activities[0].Details, &activityJson)
	require.NoError(t, err)
	require.Equal(t, float64(policy4Team2.ID), activityJson["policy_id"])
	require.Equal(t, "policy4Team2", activityJson["policy_name"])
}

func (s *integrationEnterpriseTestSuite) TestSoftwareInstallersWithoutBundleIdentifier() {
	t := s.T()
	ctx := context.Background()

	// Create a host without a team
	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	software := []fleet.Software{
		{Name: "DummyApp.app", Version: "0.0.2", Source: "apps"},
	}
	// we must ingest the title with an empty bundle identifier for this
	// test to be valid
	require.Empty(t, software[0].BundleIdentifier)
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, s.ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "dummy_installer.pkg",
		Version:       "0.0.2",
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
}

func (s *integrationEnterpriseTestSuite) TestSoftwareUploadRPM() {
	ctx := context.Background()
	t := s.T()

	// Fedora and RHEL have hosts.platform = 'rhel'.
	host := createOrbitEnrolledHost(t, "rhel", "", s.ds)

	// Upload an RPM package.
	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "install script",
		PreInstallQuery:   "pre install query",
		PostInstallScript: "post install script",
		Filename:          "ruby.rpm",
		Title:             "ruby",
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
	titleID := getSoftwareTitleID(t, s.ds, payload.Title, "rpm_packages")

	latestInstallUUID := func() string {
		var id string
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id, `SELECT execution_id FROM host_software_installs ORDER BY id DESC LIMIT 1`)
		})
		return id
	}

	// Send a request to the host to install the RPM package.
	var installSoftwareResp installSoftwareResponse
	beforeInstallRequest := time.Now()
	s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/hosts/%d/software/%d/install", host.ID, titleID), nil, http.StatusAccepted, &installSoftwareResp)
	installUUID := latestInstallUUID()

	// Simulate host installing the RPM package.
	beforeInstallResult := time.Now()
	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "1",
			"install_script_exit_code": 1,
			"install_script_output": "failed"
		}`, *host.OrbitNodeKey, installUUID)),
		http.StatusNoContent,
	)

	var resp getSoftwareInstallResultsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/install/%s/results", installUUID), nil, http.StatusOK, &resp)
	assert.Equal(t, host.ID, resp.Results.HostID)
	assert.Equal(t, installUUID, resp.Results.InstallUUID)
	assert.Equal(t, fleet.SoftwareInstallFailed, resp.Results.Status)
	assert.NotNil(t, resp.Results.PreInstallQueryOutput)
	assert.Equal(t, fleet.SoftwareInstallerQuerySuccessCopy, *resp.Results.PreInstallQueryOutput)
	assert.NotNil(t, resp.Results.Output)
	assert.Equal(t, fmt.Sprintf(fleet.SoftwareInstallerInstallFailCopy, "failed"), *resp.Results.Output)
	assert.Empty(t, resp.Results.PostInstallScriptOutput)
	assert.Less(t, beforeInstallRequest, resp.Results.CreatedAt)
	assert.Greater(t, time.Now(), resp.Results.CreatedAt)
	assert.NotNil(t, resp.Results.UpdatedAt)
	assert.Less(t, beforeInstallResult, *resp.Results.UpdatedAt)

	wantAct := fleet.ActivityTypeInstalledSoftware{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
		SoftwareTitle:   payload.Title,
		SoftwarePackage: payload.Filename,
		InstallUUID:     installUUID,
		Status:          string(fleet.SoftwareInstallFailed),
	}
	s.lastActivityMatches(wantAct.ActivityName(), string(jsonMustMarshal(t, wantAct)), 0)
}

func (s *integrationEnterpriseTestSuite) TestMaintainedApps() {
	t := s.T()
	ctx := context.Background()

	installerBytes := []byte("abc")

	// Mock server to serve the "installers"
	installerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/badinstaller":
			_, _ = w.Write([]byte("badinstaller"))
		case "/timeout":
			time.Sleep(3 * time.Second)
			_, _ = w.Write([]byte("timeout"))
		default:
			_, _ = w.Write(installerBytes)
		}
	}))
	defer installerServer.Close()

	getSoftwareInstallerIDByMAppID := func(mappID uint) uint {
		var id uint
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id, "SELECT id FROM software_installers WHERE fleet_library_app_id = ?", mappID)
		})

		return id
	}

	// Non-existent maintained app
	s.Do("POST", "/api/latest/fleet/software/fleet_maintained_apps", &addFleetMaintainedAppRequest{AppID: 1}, http.StatusNotFound)

	// Insert the list of maintained apps
	expectedApps := maintainedapps.IngestMaintainedApps(t, s.ds)

	// Edit DB to spoof URLs and SHA256 values so we don't have to actually download the installers
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		h := sha256.New()
		_, err := h.Write(installerBytes)
		require.NoError(t, err)
		spoofedSHA := hex.EncodeToString(h.Sum(nil))
		_, err = q.ExecContext(ctx, "UPDATE fleet_library_apps SET sha256 = ?, installer_url = ?", spoofedSHA, installerServer.URL+"/installer.zip")
		require.NoError(t, err)
		_, err = q.ExecContext(ctx, "UPDATE fleet_library_apps SET installer_url = ? WHERE id = 2", installerServer.URL+"/badinstaller")
		require.NoError(t, err)
		_, err = q.ExecContext(ctx, "UPDATE fleet_library_apps SET installer_url = ? WHERE id = 3", installerServer.URL+"/timeout")
		return err
	})

	// Create a team
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("Team 1")}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	// Check apps returned
	var listMAResp listFleetMaintainedAppsResponse
	s.DoJSON(http.MethodGet, "/api/latest/fleet/software/fleet_maintained_apps", listFleetMaintainedAppsRequest{}, http.StatusOK,
		&listMAResp, "team_id", fmt.Sprint(team.ID))
	require.Nil(t, listMAResp.Err)
	require.False(t, listMAResp.Meta.HasPreviousResults)
	require.False(t, listMAResp.Meta.HasNextResults)
	require.Len(t, listMAResp.FleetMaintainedApps, len(expectedApps))
	var listAppsNoID []fleet.MaintainedApp
	for _, app := range listMAResp.FleetMaintainedApps {
		app.ID = 0
		listAppsNoID = append(listAppsNoID, app)
	}
	slices.SortFunc(listAppsNoID, func(a, b fleet.MaintainedApp) int {
		return cmp.Compare(a.Name, b.Name)
	})
	slices.SortFunc(expectedApps, func(a, b fleet.MaintainedApp) int {
		return cmp.Compare(a.Name, b.Name)
	})
	require.Equal(t, expectedApps, listAppsNoID)

	var listMAResp2 listFleetMaintainedAppsResponse
	s.DoJSON(
		http.MethodGet,
		"/api/latest/fleet/software/fleet_maintained_apps",
		listFleetMaintainedAppsRequest{},
		http.StatusOK,
		&listMAResp2,
		"team_id", fmt.Sprint(team.ID),
		"per_page", "2",
		"page", "2",
	)
	require.Nil(t, listMAResp2.Err)
	require.True(t, listMAResp2.Meta.HasPreviousResults)
	require.True(t, listMAResp2.Meta.HasNextResults)
	require.Len(t, listMAResp2.FleetMaintainedApps, 2)
	require.Equal(t, listMAResp.FleetMaintainedApps[4:6], listMAResp2.FleetMaintainedApps)

	// Check individual app fetch
	var getMAResp getFleetMaintainedAppResponse
	s.DoJSON(http.MethodGet, fmt.Sprintf("/api/latest/fleet/software/fleet_maintained_apps/%d", listMAResp.FleetMaintainedApps[0].ID), getFleetMaintainedAppRequest{}, http.StatusOK, &getMAResp)
	// TODO this will change when actual install scripts are created.
	actualApp := listMAResp.FleetMaintainedApps[0]
	require.NotEmpty(t, getMAResp.FleetMaintainedApp.InstallScript)
	require.NotEmpty(t, getMAResp.FleetMaintainedApp.UninstallScript)
	getMAResp.FleetMaintainedApp.InstallScript = ""
	getMAResp.FleetMaintainedApp.UninstallScript = ""
	require.Equal(t, actualApp, *getMAResp.FleetMaintainedApp)

	// Try adding ingested app with invalid secret
	reqInvalidSecret := &addFleetMaintainedAppRequest{
		AppID:             1,
		TeamID:            &team.ID,
		SelfService:       true,
		PreInstallQuery:   "SELECT 1",
		InstallScript:     "echo foo $FLEET_SECRET_INVALID1",
		PostInstallScript: "echo done $FLEET_SECRET_INVALID2",
		UninstallScript:   "echo $FLEET_SECRET_INVALID3",
	}
	respBadSecret := s.Do("POST", "/api/latest/fleet/software/fleet_maintained_apps", reqInvalidSecret, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(respBadSecret.Body)
	require.Contains(t, errMsg, "$FLEET_SECRET_INVALID1")
	require.Contains(t, errMsg, "$FLEET_SECRET_INVALID2")
	require.Contains(t, errMsg, "$FLEET_SECRET_INVALID3")

	// Add an ingested app to the team
	var addMAResp addFleetMaintainedAppResponse
	req := &addFleetMaintainedAppRequest{
		AppID:             1,
		TeamID:            &team.ID,
		SelfService:       true,
		PreInstallQuery:   "SELECT 1",
		InstallScript:     "echo foo",
		PostInstallScript: "echo done",
	}
	s.DoJSON("POST", "/api/latest/fleet/software/fleet_maintained_apps", req, http.StatusOK, &addMAResp)
	require.Nil(t, addMAResp.Err)

	s.DoJSON(http.MethodGet, "/api/latest/fleet/software/fleet_maintained_apps", listFleetMaintainedAppsRequest{}, http.StatusOK,
		&listMAResp, "team_id", fmt.Sprint(team.ID))
	require.Nil(t, listMAResp.Err)
	require.False(t, listMAResp.Meta.HasPreviousResults)
	require.Len(t, listMAResp.FleetMaintainedApps, len(expectedApps)-1)

	// Validate software installer fields
	mapp, err := s.ds.GetMaintainedAppByID(ctx, 1)
	require.NoError(t, err)
	i, err := s.ds.GetSoftwareInstallerMetadataByID(context.Background(), getSoftwareInstallerIDByMAppID(1))
	require.NoError(t, err)
	require.Equal(t, ptr.Uint(1), i.FleetLibraryAppID)
	require.Equal(t, mapp.SHA256, i.StorageID)
	require.Equal(t, "darwin", i.Platform)
	require.NotEmpty(t, i.InstallScriptContentID)
	require.Equal(t, req.PreInstallQuery, i.PreInstallQuery)
	install, err := s.ds.GetAnyScriptContents(ctx, i.InstallScriptContentID)
	require.NoError(t, err)
	require.Equal(t, req.InstallScript, string(install))
	require.NotNil(t, i.PostInstallScriptContentID)
	postinstall, err := s.ds.GetAnyScriptContents(ctx, *i.PostInstallScriptContentID)
	require.NoError(t, err)
	require.Equal(t, req.PostInstallScript, string(postinstall))

	// The maintained app should now be in software titles
	var resp listSoftwareTitlesResponse
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"per_page", "1",
		"order_key", "name",
		"order_direction", "desc",
		"available_for_install", "true",
		"team_id", fmt.Sprintf("%d", team.ID),
	)

	require.Equal(t, 1, resp.Count)
	title := resp.SoftwareTitles[0]
	require.NotNil(t, title.BundleIdentifier)
	require.Equal(t, ptr.String(mapp.BundleIdentifier), title.BundleIdentifier)
	require.Equal(t, mapp.Version, title.SoftwarePackage.Version)
	require.Equal(t, "installer.zip", title.SoftwarePackage.Name)
	require.Equal(t, ptr.Bool(req.SelfService), title.SoftwarePackage.SelfService)

	// Check activity
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeAddedSoftware{}.ActivityName(),
		fmt.Sprintf(`{"software_title": "%[1]s", "software_package": "installer.zip", "team_name": "%s", "team_id": %d, "self_service": true, "software_title_id": %d}`, mapp.Name, team.Name, team.ID, title.ID),
		0,
	)

	// Should return an error; SHAs don't match up
	r := s.Do("POST", "/api/latest/fleet/software/fleet_maintained_apps", &addFleetMaintainedAppRequest{AppID: 2}, http.StatusInternalServerError)
	require.Contains(t, extractServerErrorText(r.Body), "mismatch in maintained app SHA256 hash")

	// Should timeout
	os.Setenv("FLEET_DEV_MAINTAINED_APPS_INSTALLER_TIMEOUT", "1s")
	r = s.Do("POST", "/api/latest/fleet/software/fleet_maintained_apps", &addFleetMaintainedAppRequest{AppID: 3}, http.StatusGatewayTimeout)
	os.Unsetenv("FLEET_DEV_MAINTAINED_APPS_INSTALLER_TIMEOUT")
	require.Contains(t, extractServerErrorText(r.Body), "Couldn't upload. Request timeout. Please make sure your server and load balancer timeout is long enough.")

	// Add a maintained app to no team

	req = &addFleetMaintainedAppRequest{
		AppID:             4,
		SelfService:       true,
		PreInstallQuery:   "SELECT 1",
		InstallScript:     "echo foo",
		PostInstallScript: "echo done",
	}

	addMAResp = addFleetMaintainedAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/fleet_maintained_apps", req, http.StatusOK, &addMAResp)
	require.Nil(t, addMAResp.Err)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"per_page", "1",
		"order_key", "name",
		"order_direction", "desc",
		"available_for_install", "true",
		"team_id", "0",
	)

	mapp, err = s.ds.GetMaintainedAppByID(ctx, 4)
	require.NoError(t, err)
	require.Equal(t, 1, resp.Count)
	title = resp.SoftwareTitles[0]
	require.NotNil(t, title.BundleIdentifier)
	require.Equal(t, ptr.String(mapp.BundleIdentifier), title.BundleIdentifier)
	require.Equal(t, mapp.Version, title.SoftwarePackage.Version)
	require.Equal(t, "installer.zip", title.SoftwarePackage.Name)

	i, err = s.ds.GetSoftwareInstallerMetadataByID(context.Background(), getSoftwareInstallerIDByMAppID(4))
	require.NoError(t, err)
	require.Equal(t, ptr.Uint(4), i.FleetLibraryAppID)
	require.Equal(t, mapp.SHA256, i.StorageID)
	require.Equal(t, "darwin", i.Platform)
	require.NotEmpty(t, i.InstallScriptContentID)
	require.Equal(t, req.PreInstallQuery, i.PreInstallQuery)
	install, err = s.ds.GetAnyScriptContents(ctx, i.InstallScriptContentID)
	require.NoError(t, err)
	require.Equal(t, req.InstallScript, string(install))
	require.NotNil(t, i.PostInstallScriptContentID)
	postinstall, err = s.ds.GetAnyScriptContents(ctx, *i.PostInstallScriptContentID)
	require.NoError(t, err)
	require.Equal(t, req.PostInstallScript, string(postinstall))

	// ===========================================================================================
	// Adding an automatically installed FMA
	// ===========================================================================================

	// Add another FMA
	req = &addFleetMaintainedAppRequest{
		AppID:             5,
		SelfService:       false,
		PreInstallQuery:   "SELECT 1",
		InstallScript:     "echo foo",
		PostInstallScript: "echo done",
		TeamID:            ptr.Uint(0),
	}

	addMAResp = addFleetMaintainedAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/fleet_maintained_apps", req, http.StatusOK, &addMAResp)
	require.NoError(t, addMAResp.Err)
	require.NotEmpty(t, addMAResp.SoftwareTitleID)

	// Add the automatic install policy
	tpParams := teamPolicyRequest{
		Name:            "[Install software]",
		Query:           "select * from osquery;",
		Description:     "Some description",
		Platform:        "darwin",
		SoftwareTitleID: &addMAResp.SoftwareTitleID,
	}
	tpResp := teamPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/teams/0/policies", tpParams, http.StatusOK, &tpResp)
	require.NotNil(t, tpResp.Policy)
	require.NotEmpty(t, tpResp.Policy.ID)

	// List software titles; we should see the policy on the software title object

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"per_page", "2",
		"order_key", "id",
		"order_direction", "desc",
		"available_for_install", "true",
		"team_id", "0",
	)

	require.Len(t, resp.SoftwareTitles, 2)
	// most recently added FMA should have 1 automatic install policy
	st := resp.SoftwareTitles[0] // sorted by ID above
	require.NotNil(t, st.SoftwarePackage)
	require.Len(t, st.SoftwarePackage.AutomaticInstallPolicies, 1)
	gotPolicy := st.SoftwarePackage.AutomaticInstallPolicies[0]
	require.Equal(t, tpResp.Policy.Name, gotPolicy.Name)
	require.Equal(t, tpResp.Policy.ID, gotPolicy.ID)

	// First FMA added doesn't have automatic install policies
	st = resp.SoftwareTitles[1] // sorted by ID above
	require.NotNil(t, st.SoftwarePackage)
	require.Empty(t, st.SoftwarePackage.AutomaticInstallPolicies)

	// Get the specific app that we set to be installed automatically
	var titleResp getSoftwareTitleResponse
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", addMAResp.SoftwareTitleID),
		getSoftwareTitleRequest{},
		http.StatusOK, &titleResp,
		"team_id", "0",
	)
	require.NotNil(t, titleResp.SoftwareTitle)
	swTitle := titleResp.SoftwareTitle
	require.NotNil(t, swTitle.SoftwarePackage)
	require.Len(t, swTitle.SoftwarePackage.AutomaticInstallPolicies, 1)
	gotPolicy = swTitle.SoftwarePackage.AutomaticInstallPolicies[0]
	require.Equal(t, tpResp.Policy.Name, gotPolicy.Name)
	require.Equal(t, tpResp.Policy.ID, gotPolicy.ID)

	// Policy should appear in the list of policies
	var listPolResp listTeamPoliciesResponse
	s.DoJSON(
		"GET", "/api/latest/fleet/teams/0/policies",
		listTeamPoliciesRequest{},
		http.StatusOK, &listPolResp,
		"page", "0",
	)

	require.Len(t, listPolResp.Policies, 1)
	policies := listPolResp.Policies
	require.Equal(t, tpResp.Policy.Name, policies[0].Name)
	require.Equal(t, tpResp.Policy.ID, policies[0].ID)
	require.Equal(t, tpResp.Policy.Description, policies[0].Description)
	require.Equal(t, tpResp.Policy.Query, policies[0].Query)
	require.Equal(t, "darwin", policies[0].Platform)
	require.False(t, policies[0].Critical)
	require.NotNil(t, policies[0].InstallSoftware)
	require.Equal(t, tpResp.Policy.InstallSoftware.Name, policies[0].InstallSoftware.Name)
	require.Equal(t, tpResp.Policy.InstallSoftware.SoftwareTitleID, policies[0].InstallSoftware.SoftwareTitleID)
}

func (s *integrationEnterpriseTestSuite) TestWindowsMigrateMDMNotEnabled() {
	t := s.T()

	res := s.Do("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "windows_migration_enabled": true }
	}`), http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Windows MDM is not enabled")
}
