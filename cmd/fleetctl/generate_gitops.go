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

func getValueAtKey(data map[string]interface{}, path string) (interface{}, bool) {
	// Split the path into parts.
	parts := strings.Split(path, ".")
	var cur interface{} = data

	// Keep traversing the map using the keys in the path.
	for _, key := range parts {
		mp, ok := cur.(map[string]interface{})
		if !ok {
			return nil, false
		}
		cur, ok = mp[key]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

type FileToWrite struct {
	Path    string
	Content string
}

type GenerateGitopsCommand struct {
	Client       client
	CLI          *cli.Context
	Messages     Messages
	FilesToWrite []FileToWrite
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
			&cli.BoolFlag{
				Name:  "insecure",
				Usage: "Output sensitive information in plaintext.",
				Value: false,
			},
			&cli.StringFlag{
				Name:  "key",
				Usage: "A key to output the config value for.",
			},
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
		cmd := &GenerateGitopsCommand{
			Client:       fleetClient,
			CLI:          c,
			Messages:     Messages{},
			FilesToWrite: []FileToWrite{},
		}
		return cmd.Run()
	}
}

func (cmd *GenerateGitopsCommand) Run() error {
	fmt.Println("Generating GitOps configuration files...")

	appConfig, err := cmd.Client.GetAppConfig()
	if err != nil {
		return err
	}

	orgSettings, err := cmd.generateOrgSettings(appConfig)
	if err != nil {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating org settings: %s\n", err)
		return ErrGeneric
	}

	if cmd.CLI.String("key") != "" {
		value, ok := getValueAtKey(*orgSettings, cmd.CLI.String("key"))
		if !ok {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Key %s not found in org settings\n", cmd.CLI.String("key"))
			return ErrGeneric
		}
		b, err := yaml.Marshal(value)
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error marshaling value: %s\n", err)
			return ErrGeneric
		}
		fmt.Fprintf(cmd.CLI.App.Writer, "%s", string(b))
		return nil
	}

	b, err := yaml.Marshal(orgSettings)

	fmt.Fprintf(cmd.CLI.App.Writer, "App Config:\n %+v\n", string(b))
	return nil
}

func (cmd *GenerateGitopsCommand) generateOrgSettings(appConfig *fleet.EnrichedAppConfig) (orgSettings *map[string]interface{}, err error) {
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
		jsonFieldName(t, "Integrations"):    cmd.generateIntegrations(&appConfig.Integrations),
		jsonFieldName(t, "WebhookSettings"): appConfig.WebhookSettings,
		jsonFieldName(t, "MDM"):             cmd.generateMDM(&appConfig.MDM),
		jsonFieldName(t, "YaraRules"):       cmd.generateYaraRules(appConfig.YaraRules),
	}
	if (*orgSettings)[jsonFieldName(t, "SSOSettings")], err = cmd.generateSSOSettings(appConfig.SSOSettings); err != nil {
		return nil, err
	}
	return orgSettings, nil
}

func (cmd *GenerateGitopsCommand) generateSSOSettings(ssoSettings *fleet.SSOSettings) (map[string]interface{}, error) {
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

func (cmd *GenerateGitopsCommand) generateIntegrations(ssoSettings *fleet.Integrations) map[string]interface{} {
	return map[string]interface{}{}
}

func (cmd *GenerateGitopsCommand) generateMDM(mdm *fleet.MDM) map[string]interface{} {
	return map[string]interface{}{}
}

func (cmd *GenerateGitopsCommand) generateYaraRules(yaraRules []fleet.YaraRule) map[string]interface{} {
	return map[string]interface{}{}
}

func (cmd *GenerateGitopsCommand) generateTeamSettings(teamID int) string {
	return fmt.Sprintf("team_settings_%d.yaml", teamID)
}

func (cmd *GenerateGitopsCommand) generateAgentOptions() string {
	return "agent_options.yaml"
}

func (cmd *GenerateGitopsCommand) generateControls() string {
	return "controls.yaml"
}

func (cmd *GenerateGitopsCommand) generatePolicies() string {
	return "policies.yaml"
}

func (cmd *GenerateGitopsCommand) generateQueries() string {
	return "queries.yaml"
}

func (cmd *GenerateGitopsCommand) generateSoftware() string {
	return "software.yaml"
}

func (cmd *GenerateGitopsCommand) generateLabels() string {
	return "labels.yaml"
}

var _ client = (*service.Client)(nil)
