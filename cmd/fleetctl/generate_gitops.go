package main

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/urfave/cli/v2"
)

type SecretWarning struct {
	Filename string
	Path     string
}

type Note struct {
	Filename string
	Note     string
}

type Messages struct {
	SecretWarnings []SecretWarning
	Notes          []Note
}

type client interface {
	GetAppConfig() (*fleet.EnrichedAppConfig, error)
}

func generateGitopsCommand() *cli.Command {
	return &cli.Command{
		Name:        "generate-gitops",
		Usage:       "Generate GitOps configuration files for Fleet.",
		Description: "This command generates GitOps configuration files for Fleet.",
		Action:      createGenerateGitopsAction(nil),
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
	}
}

func createGenerateGitopsAction(fleetClient client) func(*cli.Context) error {
	return func(c *cli.Context) error {
		var err error
		if fleetClient == nil {
			fleetClient, err = clientFromCLI(c)
			if err != nil {
				return err
			}
		}

		fmt.Println("Generating GitOps configuration files...")

		appConfig, err := fleetClient.GetAppConfig()
		if err != nil {
			return err
		}

		messages := &Messages{}

		orgSettings := generateOrgSettings(appConfig, messages)

		fmt.Printf("App Config: %+v\n", (*orgSettings)["org_info"])
		return nil
	}
}

func generateOrgSettings(appConfig *fleet.EnrichedAppConfig, messages *Messages) *map[string]interface{} {
	orgSettings := &map[string]interface{}{
		"features":             appConfig.Features,
		"fleet_desktop":        appConfig.FleetDesktop,
		"host_expiry_settings": appConfig.HostExpirySettings,
		"org_info":             appConfig.OrgInfo,
		"secrets": []map[string]interface{}{
			{
				"secret": "# TODO: Add your secret here",
			},
		},
		"server_settings":  appConfig.ServerSettings,
		"sso_settings":     generateSSOSettings(appConfig.SSOSettings, messages),
		"integrations":     generateIntegrations(&appConfig.Integrations, messages),
		"webhook_settings": appConfig.WebhookSettings,
		"mdm":              generateMDM(&appConfig.MDM, messages),
		"yara_rules":       generateYaraRules(appConfig.YaraRules, messages),
	}
	return orgSettings
}

func generateSSOSettings(ssoSettings *fleet.SSOSettings, messages *Messages) map[string]interface{} {
	return map[string]interface{}{}
}

func generateIntegrations(ssoSettings *fleet.Integrations, messages *Messages) map[string]interface{} {
	return map[string]interface{}{}
}

func generateMDM(mdm *fleet.MDM, messages *Messages) map[string]interface{} {
	return map[string]interface{}{}
}

func generateYaraRules(yaraRules []fleet.YaraRule, messages *Messages) map[string]interface{} {
	return map[string]interface{}{}
}

func generateTeamSettings(teamID int) string {
	return fmt.Sprintf("team_settings_%d.yaml", teamID)
}

func generateAgentOptions() string {
	return "agent_options.yaml"
}

func generateControls() string {
	return "controls.yaml"
}

func generatePolicies() string {
	return "policies.yaml"
}

func generateQueries() string {
	return "queries.yaml"
}

func generateSoftware() string {
	return "software.yaml"
}

func generateLabels() string {
	return "labels.yaml"
}

var _ client = (*service.Client)(nil)
