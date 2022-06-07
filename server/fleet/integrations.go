package fleet

import (
	"context"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
)

// TeamIntegrations contains the configuration for external services'
// integrations for a specific team.
type TeamIntegrations struct {
	Jira    []*TeamJiraIntegration    `json:"jira"`
	Zendesk []*TeamZendeskIntegration `json:"zendesk"`
}

// ToIntegrations converts a TeamIntegrations to an Integrations struct.
func (ti TeamIntegrations) ToIntegrations() Integrations {
	var intgs Integrations
	intgs.Jira = make([]*JiraIntegration, len(ti.Jira))
	for i, j := range ti.Jira {
		intgs.Jira[i] = j.ToJiraIntegration()
	}
	intgs.Zendesk = make([]*ZendeskIntegration, len(ti.Zendesk))
	for i, z := range ti.Zendesk {
		intgs.Zendesk[i] = z.ToZendeskIntegration()
	}
	return intgs
}

// TeamJiraIntegration configures an instance of an integration with the Jira
// system for a team.
type TeamJiraIntegration struct {
	URL                   string `json:"url"`
	Username              string `json:"username"`
	APIToken              string `json:"api_token"`
	ProjectKey            string `json:"project_key"`
	EnableFailingPolicies bool   `json:"enable_failing_policies"`
}

// ToJiraIntegration converts a TeamJiraIntegration to a JiraIntegration
// struct, leaving additional fields to their zero value.
func (ti TeamJiraIntegration) ToJiraIntegration() *JiraIntegration {
	return &JiraIntegration{TeamJiraIntegration: ti}
}

// TeamZendeskIntegration configures an instance of an integration with the
// external Zendesk service for a team.
type TeamZendeskIntegration struct {
	URL                   string `json:"url"`
	Email                 string `json:"email"`
	APIToken              string `json:"api_token"`
	GroupID               int64  `json:"group_id"`
	EnableFailingPolicies bool   `json:"enable_failing_policies"`
}

// ToZendeskIntegration converts a TeamZendeskIntegration to a ZendeskIntegration
// struct, leaving additional fields to their zero value.
func (ti TeamZendeskIntegration) ToZendeskIntegration() *ZendeskIntegration {
	return &ZendeskIntegration{TeamZendeskIntegration: ti}
}

// JiraIntegration configures an instance of an integration with the Jira
// system.
type JiraIntegration struct {
	// It is a superset of TeamJiraIntegration.
	TeamJiraIntegration

	EnableSoftwareVulnerabilities bool `json:"enable_software_vulnerabilities"`
}

// IndexJiraIntegrations indexes the provided Jira integrations in a map keyed
// by the project key. It returns an error if a duplicate configuration is
// found for the same project key. This is typically used to index the original
// integrations before applying the changes requested to modify the AppConfig.
//
// Note that the returned map uses non-pointer JiraIntegration struct values,
// so that any changes to the original value does not modify the value in the
// map. This is important because of how changes are merged with the original
// AppConfig when modifying it.
func IndexJiraIntegrations(jiraIntgs []*JiraIntegration) (map[string]JiraIntegration, error) {
	byProjKey := make(map[string]JiraIntegration, len(jiraIntgs))
	for _, intg := range jiraIntgs {
		if _, ok := byProjKey[intg.ProjectKey]; ok {
			return nil, fmt.Errorf("duplicate Jira integration for project key %s", intg.ProjectKey)
		}
		byProjKey[intg.ProjectKey] = *intg
	}
	return byProjKey, nil
}

// IndexTeamJiraIntegrations is the same as IndexJiraIntegrations, but for
// team-specific integration structs.
func IndexTeamJiraIntegrations(teamJiraIntgs []*TeamJiraIntegration) (map[string]TeamJiraIntegration, error) {
	jiraIntgs := make([]*JiraIntegration, len(teamJiraIntgs))
	for i, t := range teamJiraIntgs {
		jiraIntgs[i] = t.ToJiraIntegration()
	}

	indexed, err := IndexJiraIntegrations(jiraIntgs)
	if err != nil {
		return nil, err
	}

	teamIndexed := make(map[string]TeamJiraIntegration, len(indexed))
	for k, v := range indexed {
		teamIndexed[k] = v.TeamJiraIntegration
	}
	return teamIndexed, nil
}

