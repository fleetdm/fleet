package fleet

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
)

// TeamIntegrations contains the configuration for external services'
// integrations for a specific team.
type TeamIntegrations struct {
	Jira           []*TeamJiraIntegration         `json:"jira"`
	Zendesk        []*TeamZendeskIntegration      `json:"zendesk"`
	GoogleCalendar *TeamGoogleCalendarIntegration `json:"google_calendar"`
}

// MatchWithIntegrations matches the team integrations to their corresponding
// global integrations found in globalIntgs, returning the resulting
// integrations struct. It returns an error if any team integration does not
// map to a global integration, but it will still return the complete list
// of integrations that do match.
func (ti TeamIntegrations) MatchWithIntegrations(globalIntgs Integrations) (Integrations, error) {
	var result Integrations

	jiraIntgs, err := IndexJiraIntegrations(globalIntgs.Jira)
	if err != nil {
		return result, err
	}
	zendeskIntgs, err := IndexZendeskIntegrations(globalIntgs.Zendesk)
	if err != nil {
		return result, err
	}

	var errs []string
	for _, tmJira := range ti.Jira {
		key := tmJira.UniqueKey()
		intg, ok := jiraIntgs[key]
		if !ok {
			errs = append(errs, fmt.Sprintf("unknown Jira integration for url %s and project key %s", tmJira.URL, tmJira.ProjectKey))
			continue
		}
		intg.EnableFailingPolicies = tmJira.EnableFailingPolicies
		result.Jira = append(result.Jira, &intg)
	}
	for _, tmZendesk := range ti.Zendesk {
		key := tmZendesk.UniqueKey()
		intg, ok := zendeskIntgs[key]
		if !ok {
			errs = append(errs, fmt.Sprintf("unknown Zendesk integration for url %s and group ID %v", tmZendesk.URL, tmZendesk.GroupID))
			continue
		}
		intg.EnableFailingPolicies = tmZendesk.EnableFailingPolicies
		result.Zendesk = append(result.Zendesk, &intg)
	}

	if len(errs) > 0 {
		err = errors.New(strings.Join(errs, "\n"))
	}
	return result, err
}

// Validate validates the team integrations for uniqueness.
func (ti TeamIntegrations) Validate() error {
	jira := make(map[string]*TeamJiraIntegration, len(ti.Jira))
	for _, j := range ti.Jira {
		key := j.UniqueKey()
		if _, ok := jira[key]; ok {
			return fmt.Errorf("duplicate Jira integration for url %s and project key %s", j.URL, j.ProjectKey)
		}
		jira[key] = j
	}

	zendesk := make(map[string]*TeamZendeskIntegration, len(ti.Zendesk))
	for _, z := range ti.Zendesk {
		key := z.UniqueKey()
		if _, ok := zendesk[key]; ok {
			return fmt.Errorf("duplicate Zendesk integration for url %s and group ID %v", z.URL, z.GroupID)
		}
		zendesk[key] = z
	}
	return nil
}

// TeamJiraIntegration configures an instance of an integration with the Jira
// system for a team.
type TeamJiraIntegration struct {
	URL                   string `json:"url"`
	ProjectKey            string `json:"project_key"`
	EnableFailingPolicies bool   `json:"enable_failing_policies"`
}

// UniqueKey returns the unique key of this integration.
func (j TeamJiraIntegration) UniqueKey() string {
	return j.URL + "\n" + j.ProjectKey
}

// TeamZendeskIntegration configures an instance of an integration with the
// external Zendesk service for a team.
type TeamZendeskIntegration struct {
	URL                   string `json:"url"`
	GroupID               int64  `json:"group_id"`
	EnableFailingPolicies bool   `json:"enable_failing_policies"`
}

// UniqueKey returns the unique key of this integration.
func (z TeamZendeskIntegration) UniqueKey() string {
	return z.URL + "\n" + strconv.FormatInt(z.GroupID, 10)
}

type TeamGoogleCalendarIntegration struct {
	Enable     bool   `json:"enable_calendar_events"`
	WebhookURL string `json:"webhook_url"`
}

