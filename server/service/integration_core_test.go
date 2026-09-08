package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query/live_query_mock"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type integrationTestSuite struct {
	suite.Suite

	withServer
}

func (s *integrationTestSuite) SetupSuite() {
	s.withServer.lq = live_query_mock.New(s.T())
	s.withServer.SetupSuite("integrationTestSuite")
}

func (s *integrationTestSuite) TearDownTest() {
	s.withServer.commonTearDownTest(s.T())
}

func TestIntegrations(t *testing.T) {
	testingSuite := new(integrationTestSuite)
	testingSuite.withServer.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type slowReader struct{}

func (s *slowReader) Read(p []byte) (n int, err error) {
	time.Sleep(3 * time.Second)
	return 0, nil
}

func (s *integrationTestSuite) createHosts(t *testing.T, platforms ...string) []*fleet.Host {
	var hosts []*fleet.Host
	if len(platforms) == 0 {
		platforms = []string{"debian", "rhel", "linux"}
	}
	for i, platform := range platforms {
		host, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   new(fmt.Sprintf("%s%d", t.Name(), i)),
			NodeKey:         new(fmt.Sprintf("%s%d", t.Name(), i)),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
			Platform:        platform,
		})
		require.NoError(t, err)
		hosts = append(hosts, host)
	}
	return hosts
}

type macadminsDataResponse struct {
	Macadmins *struct {
		Munki       *fleet.HostMunkiInfo    `json:"munki"`
		MunkiIssues []*fleet.HostMunkiIssue `json:"munki_issues"`
		MDM         *struct {
			EnrollmentStatus string  `json:"enrollment_status"`
			ServerURL        string  `json:"server_url"`
			Name             *string `json:"name"`
			ID               *uint   `json:"id"`
		} `json:"mobile_device_management"`
	} `json:"macadmins"`
}

var hostIOSVitalsJSONKeys = []string{
	"udid", "model_number", "modem_firmware_version", "supplemental_build_version",
	"supplemental_os_version_extra", "bluetooth_mac", "wifi_mac", "eas_device_identifier",
	"itunes_store_account_hash", "push_token", "battery_level", "cellular_technology",
	"app_analytics_enabled", "awaiting_configuration", "data_roaming_enabled",
	"diagnostic_submission_enabled", "is_cloud_backup_enabled", "is_device_locator_service_enabled",
	"is_do_not_disturb_in_effect", "is_mdm_lost_mode_enabled", "is_network_tethered",
	"itunes_store_account_is_active", "personal_hotspot_enabled", "last_cloud_backup_date",
	"accessibility_settings", "organization_info", "mdm_options", "device_properties_attestation",
	"service_subscriptions",
}

func (s *integrationTestSuite) getHostJSON(path string) map[string]any {
	t := s.T()
	res := s.DoRaw("GET", path, nil, http.StatusOK)
	defer res.Body.Close()

	var raw struct {
		Host map[string]any `json:"host"`
	}
	require.NoError(t, json.NewDecoder(res.Body).Decode(&raw))
	return raw.Host
}

type validationErrResp struct {
	Message string `json:"message"`
	Errors  []struct {
		Name   string `json:"name"`
		Reason string `json:"reason"`
	} `json:"errors"`
}

func setOrbitEnrollment(t *testing.T, h *fleet.Host, ds fleet.Datastore) string {
	orbitKey := uuid.New().String()
	_, err := ds.EnrollOrbit(
		context.Background(),
		fleet.WithEnrollOrbitHostInfo(fleet.OrbitHostInfo{
			HardwareUUID:   *h.OsqueryHostID,
			HardwareSerial: h.HardwareSerial,
		}),
		fleet.WithEnrollOrbitNodeKey(orbitKey),
		fleet.WithEnrollOrbitTeamID(h.TeamID),
	)
	require.NoError(t, err)
	err = ds.SetOrUpdateHostOrbitInfo(
		context.Background(), h.ID, "1.22.0", sql.NullString{String: "42", Valid: true}, sql.NullBool{Bool: true, Valid: true},
	)
	require.NoError(t, err)
	return orbitKey
}

func createOrbitEnrolledHost(t *testing.T, platform, suffix string, ds fleet.Datastore) *fleet.Host {
	name := t.Name() + suffix
	h, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-time.Minute),
		OsqueryHostID:   new(name),
		NodeKey:         new(name),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", name),
		ComputerName:    name,
		HardwareSerial:  uuid.New().String(),
		Platform:        platform,
	})
	require.NoError(t, err)

	orbitKey := setOrbitEnrollment(t, h, ds)
	h.OrbitNodeKey = &orbitKey
	return h
}

// creates a session and returns it, its key is to be passed as authorization header.
func createSession(t *testing.T, uid uint, ds fleet.Datastore) *fleet.Session {
	ssn, err := ds.NewSession(context.Background(), uid, 64)
	require.NoError(t, err)

	return ssn
}