// ValidateJiraIntegrations validates that the merge of the original and new
// Jira integrations does not result in any duplicate configuration, and that
// each modified or added integration can successfully connect to the external
// Jira service.
//
// On successful return, the newJiraIntgs slice is ready to be saved - it may
// have been updated using the original integrations if the API token was
// missing.
func ValidateJiraIntegrations(ctx context.Context, oriJiraIntgsByProjKey map[string]JiraIntegration, newJiraIntgs []*JiraIntegration) error {
	newByProjKey := make(map[string]*JiraIntegration, len(newJiraIntgs))
	for i, new := range newJiraIntgs {
		// TODO(mna): uniqueness should be defined by URL + ProjectKey
		// first check for project key uniqueness
		if _, ok := newByProjKey[new.ProjectKey]; ok {
			return fmt.Errorf("duplicate Jira integration for project key %s", new.ProjectKey)
		}
		newByProjKey[new.ProjectKey] = new

		// check if existing integration is being edited
		if old, ok := oriJiraIntgsByProjKey[new.ProjectKey]; ok {
			if old == *new {
				// no further validation for unchanged integration
				continue
			}
			// use stored API token if request does not contain new token
			// intended only as a short-term accommodation for the frontend
			// will be redesigned in dedicated endpoint for integration config
			if new.APIToken == "" || new.APIToken == MaskedPassword {
				new.APIToken = old.APIToken
			}
		}

		// new or updated, test it
		if err := makeTestJiraRequest(ctx, new); err != nil {
			return fmt.Errorf("Jira integration at index %d: %w", i, err)
		}
	}
	return nil
}

// ValidateTeamJiraIntegrations applies the same validations as
// ValidateJiraIntegrations, but for team-specific integration structs.
func ValidateTeamJiraIntegrations(ctx context.Context, oriTeamJiraIntgsByProjKey map[string]TeamJiraIntegration, newTeamJiraIntgs []*TeamJiraIntegration) error {
	newJiraIntgs := make([]*JiraIntegration, len(newTeamJiraIntgs))
	for i, t := range newTeamJiraIntgs {
		newJiraIntgs[i] = t.ToJiraIntegration()
	}

	oriJiraIntgsByProjKey := make(map[string]JiraIntegration, len(oriTeamJiraIntgsByProjKey))
	for k, v := range oriTeamJiraIntgsByProjKey {
		oriJiraIntgsByProjKey[k] = *v.ToJiraIntegration()
	}

	if err := ValidateJiraIntegrations(ctx, oriJiraIntgsByProjKey, newJiraIntgs); err != nil {
		return err
	}

	// assign back the newJiraIntgs to newTeamJiraIntgs, as they may have been
	// updated by the call and we need to pass that change back to the caller
	for i, v := range newJiraIntgs {
		teamJira := newTeamJiraIntgs[i]
		*teamJira = v.TeamJiraIntegration
	}
	return nil
}

// IntegrationTestError is the type of error returned when a validation of an
// external service integration (e.g. Jira, Zendesk, etc.) failed due to the
// connection test to that external service.
//
// This is typically used to return a different status code in that case from
// an HTTP endpoint.
type IntegrationTestError struct {
	Err error
}

// Error implements the error interface for IntegrationTestError.
func (e IntegrationTestError) Error() string {
	return e.Err.Error()
}

func makeTestJiraRequest(ctx context.Context, intg *JiraIntegration) error {
	if intg.APIToken == "" || intg.APIToken == MaskedPassword {
		return IntegrationTestError{Err: errors.New("Jira integration request failed: missing or invalid API token")}
	}
	client, err := externalsvc.NewJiraClient(&externalsvc.JiraOptions{
		BaseURL:           intg.URL,
		BasicAuthUsername: intg.Username,
		BasicAuthPassword: intg.APIToken,
		ProjectKey:        intg.ProjectKey,
	})
	if err != nil {
		return IntegrationTestError{Err: fmt.Errorf("Jira integration request failed: %w", err)}
	}
	if _, err := client.GetProject(ctx); err != nil {
		return IntegrationTestError{Err: fmt.Errorf("Jira integration request failed: %w", err)}
	}
	return nil
}

// ZendeskIntegration configures an instance of an integration with the external Zendesk service.
type ZendeskIntegration struct {
	// It is a superset of TeamZendeskIntegration.
	TeamZendeskIntegration

	EnableSoftwareVulnerabilities bool `json:"enable_software_vulnerabilities"`
}

