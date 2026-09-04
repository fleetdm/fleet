package service

// Activity feed and automation-config tests for the core (no-license) suite.
//
// Belongs here: the activities listing endpoint, and the webhook/automation
// configuration for activities, host status, vulnerabilities and failing policies.
//
// Does not belong here: a single host's own past and upcoming activities
// (integration_core_hosts_reports_test.go).

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationTestSuite) TestListActivities() {
	t := s.T()

	ctx := context.Background()
	u := s.users["admin1@example.com"]

	prevActivities := s.listActivities()

	activitySvc := mysqltest.NewTestActivityService(t, s.ds)
	apiUser := &activity_api.User{ID: u.ID, Name: u.Name, Email: u.Email}
	err := activitySvc.NewActivity(ctx, apiUser, fleet.ActivityTypeAppliedSpecPack{})
	require.NoError(t, err)

	err = activitySvc.NewActivity(ctx, apiUser, fleet.ActivityTypeDeletedPack{})
	require.NoError(t, err)

	err = activitySvc.NewActivity(ctx, apiUser, fleet.ActivityTypeEditedPack{})
	require.NoError(t, err)

	lenPage := len(prevActivities) + 2

	var listResp listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listResp, "per_page", strconv.Itoa(lenPage), "order_key", "id")
	require.Len(t, listResp.Activities, lenPage)
	require.NotNil(t, listResp.Meta)
	assert.Equal(t, fleet.ActivityTypeAppliedSpecPack{}.ActivityName(), listResp.Activities[lenPage-2].Type)
	assert.Equal(t, fleet.ActivityTypeDeletedPack{}.ActivityName(), listResp.Activities[lenPage-1].Type)

	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listResp, "per_page", strconv.Itoa(lenPage), "order_key", "id", "page", "1")
	require.Len(t, listResp.Activities, 1)
	assert.Equal(t, fleet.ActivityTypeEditedPack{}.ActivityName(), listResp.Activities[0].Type)

	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listResp, "per_page", "1", "order_key", "id", "order_direction", "desc")
	require.Len(t, listResp.Activities, 1)
	assert.Equal(t, fleet.ActivityTypeEditedPack{}.ActivityName(), listResp.Activities[0].Type)

	listResp = listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listResp, "per_page", "1", "order_key", "id", "after", "0")
	require.Len(t, listResp.Activities, 1)
	require.Nil(t, listResp.Meta)
}

func (s *integrationTestSuite) TestGlobalPoliciesAutomationConfig() {
	t := s.T()

	gpParams := fleet.GlobalPolicyRequest{
		Name:  "policy1",
		Query: "select 41;",
	}
	gpResp := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)

	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(fmt.Sprintf(`{
		"webhook_settings": {
    		"failing_policies_webhook": {
     	 		"enable_failing_policies_webhook": true,
     	 		"destination_url": "http://some/url",
     			"policy_ids": [%d],
				"host_batch_size": 1000
    		},
    		"interval": "1h"
  		}
	}`, gpResp.Policy.ID)), http.StatusOK)

	config := s.getConfig()
	require.True(t, config.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.FailingPoliciesWebhook.DestinationURL)
	require.Equal(t, []uint{gpResp.Policy.ID}, config.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)
	require.Equal(t, 1*time.Hour, config.WebhookSettings.Interval.Duration)
	require.Equal(t, 1000, config.WebhookSettings.FailingPoliciesWebhook.HostBatchSize)

	deletePolicyParams := fleet.DeleteGlobalPoliciesRequest{IDs: []uint{gpResp.Policy.ID}}
	deletePolicyResp := fleet.DeleteGlobalPoliciesResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deletePolicyParams, http.StatusOK, &deletePolicyResp)

	config = s.getConfig()
	require.Empty(t, config.WebhookSettings.FailingPoliciesWebhook.PolicyIDs) // nolint:nilaway // getConfig fails the test via require internally on error, cannot be nil here
}