func (s *integrationTestSuite) cleanupQuery(queryID uint) {
	var delResp fleet.DeleteQueryByIDResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", queryID), nil, http.StatusOK, &delResp)
}

func jsonMustMarshal(t testing.TB, v any) []byte {
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// starts a test web server that mocks responses to requests to external
// services with a valid payload (if the request is valid) or a status code
// error. It returns the URL to use to make requests to that server.
//
// For Jira, the project keys "qux" and "qux2" are supported.
// For Zendesk, the group IDs "122" and "123" are supported.
//
// The basic auth's user (or password for Zendesk) "ok" means that auth is
// allowed, while "fail" means unauthorized and anything else results in status
// 502.
func startExternalServiceWebServer(t *testing.T) string {
	// create a test http server to act as the Jira and Zendesk server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(501)
			return
		}

		switch r.URL.Path {
		case "/rest/api/2/project/qux":
			switch usr, _, _ := r.BasicAuth(); usr {
			case "ok":
				_, err := w.Write([]byte(jiraProjectResponsePayload))
				assert.NoError(t, err)
			case "fail":
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(502)
			}
		case "/rest/api/2/project/qux2":
			switch usr, _, _ := r.BasicAuth(); usr {
			case "ok":
				_, err := w.Write([]byte(jiraProjectResponsePayload))
				assert.NoError(t, err)
			case "fail":
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(502)
			}
		case "/api/v2/groups/122.json":
			switch _, pwd, _ := r.BasicAuth(); pwd {
			case "ok":
				_, err := w.Write([]byte(`{"group": {"id": 122,"name": "test122"}}`))
				assert.NoError(t, err)
			case "fail":
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(502)
			}
		case "/api/v2/groups/123.json":
			switch _, pwd, _ := r.BasicAuth(); pwd {
			case "ok":
				_, err := w.Write([]byte(`{"group": {"id": 123,"name": "test123"}}`))
				assert.NoError(t, err)
			case "fail":
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(502)
			}
		default:
			w.WriteHeader(502)
		}
	}))
	t.Cleanup(srv.Close)

	return srv.URL
}

const (
	// example response from the Jira docs
	jiraProjectResponsePayload = `{
  "self": "https://your-domain.atlassian.net/rest/api/2/project/EX",
  "id": "10000",
  "key": "EX",
  "description": "This project was created as an example for REST.",
  "lead": {
    "self": "https://your-domain.atlassian.net/rest/api/2/user?accountId=5b10a2844c20165700ede21g",
    "key": "",
    "accountId": "5b10a2844c20165700ede21g",
    "accountType": "atlassian",
    "name": "",
    "avatarUrls": {
      "48x48": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=48&s=48",
      "24x24": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=24&s=24",
      "16x16": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=16&s=16",
      "32x32": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=32&s=32"
    },
    "displayName": "Mia Krystof",
    "active": false
  },
  "components": [
    {
      "self": "https://your-domain.atlassian.net/rest/api/2/component/10000",
      "id": "10000",
      "name": "Component 1",
      "description": "This is a Jira component",
      "lead": {
        "self": "https://your-domain.atlassian.net/rest/api/2/user?accountId=5b10a2844c20165700ede21g",
        "key": "",
        "accountId": "5b10a2844c20165700ede21g",
        "accountType": "atlassian",
        "name": "",
        "avatarUrls": {
          "48x48": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=48&s=48",
          "24x24": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=24&s=24",
          "16x16": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=16&s=16",
          "32x32": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=32&s=32"
        },
        "displayName": "Mia Krystof",
        "active": false
      },
      "assigneeType": "PROJECT_LEAD",
      "assignee": {
        "self": "https://your-domain.atlassian.net/rest/api/2/user?accountId=5b10a2844c20165700ede21g",
        "key": "",
        "accountId": "5b10a2844c20165700ede21g",
        "accountType": "atlassian",
        "name": "",
        "avatarUrls": {
          "48x48": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=48&s=48",
          "24x24": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=24&s=24",
          "16x16": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=16&s=16",
          "32x32": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=32&s=32"
        },
        "displayName": "Mia Krystof",
        "active": false
      },
      "realAssigneeType": "PROJECT_LEAD",
      "realAssignee": {
        "self": "https://your-domain.atlassian.net/rest/api/2/user?accountId=5b10a2844c20165700ede21g",
        "key": "",
        "accountId": "5b10a2844c20165700ede21g",
        "accountType": "atlassian",
        "name": "",
        "avatarUrls": {
          "48x48": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=48&s=48",
          "24x24": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=24&s=24",
          "16x16": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=16&s=16",
          "32x32": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=32&s=32"
        },
        "displayName": "Mia Krystof",
        "active": false
      },
      "isAssigneeTypeValid": false,
      "project": "HSP",
      "projectId": 10000
    }
  ],
  "issueTypes": [
    {
      "self": "https://your-domain.atlassian.net/rest/api/2/issueType/3",
      "id": "3",
      "description": "A task that needs to be done.",
      "iconUrl": "https://your-domain.atlassian.net/secure/viewavatar?size=xsmall&avatarId=10299&avatarType=issuetype\",",
      "name": "Task",
      "subtask": false,
      "avatarId": 1,
      "hierarchyLevel": 0
    },
    {
      "self": "https://your-domain.atlassian.net/rest/api/2/issueType/1",
      "id": "1",
      "description": "A problem with the software.",
      "iconUrl": "https://your-domain.atlassian.net/secure/viewavatar?size=xsmall&avatarId=10316&avatarType=issuetype\",",
      "name": "Bug",
      "subtask": false,
      "avatarId": 10002,
      "entityId": "9d7dd6f7-e8b6-4247-954b-7b2c9b2a5ba2",
      "hierarchyLevel": 0,
      "scope": {
        "type": "PROJECT",
        "project": {
          "id": "10000",
          "key": "KEY",
          "name": "Next Gen Project"
        }
      }
    }
  ],
  "url": "https://www.example.com",
  "email": "from-jira@example.com",
  "assigneeType": "PROJECT_LEAD",
  "versions": [],
  "name": "Example",
  "roles": {
    "Developers": "https://your-domain.atlassian.net/rest/api/2/project/EX/role/10000"
  },
  "avatarUrls": {
    "48x48": "https://your-domain.atlassian.net/secure/projectavatar?size=large&pid=10000",
    "24x24": "https://your-domain.atlassian.net/secure/projectavatar?size=small&pid=10000",
    "16x16": "https://your-domain.atlassian.net/secure/projectavatar?size=xsmall&pid=10000",
    "32x32": "https://your-domain.atlassian.net/secure/projectavatar?size=medium&pid=10000"
  },
  "projectCategory": {
    "self": "https://your-domain.atlassian.net/rest/api/2/projectCategory/10000",
    "id": "10000",
    "name": "FIRST",
    "description": "First Project Category"
  },
  "simplified": false,
  "style": "classic",
  "properties": {
    "propertyKey": "propertyValue"
  },
  "insight": {
    "totalIssueCount": 100,
    "lastIssueUpdateTime": "2022-04-05T04:51:35.670+0000"
  }
}`
)

