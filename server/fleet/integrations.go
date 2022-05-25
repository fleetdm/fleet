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

// TeamJiraIntegration configures an instance of an integration with the Jira
// system for a team.
type TeamJiraIntegration struct {
	URL                   string `json:"url"`
	Username              string `json:"username"`
	APIToken              string `json:"api_token"`
	ProjectKey            string `json:"project_key"`
	EnableFailingPolicies bool   `json:"enable_failing_policies"`
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
