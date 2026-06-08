// Package policies implements features to handle policy-related processing.
package policies

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
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

// FailingPolicyAutomationConfig holds the configuration for processing a
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
	logger *slog.Logger,
	failingPoliciesSet fleet.FailingPolicySet,
	sendFunc func(*fleet.Policy, FailingPolicyAutomationConfig) error,
) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting app config")
	}

	// build the global automation configuration
	globalAutomationCfg, err := buildFailingPolicyAutomationConfig(appConfig.WebhookSettings.FailingPoliciesWebhook, appConfig.Integrations)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build global automation config")
	}

	if globalAutomationCfg.AutomationType != "" {
		logger.DebugContext(ctx, "global failing policy enabled", "automation", string(globalAutomationCfg.AutomationType))
	} else {
		logger.DebugContext(ctx, "global failing policy disabled")
	}

	// prepare the per-team configuration caches
	getTeam := makeTeamConfigCache(ds, appConfig.Integrations)
	getDefaultTeamConfig := makeDefaultTeamConfigCache(ds, appConfig.Integrations, logger)

	policySets, err := failingPoliciesSet.ListSets()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list policies set")
	}

	for _, policyID := range policySets {
		policy, err := ds.Policy(ctx, policyID)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			logger.DebugContext(ctx, "skipping failing policy, deleted", "policyID", policyID)
			if err := failingPoliciesSet.RemoveSet(policyID); err != nil {
				logger.ErrorContext(ctx, "failed to remove policy from set", "policyID", policyID, "err", err)
			}
			continue
		case err != nil:
			return ctxerr.Wrapf(ctx, err, "get policy: %d", policyID)
		}

		// Determine which configuration to use based on policy's team
		switch {
		case policy.TeamID == nil:
			// Global policy - use global config
			if !globalAutomationCfg.PolicyIDs[policy.ID] {
				logger.DebugContext(ctx, "skipping failing policy, not found in global policy IDs", "policyID", policyID)
				if err := failingPoliciesSet.RemoveSet(policy.ID); err != nil {
					logger.ErrorContext(ctx, "failed to remove policy from set", "policyID", policyID, "err", err)
				}
				continue
			}

			if err := sendFunc(policy, globalAutomationCfg); err != nil {
				logger.ErrorContext(ctx, "failed to send failing policies", "policyID", policy.ID, "err", err)
			}
		case *policy.TeamID == 0:
			// "No Team" policy - use default team config
			cfg, err := getDefaultTeamConfig(ctx)
			if err != nil {
				// Error already logged in getDefaultTeamConfig
				continue
			}

			if cfg.AutomationType == "" {
				logger.DebugContext(ctx, "default team automation disabled", "policyID", policyID)
				if err := failingPoliciesSet.RemoveSet(policy.ID); err != nil {
					logger.ErrorContext(ctx, "failed to remove policy from set", "policyID", policyID, "err", err)
				}
				continue
			}

			if !cfg.PolicyIDs[policy.ID] {
				logger.DebugContext(ctx, "skipping failing policy, not found in default team policy IDs", "policyID", policyID)
				if err := failingPoliciesSet.RemoveSet(policy.ID); err != nil {
					logger.ErrorContext(ctx, "failed to remove policy from set", "policyID", policyID, "err", err)
				}
				continue
			}

			if err := sendFunc(policy, cfg); err != nil {
				logger.ErrorContext(ctx, "failed to send failing policies", "policyID", policy.ID, "err", err)
			}

		default:
			// Regular team policy - use team config
			teamCfg, err := getTeam(ctx, *policy.TeamID)
			switch {
			case errors.Is(err, sql.ErrNoRows):
				// shouldn't happen, unless the team was deleted after the policy was retrieved above
				logger.DebugContext(ctx, "team does not exist", "teamID", *policy.TeamID)
				if err := failingPoliciesSet.RemoveSet(policy.ID); err != nil {
					logger.ErrorContext(ctx, "failed to remove policy from set", "policyID", policy.ID, "err", err)
				}
				continue
			case err != nil:
				logger.ErrorContext(ctx, "failed to get team", "teamID", *policy.TeamID, "err", err)
				continue
			}

			if teamCfg.AutomationType == "" {
				logger.DebugContext(ctx, "team automation disabled", "teamID", *policy.TeamID, "policyID", policyID)
				if err := failingPoliciesSet.RemoveSet(policy.ID); err != nil {
					logger.ErrorContext(ctx, "failed to remove policy from set", "policyID", policyID, "err", err)
				}
				continue
			}

			if !teamCfg.PolicyIDs[policy.ID] {
				logger.DebugContext(ctx, "skipping failing policy, not found in team policy IDs", "policyID", policyID)
				if err := failingPoliciesSet.RemoveSet(policy.ID); err != nil {
					logger.ErrorContext(ctx, "failed to remove policy from set", "policyID", policyID, "err", err)
				}
				continue
			}

			if err := sendFunc(policy, teamCfg); err != nil {
				logger.ErrorContext(ctx, "failed to send failing policies", "policyID", policy.ID, "err", err)
			}
		}
	}

	return nil
}

