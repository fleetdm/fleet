package main

import (
	"bytes"
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
	Key      string
}

type Note struct {
	Filename string
	Note     string
}

type Messages struct {
	SecretWarnings []SecretWarning
	Notes          []Note
}

type Comment struct {
	Filename string
	Comment  string
	Token    string
}

type FileToWrite struct {
	Path    string
	Content map[string]interface{}
}
type client interface {
	GetAppConfig() (*fleet.EnrichedAppConfig, error)
	GetEnrollSecretSpec() (*fleet.EnrollSecretSpec, error)
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

type GenerateGitopsCommand struct {
	Client       client
	CLI          *cli.Context
	Messages     Messages
	FilesToWrite map[string]interface{}
	Comments     []Comment
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
			FilesToWrite: make(map[string]interface{}),
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

	cmd.FilesToWrite["default.yml"] = map[string]interface{}{
		"org_settings": orgSettings,
	}

	if cmd.CLI.String("key") != "" {
		// Marshal and ummarshal the data to standardize the keys.
		b, err := yaml.Marshal(cmd.FilesToWrite["default.yml"])
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error marshaling org settings: %s\n", err)
			return ErrGeneric
		}
		var data map[string]interface{}
		if err := yaml.Unmarshal(b, &data); err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error unmarshaling org settings: %s\n", err)
			return ErrGeneric
		}
		value, ok := getValueAtKey(data, cmd.CLI.String("key"))
		if !ok {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Key %s not found in org settings\n", cmd.CLI.String("key"))
			return ErrGeneric
		}
		b, err = yaml.Marshal(value)
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error marshaling value: %s\n", err)
			return ErrGeneric
		}
		fmt.Fprintf(cmd.CLI.App.Writer, "%s", string(b))
		return nil
	}

	// Add comments to the result.
	for path, fileToWrite := range cmd.FilesToWrite {
		b, err := yaml.Marshal(fileToWrite)
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error marshaling file to write: %s\n", err)
			return ErrGeneric
		}
		for _, comment := range cmd.Comments {
			if comment.Filename == path {
				b = bytes.ReplaceAll(b,
					[]byte(comment.Token),
					[]byte("# "+comment.Comment),
				)
			}
		}
		fmt.Fprintf(cmd.CLI.App.Writer, "%s:\n %+v\n", path, string(b))
	}

	return nil
}

func (cmd *GenerateGitopsCommand) AddComment(filename, comment string) string {
	token := fmt.Sprintf("___GITOPS_COMMENT_%d___", len(cmd.Comments))
	cmd.Comments = append(cmd.Comments, Comment{
		Filename: filename,
		Comment:  comment,
		Token:    token,
	})
	return token
}

func (cmd *GenerateGitopsCommand) generateOrgSettings(appConfig *fleet.EnrichedAppConfig) (orgSettings map[string]interface{}, err error) {
	t := reflect.TypeOf(fleet.EnrichedAppConfig{})
	orgSettings = map[string]interface{}{
		jsonFieldName(t, "Features"):           appConfig.Features,
		jsonFieldName(t, "FleetDesktop"):       appConfig.FleetDesktop,
		jsonFieldName(t, "HostExpirySettings"): appConfig.HostExpirySettings,
		jsonFieldName(t, "OrgInfo"):            appConfig.OrgInfo,
		jsonFieldName(t, "ServerSettings"):     appConfig.ServerSettings,
		jsonFieldName(t, "Integrations"):       cmd.generateIntegrations(&appConfig.Integrations),
		jsonFieldName(t, "WebhookSettings"):    appConfig.WebhookSettings,
		jsonFieldName(t, "MDM"):                cmd.generateMDM(&appConfig.MDM),
		jsonFieldName(t, "YaraRules"):          cmd.generateYaraRules(appConfig.YaraRules),
	}

	// If --insecure is set, add real secrets.
	if cmd.CLI.Bool("insecure") {
		enrollSecrets, err := cmd.Client.GetEnrollSecretSpec()
		if err != nil {
			return nil, err
		}
		secrets := make([]map[string]string, len(enrollSecrets.Secrets))
		for i, spec := range enrollSecrets.Secrets {
			secrets[i] = map[string]string{"secret": spec.Secret}
		}
		orgSettings["secrets"] = secrets
	} else {
		(orgSettings)["secrets"] = cmd.AddComment("default.yml", "TODO: Add your secret here")
	}

	if (orgSettings)[jsonFieldName(t, "SSOSettings")], err = cmd.generateSSOSettings(appConfig.SSOSettings); err != nil {
		return nil, err
	}
	return orgSettings, nil
}

