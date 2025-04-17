package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/ghodss/yaml"
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

func jsonFieldName(t reflect.Type, fieldName string) string {
	field, ok := t.FieldByName(fieldName)
	if !ok {
		panic(fieldName + " not found in " + t.Name())
	}
	tag := field.Tag.Get("json")
	parts := strings.Split(tag, ",")
	name := parts[0]

	if name == "-" || name == "" {
		panic(field.Name + " has no json tag")
	}
	return name
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

		orgSettings, err := generateOrgSettings(c, appConfig, messages)
		if err != nil {
			fmt.Fprintf(c.App.ErrWriter, "Error generating org settings: %s\n", err)
			return ErrGeneric
		}

		b, err := yaml.Marshal(orgSettings)

		fmt.Fprintf(c.App.Writer, "App Config:\n %+v\n", string(b))
		return nil
	}
}

func generateOrgSettings(c *cli.Context, appConfig *fleet.EnrichedAppConfig, messages *Messages) (orgSettings *map[string]interface{}, err error) {
	t := reflect.TypeOf(fleet.EnrichedAppConfig{})
	orgSettings = &map[string]interface{}{
		jsonFieldName(t, "Features"):           appConfig.Features,
		jsonFieldName(t, "FleetDesktop"):       appConfig.FleetDesktop,
		jsonFieldName(t, "HostExpirySettings"): appConfig.HostExpirySettings,
		jsonFieldName(t, "OrgInfo"):            appConfig.OrgInfo,
		"secrets": []map[string]interface{}{
			{
				"secret": "# TODO: Add your secret here",
			},
		},
		jsonFieldName(t, "ServerSettings"):  appConfig.ServerSettings,
		jsonFieldName(t, "Integrations"):    generateIntegrations(c, &appConfig.Integrations, messages),
		jsonFieldName(t, "WebhookSettings"): appConfig.WebhookSettings,
		jsonFieldName(t, "MDM"):             generateMDM(c, &appConfig.MDM, messages),
		jsonFieldName(t, "YaraRules"):       generateYaraRules(c, appConfig.YaraRules, messages),
	}
	if (*orgSettings)[jsonFieldName(t, "SSOSettings")], err = generateSSOSettings(c, appConfig.SSOSettings, messages); err != nil {
		return nil, err
	}
	return orgSettings, nil
}

func generateSSOSettings(c *cli.Context, ssoSettings *fleet.SSOSettings, messages *Messages) (map[string]interface{}, error) {
	t := reflect.TypeOf(fleet.SSOSettings{})
	result := map[string]interface{}{
		jsonFieldName(t, "EnableSSO"): ssoSettings.EnableSSO,

		jsonFieldName(t, "IDPName"):               ssoSettings.IDPName,
		jsonFieldName(t, "IDPImageURL"):           ssoSettings.IDPImageURL,
		jsonFieldName(t, "EntityID"):              ssoSettings.EntityID,
		jsonFieldName(t, "Metadata"):              ssoSettings.Metadata,
		jsonFieldName(t, "MetadataURL"):           ssoSettings.MetadataURL,
		jsonFieldName(t, "EnableJITProvisioning"): ssoSettings.EnableJITProvisioning,
		jsonFieldName(t, "EnableSSOIdPLogin"):     ssoSettings.EnableSSOIdPLogin,
	}
	return result, nil
}

func generateIntegrations(c *cli.Context, ssoSettings *fleet.Integrations, messages *Messages) map[string]interface{} {
	return map[string]interface{}{}
}

func generateMDM(c *cli.Context, mdm *fleet.MDM, messages *Messages) map[string]interface{} {
	return map[string]interface{}{}
}

func generateYaraRules(c *cli.Context, yaraRules []fleet.YaraRule, messages *Messages) map[string]interface{} {
	return map[string]interface{}{}
}

func generateTeamSettings(c *cli.Context, teamID int) string {
	return fmt.Sprintf("team_settings_%d.yaml", teamID)
}

func generateAgentOptions(c *cli.Context) string {
	return "agent_options.yaml"
}

func generateControls(c *cli.Context) string {
	return "controls.yaml"
}

func generatePolicies(c *cli.Context) string {
	return "policies.yaml"
}

func generateQueries(c *cli.Context) string {
	return "queries.yaml"
}

func generateSoftware(c *cli.Context) string {
	return "software.yaml"
}

func generateLabels(c *cli.Context) string {
	return "labels.yaml"
}

var _ client = (*service.Client)(nil)