func buildFailingPolicyAutomationConfig(webhookSettings fleet.FailingPoliciesWebhookSettings, intgs fleet.Integrations) (FailingPolicyAutomationConfig, error) {
	cfg := FailingPolicyAutomationConfig{}

	automation := getActiveAutomation(webhookSettings, intgs)
	if automation == "" {
		return cfg, nil
	}

	cfg.AutomationType = automation

	// Build policy IDs map
	polIDs := make(map[uint]bool, len(webhookSettings.PolicyIDs))
	for _, pID := range webhookSettings.PolicyIDs {
		polIDs[pID] = true
	}
	cfg.PolicyIDs = polIDs

	// Parse webhook URL if needed
	if automation == FailingPolicyWebhook {
		wurl, err := url.Parse(webhookSettings.DestinationURL)
		if err != nil {
			return cfg, fmt.Errorf("parse webhook url %s: %w", webhookSettings.DestinationURL, err)
		}
		cfg.WebhookURL = wurl
		cfg.HostBatchSize = webhookSettings.HostBatchSize
	}

	return cfg, nil
}

func makeTeamConfigCache(ds fleet.Datastore, globalIntgs fleet.Integrations) func(ctx context.Context, teamID uint) (FailingPolicyAutomationConfig, error) {
	teamCfgs := make(map[uint]FailingPolicyAutomationConfig)

	return func(ctx context.Context, teamID uint) (FailingPolicyAutomationConfig, error) {
		cfg, ok := teamCfgs[teamID]
		if ok {
			return cfg, nil
		}

		team, err := ds.TeamLite(ctx, teamID)
		if err != nil {
			return cfg, ctxerr.Wrapf(ctx, err, "get team: %d", teamID)
		}

		intgs, err := team.Config.Integrations.MatchWithIntegrations(globalIntgs)
		if err != nil {
			return cfg, ctxerr.Wrap(ctx, err, "map team integrations to global integrations")
		}

		teamCfg, err := buildFailingPolicyAutomationConfig(team.Config.WebhookSettings.FailingPoliciesWebhook, intgs)
		if err != nil {
			return cfg, ctxerr.Wrap(ctx, err, "build team automation config")
		}

		teamCfgs[teamID] = teamCfg
		return teamCfg, nil
	}
}

func makeDefaultTeamConfigCache(ds fleet.Datastore, globalIntgs fleet.Integrations, logger *slog.Logger) func(ctx context.Context) (FailingPolicyAutomationConfig, error) {
	var cached *FailingPolicyAutomationConfig
	var cachedErr error

	return func(ctx context.Context) (FailingPolicyAutomationConfig, error) {
		// Return cached result if already loaded
		if cached != nil || cachedErr != nil {
			if cachedErr != nil {
				return FailingPolicyAutomationConfig{}, cachedErr
			}
			return *cached, nil
		}

		// Load default team configuration
		var cfg FailingPolicyAutomationConfig
		defaultTeamConfig, err := ds.DefaultTeamConfig(ctx)
		if err != nil {
			cachedErr = err
			logger.ErrorContext(ctx, "failed to get default team config", "err", err)
			return cfg, err
		}

		intgs, err := defaultTeamConfig.Integrations.MatchWithIntegrations(globalIntgs)
		if err != nil {
			cachedErr = err
			logger.ErrorContext(ctx, "failed to match default team integrations", "err", err)
			return cfg, err
		}

		cfg, err = buildFailingPolicyAutomationConfig(defaultTeamConfig.WebhookSettings.FailingPoliciesWebhook, intgs)
		if err != nil {
			// Log error but don't fail - just disable automation
			logger.ErrorContext(ctx, "failed to build default team automation config", "err", err)
			cfg = FailingPolicyAutomationConfig{} // Return empty config
		}

		cached = &cfg
		return cfg, nil
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