func (cmd *GenerateGitopsCommand) generateSSOSettings(ssoSettings *fleet.SSOSettings) (map[string]interface{}, error) {
	t := reflect.TypeOf(fleet.SSOSettings{})
	result := map[string]interface{}{
		jsonFieldName(t, "EnableSSO"):             ssoSettings.EnableSSO,
		jsonFieldName(t, "IDPName"):               ssoSettings.IDPName,
		jsonFieldName(t, "IDPImageURL"):           ssoSettings.IDPImageURL,
		jsonFieldName(t, "EntityID"):              ssoSettings.EntityID,
		jsonFieldName(t, "Metadata"):              ssoSettings.Metadata,
		jsonFieldName(t, "MetadataURL"):           ssoSettings.MetadataURL,
		jsonFieldName(t, "EnableJITProvisioning"): ssoSettings.EnableJITProvisioning,
		jsonFieldName(t, "EnableSSOIdPLogin"):     ssoSettings.EnableSSOIdPLogin,
	}
	if !cmd.CLI.Bool("insecure") {
		if ssoSettings.Metadata != "" {
			result[jsonFieldName(t, "Metadata")] = cmd.AddComment("default.yml", "TODO: Add your SSO metadata here")
		}
		if ssoSettings.MetadataURL != "" {
			result[jsonFieldName(t, "MetadataURL")] = cmd.AddComment("default.yml", "TODO: Add your SSO metadata URL here")
		}
	}
	return result, nil
}

func (cmd *GenerateGitopsCommand) generateIntegrations(integrations *fleet.Integrations) map[string]interface{} {
	// t := reflect.TypeOf(fleet.Integrations{})
	// Rather than crawling through the whole struct, we'll marshall/unmarshall it
	// to get the keys we want.
	b, err := yaml.Marshal(integrations)
	if err != nil {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error marshaling integrations: %s\n", err)
		return nil
	}
	var result map[string]interface{}
	if err := yaml.Unmarshal(b, &result); err != nil {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error unmarshaling integrations: %s\n", err)
		return nil
	}
	// Obfuscate secrets if not in insecure mode.
	if !cmd.CLI.Bool("insecure") {
		if googleCalendar, ok := result["google_calendar"]; ok {
			for _, intg := range googleCalendar.([]interface{}) {
				if apiKeyJson, ok := intg.(map[string]interface{})["api_key_json"]; ok {
					apiKeyJson.(map[string]interface{})["private_key"] = cmd.AddComment("default.yml", "TODO: Add your Google Calendar API key JSON here")
				}
			}
		}
		if jira, ok := result["jira"]; ok {
			for _, intg := range jira.([]interface{}) {
				intg.(map[string]interface{})["api_token"] = cmd.AddComment("default.yml", "TODO: Add your Jira API token here")
			}
		}
		if zendesk, ok := result["zendesk"]; ok {
			for _, intg := range zendesk.([]interface{}) {
				intg.(map[string]interface{})["api_token"] = cmd.AddComment("default.yml", "TODO: Add your Zendesk API token here")
			}
		}
		if digicert, ok := result["digicert"]; ok && digicert != nil {
			for _, intg := range digicert.([]interface{}) {
				intg.(map[string]interface{})["api_token"] = cmd.AddComment("default.yml", "TODO: Add your Digicert API token here")
			}
		}
		if ndes_scep_proxy, ok := result["ndes_scep_proxy"]; ok && ndes_scep_proxy != nil {
			ndes_scep_proxy.(map[string]interface{})["password"] = cmd.AddComment("default.yml", "TODO: Add your NDES SCEP proxy password here")
		}
		if custom_scep_proxy, ok := result["custom_scep_proxy"]; ok && custom_scep_proxy != nil {
			for _, intg := range custom_scep_proxy.([]interface{}) {
				intg.(map[string]interface{})["challenge"] = cmd.AddComment("default.yml", "TODO: Add your custom SCEP proxy challenge here")
			}
		}
	}

	return result
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
