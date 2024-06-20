// Package policies implements features to handle policy-related processing.
package policies

import (
	"context"
	"database/sql"
	"errors"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// FailingPolicyAutomationType is the type of automations supported for
// failing policies.
type FailingPolicyAutomationType string

// List of supported failing policy automation types.
const (
	FailingPolicyWebhook FailingPolicyAutomationType = "webhook"
	FailingPolicyJira    FailingPolicyAutomationType = "jira"
	FailingPolicyZendesk FailingPolicyAutomationType = "zendesk"
)

// FailingPolicyAutomationConfig holds the configuration for proessing a
// failing policy to send to the configured automation.
type FailingPolicyAutomationConfig struct {
	AutomationType FailingPolicyAutomationType
	PolicyIDs      map[uint]bool
	WebhookURL     *url.URL // for webhook automation type only
	HostBatchSize  int      // for webhook automation type only
}

// TriggerFailingPoliciesAutomation triggers an automation for failing
// policies. It receives a function that takes care of sending the failed
// policy as argument, that function receives the type of automation that is
// enabled for that policy.
func TriggerFailingPoliciesAutomation(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	failingPoliciesSet fleet.FailingPolicySet,
	sendFunc func(*fleet.Policy, FailingPolicyAutomationConfig) error,
) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting app config")
	}

	// build the global automation configuration
	var globalCfg FailingPolicyAutomationConfig

	globalAutomation := getActiveAutomation(appConfig.WebhookSettings.FailingPoliciesWebhook, appConfig.Integrations)
	if globalAutomation != "" {
		level.Debug(logger).Log("global_failing_policy", "enabled", "automation", globalAutomation)
	} else {
		level.Debug(logger).Log("global_failing_policy", "disabled")
	}

	if globalAutomation != "" {
		// if global failing policies automation is enabled, keep a set of
		// policies for which the automation must run.
		globalCfg.AutomationType = globalAutomation
		globalSettings := appConfig.WebhookSettings.FailingPoliciesWebhook

		polIDs := make(map[uint]bool, len(globalSettings.PolicyIDs))
		for _, pID := range globalSettings.PolicyIDs {
			polIDs[pID] = true
		}
		globalCfg.PolicyIDs = polIDs

		if globalAutomation == FailingPolicyWebhook {
			wurl, err := url.Parse(globalSettings.DestinationURL)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "parse global webhook url: %s", globalSettings.DestinationURL)
			}
			globalCfg.WebhookURL = wurl
			globalCfg.HostBatchSize = globalSettings.HostBatchSize
		}
	}

	// prepare the per-team configuration caches
	getTeam := makeTeamConfigCache(ds, appConfig.Integrations)

	policySets, err := failingPoliciesSet.ListSets()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list policies set")
	}

	for _, policyID := range policySets {
		policy, err := ds.Policy(ctx, policyID)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			level.Debug(logger).Log("msg", "skipping failing policy, deleted", "policyID", policyID)
			if err := failingPoliciesSet.RemoveSet(policyID); err != nil {
				level.Error(logger).Log("msg", "failed to remove policy from set", "policyID", policyID, "err", err)
			}
			continue
		case err != nil:
			return ctxerr.Wrapf(ctx, err, "get policy: %d", policyID)
		}

		if policy.TeamID != nil {
			// handle team policy
			teamCfg, err := getTeam(ctx, *policy.TeamID)
			switch {
			case errors.Is(err, sql.ErrNoRows):
				// shouldn't happen, unless the team was deleted after the policy was retrieved above
				level.Debug(logger).Log("msg", "team does not exist", "teamID", *policy.TeamID)
				if err := failingPoliciesSet.RemoveSet(policy.ID); err != nil {
					level.Error(logger).Log("msg", "failed to remove policy from set", "policyID", policy.ID, "err", err)
				}
				continue
			case err != nil:
				level.Error(logger).Log("msg", "failed to get team", "teamID", *policy.TeamID, "err", err)
				continue
			}

			if teamCfg.AutomationType == "" {
				continue
			}

			if !teamCfg.PolicyIDs[policy.ID] {
				level.Debug(logger).Log("msg", "skipping failing policy, not found in team policy IDs", "policyID", policyID)
				if err := failingPoliciesSet.RemoveSet(policy.ID); err != nil {
					level.Error(logger).Log("msg", "failed to remove policy from set", "policyID", policyID, "err", err)
				}
				continue
			}

			if err := sendFunc(policy, teamCfg); err != nil {
				level.Error(logger).Log("msg", "failed to send failing policies", "policyID", policy.ID, "err", err)
			}
			continue
		}

		// handle global policy
		if !globalCfg.PolicyIDs[policy.ID] {
			level.Debug(logger).Log("msg", "skipping failing policy, not found in global policy IDs", "policyID", policyID)
			if err := failingPoliciesSet.RemoveSet(policy.ID); err != nil {
				level.Error(logger).Log("msg", "failed to remove policy from set", "policyID", policyID, "err", err)
			}
			continue
		}

		if err := sendFunc(policy, globalCfg); err != nil {
			level.Error(logger).Log("msg", "failed to send failing policies", "policyID", policy.ID, "err", err)
		}
	}

	return nil
}