func (s *integrationTestSuite) TestActivitiesWebhookConfig() {
	t := s.T()

	s.DoRaw(
		"PATCH", "/api/latest/fleet/config", []byte(
			`{
		"webhook_settings": {
			"activities_webhook": {
				"enable_activities_webhook": true,
				"destination_url": "http://some/url"
    		}
  		}
	}`,
		), http.StatusOK,
	)

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEnabledActivityAutomations{}.ActivityName(),
		`{"webhook_url": "http://some/url"}`,
		0,
	)

	appConfig := s.getConfig()
	require.True(t, appConfig.WebhookSettings.ActivitiesWebhook.Enable)
	require.Equal(t, "http://some/url", appConfig.WebhookSettings.ActivitiesWebhook.DestinationURL)

	s.DoRaw(
		"PATCH", "/api/latest/fleet/config", []byte(
			`{
		"webhook_settings": {
			"activities_webhook": {
				"enable_activities_webhook": true,
				"destination_url": "http://some/other/url"
    		}
  		}
	}`,
		), http.StatusOK,
	)

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedActivityAutomations{}.ActivityName(),
		`{"webhook_url": "http://some/other/url"}`,
		0,
	)

	s.DoRaw(
		"PATCH", "/api/latest/fleet/config", []byte(
			`{
		"webhook_settings": {
			"activities_webhook": {
				"enable_activities_webhook": true,
				"destination_url": "invalid-url"
    		}
  		}
	}`,
		), http.StatusUnprocessableEntity,
	)

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedActivityAutomations{}.ActivityName(),
		`{"webhook_url": "http://some/other/url"}`,
		0,
	)

	s.DoRaw(
		"PATCH", "/api/latest/fleet/config", []byte(
			`{
		"webhook_settings": {
			"activities_webhook": {
				"enable_activities_webhook": false
    		}
  		}
	}`,
		), http.StatusOK,
	)

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDisabledActivityAutomations{}.ActivityName(),
		``,
		0,
	)

	s.DoRaw(
		"PATCH", "/api/latest/fleet/config", []byte(
			`{
		"webhook_settings": {
			"activities_webhook": {
				"enable_activities_webhook": true,
				"destination_url": "foo.baz"
    		}
  		}
	}`,
		), http.StatusUnprocessableEntity,
	)

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEnabledActivityAutomations{}.ActivityName(),
		`{"webhook_url": "http://some/url"}`,
		0,
	)
}

func (s *integrationTestSuite) TestHostStatusWebhookConfig() {
	t := s.T()

	// enable with valid config
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"webhook_settings": {
    		"host_status_webhook": {
     	 		"enable_host_status_webhook": true,
     	 		"destination_url": "http://some/url",
				  "host_percentage": 2,
					"days_count": 1
    		},
    		"interval": "1h"
  		}
	}`), http.StatusOK)

	config := s.getConfig()
	require.True(t, config.WebhookSettings.HostStatusWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.HostStatusWebhook.DestinationURL)
	require.InDelta(t, 2.0, config.WebhookSettings.HostStatusWebhook.HostPercentage, 0.001)
	require.Equal(t, 1, config.WebhookSettings.HostStatusWebhook.DaysCount)

	// update without a destination url
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"webhook_settings": {
    		"host_status_webhook": {
     	 		"enable_host_status_webhook": true,
     	 		"destination_url": "",
				  "host_percentage": 2,
					"days_count": 1
    		},
    		"interval": "1h"
  		}
	}`), http.StatusUnprocessableEntity)

	// update without a negative days count
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"webhook_settings": {
    		"host_status_webhook": {
     	 		"enable_host_status_webhook": true,
					"destination_url": "http://other/url",
				  "host_percentage": 2,
					"days_count": -123
    		},
    		"interval": "1h"
  		}
	}`), http.StatusUnprocessableEntity)

	// update with 0%
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"webhook_settings": {
    		"host_status_webhook": {
     	 		"enable_host_status_webhook": true,
					"destination_url": "http://other/url",
				  "host_percentage": 0,
					"days_count": 12
    		},
    		"interval": "1h"
  		}
	}`), http.StatusUnprocessableEntity)

	// config left unmodified since last successful call
	config = s.getConfig()
	require.True(t, config.WebhookSettings.HostStatusWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.HostStatusWebhook.DestinationURL)
	require.InDelta(t, 2.0, config.WebhookSettings.HostStatusWebhook.HostPercentage, 0.001)
	require.Equal(t, 1, config.WebhookSettings.HostStatusWebhook.DaysCount)

	// disabling ignores the invalid parameters
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"webhook_settings": {
    		"host_status_webhook": {
     	 		"enable_host_status_webhook": false,
     	 		"destination_url": "",
				  "host_percentage": 0
    		},
    		"interval": "1h"
  		}
	}`), http.StatusOK)

	config = s.getConfig()
	require.False(t, config.WebhookSettings.HostStatusWebhook.Enable)
}

func (s *integrationTestSuite) TestVulnerabilitiesWebhookConfig() {
	t := s.T()

	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"integrations": {"jira": [], "zendesk": []},
		"webhook_settings": {
    		"vulnerabilities_webhook": {
     	 		"enable_vulnerabilities_webhook": true,
     	 		"destination_url": "http://some/url",
     	 		"host_batch_size": 1234
    		},
    		"interval": "1h"
  		}
	}`), http.StatusOK)

	config := s.getConfig()
	require.True(t, config.WebhookSettings.VulnerabilitiesWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.VulnerabilitiesWebhook.DestinationURL)
	require.Equal(t, 1234, config.WebhookSettings.VulnerabilitiesWebhook.HostBatchSize)
	require.Equal(t, 1*time.Hour, config.WebhookSettings.Interval.Duration)
}