// IndexZendeskIntegrations indexes the provided Zendesk integrations in a map
// keyed by the group ID. It returns an error if a duplicate configuration is
// found for the same group ID. This is typically used to index the original
// integrations before applying the changes requested to modify the AppConfig.
//
// Note that the returned map uses non-pointer ZendeskIntegration struct
// values, so that any changes to the original value does not modify the value
// in the map. This is important because of how changes are merged with the
// original AppConfig when modifying it.
func IndexZendeskIntegrations(zendeskIntgs []*ZendeskIntegration) (map[int64]ZendeskIntegration, error) {
	byGroupID := make(map[int64]ZendeskIntegration, len(zendeskIntgs))
	for _, intg := range zendeskIntgs {
		if _, ok := byGroupID[intg.GroupID]; ok {
			return nil, fmt.Errorf("duplicate Zendesk integration for group id %v", intg.GroupID)
		}
		byGroupID[intg.GroupID] = *intg
	}
	return byGroupID, nil
}

// IndexTeamZendeskIntegrations is the same as IndexZendeskIntegrations, but
// for team-specific integration structs.
func IndexTeamZendeskIntegrations(teamZendeskIntgs []*TeamZendeskIntegration) (map[int64]TeamZendeskIntegration, error) {
	zendeskIntgs := make([]*ZendeskIntegration, len(teamZendeskIntgs))
	for i, t := range teamZendeskIntgs {
		zendeskIntgs[i] = t.ToZendeskIntegration()
	}

	indexed, err := IndexZendeskIntegrations(zendeskIntgs)
	if err != nil {
		return nil, err
	}

	teamIndexed := make(map[int64]TeamZendeskIntegration, len(indexed))
	for k, v := range indexed {
		teamIndexed[k] = v.TeamZendeskIntegration
	}
	return teamIndexed, nil
}

// ValidateZendeskIntegrations validates that the merge of the original and
// new Zendesk integrations does not result in any duplicate configuration,
// and that each modified or added integration can successfully connect to the
// external Zendesk service.
//
// On successful return, the newZendeskIntgs slice is ready to be saved - it
// may have been updated using the original integrations if the API token was
// missing.
func ValidateZendeskIntegrations(ctx context.Context, oriZendeskIntgsByGroupID map[int64]ZendeskIntegration, newZendeskIntgs []*ZendeskIntegration) error {
	newByGroupID := make(map[int64]*ZendeskIntegration, len(newZendeskIntgs))
	for i, new := range newZendeskIntgs {
		// TODO(mna): uniqueness should be defined by URL + GroupID
		// first check for group id uniqueness
		if _, ok := newByGroupID[new.GroupID]; ok {
			return fmt.Errorf("duplicate Zendesk integration for group id %v", new.GroupID)
		}
		newByGroupID[new.GroupID] = new

		// check if existing integration is being edited
		if old, ok := oriZendeskIntgsByGroupID[new.GroupID]; ok {
			if old == *new {
				// no further validation for unchanged integration
				continue
			}
			// use stored API token if request does not contain new token
			// intended only as a short-term accommodation for the frontend
			// will be redesigned in dedicated endpoint for integration config
			if new.APIToken == "" || new.APIToken == MaskedPassword {
				new.APIToken = old.APIToken
			}
		}

		// new or updated, test it
		if err := makeTestZendeskRequest(ctx, new); err != nil {
			return fmt.Errorf("Zendesk integration at index %d: %w", i, err)
		}
	}
	return nil
}

// ValidateTeamZendeskIntegrations applies the same validations as
// ValidateZendeskIntegrations, but for team-specific integration structs.
func ValidateTeamZendeskIntegrations(ctx context.Context, oriTeamZendeskIntgsByGroupID map[int64]TeamZendeskIntegration, newTeamZendeskIntgs []*TeamZendeskIntegration) error {
	newZendeskIntgs := make([]*ZendeskIntegration, len(newTeamZendeskIntgs))
	for i, t := range newTeamZendeskIntgs {
		newZendeskIntgs[i] = t.ToZendeskIntegration()
	}

	oriZendeskIntgsByGroupID := make(map[int64]ZendeskIntegration, len(oriTeamZendeskIntgsByGroupID))
	for k, v := range oriTeamZendeskIntgsByGroupID {
		oriZendeskIntgsByGroupID[k] = *v.ToZendeskIntegration()
	}

	if err := ValidateZendeskIntegrations(ctx, oriZendeskIntgsByGroupID, newZendeskIntgs); err != nil {
		return err
	}

	// assign back the newZendeskIntgs to newTeamZendeskIntgs, as they may have
	// been updated by the call and we need to pass that change back to the
	// caller
	for i, v := range newZendeskIntgs {
		teamZendesk := newTeamZendeskIntgs[i]
		*teamZendesk = v.TeamZendeskIntegration
	}
	return nil
}

