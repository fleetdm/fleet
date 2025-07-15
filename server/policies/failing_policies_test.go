package policies

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

type policyAutomation struct {
	polID      uint
	automation FailingPolicyAutomationType
}

func TestTriggerFailingPolicies(t *testing.T) {
	ds := new(mock.Store)

	// test that every configuration works - global for specific policy IDs,
	// per-team for specific policy IDs, different configurations for different
	// teams:
	//
	// pol-global-{1-3}: global policies (only 1 and 2 is enabled), ids 1-2-3
	// pol-teamA-{4-6}: team A policies (only 4 and 5 is enabled), ids 4-5-6
	// pol-teamB-{7-9}: team B policies (only 7 and 8 is enabled), ids 7-8-9
	// pol-teamC-10: team C policy, team does not exist, id 10
	// pol-unknown-11: policy that does not exist anymore, id 11
	// pol-teamD-{12-14}: team D policies (only 12 and 13 is enabled), ids 12-13-14
	// pol-teamE-15: team E policy, integration does not exist at the global level

	// Note re No-team policies in this context: Though can't set a no team policy to trigger an automation
	// in the UI, it can still theoretically be done via the API. In such a case, current logic
	// will try to find a team 0 config, which doesn't exist, and error. TODO - either make the
	// determination that this case should always use the global config (as it now does in Primo
	// mode), or confirm we expect an error here.
	//
	// Global config uses the webhook, team A a Jira integration, team B a
	// Zendesk integration, team D a webhook.

	pols := map[uint]*fleet.PolicyData{
		1:  {ID: 1, Name: "pol-global-1"},
		2:  {ID: 2, Name: "pol-global-2"},
		3:  {ID: 3, Name: "pol-global-3"},
		4:  {ID: 4, Name: "pol-teamA-4", TeamID: ptr.Uint(1)},
		5:  {ID: 5, Name: "pol-teamA-5", TeamID: ptr.Uint(1)},
		6:  {ID: 6, Name: "pol-teamA-6", TeamID: ptr.Uint(1)},
		7:  {ID: 7, Name: "pol-teamB-7", TeamID: ptr.Uint(2)},
		8:  {ID: 8, Name: "pol-teamB-8", TeamID: ptr.Uint(2)},
		9:  {ID: 9, Name: "pol-teamB-9", TeamID: ptr.Uint(2)},
		10: {ID: 10, Name: "pol-teamC-10", TeamID: ptr.Uint(3)},
		// intentionally omit 11 for testing the edge case where a policy failure is received for a
		// non-existent policy (see below)
		12: {ID: 12, Name: "pol-teamD-12", TeamID: ptr.Uint(4)},
		13: {ID: 13, Name: "pol-teamD-13", TeamID: ptr.Uint(4)},
		14: {ID: 14, Name: "pol-teamD-14", TeamID: ptr.Uint(4)},
		15: {ID: 15, Name: "pol-teamE-15", TeamID: ptr.Uint(5)},
	}
	ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) {
		pd, ok := pols[id]
		if !ok {
			return nil, ctxerr.Wrap(ctx, sql.ErrNoRows)
		}
		return &fleet.Policy{PolicyData: *pd}, nil
	}

	teams := map[uint]*fleet.Team{
		1: {ID: 1, Name: "teamA", Config: fleet.TeamConfig{
			WebhookSettings: fleet.TeamWebhookSettings{
				FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
					PolicyIDs: []uint{4, 5},
				},
			},
			Integrations: fleet.TeamIntegrations{
				Jira: []*fleet.TeamJiraIntegration{
					{URL: "http://j.com", ProjectKey: "A", EnableFailingPolicies: true},
				},
			},
		}},
		2: {ID: 2, Name: "teamB", Config: fleet.TeamConfig{
			WebhookSettings: fleet.TeamWebhookSettings{
				FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
					PolicyIDs: []uint{7, 8},
				},
			},
			Integrations: fleet.TeamIntegrations{
				Zendesk: []*fleet.TeamZendeskIntegration{
					{URL: "http://z.com", GroupID: 1, EnableFailingPolicies: true},
				},
			},
		}},
		4: {ID: 4, Name: "teamD", Config: fleet.TeamConfig{
			WebhookSettings: fleet.TeamWebhookSettings{
				FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
					Enable:    true,
					PolicyIDs: []uint{12, 13},
				},
			},
			Integrations: fleet.TeamIntegrations{
				Zendesk: []*fleet.TeamZendeskIntegration{
					{URL: "http://z.com", GroupID: 1, EnableFailingPolicies: false},
				},
				Jira: []*fleet.TeamJiraIntegration{
					{URL: "http://j.com", ProjectKey: "A", EnableFailingPolicies: false},
				},
			},
		}},
		5: {ID: 5, Name: "teamE", Config: fleet.TeamConfig{
			WebhookSettings: fleet.TeamWebhookSettings{
				FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
					PolicyIDs: []uint{15},
				},
			},
			Integrations: fleet.TeamIntegrations{
				Zendesk: []*fleet.TeamZendeskIntegration{
					{URL: "http://notexist", GroupID: 999, EnableFailingPolicies: true},
				},
			},
		}},
	}
	ds.TeamFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
		tm, ok := teams[id]
		if !ok {
			return nil, ctxerr.Wrap(ctx, sql.ErrNoRows)
		}
		return tm, nil
	}

	// globally enable the webhook automation
	ac := &fleet.AppConfig{
		WebhookSettings: fleet.WebhookSettings{
			FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
				Enable:    true,
				PolicyIDs: []uint{1, 2},
			},
		},
		Integrations: fleet.Integrations{
			Jira: []*fleet.JiraIntegration{
				{URL: "http://j.com", ProjectKey: "A", Username: "jirauser", APIToken: "secret"},
			},
			Zendesk: []*fleet.ZendeskIntegration{
				{URL: "http://z.com", GroupID: 1, Email: "zendesk@z.com", APIToken: "secret"},
			},
		},
		ServerSettings: fleet.ServerSettings{
			ServerURL: "https://fleet.example.com",
		},
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return ac, nil
	}

	// add a failing policy host for every known policy
	failingPolicySet := service.NewMemFailingPolicySet()
	for polID := range pols {
		err := failingPolicySet.AddHost(polID, fleet.PolicySetHost{
			ID:       polID, // use policy ID as host ID, does not matter in the test
			Hostname: fmt.Sprintf("host%d.example", polID),
		})
		require.NoError(t, err)
	}
	// add a policy failure for a non-existent policy
	err := failingPolicySet.AddHost(11, fleet.PolicySetHost{
		ID:       11,
		Hostname: "host11.example",
	})
	require.NoError(t, err)

	var triggerCalls []policyAutomation
	err = TriggerFailingPoliciesAutomation(context.Background(), ds, kitlog.NewNopLogger(), failingPolicySet, func(pol *fleet.Policy, cfg FailingPolicyAutomationConfig) error {
		triggerCalls = append(triggerCalls, policyAutomation{pol.ID, cfg.AutomationType})

		hosts, err := failingPolicySet.ListHosts(pol.ID)
		require.NoError(t, err)
		err = failingPolicySet.RemoveHosts(pol.ID, hosts)
		require.NoError(t, err)

		return nil
	}, false)
	require.NoError(t, err)

	wantCalls := []policyAutomation{
		{1, FailingPolicyWebhook},
		{2, FailingPolicyWebhook},
		{4, FailingPolicyJira},
		{5, FailingPolicyJira},
		{7, FailingPolicyZendesk},
		{8, FailingPolicyZendesk},
		{12, FailingPolicyWebhook},
		{13, FailingPolicyWebhook},
	}
	// order of calls is undefined
	require.ElementsMatch(t, wantCalls, triggerCalls)

	// failing policies set is now cleared
	polSets, err := failingPolicySet.ListSets()
	require.NoError(t, err)
	var remainingHosts []uint
	for _, set := range polSets {
		hosts, err := failingPolicySet.ListHosts(set)
		require.NoError(t, err)
		for _, h := range hosts {
			remainingHosts = append(remainingHosts, h.ID)
		}
	}
	// there's one remaining host ID in the failing policy sets, and it's the one
	// with the invalid integration (it did not remove the failing policy set so
	// that it can retry once the integration is fixed).
	require.Len(t, remainingHosts, 1)
	require.Equal(t, remainingHosts[0], uint(15)) // host id used is the same as the policy id

	// trigger it again, should cause the same calls as the first time, but all
	// policy sets should be empty (no host to process).
	var countHosts int
	triggerCalls = triggerCalls[:0]
	err = TriggerFailingPoliciesAutomation(context.Background(), ds, kitlog.NewNopLogger(), failingPolicySet, func(pol *fleet.Policy, cfg FailingPolicyAutomationConfig) error {
		hosts, err := failingPolicySet.ListHosts(pol.ID)
		require.NoError(t, err)
		countHosts += len(hosts)
		triggerCalls = append(triggerCalls, policyAutomation{pol.ID, cfg.AutomationType})

		return nil
	}, false)
	require.NoError(t, err)

	// order of calls is undefined
	require.ElementsMatch(t, wantCalls, triggerCalls)
	require.Zero(t, countHosts)

	// test No team policy failure in Primo mode
	failingPolicySet = service.NewMemFailingPolicySet()
	err = failingPolicySet.AddHost(1, fleet.PolicySetHost{
		ID:       1, // use policy ID as host ID, does not matter in the test
		Hostname: fmt.Sprintf("host%d.example", 1),
	})
	require.NoError(t, err)

	triggerCalls = triggerCalls[:0]
	err = TriggerFailingPoliciesAutomation(context.Background(), ds, kitlog.NewNopLogger(), failingPolicySet, func(pol *fleet.Policy, cfg FailingPolicyAutomationConfig) error {
		triggerCalls = append(triggerCalls, policyAutomation{pol.ID, cfg.AutomationType})

		hosts, err := failingPolicySet.ListHosts(pol.ID)
		require.NoError(t, err)
		err = failingPolicySet.RemoveHosts(pol.ID, hosts)
		require.NoError(t, err)

		return nil
	}, true) // enablePrimo
	require.NoError(t, err)

	wantCalls = []policyAutomation{
		{1, FailingPolicyWebhook},
	}
	// order of calls is undefined
	require.ElementsMatch(t, wantCalls, triggerCalls)
}