func makeTeamConfigCache(ds fleet.Datastore, globalIntgs fleet.Integrations) func(ctx context.Context, teamID uint) (FailingPolicyAutomationConfig, error) {
	teamCfgs := make(map[uint]FailingPolicyAutomationConfig)

	return func(ctx context.Context, teamID uint) (FailingPolicyAutomationConfig, error) {
		cfg, ok := teamCfgs[teamID]
		if ok {
			return cfg, nil
		}

		team, err := ds.Team(ctx, teamID)
		if err != nil {
			return cfg, ctxerr.Wrapf(ctx, err, "get team: %d", teamID)
		}

		intgs, err := team.Config.Integrations.MatchWithIntegrations(globalIntgs)
		if err != nil {
			return cfg, ctxerr.Wrap(ctx, err, "map team integrations to global integrations")
		}

		teamAutomation := getActiveAutomation(team.Config.WebhookSettings.FailingPoliciesWebhook, intgs)
		teamCfg := FailingPolicyAutomationConfig{
			AutomationType: teamAutomation,
		}

		if teamAutomation != "" {
			settings := team.Config.WebhookSettings.FailingPoliciesWebhook
			polIDs := make(map[uint]bool, len(settings.PolicyIDs))
			for _, pID := range settings.PolicyIDs {
				polIDs[pID] = true
			}
			teamCfg.PolicyIDs = polIDs

			if teamAutomation == FailingPolicyWebhook {
				wurl, err := url.Parse(settings.DestinationURL)
				if err != nil {
					return cfg, ctxerr.Wrapf(ctx, err, "parse webhook url: %s", settings.DestinationURL)
				}
				teamCfg.WebhookURL = wurl
				teamCfg.HostBatchSize = settings.HostBatchSize
			}
		}
		teamCfgs[teamID] = teamCfg

		return teamCfg, nil
	}
}

func getActiveAutomation(webhook fleet.FailingPoliciesWebhookSettings, intgs fleet.Integrations) FailingPolicyAutomationType {
	// only one automation (i.e. webhook or integration) can be enabled at a
	// time, enforced when updating the appconfig or the team config.
	if webhook.Enable {
		return FailingPolicyWebhook
	}

	// check for jira integrations
	for _, j := range intgs.Jira {
		if j.EnableFailingPolicies {
			return FailingPolicyJira
		}
	}

	// check for zendesk integrations
	for _, z := range intgs.Zendesk {
		if z.EnableFailingPolicies {
			return FailingPolicyZendesk
		}
	}
	return ""
}