// JiraIntegration configures an instance of an integration with the Jira
// system.
type JiraIntegration struct {
	URL                           string `json:"url"`
	Username                      string `json:"username"`
	APIToken                      string `json:"api_token"`
	ProjectKey                    string `json:"project_key"`
	EnableFailingPolicies         bool   `json:"enable_failing_policies"`
	EnableSoftwareVulnerabilities bool   `json:"enable_software_vulnerabilities"`
}

func (j JiraIntegration) uniqueKey() string {
	return j.URL + "\n" + j.ProjectKey
}

// IndexJiraIntegrations indexes the provided Jira integrations in a map keyed
// by 'URL\nProjectKey'. It returns an error if a duplicate configuration is
// found for the same combination. This is typically used to index the original
// integrations before applying the changes requested to modify the AppConfig.
//
// Note that the returned map uses non-pointer JiraIntegration struct values,
// so that any changes to the original value does not modify the value in the
// map. This is important because of how changes are merged with the original
// AppConfig when modifying it.
func IndexJiraIntegrations(jiraIntgs []*JiraIntegration) (map[string]JiraIntegration, error) {
	indexed := make(map[string]JiraIntegration, len(jiraIntgs))
	for _, intg := range jiraIntgs {
		key := intg.uniqueKey()
		if _, ok := indexed[key]; ok {
			return nil, fmt.Errorf("duplicate Jira integration for url %s and project key %s", intg.URL, intg.ProjectKey)
		}
		indexed[key] = *intg
	}
	return indexed, nil
}

// ValidateJiraIntegrations validates that the merge of the original and new
// Jira integrations does not result in any duplicate configuration, and that
// each modified or added integration can successfully connect to the external
// Jira service. It returns the list of integrations that were deleted, if any.
//
// On successful return, the newJiraIntgs slice is ready to be saved - it may
// have been updated using the original integrations if the API token was
// missing.
func ValidateJiraIntegrations(ctx context.Context, oriJiraIntgsIndexed map[string]JiraIntegration, newJiraIntgs []*JiraIntegration) (deleted []*JiraIntegration, err error) {
	newIndexed := make(map[string]*JiraIntegration, len(newJiraIntgs))
	for i, new := range newJiraIntgs {
		// first check for uniqueness
		key := new.uniqueKey()
		if _, ok := newIndexed[key]; ok {
			return nil, fmt.Errorf("duplicate Jira integration for url %s and project key %s", new.URL, new.ProjectKey)
		}
		newIndexed[key] = new

		// check if existing integration is being edited
		if old, ok := oriJiraIntgsIndexed[key]; ok {
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
			return nil, fmt.Errorf("Jira integration at index %d: %w", i, err)
		}
	}

	// collect any deleted integration
	for key, intg := range oriJiraIntgsIndexed {
		intg := intg // do not take address of iteration variable
		if _, ok := newIndexed[key]; !ok {
			deleted = append(deleted, &intg)
		}
	}
	return deleted, nil
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
	URL                           string `json:"url"`
	Email                         string `json:"email"`
	APIToken                      string `json:"api_token"`
	GroupID                       int64  `json:"group_id"`
	EnableFailingPolicies         bool   `json:"enable_failing_policies"`
	EnableSoftwareVulnerabilities bool   `json:"enable_software_vulnerabilities"`
}

func (z ZendeskIntegration) uniqueKey() string {
	return z.URL + "\n" + strconv.FormatInt(z.GroupID, 10)
}