func makeTestZendeskRequest(ctx context.Context, intg *ZendeskIntegration) error {
	if intg.APIToken == "" || intg.APIToken == MaskedPassword {
		return IntegrationTestError{Err: errors.New("Zendesk integration request failed: missing or invalid API token")}
	}
	client, err := externalsvc.NewZendeskClient(&externalsvc.ZendeskOptions{
		URL:      intg.URL,
		Email:    intg.Email,
		APIToken: intg.APIToken,
		GroupID:  intg.GroupID,
	})
	if err != nil {
		return IntegrationTestError{Err: fmt.Errorf("Zendesk integration request failed: %w", err)}
	}
	grp, err := client.GetGroup(ctx)
	if err != nil {
		return IntegrationTestError{Err: fmt.Errorf("Zendesk integration request failed: %w", err)}
	}
	if grp.ID != intg.GroupID {
		return IntegrationTestError{Err: fmt.Errorf("Zendesk integration request failed: no matching group id: received %d, expected %d", grp.ID, intg.GroupID)}
	}
	return nil
}

// Integrations configures the integrations with external systems.
type Integrations struct {
	Jira    []*JiraIntegration    `json:"jira"`
	Zendesk []*ZendeskIntegration `json:"zendesk"`
}

// ValidateEnabledVulnerabilitiesIntegrations checks that a single integration
// is enabled for vulnerabilities. It adds any error it finds to the invalid
// argument error, that can then be checked after the call for errors using
// invalid.HasErrors.
func ValidateEnabledVulnerabilitiesIntegrations(webhook VulnerabilitiesWebhookSettings, intgs Integrations, invalid *InvalidArgumentError) {
	webhookEnabled := webhook.Enable
	var jiraEnabledCount int
	for _, jira := range intgs.Jira {
		if jira.EnableSoftwareVulnerabilities {
			jiraEnabledCount++
		}
	}
	var zendeskEnabledCount int
	for _, zendesk := range intgs.Zendesk {
		if zendesk.EnableSoftwareVulnerabilities {
			zendeskEnabledCount++
		}
	}

	if webhookEnabled && (jiraEnabledCount > 0 || zendeskEnabledCount > 0) {
		invalid.Append("vulnerabilities", "cannot enable both webhook vulnerabilities and integration automations")
	}
	if jiraEnabledCount > 0 && zendeskEnabledCount > 0 {
		invalid.Append("vulnerabilities", "cannot enable both jira integration and zendesk automations")
	}
	if jiraEnabledCount > 1 {
		invalid.Append("vulnerabilities", "cannot enable more than one jira integration")
	}
	if zendeskEnabledCount > 1 {
		invalid.Append("vulnerabilities", "cannot enable more than one zendesk integration")
	}
}

// ValidateEnabledFailingPoliciesIntegrations checks that a single integration
// is enabled for failing policies. It adds any error it finds to the invalid
// argument error, that can then be checked after the call for errors using
// invalid.HasErrors.
func ValidateEnabledFailingPoliciesIntegrations(webhook FailingPoliciesWebhookSettings, intgs Integrations, invalid *InvalidArgumentError) {
	webhookEnabled := webhook.Enable
	var jiraEnabledCount int
	for _, jira := range intgs.Jira {
		if jira.EnableFailingPolicies {
			jiraEnabledCount++
		}
	}
	var zendeskEnabledCount int
	for _, zendesk := range intgs.Zendesk {
		if zendesk.EnableFailingPolicies {
			zendeskEnabledCount++
		}
	}

	if webhookEnabled && (jiraEnabledCount > 0 || zendeskEnabledCount > 0) {
		invalid.Append("failing policies", "cannot enable both webhook failing policies and integration automations")
	}
	if jiraEnabledCount > 0 && zendeskEnabledCount > 0 {
		invalid.Append("failing policies", "cannot enable both jira and zendesk automations")
	}
	if jiraEnabledCount > 1 {
		invalid.Append("failing policies", "cannot enable more than one jira integration")
	}
	if zendeskEnabledCount > 1 {
		invalid.Append("failing policies", "cannot enable more than one zendesk integration")
	}
}

// ValidateEnabledFailingPoliciesTeamIntegrations is like
// ValidateEnabledFailingPoliciesIntegrations, but for team-specific
// integration structs.
func ValidateEnabledFailingPoliciesTeamIntegrations(webhook FailingPoliciesWebhookSettings, teamIntgs TeamIntegrations, invalid *InvalidArgumentError) {
	intgs := teamIntgs.ToIntegrations()
	ValidateEnabledFailingPoliciesIntegrations(webhook, intgs, invalid)
}