func TestTriggerFailingPoliciesWithEnablePrimo(t *testing.T) {
	// Failing no-team policy automations in Primo mode, which are on No-team (team 0) should use the global config instead of a team config
	ds := new(mock.Store)

	pols := map[uint]*fleet.PolicyData{
		1: {ID: 1, Name: "pol-primo-no-team-1"}, // expected to trigger a webhook
		// TODO
		// 2: {ID: 2, Name: "pol-primo-no-team-2"}, // expected to trigger a Jira integration
		// 3: {ID: 2, Name: "pol-primo-no-team-3"}, // expected to trigger a Zendesk integration
	}
	ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) {
		pd, ok := pols[id]
		if !ok {
			return nil, ctxerr.Wrap(ctx, sql.ErrNoRows)
		}
		return &fleet.Policy{PolicyData: *pd}, nil
	}

	// Global config with webhook
	ac := &fleet.AppConfig{
		WebhookSettings: fleet.WebhookSettings{
			FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
				Enable:    true,
				PolicyIDs: []uint{1},
			},
		},
		// Integrations: fleet.Integrations{
		// 	Jira: []*fleet.JiraIntegration{
		// 		{URL: "http://j.com", ProjectKey: "A", Username: "jirauser", APIToken: "secret"},
		// 	},
		// 	Zendesk: []*fleet.ZendeskIntegration{
		// 		{URL: "http://z.com", GroupID: 1, Email: "zendesk@z.com", APIToken: "secret"},
		// 	},
		// },
		// ServerSettings: fleet.ServerSettings{
		// 	ServerURL: "https://fleet.example.com",
		// },
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return ac, nil
	}

	// add a failing policy host for every known policy
	failingPolicySet := service.NewMemFailingPolicySet()
	for polID := range pols {
		err := failingPolicySet.AddHost(polID, fleet.PolicySetHost{
			ID:       polID, // use policy ID as host ID, does not matter in the test
			Hostname: fmt.Sprintf("host%d.example", polID),
		})
		require.NoError(t, err)
	}

	var triggerCalls []policyAutomation
	err := TriggerFailingPoliciesAutomation(context.Background(), ds, kitlog.NewNopLogger(), failingPolicySet, func(pol *fleet.Policy, cfg FailingPolicyAutomationConfig) error {
		triggerCalls = append(triggerCalls, policyAutomation{pol.ID, cfg.AutomationType})

		hosts, err := failingPolicySet.ListHosts(pol.ID)
		require.NoError(t, err)
		err = failingPolicySet.RemoveHosts(pol.ID, hosts)
		require.NoError(t, err)

		return nil
	}, true) // enablePrimo
	require.NoError(t, err)

	wantCalls := []policyAutomation{
		{1, FailingPolicyWebhook},
		// TODO
		// {2, FailingPolicyJira},
		// {3, FailingPolicyZendesk},
	}
	// order of calls is undefined
	require.ElementsMatch(t, wantCalls, triggerCalls)

	// TODO - test Jira, Zendesk global integrations
	// Global config with Jira
	// ac := &fleet.AppConfig{
	// 	WebhookSettings: fleet.WebhookSettings{
	// 		FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
	// 			Enable:    true,
	// 			PolicyIDs: []uint{1},
	// 		},
	// 	},
	// 	Integrations: fleet.Integrations{
	// 		Jira: []*fleet.JiraIntegration{
	// 			{URL: "http://j.com", ProjectKey: "A", Username: "jirauser", APIToken: "secret"},
	// 		},
	// 		Zendesk: []*fleet.ZendeskIntegration{
	// 			{URL: "http://z.com", GroupID: 1, Email: "zendesk@z.com", APIToken: "secret"},
	// 		},
	// 	},
	// 	ServerSettings: fleet.ServerSettings{
	// 		ServerURL: "https://fleet.example.com",
	// 	},
	// }

	// ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
	// 	return ac, nil
	// }

	// // add a failing policy host for every known policy
	// failingPolicySet := service.NewMemFailingPolicySet()
	// for polID := range pols {
	// 	err := failingPolicySet.AddHost(polID, fleet.PolicySetHost{
	// 		ID:       polID, // use policy ID as host ID, does not matter in the test
	// 		Hostname: fmt.Sprintf("host%d.example", polID),
	// 	})
	// 	require.NoError(t, err)
	// }

	// var triggerCalls []policyAutomation
	// err := TriggerFailingPoliciesAutomation(context.Background(), ds, kitlog.NewNopLogger(), failingPolicySet, func(pol *fleet.Policy, cfg FailingPolicyAutomationConfig) error {
	// 	triggerCalls = append(triggerCalls, policyAutomation{pol.ID, cfg.AutomationType})

	// 	hosts, err := failingPolicySet.ListHosts(pol.ID)
	// 	require.NoError(t, err)
	// 	err = failingPolicySet.RemoveHosts(pol.ID, hosts)
	// 	require.NoError(t, err)

	// 	return nil
	// }, true) // enablePrimo
	// require.NoError(t, err)

	// wantCalls := []policyAutomation{
	// 	{1, FailingPolicyWebhook},
	// 	{2, FailingPolicyJira},
	// 	{3, FailingPolicyZendesk},
	// }
	// // order of calls is undefined
	// require.ElementsMatch(t, wantCalls, triggerCalls)
	// // Global config with webhook
	// ac := &fleet.AppConfig{
	// 	WebhookSettings: fleet.WebhookSettings{
	// 		FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
	// 			Enable:    true,
	// 			PolicyIDs: []uint{1},
	// 		},
	// 	},
	// 	Integrations: fleet.Integrations{
	// 		Jira: []*fleet.JiraIntegration{
	// 			{URL: "http://j.com", ProjectKey: "A", Username: "jirauser", APIToken: "secret"},
	// 		},
	// 		Zendesk: []*fleet.ZendeskIntegration{
	// 			{URL: "http://z.com", GroupID: 1, Email: "zendesk@z.com", APIToken: "secret"},
	// 		},
	// 	},
	// 	ServerSettings: fleet.ServerSettings{
	// 		ServerURL: "https://fleet.example.com",
	// 	},
	// }

	// ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
	// 	return ac, nil
	// }

	// // add a failing policy host for every known policy
	// failingPolicySet := service.NewMemFailingPolicySet()
	// for polID := range pols {
	// 	err := failingPolicySet.AddHost(polID, fleet.PolicySetHost{
	// 		ID:       polID, // use policy ID as host ID, does not matter in the test
	// 		Hostname: fmt.Sprintf("host%d.example", polID),
	// 	})
	// 	require.NoError(t, err)
	// }

	// var triggerCalls []policyAutomation
	// err := TriggerFailingPoliciesAutomation(context.Background(), ds, kitlog.NewNopLogger(), failingPolicySet, func(pol *fleet.Policy, cfg FailingPolicyAutomationConfig) error {
	// 	triggerCalls = append(triggerCalls, policyAutomation{pol.ID, cfg.AutomationType})

	// 	hosts, err := failingPolicySet.ListHosts(pol.ID)
	// 	require.NoError(t, err)
	// 	err = failingPolicySet.RemoveHosts(pol.ID, hosts)
	// 	require.NoError(t, err)

	// 	return nil
	// }, true) // enablePrimo
	// require.NoError(t, err)

	// wantCalls := []policyAutomation{
	// 	{1, FailingPolicyWebhook},
	// 	{2, FailingPolicyJira},
	// 	{3, FailingPolicyZendesk},
	// }
	// // order of calls is undefined
	// require.ElementsMatch(t, wantCalls, triggerCalls)
}