// IndexZendeskIntegrations indexes the provided Zendesk integrations in a map
// keyed by 'URL\nGroupID'. It returns an error if a duplicate configuration is
// found for the same combination. This is typically used to index the original
// integrations before applying the changes requested to modify the AppConfig.
//
// Note that the returned map uses non-pointer ZendeskIntegration struct
// values, so that any changes to the original value does not modify the value
// in the map. This is important because of how changes are merged with the
// original AppConfig when modifying it.
func IndexZendeskIntegrations(zendeskIntgs []*ZendeskIntegration) (map[string]ZendeskIntegration, error) {
	indexed := make(map[string]ZendeskIntegration, len(zendeskIntgs))
	for _, intg := range zendeskIntgs {
		key := intg.uniqueKey()
		if _, ok := indexed[key]; ok {
			return nil, fmt.Errorf("duplicate Zendesk integration for url %s and group id %v", intg.URL, intg.GroupID)
		}
		indexed[key] = *intg
	}
	return indexed, nil
}

// ValidateZendeskIntegrations validates that the merge of the original and
// new Zendesk integrations does not result in any duplicate configuration,
// and that each modified or added integration can successfully connect to the
// external Zendesk service. It returns the list of integrations that were
// deleted, if any.
//
// On successful return, the newZendeskIntgs slice is ready to be saved - it
// may have been updated using the original integrations if the API token was
// missing.
func ValidateZendeskIntegrations(ctx context.Context, oriZendeskIntgsIndexed map[string]ZendeskIntegration, newZendeskIntgs []*ZendeskIntegration) (deleted []*ZendeskIntegration, err error) {
	newIndexed := make(map[string]*ZendeskIntegration, len(newZendeskIntgs))
	for i, new := range newZendeskIntgs {
		key := new.uniqueKey()
		// first check for uniqueness
		if _, ok := newIndexed[key]; ok {
			return nil, fmt.Errorf("duplicate Zendesk integration for url %s and group id %v", new.URL, new.GroupID)
		}
		newIndexed[key] = new

		// check if existing integration is being edited
		if old, ok := oriZendeskIntgsIndexed[key]; ok {
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
			return nil, fmt.Errorf("Zendesk integration at index %d: %w", i, err)
		}
	}

	// collect any deleted integration
	for key, intg := range oriZendeskIntgsIndexed {
		intg := intg // do not take address of iteration variable
		if _, ok := newIndexed[key]; !ok {
			deleted = append(deleted, &intg)
		}
	}
	return deleted, nil
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

const (
	GoogleCalendarEmail      = "client_email"
	GoogleCalendarPrivateKey = "private_key"
)

type GoogleCalendarIntegration struct {
	Domain string            `json:"domain"`
	ApiKey map[string]string `json:"api_key_json"`
}

// NDESSCEPProxyIntegration configures SCEP proxy for NDES SCEP server. Premium feature.
type NDESSCEPProxyIntegration struct {
	URL      string `json:"url"`
	AdminURL string `json:"admin_url"`
	Username string `json:"username"`
	Password string `json:"password"` // not stored here -- encrypted in DB
}

// Integrations configures the integrations with external systems.
type Integrations struct {
	Jira           []*JiraIntegration           `json:"jira"`
	Zendesk        []*ZendeskIntegration        `json:"zendesk"`
	GoogleCalendar []*GoogleCalendarIntegration `json:"google_calendar"`
	// NDESSCEPProxy settings. In JSON, not specifying this field means keep current setting, null means clear settings.
	NDESSCEPProxy optjson.Any[NDESSCEPProxyIntegration] `json:"ndes_scep_proxy"`
}

func ValidateEnabledActivitiesWebhook(webhook ActivitiesWebhookSettings, invalid *InvalidArgumentError) {
	if webhook.Enable {
		if webhook.DestinationURL == "" {
			invalid.Append(
				"webhook_settings.activities_webhook.destination_url", "destination_url is required to enable the activities webhook",
			)
		} else {
			if u, err := url.ParseRequestURI(webhook.DestinationURL); err != nil {
				invalid.Append("webhook_settings.activities_webhook.destination_url", err.Error())
			} else if (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
				invalid.Append(
					"webhook_settings.activities_webhook.destination_url", "destination_url must be https or http, and have a host",
				)
			}
		}
	}
}

// ValidateEnabledHostStatusIntegrations checks that the host status integrations
// is properly configured if enabled. It adds any error it finds to the invalid
// argument error, that can then be checked after the call for errors using
// invalid.HasErrors.
func ValidateEnabledHostStatusIntegrations(webhook HostStatusWebhookSettings, invalid *InvalidArgumentError) {
	if webhook.Enable {
		if webhook.DestinationURL == "" {
			invalid.Append("destination_url", "destination_url is required to enable the host status webhook")
		}
		if webhook.DaysCount <= 0 {
			invalid.Append("days_count", "days_count must be > 0 to enable the host status webhook")
		}
		if webhook.HostPercentage <= 0 {
			invalid.Append("host_percentage", "host_percentage must be > 0 to enable the host status webhook")
		}
	}
}

func ValidateGoogleCalendarIntegrations(intgs []*GoogleCalendarIntegration, invalid *InvalidArgumentError) {
	if len(intgs) > 1 {
		invalid.Append("integrations.google_calendar", "integrating with >1 Google Workspace service account is not yet supported.")
	}
	for _, intg := range intgs {
		if email, ok := intg.ApiKey[GoogleCalendarEmail]; !ok {
			invalid.Append(
				fmt.Sprintf("integrations.google_calendar.api_key_json.%s", GoogleCalendarEmail),
				fmt.Sprintf("%s is required", GoogleCalendarEmail),
			)
		} else {
			email = strings.TrimSpace(email)
			intg.ApiKey[GoogleCalendarEmail] = email
			if email == "" {
				invalid.Append(
					fmt.Sprintf("integrations.google_calendar.api_key_json.%s", GoogleCalendarEmail),
					fmt.Sprintf("%s cannot be blank", GoogleCalendarEmail),
				)
			}
		}
		if privateKey, ok := intg.ApiKey["private_key"]; !ok {
			invalid.Append(
				fmt.Sprintf("integrations.google_calendar.api_key_json.%s", GoogleCalendarPrivateKey),
				fmt.Sprintf("%s is required", GoogleCalendarPrivateKey),
			)
		} else {
			privateKey = strings.TrimSpace(privateKey)
			intg.ApiKey[GoogleCalendarPrivateKey] = privateKey
			if privateKey == "" {
				invalid.Append(
					fmt.Sprintf("integrations.google_calendar.api_key_json.%s", GoogleCalendarPrivateKey),
					fmt.Sprintf("%s cannot be blank", GoogleCalendarPrivateKey),
				)
			}
		}
		intg.Domain = strings.TrimSpace(intg.Domain)
		if intg.Domain == "" {
			invalid.Append("integrations.google_calendar.domain", "domain is required")
		}
	}
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
	if webhookEnabled && webhook.DestinationURL == "" {
		invalid.Append("destination_url", "destination_url is required to enable the vulnerabilities webhook")
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
	if webhookEnabled && webhook.DestinationURL == "" {
		invalid.Append("destination_url", "destination_url is required to enable the failing policies webhook")
	}
}

// ValidateEnabledFailingPoliciesTeamIntegrations is like
// ValidateEnabledFailingPoliciesIntegrations, but for team-specific
// integration structs.
func ValidateEnabledFailingPoliciesTeamIntegrations(webhook FailingPoliciesWebhookSettings, teamIntgs TeamIntegrations, invalid *InvalidArgumentError) {
	intgs := Integrations{
		Jira:    make([]*JiraIntegration, len(teamIntgs.Jira)),
		Zendesk: make([]*ZendeskIntegration, len(teamIntgs.Zendesk)),
	}
	for i, j := range teamIntgs.Jira {
		intgs.Jira[i] = &JiraIntegration{
			URL:                   j.URL,
			ProjectKey:            j.ProjectKey,
			EnableFailingPolicies: j.EnableFailingPolicies,
		}
	}
	for i, z := range teamIntgs.Zendesk {
		intgs.Zendesk[i] = &ZendeskIntegration{
			URL:                   z.URL,
			GroupID:               z.GroupID,
			EnableFailingPolicies: z.EnableFailingPolicies,
		}
	}
	ValidateEnabledFailingPoliciesIntegrations(webhook, intgs, invalid)
}