// Creates a set of results for use in tests for Query Results.
func results(num int, hostID string) string {
	b := strings.Builder{}
	for i := range num {
		b.WriteString(`    {
      "build_distro": "centos7",
      "build_platform": "linux",
      "config_hash": "eed0d8296e5f90b790a23814a9db7a127b13498d",
      "config_valid": "1",
      "extensions": "active",
      "instance_id": "e5799132-85ab-4cfa-89f3-03e0dd3c509a",
      "pid": "3574",
      "platform_mask": "9",
      "start_time": "1696502961",
      "uuid": "` + hostID + `",
      "version": "5.9.2",
      "watcher": "3570"
    }`)
		if i != num-1 {
			b.WriteString(",")
		}
	}

	return b.String()
}

// createAndroidHostForTest creates an android host. companyOwned=false stores it as BYO
// (is_personal_enrollment=true); companyOwned=true stores it as COBO.
func createAndroidHostForTest(t *testing.T, ds *mysql.Datastore, teamID *uint, companyOwned bool) uint {
	host := &fleet.AndroidHost{
		Host: &fleet.Host{
			Hostname:                  "android-storage-host",
			ComputerName:              "Android Storage Test Device",
			Platform:                  "android",
			OSVersion:                 "Android 14",
			Build:                     "UPB4.230623.005",
			Memory:                    8192, // 8GB RAM
			TeamID:                    teamID,
			HardwareSerial:            "STORAGE-TEST-" + uuid.NewString(),
			GigsTotalDiskSpace:        128.0, // 64GB system + 64GB external
			GigsDiskSpaceAvailable:    35.0,  // 10GB + 25GB available
			PercentDiskSpaceAvailable: 27.34, // 35/128 * 100
			UUID:                      uuid.NewString(),
		},
		Device: &android.Device{
			DeviceID:             strings.ReplaceAll(uuid.NewString(), "-", ""),
			EnterpriseSpecificID: new(uuid.NewString()),
			AppliedPolicyID:      new("1"),
			LastPolicySyncTime:   new(time.Now().Add(-time.Hour)),
		},
	}
	host.SetNodeKey(*host.Device.EnterpriseSpecificID)
	ahost, err := ds.NewAndroidHost(context.Background(), host, companyOwned)
	require.NoError(t, err)
	return ahost.Host.ID
}
